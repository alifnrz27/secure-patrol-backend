package repository

import (
	"fmt"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"time"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewPatrolPointRepository(db *gorm.DB) PatrolPointRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(pagination helper.Pagination, unitID int64, areaID int64) (points []models.PatrolPoint, total int64, err error) {
	query := r.db.Model(&models.PatrolPoint{})

	if unitID > 0 {
		query = query.Where("unit_id = ?", unitID)
	}
	if areaID > 0 {
		query = query.Where("area_id = ?", areaID)
	}

	if pagination.Search != "" {
		like := "%" + pagination.Search + "%"
		query = query.Where("name ILIKE ? OR location ILIKE ? OR nfc_code ILIKE ?", like, like, like)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		Preload("Area").
		Order("name ASC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&points).Error

	return points, total, err
}

func (r *repository) FindByID(id int64) (point models.PatrolPoint, err error) {
	err = r.db.Preload("Area").First(&point, id).Error
	return point, err
}

func (r *repository) FindByNFCCode(nfcCode string) (point models.PatrolPoint, err error) {
	err = r.db.Preload("Area").Where("nfc_code = ?", nfcCode).First(&point).Error
	return point, err
}

func (r *repository) Create(point *models.PatrolPoint) error {
	return r.db.Create(point).Error
}

func (r *repository) Update(point *models.PatrolPoint) error {
	return r.db.Model(point).
		Select(
			"name",
			"area_id",
			"location",
			"nfc_code",
			"latitude",
			"longitude",
			"is_location_match_required",
			"is_face_validation_required",
			"updated_by",
		).
		Updates(point).Error
}

// Delete soft deletes the patrol point and frees its NFC code so the tag can be reused.
func (r *repository) Delete(point models.PatrolPoint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		releasedCode := fmt.Sprintf("deleted_%d_%d_%s", point.ID, time.Now().Unix(), point.NFCCode)
		if len(releasedCode) > 100 {
			releasedCode = releasedCode[:100]
		}

		if err := tx.Model(&models.PatrolPoint{}).Where("id = ?", point.ID).
			Update("nfc_code", releasedCode).Error; err != nil {
			return err
		}

		return tx.Delete(&models.PatrolPoint{}, point.ID).Error
	})
}

func (r *repository) FindArea(id int64) (area models.PatrolArea, err error) {
	err = r.db.First(&area, id).Error
	return area, err
}
