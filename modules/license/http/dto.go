package http

import (
	"math"
	"secure-patrol-backend/modules/license/service"
	"time"
)

type LimitDTO struct {
	Max  int   `json:"max"`
	Used int64 `json:"used"`
}

type LicenseInfoDTO struct {
	LicenseID  string    `json:"license_id"`
	Customer   string    `json:"customer"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	GraceDays  int       `json:"grace_days"`
	GraceUntil time.Time `json:"grace_until"`
}

// LicenseDTO is the full license page for the Super-Admin.
type LicenseDTO struct {
	Status    service.State   `json:"status"`
	Reason    string          `json:"reason"`
	Locked    bool            `json:"locked"`
	InstallID string          `json:"install_id"`
	License   *LicenseInfoDTO `json:"license"`
	// DaysLeft counts down to expires_at (active) or grace_until (grace); null otherwise.
	DaysLeft *int `json:"days_left"`
	Limits   struct {
		Units      LimitDTO `json:"units"`
		AppClients LimitDTO `json:"app_clients"`
	} `json:"limits"`
	OverLimit struct {
		Units      bool `json:"units"`
		AppClients bool `json:"app_clients"`
	} `json:"over_limit"`
	CheckedAt time.Time `json:"checked_at"`
}

// LicenseSummaryDTO is the short status sent with the login and profile, for banners.
type LicenseSummaryDTO struct {
	Status     service.State `json:"status"`
	ExpiresAt  *time.Time    `json:"expires_at"`
	GraceUntil *time.Time    `json:"grace_until"`
	DaysLeft   *int          `json:"days_left"`
}

func daysLeft(status service.Status) *int {
	if status.ExpiringIn == nil {
		return nil
	}
	days := int(math.Ceil(status.ExpiringIn.Hours() / 24))
	return &days
}

func ToLicenseDTO(status service.Status, usage service.Usage) LicenseDTO {
	result := LicenseDTO{
		Status:    status.State,
		Reason:    status.Reason,
		Locked:    status.Locked(),
		InstallID: status.InstallID,
		DaysLeft:  daysLeft(status),
		CheckedAt: status.CheckedAt,
	}
	if p := status.Payload; p != nil {
		result.License = &LicenseInfoDTO{
			LicenseID: p.LicenseID, Customer: p.Customer, IssuedAt: p.IssuedAt,
			ExpiresAt: p.ExpiresAt, GraceDays: p.GraceDays, GraceUntil: p.GraceUntil(),
		}
	}
	result.Limits.Units = LimitDTO{Max: usage.MaxUnits, Used: usage.Units}
	result.Limits.AppClients = LimitDTO{Max: usage.MaxAppClients, Used: usage.AppClients}
	result.OverLimit.Units = usage.UnitsOver()
	result.OverLimit.AppClients = usage.AppClientsOver()
	return result
}

// Summary returns the short license status, or nil before startup (unit tests).
func Summary() *LicenseSummaryDTO {
	licenses := service.Instance()
	if licenses == nil {
		return nil
	}
	status := licenses.Status()
	summary := &LicenseSummaryDTO{Status: status.State, DaysLeft: daysLeft(status)}
	if p := status.Payload; p != nil {
		expires, grace := p.ExpiresAt, p.GraceUntil()
		summary.ExpiresAt, summary.GraceUntil = &expires, &grace
	}
	return summary
}

// PublicStatusDTO is the license status for the apps before login, so they can
// show a "no active license" screen. It holds no customer or installation data.
type PublicStatusDTO struct {
	Status service.State `json:"status"`
	// Locked is true when only the Super-Admin may sign in, to install a license.
	Locked     bool       `json:"locked"`
	Message    string     `json:"message"`
	ExpiresAt  *time.Time `json:"expires_at"`
	GraceUntil *time.Time `json:"grace_until"`
	DaysLeft   *int       `json:"days_left"`
}

var publicMessages = map[service.State]string{
	service.StateMissing: "no license is installed, the Super-Admin must install one",
	service.StateActive:  "the license is active",
	service.StateGrace:   "the license has expired and the grace period is running, renew it",
	service.StateExpired: "the license has expired, the Super-Admin must install a new one",
	service.StateInvalid: "the license is not valid, the Super-Admin must install a valid one",
}

func ToPublicStatusDTO(status service.Status) PublicStatusDTO {
	result := PublicStatusDTO{
		Status:   status.State,
		Locked:   status.Locked(),
		Message:  publicMessages[status.State],
		DaysLeft: daysLeft(status),
	}
	if p := status.Payload; p != nil && status.State != service.StateInvalid {
		expires, grace := p.ExpiresAt, p.GraceUntil()
		result.ExpiresAt, result.GraceUntil = &expires, &grace
	}
	return result
}
