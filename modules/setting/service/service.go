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

// Setting is one setting with its effective value and metadata.
type Setting struct {
	Definition
	Value     string
	IsDefault bool
	UpdatedBy *int64
	UpdatedAt *time.Time
}

type SettingService interface {
	// Current returns the effective settings (cached for a short time).
	Current() Values
	List() ([]Setting, error)
	// Update sets several values at once; a JSON null resets a setting to its default.
	Update(changes map[string]json.RawMessage, actorID int64) ([]Setting, error)
	// EnsureDefaults creates missing settings with their default value.
	EnsureDefaults() (int64, error)
}

type service struct {
	repo repository.SettingRepository

	mu       sync.Mutex
	cached   Values
	cachedAt time.Time
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &service{repo: repo}
}

var (
	defaultService   SettingService
	defaultServiceMu sync.RWMutex
)

// Init sets the service used by Current(). It is called once at startup.
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

// Current returns the effective settings from the service set by Init, or the
// built-in defaults when it was not initialized (e.g. in unit tests).
func Current() Values {
	defaultServiceMu.RLock()
	s := defaultService
	defaultServiceMu.RUnlock()
	if s == nil {
		return DefaultValues()
	}
	return s.Current()
}

func (s *service) Current() Values {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cachedAt.IsZero() && time.Since(s.cachedAt) < cacheTTL {
		return s.cached
	}

	settings, err := s.repo.FindAll()
	if err != nil {
		// Keep serving the last known (or default) values rather than failing requests.
		log.Errorf("settings: cannot load, using previous values: %v", err)
		if s.cachedAt.IsZero() {
			return DefaultValues()
		}
		return s.cached
	}

	stored := make(map[string]string, len(settings))
	for _, setting := range settings {
		stored[setting.Key] = setting.Value
	}
	s.cached = buildValues(stored, func(key, value string, err error) {
		log.Warnf("settings: stored value %q for %s is invalid (%v), using the default", value, key, err)
	})
	s.cachedAt = time.Now()
	return s.cached
}

func (s *service) invalidate() {
	s.mu.Lock()
	s.cachedAt = time.Time{}
	s.mu.Unlock()
}

func (s *service) List() ([]Setting, error) {
	stored, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]models.SystemSetting, len(stored))
	for _, setting := range stored {
		byKey[setting.Key] = setting
	}

	values := s.Current()
	result := make([]Setting, 0, len(Definitions))
	for _, def := range Definitions {
		setting := Setting{Definition: def, Value: values.Raw[def.Key]}
		setting.IsDefault = setting.Value == mustNormalize(def, def.Default)
		if row, ok := byKey[def.Key]; ok {
			updatedAt := row.UpdatedAt
			setting.UpdatedAt = &updatedAt
			setting.UpdatedBy = row.UpdatedBy
		}
		result = append(result, setting)
	}
	return result, nil
}

func (s *service) Update(changes map[string]json.RawMessage, actorID int64) ([]Setting, error) {
	if len(changes) == 0 {
		return nil, ErrNoChanges
	}

	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make(map[string]string, len(changes))
	var messages []string
	for _, key := range keys {
		def, ok := definition(key)
		if !ok {
			messages = append(messages, fmt.Sprintf("%s: %v", key, ErrUnknownSetting))
			continue
		}

		raw := strings.TrimSpace(string(changes[key]))
		if raw == "null" {
			values[key] = mustNormalize(def, def.Default)
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

	if err := s.repo.Save(values, actorID); err != nil {
		return nil, err
	}
	s.invalidate()
	return s.List()
}

func (s *service) EnsureDefaults() (int64, error) {
	settings := make([]models.SystemSetting, 0, len(Definitions))
	for _, def := range Definitions {
		settings = append(settings, models.SystemSetting{Key: def.Key, Value: mustNormalize(def, def.Default)})
	}
	created, err := s.repo.InsertMissing(settings)
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
