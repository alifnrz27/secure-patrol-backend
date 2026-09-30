package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func AppClientRoutes(app *fiber.App, handler *AppClientHandler) {
	// App client management is only available from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	superAdmin := middleware.RequireRoles(models.RoleSuperAdmin)

	app.Get("/app-clients", webOnly, superAdmin, handler.GetAppClients)
	app.Get("/app-clients/:id", webOnly, superAdmin, handler.GetAppClient)
	app.Post("/app-clients", webOnly, superAdmin, handler.CreateAppClient)
	app.Put("/app-clients/:id", webOnly, superAdmin, handler.UpdateAppClient)
	app.Post("/app-clients/:id/rotate-key", webOnly, superAdmin, handler.RotateKey)
	app.Delete("/app-clients/:id", webOnly, superAdmin, handler.DeleteAppClient)
}
