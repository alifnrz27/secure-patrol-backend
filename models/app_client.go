package models

import (
	"time"

	"gorm.io/gorm"
)

// Supported app client platforms.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
	PlatformServer  = "server"
)

// WebRoles are the roles allowed to sign in from the web and server platforms.
// Android and iOS apps accept every role.
var WebRoles = []string{
	RoleSuperAdmin,
	RoleSecurityManager,
	RoleSecurityHead,
	RoleSecurityAdmin,
}

// RoleCanUsePlatform reports whether a user with roleCode may sign in from an
// app client of the given platform. Unknown platforms are denied.
func RoleCanUsePlatform(roleCode string, platform string) bool {
	switch platform {
	case PlatformAndroid, PlatformIOS:
		return true
	case PlatformWeb, PlatformServer:
		for _, role := range WebRoles {
			if role == roleCode {
				return true
			}
		}
	}
	return false
}

// AppClient is an application (mobile, web, server) that is allowed to call the API.
// The app key is stored encrypted (AES-256-GCM) because the server needs the plain
// key to verify HMAC request signatures.
type AppClient struct {
	ID                   int64          `json:"id" gorm:"primaryKey"`
	Name                 string         `json:"name" gorm:"type:varchar(100);not null"`
	Platform             string         `json:"platform" gorm:"type:varchar(20);not null"`
	AppID                string         `json:"app_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	KeyEncrypted         string         `json:"-" gorm:"type:varchar(255);not null"`
	KeyHint              string         `json:"key_hint" gorm:"type:varchar(8)"`
	PreviousKeyEncrypted *string        `json:"-" gorm:"type:varchar(255)"`
	PreviousKeyExpiresAt *time.Time     `json:"previous_key_expires_at"`
	Description          string         `json:"description" gorm:"type:varchar(255)"`
	IsActive             bool           `json:"is_active" gorm:"not null"`
	ExpiresAt            *time.Time     `json:"expires_at"`
	LastUsedAt           *time.Time     `json:"last_used_at"`
	KeyRotatedAt         *time.Time     `json:"key_rotated_at"`
	CreatedBy            *int64         `json:"created_by"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}
