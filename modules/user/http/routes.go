package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App, handler *UserHandler) {
	// User management is only available from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(models.RoleSuperAdmin, models.RoleSecurityManager, models.RoleSecurityAdmin)

	app.Get("/users", webOnly, canManage, handler.GetUsers)
	app.Get("/users/:id", webOnly, canManage, handler.GetUser)
	app.Get("/users/:id/face-photo", webOnly, canManage, handler.GetFacePhoto)
	app.Post("/users", webOnly, canManage, handler.CreateUser)
	app.Put("/users/:id", webOnly, canManage, handler.UpdateUser)
	app.Put("/users/:id/reset-password", webOnly, canManage, handler.ResetPassword)
	app.Delete("/users/:id", webOnly, canManage, handler.DeleteUser)
}
