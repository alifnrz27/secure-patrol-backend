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

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(pagination helper.Pagination) (roles []models.Role, total int64, err error) {
	query := r.db.Model(&models.Role{})

	if pagination.Search != "" {
		like := "%" + pagination.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		Order("id ASC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&roles).Error

	return roles, total, err
}

func (r *repository) FindByID(id int64) (role models.Role, err error) {
	err = r.db.First(&role, id).Error
	return role, err
}

func (r *repository) FindByCode(code string) (role models.Role, err error) {
	err = r.db.Where("code = ?", code).First(&role).Error
	return role, err
}

func (r *repository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *repository) Update(role *models.Role) error {
	// Select all columns so false values (is_active) are written too.
	return r.db.Model(role).
		Select("name", "description", "is_active").
		Updates(role).Error
}

// Delete soft deletes the role and frees its code so it can be used again.
func (r *repository) Delete(id int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}

		releasedCode := fmt.Sprintf("deleted_%d_%d_%s", role.ID, time.Now().Unix(), role.Code)
		if len(releasedCode) > 50 {
			releasedCode = releasedCode[:50]
		}

		if err := tx.Model(&models.Role{}).Where("id = ?", id).Update("code", releasedCode).Error; err != nil {
			return err
		}

		return tx.Delete(&models.Role{}, id).Error
	})
}

func (r *repository) CountUsers(roleID int64) (count int64, err error) {
	err = r.db.Model(&models.User{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}
