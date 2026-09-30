package service

import (
	"errors"
	"fmt"
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

func (s *service) GetShifts() ([]models.PatrolShift, error) {
	return s.repo.FindAll()
}

func (s *service) GetShiftByID(id int64) (models.PatrolShift, error) {
	shift, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shift, ErrShiftNotFound
	}
	return shift, err
}

// Changes only affect patrol groups created afterwards; existing groups keep
// the start and end they were created with.
func (s *service) CreateShift(shift models.PatrolShift, actorID int64) (models.PatrolShift, error) {
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

func (s *service) UpdateShift(id int64, input models.PatrolShift, actorID int64) (models.PatrolShift, error) {
	shift, err := s.GetShiftByID(id)
	if err != nil {
		return shift, err
	}

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

	return s.GetShiftByID(shift.ID)
}

func (s *service) DeleteShift(id int64) error {
	if _, err := s.GetShiftByID(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// validate checks the time format and that active shifts never overlap, so
// every moment belongs to at most one shift.
func (s *service) validate(shift models.PatrolShift) error {
	if _, _, err := Bounds(shift); err != nil {
		return err
	}

	if !shift.IsActive {
		return nil
	}

	others, err := s.repo.FindAll()
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
