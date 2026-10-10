// internal/app/db.go
// PostgreSQL connection pool setup and automatic self-healing migration runner.

package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBPool creates a validated, health-checked connection pool and runs migrations.
func NewDBPool(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: failed to parse config: %w", err)
	}

	// Connection pool tuning
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 1 * time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("db: failed to create pool: %w", err)
	}

	// Verify connection is alive
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("db: failed to ping database: %w", err)
	}

	// Run automatic migrations
	if err := runMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("db: migration failed: %w", err)
	}

	return pool, nil
}

// runMigrations discovers and executes all pending SQL migrations in migrations/
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// 1. Ensure schema tracking table exists
	createSchemaTable := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := pool.Exec(ctx, createSchemaTable); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Discover migration files
	files, err := os.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasSuffix(name, ".sql") && !strings.HasSuffix(name, ".down.sql") {
			migrations = append(migrations, name)
		}
	}

	// Sort migrations historically (lexicographically)
	sort.Strings(migrations)

	// 3. Self-Healing Bootstrapping: If "users" table already exists, mark historical migrations as applied
	var usersExist bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users')").Scan(&usersExist)
	if err != nil {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	if usersExist {
		slog.Info("bootstrapping migrations tracking: database has pre-existing schema. marking legacy migrations as applied")
		for _, filename := range migrations {
			// Mark all old sequential migrations starting with "0000" as already applied
			if strings.HasPrefix(filename, "0000") {
				_, err := pool.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", filename)
				if err != nil {
					return fmt.Errorf("failed to bootstrap tracking for migration %s: %w", filename, err)
				}
			}
		}
	}

	// 4. Run outstanding migrations
	for _, filename := range migrations {
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)", filename).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", filename, err)
		}

		if exists {
			continue
		}

		slog.Info("running outstanding database migration", "filename", filename)

		// Read migration content
		path := filepath.Join("migrations", filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration query inside transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// Record migration tracking row
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", filename); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		slog.Info("migration completed successfully", "filename", filename)
	}

	return nil
}
