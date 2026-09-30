package config

import (
	"log"
	"os"
)

// ValidateSecurityEnv stops the app when the secrets used for tokens and app keys are missing or weak.
func ValidateSecurityEnv() {
	for _, name := range []string{"JWT_SECRET", "APP_MASTER_SECRET"} {
		if len(os.Getenv(name)) < 32 {
			log.Fatalf("%s must be set and at least 32 characters long", name)
		}
	}
}
