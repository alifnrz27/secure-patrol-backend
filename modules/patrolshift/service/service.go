package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

var (
	ErrShiftNotFound = errors.New("patrol shift not found")
	// ErrUnitRequired is returned when a head office user tries to manage shifts;
	// every unit manages its own shifts.
	ErrUnitRequired = errors.New("shifts are managed by each unit")
	ErrShiftOverlap = errors.New("patrol shift overlaps with another active shift")
)

// Every method only reaches the shifts the scope may see: unit users their own
// unit, head office users every unit.
type PatrolShiftService interface {
	// GetShifts lists the shifts of a unit (0 = every unit the scope may see).
	GetShifts(scope helper.Scope, unitID int64) ([]models.PatrolShift, error)
	GetShiftByID(scope helper.Scope, id int64) (models.PatrolShift, error)
	// CreateShift creates the shift in the unit of the scope.
	CreateShift(scope helper.Scope, shift models.PatrolShift) (models.PatrolShift, error)
	UpdateShift(scope helper.Scope, id int64, input models.PatrolShift) (models.PatrolShift, error)
	DeleteShift(scope helper.Scope, id int64) error
}
