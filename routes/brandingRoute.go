package routes

import (
	"secure-patrol-backend/modules/branding/http"
	"secure-patrol-backend/modules/branding/repository"
	"secure-patrol-backend/modules/branding/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func newBrandingHandler(db *gorm.DB) *http.BrandingHandler {
	return http.NewBrandingHandler(service.NewBrandingService(repository.NewBrandingRepository(db)))
}

// BrandingPublicRouter registers the branding read route (app signature only).
func BrandingPublicRouter(app *fiber.App, db *gorm.DB) {
	http.BrandingPublicRoutes(app, newBrandingHandler(db))
}

// BrandingRouter registers the branding update route (logged in user).
func BrandingRouter(app *fiber.App, db *gorm.DB) {
	http.BrandingRoutes(app, newBrandingHandler(db))
}
