package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                 int64          `json:"id" gorm:"primaryKey"`
	RoleID             int64          `json:"role_id" gorm:"not null;index"`
	Role               Role           `json:"role" gorm:"foreignKey:RoleID"`
	UnitID             *int64         `json:"unit_id" gorm:"index"`
	Unit               *Unit          `json:"unit,omitempty" gorm:"foreignKey:UnitID"`
	Name               string         `json:"name" gorm:"type:varchar(150);not null"`
	Email              string         `json:"email" gorm:"type:varchar(191);uniqueIndex;not null"`
	PasswordHash       string         `json:"-" gorm:"type:varchar(255);not null"`
	FacePhotoPath      string         `json:"-" gorm:"type:varchar(255)"`
	FacePhotoUpdatedAt *time.Time     `json:"face_photo_updated_at"`
	IsActive           bool           `json:"is_active" gorm:"not null"`
	FailedLoginCount   int            `json:"-" gorm:"not null;default:0"`
	LockedUntil        *time.Time     `json:"-"`
	LastLoginAt        *time.Time     `json:"last_login_at"`
	PasswordChangedAt  *time.Time     `json:"-"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}
