package service

import (
	"errors"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/unit/repository"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	repo repository.UnitRepository
}

func NewUnitService(repo repository.UnitRepository) UnitService {
	return &service{repo: repo}
}

func (s *service) GetUnits(filter repository.UnitFilter) ([]models.Unit, int64, error) {
	return s.repo.FindAll(filter)
}

func (s *service) GetUnitByID(id int64) (models.Unit, error) {
	unit, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return unit, ErrUnitNotFound
	}
	return unit, err
}

func (s *service) CreateUnit(unit models.Unit, actorID int64) (models.Unit, error) {
	unit.Code = NormalizeCode(unit.Code)
	unit.Name = strings.TrimSpace(unit.Name)

	if err := s.checkCodeAvailable(unit.Code, 0); err != nil {
		return unit, err
	}

	unit.CreatedBy = &actorID
	unit.UpdatedBy = &actorID

	if err := s.repo.Create(&unit); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return unit, ErrUnitCodeTaken
		}
		return unit, err
	}

	return s.GetUnitByID(unit.ID)
}

func (s *service) UpdateUnit(id int64, input models.Unit, actorID int64) (models.Unit, error) {
	unit, err := s.GetUnitByID(id)
	if err != nil {
		return unit, err
	}

	code := NormalizeCode(input.Code)
	if code != unit.Code {
		if err := s.checkCodeAvailable(code, unit.ID); err != nil {
			return unit, err
		}
	}

	unit.Code = code
	unit.Name = strings.TrimSpace(input.Name)
	unit.Latitude = input.Latitude
	unit.Longitude = input.Longitude
	unit.IsActive = input.IsActive
	unit.UpdatedBy = &actorID

	if err := s.repo.Update(&unit); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return unit, ErrUnitCodeTaken
		}
		return unit, err
	}

	return s.GetUnitByID(unit.ID)
}

// DeleteUnit only deletes an empty unit, so no user or patrol point is left
// without a unit. A unit that is only closed for a while should be deactivated.
func (s *service) DeleteUnit(id int64) error {
	unit, err := s.GetUnitByID(id)
	if err != nil {
		return err
	}

	users, err := s.repo.CountUsers(unit.ID)
	if err != nil {
		return err
	}
	points, err := s.repo.CountPatrolPoints(unit.ID)
	if err != nil {
		return err
	}
	if users > 0 || points > 0 {
		return ErrUnitInUse
	}

	return s.repo.Delete(unit)
}

func (s *service) checkCodeAvailable(code string, exceptID int64) error {
	existing, err := s.repo.FindByCode(code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != exceptID {
		return ErrUnitCodeTaken
	}
	return nil
}

// NormalizeCode trims and upper-cases a unit code, so "jkt-01" and "JKT-01" are the same unit.
func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
