// Package migrate handles database schema migrations using golang-migrate.
//
// Reference: Master Prompt §18 — Database Migrations
package migrate

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up runs all pending migrations.
func Up(pool *pgxpool.Pool, migrationsPath string, logger *slog.Logger) error {
	m, err := newMigrator(pool, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ := m.Version()
	logger.Info("migrations_applied", "version", version, "dirty", dirty)
	return nil
}

// Down rolls back the last migration.
func Down(pool *pgxpool.Pool, migrationsPath string, logger *slog.Logger) error {
	m, err := newMigrator(pool, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}

	version, dirty, _ := m.Version()
	logger.Info("migration_rolled_back", "version", version, "dirty", dirty)
	return nil
}

// Validate checks that all migration files are properly paired (up + down).
func Validate(migrationsPath string, logger *slog.Logger) error {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	ups := make(map[string]bool)
	downs := make(map[string]bool)

	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			ups[strings.TrimSuffix(name, ".up.sql")] = true
		} else if strings.HasSuffix(name, ".down.sql") {
			downs[strings.TrimSuffix(name, ".down.sql")] = true
		}
	}

	var missing []string
	for base := range ups {
		if !downs[base] {
			missing = append(missing, base+".down.sql")
		}
	}
	for base := range downs {
		if !ups[base] {
			missing = append(missing, base+".up.sql")
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("unpaired migration files: %v", missing)
	}

	logger.Info("migrations_validated", "count", len(ups))
	return nil
}

func newMigrator(pool *pgxpool.Pool, migrationsPath string) (*migrate.Migrate, error) {
	db := stdlib.OpenDBFromPool(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("migrate: create driver: %w", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("migrate: resolve path: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+absPath, "nenav", driver)
	if err != nil {
		return nil, fmt.Errorf("migrate: create instance: %w", err)
	}

	return m, nil
}
