package routes

import (
	"secure-patrol-backend/modules/auditlog/dto"
	"secure-patrol-backend/modules/auditlog/http"
	"secure-patrol-backend/modules/auditlog/repository"
	"secure-patrol-backend/modules/auditlog/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuditLogRouter(app *fiber.App, db *gorm.DB) {
	auditLogRepo := repository.NewAuditLogRepository(db)
	auditLogService := service.NewAuditLogService(auditLogRepo)
	auditLogDto := dto.NewAuditLogDto()
	auditLogHandler := http.NewAuditLogHandler(auditLogService, auditLogDto)

	http.AuditLogRoutes(app, auditLogHandler)
}
