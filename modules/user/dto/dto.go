package dto

import (
	"fmt"
	"secure-patrol-backend/models"
	"time"
)

type UserDto interface {
	ToUserDTO(user models.User) UserDTO
	ToUserDTOs(users []models.User) []UserDTO
}

type dto struct{}

func NewUserDto() UserDto {
	return &dto{}
}

func (d *dto) ToUserDTO(user models.User) UserDTO {
	var facePhotoURL *string
	if user.FacePhotoPath != "" {
		url := fmt.Sprintf("/api/v1/users/%d/face-photo", user.ID)
		facePhotoURL = &url
	}

	return UserDTO{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role: UserRoleDTO{
			ID:   user.Role.ID,
			Code: user.Role.Code,
			Name: user.Role.Name,
		},
		IsActive:           user.IsActive,
		IsLocked:           user.IsLocked(time.Now()),
		FacePhotoURL:       facePhotoURL,
		FacePhotoUpdatedAt: user.FacePhotoUpdatedAt,
		LastLoginAt:        user.LastLoginAt,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}
}

func (d *dto) ToUserDTOs(users []models.User) []UserDTO {
	result := make([]UserDTO, 0, len(users))
	for _, user := range users {
		result = append(result, d.ToUserDTO(user))
	}
	return result
}
