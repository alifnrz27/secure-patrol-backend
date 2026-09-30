package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func PatrolPointRoutes(app *fiber.App, handler *PatrolPointHandler) {
	// Every logged in user on any platform can read patrol points (of their own
	// unit), so officers can look up a scanned NFC tag from the mobile app.
	app.Get("/patrol-points", handler.GetPatrolPoints)
	app.Get("/patrol-points/by-nfc", handler.GetPatrolPointByNFC)
	app.Get("/patrol-points/:id", handler.GetPatrolPoint)

	// Every unit manages its own patrol points from the web (unit head and
	// admin). The head office only views them.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(models.UnitManagerRoles...)

	app.Post("/patrol-points", webOnly, canManage, handler.CreatePatrolPoint)
	app.Put("/patrol-points/:id", webOnly, canManage, handler.UpdatePatrolPoint)
	app.Delete("/patrol-points/:id", webOnly, canManage, handler.DeletePatrolPoint)
}
