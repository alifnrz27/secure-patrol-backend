package http

import (
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/service"

	"github.com/gofiber/fiber/v2"
)

func HelpDeskRoutes(app *fiber.App, handler *HelpDeskHandler) {
	// Every logged in user on any platform can read the help desk.
	app.Get("/help-desk-articles", handler.GetArticles)
	app.Get("/help-desk-articles/:id", handler.GetArticle)

	// The help desk is shared by every unit and managed by the head office
	// Super-Admin, from the web platform.
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	canManage := middleware.RequireRoles(service.EditorRoles...)

	app.Post("/help-desk-articles", webOnly, canManage, handler.CreateArticle)
	// Registered before "/:id" so "reorder" is not taken as an article id.
	app.Put("/help-desk-articles/reorder", webOnly, canManage, handler.ReorderArticles)
	app.Put("/help-desk-articles/:id", webOnly, canManage, handler.UpdateArticle)
	app.Delete("/help-desk-articles/:id", webOnly, canManage, handler.DeleteArticle)
}
