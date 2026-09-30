package service

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/dto"
)

// ViewerRoles can read the audit log.
var ViewerRoles = []string{models.RoleSuperAdmin, models.RoleSecurityManager}

type AuditLogService interface {
	Record(log models.AuditLog) error
	GetLogs(filter dto.AuditLogFilter) ([]models.AuditLog, int64, error)
}
