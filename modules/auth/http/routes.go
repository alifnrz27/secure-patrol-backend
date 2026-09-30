package http

import "github.com/gofiber/fiber/v2"

// AuthPublicRoutes only require a signed app request (no user token).
func AuthPublicRoutes(app *fiber.App, handler *AuthHandler) {
	app.Post("/auth/login", handler.Login)
	app.Post("/auth/refresh", handler.Refresh)
}

// AuthRoutes require a logged in user.
func AuthRoutes(app *fiber.App, handler *AuthHandler) {
	app.Get("/auth/me", handler.Me)
	app.Get("/app-config", handler.AppConfig)
	app.Get("/auth/me/face-photo", handler.MyFacePhoto)
	app.Post("/auth/logout", handler.Logout)
	app.Post("/auth/logout-all", handler.LogoutAll)
	app.Put("/auth/change-password", handler.ChangePassword)
}
