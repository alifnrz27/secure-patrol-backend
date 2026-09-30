package dto

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/pkg/log"
	"time"
)

// BrandingDTO is the application name and logo. A null logo means the apps show
// their built-in logo. LogoUpdatedAt changes whenever the logo changes, so the
// apps can cache the logo and only decode it again when the value differs.
type BrandingDTO struct {
	AppName       string     `json:"app_name"`
	LogoBase64    *string    `json:"logo_base64"`
	LogoMimeType  *string    `json:"logo_mime_type"`
	LogoUpdatedAt *time.Time `json:"logo_updated_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// UpdateBrandingRequest is sent as multipart/form-data with an optional "logo" file.
type UpdateBrandingRequest struct {
	AppName    string `form:"app_name" validate:"required,max=100"`
	RemoveLogo bool   `form:"remove_logo"`
}

func ToBrandingDTO(branding models.Branding) BrandingDTO {
	result := BrandingDTO{
		AppName:       branding.AppName,
		LogoUpdatedAt: branding.LogoUpdatedAt,
		UpdatedAt:     branding.UpdatedAt,
	}
	if branding.LogoPath == "" {
		return result
	}

	// A missing file must not break the apps; they fall back to their own logo.
	encoded, contentType, err := helper.ReadImageBase64(branding.LogoPath)
	if err != nil {
		log.Warnf("branding: cannot read logo: %v", err)
		return result
	}
	result.LogoBase64 = &encoded
	result.LogoMimeType = &contentType
	return result
}
