package repository

import (
	"secure-patrol-backend/models"
	"time"

	"gorm.io/gorm"
)

type BrandingRepository interface {
	Find() (models.Branding, error)
	Update(appName string, logoPath string, logoUpdatedAt *time.Time, updatedBy int64) error
}

type repository struct {
	db *gorm.DB
}

func NewBrandingRepository(db *gorm.DB) BrandingRepository {
	return &repository{db: db}
}

func (r *repository) Find() (branding models.Branding, err error) {
	err = r.db.First(&branding, models.BrandingID).Error
	return branding, err
}

func (r *repository) Update(appName string, logoPath string, logoUpdatedAt *time.Time, updatedBy int64) error {
	return r.db.Model(&models.Branding{}).Where("id = ?", models.BrandingID).Updates(map[string]interface{}{
		"app_name":        appName,
		"logo_path":       logoPath,
		"logo_updated_at": logoUpdatedAt,
		"updated_by":      updatedBy,
	}).Error
}
