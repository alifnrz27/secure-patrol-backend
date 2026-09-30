package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func SettingRoutes(app *fiber.App, handler *SettingHandler) {
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)

	// Management roles can see the settings; only the Super-Admin can change them.
	app.Get("/settings", webOnly, middleware.RequireRoles(models.WebRoles...), handler.GetSettings)
	app.Put("/settings", webOnly, middleware.RequireRoles(models.RoleSuperAdmin), handler.UpdateSettings)
}
