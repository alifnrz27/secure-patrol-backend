package dto

import "time"

// AreaDTO is the area a patrol point belongs to; null when it has none.
type AreaDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PatrolPointDTO struct {
	ID                       int64     `json:"id"`
	UnitID                   int64     `json:"unit_id"`
	AreaID                   *int64    `json:"area_id"`
	Area                     *AreaDTO  `json:"area"`
	Name                     string    `json:"name"`
	Location                 string    `json:"location"`
	NFCCode                  string    `json:"nfc_code"`
	Latitude                 float64   `json:"latitude"`
	Longitude                float64   `json:"longitude"`
	IsLocationMatchRequired  bool      `json:"is_location_match_required"`
	IsFaceValidationRequired bool      `json:"is_face_validation_required"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// Latitude and longitude are pointers so that 0 (a valid coordinate) is
// distinguished from a missing value.
type CreatePatrolPointRequest struct {
	Name                     string   `json:"name" validate:"required,max=150"`
	AreaID                   *int64   `json:"area_id" validate:"omitempty,gt=0"`
	Location                 string   `json:"location" validate:"required,max=255"`
	NFCCode                  string   `json:"nfc_code" validate:"required,max=100"`
	Latitude                 *float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude                *float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	IsLocationMatchRequired  *bool    `json:"is_location_match_required"`
	IsFaceValidationRequired *bool    `json:"is_face_validation_required"`
}

type UpdatePatrolPointRequest struct {
	Name string `json:"name" validate:"required,max=150"`
	// AreaID null or missing removes the point from its area.
	AreaID                   *int64   `json:"area_id" validate:"omitempty,gt=0"`
	Location                 string   `json:"location" validate:"required,max=255"`
	NFCCode                  string   `json:"nfc_code" validate:"required,max=100"`
	Latitude                 *float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude                *float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	IsLocationMatchRequired  *bool    `json:"is_location_match_required" validate:"required"`
	IsFaceValidationRequired *bool    `json:"is_face_validation_required" validate:"required"`
}
