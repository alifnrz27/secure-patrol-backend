package routes

import (
	"secure-patrol-backend/modules/auth/dto"
	"secure-patrol-backend/modules/auth/http"
	"secure-patrol-backend/modules/auth/repository"
	"secure-patrol-backend/modules/auth/service"
	userdto "secure-patrol-backend/modules/user/dto"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PublicRouter registers routes that only need a signed app request (no user login).
func PublicRouter(app *fiber.App, db *gorm.DB) {
	authHandler := newAuthHandler(db)

	http.AuthPublicRoutes(app, authHandler)
}

func newAuthHandler(db *gorm.DB) *http.AuthHandler {
	authRepo := repository.NewAuthRepository(db)
	authService := service.NewAuthService(authRepo)

	return http.NewAuthHandler(authService, dto.NewAuthDto(), userdto.NewUserDto())
}
