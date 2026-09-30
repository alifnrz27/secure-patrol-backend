package dto

import "time"

type RoleDTO struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateRoleRequest struct {
	Code        string `json:"code" validate:"required,min=3,max=50"`
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
	IsActive    *bool  `json:"is_active" validate:"required"`
}
