package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func SettingRoutes(app *fiber.App, handler *SettingHandler) {
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)

	// Management roles can see the settings. The Super-Admin changes the global
	// values; unit heads and admins change their own unit's values. The
	// Security Manager only monitors.
	app.Get("/settings", webOnly, middleware.RequireRoles(models.WebRoles...), handler.GetSettings)
	app.Put("/settings", webOnly, middleware.RequireRoles(
		models.RoleSuperAdmin,
		models.RoleSecurityHead,
		models.RoleSecurityAdmin,
	), handler.UpdateSettings)
}
