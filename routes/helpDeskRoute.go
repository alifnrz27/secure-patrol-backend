package routes

import (
	"secure-patrol-backend/modules/helpdesk/dto"
	"secure-patrol-backend/modules/helpdesk/http"
	"secure-patrol-backend/modules/helpdesk/repository"
	"secure-patrol-backend/modules/helpdesk/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func HelpDeskRouter(app *fiber.App, db *gorm.DB) {
	helpDeskRepo := repository.NewHelpDeskRepository(db)
	helpDeskService := service.NewHelpDeskService(helpDeskRepo)
	helpDeskDto := dto.NewHelpDeskDto()
	helpDeskHandler := http.NewHelpDeskHandler(helpDeskService, helpDeskDto)

	http.HelpDeskRoutes(app, helpDeskHandler)
}
