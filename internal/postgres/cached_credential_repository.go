// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/cache"
	"github.com/google/uuid"
)

// CachedCredentialRepository wraps a CredentialRepository with caching
type CachedCredentialRepository struct {
	underlying          llmgw.CredentialRepository
	openRouterCredCache *cache.Cache[uuid.UUID, *llmgw.OpenRouterCredential]
	openRouterListCache *cache.Cache[string, []llmgw.OpenRouterCredential]
	cacheTTL            time.Duration
}

// NewCachedCredentialRepository creates a new cached credential repository
func NewCachedCredentialRepository(underlying llmgw.CredentialRepository, cacheTTL time.Duration) *CachedCredentialRepository {
	return &CachedCredentialRepository{
		underlying:          underlying,
		openRouterCredCache: cache.New[uuid.UUID, *llmgw.OpenRouterCredential](cacheTTL),
		openRouterListCache: cache.New[string, []llmgw.OpenRouterCredential](cacheTTL),
		cacheTTL:            cacheTTL,
	}
}

// GetOpenRouterCredential retrieves an OpenRouter credential with caching
func (r *CachedCredentialRepository) GetOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) (*llmgw.OpenRouterCredential, error) {
	// Try cache first
	if cached, found := r.openRouterCredCache.Get(credentialID); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	credential, err := r.underlying.GetOpenRouterCredential(ctx, credentialID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.openRouterCredCache.Set(credentialID, credential)

	return credential, nil
}

// ListOpenRouterCredentials retrieves all OpenRouter credentials with caching
func (r *CachedCredentialRepository) ListOpenRouterCredentials(ctx context.Context) ([]llmgw.OpenRouterCredential, error) {
	cacheKey := "all_openrouter_credentials"

	// Try cache first
	if cached, found := r.openRouterListCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - fetch from underlying repository
	credentials, err := r.underlying.ListOpenRouterCredentials(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.openRouterListCache.Set(cacheKey, credentials)

	// Also cache individual credentials for Get operations
	for _, cred := range credentials {
		r.openRouterCredCache.Set(cred.ID, &cred)
	}

	return credentials, nil
}

// CreateOpenRouterCredential creates a new OpenRouter credential and invalidates cache
func (r *CachedCredentialRepository) CreateOpenRouterCredential(ctx context.Context, cred *llmgw.OpenRouterCredential) error {
	err := r.underlying.CreateOpenRouterCredential(ctx, cred)
	if err != nil {
		return err
	}

	// Invalidate list cache since we added a new credential
	r.openRouterListCache.Clear()

	// Cache the new credential
	r.openRouterCredCache.Set(cred.ID, cred)

	return nil
}

// UpdateOpenRouterCredential updates an existing OpenRouter credential and invalidates cache
func (r *CachedCredentialRepository) UpdateOpenRouterCredential(ctx context.Context, cred *llmgw.OpenRouterCredential) error {
	err := r.underlying.UpdateOpenRouterCredential(ctx, cred)
	if err != nil {
		return err
	}

	// Invalidate caches
	r.openRouterCredCache.Delete(cred.ID)
	r.openRouterListCache.Clear()

	return nil
}

// DeleteOpenRouterCredential removes an OpenRouter credential and invalidates cache
func (r *CachedCredentialRepository) DeleteOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) error {
	err := r.underlying.DeleteOpenRouterCredential(ctx, credentialID)
	if err != nil {
		return err
	}

	// Invalidate caches
	r.openRouterCredCache.Delete(credentialID)
	r.openRouterListCache.Clear()

	return nil
}

// Close stops the cache cleanup goroutines
func (r *CachedCredentialRepository) Close() {
	r.openRouterCredCache.Close()
	r.openRouterListCache.Close()
}

// CacheStats returns cache statistics for monitoring
func (r *CachedCredentialRepository) CacheStats() map[string]any {
	return map[string]any{
		"credential_cache_size": r.openRouterCredCache.Size(),
		"list_cache_size":       r.openRouterListCache.Size(),
		"cache_ttl_seconds":     r.cacheTTL.Seconds(),
	}
}
