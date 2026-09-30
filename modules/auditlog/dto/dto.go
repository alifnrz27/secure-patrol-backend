package dto

import (
	"regexp"
	"secure-patrol-backend/models"
)

// Deleted accounts get their email renamed to free it (deleted_<id>_<time>_<email>).
var releasedEmail = regexp.MustCompile(`^deleted_\d+_\d+_`)

// OriginalEmail returns the email a deleted account had before it was released.
func OriginalEmail(email string) string {
	return releasedEmail.ReplaceAllString(email, "")
}

type AuditLogDto interface {
	ToAuditLogDTOs(logs []models.AuditLog) []AuditLogDTO
}

type dto struct{}

func NewAuditLogDto() AuditLogDto {
	return &dto{}
}

func (d *dto) ToAuditLogDTOs(logs []models.AuditLog) []AuditLogDTO {
	result := make([]AuditLogDTO, 0, len(logs))
	for _, log := range logs {
		var user *AuditUserDTO
		if log.UserID != nil {
			user = &AuditUserDTO{ID: *log.UserID, RoleCode: log.RoleCode}
			if log.User != nil {
				user.Name = log.User.Name
				user.Email = OriginalEmail(log.User.Email)
			}
		}

		result = append(result, AuditLogDTO{
			ID:          log.ID,
			Action:      log.Action,
			Resource:    log.Resource,
			ResourceID:  log.ResourceID,
			Endpoint:    log.Endpoint,
			Method:      log.Method,
			Path:        log.Path,
			StatusCode:  log.StatusCode,
			User:        user,
			AppPlatform: log.AppPlatform,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			Source:      log.Source,
			CreatedAt:   log.CreatedAt,
		})
	}
	return result
}
