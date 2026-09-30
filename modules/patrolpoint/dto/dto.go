package dto

import "secure-patrol-backend/models"

type PatrolPointDto interface {
	ToPatrolPointDTO(point models.PatrolPoint) PatrolPointDTO
	ToPatrolPointDTOs(points []models.PatrolPoint) []PatrolPointDTO
}

type dto struct{}

func NewPatrolPointDto() PatrolPointDto {
	return &dto{}
}

func (d *dto) ToPatrolPointDTO(point models.PatrolPoint) PatrolPointDTO {
	return PatrolPointDTO{
		ID:                       point.ID,
		UnitID:                   point.UnitID,
		Name:                     point.Name,
		Location:                 point.Location,
		NFCCode:                  point.NFCCode,
		Latitude:                 point.Latitude,
		Longitude:                point.Longitude,
		IsLocationMatchRequired:  point.IsLocationMatchRequired,
		IsFaceValidationRequired: point.IsFaceValidationRequired,
		CreatedAt:                point.CreatedAt,
		UpdatedAt:                point.UpdatedAt,
	}
}

func (d *dto) ToPatrolPointDTOs(points []models.PatrolPoint) []PatrolPointDTO {
	result := make([]PatrolPointDTO, 0, len(points))
	for _, point := range points {
		result = append(result, d.ToPatrolPointDTO(point))
	}
	return result
}
