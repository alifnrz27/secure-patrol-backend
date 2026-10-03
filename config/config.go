// config/config.go
package config

import (
	"log"
	"net"
	"net/url"
	"os"
	licenseservice "secure-patrol-backend/modules/license/service"
	"strconv"

	"secure-patrol-backend/migration"
	settingrepository "secure-patrol-backend/modules/setting/repository"
	settingservice "secure-patrol-backend/modules/setting/service"
	"secure-patrol-backend/seeder"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect() *gorm.DB {
	var (
		// db configs
		dbHost     = os.Getenv("DB_HOST")
		dbPort     = os.Getenv("DB_PORT")
		dbUsername = os.Getenv("DB_USERNAME")
		dbPassword = os.Getenv("DB_PASSWORD")
		dbName     = os.Getenv("DB_NAME")
		dbSSLMode  = os.Getenv("DB_SSLMODE")
		dbTimezone = os.Getenv("DB_TIMEZONE")
	)

	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	query := url.Values{}
	query.Set("sslmode", dbSSLMode)
	if dbTimezone != "" {
		query.Set("TimeZone", dbTimezone)
	}

	// Build the DSN as a URL so credentials with special characters are escaped.
	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUsername, dbPassword),
		Host:     net.JoinHostPort(dbHost, dbPort),
		Path:     "/" + dbName,
		RawQuery: query.Encode(),
	}).String()
	customLogLevel, _ := strconv.Atoi(os.Getenv("DB_LOG_LEVEL"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(customLogLevel)),
		// Turn unique constraint violations into gorm.ErrDuplicatedKey so services
		// can answer 409 when two requests race past the uniqueness check.
		TranslateError: true,
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	if err := migration.Migrate(db); err != nil {
		log.Fatal(err.Error())
	}

	// Adjustable settings live in the database; missing ones get their default.
	settings := settingservice.NewSettingService(settingrepository.NewSettingRepository(db))
	created, err := settings.EnsureDefaults()
	if err != nil {
		log.Fatal(err.Error())
	}
	if created > 0 {
		log.Printf("[settings] created %d setting(s) with their default value", created)
	}
	settingservice.Init(settings)

	// The license is verified at startup and then every minute by a worker.
	licenses := licenseservice.NewLicenseService(db)
	licenseservice.Init(licenses)
	if status := licenses.Status(); status.Locked() {
		log.Printf("[license] %s: %s (install id %s)", status.State, status.Reason, status.InstallID)
	}

	if os.Getenv("SEED_DUMMY_DATA") == "true" {
		if err := seeder.Seed(db); err != nil {
			log.Fatal(err.Error())
		}
	}

	return db
}
