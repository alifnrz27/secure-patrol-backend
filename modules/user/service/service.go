package service

import (
	"errors"
	"mime/multipart"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/user/dto"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrRoleInvalid        = errors.New("role not found or inactive")
	ErrFacePhotoRequired  = errors.New("face photo is required")
	ErrForbiddenRole      = errors.New("you are not allowed to manage users with this role")
	ErrCannotModifySelf   = errors.New("you cannot delete, deactivate or change the role of your own account")
	ErrFacePhotoNotExists = errors.New("user has no face photo")
	ErrUnitRequired       = errors.New("unit_id is required for this role")
	ErrUnitInvalid        = errors.New("unit not found")
)

// Actor is the logged in user performing the action.
type Actor struct {
	UserID   int64
	RoleCode string
	// UnitID is the unit of a unit manager; nil for head office users.
	UnitID *int64
}

type UserService interface {
	GetUsers(filter dto.UserFilter) ([]models.User, int64, error)
	// GetUserByID only finds users the actor may see (unit managers: their own unit).
	GetUserByID(actor Actor, id int64) (models.User, error)
	CreateUser(actor Actor, user models.User, password string, facePhoto *multipart.FileHeader) (models.User, error)
	UpdateUser(actor Actor, id int64, input models.User, facePhoto *multipart.FileHeader) (models.User, error)
	DeleteUser(actor Actor, id int64) error
	ResetPassword(actor Actor, id int64, password string) error
	GetFacePhotoPath(actor Actor, id int64) (string, error)
}
