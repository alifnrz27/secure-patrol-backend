package routes

import (
	"context"
	"secure-patrol-backend/modules/patrol/dto"
	"secure-patrol-backend/modules/patrol/http"
	"secure-patrol-backend/modules/patrol/repository"
	"secure-patrol-backend/modules/patrol/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func PatrolRouter(app *fiber.App, db *gorm.DB) {
	patrolRepo := repository.NewPatrolRepository(db)
	patrolService := service.NewPatrolService(patrolRepo)
	patrolDto := dto.NewPatrolDto()
	patrolHandler := http.NewPatrolHandler(patrolService, patrolDto)

	http.PatrolRoutes(app, patrolHandler)

	// Create patrol groups automatically when a shift starts.
	go patrolService.RunScheduler(context.Background())
	go patrolService.BackfillThumbnails(context.Background())
}
