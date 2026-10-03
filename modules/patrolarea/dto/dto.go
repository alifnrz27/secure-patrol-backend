package dto

import (
	"secure-patrol-backend/models"
	"time"
)

type PatrolAreaDTO struct {
	ID                int64     `json:"id"`
	UnitID            int64     `json:"unit_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	PatrolPointsCount int64     `json:"patrol_points_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PatrolAreaRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
}

func ToPatrolAreaDTO(area models.PatrolArea) PatrolAreaDTO {
	return PatrolAreaDTO{
		ID:                area.ID,
		UnitID:            area.UnitID,
		Name:              area.Name,
		Description:       area.Description,
		PatrolPointsCount: area.PatrolPointsCount,
		CreatedAt:         area.CreatedAt,
		UpdatedAt:         area.UpdatedAt,
	}
}

func ToPatrolAreaDTOs(areas []models.PatrolArea) []PatrolAreaDTO {
	result := make([]PatrolAreaDTO, 0, len(areas))
	for _, area := range areas {
		result = append(result, ToPatrolAreaDTO(area))
	}
	return result
}
