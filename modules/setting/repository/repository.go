package repository

import "secure-patrol-backend/models"

type SettingRepository interface {
	FindAll() ([]models.SystemSetting, error)
	// InsertMissing creates the given settings that do not exist yet; existing
	// values are never overwritten.
	InsertMissing(settings []models.SystemSetting) (int64, error)
	// Save writes several values in one transaction.
	Save(values map[string]string, updatedBy int64) error
}
