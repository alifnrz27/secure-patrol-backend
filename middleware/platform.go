package middleware

import (
	"net/http"
	"secure-patrol-backend/helper"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequirePlatforms only lets requests signed by an app client of one of the
// given platforms through. It must be used after AppAuth.
func RequirePlatforms(platforms ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		platform, _ := c.Locals(helper.LocalAppPlatform).(string)
		if !helper.Includes(platforms, platform) {
			response := helper.APIResponse(
				"Forbidden",
				http.StatusForbidden,
				"Error",
				"This feature is only available on the "+strings.Join(platforms, " or ")+" platform",
			)
			return c.Status(http.StatusForbidden).JSON(response)
		}
		return c.Next()
	}
}
