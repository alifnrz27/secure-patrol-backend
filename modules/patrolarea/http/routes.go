package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func PatrolAreaRoutes(app *fiber.App, handler *PatrolAreaHandler) {
	// Every logged in user can read the areas of their unit (the mobile app
	// groups the patrol list by area); head office users see every unit.
	app.Get("/patrol-areas", handler.GetAreas)
	app.Get("/patrol-areas/:id", handler.GetArea)

	// Each unit manages its own areas from the web (unit head and admin).
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(models.UnitManagerRoles...)

	app.Post("/patrol-areas", webOnly, canManage, handler.CreateArea)
	app.Put("/patrol-areas/:id", webOnly, canManage, handler.UpdateArea)
	app.Delete("/patrol-areas/:id", webOnly, canManage, handler.DeleteArea)
}
