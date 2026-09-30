package repository

import (
	"fmt"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/user/dto"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(filter dto.UserFilter) (users []models.User, total int64, err error) {
	query := r.db.Model(&models.User{})

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}
	if filter.RoleID > 0 {
		query = query.Where("role_id = ?", filter.RoleID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.UnitID > 0 {
		query = query.Where("unit_id = ?", filter.UnitID)
	} else if filter.HeadOfficeOnly {
		query = query.Where("unit_id IS NULL")
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		Preload("Role").Preload("Unit").
		Order("id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&users).Error

	return users, total, err
}

func (r *repository) FindByID(id int64) (user models.User, err error) {
	err = r.db.Preload("Role").Preload("Unit").First(&user, id).Error
	return user, err
}

func (r *repository) FindByEmail(email string) (user models.User, err error) {
	err = r.db.Preload("Role").Preload("Unit").Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *repository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return err
	}
	return r.db.Preload("Role").Preload("Unit").First(user, user.ID).Error
}

func (r *repository) Update(user *models.User) error {
	err := r.db.Model(user).
		Select("name", "email", "role_id", "unit_id", "is_active", "face_photo_path", "face_photo_updated_at").
		Updates(user).Error
	if err != nil {
		return err
	}
	return r.db.Preload("Role").Preload("Unit").First(user, user.ID).Error
}

func (r *repository) UpdatePassword(userID int64, passwordHash string, changedAt time.Time) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash":       passwordHash,
		"password_changed_at": changedAt,
		"failed_login_count":  0,
		"locked_until":        nil,
	}).Error
}

// Delete soft deletes the user and frees its email so it can be registered again.
func (r *repository) Delete(user models.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		releasedEmail := fmt.Sprintf("deleted_%d_%d_%s", user.ID, time.Now().Unix(), user.Email)
		if len(releasedEmail) > 191 {
			releasedEmail = releasedEmail[:191]
		}

		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
			Updates(map[string]interface{}{"email": releasedEmail, "is_active": false}).Error; err != nil {
			return err
		}

		if err := revokeSessions(tx, user.ID); err != nil {
			return err
		}

		return tx.Delete(&models.User{}, user.ID).Error
	})
}

func (r *repository) RevokeSessions(userID int64) error {
	return revokeSessions(r.db, userID)
}

func revokeSessions(db *gorm.DB, userID int64) error {
	return db.Model(&models.UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}
