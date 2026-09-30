package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

type UnitFilter struct {
	helper.Pagination
	IsActive *bool
	// OnlyID limits the list to one unit (the unit of a unit user).
	OnlyID *int64
}

type UnitRepository interface {
	FindAll(filter UnitFilter) ([]models.Unit, int64, error)
	FindByID(id int64) (models.Unit, error)
	FindByCode(code string) (models.Unit, error)
	// Create stores the unit together with its default shifts.
	Create(unit *models.Unit) error
	Update(unit *models.Unit) error
	Delete(unit models.Unit) error
	CountUsers(unitID int64) (int64, error)
	CountPatrolPoints(unitID int64) (int64, error)
}
