package middleware

import (
	"net/http"
	"secure-patrol-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// RequireRoles only lets users whose role code is in roles through.
// It must be used after BearerAuth.
func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !helper.Includes(roles, helper.CurrentRoleCode(c)) {
			response := helper.APIResponse("Forbidden", http.StatusForbidden, "Error", "You do not have access to this resource")
			return c.Status(http.StatusForbidden).JSON(response)
		}
		return c.Next()
	}
}
