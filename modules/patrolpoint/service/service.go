package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

var (
	ErrPatrolPointNotFound = errors.New("patrol point not found")
	ErrNFCCodeTaken        = errors.New("nfc code is already used by another patrol point")
	// ErrUnitRequired is returned when a head office user tries to manage patrol
	// points; every unit manages its own points.
	ErrUnitRequired = errors.New("patrol points are managed by each unit")
	ErrAreaInvalid  = errors.New("area not found in this unit")
)

// Every method only reaches the points the scope may see: unit users their own
// unit, head office users every unit. NFC codes are unique across all units.
type PatrolPointService interface {
	// GetPatrolPoints lists the points of a unit (0 = every unit the scope may see).
	// GetPatrolPoints lists the points of a unit (0 = every unit the scope may
	// see), optionally of one area.
	GetPatrolPoints(scope helper.Scope, pagination helper.Pagination, unitID int64, areaID int64) ([]models.PatrolPoint, int64, error)
	GetPatrolPointByID(scope helper.Scope, id int64) (models.PatrolPoint, error)
	GetPatrolPointByNFCCode(scope helper.Scope, nfcCode string) (models.PatrolPoint, error)
	// CreatePatrolPoint creates the point in the unit of the scope.
	CreatePatrolPoint(scope helper.Scope, point models.PatrolPoint) (models.PatrolPoint, error)
	UpdatePatrolPoint(scope helper.Scope, id int64, input models.PatrolPoint) (models.PatrolPoint, error)
	DeletePatrolPoint(scope helper.Scope, id int64) error
}
