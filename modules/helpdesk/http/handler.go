package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/helpdesk/dto"
	"secure-patrol-backend/modules/helpdesk/service"
	"secure-patrol-backend/pkg/log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type HelpDeskHandler struct {
	service service.HelpDeskService
	dto     dto.HelpDeskDto
}

func NewHelpDeskHandler(service service.HelpDeskService, dto dto.HelpDeskDto) *HelpDeskHandler {
	return &HelpDeskHandler{service: service, dto: dto}
}

func (h *HelpDeskHandler) GetArticles(c *fiber.Ctx) error {
	category := c.Query("category")
	if category != "" && category != models.HelpDeskCategoryRule && category != models.HelpDeskCategoryGuide && category != models.HelpDeskCategoryFAQ {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", []string{"category must be rule, guide or faq"})
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	filter := dto.ArticleFilter{
		Pagination: helper.NewPagination(c),
		Category:   category,
	}
	if isPublished, err := strconv.ParseBool(c.Query("is_published")); err == nil {
		filter.IsPublished = &isPublished
	}

	articles, total, err := h.service.GetArticles(actor(c), filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToArticleDTOs(articles), filter.Pagination, total)
	response := helper.APIResponse("Get help desk articles success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *HelpDeskHandler) GetArticle(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrArticleNotFound)
	}

	article, err := h.service.GetArticle(actor(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get help desk article success", http.StatusOK, "success", h.dto.ToArticleDTO(article))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *HelpDeskHandler) CreateArticle(c *fiber.Ctx) error {
	var req dto.CreateHelpDeskArticleRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	article, err := h.service.CreateArticle(models.HelpDeskArticle{
		Category:    req.Category,
		Title:       req.Title,
		Content:     req.Content,
		IsPublished: req.IsPublished == nil || *req.IsPublished,
	}, req.SortOrder, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create help desk article success", http.StatusCreated, "success", h.dto.ToArticleDTO(article))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *HelpDeskHandler) UpdateArticle(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrArticleNotFound)
	}

	var req dto.UpdateHelpDeskArticleRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	article, err := h.service.UpdateArticle(int64(id), models.HelpDeskArticle{
		Category:    req.Category,
		Title:       req.Title,
		Content:     req.Content,
		IsPublished: *req.IsPublished,
	}, req.SortOrder, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update help desk article success", http.StatusOK, "success", h.dto.ToArticleDTO(article))
	return c.Status(http.StatusOK).JSON(response)
}

// ReorderArticles saves a new order for one category, e.g. after drag and drop.
func (h *HelpDeskHandler) ReorderArticles(c *fiber.Ctx) error {
	var req dto.ReorderHelpDeskArticlesRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	articles, err := h.service.ReorderArticles(req.Category, req.ArticleIDs, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Reorder help desk articles success", http.StatusOK, "success", h.dto.ToArticleDTOs(articles))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *HelpDeskHandler) DeleteArticle(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrArticleNotFound)
	}

	if err := h.service.DeleteArticle(int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete help desk article success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *HelpDeskHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrArticleNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrReorderMismatch):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("help desk handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}

func actor(c *fiber.Ctx) service.Actor {
	return service.Actor{
		UserID:   helper.CurrentUserID(c),
		RoleCode: helper.CurrentRoleCode(c),
	}
}
