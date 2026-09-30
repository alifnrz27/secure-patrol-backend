package middleware

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/auth/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// BearerAuth validates the user access token. It must be used after AppAuth,
// because a token is only accepted from the app it was issued to.
func BearerAuth(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(fiber.HeaderAuthorization)

		if !strings.HasPrefix(authHeader, "Bearer ") {
			response := helper.APIResponse("Unauthorized", http.StatusUnauthorized, "Error", "Token bearer is missing or invalid")
			return c.Status(http.StatusUnauthorized).JSON(response)
		}

		principal, err := authService.Authenticate(authHeader[7:], helper.CurrentAppID(c), helper.CurrentAppPlatform(c))
		if errors.Is(err, service.ErrPlatformNotAllowed) || errors.Is(err, service.ErrUnitInactive) {
			response := helper.APIResponse("Forbidden", http.StatusForbidden, "Error", err.Error())
			return c.Status(http.StatusForbidden).JSON(response)
		}
		if err != nil {
			message := "Token is invalid"
			switch {
			case errors.Is(err, helper.ErrTokenExpired):
				message = "Token is expired"
			case errors.Is(err, service.ErrTokenAppMismatch), errors.Is(err, service.ErrSessionInvalid):
				message = err.Error()
			}

			response := helper.APIResponse("Unauthorized", http.StatusUnauthorized, "Error", message)
			return c.Status(http.StatusUnauthorized).JSON(response)
		}

		c.Locals(helper.LocalUserID, principal.UserID)
		c.Locals(helper.LocalRoleCode, principal.RoleCode)
		c.Locals(helper.LocalSessionID, principal.SessionID)
		c.Locals(helper.LocalUnitID, principal.UnitID)

		return c.Next()
	}
}
