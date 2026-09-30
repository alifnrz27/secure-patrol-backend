package helper

import "github.com/gofiber/fiber/v2"

// Keys for values stored in fiber.Ctx Locals by the auth middlewares.
const (
	LocalAppClientID = "x-app-client-id"
	LocalAppID       = "x-app-id"
	LocalAppPlatform = "x-app-platform"
	LocalUserID      = "x-user-id"
	LocalRoleCode    = "x-role-code"
	LocalSessionID   = "x-session-id"
)

func CurrentUserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(LocalUserID).(int64)
	return id
}

func CurrentRoleCode(c *fiber.Ctx) string {
	code, _ := c.Locals(LocalRoleCode).(string)
	return code
}

func CurrentSessionID(c *fiber.Ctx) string {
	id, _ := c.Locals(LocalSessionID).(string)
	return id
}

func CurrentAppClientID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(LocalAppClientID).(int64)
	return id
}

func CurrentAppPlatform(c *fiber.Ctx) string {
	platform, _ := c.Locals(LocalAppPlatform).(string)
	return platform
}

func CurrentAppID(c *fiber.Ctx) string {
	id, _ := c.Locals(LocalAppID).(string)
	return id
}
