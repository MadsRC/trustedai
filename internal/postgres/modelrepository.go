// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"encoding/json"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/models"
	"codeberg.org/gai-org/gai"
	"github.com/google/uuid"
)

type ModelRepository struct {
	pool PgxPoolInterface
}

func NewModelRepository(pool PgxPoolInterface) *ModelRepository {
	return &ModelRepository{pool: pool}
}

func (r *ModelRepository) GetAllModels(ctx context.Context) ([]gai.Model, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities
		FROM models m
		WHERE m.enabled = true
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modelList []gai.Model
	for rows.Next() {
		var model gai.Model
		var pricingJSON, capabilitiesJSON []byte

		err := rows.Scan(
			&model.ID,
			&model.Name,
			&model.Provider,
			&pricingJSON,
			&capabilitiesJSON,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(pricingJSON, &model.Pricing); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(capabilitiesJSON, &model.Capabilities); err != nil {
			return nil, err
		}

		modelList = append(modelList, model)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return modelList, nil
}

func (r *ModelRepository) GetAllModelsWithReference(ctx context.Context) ([]llmgw.ModelWithReference, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities, m.model_reference
		FROM models m
		WHERE m.enabled = true
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modelList []llmgw.ModelWithReference
	for rows.Next() {
		var modelWithRef llmgw.ModelWithReference
		var pricingJSON, capabilitiesJSON []byte

		err := rows.Scan(
			&modelWithRef.Model.ID,
			&modelWithRef.Model.Name,
			&modelWithRef.Model.Provider,
			&pricingJSON,
			&capabilitiesJSON,
			&modelWithRef.ModelReference,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(pricingJSON, &modelWithRef.Model.Pricing); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(capabilitiesJSON, &modelWithRef.Model.Capabilities); err != nil {
			return nil, err
		}

		modelList = append(modelList, modelWithRef)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return modelList, nil
}

func (r *ModelRepository) GetModelByID(ctx context.Context, modelID string) (*gai.Model, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities
		FROM models m
		WHERE m.id = $1 AND m.enabled = true
	`

	row := r.pool.QueryRow(ctx, query, modelID)

	var model gai.Model
	var pricingJSON, capabilitiesJSON []byte

	err := row.Scan(
		&model.ID,
		&model.Name,
		&model.Provider,
		&pricingJSON,
		&capabilitiesJSON,
	)
	if err != nil {
		return nil, models.ErrModelNotFound
	}

	if err := json.Unmarshal(pricingJSON, &model.Pricing); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(capabilitiesJSON, &model.Capabilities); err != nil {
		return nil, err
	}

	return &model, nil
}

func (r *ModelRepository) GetModelByIDWithReference(ctx context.Context, modelID string) (*llmgw.ModelWithReference, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities, m.model_reference
		FROM models m
		WHERE m.id = $1 AND m.enabled = true
	`

	row := r.pool.QueryRow(ctx, query, modelID)

	var modelWithRef llmgw.ModelWithReference
	var pricingJSON, capabilitiesJSON []byte

	err := row.Scan(
		&modelWithRef.Model.ID,
		&modelWithRef.Model.Name,
		&modelWithRef.Model.Provider,
		&pricingJSON,
		&capabilitiesJSON,
		&modelWithRef.ModelReference,
	)
	if err != nil {
		return nil, models.ErrModelNotFound
	}

	if err := json.Unmarshal(pricingJSON, &modelWithRef.Model.Pricing); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(capabilitiesJSON, &modelWithRef.Model.Capabilities); err != nil {
		return nil, err
	}

	return &modelWithRef, nil
}

func (r *ModelRepository) GetModelWithCredentials(ctx context.Context, modelID string) (*llmgw.ModelWithCredentials, error) {
	query := `
		SELECT m.id, m.name, m.provider_id, m.pricing, m.capabilities, m.credential_id, m.credential_type, m.model_reference
		FROM models m
		WHERE m.id = $1 AND m.enabled = true
	`

	row := r.pool.QueryRow(ctx, query, modelID)

	var modelWithCreds llmgw.ModelWithCredentials
	var pricingJSON, capabilitiesJSON []byte

	err := row.Scan(
		&modelWithCreds.Model.ID,
		&modelWithCreds.Model.Name,
		&modelWithCreds.Model.Provider,
		&pricingJSON,
		&capabilitiesJSON,
		&modelWithCreds.CredentialID,
		&modelWithCreds.CredentialType,
		&modelWithCreds.ModelReference,
	)
	if err != nil {
		return nil, models.ErrModelNotFound
	}

	if err := json.Unmarshal(pricingJSON, &modelWithCreds.Model.Pricing); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(capabilitiesJSON, &modelWithCreds.Model.Capabilities); err != nil {
		return nil, err
	}

	return &modelWithCreds, nil
}

func (r *ModelRepository) CreateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType string, modelReference string) error {
	query := `
		INSERT INTO models (id, name, provider_id, pricing, capabilities, credential_id, credential_type, model_reference, enabled)
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

	_, err = r.pool.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Provider,
		pricingJSON,
		capabilitiesJSON,
		credentialID,
		credentialType,
		modelReference,
		true, // enabled by default
	)
	return err
}

func (r *ModelRepository) UpdateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType string, modelReference string) error {
	query := `
		UPDATE models 
		SET name = $2, provider_id = $3, pricing = $4, capabilities = $5, credential_id = $6, credential_type = $7, model_reference = $8
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

	result, err := r.pool.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Provider,
		pricingJSON,
		capabilitiesJSON,
		credentialID,
		credentialType,
		modelReference,
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
