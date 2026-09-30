package service

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/dto"
	"secure-patrol-backend/modules/auditlog/repository"
	"time"
)

type service struct {
	repo repository.AuditLogRepository
}

func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &service{repo: repo}
}

func truncate(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}

func (s *service) Record(log models.AuditLog) error {
	log.Path = truncate(log.Path, 255)
	log.Endpoint = truncate(log.Endpoint, 150)
	log.UserAgent = truncate(log.UserAgent, 255)
	log.IPAddress = truncate(log.IPAddress, 45)
	return s.repo.Create(&log)
}

// GetLogs filters dates on the business time zone: date_to includes the whole day.
func (s *service) GetLogs(filter dto.AuditLogFilter) ([]models.AuditLog, int64, error) {
	var from, to *time.Time
	loc := helper.AppLocation()

	if filter.DateFrom != "" {
		day, _ := time.ParseInLocation("2006-01-02", filter.DateFrom, loc)
		from = &day
	}
	if filter.DateTo != "" {
		day, _ := time.ParseInLocation("2006-01-02", filter.DateTo, loc)
		next := day.AddDate(0, 0, 1)
		to = &next
	}

	return s.repo.FindAll(filter, from, to)
}
