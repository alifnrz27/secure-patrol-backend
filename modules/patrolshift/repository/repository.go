package repository

import "secure-patrol-backend/models"

type PatrolShiftRepository interface {
	FindAll() ([]models.PatrolShift, error)
	FindByID(id int64) (models.PatrolShift, error)
	Create(shift *models.PatrolShift) error
	Update(shift *models.PatrolShift) error
	Delete(id int64) error
}
