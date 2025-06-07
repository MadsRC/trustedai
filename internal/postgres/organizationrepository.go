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

// Create adds a new organization to the database
func (r *OrganizationRepository) Create(ctx context.Context, org *llmgw.Organization) error {
	query := `
		INSERT INTO organizations (
			id, name, display_name, is_system, 
			created_at, sso_type, sso_config
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.options.Db.Exec(ctx, query,
		org.ID,
		org.Name,
		org.DisplayName,
		org.IsSystem,
		org.CreatedAt,
		org.SSOType,
		org.SSOConfig,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return llmgw.ErrDuplicateEntry
		}
		r.options.Logger.Error("Failed to create organization", "error", err)
		return err
	}
	return nil
}

// Get retrieves an organization by ID
func (r *OrganizationRepository) Get(ctx context.Context, id string) (*llmgw.Organization, error) {
	query := `
		SELECT id, name, display_name, is_system, 
			created_at, sso_type, sso_config
		FROM organizations
		WHERE id = $1`

	row := r.options.Db.QueryRow(ctx, query, id)

	var org llmgw.Organization
	err := row.Scan(
		&org.ID,
		&org.Name,
		&org.DisplayName,
		&org.IsSystem,
		&org.CreatedAt,
		&org.SSOType,
		&org.SSOConfig,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get organization", "error", err, "id", id)
		return nil, err
	}
	return &org, nil
}

// GetByName retrieves an organization by its unique name
func (r *OrganizationRepository) GetByName(ctx context.Context, name string) (*llmgw.Organization, error) {
	query := `
		SELECT id, name, display_name, is_system, 
			created_at, sso_type, sso_config
		FROM organizations
		WHERE name = $1`

	row := r.options.Db.QueryRow(ctx, query, name)

	var org llmgw.Organization
	err := row.Scan(
		&org.ID,
		&org.Name,
		&org.DisplayName,
		&org.IsSystem,
		&org.CreatedAt,
		&org.SSOType,
		&org.SSOConfig,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get organization by name", "error", err, "name", name)
		return nil, err
	}
	return &org, nil
}

// List retrieves all organizations
func (r *OrganizationRepository) List(ctx context.Context) ([]*llmgw.Organization, error) {
	query := `
		SELECT id, name, display_name, is_system, 
			created_at, sso_type, sso_config
		FROM organizations
		ORDER BY name`

	rows, err := r.options.Db.Query(ctx, query)
	if err != nil {
		r.options.Logger.Error("Failed to list organizations", "error", err)
		return nil, err
	}
	defer rows.Close()

	var orgs []*llmgw.Organization
	for rows.Next() {
		var org llmgw.Organization
		err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.DisplayName,
			&org.IsSystem,
			&org.CreatedAt,
			&org.SSOType,
			&org.SSOConfig,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan organization row", "error", err)
			return nil, err
		}
		orgs = append(orgs, &org)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating organization rows", "error", err)
		return nil, err
	}

	return orgs, nil
}

// Update modifies an existing organization
func (r *OrganizationRepository) Update(ctx context.Context, org *llmgw.Organization) error {
	query := `
		UPDATE organizations SET
			name = $2,
			display_name = $3,
			is_system = $4,
			sso_type = $5,
			sso_config = $6
		WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query,
		org.ID,
		org.Name,
		org.DisplayName,
		org.IsSystem,
		org.SSOType,
		org.SSOConfig,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return llmgw.ErrDuplicateEntry
		}
		r.options.Logger.Error("Failed to update organization", "error", err, "id", org.ID)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}

// Delete removes an organization by ID
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM organizations WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query, id)
	if err != nil {
		r.options.Logger.Error("Failed to delete organization", "error", err, "id", id)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}
