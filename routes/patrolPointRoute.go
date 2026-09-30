package routes

import (
	"secure-patrol-backend/modules/patrolpoint/dto"
	"secure-patrol-backend/modules/patrolpoint/http"
	"secure-patrol-backend/modules/patrolpoint/repository"
	"secure-patrol-backend/modules/patrolpoint/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func PatrolPointRouter(app *fiber.App, db *gorm.DB) {
	patrolPointRepo := repository.NewPatrolPointRepository(db)
	patrolPointService := service.NewPatrolPointService(patrolPointRepo)
	patrolPointDto := dto.NewPatrolPointDto()
	patrolPointHandler := http.NewPatrolPointHandler(patrolPointService, patrolPointDto)

	http.PatrolPointRoutes(app, patrolPointHandler)
}
