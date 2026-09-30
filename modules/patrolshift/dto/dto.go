package dto

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolshift/service"
)

type PatrolShiftDto interface {
	ToPatrolShiftDTO(shift models.PatrolShift) PatrolShiftDTO
	ToPatrolShiftDTOs(shifts []models.PatrolShift) []PatrolShiftDTO
}

type dto struct{}

func NewPatrolShiftDto() PatrolShiftDto {
	return &dto{}
}

func (d *dto) ToPatrolShiftDTO(shift models.PatrolShift) PatrolShiftDTO {
	start, duration, _ := service.Bounds(shift)

	return PatrolShiftDTO{
		ID:              shift.ID,
		UnitID:          shift.UnitID,
		Name:            shift.Name,
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		DurationMinutes: duration,
		CrossesMidnight: start+duration > 24*60,
		IsActive:        shift.IsActive,
		CreatedAt:       shift.CreatedAt,
		UpdatedAt:       shift.UpdatedAt,
	}
}

func (d *dto) ToPatrolShiftDTOs(shifts []models.PatrolShift) []PatrolShiftDTO {
	result := make([]PatrolShiftDTO, 0, len(shifts))
	for _, shift := range shifts {
		result = append(result, d.ToPatrolShiftDTO(shift))
	}
	return result
}
