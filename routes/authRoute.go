package routes

import (
	"secure-patrol-backend/modules/auth/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRouter(app *fiber.App, db *gorm.DB) {
	http.AuthRoutes(app, newAuthHandler(db))
}
