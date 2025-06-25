// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package modelrouter

import (
	"time"

	"codeberg.org/MadsRC/llmgw/internal/postgres"
)

// ExampleUsage demonstrates how to create a ModelRouter with caching enabled
func ExampleUsage() {
	// Assume you have a database pool
	var pool postgres.PgxPoolInterface

	// Create base repositories
	baseModelRepo := postgres.NewModelRepository(pool)
	baseCredRepo := postgres.NewCredentialRepository(pool)

	// Wrap with caching (5-second TTL)
	cachedModelRepo := postgres.NewCachedModelRepository(baseModelRepo, 5*time.Second)
	cachedCredRepo := postgres.NewCachedCredentialRepository(baseCredRepo, 5*time.Second)

	// Create router with cached repositories
	router := New(
		WithModelRepository(cachedModelRepo),
		WithCredentialRepository(cachedCredRepo),
		WithLogger(nil), // Pass your logger here
	)

	// Alternative: Non-cached version for comparison
	_ = New(
		WithDatabase(pool), // Uses direct database repositories
		WithLogger(nil),
	)

	// Use the router normally - caching is transparent
	_ = router

	// To monitor cache performance:
	stats := router.GetCacheStats()
	_ = stats
	// Stats will contain entries like:
	// - model_cache_size: number of cached models
	// - model_cache_ttl_seconds: cache TTL in seconds
	// - credential_cache_size: number of cached credentials
	// etc.

	// Benefits of caching:
	// 1. Frequently accessed models/credentials are served from memory
	// 2. Reduces database load for read operations
	// 3. Improves response times for chat/completions requests
	// 4. Automatically invalidates cache on writes (create/update/delete)
	// 5. 5-second TTL ensures data freshness while providing performance benefits

	// The cache is completely transparent - no changes needed to existing code
	// that uses the ModelRouter. The same methods work exactly the same way.
}
