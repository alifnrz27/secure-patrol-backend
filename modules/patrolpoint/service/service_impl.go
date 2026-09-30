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

func (s *service) GetPatrolPoints(pagination helper.Pagination) ([]models.PatrolPoint, int64, error) {
	return s.repo.FindAll(pagination)
}

func (s *service) GetPatrolPointByID(id int64) (models.PatrolPoint, error) {
	point, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return point, ErrPatrolPointNotFound
	}
	return point, err
}

func (s *service) GetPatrolPointByNFCCode(nfcCode string) (models.PatrolPoint, error) {
	point, err := s.repo.FindByNFCCode(NormalizeNFCCode(nfcCode))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return point, ErrPatrolPointNotFound
	}
	return point, err
}

func (s *service) CreatePatrolPoint(point models.PatrolPoint, actorID int64) (models.PatrolPoint, error) {
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

func (s *service) UpdatePatrolPoint(id int64, input models.PatrolPoint, actorID int64) (models.PatrolPoint, error) {
	point, err := s.GetPatrolPointByID(id)
	if err != nil {
		return point, err
	}

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

	return s.GetPatrolPointByID(point.ID)
}

func (s *service) DeletePatrolPoint(id int64) error {
	point, err := s.GetPatrolPointByID(id)
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
