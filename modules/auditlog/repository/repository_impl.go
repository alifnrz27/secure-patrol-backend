package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/dto"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &repository{db: db}
}

func (r *repository) Create(log *models.AuditLog) error {
	return r.db.Omit("User").Create(log).Error
}

func (r *repository) FindAll(filter dto.AuditLogFilter, from *time.Time, to *time.Time) (logs []models.AuditLog, total int64, err error) {
	query := r.db.Model(&models.AuditLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.UnitID > 0 {
		query = query.Where("unit_id = ?", filter.UnitID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Resource != "" {
		query = query.Where("resource = ?", filter.Resource)
	}
	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at < ?", *to)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("path ILIKE ? OR ip_address ILIKE ? OR resource_id = ?", like, like, filter.Search)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		// Deleted users stay visible in the log.
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at DESC, id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&logs).Error

	return logs, total, err
}
