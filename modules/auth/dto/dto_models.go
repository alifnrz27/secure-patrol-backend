package dto

import (
	licensehttp "secure-patrol-backend/modules/license/http"
	userdto "secure-patrol-backend/modules/user/dto"
	"time"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ChangePasswordRequest struct {
	OldPassword          string `json:"old_password" validate:"required"`
	Password             string `json:"password" validate:"required"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
}

type TokenDTO struct {
	TokenType        string       `json:"token_type"`
	AccessToken      string       `json:"access_token"`
	ExpiresIn        int64        `json:"expires_in"`
	ExpiresAt        time.Time    `json:"expires_at"`
	RefreshToken     string       `json:"refresh_token"`
	RefreshExpiresAt time.Time    `json:"refresh_expires_at"`
	User             LoginUserDTO `json:"user"`
	Config           AppConfigDTO `json:"config"`
	// Settings are the adjustable settings the apps need (radius, lockout, token lifetimes, ...).
	Settings map[string]interface{} `json:"settings"`
	// License is the license status, for banners (expiring soon, grace period).
	License *licensehttp.LicenseSummaryDTO `json:"license"`
}

// ProfileDTO is the logged in user's profile together with the app settings.
type ProfileDTO struct {
	userdto.UserDTO
	Settings map[string]interface{}         `json:"settings"`
	License  *licensehttp.LicenseSummaryDTO `json:"license"`
}

// AppConfigDTO holds the settings the mobile app needs to validate a scan
// before sending it and to work offline. ServerTime lets the app correct its
// clock, because signed requests are rejected when the device clock is off by
// more than 5 minutes.
type AppConfigDTO struct {
	ServerTime                 time.Time `json:"server_time"`
	Timezone                   string    `json:"timezone"`
	RequestTimestampToleranceS int       `json:"request_timestamp_tolerance_seconds"`
	LocationRadiusMeters       float64   `json:"location_radius_meters"`
	FaceMatchMinScore          *float64  `json:"face_match_min_score"`
	MaxOfflineHours            int       `json:"max_offline_hours"`
	MaxScanPhotos              int       `json:"max_scan_photos"`
	MaxPhotoSizeBytes          int64     `json:"max_photo_size_bytes"`
}

// LoginUserDTO is the user returned on login. The face photo is only included
// on login (not on refresh) to keep the refresh response small.
type LoginUserDTO struct {
	userdto.UserDTO
	FacePhotoBase64   *string `json:"face_photo_base64"`
	FacePhotoMimeType *string `json:"face_photo_mime_type"`
}
