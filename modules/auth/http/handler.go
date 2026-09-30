package http

import (
	"errors"
	"math"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/auth/dto"
	"secure-patrol-backend/modules/auth/service"
	userdto "secure-patrol-backend/modules/user/dto"
	"secure-patrol-backend/pkg/log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	service service.AuthService
	dto     dto.AuthDto
	userDto userdto.UserDto
}

func NewAuthHandler(service service.AuthService, dto dto.AuthDto, userDto userdto.UserDto) *AuthHandler {
	return &AuthHandler{service: service, dto: dto, userDto: userDto}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
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

	pair, err := h.service.Login(req.Email, req.Password, clientInfo(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Login success", http.StatusOK, "success", h.dto.ToLoginTokenDTO(pair))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	pair, err := h.service.Refresh(req.RefreshToken, clientInfo(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Refresh token success", http.StatusOK, "success", h.dto.ToTokenDTO(pair))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	user, err := h.service.Me(helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get profile success", http.StatusOK, "success", h.dto.ToProfileDTO(user))
	return c.Status(http.StatusOK).JSON(response)
}

// AppConfig returns the settings the mobile app needs (radius, limits, server time).
func (h *AuthHandler) AppConfig(c *fiber.Ctx) error {
	response := helper.APIResponse("Get app config success", http.StatusOK, "success", h.dto.ToAppConfigDTO(helper.CurrentScope(c).UnitID))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) MyFacePhoto(c *fiber.Ctx) error {
	user, err := h.service.Me(helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	if user.FacePhotoPath == "" {
		response := helper.APIResponse("User has no face photo", http.StatusNotFound, "Error", nil)
		return c.Status(http.StatusNotFound).JSON(response)
	}

	path, err := helper.StoragePath(user.FacePhotoPath)
	if err != nil {
		return h.errorResponse(c, err)
	}

	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendFile(path)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	if err := h.service.Logout(helper.CurrentSessionID(c)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Logout success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	if err := h.service.LogoutAll(helper.CurrentUserID(c)); err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Logout from all devices success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req dto.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	err := h.service.ChangePassword(helper.CurrentUserID(c), helper.CurrentSessionID(c), req.OldPassword, req.Password)
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Change password success", http.StatusOK, "success", nil)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *AuthHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	var lockedErr *service.AccountLockedError
	if errors.As(err, &lockedErr) {
		retryAfter := int(math.Ceil(time.Until(lockedErr.Until).Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Set(fiber.HeaderRetryAfter, strconv.Itoa(retryAfter))
		response := helper.APIResponse(err.Error(), http.StatusTooManyRequests, "Error", fiber.Map{
			"locked_until":        lockedErr.Until,
			"retry_after_seconds": retryAfter,
		})
		return c.Status(http.StatusTooManyRequests).JSON(response)
	}

	switch {
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrSessionInvalid):
		code, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, service.ErrAccountInactive), errors.Is(err, service.ErrPlatformNotAllowed),
		errors.Is(err, service.ErrUnitInactive):
		code, message = http.StatusForbidden, err.Error()
	case errors.Is(err, service.ErrOldPasswordInvalid),
		errors.Is(err, service.ErrSamePassword),
		errors.Is(err, helper.ErrPasswordWeak):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("auth handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}

func clientInfo(c *fiber.Ctx) service.ClientInfo {
	return service.ClientInfo{
		AppClientID: helper.CurrentAppClientID(c),
		AppID:       helper.CurrentAppID(c),
		AppPlatform: helper.CurrentAppPlatform(c),
		UserAgent:   c.Get(fiber.HeaderUserAgent),
		IPAddress:   c.IP(),
	}
}
