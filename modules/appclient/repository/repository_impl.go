package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewAppClientRepository(db *gorm.DB) AppClientRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(pagination helper.Pagination) (clients []models.AppClient, total int64, err error) {
	query := r.db.Model(&models.AppClient{})

	if pagination.Search != "" {
		like := "%" + pagination.Search + "%"
		query = query.Where("name ILIKE ? OR app_id ILIKE ?", like, like)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		Order("id DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&clients).Error

	return clients, total, err
}

func (r *repository) FindByID(id int64) (client models.AppClient, err error) {
	err = r.db.First(&client, id).Error
	return client, err
}

func (r *repository) FindByAppID(appID string) (client models.AppClient, err error) {
	err = r.db.Where("app_id = ?", appID).First(&client).Error
	return client, err
}

func (r *repository) Create(client *models.AppClient) error {
	return r.db.Create(client).Error
}

func (r *repository) Update(client *models.AppClient) error {
	return r.db.Model(client).
		Select("name", "description", "is_active", "expires_at").
		Updates(client).Error
}

func (r *repository) UpdateKeys(client *models.AppClient) error {
	return r.db.Model(client).
		Select("key_encrypted", "key_hint", "previous_key_encrypted", "previous_key_expires_at", "key_rotated_at").
		Updates(client).Error
}

func (r *repository) Delete(id int64) error {
	return r.db.Delete(&models.AppClient{}, id).Error
}

// TouchLastUsed updates last_used_at at most once per minute to avoid a write on every request.
func (r *repository) TouchLastUsed(id int64, at time.Time) error {
	return r.db.Model(&models.AppClient{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, at.Add(-time.Minute)).
		Update("last_used_at", at).Error
}
