package repository

import (
	"secure-patrol-backend/models"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &repository{db: db}
}

func (r *repository) FindUserByEmail(email string) (user models.User, err error) {
	err = r.db.Preload("Role").Preload("Unit").Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *repository) FindUserByID(id int64) (user models.User, err error) {
	err = r.db.Preload("Role").Preload("Unit").First(&user, id).Error
	return user, err
}

func (r *repository) RecordLoginFailure(userID int64, failedCount int, lockedUntil *time.Time) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"failed_login_count": failedCount,
		"locked_until":       lockedUntil,
	}).Error
}

func (r *repository) RecordLoginSuccess(userID int64, at time.Time) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"failed_login_count": 0,
		"locked_until":       nil,
		"last_login_at":      at,
	}).Error
}

func (r *repository) UpdatePassword(userID int64, passwordHash string, changedAt time.Time) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash":       passwordHash,
		"password_changed_at": changedAt,
	}).Error
}

func (r *repository) CreateSession(session *models.UserSession) error {
	return r.db.Create(session).Error
}

func (r *repository) FindSessionByID(id string) (session models.UserSession, err error) {
	err = r.db.Where("id = ?", id).First(&session).Error
	return session, err
}

func (r *repository) RotateSessionToken(id string, refreshTokenHash string, expiresAt time.Time, usedAt time.Time) error {
	return r.db.Model(&models.UserSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"refresh_token_hash": refreshTokenHash,
		"expires_at":         expiresAt,
		"last_used_at":       usedAt,
	}).Error
}

func (r *repository) RevokeSession(id string) error {
	return r.db.Model(&models.UserSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now()).Error
}

func (r *repository) RevokeUserSessions(userID int64, exceptSessionID string) error {
	return r.db.Model(&models.UserSession{}).
		Where("user_id = ? AND id <> ? AND revoked_at IS NULL", userID, exceptSessionID).
		Update("revoked_at", time.Now()).Error
}
