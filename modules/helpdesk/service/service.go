package service

import (
	"errors"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/dto"
)

var (
	ErrArticleNotFound = errors.New("help desk article not found")
	ErrReorderMismatch = errors.New("article_ids must contain every article of the category exactly once")
)

// ManagerRoles can manage the help desk and see unpublished (draft) articles.
var ManagerRoles = []string{
	models.RoleSuperAdmin,
	models.RoleSecurityManager,
	models.RoleSecurityHead,
	models.RoleSecurityAdmin,
}

// Actor is the logged in user performing the request.
type Actor struct {
	UserID   int64
	RoleCode string
}

type HelpDeskService interface {
	GetArticles(actor Actor, filter dto.ArticleFilter) ([]models.HelpDeskArticle, int64, error)
	GetArticle(actor Actor, id int64) (models.HelpDeskArticle, error)
	// CreateArticle places the article last in its category when sortOrder is nil.
	CreateArticle(article models.HelpDeskArticle, sortOrder *int, actorID int64) (models.HelpDeskArticle, error)
	UpdateArticle(id int64, input models.HelpDeskArticle, sortOrder *int, actorID int64) (models.HelpDeskArticle, error)
	DeleteArticle(id int64) error
	// ReorderArticles sets the order of all articles of a category and returns them in that order.
	ReorderArticles(category string, articleIDs []int64, actorID int64) ([]models.HelpDeskArticle, error)
}
