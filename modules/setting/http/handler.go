package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/setting/dto"
	"secure-patrol-backend/modules/setting/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type SettingHandler struct {
	service service.SettingService
	dto     dto.SettingDto
}

func NewSettingHandler(service service.SettingService, dto dto.SettingDto) *SettingHandler {
	return &SettingHandler{service: service, dto: dto}
}

func (h *SettingHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.service.List()
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get settings success", http.StatusOK, "success", h.dto.ToSettingDTOs(settings))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *SettingHandler) UpdateSettings(c *fiber.Ctx) error {
	var req dto.UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	settings, err := h.service.Update(req.Values, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update settings success", http.StatusOK, "success", h.dto.ToSettingDTOs(settings))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *SettingHandler) errorResponse(c *fiber.Ctx, err error) error {
	var validationErr *service.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", validationErr.Messages)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	case errors.Is(err, service.ErrNoChanges):
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", []string{err.Error()})
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	log.Errorf("setting handler: %v", err)
	response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
	return c.Status(http.StatusInternalServerError).JSON(response)
}
