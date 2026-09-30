package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func PatrolShiftRoutes(app *fiber.App, handler *PatrolShiftHandler) {
	app.Get("/patrol-shifts", handler.GetShifts)
	app.Get("/patrol-shifts/:id", handler.GetShift)

	// Shift settings are only managed from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(
		models.RoleSuperAdmin,
		models.RoleSecurityManager,
		models.RoleSecurityHead,
		models.RoleSecurityAdmin,
	)

	app.Post("/patrol-shifts", webOnly, canManage, handler.CreateShift)
	app.Put("/patrol-shifts/:id", webOnly, canManage, handler.UpdateShift)
	app.Delete("/patrol-shifts/:id", webOnly, canManage, handler.DeleteShift)
}
