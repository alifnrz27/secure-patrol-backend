package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Pagination struct {
	Page   int
	Limit  int
	Search string
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedData struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// NewPagination reads ?page, ?limit and ?search from the query string.
func NewPagination(c *fiber.Ctx) Pagination {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	return Pagination{
		Page:   page,
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
	}
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

func NewPaginatedData(items interface{}, p Pagination, total int64) PaginatedData {
	totalPages := int((total + int64(p.Limit) - 1) / int64(p.Limit))

	return PaginatedData{
		Items: items,
		Pagination: PaginationMeta{
			Page:       p.Page,
			Limit:      p.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
