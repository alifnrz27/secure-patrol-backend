package models

import (
	"time"

	"gorm.io/gorm"
)

// Patrol scan conditions.
const (
	PatrolConditionNormal   = "normal"
	PatrolConditionAbnormal = "abnormal"
)

// PatrolShift is a daily time window configured by the admin, e.g. 08:00-16:00.
// The end time is the cut-off: a scan at or after it belongs to the next shift.
type PatrolShift struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null"`
	StartTime string         `json:"start_time" gorm:"type:varchar(5);not null"` // HH:MM
	EndTime   string         `json:"end_time" gorm:"type:varchar(5);not null"`   // HH:MM, 24:00 allowed
	IsActive  bool           `json:"is_active" gorm:"not null"`
	CreatedBy *int64         `json:"created_by"`
	UpdatedBy *int64         `json:"updated_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// PatrolGroup is one shift on one date, created automatically. Its start and
// end are copied from the shift, so later shift changes do not alter history.
type PatrolGroup struct {
	ID            int64          `json:"id" gorm:"primaryKey"`
	PatrolShiftID int64          `json:"patrol_shift_id" gorm:"not null;uniqueIndex:idx_patrol_groups_shift_date"`
	PatrolShift   PatrolShift    `json:"-" gorm:"foreignKey:PatrolShiftID"`
	ShiftName     string         `json:"shift_name" gorm:"type:varchar(100);not null"`
	ShiftDate     time.Time      `json:"shift_date" gorm:"type:date;not null;uniqueIndex:idx_patrol_groups_shift_date"`
	StartAt       time.Time      `json:"start_at" gorm:"not null;index"`
	EndAt         time.Time      `json:"end_at" gorm:"not null;index"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Progress, filled by queries only.
	TotalPoints   int64 `json:"-" gorm:"->;-:migration"`
	ScannedPoints int64 `json:"-" gorm:"->;-:migration"`
	TotalScans    int64 `json:"-" gorm:"->;-:migration"`
	AbnormalScans int64 `json:"-" gorm:"->;-:migration"`
}

// PatrolListItem is a copy of a patrol point inside a patrol group (the patrol list).
type PatrolListItem struct {
	ID                       int64          `json:"id" gorm:"primaryKey"`
	PatrolGroupID            int64          `json:"patrol_group_id" gorm:"not null;uniqueIndex:idx_patrol_list_items_group_point"`
	PatrolGroup              *PatrolGroup   `json:"-" gorm:"foreignKey:PatrolGroupID"`
	PatrolPointID            int64          `json:"patrol_point_id" gorm:"not null;uniqueIndex:idx_patrol_list_items_group_point;index"`
	Name                     string         `json:"name" gorm:"type:varchar(150);not null"`
	Location                 string         `json:"location" gorm:"type:varchar(255);not null"`
	NFCCode                  string         `json:"nfc_code" gorm:"column:nfc_code;type:varchar(100);not null"`
	Latitude                 float64        `json:"latitude" gorm:"type:double precision;not null"`
	Longitude                float64        `json:"longitude" gorm:"type:double precision;not null"`
	IsLocationMatchRequired  bool           `json:"is_location_match_required" gorm:"not null"`
	IsFaceValidationRequired bool           `json:"is_face_validation_required" gorm:"not null"`
	ScanCount                int            `json:"scan_count" gorm:"not null;default:0"`
	LastScannedAt            *time.Time     `json:"last_scanned_at"`
	LastScannedBy            *int64         `json:"last_scanned_by"`
	LastScannedByUser        *User          `json:"-" gorm:"foreignKey:LastScannedBy"`
	LastCondition            *string        `json:"last_condition" gorm:"type:varchar(10)"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `json:"-" gorm:"index"`
}

// PatrolScan is one NFC scan. A point can be scanned many times in one group.
type PatrolScan struct {
	ID               int64             `json:"id" gorm:"primaryKey"`
	ClientScanID     string            `json:"client_scan_id" gorm:"type:varchar(64);not null;uniqueIndex:idx_patrol_scans_client"`
	ScannedBy        int64             `json:"scanned_by" gorm:"not null;uniqueIndex:idx_patrol_scans_client;index"`
	ScannedByUser    User              `json:"-" gorm:"foreignKey:ScannedBy"`
	PatrolGroupID    int64             `json:"patrol_group_id" gorm:"not null;index"`
	PatrolGroup      PatrolGroup       `json:"-" gorm:"foreignKey:PatrolGroupID"`
	PatrolListItemID int64             `json:"patrol_list_item_id" gorm:"not null;index"`
	PatrolListItem   PatrolListItem    `json:"-" gorm:"foreignKey:PatrolListItemID"`
	PatrolPointID    int64             `json:"patrol_point_id" gorm:"not null;index"`
	AppClientID      int64             `json:"app_client_id" gorm:"not null"`
	NFCCode          string            `json:"nfc_code" gorm:"column:nfc_code;type:varchar(100);not null"`
	Condition        string            `json:"condition" gorm:"type:varchar(10);not null"`
	Note             string            `json:"note" gorm:"type:text"`
	Latitude         float64           `json:"latitude" gorm:"type:double precision;not null"`
	Longitude        float64           `json:"longitude" gorm:"type:double precision;not null"`
	DistanceMeters   float64           `json:"distance_meters" gorm:"type:double precision;not null"`
	IsLocationValid  bool              `json:"is_location_valid" gorm:"not null"`
	IsFaceVerified   bool              `json:"is_face_verified" gorm:"not null"`
	FaceMatchScore   *float64          `json:"face_match_score" gorm:"type:double precision"`
	ScannedAt        time.Time         `json:"scanned_at" gorm:"not null;index"`
	ReceivedAt       time.Time         `json:"received_at" gorm:"not null"`
	Photos           []PatrolScanPhoto `json:"-" gorm:"foreignKey:PatrolScanID"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `json:"-" gorm:"index"`
}

type PatrolScanPhoto struct {
	ID           int64          `json:"id" gorm:"primaryKey"`
	PatrolScanID int64          `json:"patrol_scan_id" gorm:"not null;index"`
	Path         string         `json:"-" gorm:"type:varchar(255);not null"`
	SortOrder    int            `json:"sort_order" gorm:"not null"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
