package models

import "time"

// DefaultAppName is the application name used until the Super-Admin changes it.
const DefaultAppName = "Secure Patrol"

// BrandingID is the id of the only branding row.
const BrandingID = 1

// Branding is the application name and logo shown by the web and mobile apps.
// There is exactly one row (id 1); it is changed, never deleted.
type Branding struct {
	ID            int64      `json:"id" gorm:"primaryKey"`
	AppName       string     `json:"app_name" gorm:"type:varchar(100);not null"`
	LogoPath      string     `json:"-" gorm:"type:varchar(255)"`
	LogoUpdatedAt *time.Time `json:"logo_updated_at"`
	UpdatedBy     *int64     `json:"updated_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
