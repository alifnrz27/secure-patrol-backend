package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

// BrandingPublicRoutes only require a signed app request, because the login
// screen shows the name and logo before anyone is logged in.
func BrandingPublicRoutes(app *fiber.App, handler *BrandingHandler) {
	app.Get("/branding", handler.GetBranding)
}

// BrandingRoutes require a logged in user: only the Super-Admin changes the
// branding, from the web platform.
func BrandingRoutes(app *fiber.App, handler *BrandingHandler) {
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	superAdmin := middleware.RequireRoles(models.RoleSuperAdmin)

	app.Put("/branding", webOnly, superAdmin, handler.UpdateBranding)
}
