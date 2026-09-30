package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App, handler *UserHandler) {
	// User management is only available from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	// The Security Manager only monitors; unit heads and admins manage the users
	// of their own unit.
	canView := middleware.RequireRoles(models.WebRoles...)
	canManage := middleware.RequireRoles(models.RoleSuperAdmin, models.RoleSecurityHead, models.RoleSecurityAdmin)

	app.Get("/users", webOnly, canView, handler.GetUsers)
	app.Get("/users/:id", webOnly, canView, handler.GetUser)
	app.Get("/users/:id/face-photo", webOnly, canView, handler.GetFacePhoto)
	app.Post("/users", webOnly, canManage, handler.CreateUser)
	app.Put("/users/:id", webOnly, canManage, handler.UpdateUser)
	app.Put("/users/:id/reset-password", webOnly, canManage, handler.ResetPassword)
	app.Delete("/users/:id", webOnly, canManage, handler.DeleteUser)
}
