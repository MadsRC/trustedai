// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		user := &llmgw.User{
			ID:             "user-123",
			Email:          "test@example.com",
			Name:           "Test User",
			OrganizationID: "org-123",
			ExternalID:     "ext-123",
			Provider:       "github",
			SystemAdmin:    true,
			CreatedAt:      now,
			LastLogin:      now,
		}

		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(
				user.ID,
				user.Email,
				user.Name,
				user.OrganizationID,
				user.ExternalID,
				user.Provider,
				user.SystemAdmin,
				user.CreatedAt,
				user.LastLogin,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Create(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate email", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
			).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Create(context.Background(), &llmgw.User{})
		assert.ErrorIs(t, err, llmgw.ErrDuplicateEntry)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
			).
			WillReturnError(errors.New("database connection failed"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Create(context.Background(), &llmgw.User{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		expectedUser := &llmgw.User{
			ID:             "user-123",
			Email:          "test@example.com",
			Name:           "Test User",
			OrganizationID: "org-123",
			ExternalID:     "ext-123",
			Provider:       "github",
			SystemAdmin:    true,
			CreatedAt:      now,
			LastLogin:      now,
		}

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		}).AddRow(
			expectedUser.ID,
			expectedUser.Email,
			expectedUser.Name,
			expectedUser.OrganizationID,
			expectedUser.ExternalID,
			expectedUser.Provider,
			expectedUser.SystemAdmin,
			expectedUser.CreatedAt,
			expectedUser.LastLogin,
		)

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs("user-123").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		user, err := repo.Get(context.Background(), "user-123")
		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs("missing").
			WillReturnError(pgx.ErrNoRows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), "missing")
		assert.ErrorIs(t, err, llmgw.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs("user-123").
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), "user-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		expectedUser := &llmgw.User{
			ID:             "user-123",
			Email:          "test@example.com",
			Name:           "Test User",
			OrganizationID: "org-123",
			ExternalID:     "ext-123",
			Provider:       "github",
			SystemAdmin:    true,
			CreatedAt:      now,
			LastLogin:      now,
		}

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		}).AddRow(
			expectedUser.ID,
			expectedUser.Email,
			expectedUser.Name,
			expectedUser.OrganizationID,
			expectedUser.ExternalID,
			expectedUser.Provider,
			expectedUser.SystemAdmin,
			expectedUser.CreatedAt,
			expectedUser.LastLogin,
		)

		mock.ExpectQuery(`SELECT .* FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		user, err := repo.GetByEmail(context.Background(), "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE email = \$1`).
			WithArgs("missing@example.com").
			WillReturnError(pgx.ErrNoRows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.GetByEmail(context.Background(), "missing@example.com")
		assert.ErrorIs(t, err, llmgw.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.GetByEmail(context.Background(), "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByExternalID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		expectedUser := &llmgw.User{
			ID:             "user-123",
			Email:          "test@example.com",
			Name:           "Test User",
			OrganizationID: "org-123",
			ExternalID:     "ext-123",
			Provider:       "github",
			SystemAdmin:    true,
			CreatedAt:      now,
			LastLogin:      now,
		}

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		}).AddRow(
			expectedUser.ID,
			expectedUser.Email,
			expectedUser.Name,
			expectedUser.OrganizationID,
			expectedUser.ExternalID,
			expectedUser.Provider,
			expectedUser.SystemAdmin,
			expectedUser.CreatedAt,
			expectedUser.LastLogin,
		)

		mock.ExpectQuery(`SELECT .* FROM users WHERE provider = \$1 AND external_id = \$2`).
			WithArgs("github", "ext-123").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		user, err := repo.GetByExternalID(context.Background(), "github", "ext-123")
		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE provider = \$1 AND external_id = \$2`).
			WithArgs("github", "missing").
			WillReturnError(pgx.ErrNoRows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.GetByExternalID(context.Background(), "github", "missing")
		assert.ErrorIs(t, err, llmgw.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE provider = \$1 AND external_id = \$2`).
			WithArgs("github", "ext-123").
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.GetByExternalID(context.Background(), "github", "ext-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_ListByOrganization(t *testing.T) {
	t.Run("success with multiple users", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		user1 := &llmgw.User{
			ID:             "user-1",
			Email:          "user1@example.com",
			Name:           "User One",
			OrganizationID: "org-123",
			ExternalID:     "ext-1",
			Provider:       "github",
			SystemAdmin:    true,
			CreatedAt:      now,
			LastLogin:      now,
		}

		user2 := &llmgw.User{
			ID:             "user-2",
			Email:          "user2@example.com",
			Name:           "User Two",
			OrganizationID: "org-123",
			ExternalID:     "ext-2",
			Provider:       "github",
			SystemAdmin:    false,
			CreatedAt:      now,
			LastLogin:      now,
		}

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		}).
			AddRow(
				user1.ID,
				user1.Email,
				user1.Name,
				user1.OrganizationID,
				user1.ExternalID,
				user1.Provider,
				user1.SystemAdmin,
				user1.CreatedAt,
				user1.LastLogin,
			).
			AddRow(
				user2.ID,
				user2.Email,
				user2.Name,
				user2.OrganizationID,
				user2.ExternalID,
				user2.Provider,
				user2.SystemAdmin,
				user2.CreatedAt,
				user2.LastLogin,
			)

		mock.ExpectQuery(`SELECT .* FROM users WHERE organization_id = \$1`).
			WithArgs("org-123").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		users, err := repo.ListByOrganization(context.Background(), "org-123")
		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, user1, users[0])
		assert.Equal(t, user2, users[1])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with empty result", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		})

		mock.ExpectQuery(`SELECT .* FROM users WHERE organization_id = \$1`).
			WithArgs("empty-org").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		users, err := repo.ListByOrganization(context.Background(), "empty-org")
		require.NoError(t, err)
		assert.Empty(t, users)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT .* FROM users WHERE organization_id = \$1`).
			WithArgs("org-123").
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.ListByOrganization(context.Background(), "org-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("row scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Create a row with mismatched types to cause a scan error
		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "organization_id",
			"external_id", "provider", "system_admin",
			"created_at", "last_login",
		}).AddRow(
			"user-1",
			"user1@example.com",
			"User One",
			"org-123",
			"ext-1",
			"github",
			"not-a-boolean", // This will cause scan error
			time.Now(),
			time.Now(),
		)

		mock.ExpectQuery(`SELECT .* FROM users WHERE organization_id = \$1`).
			WithArgs("org-123").
			WillReturnRows(rows)

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		_, err = repo.ListByOrganization(context.Background(), "org-123")
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		user := &llmgw.User{
			ID:             "user-123",
			Email:          "new@example.com",
			Name:           "Updated Name",
			OrganizationID: "org-456",
			ExternalID:     "ext-456",
			Provider:       "google",
			SystemAdmin:    false,
			LastLogin:      now,
		}

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs(
				user.ID,
				user.Email,
				user.Name,
				user.OrganizationID,
				user.ExternalID,
				user.Provider,
				user.SystemAdmin,
				user.LastLogin,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Update(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		user := &llmgw.User{
			ID: "non-existent",
		}

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs(
				user.ID,
				user.Email,
				user.Name,
				user.OrganizationID,
				user.ExternalID,
				user.Provider,
				user.SystemAdmin,
				user.LastLogin,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Update(context.Background(), user)
		assert.ErrorIs(t, err, llmgw.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		user := &llmgw.User{
			ID: "user-123",
		}

		mock.ExpectExec(`UPDATE users SET`).
			WithArgs(
				user.ID,
				user.Email,
				user.Name,
				user.OrganizationID,
				user.ExternalID,
				user.Provider,
				user.SystemAdmin,
				user.LastLogin,
			).
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Update(context.Background(), user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs("user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Delete(context.Background(), "user-123")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs("non-existent").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Delete(context.Background(), "non-existent")
		assert.ErrorIs(t, err, llmgw.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs("user-123").
			WillReturnError(errors.New("database error"))

		repo, err := NewUserRepository(WithUserRepositoryDb(mock))
		require.NoError(t, err)
		err = repo.Delete(context.Background(), "user-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
