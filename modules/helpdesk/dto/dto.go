package dto

import "secure-patrol-backend/models"

type HelpDeskDto interface {
	ToArticleDTO(article models.HelpDeskArticle) HelpDeskArticleDTO
	ToArticleDTOs(articles []models.HelpDeskArticle) []HelpDeskArticleDTO
}

type dto struct{}

func NewHelpDeskDto() HelpDeskDto {
	return &dto{}
}

func (d *dto) ToArticleDTO(article models.HelpDeskArticle) HelpDeskArticleDTO {
	return HelpDeskArticleDTO{
		ID:          article.ID,
		Category:    article.Category,
		Title:       article.Title,
		Content:     article.Content,
		SortOrder:   article.SortOrder,
		IsPublished: article.IsPublished,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}
}

func (d *dto) ToArticleDTOs(articles []models.HelpDeskArticle) []HelpDeskArticleDTO {
	result := make([]HelpDeskArticleDTO, 0, len(articles))
	for _, article := range articles {
		result = append(result, d.ToArticleDTO(article))
	}
	return result
}
