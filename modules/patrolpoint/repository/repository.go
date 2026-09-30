package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

type PatrolPointRepository interface {
	FindAll(pagination helper.Pagination) ([]models.PatrolPoint, int64, error)
	FindByID(id int64) (models.PatrolPoint, error)
	FindByNFCCode(nfcCode string) (models.PatrolPoint, error)
	Create(point *models.PatrolPoint) error
	Update(point *models.PatrolPoint) error
	Delete(point models.PatrolPoint) error
}
