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

	// Google OAuth (optional — app works without it)
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Storage (Cloudflare R2 — configured later)
	R2AccountID string
	R2AccessKey string
	R2SecretKey string
	R2Bucket    string
	R2PublicURL string
}

// LoadConfig loads and validates all environment variables.
func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("PORT", getEnv("APP_PORT", "8080")),
	}

	// Required
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

	// Optional — Google OAuth
	cfg.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
	cfg.GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
	cfg.GoogleRedirectURL = getEnv("GOOGLE_REDIRECT_URL", "")

	// Optional — Storage
	cfg.R2AccountID = getEnv("R2_ACCOUNT_ID", "")
	cfg.R2AccessKey = getEnv("R2_ACCESS_KEY", "")
	cfg.R2SecretKey = getEnv("R2_SECRET_KEY", "")
	cfg.R2Bucket = getEnv("R2_BUCKET", "")
	cfg.R2PublicURL = getEnv("R2_PUBLIC_URL", "")

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// GoogleOAuthEnabled returns true if Google OAuth credentials are configured.
func (c *Config) GoogleOAuthEnabled() bool {
	return c.GoogleClientID != ""
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func requireEnv(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}
