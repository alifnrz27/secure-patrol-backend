package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/setting/repository"
	"secure-patrol-backend/pkg/log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cacheTTL bounds how long another server instance may keep old values.
const cacheTTL = 30 * time.Second

var (
	ErrUnknownSetting = errors.New("unknown setting")
	ErrNoChanges      = errors.New("values must contain at least one setting")
)

// ValidationError lists every invalid value of an update.
type ValidationError struct {
	Messages []string
}

func (e *ValidationError) Error() string { return strings.Join(e.Messages, "; ") }

// Setting is one setting with its effective value and where the value comes from.
type Setting struct {
	Definition
	Value       string
	GlobalValue string
	// UnitValue is the unit's own value; nil when the unit follows the global value.
	UnitValue *string
	IsDefault bool
	UpdatedBy *int64
	UpdatedAt *time.Time
}

type SettingService interface {
	// Current returns the effective global settings (cached for a short time).
	Current() Values
	// ForUnit returns the effective settings of a unit (global values with the
	// unit's overrides); nil returns the global settings.
	ForUnit(unitID *int64) Values
	// List returns the settings of the global level (unitID nil) or of a unit.
	List(unitID *int64) ([]Setting, error)
	// Update sets several values at once for the global level (unitID nil) or a
	// unit. A JSON null resets a global value to its default, or makes a unit
	// follow the global value again.
	Update(unitID *int64, changes map[string]json.RawMessage, actorID int64) ([]Setting, error)
	// EnsureDefaults creates missing global settings with their default value.
	EnsureDefaults() (int64, error)
}

type cachedValues struct {
	values   Values
	loadedAt time.Time
}

type service struct {
	repo repository.SettingRepository

	mu    sync.Mutex
	cache map[int64]cachedValues // key 0 = global
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &service{repo: repo, cache: map[int64]cachedValues{}}
}

var (
	defaultService   SettingService
	defaultServiceMu sync.RWMutex
)

// Init sets the service used by Current() and ForUnit(). It is called once at startup.
func Init(s SettingService) {
	defaultServiceMu.Lock()
	defer defaultServiceMu.Unlock()
	defaultService = s
}

// Instance returns the service set by Init, or nil before startup.
func Instance() SettingService {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	return defaultService
}

// Current returns the effective global settings, or the built-in defaults when
// Init was not called (e.g. in unit tests).
func Current() Values {
	if s := Instance(); s != nil {
		return s.Current()
	}
	return DefaultValues()
}

// ForUnit returns the effective settings of a unit (nil = global).
func ForUnit(unitID *int64) Values {
	if s := Instance(); s != nil {
		return s.ForUnit(unitID)
	}
	return DefaultValues()
}

func toMap(settings []models.SystemSetting) map[string]string {
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	return result
}

func warnInvalid(key, value string, err error) {
	log.Warnf("settings: stored value %q for %s is invalid (%v), ignoring it", value, key, err)
}

func (s *service) Current() Values {
	return s.ForUnit(nil)
}

func (s *service) ForUnit(unitID *int64) Values {
	var cacheKey int64
	if unitID != nil {
		cacheKey = *unitID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if cached, ok := s.cache[cacheKey]; ok && time.Since(cached.loadedAt) < cacheTTL {
		return cached.values
	}

	global, err := s.repo.FindGlobal()
	if err != nil {
		// Keep serving the last known (or default) values rather than failing requests.
		log.Errorf("settings: cannot load, using previous values: %v", err)
		if cached, ok := s.cache[cacheKey]; ok {
			return cached.values
		}
		return DefaultValues()
	}

	layers := []map[string]string{toMap(global)}
	if unitID != nil {
		unit, err := s.repo.FindByUnit(*unitID)
		if err != nil {
			log.Errorf("settings: cannot load unit %d settings, using global values: %v", *unitID, err)
		} else {
			layers = append(layers, toMap(unit))
		}
	}

	values := buildValues(layers, warnInvalid)
	s.cache[cacheKey] = cachedValues{values: values, loadedAt: time.Now()}
	return values
}

func (s *service) invalidate() {
	s.mu.Lock()
	s.cache = map[int64]cachedValues{}
	s.mu.Unlock()
}

func (s *service) List(unitID *int64) ([]Setting, error) {
	global, err := s.repo.FindGlobal()
	if err != nil {
		return nil, err
	}
	globalValues := buildValues([]map[string]string{toMap(global)}, warnInvalid)
	globalRows := make(map[string]models.SystemSetting, len(global))
	for _, row := range global {
		globalRows[row.Key] = row
	}

	unitRows := map[string]models.SystemSetting{}
	if unitID != nil {
		unit, err := s.repo.FindByUnit(*unitID)
		if err != nil {
			return nil, err
		}
		for _, row := range unit {
			unitRows[row.Key] = row
		}
	}

	values := s.ForUnit(unitID)
	result := make([]Setting, 0, len(Definitions))
	for _, def := range Definitions {
		setting := Setting{Definition: def, Value: values.Raw[def.Key], GlobalValue: globalValues.Raw[def.Key]}
		setting.IsDefault = setting.Value == mustNormalize(def, def.Default)

		row, hasRow := globalRows[def.Key]
		if unitID != nil {
			if unitRow, ok := unitRows[def.Key]; ok {
				if normalized, err := def.normalize(unitRow.Value); err == nil {
					setting.UnitValue = &normalized
				}
				row, hasRow = unitRow, true
			} else {
				hasRow = false
			}
		}
		if hasRow {
			updatedAt := row.UpdatedAt
			setting.UpdatedAt = &updatedAt
			setting.UpdatedBy = row.UpdatedBy
		}
		result = append(result, setting)
	}
	return result, nil
}

func (s *service) Update(unitID *int64, changes map[string]json.RawMessage, actorID int64) ([]Setting, error) {
	if len(changes) == 0 {
		return nil, ErrNoChanges
	}

	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make(map[string]string, len(changes))
	var resets []string
	var messages []string
	for _, key := range keys {
		def, ok := definition(key)
		if !ok {
			messages = append(messages, fmt.Sprintf("%s: %v", key, ErrUnknownSetting))
			continue
		}

		raw := strings.TrimSpace(string(changes[key]))
		if raw == "null" {
			if unitID == nil {
				values[key] = mustNormalize(def, def.Default)
			} else {
				resets = append(resets, key)
			}
			continue
		}
		// Accept JSON numbers, booleans and strings holding them ("150", "true").
		if unquoted, err := strconv.Unquote(raw); err == nil {
			raw = unquoted
		}
		normalized, err := def.normalize(raw)
		if err != nil {
			messages = append(messages, err.Error())
			continue
		}
		values[key] = normalized
	}

	// All or nothing: one invalid value rejects the whole update.
	if len(messages) > 0 {
		return nil, &ValidationError{Messages: messages}
	}

	if len(values) > 0 {
		if err := s.repo.Save(unitID, values, actorID); err != nil {
			return nil, err
		}
	}
	if unitID != nil && len(resets) > 0 {
		if err := s.repo.DeleteUnitOverrides(*unitID, resets); err != nil {
			return nil, err
		}
	}
	s.invalidate()
	return s.List(unitID)
}

func (s *service) EnsureDefaults() (int64, error) {
	settings := make([]models.SystemSetting, 0, len(Definitions))
	for _, def := range Definitions {
		settings = append(settings, models.SystemSetting{Key: def.Key, Value: mustNormalize(def, def.Default)})
	}
	created, err := s.repo.InsertMissingGlobal(settings)
	if err == nil {
		s.invalidate()
	}
	return created, err
}

func mustNormalize(def Definition, value string) string {
	normalized, err := def.normalize(value)
	if err != nil {
		panic(fmt.Sprintf("setting %s has an invalid default %q: %v", def.Key, value, err))
	}
	return normalized
}
