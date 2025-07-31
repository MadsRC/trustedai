// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"encoding/json"

	"codeberg.org/gai-org/gai"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/internal/models"
	"github.com/google/uuid"
)

type ModelRepository struct {
	pool PgxPoolInterface
}

func NewModelRepository(pool PgxPoolInterface) *ModelRepository {
	return &ModelRepository{pool: pool}
}

func (r *ModelRepository) GetAllModels(ctx context.Context) ([]trustedai.ModelWithCredentials, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities, m.credential_id, m.credential_type, m.metadata
		FROM models m
		WHERE m.enabled = true
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modelList []trustedai.ModelWithCredentials
	for rows.Next() {
		var modelWithCreds trustedai.ModelWithCredentials
		var pricingJSON, capabilitiesJSON, metadataJSON []byte
		var credentialTypeInt int

		err := rows.Scan(
			&modelWithCreds.Model.ID,
			&modelWithCreds.Model.Name,
			&modelWithCreds.Model.Provider,
			&pricingJSON,
			&capabilitiesJSON,
			&modelWithCreds.CredentialID,
			&credentialTypeInt,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		// Convert integer credential type to CredentialType
		modelWithCreds.CredentialType = trustedaiv1.CredentialType(credentialTypeInt)

		if err := json.Unmarshal(pricingJSON, &modelWithCreds.Model.Pricing); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(capabilitiesJSON, &modelWithCreds.Model.Capabilities); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &modelWithCreds.Model.Metadata); err != nil {
			return nil, err
		}

		modelList = append(modelList, modelWithCreds)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return modelList, nil
}

func (r *ModelRepository) GetModelByID(ctx context.Context, modelID string) (*trustedai.ModelWithCredentials, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities, m.credential_id, m.credential_type, m.metadata
		FROM models m
		WHERE m.id = $1 AND m.enabled = true
	`

	row := r.pool.QueryRow(ctx, query, modelID)

	var modelWithCreds trustedai.ModelWithCredentials
	var pricingJSON, capabilitiesJSON, metadataJSON []byte
	var credentialTypeInt int

	err := row.Scan(
		&modelWithCreds.Model.ID,
		&modelWithCreds.Model.Name,
		&modelWithCreds.Model.Provider,
		&pricingJSON,
		&capabilitiesJSON,
		&modelWithCreds.CredentialID,
		&credentialTypeInt,
		&metadataJSON,
	)
	if err != nil {
		return nil, models.ErrModelNotFound
	}

	// Convert integer credential type to CredentialType
	modelWithCreds.CredentialType = trustedaiv1.CredentialType(credentialTypeInt)

	if err := json.Unmarshal(pricingJSON, &modelWithCreds.Model.Pricing); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(capabilitiesJSON, &modelWithCreds.Model.Capabilities); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(metadataJSON, &modelWithCreds.Model.Metadata); err != nil {
		return nil, err
	}

	return &modelWithCreds, nil
}

func (r *ModelRepository) CreateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	query := `
		INSERT INTO models (id, name, provider_id, pricing, capabilities, credential_id, credential_type, metadata, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	pricingJSON, err := json.Marshal(model.Pricing)
	if err != nil {
		return err
	}

	capabilitiesJSON, err := json.Marshal(model.Capabilities)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(model.Metadata)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Provider,
		pricingJSON,
		capabilitiesJSON,
		credentialID,
		int(credentialType),
		metadataJSON,
		true, // enabled by default
	)
	return err
}

func (r *ModelRepository) UpdateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	query := `
		UPDATE models 
		SET name = $2, provider_id = $3, pricing = $4, capabilities = $5, credential_id = $6, credential_type = $7, metadata = $8
		WHERE id = $1
	`

	pricingJSON, err := json.Marshal(model.Pricing)
	if err != nil {
		return err
	}

	capabilitiesJSON, err := json.Marshal(model.Capabilities)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(model.Metadata)
	if err != nil {
		return err
	}

	result, err := r.pool.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Provider,
		pricingJSON,
		capabilitiesJSON,
		credentialID,
		int(credentialType),
		metadataJSON,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return models.ErrModelNotFound
	}

	return nil
}

func (r *ModelRepository) DeleteModel(ctx context.Context, modelID string) error {
	query := `UPDATE models SET enabled = false WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, modelID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return models.ErrModelNotFound
	}

	return nil
}
