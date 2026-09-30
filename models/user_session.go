package models

import (
	"time"

	"gorm.io/gorm"
)

// UserSession is one login of a user from one app client.
// The refresh token secret is only stored as a SHA-256 hash.
type UserSession struct {
	ID               string         `json:"id" gorm:"type:varchar(32);primaryKey"`
	UserID           int64          `json:"user_id" gorm:"not null;index"`
	AppClientID      int64          `json:"app_client_id" gorm:"not null;index"`
	RefreshTokenHash string         `json:"-" gorm:"type:varchar(64);not null"`
	UserAgent        string         `json:"user_agent" gorm:"type:varchar(255)"`
	IPAddress        string         `json:"ip_address" gorm:"type:varchar(45)"`
	ExpiresAt        time.Time      `json:"expires_at" gorm:"not null"`
	LastUsedAt       time.Time      `json:"last_used_at"`
	RevokedAt        *time.Time     `json:"revoked_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (s UserSession) IsValid(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now)
}
