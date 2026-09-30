package repository

import (
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewPatrolShiftRepository(db *gorm.DB) PatrolShiftRepository {
	return &repository{db: db}
}

func (r *repository) FindAll() (shifts []models.PatrolShift, err error) {
	err = r.db.Order("start_time ASC, id ASC").Find(&shifts).Error
	return shifts, err
}

func (r *repository) FindByID(id int64) (shift models.PatrolShift, err error) {
	err = r.db.First(&shift, id).Error
	return shift, err
}

func (r *repository) Create(shift *models.PatrolShift) error {
	return r.db.Create(shift).Error
}

func (r *repository) Update(shift *models.PatrolShift) error {
	return r.db.Model(shift).
		Select("name", "start_time", "end_time", "is_active", "updated_by").
		Updates(shift).Error
}

// Delete soft deletes the shift; patrol groups created from it keep their history.
func (r *repository) Delete(id int64) error {
	return r.db.Delete(&models.PatrolShift{}, id).Error
}
