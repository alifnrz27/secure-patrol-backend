package routes

import (
	"secure-patrol-backend/modules/license/http"
	"secure-patrol-backend/modules/license/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func newLicenseHandler(db *gorm.DB) *http.LicenseHandler {
	licenses := service.Instance()
	if licenses == nil {
		licenses = service.NewLicenseService(db)
	}
	return http.NewLicenseHandler(licenses)
}

// LicensePublicRouter registers the license status check (app signature only).
func LicensePublicRouter(app *fiber.App, db *gorm.DB) {
	http.LicensePublicRoutes(app, newLicenseHandler(db))
}

// LicenseRouter registers the Super-Admin license page endpoints.
func LicenseRouter(app *fiber.App, db *gorm.DB) {
	http.LicenseRoutes(app, newLicenseHandler(db))
}
