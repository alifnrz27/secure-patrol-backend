package repository

import (
	"fmt"
	"secure-patrol-backend/models"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewUnitRepository(db *gorm.DB) UnitRepository {
	return &repository{db: db}
}

// withCounts adds the number of users and patrol points of each unit.
func (r *repository) withCounts(query *gorm.DB) *gorm.DB {
	return query.Select(`units.*,
		(SELECT COUNT(*) FROM users WHERE users.unit_id = units.id AND users.deleted_at IS NULL) AS users_count,
		(SELECT COUNT(*) FROM patrol_points WHERE patrol_points.unit_id = units.id AND patrol_points.deleted_at IS NULL) AS patrol_points_count`)
}

func (r *repository) FindAll(filter UnitFilter) (units []models.Unit, total int64, err error) {
	query := r.db.Model(&models.Unit{})

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("units.name ILIKE ? OR units.code ILIKE ?", like, like)
	}
	if filter.IsActive != nil {
		query = query.Where("units.is_active = ?", *filter.IsActive)
	}
	if filter.OnlyID != nil {
		query = query.Where("units.id = ?", *filter.OnlyID)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = r.withCounts(query).
		Order("units.name ASC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&units).Error

	return units, total, err
}

func (r *repository) FindByID(id int64) (unit models.Unit, err error) {
	err = r.withCounts(r.db.Model(&models.Unit{})).Where("units.id = ?", id).First(&unit).Error
	return unit, err
}

func (r *repository) FindByCode(code string) (unit models.Unit, err error) {
	err = r.db.Where("code = ?", code).First(&unit).Error
	return unit, err
}

func (r *repository) Create(unit *models.Unit) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(unit).Error; err != nil {
			return err
		}

		shifts := models.DefaultShifts(unit.ID)
		for i := range shifts {
			shifts[i].CreatedBy = unit.CreatedBy
			shifts[i].UpdatedBy = unit.CreatedBy
		}
		return tx.Create(&shifts).Error
	})
}

func (r *repository) Update(unit *models.Unit) error {
	return r.db.Model(unit).
		Select("code", "name", "latitude", "longitude", "is_active", "updated_by").
		Updates(unit).Error
}

// Delete soft deletes the unit and its shifts, and frees its code so it can be reused.
func (r *repository) Delete(unit models.Unit) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		releasedCode := fmt.Sprintf("deleted_%d_%d_%s", unit.ID, time.Now().Unix(), unit.Code)
		if len(releasedCode) > 30 {
			releasedCode = releasedCode[:30]
		}

		if err := tx.Model(&models.Unit{}).Where("id = ?", unit.ID).
			Update("code", releasedCode).Error; err != nil {
			return err
		}
		if err := tx.Where("unit_id = ?", unit.ID).Delete(&models.PatrolShift{}).Error; err != nil {
			return err
		}
		if err := tx.Where("unit_id = ?", unit.ID).Delete(&models.SystemSetting{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Unit{}, unit.ID).Error
	})
}

func (r *repository) CountUsers(unitID int64) (count int64, err error) {
	err = r.db.Model(&models.User{}).Where("unit_id = ?", unitID).Count(&count).Error
	return count, err
}

func (r *repository) CountPatrolPoints(unitID int64) (count int64, err error) {
	err = r.db.Model(&models.PatrolPoint{}).Where("unit_id = ?", unitID).Count(&count).Error
	return count, err
}
