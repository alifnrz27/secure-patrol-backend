package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/branding/dto"
	"secure-patrol-backend/modules/branding/service"
	"secure-patrol-backend/pkg/log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type BrandingHandler struct {
	service service.BrandingService
}

func NewBrandingHandler(service service.BrandingService) *BrandingHandler {
	return &BrandingHandler{service: service}
}

func (h *BrandingHandler) GetBranding(c *fiber.Ctx) error {
	branding, err := h.service.Get()
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get branding success", http.StatusOK, "success", dto.ToBrandingDTO(branding))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *BrandingHandler) UpdateBranding(c *fiber.Ctx) error {
	var req dto.UpdateBrandingRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	req.AppName = strings.TrimSpace(req.AppName)
	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	logo, err := c.FormFile("logo")
	if err != nil {
		logo = nil
	}

	branding, err := h.service.Update(service.UpdateInput{
		AppName:    req.AppName,
		Logo:       logo,
		RemoveLogo: req.RemoveLogo,
	}, helper.CurrentUserID(c))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Update branding success", http.StatusOK, "success", dto.ToBrandingDTO(branding))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *BrandingHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, service.ErrLogoTooLarge),
		errors.Is(err, service.ErrAppNameMissing),
		errors.Is(err, helper.ErrFileTooLarge),
		errors.Is(err, helper.ErrFileTypeNotAllowed):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("branding handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}
