package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/service"

	"github.com/gofiber/fiber/v2"
)

func AuditLogRoutes(app *fiber.App, handler *AuditLogHandler) {
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canView := middleware.RequireRoles(service.ViewerRoles...)

	app.Get("/audit-logs", webOnly, canView, handler.GetLogs)
}
