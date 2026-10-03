package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrolarea/dto"
	"secure-patrol-backend/modules/patrolarea/service"
	"secure-patrol-backend/pkg/log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type PatrolAreaHandler struct {
	service service.PatrolAreaService
}

func NewPatrolAreaHandler(service service.PatrolAreaService) *PatrolAreaHandler {
	return &PatrolAreaHandler{service: service}
}

func (h *PatrolAreaHandler) GetAreas(c *fiber.Ctx) error {
	pagination := helper.NewPagination(c)
	areas, total, err := h.service.GetAreas(helper.CurrentScope(c), pagination, int64(c.QueryInt("unit_id", 0)))
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(dto.ToPatrolAreaDTOs(areas), pagination, total)
	response := helper.APIResponse("Get patrol areas success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolAreaHandler) GetArea(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAreaNotFound)
	}
	area, err := h.service.GetAreaByID(helper.CurrentScope(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}
	response := helper.APIResponse("Get patrol area success", http.StatusOK, "success", dto.ToPatrolAreaDTO(area))
	return c.Status(http.StatusOK).JSON(response)
}

// parse reads and validates the request; ok is false when an error response was sent.
func (h *PatrolAreaHandler) parse(c *fiber.Ctx) (area models.PatrolArea, ok bool, err error) {
	var req dto.PatrolAreaRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return area, false, c.Status(http.StatusBadRequest).JSON(response)
	}
	req.Name = strings.TrimSpace(req.Name)
	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return area, false, c.Status(http.StatusUnprocessableEntity).JSON(response)
	}
	return models.PatrolArea{Name: req.Name, Description: req.Description}, true, nil
}

func (h *PatrolAreaHandler) CreateArea(c *fiber.Ctx) error {
	input, ok, err := h.parse(c)
	if !ok {
		return err
	}
	area, err := h.service.CreateArea(helper.CurrentScope(c), input)
	if err != nil {
		return h.errorResponse(c, err)
	}
	response := helper.APIResponse("Create patrol area success", http.StatusCreated, "success", dto.ToPatrolAreaDTO(area))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *PatrolAreaHandler) UpdateArea(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAreaNotFound)
	}
	input, ok, err := h.parse(c)
	if !ok {
		return err
	}
	area, err := h.service.UpdateArea(helper.CurrentScope(c), int64(id), input)
	if err != nil {
		return h.errorResponse(c, err)
	}
	response := helper.APIResponse("Update patrol area success", http.StatusOK, "success", dto.ToPatrolAreaDTO(area))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolAreaHandler) DeleteArea(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrAreaNotFound)
	}
	if err := h.service.DeleteArea(helper.CurrentScope(c), int64(id)); err != nil {
		return h.errorResponse(c, err)
	}
	response := helper.APIResponse("Delete patrol area success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolAreaHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrAreaNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrUnitRequired):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrAreaNameTaken), errors.Is(err, service.ErrAreaInUse):
		code, message = http.StatusConflict, err.Error()
	default:
		log.Errorf("patrol area handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
