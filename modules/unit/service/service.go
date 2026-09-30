package service

import (
	"errors"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/unit/repository"
)

var (
	ErrUnitNotFound  = errors.New("unit not found")
	ErrUnitCodeTaken = errors.New("unit code is already used by another unit")
	ErrUnitInUse     = errors.New("unit still has users or patrol points, move or delete them first")
)

type UnitService interface {
	GetUnits(filter repository.UnitFilter) ([]models.Unit, int64, error)
	GetUnitByID(id int64) (models.Unit, error)
	CreateUnit(unit models.Unit, actorID int64) (models.Unit, error)
	UpdateUnit(id int64, input models.Unit, actorID int64) (models.Unit, error)
	DeleteUnit(id int64) error
}
