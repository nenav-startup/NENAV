// Package app wires all dependencies and manages the application lifecycle.
//
// This is the composition root: it creates infrastructure, connects services,
// starts the HTTP server, and handles graceful shutdown.
//
// Reference: Master Prompt §40 — Graceful Shutdown
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nenav/nenav/backend/internal/config"
	"github.com/nenav/nenav/backend/internal/observability"
	"github.com/nenav/nenav/backend/internal/platform/cache"
	"github.com/nenav/nenav/backend/internal/platform/migrate"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/repository/postgres"
	router "github.com/nenav/nenav/backend/internal/transport/http"
	"github.com/nenav/nenav/backend/internal/transport/http/handler"
)

// Run is the main application entry point. It loads configuration,
// connects to all dependencies, starts the HTTP server, and blocks
// until a shutdown signal is received.
func Run(migrateUp, migrateDown, validateMigrations bool) error {
	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Logger ---
	logger := observability.NewLogger(cfg.App.LogLevel, os.Stdout)
	slog.SetDefault(logger)

	logger.Info("nenav_starting",
		"env", cfg.App.Env,
		"port", cfg.App.Port,
	)

	// --- Migration-only mode (validate) ---
	migrationsPath := findMigrationsPath()

	if validateMigrations {
		return migrate.Validate(migrationsPath, logger)
	}

	// --- PostgreSQL ---
	ctx := context.Background()
	var db handler.Pinger
	var store repository.Store

	pgDB, err := postgres.New(ctx, cfg.DB, logger)
	if err != nil {
		if cfg.App.Env == "local" || cfg.App.Env == "development" {
			logger.Warn("postgres_unavailable_fallback_to_memorystore", "error", err.Error())
			store = repository.NewMemoryStore()
		} else {
			return fmt.Errorf("postgres: %w", err)
		}
	} else {
		defer pgDB.Close()
		db = pgDB
	}

	// --- Migration commands ---
	if migrateUp && pgDB != nil {
		return migrate.Up(pgDB.Pool, migrationsPath, logger)
	}
	if migrateDown && pgDB != nil {
		return migrate.Down(pgDB.Pool, migrationsPath, logger)
	}

	// --- Redis ---
	var redisCache handler.Pinger
	rCache, err := cache.New(ctx, cfg.Redis, logger)
	if err != nil {
		if cfg.App.Env == "local" || cfg.App.Env == "development" {
			logger.Warn("redis_unavailable_fallback_to_mock", "error", err.Error())
		} else {
			return fmt.Errorf("redis: %w", err)
		}
	} else {
		defer rCache.Close()
		redisCache = rCache
	}

	// --- HTTP Router ---
	routerHandler := router.New(router.Deps{
		Config: cfg,
		Logger: logger,
		DB:     db,
		Cache:  redisCache,
		Store:  store,
	})

	// --- HTTP Server ---
	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      routerHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Start server in goroutine ---
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http_listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// --- Wait for shutdown signal ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("shutdown_signal", "signal", sig.String())
	case err := <-errCh:
		return fmt.Errorf("server: %w", err)
	}

	// --- Graceful shutdown ---
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	logger.Info("shutting_down", "timeout", cfg.App.ShutdownTimeout.String())
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	logger.Info("nenav_stopped")
	return nil
}

// findMigrationsPath locates the migrations directory relative to the binary.
func findMigrationsPath() string {
	candidates := []string{
		"migrations",
		"backend/migrations",
		"../migrations",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return "migrations"
}
