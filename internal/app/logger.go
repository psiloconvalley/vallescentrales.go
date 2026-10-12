// internal/app/logger.go
// Configures the global slog default logger with environment-aware output.
// Must be called before any other initialization so startup logs are formatted correctly.

package app

import (
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures slog.Default() based on the application environment.
//
// Production (APP_ENV=production):
//   - JSON output to stdout (Railway captures stdout for log aggregation)
//   - Default level: Info (overridable via LOG_LEVEL)
//   - Service metadata attached to every log line
//
// Development (anything else):
//   - Human-readable text output to stderr
//   - Default level: Debug (overridable via LOG_LEVEL)
//   - Source file location included for easier debugging
func SetupLogger(appEnv string) {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"), appEnv)

	var handler slog.Handler
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	// Attach service-level metadata to every log line.
	// Any downstream slog.Info/Error/Warn/Debug call inherits these fields.
	logger := slog.New(handler).With(
		slog.String("service", "vallescentrales"),
		slog.String("env", appEnv),
	)

	slog.SetDefault(logger)
}

// parseLogLevel converts LOG_LEVEL env var to slog.Level.
// Falls back to Info in production, Debug in development.
func parseLogLevel(raw, appEnv string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}

	if appEnv == "production" {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}
