package routes

import (
	"secure-patrol-backend/modules/patrolshift/dto"
	"secure-patrol-backend/modules/patrolshift/http"
	"secure-patrol-backend/modules/patrolshift/repository"
	"secure-patrol-backend/modules/patrolshift/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func PatrolShiftRouter(app *fiber.App, db *gorm.DB) {
	patrolShiftRepo := repository.NewPatrolShiftRepository(db)
	patrolShiftService := service.NewPatrolShiftService(patrolShiftRepo)
	patrolShiftDto := dto.NewPatrolShiftDto()
	patrolShiftHandler := http.NewPatrolShiftHandler(patrolShiftService, patrolShiftDto)

	http.PatrolShiftRoutes(app, patrolShiftHandler)
}
