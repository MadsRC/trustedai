// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrate/*.sql
var migrationFiles embed.FS

// GetMigrationFiles returns the embedded migration files for testing
func GetMigrationFiles() embed.FS {
	return migrationFiles
}

// RunMigrations runs all pending database migrations
func RunMigrations(logger *slog.Logger, databaseURL string) error {
	logger.Info("Running database migrations...")

	// Create migration source from embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrate")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Convert postgres:// to pgx5:// URL scheme for pgx/v5 driver
	migrationURL := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, migrationURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	// Get current version
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		logger.Warn("Database is in dirty state, attempting to force version", "version", currentVersion)
		if err := m.Force(int(currentVersion)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
	}

	if err == migrate.ErrNilVersion {
		logger.Info("No migrations have been applied yet")
	} else {
		logger.Info("Current migration version", "version", currentVersion)
	}

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		logger.Info("Database schema is up to date")
	} else {
		newVersion, _, _ := m.Version()
		logger.Info("Database migrations completed successfully", "new_version", newVersion)
	}

	return nil
}
