// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MadsRC/trustedai"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "test-org",
			DisplayName: "Test Organization",
			IsSystem:    false,
			CreatedAt:   now,
			SSOType:     "",
			SSOConfig:   nil,
		}

		mock.ExpectExec(`
			INSERT INTO organizations \(
				id, name, display_name, is_system, 
				created_at, sso_type, sso_config
			\)
			VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.CreatedAt,
			org.SSOType,
			org.SSOConfig,
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Create(context.Background(), org)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate entry", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "test-org",
			DisplayName: "Test Organization",
			IsSystem:    false,
			CreatedAt:   time.Now(),
		}

		mock.ExpectExec(`INSERT INTO organizations`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.CreatedAt,
			org.SSOType,
			org.SSOConfig,
		).WillReturnError(&pgconn.PgError{Code: "23505"})

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Create(context.Background(), org)
		assert.ErrorIs(t, err, trustedai.ErrDuplicateEntry)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "test-org",
			DisplayName: "Test Organization",
			IsSystem:    false,
			CreatedAt:   time.Now(),
		}

		mock.ExpectExec(`INSERT INTO organizations`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.CreatedAt,
			org.SSOType,
			org.SSOConfig,
		).WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Create(context.Background(), org)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		expectedOrg := &trustedai.Organization{
			ID:          "org-123",
			Name:        "test-org",
			DisplayName: "Test Org",
			IsSystem:    false,
			CreatedAt:   now,
			SSOType:     "oidc",
			SSOConfig:   map[string]any{"client_id": "test"},
		}

		rows := pgxmock.NewRows([]string{
			"id", "name", "display_name", "is_system",
			"created_at", "sso_type", "sso_config",
		}).AddRow(
			expectedOrg.ID,
			expectedOrg.Name,
			expectedOrg.DisplayName,
			expectedOrg.IsSystem,
			expectedOrg.CreatedAt,
			expectedOrg.SSOType,
			expectedOrg.SSOConfig,
		)

		mock.ExpectQuery(`
			SELECT id, name, display_name, is_system, 
				created_at, sso_type, sso_config
			FROM organizations
			WHERE id = \$1`).
			WithArgs(expectedOrg.ID).
			WillReturnRows(rows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		org, err := repo.Get(context.Background(), expectedOrg.ID)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrg.ID, org.ID)
		assert.Equal(t, expectedOrg.Name, org.Name)
		assert.Equal(t, expectedOrg.DisplayName, org.DisplayName)
		assert.Equal(t, expectedOrg.IsSystem, org.IsSystem)
		assert.Equal(t, expectedOrg.CreatedAt, org.CreatedAt)
		assert.Equal(t, expectedOrg.SSOType, org.SSOType)
		assert.Equal(t, expectedOrg.SSOConfig, org.SSOConfig)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WithArgs("missing-id").
			WillReturnError(pgx.ErrNoRows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.Get(context.Background(), "missing-id")
		assert.ErrorIs(t, err, trustedai.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WithArgs("org-123").
			WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.Get(context.Background(), "org-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_GetByName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		expectedOrg := &trustedai.Organization{
			ID:          "org-123",
			Name:        "test-org",
			DisplayName: "Test Org",
			IsSystem:    false,
			CreatedAt:   now,
			SSOType:     "oidc",
			SSOConfig:   map[string]any{"client_id": "test"},
		}

		rows := pgxmock.NewRows([]string{
			"id", "name", "display_name", "is_system",
			"created_at", "sso_type", "sso_config",
		}).AddRow(
			expectedOrg.ID,
			expectedOrg.Name,
			expectedOrg.DisplayName,
			expectedOrg.IsSystem,
			expectedOrg.CreatedAt,
			expectedOrg.SSOType,
			expectedOrg.SSOConfig,
		)

		mock.ExpectQuery(`
			SELECT id, name, display_name, is_system, 
				created_at, sso_type, sso_config
			FROM organizations
			WHERE name = \$1`).
			WithArgs(expectedOrg.Name).
			WillReturnRows(rows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		org, err := repo.GetByName(context.Background(), expectedOrg.Name)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrg.ID, org.ID)
		assert.Equal(t, expectedOrg.Name, org.Name)
		assert.Equal(t, expectedOrg.DisplayName, org.DisplayName)
		assert.Equal(t, expectedOrg.IsSystem, org.IsSystem)
		assert.Equal(t, expectedOrg.CreatedAt, org.CreatedAt)
		assert.Equal(t, expectedOrg.SSOType, org.SSOType)
		assert.Equal(t, expectedOrg.SSOConfig, org.SSOConfig)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WithArgs("missing-name").
			WillReturnError(pgx.ErrNoRows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.GetByName(context.Background(), "missing-name")
		assert.ErrorIs(t, err, trustedai.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WithArgs("test-org").
			WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.GetByName(context.Background(), "test-org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_List(t *testing.T) {
	t.Run("success with multiple organizations", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		org1 := &trustedai.Organization{
			ID:          "org-1",
			Name:        "org-one",
			DisplayName: "Organization One",
			IsSystem:    false,
			CreatedAt:   now,
			SSOType:     "",
		}

		org2 := &trustedai.Organization{
			ID:          "org-2",
			Name:        "org-two",
			DisplayName: "Organization Two",
			IsSystem:    true,
			CreatedAt:   now.Add(time.Hour),
			SSOType:     "oidc",
			SSOConfig:   map[string]any{"client_id": "test-client"},
		}

		rows := pgxmock.NewRows([]string{
			"id", "name", "display_name", "is_system",
			"created_at", "sso_type", "sso_config",
		}).
			AddRow(
				org1.ID, org1.Name, org1.DisplayName, org1.IsSystem,
				org1.CreatedAt, org1.SSOType, org1.SSOConfig,
			).
			AddRow(
				org2.ID, org2.Name, org2.DisplayName, org2.IsSystem,
				org2.CreatedAt, org2.SSOType, org2.SSOConfig,
			)

		mock.ExpectQuery(`
			SELECT id, name, display_name, is_system, 
				created_at, sso_type, sso_config
			FROM organizations
			ORDER BY name`).
			WillReturnRows(rows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		orgs, err := repo.List(context.Background())
		assert.NoError(t, err)
		assert.Len(t, orgs, 2)

		// Check first org
		assert.Equal(t, org1.ID, orgs[0].ID)
		assert.Equal(t, org1.Name, orgs[0].Name)
		assert.Equal(t, org1.DisplayName, orgs[0].DisplayName)
		assert.Equal(t, org1.IsSystem, orgs[0].IsSystem)
		assert.Equal(t, org1.CreatedAt, orgs[0].CreatedAt)
		assert.Equal(t, org1.SSOType, orgs[0].SSOType)
		assert.Equal(t, org1.SSOConfig, orgs[0].SSOConfig)

		// Check second org
		assert.Equal(t, org2.ID, orgs[1].ID)
		assert.Equal(t, org2.Name, orgs[1].Name)
		assert.Equal(t, org2.DisplayName, orgs[1].DisplayName)
		assert.Equal(t, org2.IsSystem, orgs[1].IsSystem)
		assert.Equal(t, org2.CreatedAt, orgs[1].CreatedAt)
		assert.Equal(t, org2.SSOType, orgs[1].SSOType)
		assert.Equal(t, org2.SSOConfig, orgs[1].SSOConfig)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with empty list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{
			"id", "name", "display_name", "is_system",
			"created_at", "sso_type", "sso_config",
		})

		mock.ExpectQuery(`
			SELECT id, name, display_name, is_system, 
				created_at, sso_type, sso_config
			FROM organizations
			ORDER BY name`).
			WillReturnRows(rows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		orgs, err := repo.List(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, orgs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.List(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("row scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Return rows with a boolean value for ID column to force scan error
		rows := pgxmock.NewRows([]string{
			"id", "name", "display_name", "is_system",
			"created_at", "sso_type", "sso_config",
		}).AddRow(
			true, // Changed from 123 to boolean to create real scan error
			"test-org",
			"Test Org",
			false,
			time.Now(),
			"",
			nil,
		)

		mock.ExpectQuery(`SELECT .* FROM organizations`).
			WillReturnRows(rows)

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		_, err = repo.List(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Destination kind 'string' not supported for value kind 'bool'")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "updated-org",
			DisplayName: "Updated Organization",
			IsSystem:    false,
			SSOType:     "oidc",
			SSOConfig:   map[string]any{"client_id": "updated-client"},
		}

		mock.ExpectExec(`
			UPDATE organizations SET
				name = \$2,
				display_name = \$3,
				is_system = \$4,
				sso_type = \$5,
				sso_config = \$6
			WHERE id = \$1`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.SSOType,
			org.SSOConfig,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Update(context.Background(), org)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "non-existent-id",
			Name:        "updated-org",
			DisplayName: "Updated Organization",
		}

		mock.ExpectExec(`UPDATE organizations SET`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.SSOType,
			org.SSOConfig,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Update(context.Background(), org)
		assert.ErrorIs(t, err, trustedai.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate entry", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "duplicate-name",
			DisplayName: "Duplicate Organization",
		}

		mock.ExpectExec(`UPDATE organizations SET`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.SSOType,
			org.SSOConfig,
		).WillReturnError(&pgconn.PgError{Code: "23505"})

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Update(context.Background(), org)
		assert.ErrorIs(t, err, trustedai.ErrDuplicateEntry)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		org := &trustedai.Organization{
			ID:          "org-123",
			Name:        "updated-org",
			DisplayName: "Updated Organization",
		}

		mock.ExpectExec(`UPDATE organizations SET`).WithArgs(
			org.ID,
			org.Name,
			org.DisplayName,
			org.IsSystem,
			org.SSOType,
			org.SSOConfig,
		).WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Update(context.Background(), org)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		orgID := "org-123"

		mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
			WithArgs(orgID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Delete(context.Background(), orgID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		orgID := "non-existent-id"

		mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
			WithArgs(orgID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Delete(context.Background(), orgID)
		assert.ErrorIs(t, err, trustedai.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		orgID := "org-123"

		mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
			WithArgs(orgID).
			WillReturnError(errors.New("database error"))

		repo := &OrganizationRepository{
			options: &organizationRepositoryOptions{
				Db:     mock,
				Logger: slog.Default(),
			},
		}

		err = repo.Delete(context.Background(), orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrganizationRepository_GlobalOptions(t *testing.T) {
	customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("Global options are applied", func(t *testing.T) {
		// Set global options
		GlobalOrganizationRepositoryOptions = []OrganizationRepositoryOption{
			WithOrganizationRepositoryLogger(customLogger),
		}

		// Create two repositories (with error handling)
		repo1, err := NewOrganizationRepository()
		require.NoError(t, err)
		repo2, err := NewOrganizationRepository()
		require.NoError(t, err)

		// Both should have the custom logger
		assert.Equal(t, customLogger, repo1.options.Logger)
		assert.Equal(t, customLogger, repo2.options.Logger)
		assert.Equal(t, repo1.options.Logger, repo2.options.Logger)

		// Reset global options
		GlobalOrganizationRepositoryOptions = []OrganizationRepositoryOption{}

		// New repository should have default logger (with error handling)
		repo3, err := NewOrganizationRepository()
		require.NoError(t, err)
		assert.NotEqual(t, customLogger, repo3.options.Logger)
		assert.Equal(t, slog.Default(), repo3.options.Logger)
	})
}
