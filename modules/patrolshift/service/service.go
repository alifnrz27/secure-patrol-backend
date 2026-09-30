package service

import (
	"errors"
	"secure-patrol-backend/models"
)

var (
	ErrShiftNotFound = errors.New("patrol shift not found")
	ErrShiftOverlap  = errors.New("patrol shift overlaps with another active shift")
)

type PatrolShiftService interface {
	GetShifts() ([]models.PatrolShift, error)
	GetShiftByID(id int64) (models.PatrolShift, error)
	CreateShift(shift models.PatrolShift, actorID int64) (models.PatrolShift, error)
	UpdateShift(id int64, input models.PatrolShift, actorID int64) (models.PatrolShift, error)
	DeleteShift(id int64) error
}
