package config

import (
	"errors"
	"log"
	"os"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/middleware"
	appclientrepository "secure-patrol-backend/modules/appclient/repository"
	appclientservice "secure-patrol-backend/modules/appclient/service"
	auditlogrepository "secure-patrol-backend/modules/auditlog/repository"
	auditlogservice "secure-patrol-backend/modules/auditlog/service"
	authrepository "secure-patrol-backend/modules/auth/repository"
	authservice "secure-patrol-backend/modules/auth/service"
	applog "secure-patrol-backend/pkg/log"
	"secure-patrol-backend/pkg/nonce"
	"secure-patrol-backend/routes"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
)

func Route(db *gorm.DB) {

	proxyConfig := proxyConfig()

	app := fiber.New(fiber.Config{
		ErrorHandler:            jsonErrorHandler,
		ProxyHeader:             proxyConfig.ProxyHeader,
		EnableIPValidation:      proxyConfig.EnableIPValidation,
		EnableTrustedProxyCheck: proxyConfig.EnableTrustedProxyCheck,
		TrustedProxies:          proxyConfig.TrustedProxies,
		// Leaves room for 3 patrol photos of 5 MB each plus the other multipart fields.
		BodyLimit: 20 * 1024 * 1024,
		// Keep the raw multipart body. When fasthttp pre-parses it, Body() re-serializes
		// the form in random map order and request signatures can no longer be verified.
		DisablePreParseMultipartForm: true,
	})
	// Use the cors middleware to allow all origins and methods
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization," +
			middleware.HeaderAppID + "," + middleware.HeaderTimestamp + "," +
			middleware.HeaderNonce + "," + middleware.HeaderSignature,
		// Browsers hide the Date header from cross-origin JavaScript unless it is
		// exposed; the web app needs it to correct its clock for signed requests.
		ExposeHeaders: "Date",
	}))

	// Health check for load balancers, no app credential required
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// API documentation, only registered when APP_ENV=development
	routes.DocsRouter(app)

	// Create a new Fiber app for the "api/v1" prefix group
	api := fiber.New(fiber.Config{
		ErrorHandler:            jsonErrorHandler,
		ProxyHeader:             proxyConfig.ProxyHeader,
		EnableIPValidation:      proxyConfig.EnableIPValidation,
		EnableTrustedProxyCheck: proxyConfig.EnableTrustedProxyCheck,
		TrustedProxies:          proxyConfig.TrustedProxies,
	})

	appClientService := appclientservice.NewAppClientService(
		appclientrepository.NewAppClientRepository(db),
		nonce.Default(),
	)
	authService := authservice.NewAuthService(authrepository.NewAuthRepository(db))
	auditLogService := auditlogservice.NewAuditLogService(auditlogrepository.NewAuditLogRepository(db))

	// Every request must be signed by a registered app client (mobile, web, server)
	api.Use(middleware.AppAuth(appClientService))

	// Public group api's (app signature only)
	routes.PublicRouter(api, db)

	// Authenticated group api's (app signature + user access token)
	api.Use(middleware.BearerAuth(authService))

	// Record every successful create, update and delete made by a logged in user
	api.Use(middleware.AuditLog(auditLogService))

	al := &lumberjack.Logger{
		Filename:  "./logs/access/access.log",
		MaxAge:    1,
		LocalTime: true,
		Compress:  true}

	go customLogger(al)
	api.Use(logger.New(logger.Config{
		Output: al,
	}))

	routes.AuthRouter(api, db)
	routes.UnitRouter(api, db)
	routes.RoleRouter(api, db)
	routes.UserRouter(api, db)
	routes.AppClientRouter(api, db)
	routes.PatrolPointRouter(api, db)
	routes.PatrolShiftRouter(api, db)
	routes.PatrolRouter(api, db)
	routes.HelpDeskRouter(api, db)
	routes.AuditLogRouter(api, db)
	routes.SettingRouter(api, db)

	app.Mount("/api/v1", api)

	log.Fatalln(app.Listen(":" + os.Getenv("PORT")))
}

// Logger is the access loger for request/response HTTP from client
// only heandler defined after this method that will captured.
func customLogger(accesslogger *lumberjack.Logger) {

	midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+1, 0, 0, 0, 0, time.Local)
	durationUntilMidnight := midnight.Sub(time.Now())
	ticker := time.NewTicker(durationUntilMidnight)

	defer ticker.Stop()

	for range ticker.C {
		accesslogger.Rotate()
		ticker.Reset(2 * time.Minute)
	}

}

// jsonErrorHandler answers framework errors (unknown route, body too large, ...)
// in the same JSON envelope as the handlers instead of plain text.
func jsonErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	switch code {
	case fiber.StatusRequestEntityTooLarge:
		message = "Request body is too large (maximum 20 MB, each photo maximum 5 MB)"
	case fiber.StatusInternalServerError:
		applog.Errorf("unhandled error on %s %s: %v", c.Method(), c.Path(), err)
	}

	return c.Status(code).JSON(helper.APIResponse(message, code, "Error", nil))
}

// proxyConfig makes c.IP() return the real client IP behind a reverse proxy.
// PROXY_HEADER (e.g. X-Forwarded-For) is only trusted from TRUSTED_PROXIES when
// that list is set; without it, clients could fake their IP through the header.
func proxyConfig() fiber.Config {
	header := strings.TrimSpace(os.Getenv("PROXY_HEADER"))
	if header == "" {
		return fiber.Config{}
	}

	var trusted []string
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			trusted = append(trusted, proxy)
		}
	}
	if len(trusted) == 0 {
		log.Println("WARNING: PROXY_HEADER is set without TRUSTED_PROXIES; client IPs in logs can be spoofed")
	}

	return fiber.Config{
		ProxyHeader:             header,
		EnableIPValidation:      true,
		EnableTrustedProxyCheck: len(trusted) > 0,
		TrustedProxies:          trusted,
	}
}
