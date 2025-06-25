// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"

	"codeberg.org/MadsRC/llmgw"
	"github.com/google/uuid"
)

type CredentialRepository struct {
	pool PgxPoolInterface
}

func NewCredentialRepository(pool PgxPoolInterface) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

func (r *CredentialRepository) GetOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) (*llmgw.OpenRouterCredential, error) {
	query := `
		SELECT id, name, description, api_key, site_name, http_referer, enabled
		FROM openrouter_credentials
		WHERE id = $1 AND enabled = true
	`

	row := r.pool.QueryRow(ctx, query, credentialID)

	var cred llmgw.OpenRouterCredential
	err := row.Scan(
		&cred.ID,
		&cred.Name,
		&cred.Description,
		&cred.APIKey,
		&cred.SiteName,
		&cred.HTTPReferer,
		&cred.Enabled,
	)
	if err != nil {
		return nil, err
	}

	return &cred, nil
}

func (r *CredentialRepository) ListOpenRouterCredentials(ctx context.Context) ([]llmgw.OpenRouterCredential, error) {
	query := `
		SELECT id, name, description, api_key, site_name, http_referer, enabled
		FROM openrouter_credentials
		WHERE enabled = true
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []llmgw.OpenRouterCredential
	for rows.Next() {
		var cred llmgw.OpenRouterCredential
		err := rows.Scan(
			&cred.ID,
			&cred.Name,
			&cred.Description,
			&cred.APIKey,
			&cred.SiteName,
			&cred.HTTPReferer,
			&cred.Enabled,
		)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, cred)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return credentials, nil
}

func (r *CredentialRepository) CreateOpenRouterCredential(ctx context.Context, cred *llmgw.OpenRouterCredential) error {
	query := `
		INSERT INTO openrouter_credentials (id, name, description, api_key, site_name, http_referer, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if cred.ID == uuid.Nil {
		cred.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		cred.ID,
		cred.Name,
		cred.Description,
		cred.APIKey,
		cred.SiteName,
		cred.HTTPReferer,
		cred.Enabled,
	)

	return err
}

func (r *CredentialRepository) UpdateOpenRouterCredential(ctx context.Context, cred *llmgw.OpenRouterCredential) error {
	query := `
		UPDATE openrouter_credentials 
		SET name = $2, description = $3, api_key = $4, site_name = $5, http_referer = $6, enabled = $7, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		cred.ID,
		cred.Name,
		cred.Description,
		cred.APIKey,
		cred.SiteName,
		cred.HTTPReferer,
		cred.Enabled,
	)

	return err
}

func (r *CredentialRepository) DeleteOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) error {
	query := `UPDATE openrouter_credentials SET enabled = false, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, credentialID)
	return err
}
