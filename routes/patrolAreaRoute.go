package routes

import (
	"secure-patrol-backend/modules/patrolarea/http"
	"secure-patrol-backend/modules/patrolarea/repository"
	"secure-patrol-backend/modules/patrolarea/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func PatrolAreaRouter(app *fiber.App, db *gorm.DB) {
	areaService := service.NewPatrolAreaService(repository.NewPatrolAreaRepository(db))
	http.PatrolAreaRoutes(app, http.NewPatrolAreaHandler(areaService))
}
