package dto

import "time"

type AppClientDTO struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	Platform             string     `json:"platform"`
	AppID                string     `json:"app_id"`
	KeyHint              string     `json:"key_hint"`
	Description          string     `json:"description"`
	IsActive             bool       `json:"is_active"`
	ExpiresAt            *time.Time `json:"expires_at"`
	LastUsedAt           *time.Time `json:"last_used_at"`
	KeyRotatedAt         *time.Time `json:"key_rotated_at"`
	PreviousKeyExpiresAt *time.Time `json:"previous_key_expires_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// AppClientCredentialDTO is only returned on create and key rotation.
type AppClientCredentialDTO struct {
	AppClientDTO
	AppKey  string `json:"app_key"`
	Warning string `json:"warning"`
}

type CreateAppClientRequest struct {
	Name        string     `json:"name" validate:"required,max=100"`
	Platform    string     `json:"platform" validate:"required,oneof=android ios web server"`
	Description string     `json:"description" validate:"max=255"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type UpdateAppClientRequest struct {
	Name        string     `json:"name" validate:"required,max=100"`
	Description string     `json:"description" validate:"max=255"`
	IsActive    *bool      `json:"is_active" validate:"required"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type RotateKeyRequest struct {
	// How long the old key keeps working, 0-168 hours (7 days).
	GracePeriodHours int `json:"grace_period_hours" validate:"min=0,max=168"`
}
