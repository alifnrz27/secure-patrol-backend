package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func PatrolPointRoutes(app *fiber.App, handler *PatrolPointHandler) {
	// Every logged in user on any platform can read patrol points, so officers
	// can look up a scanned NFC tag from the mobile app.
	app.Get("/patrol-points", handler.GetPatrolPoints)
	app.Get("/patrol-points/by-nfc", handler.GetPatrolPointByNFC)
	app.Get("/patrol-points/:id", handler.GetPatrolPoint)

	// Managing patrol points is only available from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(
		models.RoleSuperAdmin,
		models.RoleSecurityManager,
		models.RoleSecurityHead,
		models.RoleSecurityAdmin,
	)

	app.Post("/patrol-points", webOnly, canManage, handler.CreatePatrolPoint)
	app.Put("/patrol-points/:id", webOnly, canManage, handler.UpdatePatrolPoint)
	app.Delete("/patrol-points/:id", webOnly, canManage, handler.DeletePatrolPoint)
}
