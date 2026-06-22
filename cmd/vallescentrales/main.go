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

	userRepo    := repo.NewUserRepo(db)
	sessionRepo := repo.NewSessionRepo(db)
	listingRepo := repo.NewListingRepo(db)

	sessionMgr := auth.NewSessionManager(sessionRepo, cfg.IsProduction())

	googleAuth := auth.NewGoogleOAuth(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.IsProduction(),
	)

	authMW := middleware.NewAuthMiddleware(sessionMgr, userRepo)

	tmpl, err := app.NewTemplateRenderer()
	if err != nil {
		slog.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	authH    := handlers.NewAuthHandler(userRepo, sessionMgr, googleAuth, tmpl)
	listingH := handlers.NewListingHandler(listingRepo, tmpl)
	profileH := handlers.NewProfileHandler(userRepo, tmpl)

	server, err := app.NewServer(cfg, db, authMW, authH, listingH, profileH, tmpl)
	if err != nil {
		slog.Error("failed to build server", "error", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
