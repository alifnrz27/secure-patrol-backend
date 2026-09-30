package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/role/dto"
	"secure-patrol-backend/modules/role/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type RoleHandler struct {
	service service.RoleService
	dto     dto.RoleDto
}

func NewRoleHandler(service service.RoleService, dto dto.RoleDto) *RoleHandler {
	return &RoleHandler{service: service, dto: dto}
}

func (h *RoleHandler) GetRoles(c *fiber.Ctx) error {
	pagination := helper.NewPagination(c)

	roles, total, err := h.service.GetRoles(pagination)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToRoleDTOs(roles), pagination, total)
	response := helper.APIResponse("Get roles success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *RoleHandler) GetRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrRoleNotFound)
	}

	role, err := h.service.GetRoleByID(int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get role success", http.StatusOK, "success", h.dto.ToRoleDTO(role))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *RoleHandler) CreateRole(c *fiber.Ctx) error {
	var req dto.CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	role, err := h.service.CreateRole(models.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    isActive,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create role success", http.StatusCreated, "success", h.dto.ToRoleDTO(role))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *RoleHandler) UpdateRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrRoleNotFound)
	}

	var req dto.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	role, err := h.service.UpdateRole(int64(id), models.Role{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    *req.IsActive,
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update role success", http.StatusOK, "success", h.dto.ToRoleDTO(role))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *RoleHandler) DeleteRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrRoleNotFound)
	}

	if err := h.service.DeleteRole(int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete role success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *RoleHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrRoleNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrRoleCodeTaken):
		code, message = http.StatusConflict, err.Error()
	case errors.Is(err, service.ErrRoleCodeInvalid):
		code, message = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, service.ErrRoleSystemLocked), errors.Is(err, service.ErrRoleInUse):
		code, message = http.StatusBadRequest, err.Error()
	default:
		log.Errorf("role handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
