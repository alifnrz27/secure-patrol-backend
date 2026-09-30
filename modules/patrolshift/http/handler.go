package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolshift/dto"
	"secure-patrol-backend/modules/patrolshift/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type PatrolShiftHandler struct {
	service service.PatrolShiftService
	dto     dto.PatrolShiftDto
}

func NewPatrolShiftHandler(service service.PatrolShiftService, dto dto.PatrolShiftDto) *PatrolShiftHandler {
	return &PatrolShiftHandler{service: service, dto: dto}
}

func (h *PatrolShiftHandler) GetShifts(c *fiber.Ctx) error {
	shifts, err := h.service.GetShifts(helper.CurrentScope(c), int64(c.QueryInt("unit_id", 0)))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol shifts success", http.StatusOK, "success", h.dto.ToPatrolShiftDTOs(shifts))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolShiftHandler) GetShift(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrShiftNotFound)
	}

	shift, err := h.service.GetShiftByID(helper.CurrentScope(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol shift success", http.StatusOK, "success", h.dto.ToPatrolShiftDTO(shift))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolShiftHandler) CreateShift(c *fiber.Ctx) error {
	var req dto.CreatePatrolShiftRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	shift, err := h.service.CreateShift(helper.CurrentScope(c), models.PatrolShift{
		Name:      req.Name,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		IsActive:  req.IsActive == nil || *req.IsActive,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create patrol shift success", http.StatusCreated, "success", h.dto.ToPatrolShiftDTO(shift))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *PatrolShiftHandler) UpdateShift(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrShiftNotFound)
	}

	var req dto.UpdatePatrolShiftRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	shift, err := h.service.UpdateShift(helper.CurrentScope(c), int64(id), models.PatrolShift{
		Name:      req.Name,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		IsActive:  *req.IsActive,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update patrol shift success", http.StatusOK, "success", h.dto.ToPatrolShiftDTO(shift))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolShiftHandler) DeleteShift(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrShiftNotFound)
	}

	if err := h.service.DeleteShift(helper.CurrentScope(c), int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete patrol shift success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolShiftHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrShiftNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrUnitRequired):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrShiftOverlap):
		code, message = http.StatusConflict, err.Error()
	case errors.Is(err, service.ErrShiftTimeInvalid), errors.Is(err, service.ErrShiftDurationInvalid):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("patrol shift handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
