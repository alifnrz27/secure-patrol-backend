package main

import (
	stdlog "log"
	"os"
	"secure-patrol-backend/config"
	"secure-patrol-backend/console"
	"time"
	// Embed the time zone database so shift times work on servers without tzdata.
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"gopkg.in/natefinch/lumberjack.v2"

	"secure-patrol-backend/pkg/log"
)

func main() {
	// initialize the logger
	go InitDebutLogger()

	// .env is optional: in Docker the values can also come from the environment.
	// Existing environment variables are never overridden by the file.
	if err := godotenv.Load(); err != nil {
		stdlog.Println(".env file not found, using environment variables")
	}

	config.ValidateSecurityEnv()

	db := config.Connect()

	// Console commands, e.g. `go run . create-app-client -name "Android" -platform android`
	if console.Run(db, os.Args[1:]) {
		return
	}

	config.Route(db)
}

// Set the logger, the logger using GoFiber log
// and the output will placed on ./logs/debug
func InitDebutLogger() {
	// Create a new Lumberjack logger
	logger := &lumberjack.Logger{
		Filename:   "./logs/debug/debug.log",
		MaxBackups: 90,
		Compress:   true,
		LocalTime:  true,
	}

	// Set logger as the output for log messages
	log.SetOutput(logger)

	midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+1, 0, 0, 0, 0, time.Local)
	durationUntilMidnight := midnight.Sub(time.Now())
	ticker := time.NewTicker(durationUntilMidnight)
	defer ticker.Stop()

	for range ticker.C {
		log.Debug("rotate")
		logger.Rotate()
	}
}
