package http

import (
	"errors"
	"mime/multipart"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/user/dto"
	"secure-patrol-backend/modules/user/service"
	"secure-patrol-backend/pkg/facedetect"
	"secure-patrol-backend/pkg/log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	service service.UserService
	dto     dto.UserDto
}

func NewUserHandler(service service.UserService, dto dto.UserDto) *UserHandler {
	return &UserHandler{service: service, dto: dto}
}

func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	filter := dto.UserFilter{
		Pagination: helper.NewPagination(c),
		RoleID:     int64(c.QueryInt("role_id", 0)),
		UnitID:     helper.UnitFilterFromQuery(c),
	}
	// ?head_office=true lists the head office users (no unit); head office users only.
	if headOffice, _ := strconv.ParseBool(c.Query("head_office")); headOffice && helper.CurrentScope(c).IsCentral() {
		filter.HeadOfficeOnly = true
	}

	if isActive, err := strconv.ParseBool(c.Query("is_active")); err == nil {
		filter.IsActive = &isActive
	}

	users, total, err := h.service.GetUsers(filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToUserDTOs(users), filter.Pagination, total)
	response := helper.APIResponse("Get users success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUserNotFound)
	}

	user, err := h.service.GetUserByID(actor(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get user success", http.StatusOK, "success", h.dto.ToUserDTO(user))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	// Mobile keyboards often add a trailing space after autocompleting an email.
	req.Email = strings.TrimSpace(req.Email)

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user, err := h.service.CreateUser(actor(c), models.User{
		Name:     req.Name,
		Email:    req.Email,
		RoleID:   req.RoleID,
		UnitID:   optionalID(req.UnitID),
		IsActive: isActive,
	}, req.Password, facePhoto(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Create user success", http.StatusCreated, "success", h.dto.ToUserDTO(user))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUserNotFound)
	}

	var req dto.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	req.Email = strings.TrimSpace(req.Email)

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	user, err := h.service.UpdateUser(actor(c), int64(id), models.User{
		Name:     req.Name,
		Email:    req.Email,
		RoleID:   req.RoleID,
		UnitID:   optionalID(req.UnitID),
		IsActive: *req.IsActive,
	}, facePhoto(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update user success", http.StatusOK, "success", h.dto.ToUserDTO(user))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUserNotFound)
	}

	if err := h.service.DeleteUser(actor(c), int64(id)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Delete user success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUserNotFound)
	}

	var req dto.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	if err := h.service.ResetPassword(actor(c), int64(id), req.Password); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Reset password success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *UserHandler) GetFacePhoto(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrUserNotFound)
	}

	path, err := h.service.GetFacePhotoPath(actor(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendFile(path)
}

func (h *UserHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrFacePhotoNotExists):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, service.ErrEmailTaken):
		code, message = http.StatusConflict, err.Error()
	case errors.Is(err, service.ErrForbiddenRole):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrRoleInvalid),
		errors.Is(err, service.ErrUnitRequired),
		errors.Is(err, service.ErrUnitInvalid),
		errors.Is(err, service.ErrFacePhotoRequired),
		errors.Is(err, service.ErrCannotModifySelf),
		errors.Is(err, helper.ErrFileTooLarge),
		errors.Is(err, helper.ErrFileTypeNotAllowed),
		errors.Is(err, helper.ErrPasswordWeak),
		errors.Is(err, facedetect.ErrImageUnreadable),
		errors.Is(err, facedetect.ErrNoFace),
		errors.Is(err, facedetect.ErrMultipleFaces),
		errors.Is(err, facedetect.ErrFaceTooSmall),
		errors.Is(err, facedetect.ErrFaceNotFrontal):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("user handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}

func actor(c *fiber.Ctx) service.Actor {
	return service.Actor{
		UserID:   helper.CurrentUserID(c),
		RoleCode: helper.CurrentRoleCode(c),
		UnitID:   helper.CurrentScope(c).UnitID,
	}
}

func optionalID(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	return &id
}

func facePhoto(c *fiber.Ctx) *multipart.FileHeader {
	file, err := c.FormFile("face_photo")
	if err != nil {
		return nil
	}
	return file
}
