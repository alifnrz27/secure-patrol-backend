package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

var (
	ErrPatrolPointNotFound = errors.New("patrol point not found")
	ErrNFCCodeTaken        = errors.New("nfc code is already used by another patrol point")
)

type PatrolPointService interface {
	GetPatrolPoints(pagination helper.Pagination) ([]models.PatrolPoint, int64, error)
	GetPatrolPointByID(id int64) (models.PatrolPoint, error)
	GetPatrolPointByNFCCode(nfcCode string) (models.PatrolPoint, error)
	CreatePatrolPoint(point models.PatrolPoint, actorID int64) (models.PatrolPoint, error)
	UpdatePatrolPoint(id int64, input models.PatrolPoint, actorID int64) (models.PatrolPoint, error)
	DeletePatrolPoint(id int64) error
}
