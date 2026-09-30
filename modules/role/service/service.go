package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
)

var (
	ErrRoleNotFound     = errors.New("role not found")
	ErrRoleCodeTaken    = errors.New("role code is already used")
	ErrRoleCodeInvalid  = errors.New("role code must be 3-50 characters of lowercase letters, digits or underscore, starting with a letter")
	ErrRoleSystemLocked = errors.New("system role cannot be deleted or deactivated")
	ErrRoleInUse        = errors.New("role is still assigned to users")
)

type RoleService interface {
	GetRoles(pagination helper.Pagination) ([]models.Role, int64, error)
	GetRoleByID(id int64) (models.Role, error)
	CreateRole(role models.Role) (models.Role, error)
	UpdateRole(id int64, input models.Role) (models.Role, error)
	DeleteRole(id int64) error
}
