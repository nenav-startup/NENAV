// Package postgres manages the PostgreSQL connection pool.
//
// Reference: Master Prompt §17 — Database Principles, §93 — Connection Pooling
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nenav/nenav/backend/internal/config"
)

// DB wraps the pgx connection pool and provides a health check.
type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// New creates a new connection pool with the given configuration.
// It pings the database to verify connectivity before returning.
func New(ctx context.Context, cfg config.Database, logger *slog.Logger) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	logger.Info("postgres_connected",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Name,
		"max_conns", cfg.MaxOpenConns,
		"min_conns", cfg.MaxIdleConns,
	)

	return &DB{Pool: pool, logger: logger}, nil
}

// Ping checks database connectivity. Used by health checks.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
	db.logger.Info("postgres_closed")
}

// Stats returns current pool statistics for observability.
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
