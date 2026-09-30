package repository

import (
	"secure-patrol-backend/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &repository{db: db}
}

func (r *repository) FindAll() (settings []models.SystemSetting, err error) {
	err = r.db.Order("key ASC").Find(&settings).Error
	return settings, err
}

func (r *repository) InsertMissing(settings []models.SystemSetting) (int64, error) {
	if len(settings) == 0 {
		return 0, nil
	}
	result := r.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoNothing: true}).Create(&settings)
	return result.RowsAffected, result.Error
}

func (r *repository) Save(values map[string]string, updatedBy int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			setting := models.SystemSetting{Key: key, Value: value, UpdatedBy: &updatedBy}
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at", "deleted_at"}),
			}).Create(&setting).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}
