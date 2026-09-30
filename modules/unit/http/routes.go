package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func UnitRoutes(app *fiber.App, handler *UnitHandler) {
	// Every logged in user can read units; unit users only see their own unit.
	app.Get("/units", handler.GetUnits)
	app.Get("/units/:id", handler.GetUnit)

	// Only the head office Super-Admin manages units, from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	superAdmin := middleware.RequireRoles(models.RoleSuperAdmin)

	app.Post("/units", webOnly, superAdmin, handler.CreateUnit)
	app.Put("/units/:id", webOnly, superAdmin, handler.UpdateUnit)
	app.Delete("/units/:id", webOnly, superAdmin, handler.DeleteUnit)
}
