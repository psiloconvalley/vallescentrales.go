// internal/app/config.go
// Environment variable loading and validation.
// App will not start if required variables are missing.
// All secrets come from environment — never from code.

package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// App
	AppEnv    string
	AppPort   string
	AppSecret string

	// Database
	DatabaseURL string

	// Storage (Cloudflare R2 — configured later)
	R2AccountID string
	R2AccessKey string
	R2SecretKey string
	R2Bucket    string
	R2PublicURL string
}

// LoadConfig loads and validates all environment variables.
// In development: reads from .env file.
// In production: reads from Railway injected environment.
// Returns an error if any required variable is missing.
// The app must not start if this returns an error.
func LoadConfig() (*Config, error) {
	// .env is ignored in production — Railway injects vars directly
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
	}

	// Validate required variables — fail fast with clear messages
	var errs []error

	if v, ok := requireEnv("APP_SECRET"); ok {
		cfg.AppSecret = v
	} else {
		errs = append(errs, fmt.Errorf("APP_SECRET is required"))
	}

	if v, ok := requireEnv("DATABASE_URL"); ok {
		cfg.DatabaseURL = v
	} else {
		errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// Optional — storage configured later
	cfg.R2AccountID = getEnv("R2_ACCOUNT_ID", "")
	cfg.R2AccessKey = getEnv("R2_ACCESS_KEY", "")
	cfg.R2SecretKey = getEnv("R2_SECRET_KEY", "")
	cfg.R2Bucket    = getEnv("R2_BUCKET", "")
	cfg.R2PublicURL = getEnv("R2_PUBLIC_URL", "")

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// getEnv returns the env value or a fallback default.
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// requireEnv returns the value and true if set, empty string and false if missing.
func requireEnv(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}
