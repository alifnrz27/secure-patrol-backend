package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

var (
	ErrAreaNotFound  = errors.New("patrol area not found")
	ErrAreaNameTaken = errors.New("an area with this name already exists in the unit")
	ErrAreaInUse     = errors.New("area still has patrol points, move them to another area first")
	// ErrUnitRequired is returned when a head office user tries to manage areas;
	// every unit manages its own areas.
	ErrUnitRequired = errors.New("patrol areas are managed by each unit")
)

// Every method only reaches the areas the scope may see: unit users their own
// unit, head office users every unit.
type PatrolAreaService interface {
	GetAreas(scope helper.Scope, pagination helper.Pagination, unitID int64) ([]models.PatrolArea, int64, error)
	GetAreaByID(scope helper.Scope, id int64) (models.PatrolArea, error)
	CreateArea(scope helper.Scope, area models.PatrolArea) (models.PatrolArea, error)
	UpdateArea(scope helper.Scope, id int64, input models.PatrolArea) (models.PatrolArea, error)
	DeleteArea(scope helper.Scope, id int64) error
}
