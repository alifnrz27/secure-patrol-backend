package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"time"
)

type AppClientRepository interface {
	FindAll(pagination helper.Pagination) ([]models.AppClient, int64, error)
	FindByID(id int64) (models.AppClient, error)
	FindByAppID(appID string) (models.AppClient, error)
	Create(client *models.AppClient) error
	Update(client *models.AppClient) error
	UpdateKeys(client *models.AppClient) error
	Delete(id int64) error
	TouchLastUsed(id int64, at time.Time) error
}
