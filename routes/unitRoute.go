package routes

import (
	"secure-patrol-backend/modules/unit/dto"
	"secure-patrol-backend/modules/unit/http"
	"secure-patrol-backend/modules/unit/repository"
	"secure-patrol-backend/modules/unit/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UnitRouter(app *fiber.App, db *gorm.DB) {
	unitRepo := repository.NewUnitRepository(db)
	unitService := service.NewUnitService(unitRepo)
	unitDto := dto.NewUnitDto()
	unitHandler := http.NewUnitHandler(unitService, unitDto)

	http.UnitRoutes(app, unitHandler)
}
