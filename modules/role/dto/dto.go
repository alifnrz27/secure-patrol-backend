package dto

import "secure-patrol-backend/models"

type RoleDto interface {
	ToRoleDTO(role models.Role) RoleDTO
	ToRoleDTOs(roles []models.Role) []RoleDTO
}

type dto struct{}

func NewRoleDto() RoleDto {
	return &dto{}
}

func (d *dto) ToRoleDTO(role models.Role) RoleDTO {
	return RoleDTO{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsActive:    role.IsActive,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func (d *dto) ToRoleDTOs(roles []models.Role) []RoleDTO {
	result := make([]RoleDTO, 0, len(roles))
	for _, role := range roles {
		result = append(result, d.ToRoleDTO(role))
	}
	return result
}
