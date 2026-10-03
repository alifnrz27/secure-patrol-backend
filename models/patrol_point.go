package models

import (
	"time"

	"gorm.io/gorm"
)

// PatrolPoint is a checkpoint that security officers visit during a patrol,
// identified on site by its NFC tag.
type PatrolPoint struct {
	ID                       int64          `json:"id" gorm:"primaryKey"`
	UnitID                   int64          `json:"unit_id" gorm:"not null;default:0;index"`
	AreaID                   *int64         `json:"area_id" gorm:"index"`
	Area                     *PatrolArea    `json:"-" gorm:"foreignKey:AreaID"`
	Name                     string         `json:"name" gorm:"type:varchar(150);not null"`
	Location                 string         `json:"location" gorm:"type:varchar(255);not null"`
	NFCCode                  string         `json:"nfc_code" gorm:"column:nfc_code;type:varchar(100);uniqueIndex;not null"`
	Latitude                 float64        `json:"latitude" gorm:"type:double precision;not null"`
	Longitude                float64        `json:"longitude" gorm:"type:double precision;not null"`
	IsLocationMatchRequired  bool           `json:"is_location_match_required" gorm:"not null"`
	IsFaceValidationRequired bool           `json:"is_face_validation_required" gorm:"not null"`
	CreatedBy                *int64         `json:"created_by"`
	UpdatedBy                *int64         `json:"updated_by"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `json:"-" gorm:"index"`
}
