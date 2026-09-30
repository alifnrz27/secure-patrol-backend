package dto

import "time"

type ProgressDTO struct {
	TotalPoints     int64 `json:"total_points"`
	ScannedPoints   int64 `json:"scanned_points"`
	UnscannedPoints int64 `json:"unscanned_points"`
	TotalScans      int64 `json:"total_scans"`
	AbnormalScans   int64 `json:"abnormal_scans"`
}

// UnitSummaryDTO is the unit a patrol group belongs to.
type UnitSummaryDTO struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type ShiftSummaryDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PatrolGroupDTO struct {
	ID        int64           `json:"id"`
	Unit      UnitSummaryDTO  `json:"unit"`
	Shift     ShiftSummaryDTO `json:"shift"`
	ShiftDate string          `json:"shift_date"`
	StartAt   time.Time       `json:"start_at"`
	EndAt     time.Time       `json:"end_at"`
	Status    string          `json:"status"` // upcoming | ongoing | finished
	Progress  ProgressDTO     `json:"progress"`
	CreatedAt time.Time       `json:"created_at"`
}

type PatrolGroupDetailDTO struct {
	PatrolGroupDTO
	Items []PatrolListItemDTO `json:"items"`
}

type GroupSummaryDTO struct {
	ID        int64     `json:"id"`
	UnitID    int64     `json:"unit_id"`
	UnitName  string    `json:"unit_name"`
	ShiftID   int64     `json:"shift_id"`
	ShiftName string    `json:"shift_name"`
	ShiftDate string    `json:"shift_date"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
}

type UserSummaryDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PatrolListItemDTO struct {
	ID                       int64            `json:"id"`
	Group                    *GroupSummaryDTO `json:"group,omitempty"`
	PatrolPointID            int64            `json:"patrol_point_id"`
	Name                     string           `json:"name"`
	Location                 string           `json:"location"`
	NFCCode                  string           `json:"nfc_code"`
	Latitude                 float64          `json:"latitude"`
	Longitude                float64          `json:"longitude"`
	IsLocationMatchRequired  bool             `json:"is_location_match_required"`
	IsFaceValidationRequired bool             `json:"is_face_validation_required"`
	IsScanned                bool             `json:"is_scanned"`
	ScanCount                int              `json:"scan_count"`
	LastScannedAt            *time.Time       `json:"last_scanned_at"`
	LastScannedBy            *UserSummaryDTO  `json:"last_scanned_by"`
	LastCondition            *string          `json:"last_condition"`
}

type ScanPointDTO struct {
	PatrolListItemID int64  `json:"patrol_list_item_id"`
	PatrolPointID    int64  `json:"patrol_point_id"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	NFCCode          string `json:"nfc_code"`
}

type ScanPhotoDTO struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type PatrolScanDTO struct {
	ID              int64           `json:"id"`
	ClientScanID    string          `json:"client_scan_id"`
	Group           GroupSummaryDTO `json:"group"`
	PatrolPoint     ScanPointDTO    `json:"patrol_point"`
	Condition       string          `json:"condition"`
	Note            string          `json:"note"`
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	DistanceMeters  float64         `json:"distance_meters"`
	IsLocationValid bool            `json:"is_location_valid"`
	IsFaceVerified  bool            `json:"is_face_verified"`
	FaceMatchScore  *float64        `json:"face_match_score"`
	ScannedAt       time.Time       `json:"scanned_at"`
	ReceivedAt      time.Time       `json:"received_at"`
	ScannedBy       UserSummaryDTO  `json:"scanned_by"`
	Photos          []ScanPhotoDTO  `json:"photos"`
}

// ScanRequest is sent as multipart/form-data with up to 3 files in "photos".
type ScanRequest struct {
	ClientScanID   string   `form:"client_scan_id" validate:"required,max=64"`
	NFCCode        string   `form:"nfc_code" validate:"required,max=100"`
	Condition      string   `form:"condition" validate:"required,oneof=normal abnormal"`
	Note           string   `form:"note" validate:"max=1000"`
	Latitude       *float64 `form:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude      *float64 `form:"longitude" validate:"required,gte=-180,lte=180"`
	ScannedAt      string   `form:"scanned_at"`
	FaceVerified   *bool    `form:"face_verified"`
	FaceMatchScore *float64 `form:"face_match_score" validate:"omitempty,gte=0,lte=1"`
}
