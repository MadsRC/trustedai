// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"

	"codeberg.org/MadsRC/llmgw"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Create adds a new user to the database
func (r *UserRepository) Create(ctx context.Context, user *llmgw.User) error {
	query := `
		INSERT INTO users (
			id, email, name, organization_id, 
			external_id, provider, system_admin, 
			created_at, last_login
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.options.Db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.OrganizationID,
		user.ExternalID,
		user.Provider,
		user.SystemAdmin,
		user.CreatedAt,
		user.LastLogin,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return llmgw.ErrDuplicateEntry
		}
		r.options.Logger.Error("Failed to create user", "error", err)
		return err
	}
	return nil
}

// Get retrieves a user by ID
func (r *UserRepository) Get(ctx context.Context, id string) (*llmgw.User, error) {
	query := `
		SELECT id, email, name, organization_id, 
			external_id, provider, system_admin, 
			created_at, last_login
		FROM users
		WHERE id = $1`

	row := r.options.Db.QueryRow(ctx, query, id)

	var user llmgw.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.OrganizationID,
		&user.ExternalID,
		&user.Provider,
		&user.SystemAdmin,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get user", "error", err, "id", id)
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email address
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*llmgw.User, error) {
	query := `
		SELECT id, email, name, organization_id, 
			external_id, provider, system_admin, 
			created_at, last_login
		FROM users
		WHERE email = $1`

	row := r.options.Db.QueryRow(ctx, query, email)

	var user llmgw.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.OrganizationID,
		&user.ExternalID,
		&user.Provider,
		&user.SystemAdmin,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get user by email", "error", err, "email", email)
		return nil, err
	}
	return &user, nil
}

// GetByExternalID retrieves a user by their external provider ID
func (r *UserRepository) GetByExternalID(ctx context.Context, provider, externalID string) (*llmgw.User, error) {
	query := `
		SELECT id, email, name, organization_id, 
			external_id, provider, system_admin, 
			created_at, last_login
		FROM users
		WHERE provider = $1 AND external_id = $2`

	row := r.options.Db.QueryRow(ctx, query, provider, externalID)

	var user llmgw.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.OrganizationID,
		&user.ExternalID,
		&user.Provider,
		&user.SystemAdmin,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get user by external ID",
			"error", err, "provider", provider, "externalID", externalID)
		return nil, err
	}
	return &user, nil
}

// ListByOrganization retrieves all users belonging to an organization
func (r *UserRepository) ListByOrganization(ctx context.Context, orgID string) ([]*llmgw.User, error) {
	query := `
		SELECT id, email, name, organization_id, 
			external_id, provider, system_admin, 
			created_at, last_login
		FROM users
		WHERE organization_id = $1`

	rows, err := r.options.Db.Query(ctx, query, orgID)
	if err != nil {
		r.options.Logger.Error("Failed to list users by organization", "error", err, "orgID", orgID)
		return nil, err
	}
	defer rows.Close()

	var users []*llmgw.User
	for rows.Next() {
		var user llmgw.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.OrganizationID,
			&user.ExternalID,
			&user.Provider,
			&user.SystemAdmin,
			&user.CreatedAt,
			&user.LastLogin,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan user row", "error", err)
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating user rows", "error", err)
		return nil, err
	}

	return users, nil
}

// Update modifies an existing user
func (r *UserRepository) Update(ctx context.Context, user *llmgw.User) error {
	query := `
		UPDATE users SET
			email = $2,
			name = $3,
			organization_id = $4,
			external_id = $5,
			provider = $6,
			system_admin = $7,
			last_login = $8
		WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.OrganizationID,
		user.ExternalID,
		user.Provider,
		user.SystemAdmin,
		user.LastLogin,
	)

	if err != nil {
		r.options.Logger.Error("Failed to update user", "error", err, "id", user.ID)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}

// Delete removes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query, id)
	if err != nil {
		r.options.Logger.Error("Failed to delete user", "error", err, "id", id)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}

// ListByOrganizationForUser retrieves users from the specified organization
// if the requesting user has permission to see them
func (r *UserRepository) ListByOrganizationForUser(ctx context.Context, requestingUser *llmgw.User, orgID string) ([]*llmgw.User, error) {
	// Authorization check: users can only list users from their own organization
	// unless they are system admins
	if !requestingUser.IsSystemAdmin() && requestingUser.OrganizationID != orgID {
		return nil, llmgw.ErrUnauthorized
	}

	return r.ListByOrganization(ctx, orgID)
}

// ListAllForUser retrieves all users visible to the requesting user
// System admins see all users across all organizations
// Regular users see only users from their own organization
func (r *UserRepository) ListAllForUser(ctx context.Context, requestingUser *llmgw.User) ([]*llmgw.User, error) {
	if requestingUser.IsSystemAdmin() {
		query := `
			SELECT id, email, name, organization_id, 
				external_id, provider, system_admin, 
				created_at, last_login
			FROM users
			ORDER BY organization_id, name`

		rows, err := r.options.Db.Query(ctx, query)
		if err != nil {
			r.options.Logger.Error("Failed to list all users", "error", err)
			return nil, err
		}
		defer rows.Close()

		var users []*llmgw.User
		for rows.Next() {
			var user llmgw.User
			err := rows.Scan(
				&user.ID,
				&user.Email,
				&user.Name,
				&user.OrganizationID,
				&user.ExternalID,
				&user.Provider,
				&user.SystemAdmin,
				&user.CreatedAt,
				&user.LastLogin,
			)
			if err != nil {
				r.options.Logger.Error("Failed to scan user row", "error", err)
				return nil, err
			}
			users = append(users, &user)
		}

		return users, rows.Err()
	}

	// Regular users see only users from their own organization
	return r.ListByOrganization(ctx, requestingUser.OrganizationID)
}
