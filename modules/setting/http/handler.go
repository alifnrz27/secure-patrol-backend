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

var errUnitNotFound = errors.New("unit not found")

type SettingHandler struct {
	service service.SettingService
	dto     dto.SettingDto
	// unitExists reports whether a unit exists (used for ?unit_id of head office users).
	unitExists func(unitID int64) (bool, error)
}

func NewSettingHandler(service service.SettingService, dto dto.SettingDto, unitExists func(unitID int64) (bool, error)) *SettingHandler {
	return &SettingHandler{service: service, dto: dto, unitExists: unitExists}
}

// GetSettings returns the settings of the user's unit for unit users. Head
// office users get the global settings, or a unit's settings with ?unit_id.
func (h *SettingHandler) GetSettings(c *fiber.Ctx) error {
	var unitID *int64
	if filter := helper.UnitFilterFromQuery(c); filter > 0 {
		unitID = &filter
	}

	if unitID != nil && helper.CurrentScope(c).IsCentral() {
		exists, err := h.unitExists(*unitID)
		if err != nil {
			return h.errorResponse(c, err)
		}
		if !exists {
			return h.errorResponse(c, errUnitNotFound)
		}
	}

	settings, err := h.service.List(unitID)
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get settings success", http.StatusOK, "success", h.dto.ToSettingDTOs(settings, unitID != nil))
	return c.Status(http.StatusOK).JSON(response)
}

// UpdateSettings changes the unit's own values for unit managers, and the
// global values (used by every unit without its own value) for the Super-Admin.
func (h *SettingHandler) UpdateSettings(c *fiber.Ctx) error {
	var req dto.UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	unitID := helper.CurrentScope(c).UnitID
	settings, err := h.service.Update(unitID, req.Values, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update settings success", http.StatusOK, "success", h.dto.ToSettingDTOs(settings, unitID != nil))
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
	case errors.Is(err, errUnitNotFound):
		response := helper.APIResponse(err.Error(), http.StatusNotFound, "Error", nil)
		return c.Status(http.StatusNotFound).JSON(response)
	}

	log.Errorf("setting handler: %v", err)
	response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
	return c.Status(http.StatusInternalServerError).JSON(response)
}
