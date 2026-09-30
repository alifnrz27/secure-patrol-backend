package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func PatrolRoutes(app *fiber.App, handler *PatrolHandler) {
	app.Get("/patrol-groups", handler.GetGroups)
	app.Get("/patrol-groups/current", handler.GetCurrentGroup)
	app.Get("/patrol-groups/:id", handler.GetGroup)

	app.Get("/patrol-list-items", handler.GetItems)

	// Scanning NFC tags is only available from the mobile apps.
	mobileOnly := middleware.RequirePlatforms(models.PlatformAndroid, models.PlatformIOS)
	app.Post("/patrol-scans", mobileOnly, handler.Scan)

	app.Get("/patrol-scans", handler.GetScans)
	// Registered before "/:id" so "export" is not taken as a scan id.
	app.Get("/patrol-scans/export",
		middleware.RequirePlatforms(models.PlatformWeb),
		middleware.RequireRoles(models.WebRoles...),
		handler.ExportScans,
	)
	app.Get("/patrol-scans/:id", handler.GetScan)
	app.Get("/patrol-scans/:id/photos/:photo_id", handler.GetScanPhoto)
}
