package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func RoleRoutes(app *fiber.App, handler *RoleHandler) {
	// Role management is only available from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	// Unit heads and admins read roles to fill the user form; only the
	// Super-Admin changes them (roles are shared by every unit).
	canRead := middleware.RequireRoles(models.WebRoles...)
	canWrite := middleware.RequireRoles(models.RoleSuperAdmin)

	app.Get("/roles", webOnly, canRead, handler.GetRoles)
	app.Get("/roles/:id", webOnly, canRead, handler.GetRole)
	app.Post("/roles", webOnly, canWrite, handler.CreateRole)
	app.Put("/roles/:id", webOnly, canWrite, handler.UpdateRole)
	app.Delete("/roles/:id", webOnly, canWrite, handler.DeleteRole)
}
