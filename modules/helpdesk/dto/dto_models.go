package dto

import (
	"secure-patrol-backend/helper"
	"time"
)

type HelpDeskArticleDTO struct {
	ID          int64     `json:"id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	SortOrder   int       `json:"sort_order"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ArticleFilter struct {
	helper.Pagination
	Category string
	// PublishedOnly hides drafts; set for users who cannot manage the help desk.
	PublishedOnly bool
	IsPublished   *bool
}

// SortOrder is optional: when omitted the article is placed last in its category.
type CreateHelpDeskArticleRequest struct {
	Category    string `json:"category" validate:"required,oneof=rule guide faq"`
	Title       string `json:"title" validate:"required,max=200"`
	Content     string `json:"content" validate:"required,max=20000"`
	SortOrder   *int   `json:"sort_order" validate:"omitempty,gte=0"`
	IsPublished *bool  `json:"is_published"`
}

// SortOrder is optional: when omitted the order is kept, or the article is
// placed last when it moves to another category.
type UpdateHelpDeskArticleRequest struct {
	Category    string `json:"category" validate:"required,oneof=rule guide faq"`
	Title       string `json:"title" validate:"required,max=200"`
	Content     string `json:"content" validate:"required,max=20000"`
	SortOrder   *int   `json:"sort_order" validate:"omitempty,gte=0"`
	IsPublished *bool  `json:"is_published" validate:"required"`
}

// ReorderHelpDeskArticlesRequest lists every article of a category in the new order.
type ReorderHelpDeskArticlesRequest struct {
	Category   string  `json:"category" validate:"required,oneof=rule guide faq"`
	ArticleIDs []int64 `json:"article_ids" validate:"required,min=1,unique,dive,gt=0"`
}
