package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/appclient/dto"
	"secure-patrol-backend/modules/appclient/service"
	"secure-patrol-backend/pkg/log"
	"time"

	"github.com/gofiber/fiber/v2"
)

type AppClientHandler struct {
	service service.AppClientService
	dto     dto.AppClientDto
}

func NewAppClientHandler(service service.AppClientService, dto dto.AppClientDto) *AppClientHandler {
	return &AppClientHandler{service: service, dto: dto}
}

func (h *AppClientHandler) GetAppClients(c *fiber.Ctx) error {
	pagination := helper.NewPagination(c)

	clients, total, err := h.service.GetAppClients(pagination)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToAppClientDTOs(clients), pagination, total)
	response := helper.APIResponse("Get app clients success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AppClientHandler) GetAppClient(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAppClientNotFound)
	}

	client, err := h.service.GetAppClientByID(int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get app client success", http.StatusOK, "success", h.dto.ToAppClientDTO(client))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AppClientHandler) CreateAppClient(c *fiber.Ctx) error {
	var req dto.CreateAppClientRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	createdBy := helper.CurrentUserID(c)
	client, appKey, err := h.service.CreateAppClient(models.AppClient{
		Name:        req.Name,
		Platform:    req.Platform,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   &createdBy,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create app client success", http.StatusCreated, "success", h.dto.ToCredentialDTO(client, appKey))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *AppClientHandler) UpdateAppClient(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAppClientNotFound)
	}

	var req dto.UpdateAppClientRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	client, err := h.service.UpdateAppClient(int64(id), models.AppClient{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    *req.IsActive,
		ExpiresAt:   req.ExpiresAt,
	}, helper.CurrentAppClientID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update app client success", http.StatusOK, "success", h.dto.ToAppClientDTO(client))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AppClientHandler) RotateKey(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAppClientNotFound)
	}

	var req dto.RotateKeyRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
			return c.Status(http.StatusBadRequest).JSON(response)
		}
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	client, appKey, err := h.service.RotateKey(int64(id), time.Duration(req.GracePeriodHours)*time.Hour)
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Rotate app key success", http.StatusOK, "success", h.dto.ToCredentialDTO(client, appKey))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AppClientHandler) DeleteAppClient(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAppClientNotFound)
	}

	if err := h.service.DeleteAppClient(int64(id), helper.CurrentAppClientID(c)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete app client success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AppClientHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrAppClientNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrPlatformInvalid), errors.Is(err, service.ErrCannotModifyActive):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("app client handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
