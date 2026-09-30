package routes

import (
	"secure-patrol-backend/modules/appclient/dto"
	"secure-patrol-backend/modules/appclient/http"
	"secure-patrol-backend/modules/appclient/repository"
	"secure-patrol-backend/modules/appclient/service"
	"secure-patrol-backend/pkg/nonce"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AppClientRouter(app *fiber.App, db *gorm.DB) {
	appClientRepo := repository.NewAppClientRepository(db)
	appClientService := service.NewAppClientService(appClientRepo, nonce.Default())
	appClientDto := dto.NewAppClientDto()
	appClientHandler := http.NewAppClientHandler(appClientService, appClientDto)

	http.AppClientRoutes(app, appClientHandler)
}
