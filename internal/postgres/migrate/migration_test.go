// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build integration

package migrate

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgMigrate "github.com/MadsRC/trustedai/internal/postgres"
)

func TestMigrations(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Clean up container after test
	t.Cleanup(func() {
		assert.NoError(t, pgContainer.Terminate(ctx))
	})

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	t.Logf("Connection string: %s", connStr)

	// Create logger for migrations
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("apply_migrations_up", func(t *testing.T) {
		// Apply all migrations up
		err := pgMigrate.RunMigrations(logger, connStr)
		require.NoError(t, err)

		// Verify tables exist by connecting to database
		conn, err := pgx.Connect(ctx, connStr)
		require.NoError(t, err)
		defer conn.Close(ctx)

		// Check that migration tables were created
		var tableCount int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name IN ('organizations', 'users', 'tokens', 'models', 'usage_events', 'billing_summaries')
		`).Scan(&tableCount)
		require.NoError(t, err)
		assert.Equal(t, 6, tableCount, "Expected 6 tables to be created")

		// Verify migration version is at latest
		var version int
		err = conn.QueryRow(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
		require.NoError(t, err)
		assert.Equal(t, 9, version, "Expected migration version to be 9")
	})

	t.Run("rollback_migrations_down", func(t *testing.T) {
		// First apply migrations to ensure we have something to rollback
		err := pgMigrate.RunMigrations(logger, connStr)
		require.NoError(t, err)

		// Create migrate instance for manual down migration
		sourceDriver, err := iofs.New(pgMigrate.GetMigrationFiles(), "migrate")
		require.NoError(t, err)

		// Convert postgres:// to pgx5:// URL scheme for pgx/v5 driver
		migrationURL := strings.Replace(connStr, "postgres://", "pgx5://", 1)

		m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, migrationURL)
		require.NoError(t, err)
		defer m.Close()

		// Roll back all migrations
		err = m.Down()
		require.NoError(t, err)

		// Verify tables are gone (except schema_migrations)
		conn, err := pgx.Connect(ctx, connStr)
		require.NoError(t, err)
		defer conn.Close(ctx)

		var tableCount int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name IN ('organizations', 'users', 'tokens', 'models', 'usage_events', 'billing_summaries')
		`).Scan(&tableCount)
		require.NoError(t, err)
		assert.Equal(t, 0, tableCount, "Expected all application tables to be dropped")

		// Verify schema_migrations table still exists but is empty
		var migrationCount int
		err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
		require.NoError(t, err)
		assert.Equal(t, 0, migrationCount, "Expected no migration records after rollback")
	})

	t.Run("reapply_migrations_after_rollback", func(t *testing.T) {
		// Apply migrations again to test idempotency
		err := pgMigrate.RunMigrations(logger, connStr)
		require.NoError(t, err)

		// Verify tables exist again
		conn, err := pgx.Connect(ctx, connStr)
		require.NoError(t, err)
		defer conn.Close(ctx)

		var tableCount int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name IN ('organizations', 'users', 'tokens', 'models', 'usage_events', 'billing_summaries')
		`).Scan(&tableCount)
		require.NoError(t, err)
		assert.Equal(t, 6, tableCount, "Expected 6 tables to be recreated")

		// Running migrations again should be idempotent
		err = pgMigrate.RunMigrations(logger, connStr)
		require.NoError(t, err)
	})
}
