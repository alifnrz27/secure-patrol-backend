package dto

import "time"

type UnitDTO struct {
	ID                int64     `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	IsActive          bool      `json:"is_active"`
	UsersCount        int64     `json:"users_count"`
	PatrolPointsCount int64     `json:"patrol_points_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Latitude and longitude are pointers so that 0 (a valid coordinate) is
// distinguished from a missing value.
type CreateUnitRequest struct {
	Code      string   `json:"code" validate:"required,max=20"`
	Name      string   `json:"name" validate:"required,max=150"`
	Latitude  *float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	IsActive  *bool    `json:"is_active"`
}

type UpdateUnitRequest struct {
	Code      string   `json:"code" validate:"required,max=20"`
	Name      string   `json:"name" validate:"required,max=150"`
	Latitude  *float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	IsActive  *bool    `json:"is_active" validate:"required"`
}
