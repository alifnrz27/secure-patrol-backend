// Package docs serves the OpenAPI documentation (Swagger UI) of the API.
// It is only registered when APP_ENV is a development environment.
package docs

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

var (
	//go:embed index.html
	indexHTML []byte

	//go:embed openapi.yaml
	openAPISpec []byte

	//go:embed signer.js
	signerJS []byte
)

func Routes(app *fiber.App) {
	app.Get("/docs", serve(indexHTML, "text/html; charset=utf-8"))
	app.Get("/docs/openapi.yaml", serve(openAPISpec, "application/yaml; charset=utf-8"))
	app.Get("/docs/signer.js", serve(signerJS, "text/javascript; charset=utf-8"))
}

func serve(content []byte, contentType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, contentType)
		c.Set(fiber.HeaderCacheControl, "no-store")
		c.Set("X-Robots-Tag", "noindex, nofollow")
		return c.Send(content)
	}
}
