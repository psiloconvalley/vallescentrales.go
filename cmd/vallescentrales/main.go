// cmd/vallescentrales/main.go
// Entry point only. No business logic here.

package main

import (
	"context"
	"log/slog"
	"os"

	"vallescentrales/internal/app"
	"vallescentrales/internal/auth"
	"vallescentrales/internal/handlers"
	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
	"vallescentrales/internal/services"
	"vallescentrales/internal/wizard"
)

func main() {
	ctx := context.Background()

	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := app.NewDBPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Repositories
	userRepo := repo.NewUserRepo(db)
	sessionRepo := repo.NewSessionRepo(db)
	listingRepo := repo.NewListingRepo(db)
	passkeyRepo := repo.NewPasskeyRepo(db)

	// Wizard Engine (shared across all multi-step flows)
	wizardEngine := wizard.NewEngine(db)

	// Auth
	sessionMgr := auth.NewSessionManager(sessionRepo, cfg.IsProduction(), cfg.BaseDomain)

	googleAuth := auth.NewGoogleOAuth(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.IsProduction(),
		cfg.BaseDomain,
	)

	// WebAuthn (passkeys)
	webAuthn, err := auth.NewWebAuthn(
		cfg.WebAuthnRPID,
		cfg.WebAuthnRPDisplayName,
		cfg.WebAuthnOrigins,
	)
	if err != nil {
		slog.Error("failed to initialize WebAuthn", "error", err)
		os.Exit(1)
	}

	// Infrastructure services
	storageSvc := services.NewStorageService(
		cfg.R2AccountID,
		cfg.R2AccessKey,
		cfg.R2SecretKey,
		cfg.R2Bucket,
		cfg.R2PublicURL,
	)

	currencySvc := services.NewCurrencyService(db)

	// Middleware
	authMW := middleware.NewAuthMiddleware(sessionMgr, userRepo)

	// Templates
	tmpl, err := app.NewTemplateRenderer(currencySvc)
	if err != nil {
		slog.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	// Handlers (with exact production-matching arguments)
	authH := handlers.NewAuthHandler(userRepo, sessionMgr, googleAuth, tmpl, cfg.IsProduction())
	listingH := handlers.NewListingHandler(listingRepo, userRepo, tmpl)
	profileH := handlers.NewProfileHandler(userRepo, passkeyRepo, listingRepo, tmpl)
	passkeyH := handlers.NewPasskeyHandler(webAuthn, passkeyRepo, userRepo, sessionMgr)
	uploadH := handlers.NewUploadHandler(storageSvc, listingRepo, userRepo)

	// Wizard Handlers
	accountWizardH := handlers.NewWizardAccountHandler(wizardEngine, userRepo, tmpl)
	propertyWizardH := handlers.NewWizardPropertyHandler(wizardEngine, listingRepo, tmpl, storageSvc)

	// Server
	server, err := app.NewServer(cfg, db, authMW, authH, listingH, profileH, passkeyH, uploadH, accountWizardH, propertyWizardH, tmpl)
	if err != nil {
		slog.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
