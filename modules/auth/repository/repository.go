package repository

import (
	"secure-patrol-backend/models"
	"time"
)

type AuthRepository interface {
	FindUserByEmail(email string) (models.User, error)
	FindUserByID(id int64) (models.User, error)
	RecordLoginFailure(userID int64, failedCount int, lockedUntil *time.Time) error
	RecordLoginSuccess(userID int64, at time.Time) error
	UpdatePassword(userID int64, passwordHash string, changedAt time.Time) error

	CreateSession(session *models.UserSession) error
	FindSessionByID(id string) (models.UserSession, error)
	RotateSessionToken(id string, refreshTokenHash string, expiresAt time.Time, usedAt time.Time) error
	RevokeSession(id string) error
	RevokeUserSessions(userID int64, exceptSessionID string) error
}
