package models

import (
	"time"

	"gorm.io/gorm"
)

// SystemSetting is one adjustable setting, stored as text. The allowed type,
// range and default of every key are defined in code (modules/setting).
type SystemSetting struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	Key       string         `json:"key" gorm:"type:varchar(100);uniqueIndex;not null"`
	Value     string         `json:"value" gorm:"type:varchar(255);not null"`
	UpdatedBy *int64         `json:"updated_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
