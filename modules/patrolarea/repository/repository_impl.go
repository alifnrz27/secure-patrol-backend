package repository

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewPatrolAreaRepository(db *gorm.DB) PatrolAreaRepository {
	return &repository{db: db}
}

func (r *repository) withCounts(query *gorm.DB) *gorm.DB {
	return query.Select(`patrol_areas.*,
		(SELECT COUNT(*) FROM patrol_points p WHERE p.area_id = patrol_areas.id AND p.deleted_at IS NULL) AS patrol_points_count`)
}

func (r *repository) FindAll(pagination helper.Pagination, unitID int64) (areas []models.PatrolArea, total int64, err error) {
	query := r.db.Model(&models.PatrolArea{})
	if unitID > 0 {
		query = query.Where("patrol_areas.unit_id = ?", unitID)
	}
	if pagination.Search != "" {
		like := "%" + pagination.Search + "%"
		query = query.Where("patrol_areas.name ILIKE ? OR patrol_areas.description ILIKE ?", like, like)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = r.withCounts(query).
		Order("patrol_areas.unit_id ASC, patrol_areas.name ASC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&areas).Error
	return areas, total, err
}

func (r *repository) FindByID(id int64) (area models.PatrolArea, err error) {
	err = r.withCounts(r.db.Model(&models.PatrolArea{})).Where("patrol_areas.id = ?", id).First(&area).Error
	return area, err
}

func (r *repository) FindByName(unitID int64, name string) (area models.PatrolArea, err error) {
	err = r.db.Where("unit_id = ? AND LOWER(name) = LOWER(?)", unitID, name).First(&area).Error
	return area, err
}

func (r *repository) Create(area *models.PatrolArea) error {
	return r.db.Create(area).Error
}

func (r *repository) Update(area *models.PatrolArea) error {
	return r.db.Model(area).Select("name", "description", "updated_by").Updates(area).Error
}

// Delete soft deletes the area; the partial unique index frees its name.
func (r *repository) Delete(id int64) error {
	return r.db.Delete(&models.PatrolArea{}, id).Error
}

func (r *repository) CountPoints(areaID int64) (count int64, err error) {
	err = r.db.Model(&models.PatrolPoint{}).Where("area_id = ?", areaID).Count(&count).Error
	return count, err
}
