package models

import (
	"time"

	"gorm.io/gorm"
)

// License is an installed license code. The newest row is the current license;
// older rows are kept as history. Limits are always read from the verified
// code, never from other columns.
type License struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	Code        string         `json:"-" gorm:"type:text;not null"`
	LicenseID   string         `json:"license_id" gorm:"type:varchar(100);not null"`
	InstalledBy *int64         `json:"installed_by"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// LicenseState remembers the latest time the server has seen, to detect a
// server clock that was moved back. Mac signs the value.
type LicenseState struct {
	ID         int64     `gorm:"primaryKey"`
	LastSeenAt time.Time `gorm:"not null"`
	Mac        string    `gorm:"type:varchar(64);not null"`
	UpdatedAt  time.Time
}

// AuditSourceLicense marks audit entries written by the license checker.
const AuditSourceLicense = "license"
