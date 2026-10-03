package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	licenseservice "secure-patrol-backend/modules/license/service"
	"secure-patrol-backend/modules/unit/dto"
	"secure-patrol-backend/modules/unit/repository"
	"secure-patrol-backend/modules/unit/service"
	"secure-patrol-backend/pkg/log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type UnitHandler struct {
	service service.UnitService
	dto     dto.UnitDto
}

func NewUnitHandler(service service.UnitService, dto dto.UnitDto) *UnitHandler {
	return &UnitHandler{service: service, dto: dto}
}

// GetUnits lists every unit for head office users, and only their own unit for unit users.
func (h *UnitHandler) GetUnits(c *fiber.Ctx) error {
	filter := repository.UnitFilter{
		Pagination: helper.NewPagination(c),
		OnlyID:     helper.CurrentScope(c).UnitID,
	}
	if isActive, err := strconv.ParseBool(c.Query("is_active")); err == nil {
		filter.IsActive = &isActive
	}

	units, total, err := h.service.GetUnits(filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToUnitDTOs(units), filter.Pagination, total)
	response := helper.APIResponse("Get units success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UnitHandler) GetUnit(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || !helper.CurrentScope(c).CanAccessUnit(int64(id)) {
		return h.errorResponse(c, service.ErrUnitNotFound)
	}

	unit, err := h.service.GetUnitByID(int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get unit success", http.StatusOK, "success", h.dto.ToUnitDTO(unit))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UnitHandler) CreateUnit(c *fiber.Ctx) error {
	var req dto.CreateUnitRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	unit, err := h.service.CreateUnit(models.Unit{
		Code:      req.Code,
		Name:      req.Name,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		IsActive:  req.IsActive == nil || *req.IsActive,
	}, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create unit success", http.StatusCreated, "success", h.dto.ToUnitDTO(unit))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *UnitHandler) UpdateUnit(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUnitNotFound)
	}

	var req dto.UpdateUnitRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	unit, err := h.service.UpdateUnit(int64(id), models.Unit{
		Code:      req.Code,
		Name:      req.Name,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		IsActive:  *req.IsActive,
	}, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update unit success", http.StatusOK, "success", h.dto.ToUnitDTO(unit))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UnitHandler) DeleteUnit(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUnitNotFound)
	}

	if err := h.service.DeleteUnit(int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete unit success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UnitHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrUnitNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, licenseservice.ErrUnitLimitReached), errors.Is(err, licenseservice.ErrLicenseInactive):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrUnitCodeTaken), errors.Is(err, service.ErrUnitInUse):
		code, message = http.StatusConflict, err.Error()
	default:
		log.Errorf("unit handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
