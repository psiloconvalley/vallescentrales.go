// internal/app/db.go
// PostgreSQL connection pool setup.
// App will not start if database is unreachable.

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBPool creates a validated, health-checked connection pool.
// The app will not start if the database is unreachable.
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

	// Verify the connection is actually alive before proceeding
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("db: failed to ping database: %w", err)
	}

	return pool, nil
}
