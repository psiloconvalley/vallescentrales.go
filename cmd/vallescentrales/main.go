// cmd/vallescentrales/main.go
// Entry point only. No business logic here.
// Wires config → database → repos → auth → middleware → handlers → templates → server.

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

	// 1. Load and validate config
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Connect to database
	db, err := app.NewDBPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 3. Build repo layer
	userRepo    := repo.NewUserRepo(db)
	sessionRepo := repo.NewSessionRepo(db)
	listingRepo := repo.NewListingRepo(db)

	// 4. Build auth layer
	sessionMgr := auth.NewSessionManager(sessionRepo, cfg.IsProduction())

	// 5. Build middleware
	authMW := middleware.NewAuthMiddleware(sessionMgr, userRepo)

	// 6. Parse templates
	tmpl, err := app.NewTemplateRenderer()
	if err != nil {
		slog.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	// 7. Build handlers
	authH    := handlers.NewAuthHandler(userRepo, sessionMgr)
	listingH := handlers.NewListingHandler(listingRepo)

	// 8. Build server and block until shutdown signal
	server, err := app.NewServer(cfg, db, authMW, authH, listingH, tmpl)
	if err != nil {
		slog.Error("failed to build server", "error", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
