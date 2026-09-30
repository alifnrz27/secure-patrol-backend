package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/dto"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewHelpDeskRepository(db *gorm.DB) HelpDeskRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(filter dto.ArticleFilter) (articles []models.HelpDeskArticle, total int64, err error) {
	query := r.db.Model(&models.HelpDeskArticle{})

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	// Both conditions apply together: an officer asking for drafts gets nothing.
	if filter.PublishedOnly {
		query = query.Where("is_published = ?", true)
	}
	if filter.IsPublished != nil {
		query = query.Where("is_published = ?", *filter.IsPublished)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", like, like)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Rules first, then guides, then FAQ; admins control the order inside a category.
	err = query.
		Order("CASE category WHEN 'rule' THEN 1 WHEN 'guide' THEN 2 ELSE 3 END").
		Order("sort_order ASC, id ASC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&articles).Error

	return articles, total, err
}

func (r *repository) FindByID(id int64) (article models.HelpDeskArticle, err error) {
	err = r.db.First(&article, id).Error
	return article, err
}

func (r *repository) Create(article *models.HelpDeskArticle) error {
	return r.db.Create(article).Error
}

func (r *repository) Update(article *models.HelpDeskArticle) error {
	return r.db.Model(article).
		Select("category", "title", "content", "sort_order", "is_published", "updated_by").
		Updates(article).Error
}

func (r *repository) Delete(id int64) error {
	return r.db.Delete(&models.HelpDeskArticle{}, id).Error
}

func (r *repository) NextSortOrder(category string) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.HelpDeskArticle{}).
		Where("category = ?", category).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder).Error
	return maxOrder + 1, err
}

func (r *repository) FindIDsByCategory(category string) (ids []int64, err error) {
	err = r.db.Model(&models.HelpDeskArticle{}).
		Where("category = ?", category).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *repository) UpdateSortOrders(ids []int64, actorID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&models.HelpDeskArticle{}).Where("id = ?", id).
				Updates(map[string]interface{}{"sort_order": i + 1, "updated_by": actorID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
