package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/dto"
	"secure-patrol-backend/modules/helpdesk/repository"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	repo repository.HelpDeskRepository
}

func NewHelpDeskService(repo repository.HelpDeskRepository) HelpDeskService {
	return &service{repo: repo}
}

func canSeeDrafts(actor Actor) bool {
	return helper.Includes(ManagerRoles, actor.RoleCode)
}

func (s *service) GetArticles(actor Actor, filter dto.ArticleFilter) ([]models.HelpDeskArticle, int64, error) {
	filter.PublishedOnly = !canSeeDrafts(actor)
	return s.repo.FindAll(filter)
}

func (s *service) GetArticle(actor Actor, id int64) (models.HelpDeskArticle, error) {
	article, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return article, ErrArticleNotFound
	}
	if err != nil {
		return article, err
	}

	// Drafts are hidden from officers as if they did not exist.
	if !article.IsPublished && !canSeeDrafts(actor) {
		return models.HelpDeskArticle{}, ErrArticleNotFound
	}

	return article, nil
}

func (s *service) CreateArticle(article models.HelpDeskArticle, sortOrder *int, actorID int64) (models.HelpDeskArticle, error) {
	article.Title = strings.TrimSpace(article.Title)
	article.Content = strings.TrimSpace(article.Content)

	if sortOrder != nil {
		article.SortOrder = *sortOrder
	} else {
		next, err := s.repo.NextSortOrder(article.Category)
		if err != nil {
			return article, err
		}
		article.SortOrder = next
	}

	article.CreatedBy = &actorID
	article.UpdatedBy = &actorID

	if err := s.repo.Create(&article); err != nil {
		return article, err
	}
	return article, nil
}

func (s *service) UpdateArticle(id int64, input models.HelpDeskArticle, sortOrder *int, actorID int64) (models.HelpDeskArticle, error) {
	article, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return article, ErrArticleNotFound
	}
	if err != nil {
		return article, err
	}

	switch {
	case sortOrder != nil:
		article.SortOrder = *sortOrder
	case input.Category != article.Category:
		// Moving to another category places the article last there.
		next, err := s.repo.NextSortOrder(input.Category)
		if err != nil {
			return article, err
		}
		article.SortOrder = next
	}

	article.Category = input.Category
	article.Title = strings.TrimSpace(input.Title)
	article.Content = strings.TrimSpace(input.Content)
	article.IsPublished = input.IsPublished
	article.UpdatedBy = &actorID

	if err := s.repo.Update(&article); err != nil {
		return article, err
	}
	return s.repo.FindByID(article.ID)
}

func (s *service) ReorderArticles(category string, articleIDs []int64, actorID int64) ([]models.HelpDeskArticle, error) {
	existing, err := s.repo.FindIDsByCategory(category)
	if err != nil {
		return nil, err
	}

	// The new order must be complete: same articles, each exactly once.
	if len(existing) != len(articleIDs) {
		return nil, ErrReorderMismatch
	}
	inCategory := make(map[int64]bool, len(existing))
	for _, id := range existing {
		inCategory[id] = true
	}
	for _, id := range articleIDs {
		if !inCategory[id] {
			return nil, ErrReorderMismatch
		}
		delete(inCategory, id)
	}

	if err := s.repo.UpdateSortOrders(articleIDs, actorID); err != nil {
		return nil, err
	}

	articles, _, err := s.repo.FindAll(dto.ArticleFilter{
		Pagination: helper.Pagination{Page: 1, Limit: len(articleIDs)},
		Category:   category,
	})
	return articles, err
}

func (s *service) DeleteArticle(id int64) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}
