package dto

import "time"

type PatrolShiftDTO struct {
	ID              int64     `json:"id"`
	UnitID          int64     `json:"unit_id"`
	Name            string    `json:"name"`
	StartTime       string    `json:"start_time"`
	EndTime         string    `json:"end_time"`
	DurationMinutes int       `json:"duration_minutes"`
	CrossesMidnight bool      `json:"crosses_midnight"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreatePatrolShiftRequest struct {
	Name      string `json:"name" validate:"required,max=100"`
	StartTime string `json:"start_time" validate:"required"`
	EndTime   string `json:"end_time" validate:"required"`
	IsActive  *bool  `json:"is_active"`
}

type UpdatePatrolShiftRequest struct {
	Name      string `json:"name" validate:"required,max=100"`
	StartTime string `json:"start_time" validate:"required"`
	EndTime   string `json:"end_time" validate:"required"`
	IsActive  *bool  `json:"is_active" validate:"required"`
}
