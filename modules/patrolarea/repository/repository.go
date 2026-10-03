package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

type PatrolAreaRepository interface {
	// FindAll lists the areas of a unit (0 = every unit) with their point count.
	FindAll(pagination helper.Pagination, unitID int64) ([]models.PatrolArea, int64, error)
	FindByID(id int64) (models.PatrolArea, error)
	// FindByName finds an area of the unit by name, ignoring letter case.
	FindByName(unitID int64, name string) (models.PatrolArea, error)
	Create(area *models.PatrolArea) error
	Update(area *models.PatrolArea) error
	Delete(id int64) error
	CountPoints(areaID int64) (int64, error)
}
