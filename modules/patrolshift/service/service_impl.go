package service

import (
	"errors"
	"fmt"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolshift/repository"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	repo repository.PatrolShiftRepository
}

func NewPatrolShiftService(repo repository.PatrolShiftRepository) PatrolShiftService {
	return &service{repo: repo}
}

func (s *service) GetShifts(scope helper.Scope, unitID int64) ([]models.PatrolShift, error) {
	return s.repo.FindAll(scope.UnitFilter(unitID))
}

func (s *service) GetShiftByID(scope helper.Scope, id int64) (models.PatrolShift, error) {
	shift, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shift, ErrShiftNotFound
	}
	if err != nil {
		return shift, err
	}
	if !scope.CanAccessUnit(shift.UnitID) {
		return models.PatrolShift{}, ErrShiftNotFound
	}
	return shift, nil
}

// Changes only affect patrol groups created afterwards; existing groups keep
// the start and end they were created with.
func (s *service) CreateShift(scope helper.Scope, shift models.PatrolShift) (models.PatrolShift, error) {
	if scope.UnitID == nil {
		return shift, ErrUnitRequired
	}
	actorID := scope.UserID
	shift.UnitID = *scope.UnitID
	shift.Name = strings.TrimSpace(shift.Name)
	shift.StartTime = strings.TrimSpace(shift.StartTime)
	shift.EndTime = strings.TrimSpace(shift.EndTime)

	if err := s.validate(shift); err != nil {
		return shift, err
	}

	shift.CreatedBy = &actorID
	shift.UpdatedBy = &actorID

	if err := s.repo.Create(&shift); err != nil {
		return shift, err
	}

	return shift, nil
}

func (s *service) UpdateShift(scope helper.Scope, id int64, input models.PatrolShift) (models.PatrolShift, error) {
	if scope.UnitID == nil {
		return models.PatrolShift{}, ErrUnitRequired
	}
	shift, err := s.GetShiftByID(scope, id)
	if err != nil {
		return shift, err
	}
	actorID := scope.UserID

	shift.Name = strings.TrimSpace(input.Name)
	shift.StartTime = strings.TrimSpace(input.StartTime)
	shift.EndTime = strings.TrimSpace(input.EndTime)
	shift.IsActive = input.IsActive
	shift.UpdatedBy = &actorID

	if err := s.validate(shift); err != nil {
		return shift, err
	}

	if err := s.repo.Update(&shift); err != nil {
		return shift, err
	}

	return s.GetShiftByID(scope, shift.ID)
}

func (s *service) DeleteShift(scope helper.Scope, id int64) error {
	if scope.UnitID == nil {
		return ErrUnitRequired
	}
	if _, err := s.GetShiftByID(scope, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// validate checks the time format and that active shifts of the same unit never
// overlap, so every moment belongs to at most one shift of a unit.
func (s *service) validate(shift models.PatrolShift) error {
	if _, _, err := Bounds(shift); err != nil {
		return err
	}

	if !shift.IsActive {
		return nil
	}

	others, err := s.repo.FindAll(shift.UnitID)
	if err != nil {
		return err
	}

	for _, other := range others {
		if other.ID == shift.ID || !other.IsActive {
			continue
		}

		overlaps, err := Overlaps(shift, other)
		if err != nil {
			continue
		}
		if overlaps {
			return fmt.Errorf("%w: %s (%s-%s)", ErrShiftOverlap, other.Name, other.StartTime, other.EndTime)
		}
	}

	return nil
}
