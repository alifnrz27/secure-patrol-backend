package models

import (
	"time"

	"gorm.io/gorm"
)

// Unit is a site (building, area, branch) with its own security team, patrol
// points, shifts and settings. Central roles do not belong to a unit.
type Unit struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	Code      string         `json:"code" gorm:"type:varchar(30);uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"type:varchar(150);not null"`
	Latitude  float64        `json:"latitude" gorm:"type:double precision;not null"`
	Longitude float64        `json:"longitude" gorm:"type:double precision;not null"`
	IsActive  bool           `json:"is_active" gorm:"not null"`
	CreatedBy *int64         `json:"created_by"`
	UpdatedBy *int64         `json:"updated_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Filled by queries only.
	UsersCount        int64 `json:"-" gorm:"->;-:migration"`
	PatrolPointsCount int64 `json:"-" gorm:"->;-:migration"`
}

// DefaultShifts are the shifts a new unit starts with; the unit can change them.
func DefaultShifts(unitID int64) []PatrolShift {
	return []PatrolShift{
		{UnitID: unitID, Name: "Shift 1", StartTime: "08:00", EndTime: "16:00", IsActive: true},
		{UnitID: unitID, Name: "Shift 2", StartTime: "16:00", EndTime: "24:00", IsActive: true},
		{UnitID: unitID, Name: "Shift 3", StartTime: "00:00", EndTime: "08:00", IsActive: true},
	}
}
