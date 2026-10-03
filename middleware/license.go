package middleware

import (
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	licenseservice "secure-patrol-backend/modules/license/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// licenseFreePaths stay available to the Super-Admin while the license is not
// active, so the session and the license page keep working.
var licenseFreePaths = []string{"/api/v1/auth/", "/api/v1/app-config", "/api/v1/license"}

// LicenseGuard blocks every request while the license is missing, expired or
// invalid, except the Super-Admin's session and license endpoints. It must be
// used after BearerAuth.
func LicenseGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		licenses := licenseservice.Instance()
		if licenses == nil {
			return c.Next()
		}
		status := licenses.Status()
		if !status.Locked() {
			return c.Next()
		}

		if helper.CurrentRoleCode(c) == models.RoleSuperAdmin {
			for _, prefix := range licenseFreePaths {
				if strings.HasPrefix(c.Path(), prefix) {
					return c.Next()
				}
			}
		}

		response := helper.APIResponse(licenseservice.ErrLicenseInactive.Error(), http.StatusForbidden, "Error", fiber.Map{
			"license_status": status.State,
		})
		return c.Status(http.StatusForbidden).JSON(response)
	}
}
