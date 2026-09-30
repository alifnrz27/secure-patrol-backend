package repository

import (
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &repository{db: db}
}

func (r *repository) FindGlobal() (settings []models.SystemSetting, err error) {
	err = r.db.Where("unit_id IS NULL").Order("key ASC").Find(&settings).Error
	return settings, err
}

func (r *repository) FindByUnit(unitID int64) (settings []models.SystemSetting, err error) {
	err = r.db.Where("unit_id = ?", unitID).Order("key ASC").Find(&settings).Error
	return settings, err
}

func (r *repository) InsertMissingGlobal(settings []models.SystemSetting) (int64, error) {
	var created int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range settings {
			var count int64
			if err := tx.Model(&models.SystemSetting{}).Where("unit_id IS NULL AND key = ?", setting.Key).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			setting.UnitID = nil
			if err := tx.Create(&setting).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	return created, err
}

func (r *repository) Save(unitID *int64, values map[string]string, updatedBy int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			query := tx.Model(&models.SystemSetting{}).Where("key = ?", key)
			if unitID == nil {
				query = query.Where("unit_id IS NULL")
			} else {
				query = query.Where("unit_id = ?", *unitID)
			}

			result := query.Updates(map[string]interface{}{"value": value, "updated_by": updatedBy})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				continue
			}

			setting := models.SystemSetting{UnitID: unitID, Key: key, Value: value, UpdatedBy: &updatedBy}
			if err := tx.Create(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) DeleteUnitOverrides(unitID int64, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.db.Where("unit_id = ? AND key IN ?", unitID, keys).Delete(&models.SystemSetting{}).Error
}
