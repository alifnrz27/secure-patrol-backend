package helper

import (
	stdlog "log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EnvInt reads an integer environment variable, falling back to def when empty or invalid.
func EnvInt(name string, def int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return def
	}
	return value
}

// IsDevelopment reports whether APP_ENV is a development environment.
// Anything else, including an empty APP_ENV, is treated as non-development.
func IsDevelopment() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "development", "dev", "local":
		return true
	}
	return false
}

// EnvFloat reads a float environment variable, falling back to def when empty or invalid.
func EnvFloat(name string, def float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(name)), 64)
	if err != nil {
		return def
	}
	return value
}

var (
	appLocation     *time.Location
	appLocationOnce sync.Once
)

// AppLocation is the business time zone used for shifts and dates
// (APP_TIMEZONE, then DB_TIMEZONE, default Asia/Jakarta).
func AppLocation() *time.Location {
	appLocationOnce.Do(func() {
		name := os.Getenv("APP_TIMEZONE")
		if name == "" {
			name = os.Getenv("DB_TIMEZONE")
		}
		if name == "" {
			name = "Asia/Jakarta"
		}

		loc, err := time.LoadLocation(name)
		if err != nil {
			stdlog.Printf("invalid time zone %q, using UTC+7: %v", name, err)
			loc = time.FixedZone("UTC+7", 7*60*60)
		}
		appLocation = loc
	})

	return appLocation
}
