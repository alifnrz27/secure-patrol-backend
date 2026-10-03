// Package service checks the installed license and enforces its limits.
//
// Limits (units, app clients) always come from the verified license code, and
// usage is always counted from the real data, so editing the database does
// not help: rows above the limit simply stop working. A worker re-verifies the
// stored code every minute and detects a server clock that was moved back.
package service

import (
	"context"
	"errors"
	"fmt"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/pkg/license"
	"secure-patrol-backend/pkg/log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// State of the license.
type State string

const (
	StateMissing State = "missing" // no license installed
	StateActive  State = "active"
	StateGrace   State = "grace"   // expired, still working during the grace days
	StateExpired State = "expired" // grace days are over
	StateInvalid State = "invalid" // tampered code, other installation, or clock moved back
)

const (
	// checkInterval is how often the worker re-verifies the license.
	checkInterval = time.Minute
	// clockTolerance is how far the clock may go back (e.g. an NTP correction).
	clockTolerance = 24 * time.Hour
	// warnBefore starts the "license expires soon" warning.
	warnBefore = 30 * 24 * time.Hour
)

var (
	ErrLicenseInactive     = errors.New("license is not active, contact your administrator")
	ErrUnitLimitReached    = errors.New("the license unit limit has been reached")
	ErrAppClientLimit      = errors.New("the license app client limit has been reached")
	ErrUnitOverLicense     = errors.New("your unit exceeds the license limit, contact the head office")
	ErrAppClientOverLimit  = errors.New("this app client exceeds the license limit")
	ErrOtherInstallation   = errors.New("this license was issued for another installation")
	ErrLicenseAlreadyEnded = errors.New("this license has already expired")
)

// Status is the license as the server sees it now.
type Status struct {
	State      State
	Reason     string
	InstallID  string
	Payload    *license.Payload
	CheckedAt  time.Time
	ExpiringIn *time.Duration // set when the license ends within warnBefore
}

// Locked reports whether only the Super-Admin may sign in (to install a license).
func (s Status) Locked() bool {
	return s.State == StateMissing || s.State == StateExpired || s.State == StateInvalid
}

// Usage is the number of active units and app clients against the limits.
type Usage struct {
	Units         int64
	AppClients    int64
	MaxUnits      int
	MaxAppClients int
}

func (u Usage) UnitsOver() bool { return u.MaxUnits > 0 && u.Units > int64(u.MaxUnits) }
func (u Usage) AppClientsOver() bool {
	return u.MaxAppClients > 0 && u.AppClients > int64(u.MaxAppClients)
}

type LicenseService interface {
	// Status is cheap: the verified payload is cached and the time-based state
	// is computed on every call, so expiry takes effect to the second.
	Status() Status
	// Refresh re-reads and re-verifies the stored license and checks the clock.
	Refresh() Status
	// Install verifies and stores a new license code.
	Install(code string, installedBy *int64) (Status, error)
	Usage() (Usage, error)

	// CanActivateUnit / CanActivateAppClient are checked before a unit or app
	// client is created or activated.
	CanActivateUnit() error
	CanActivateAppClient() error
	// UnitAllowed / AppClientAllowed report whether a row is within the limit:
	// the oldest active rows are kept, newer rows above the limit do not work.
	UnitAllowed(unitID int64) (bool, error)
	AppClientAllowed(appClientID int64) (bool, error)

	RunWorker(ctx context.Context)
}

type verified struct {
	payload *license.Payload
	invalid string // reason when the stored code is not usable
	clock   string // reason when the clock was moved back
	missing bool
}

type service struct {
	db *gorm.DB

	mu      sync.RWMutex
	current verified
	checked time.Time

	lastLogged string
}

func NewLicenseService(db *gorm.DB) LicenseService {
	s := &service{db: db}
	s.Refresh()
	return s
}

var (
	instance   LicenseService
	instanceMu sync.RWMutex
)

// Init sets the service used by Instance(); called once at startup.
func Init(s LicenseService) {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	instance = s
}

// Instance returns the service set by Init, or nil (unit tests).
func Instance() LicenseService {
	instanceMu.RLock()
	defer instanceMu.RUnlock()
	return instance
}

func (s *service) Status() Status {
	s.mu.RLock()
	current, checked := s.current, s.checked
	s.mu.RUnlock()
	return evaluate(current, checked, time.Now())
}

func evaluate(v verified, checked time.Time, now time.Time) Status {
	status := Status{InstallID: helper.InstallID(), Payload: v.payload, CheckedAt: checked}
	switch {
	case v.missing:
		status.State, status.Reason = StateMissing, "no license is installed"
	case v.invalid != "":
		status.State, status.Reason = StateInvalid, v.invalid
	case v.clock != "":
		status.State, status.Reason = StateInvalid, v.clock
	case now.Before(v.payload.ExpiresAt):
		status.State = StateActive
		if left := v.payload.ExpiresAt.Sub(now); left <= warnBefore {
			status.ExpiringIn = &left
		}
	case now.Before(v.payload.GraceUntil()):
		status.State, status.Reason = StateGrace, "the license has expired, renew it before the grace period ends"
		left := v.payload.GraceUntil().Sub(now)
		status.ExpiringIn = &left
	default:
		status.State, status.Reason = StateExpired, "the license and its grace period have expired"
	}
	return status
}

// verify checks a code against the embedded public key and this installation.
func verify(code string) (license.Payload, error) {
	payload, err := license.Verify(code, license.PublicKey())
	if err != nil {
		return payload, err
	}
	if !strings.EqualFold(strings.TrimSpace(payload.InstallID), helper.InstallID()) {
		return payload, ErrOtherInstallation
	}
	return payload, nil
}

func (s *service) Refresh() Status {
	var v verified

	var stored models.License
	err := s.db.Order("id DESC").First(&stored).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		v.missing = true
	case err != nil:
		// Keep the last known result when the database is briefly unavailable.
		log.Errorf("license: cannot load license: %v", err)
		return s.Status()
	default:
		payload, err := verify(stored.Code)
		if err != nil {
			v.invalid = "the installed license is not valid: " + err.Error()
		} else {
			v.payload = &payload
			v.clock = s.checkClock(payload)
		}
	}

	s.mu.Lock()
	s.current, s.checked = v, time.Now()
	s.mu.Unlock()

	status := s.Status()
	s.logChanges(status)
	return status
}

// checkClock detects a server clock that was moved back to stretch the
// license. The latest time ever seen is kept in a signed row, and data
// timestamps (audit log, scans) and the license issue date give further floors.
func (s *service) checkClock(payload license.Payload) string {
	now := time.Now()
	floor := payload.IssuedAt

	var state models.LicenseState
	if err := s.db.First(&state, 1).Error; err == nil {
		if helper.LicenseStateMAC(state.LastSeenAt.UTC().Format(time.RFC3339)) == state.Mac {
			if state.LastSeenAt.After(floor) {
				floor = state.LastSeenAt
			}
		} else {
			s.recordEvent("license state was modified in the database")
		}
	}

	var latest struct{ At *time.Time }
	s.db.Raw(`SELECT GREATEST(
		(SELECT MAX(created_at) FROM audit_logs WHERE source <> ?),
		(SELECT MAX(received_at) FROM patrol_scans)) AS at`, models.AuditSourceLicense).Scan(&latest)
	if latest.At != nil && latest.At.After(floor) {
		floor = *latest.At
	}

	if now.Before(floor.Add(-clockTolerance)) {
		return fmt.Sprintf("the server clock (%s) is behind data already recorded (%s); correct the server clock",
			now.Format(time.RFC3339), floor.Format(time.RFC3339))
	}

	// Only move forward, so a wrong clock cannot erase the latest time seen.
	seen := now
	if floor.After(seen) {
		seen = floor
	}
	seen = seen.UTC().Truncate(time.Second)
	s.db.Save(&models.LicenseState{ID: 1, LastSeenAt: seen, Mac: helper.LicenseStateMAC(seen.Format(time.RFC3339))})
	return ""
}

func (s *service) Install(code string, installedBy *int64) (Status, error) {
	code = strings.TrimSpace(code)
	payload, err := verify(code)
	if err != nil {
		return s.Status(), err
	}
	if !time.Now().Before(payload.GraceUntil()) {
		return s.Status(), ErrLicenseAlreadyEnded
	}

	if err := s.db.Create(&models.License{Code: code, LicenseID: payload.LicenseID, InstalledBy: installedBy}).Error; err != nil {
		return s.Status(), err
	}
	return s.Refresh(), nil
}

func (s *service) Usage() (Usage, error) {
	var usage Usage
	if payload := s.Status().Payload; payload != nil {
		usage.MaxUnits, usage.MaxAppClients = payload.MaxUnits, payload.MaxAppClients
	}
	if err := s.activeUnits().Count(&usage.Units).Error; err != nil {
		return usage, err
	}
	err := s.activeAppClients().Count(&usage.AppClients).Error
	return usage, err
}

func (s *service) activeUnits() *gorm.DB {
	return s.db.Model(&models.Unit{}).Where("is_active = ?", true)
}

func (s *service) activeAppClients() *gorm.DB {
	return s.db.Model(&models.AppClient{}).
		Where("is_active = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now())
}

func (s *service) limits() (*license.Payload, error) {
	status := s.Status()
	if status.Locked() || status.Payload == nil {
		return nil, ErrLicenseInactive
	}
	return status.Payload, nil
}

func (s *service) CanActivateUnit() error {
	payload, err := s.limits()
	if err != nil {
		return err
	}
	var count int64
	if err := s.activeUnits().Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(payload.MaxUnits) {
		return fmt.Errorf("%w (%d)", ErrUnitLimitReached, payload.MaxUnits)
	}
	return nil
}

// CanActivateAppClient allows creating app clients before a license is
// installed, because the first web app client is needed to install it.
func (s *service) CanActivateAppClient() error {
	status := s.Status()
	if status.Payload == nil {
		return nil
	}
	var count int64
	if err := s.activeAppClients().Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(status.Payload.MaxAppClients) {
		return fmt.Errorf("%w (%d)", ErrAppClientLimit, status.Payload.MaxAppClients)
	}
	return nil
}

func (s *service) UnitAllowed(unitID int64) (bool, error) {
	payload, err := s.limits()
	if err != nil {
		return false, nil
	}
	var older int64
	if err := s.activeUnits().Where("id < ?", unitID).Count(&older).Error; err != nil {
		return false, err
	}
	return older < int64(payload.MaxUnits), nil
}

// AppClientAllowed accepts every app client while no valid license payload is
// known (the server is locked then anyway, except for installing a license).
func (s *service) AppClientAllowed(appClientID int64) (bool, error) {
	payload := s.Status().Payload
	if payload == nil {
		return true, nil
	}
	var older int64
	if err := s.activeAppClients().Where("id < ?", appClientID).Count(&older).Error; err != nil {
		return false, err
	}
	return older < int64(payload.MaxAppClients), nil
}

// logChanges writes the license state and over-limit usage to the audit log
// when they change, so violations stay visible.
func (s *service) logChanges(status Status) {
	summary := string(status.State)
	if status.Reason != "" && status.State == StateInvalid {
		summary += ": " + status.Reason
	}
	if usage, err := s.Usage(); err == nil {
		if usage.UnitsOver() {
			summary += fmt.Sprintf("; units over limit %d/%d", usage.Units, usage.MaxUnits)
		}
		if usage.AppClientsOver() {
			summary += fmt.Sprintf("; app clients over limit %d/%d", usage.AppClients, usage.MaxAppClients)
		}
	}

	s.mu.Lock()
	changed := summary != s.lastLogged
	s.lastLogged = summary
	s.mu.Unlock()

	if changed {
		log.Warnf("license: %s", summary)
		// "missing" is the normal state of a new installation, not a violation.
		if (status.State != StateActive && status.State != StateMissing) || strings.Contains(summary, "over limit") {
			s.recordEvent(summary)
		}
	}
}

func (s *service) recordEvent(message string) {
	if len(message) > 255 {
		message = message[:255]
	}
	s.db.Create(&models.AuditLog{
		Action:   models.AuditActionUpdate,
		Resource: "license",
		Endpoint: "LICENSE CHECK",
		Method:   "SYSTEM",
		Path:     message,
		Source:   models.AuditSourceLicense,
	})
}

// RunWorker re-verifies the license every minute.
func (s *service) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Refresh()
		}
	}
}
