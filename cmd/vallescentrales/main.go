// cmd/vallescentrales/main.go
// Entry point only. No business logic here.
// Wires config → database → server and starts.

package main

import (
	"context"
	"log"
	"os"

	"vallescentrales/internal/app"
)

func main() {
	ctx := context.Background()

	// 1. Load and validate config — panics if required vars missing
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Connect to database — fails fast if unreachable
	db, err := app.NewDBPool(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// 3. Build server and block until shutdown signal
	server := app.NewServer(cfg, db)
	if err := server.Start(); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
