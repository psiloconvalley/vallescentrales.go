// internal/app/config.go
// Environment variable loading and validation.
// App will not start if required variables are missing.
// All secrets come from environment — never from code.

package app

import (
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
func LoadConfig() (*Config, error) {
	// .env is ignored in production — Railway injects vars directly
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:    getEnv("APP_ENV", "development"),
		AppPort:   getEnv("APP_PORT", "8080"),
		AppSecret: mustGetEnv("APP_SECRET"),

		DatabaseURL: mustGetEnv("DATABASE_URL"),

		R2AccountID: getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey: getEnv("R2_ACCESS_KEY", ""),
		R2SecretKey: getEnv("R2_SECRET_KEY", ""),
		R2Bucket:    getEnv("R2_BUCKET", ""),
		R2PublicURL: getEnv("R2_PUBLIC_URL", ""),
	}

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

// mustGetEnv panics at startup if a required variable is missing.
// A misconfigured app must never start silently.
func mustGetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		panic(fmt.Sprintf("FATAL: required environment variable %q is not set", key))
	}
	return val
}
