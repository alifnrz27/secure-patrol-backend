package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolpoint/dto"
	"secure-patrol-backend/modules/patrolpoint/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type PatrolPointHandler struct {
	service service.PatrolPointService
	dto     dto.PatrolPointDto
}

func NewPatrolPointHandler(service service.PatrolPointService, dto dto.PatrolPointDto) *PatrolPointHandler {
	return &PatrolPointHandler{service: service, dto: dto}
}

func (h *PatrolPointHandler) GetPatrolPoints(c *fiber.Ctx) error {
	pagination := helper.NewPagination(c)

	points, total, err := h.service.GetPatrolPoints(helper.CurrentScope(c), pagination, int64(c.QueryInt("unit_id", 0)), int64(c.QueryInt("area_id", 0)))
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToPatrolPointDTOs(points), pagination, total)
	response := helper.APIResponse("Get patrol points success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolPointHandler) GetPatrolPoint(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrPatrolPointNotFound)
	}

	point, err := h.service.GetPatrolPointByID(helper.CurrentScope(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol point success", http.StatusOK, "success", h.dto.ToPatrolPointDTO(point))
	return c.Status(http.StatusOK).JSON(response)
}

// GetPatrolPointByNFC looks up the patrol point of a scanned NFC tag (?code=...).
func (h *PatrolPointHandler) GetPatrolPointByNFC(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", []string{"code failed on 'required' validation"})
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	point, err := h.service.GetPatrolPointByNFCCode(helper.CurrentScope(c), code)
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol point success", http.StatusOK, "success", h.dto.ToPatrolPointDTO(point))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolPointHandler) CreatePatrolPoint(c *fiber.Ctx) error {
	var req dto.CreatePatrolPointRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	point, err := h.service.CreatePatrolPoint(helper.CurrentScope(c), models.PatrolPoint{
		Name:                     req.Name,
		AreaID:                   req.AreaID,
		Location:                 req.Location,
		NFCCode:                  req.NFCCode,
		Latitude:                 *req.Latitude,
		Longitude:                *req.Longitude,
		IsLocationMatchRequired:  req.IsLocationMatchRequired != nil && *req.IsLocationMatchRequired,
		IsFaceValidationRequired: req.IsFaceValidationRequired != nil && *req.IsFaceValidationRequired,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create patrol point success", http.StatusCreated, "success", h.dto.ToPatrolPointDTO(point))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *PatrolPointHandler) UpdatePatrolPoint(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrPatrolPointNotFound)
	}

	var req dto.UpdatePatrolPointRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	point, err := h.service.UpdatePatrolPoint(helper.CurrentScope(c), int64(id), models.PatrolPoint{
		Name:                     req.Name,
		AreaID:                   req.AreaID,
		Location:                 req.Location,
		NFCCode:                  req.NFCCode,
		Latitude:                 *req.Latitude,
		Longitude:                *req.Longitude,
		IsLocationMatchRequired:  *req.IsLocationMatchRequired,
		IsFaceValidationRequired: *req.IsFaceValidationRequired,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update patrol point success", http.StatusOK, "success", h.dto.ToPatrolPointDTO(point))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolPointHandler) DeletePatrolPoint(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrPatrolPointNotFound)
	}

	if err := h.service.DeletePatrolPoint(helper.CurrentScope(c), int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete patrol point success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolPointHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrPatrolPointNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrUnitRequired):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrAreaInvalid):
		code, message = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, service.ErrNFCCodeTaken):
		code, message = http.StatusConflict, err.Error()
	default:
		log.Errorf("patrol point handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
