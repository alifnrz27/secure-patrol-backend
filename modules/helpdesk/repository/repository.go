package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/dto"
)

type HelpDeskRepository interface {
	FindAll(filter dto.ArticleFilter) ([]models.HelpDeskArticle, int64, error)
	FindByID(id int64) (models.HelpDeskArticle, error)
	Create(article *models.HelpDeskArticle) error
	Update(article *models.HelpDeskArticle) error
	Delete(id int64) error
	// NextSortOrder returns the position after the last article of the category.
	NextSortOrder(category string) (int, error)
	FindIDsByCategory(category string) ([]int64, error)
	// UpdateSortOrders sets sort_order to 1..n following the order of ids.
	UpdateSortOrders(ids []int64, actorID int64) error
}
