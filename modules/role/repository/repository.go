package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

type RoleRepository interface {
	FindAll(pagination helper.Pagination) ([]models.Role, int64, error)
	FindByID(id int64) (models.Role, error)
	FindByCode(code string) (models.Role, error)
	Create(role *models.Role) error
	Update(role *models.Role) error
	Delete(id int64) error
	CountUsers(roleID int64) (int64, error)
}
