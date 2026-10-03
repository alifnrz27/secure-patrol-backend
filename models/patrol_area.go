package models

import (
	"time"

	"gorm.io/gorm"
)

// PatrolArea divides the patrol points of a unit into larger areas (e.g. a
// building, a floor or a parking lot). A point belongs to at most one area.
type PatrolArea struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	UnitID      int64          `json:"unit_id" gorm:"not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:varchar(255)"`
	CreatedBy   *int64         `json:"created_by"`
	UpdatedBy   *int64         `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Filled by queries only.
	PatrolPointsCount int64 `json:"-" gorm:"->;-:migration"`
}
