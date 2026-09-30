package routes

import (
	rolerepository "secure-patrol-backend/modules/role/repository"
	"secure-patrol-backend/modules/user/dto"
	"secure-patrol-backend/modules/user/http"
	"secure-patrol-backend/modules/user/repository"
	"secure-patrol-backend/modules/user/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserRouter(app *fiber.App, db *gorm.DB) {
	userRepo := repository.NewUserRepository(db)
	roleRepo := rolerepository.NewRoleRepository(db)

	userService := service.NewUserService(userRepo, roleRepo)
	userDto := dto.NewUserDto()
	userHandler := http.NewUserHandler(userService, userDto)

	http.UserRoutes(app, userHandler)
}
