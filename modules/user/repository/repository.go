package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/user/dto"
	"time"
)

type UserRepository interface {
	FindAll(filter dto.UserFilter) ([]models.User, int64, error)
	FindByID(id int64) (models.User, error)
	FindByEmail(email string) (models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	UpdatePassword(userID int64, passwordHash string, changedAt time.Time) error
	Delete(user models.User) error
	RevokeSessions(userID int64) error
}
