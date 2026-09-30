package routes

import (
	"secure-patrol-backend/docs"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

// DocsRouter registers the API documentation only when APP_ENV=development.
// In any other environment the routes do not exist and return 404.
func DocsRouter(app *fiber.App) {
	if !helper.IsDevelopment() {
		return
	}

	docs.Routes(app)
	log.Info("API documentation is enabled at /docs (APP_ENV=development)")
}
