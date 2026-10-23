// Package cache manages the Redis connection.
//
// Reference: Master Prompt §36 — Redis, §37 — Cache Strategy
package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/nenav/nenav/backend/internal/config"
)

// Redis wraps the Redis client and provides a health check.
type Redis struct {
	Client *redis.Client
	logger *slog.Logger
}

// New creates a new Redis connection and verifies it with a ping.
func New(ctx context.Context, cfg config.Redis, logger *slog.Logger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	logger.Info("redis_connected",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DB,
	)

	return &Redis{Client: client, logger: logger}, nil
}

// Ping checks Redis connectivity. Used by health checks.
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close shuts down the Redis connection.
func (r *Redis) Close() error {
	r.logger.Info("redis_closed")
	return r.Client.Close()
}
