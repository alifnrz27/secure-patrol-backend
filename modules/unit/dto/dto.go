package dto

import "secure-patrol-backend/models"

type UnitDto interface {
	ToUnitDTO(unit models.Unit) UnitDTO
	ToUnitDTOs(units []models.Unit) []UnitDTO
}

type dto struct{}

func NewUnitDto() UnitDto {
	return &dto{}
}

func (d *dto) ToUnitDTO(unit models.Unit) UnitDTO {
	return UnitDTO{
		ID:                unit.ID,
		Code:              unit.Code,
		Name:              unit.Name,
		Latitude:          unit.Latitude,
		Longitude:         unit.Longitude,
		IsActive:          unit.IsActive,
		UsersCount:        unit.UsersCount,
		PatrolPointsCount: unit.PatrolPointsCount,
		CreatedAt:         unit.CreatedAt,
		UpdatedAt:         unit.UpdatedAt,
	}
}

func (d *dto) ToUnitDTOs(units []models.Unit) []UnitDTO {
	result := make([]UnitDTO, 0, len(units))
	for _, unit := range units {
		result = append(result, d.ToUnitDTO(unit))
	}
	return result
}
