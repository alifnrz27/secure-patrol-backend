package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolpoint/repository"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	repo repository.PatrolPointRepository
}

func NewPatrolPointService(repo repository.PatrolPointRepository) PatrolPointService {
	return &service{repo: repo}
}

func (s *service) GetPatrolPoints(scope helper.Scope, pagination helper.Pagination, unitID int64) ([]models.PatrolPoint, int64, error) {
	return s.repo.FindAll(pagination, scope.UnitFilter(unitID))
}

func (s *service) GetPatrolPointByID(scope helper.Scope, id int64) (models.PatrolPoint, error) {
	point, err := s.repo.FindByID(id)
	return scoped(scope, point, err)
}

func (s *service) GetPatrolPointByNFCCode(scope helper.Scope, nfcCode string) (models.PatrolPoint, error) {
	point, err := s.repo.FindByNFCCode(NormalizeNFCCode(nfcCode))
	return scoped(scope, point, err)
}

// scoped hides points of other units, so they look like they do not exist.
func scoped(scope helper.Scope, point models.PatrolPoint, err error) (models.PatrolPoint, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return point, ErrPatrolPointNotFound
	}
	if err != nil {
		return point, err
	}
	if !scope.CanAccessUnit(point.UnitID) {
		return models.PatrolPoint{}, ErrPatrolPointNotFound
	}
	return point, nil
}

func (s *service) CreatePatrolPoint(scope helper.Scope, point models.PatrolPoint) (models.PatrolPoint, error) {
	if scope.UnitID == nil {
		return point, ErrUnitRequired
	}
	actorID := scope.UserID
	point.UnitID = *scope.UnitID
	point.Name = strings.TrimSpace(point.Name)
	point.Location = strings.TrimSpace(point.Location)
	point.NFCCode = NormalizeNFCCode(point.NFCCode)

	if err := s.checkNFCCodeAvailable(point.NFCCode, 0); err != nil {
		return point, err
	}

	point.CreatedBy = &actorID
	point.UpdatedBy = &actorID

	if err := s.repo.Create(&point); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return point, ErrNFCCodeTaken
		}
		return point, err
	}

	return point, nil
}

func (s *service) UpdatePatrolPoint(scope helper.Scope, id int64, input models.PatrolPoint) (models.PatrolPoint, error) {
	if scope.UnitID == nil {
		return models.PatrolPoint{}, ErrUnitRequired
	}
	point, err := s.GetPatrolPointByID(scope, id)
	if err != nil {
		return point, err
	}
	actorID := scope.UserID

	nfcCode := NormalizeNFCCode(input.NFCCode)
	if nfcCode != point.NFCCode {
		if err := s.checkNFCCodeAvailable(nfcCode, point.ID); err != nil {
			return point, err
		}
	}

	point.Name = strings.TrimSpace(input.Name)
	point.Location = strings.TrimSpace(input.Location)
	point.NFCCode = nfcCode
	point.Latitude = input.Latitude
	point.Longitude = input.Longitude
	point.IsLocationMatchRequired = input.IsLocationMatchRequired
	point.IsFaceValidationRequired = input.IsFaceValidationRequired
	point.UpdatedBy = &actorID

	if err := s.repo.Update(&point); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return point, ErrNFCCodeTaken
		}
		return point, err
	}

	return s.GetPatrolPointByID(scope, point.ID)
}

func (s *service) DeletePatrolPoint(scope helper.Scope, id int64) error {
	if scope.UnitID == nil {
		return ErrUnitRequired
	}
	point, err := s.GetPatrolPointByID(scope, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(point)
}

func (s *service) checkNFCCodeAvailable(nfcCode string, exceptID int64) error {
	existing, err := s.repo.FindByNFCCode(nfcCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != exceptID {
		return ErrNFCCodeTaken
	}
	return nil
}

// NormalizeNFCCode trims and upper-cases the code, because NFC readers report
// the same tag UID in different letter cases (e.g. "04:a2:1f" vs "04:A2:1F").
func NormalizeNFCCode(nfcCode string) string {
	return strings.ToUpper(strings.TrimSpace(nfcCode))
}
