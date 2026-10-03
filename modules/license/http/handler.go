package http

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/middleware"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/license/service"
	"secure-patrol-backend/pkg/license"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type LicenseHandler struct {
	service service.LicenseService
}

func NewLicenseHandler(service service.LicenseService) *LicenseHandler {
	return &LicenseHandler{service: service}
}

type InstallLicenseRequest struct {
	Code string `json:"code" validate:"required,max=4096"`
}

func (h *LicenseHandler) respond(c *fiber.Ctx, message string, status service.Status) error {
	usage, err := h.service.Usage()
	if err != nil {
		log.Errorf("license handler: %v", err)
		response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
		return c.Status(http.StatusInternalServerError).JSON(response)
	}
	response := helper.APIResponse(message, http.StatusOK, "success", ToLicenseDTO(status, usage))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *LicenseHandler) GetLicense(c *fiber.Ctx) error {
	return h.respond(c, "Get license success", h.service.Status())
}

// GetPublicStatus tells the apps, before login, whether the system has an
// active license. It needs a signed app request but no user.
func (h *LicenseHandler) GetPublicStatus(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	response := helper.APIResponse("Get license status success", http.StatusOK, "success", ToPublicStatusDTO(h.service.Status()))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *LicenseHandler) InstallLicense(c *fiber.Ctx) error {
	var req InstallLicenseRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	actorID := helper.CurrentUserID(c)
	status, err := h.service.Install(req.Code, &actorID)
	if err != nil {
		code, message := http.StatusInternalServerError, "Internal server error"
		switch {
		case errors.Is(err, license.ErrMalformed), errors.Is(err, license.ErrSignatureInvalid),
			errors.Is(err, license.ErrPayloadInvalid), errors.Is(err, service.ErrOtherInstallation),
			errors.Is(err, service.ErrLicenseAlreadyEnded):
			code, message = http.StatusUnprocessableEntity, err.Error()
		default:
			log.Errorf("license handler: %v", err)
		}
		response := helper.APIResponse(message, code, "Error", nil)
		return c.Status(code).JSON(response)
	}

	return h.respond(c, "Install license success", status)
}

// LicensePublicRoutes only need a signed app request (no login): the apps check
// the license before showing the login page.
func LicensePublicRoutes(app *fiber.App, handler *LicenseHandler) {
	app.Get("/license/status", handler.GetPublicStatus)
}

// LicenseRoutes are for the Super-Admin on the web; they stay available while
// the license is not active (see middleware.LicenseGuard).
func LicenseRoutes(app *fiber.App, handler *LicenseHandler) {
	webOnly := middleware.RequirePlatforms(models.PlatformWeb)
	superAdmin := middleware.RequireRoles(models.RoleSuperAdmin)

	app.Get("/license", webOnly, superAdmin, handler.GetLicense)
	app.Put("/license", webOnly, superAdmin, handler.InstallLicense)
}
