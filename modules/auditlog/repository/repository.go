package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/dto"
	"time"
)

type AuditLogRepository interface {
	Create(log *models.AuditLog) error
	FindAll(filter dto.AuditLogFilter, from *time.Time, to *time.Time) ([]models.AuditLog, int64, error)
}
