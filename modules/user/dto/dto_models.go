package dto

import (
	"secure-patrol-backend/helper"
	"time"
)

type UserRoleDTO struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserUnitDTO is the unit of a unit user; null for head office users.
type UserUnitDTO struct {
	ID        int64   `json:"id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	IsActive  bool    `json:"is_active"`
}

type UserDTO struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Role         UserRoleDTO  `json:"role"`
	UnitID       *int64       `json:"unit_id"`
	Unit         *UserUnitDTO `json:"unit"`
	IsActive     bool         `json:"is_active"`
	IsLocked     bool         `json:"is_locked"`
	FacePhotoURL *string      `json:"face_photo_url"`
	// FacePhotoUpdatedAt tells the mobile app when to download the face photo again.
	FacePhotoUpdatedAt *time.Time `json:"face_photo_updated_at"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UserFilter struct {
	helper.Pagination
	RoleID   int64
	IsActive *bool
	// UnitID limits the list to one unit (0 = every unit and the head office).
	UnitID int64
	// HeadOfficeOnly limits the list to users without a unit.
	HeadOfficeOnly bool
	// HiddenRoles leaves out users with these role codes (see service.hiddenRoles).
	HiddenRoles []string
}

// CreateUserRequest is sent as multipart/form-data together with the face_photo file.
type CreateUserRequest struct {
	Name                 string `form:"name" validate:"required,max=150"`
	Email                string `form:"email" validate:"required,email,max=150"`
	Password             string `form:"password" validate:"required"`
	PasswordConfirmation string `form:"password_confirmation" validate:"required,eqfield=Password"`
	RoleID               int64  `form:"role_id" validate:"required,gt=0"`
	// UnitID is required for unit roles when a head office user creates the
	// user; unit managers always create users in their own unit.
	UnitID   int64 `form:"unit_id" validate:"omitempty,gt=0"`
	IsActive *bool `form:"is_active"`
}

// UpdateUserRequest is sent as multipart/form-data; face_photo is optional.
type UpdateUserRequest struct {
	Name     string `form:"name" validate:"required,max=150"`
	Email    string `form:"email" validate:"required,email,max=150"`
	RoleID   int64  `form:"role_id" validate:"required,gt=0"`
	UnitID   int64  `form:"unit_id" validate:"omitempty,gt=0"`
	IsActive *bool  `form:"is_active" validate:"required"`
}

type ResetPasswordRequest struct {
	Password             string `json:"password" validate:"required"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
}
