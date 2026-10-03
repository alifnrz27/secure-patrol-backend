package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolarea/repository"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	repo repository.PatrolAreaRepository
}

func NewPatrolAreaService(repo repository.PatrolAreaRepository) PatrolAreaService {
	return &service{repo: repo}
}

func (s *service) GetAreas(scope helper.Scope, pagination helper.Pagination, unitID int64) ([]models.PatrolArea, int64, error) {
	return s.repo.FindAll(pagination, scope.UnitFilter(unitID))
}

func (s *service) GetAreaByID(scope helper.Scope, id int64) (models.PatrolArea, error) {
	area, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return area, ErrAreaNotFound
	}
	if err != nil {
		return area, err
	}
	if !scope.CanAccessUnit(area.UnitID) {
		return models.PatrolArea{}, ErrAreaNotFound
	}
	return area, nil
}

func (s *service) CreateArea(scope helper.Scope, area models.PatrolArea) (models.PatrolArea, error) {
	if scope.UnitID == nil {
		return area, ErrUnitRequired
	}
	area.UnitID = *scope.UnitID
	area.Name = strings.TrimSpace(area.Name)
	area.Description = strings.TrimSpace(area.Description)

	if err := s.checkNameAvailable(area.UnitID, area.Name, 0); err != nil {
		return area, err
	}

	area.CreatedBy = &scope.UserID
	area.UpdatedBy = &scope.UserID
	if err := s.repo.Create(&area); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return area, ErrAreaNameTaken
		}
		return area, err
	}
	return s.GetAreaByID(scope, area.ID)
}

func (s *service) UpdateArea(scope helper.Scope, id int64, input models.PatrolArea) (models.PatrolArea, error) {
	if scope.UnitID == nil {
		return models.PatrolArea{}, ErrUnitRequired
	}
	area, err := s.GetAreaByID(scope, id)
	if err != nil {
		return area, err
	}

	name := strings.TrimSpace(input.Name)
	if err := s.checkNameAvailable(area.UnitID, name, area.ID); err != nil {
		return area, err
	}

	area.Name = name
	area.Description = strings.TrimSpace(input.Description)
	area.UpdatedBy = &scope.UserID
	if err := s.repo.Update(&area); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return area, ErrAreaNameTaken
		}
		return area, err
	}
	return s.GetAreaByID(scope, area.ID)
}

// DeleteArea only deletes an area without patrol points, so no point silently
// loses its area.
func (s *service) DeleteArea(scope helper.Scope, id int64) error {
	if scope.UnitID == nil {
		return ErrUnitRequired
	}
	area, err := s.GetAreaByID(scope, id)
	if err != nil {
		return err
	}
	count, err := s.repo.CountPoints(area.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrAreaInUse
	}
	return s.repo.Delete(area.ID)
}

func (s *service) checkNameAvailable(unitID int64, name string, exceptID int64) error {
	existing, err := s.repo.FindByName(unitID, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != exceptID {
		return ErrAreaNameTaken
	}
	return nil
}
