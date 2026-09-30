package routes

import (
	"secure-patrol-backend/modules/role/dto"
	"secure-patrol-backend/modules/role/http"
	"secure-patrol-backend/modules/role/repository"
	"secure-patrol-backend/modules/role/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RoleRouter(app *fiber.App, db *gorm.DB) {
	roleRepo := repository.NewRoleRepository(db)
	roleService := service.NewRoleService(roleRepo)
	roleDto := dto.NewRoleDto()
	roleHandler := http.NewRoleHandler(roleService, roleDto)

	http.RoleRoutes(app, roleHandler)
}
