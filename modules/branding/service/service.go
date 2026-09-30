package service

import (
	"errors"
	"mime/multipart"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/branding/repository"
	"strings"
	"time"
)

// MaxLogoSize keeps the logo small, because it is sent as base64 to every app on start.
const MaxLogoSize = 1024 * 1024 // 1 MB

const logoDir = "branding"

var (
	ErrLogoTooLarge   = errors.New("logo size must not exceed 1 MB")
	ErrAppNameMissing = errors.New("app_name must not be empty")
)

// UpdateInput changes the branding. A nil Logo keeps the current logo unless
// RemoveLogo is set, which brings back the apps' built-in logo.
type UpdateInput struct {
	AppName    string
	Logo       *multipart.FileHeader
	RemoveLogo bool
}

type BrandingService interface {
	Get() (models.Branding, error)
	Update(input UpdateInput, actorID int64) (models.Branding, error)
}

type service struct {
	repo repository.BrandingRepository
}

func NewBrandingService(repo repository.BrandingRepository) BrandingService {
	return &service{repo: repo}
}

func (s *service) Get() (models.Branding, error) {
	return s.repo.Find()
}

func (s *service) Update(input UpdateInput, actorID int64) (models.Branding, error) {
	appName := strings.TrimSpace(input.AppName)
	if appName == "" {
		return models.Branding{}, ErrAppNameMissing
	}

	current, err := s.repo.Find()
	if err != nil {
		return current, err
	}

	logoPath, logoUpdatedAt := current.LogoPath, current.LogoUpdatedAt
	now := time.Now()
	switch {
	case input.Logo != nil:
		if input.Logo.Size > MaxLogoSize {
			return current, ErrLogoTooLarge
		}
		data, ext, err := helper.ReadImage(input.Logo)
		if err != nil {
			return current, err
		}
		if len(data) > MaxLogoSize {
			return current, ErrLogoTooLarge
		}
		if logoPath, err = helper.SaveImageBytes(data, ext, logoDir); err != nil {
			return current, err
		}
		logoUpdatedAt = &now
	case input.RemoveLogo && current.LogoPath != "":
		logoPath, logoUpdatedAt = "", &now
	}

	if err := s.repo.Update(appName, logoPath, logoUpdatedAt, actorID); err != nil {
		if logoPath != current.LogoPath {
			helper.DeleteStoredFile(logoPath)
		}
		return current, err
	}
	if current.LogoPath != "" && logoPath != current.LogoPath {
		helper.DeleteStoredFile(current.LogoPath)
	}

	return s.repo.Find()
}
