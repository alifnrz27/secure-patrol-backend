package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

type PatrolPointRepository interface {
	// FindAll lists the points of a unit (0 = every unit).
	FindAll(pagination helper.Pagination, unitID int64, areaID int64) ([]models.PatrolPoint, int64, error)
	FindArea(id int64) (models.PatrolArea, error)
	FindByID(id int64) (models.PatrolPoint, error)
	FindByNFCCode(nfcCode string) (models.PatrolPoint, error)
	Create(point *models.PatrolPoint) error
	Update(point *models.PatrolPoint) error
	Delete(point models.PatrolPoint) error
}
