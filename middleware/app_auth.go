package middleware

import (
	"errors"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/appclient/service"
	licenseservice "secure-patrol-backend/modules/license/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

// Headers every client (mobile, web, server) must send on every request.
const (
	HeaderAppID     = "X-App-Id"
	HeaderTimestamp = "X-Timestamp"
	HeaderNonce     = "X-Nonce"
	HeaderSignature = "X-Signature"
)

// AppAuth verifies that the request comes from a registered app client and was
// signed with its app key. See docs/AUTH.md for the signing algorithm.
func AppAuth(appClientService service.AppClientService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		client, err := appClientService.VerifyRequest(service.SignedRequest{
			AppID:      c.Get(HeaderAppID),
			Timestamp:  c.Get(HeaderTimestamp),
			Nonce:      c.Get(HeaderNonce),
			Signature:  c.Get(HeaderSignature),
			Method:     c.Method(),
			RequestURI: c.OriginalURL(),
			Body:       c.Body(),
		})
		if err != nil {
			if !isAppAuthError(err) {
				log.Errorf("app auth: %v", err)
				response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
				return c.Status(http.StatusInternalServerError).JSON(response)
			}

			response := helper.APIResponse("Unauthorized app", http.StatusUnauthorized, "Error", err.Error())
			return c.Status(http.StatusUnauthorized).JSON(response)
		}

		// App clients above the license limit (the newest ones) do not work,
		// also when they were added or re-activated directly in the database.
		if licenses := licenseservice.Instance(); licenses != nil {
			allowed, err := licenses.AppClientAllowed(client.ID)
			if err != nil {
				log.Errorf("app auth: license check: %v", err)
				response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
				return c.Status(http.StatusInternalServerError).JSON(response)
			}
			if !allowed {
				response := helper.APIResponse("Forbidden", http.StatusForbidden, "Error", licenseservice.ErrAppClientOverLimit.Error())
				return c.Status(http.StatusForbidden).JSON(response)
			}
		}

		c.Locals(helper.LocalAppClientID, client.ID)
		c.Locals(helper.LocalAppID, client.AppID)
		c.Locals(helper.LocalAppPlatform, client.Platform)

		return c.Next()
	}
}

func isAppAuthError(err error) bool {
	for _, target := range []error{
		service.ErrAppCredentialMissing,
		service.ErrAppCredentialInvalid,
		service.ErrAppClientDisabled,
		service.ErrTimestampInvalid,
		service.ErrNonceInvalid,
		service.ErrNonceReused,
		service.ErrSignatureInvalid,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
