// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"time"

	"codeberg.org/gai-org/gai"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/internal/cache"
	"github.com/google/uuid"
)

// CachedModelRepository wraps a ModelRepository with caching
type CachedModelRepository struct {
	underlying          trustedai.ModelRepository
	modelWithCredsCache *cache.Cache[string, *trustedai.ModelWithCredentials]
	allModelsCache      *cache.Cache[string, []trustedai.ModelWithCredentials]
	cacheTTL            time.Duration
}

// NewCachedModelRepository creates a new cached model repository
func NewCachedModelRepository(underlying trustedai.ModelRepository, cacheTTL time.Duration) *CachedModelRepository {
	return &CachedModelRepository{
		underlying:          underlying,
		modelWithCredsCache: cache.New[string, *trustedai.ModelWithCredentials](cacheTTL),
		allModelsCache:      cache.New[string, []trustedai.ModelWithCredentials](cacheTTL),
		cacheTTL:            cacheTTL,
	}
}

// GetAllModels retrieves all models with caching
func (r *CachedModelRepository) GetAllModels(ctx context.Context) ([]trustedai.ModelWithCredentials, error) {
	cacheKey := "all_models"

	// Try cache first
	if cached, found := r.allModelsCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	models, err := r.underlying.GetAllModels(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.allModelsCache.Set(cacheKey, models)

	// Also cache individual models
	for _, model := range models {
		r.modelWithCredsCache.Set(model.Model.ID, &model)
	}

	return models, nil
}

// GetModelByID retrieves a model by ID with caching
func (r *CachedModelRepository) GetModelByID(ctx context.Context, modelID string) (*trustedai.ModelWithCredentials, error) {
	// Try cache first
	if cached, found := r.modelWithCredsCache.Get(modelID); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	model, err := r.underlying.GetModelByID(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.modelWithCredsCache.Set(modelID, model)

	return model, nil
}

// CreateModel creates a new model and invalidates caches
func (r *CachedModelRepository) CreateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	err := r.underlying.CreateModel(ctx, model, credentialID, credentialType)
	if err != nil {
		return err
	}

	// Invalidate list caches since we added a new model
	r.allModelsCache.Clear()

	// Cache the new model entry
	r.modelWithCredsCache.Set(model.ID, &trustedai.ModelWithCredentials{
		Model:          *model,
		CredentialID:   credentialID,
		CredentialType: credentialType,
	})

	return nil
}

// UpdateModel updates an existing model and invalidates caches
func (r *CachedModelRepository) UpdateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	err := r.underlying.UpdateModel(ctx, model, credentialID, credentialType)
	if err != nil {
		return err
	}

	// Invalidate caches for this model
	r.modelWithCredsCache.Delete(model.ID)
	r.allModelsCache.Clear()

	return nil
}

// DeleteModel removes a model and invalidates caches
func (r *CachedModelRepository) DeleteModel(ctx context.Context, modelID string) error {
	err := r.underlying.DeleteModel(ctx, modelID)
	if err != nil {
		return err
	}

	// Invalidate caches
	r.modelWithCredsCache.Delete(modelID)
	r.allModelsCache.Clear()

	return nil
}

// Close stops the cache cleanup goroutines
func (r *CachedModelRepository) Close() {
	r.modelWithCredsCache.Close()
	r.allModelsCache.Close()
}

// CacheStats returns cache statistics for monitoring
func (r *CachedModelRepository) CacheStats() map[string]any {
	return map[string]any{
		"model_with_creds_cache_size": r.modelWithCredsCache.Size(),
		"all_models_cache_size":       r.allModelsCache.Size(),
		"cache_ttl_seconds":           r.cacheTTL.Seconds(),
	}
}
