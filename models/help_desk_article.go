package models

import (
	"time"

	"gorm.io/gorm"
)

// Help desk article categories.
const (
	HelpDeskCategoryRule  = "rule"
	HelpDeskCategoryGuide = "guide"
	HelpDeskCategoryFAQ   = "faq"
)

// HelpDeskArticle is a rule, usage guide or FAQ shown in the help desk menu.
// Content is Markdown.
type HelpDeskArticle struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	Category    string         `json:"category" gorm:"type:varchar(20);not null;index"`
	Title       string         `json:"title" gorm:"type:varchar(200);not null"`
	Content     string         `json:"content" gorm:"type:text;not null"`
	SortOrder   int            `json:"sort_order" gorm:"not null;default:0"`
	IsPublished bool           `json:"is_published" gorm:"not null"`
	CreatedBy   *int64         `json:"created_by"`
	UpdatedBy   *int64         `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
