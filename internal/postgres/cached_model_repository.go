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
	underlying            trustedai.ModelRepository
	modelCache            *cache.Cache[string, *gai.Model]
	modelWithCredsCache   *cache.Cache[string, *trustedai.ModelWithCredentials]
	modelWithRefCache     *cache.Cache[string, *trustedai.ModelWithReference]
	allModelsCache        *cache.Cache[string, []gai.Model]
	allModelsWithRefCache *cache.Cache[string, []trustedai.ModelWithReference]
	cacheTTL              time.Duration
}

// NewCachedModelRepository creates a new cached model repository
func NewCachedModelRepository(underlying trustedai.ModelRepository, cacheTTL time.Duration) *CachedModelRepository {
	return &CachedModelRepository{
		underlying:            underlying,
		modelCache:            cache.New[string, *gai.Model](cacheTTL),
		modelWithCredsCache:   cache.New[string, *trustedai.ModelWithCredentials](cacheTTL),
		modelWithRefCache:     cache.New[string, *trustedai.ModelWithReference](cacheTTL),
		allModelsCache:        cache.New[string, []gai.Model](cacheTTL),
		allModelsWithRefCache: cache.New[string, []trustedai.ModelWithReference](cacheTTL),
		cacheTTL:              cacheTTL,
	}
}

// GetAllModels retrieves all models with caching
func (r *CachedModelRepository) GetAllModels(ctx context.Context) ([]gai.Model, error) {
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
		r.modelCache.Set(model.ID, &model)
	}

	return models, nil
}

// GetAllModelsWithReference retrieves all models with references with caching
func (r *CachedModelRepository) GetAllModelsWithReference(ctx context.Context) ([]trustedai.ModelWithReference, error) {
	cacheKey := "all_models_with_ref"

	// Try cache first
	if cached, found := r.allModelsWithRefCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	models, err := r.underlying.GetAllModelsWithReference(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.allModelsWithRefCache.Set(cacheKey, models)

	// Also cache individual models
	for _, model := range models {
		r.modelCache.Set(model.Model.ID, &model.Model)
		r.modelWithRefCache.Set(model.Model.ID, &model)
	}

	return models, nil
}

// GetModelByID retrieves a model by ID with caching
func (r *CachedModelRepository) GetModelByID(ctx context.Context, modelID string) (*gai.Model, error) {
	// Try cache first
	if cached, found := r.modelCache.Get(modelID); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	model, err := r.underlying.GetModelByID(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.modelCache.Set(modelID, model)

	return model, nil
}

// GetModelByIDWithReference retrieves a model with reference by ID with caching
func (r *CachedModelRepository) GetModelByIDWithReference(ctx context.Context, modelID string) (*trustedai.ModelWithReference, error) {
	// Try cache first
	if cached, found := r.modelWithRefCache.Get(modelID); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	model, err := r.underlying.GetModelByIDWithReference(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.modelWithRefCache.Set(modelID, model)
	r.modelCache.Set(modelID, &model.Model)

	return model, nil
}

// GetModelWithCredentials retrieves a model with credentials by ID with caching
func (r *CachedModelRepository) GetModelWithCredentials(ctx context.Context, modelID string) (*trustedai.ModelWithCredentials, error) {
	// Try cache first
	if cached, found := r.modelWithCredsCache.Get(modelID); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	model, err := r.underlying.GetModelWithCredentials(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.modelWithCredsCache.Set(modelID, model)
	r.modelCache.Set(modelID, &model.Model)

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
	r.allModelsWithRefCache.Clear()

	// Cache the new model entries
	r.modelCache.Set(model.ID, model)
	r.modelWithCredsCache.Set(model.ID, &trustedai.ModelWithCredentials{
		Model:          *model,
		CredentialID:   credentialID,
		CredentialType: credentialType,
	})
	r.modelWithRefCache.Set(model.ID, &trustedai.ModelWithReference{
		Model: *model,
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
	r.modelCache.Delete(model.ID)
	r.modelWithCredsCache.Delete(model.ID)
	r.modelWithRefCache.Delete(model.ID)
	r.allModelsCache.Clear()
	r.allModelsWithRefCache.Clear()

	return nil
}

// DeleteModel removes a model and invalidates caches
func (r *CachedModelRepository) DeleteModel(ctx context.Context, modelID string) error {
	err := r.underlying.DeleteModel(ctx, modelID)
	if err != nil {
		return err
	}

	// Invalidate caches
	r.modelCache.Delete(modelID)
	r.modelWithCredsCache.Delete(modelID)
	r.modelWithRefCache.Delete(modelID)
	r.allModelsCache.Clear()
	r.allModelsWithRefCache.Clear()

	return nil
}

// Close stops the cache cleanup goroutines
func (r *CachedModelRepository) Close() {
	r.modelCache.Close()
	r.modelWithCredsCache.Close()
	r.modelWithRefCache.Close()
	r.allModelsCache.Close()
	r.allModelsWithRefCache.Close()
}

// CacheStats returns cache statistics for monitoring
func (r *CachedModelRepository) CacheStats() map[string]any {
	return map[string]any{
		"model_cache_size":               r.modelCache.Size(),
		"model_with_creds_cache_size":    r.modelWithCredsCache.Size(),
		"model_with_ref_cache_size":      r.modelWithRefCache.Size(),
		"all_models_cache_size":          r.allModelsCache.Size(),
		"all_models_with_ref_cache_size": r.allModelsWithRefCache.Size(),
		"cache_ttl_seconds":              r.cacheTTL.Seconds(),
	}
}
