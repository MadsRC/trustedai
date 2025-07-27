// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"codeberg.org/gai-org/gai"
	openrouter "codeberg.org/gai-org/gai-provider-openrouter"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/internal/models"
	"github.com/MadsRC/trustedai/internal/postgres"
)

type ModelRouter struct {
	hardcodedProviders map[string]*models.Provider
	modelRepo          trustedai.ModelRepository
	credentialRepo     trustedai.CredentialRepository
	logger             *slog.Logger
}

// modelIDTransformingClient wraps a provider client to transform aliased model IDs to actual provider model IDs
type modelIDTransformingClient struct {
	gai.ProviderClient
	aliasedModelID string
	actualModelID  string
	logger         *slog.Logger
}

// Generate transforms the model ID in the request before forwarding to the underlying provider
func (c *modelIDTransformingClient) Generate(ctx context.Context, req gai.GenerateRequest) (*gai.Response, error) {
	if req.ModelID == c.aliasedModelID {
		if c.logger != nil {
			c.logger.Debug("Transforming aliased model ID to actual provider model ID",
				"aliased", c.aliasedModelID, "actual", c.actualModelID)
		}
		req.ModelID = c.actualModelID
	}
	return c.ProviderClient.Generate(ctx, req)
}

// GenerateStream transforms the model ID in the request before forwarding to the underlying provider
func (c *modelIDTransformingClient) GenerateStream(ctx context.Context, req gai.GenerateRequest) (gai.ResponseStream, error) {
	if req.ModelID == c.aliasedModelID {
		if c.logger != nil {
			c.logger.Debug("Transforming aliased model ID to actual provider model ID for streaming",
				"aliased", c.aliasedModelID, "actual", c.actualModelID)
		}
		req.ModelID = c.actualModelID
	}
	return c.ProviderClient.GenerateStream(ctx, req)
}

// extractActualModelID extracts the actual provider model ID from a model reference (provider:modelID)
func extractActualModelID(modelReference string) (string, error) {
	parts := strings.SplitN(modelReference, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid model reference format: %s (expected provider:modelID)", modelReference)
	}
	return parts[1], nil
}

type Option func(*ModelRouter)

func WithDatabase(pool postgres.PgxPoolInterface) Option {
	return func(mr *ModelRouter) {
		mr.modelRepo = postgres.NewModelRepository(pool)
		mr.credentialRepo = postgres.NewCredentialRepository(pool)
	}
}

func WithModelRepository(repo trustedai.ModelRepository) Option {
	return func(mr *ModelRouter) {
		mr.modelRepo = repo
	}
}

func WithCredentialRepository(repo trustedai.CredentialRepository) Option {
	return func(mr *ModelRouter) {
		mr.credentialRepo = repo
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(mr *ModelRouter) {
		mr.logger = logger
	}
}

func New(opts ...Option) *ModelRouter {
	mr := &ModelRouter{
		hardcodedProviders: map[string]*models.Provider{
			models.OpenRouterProvider.ID: &models.OpenRouterProvider,
		},
	}

	for _, opt := range opts {
		opt(mr)
	}

	return mr
}

func (mr *ModelRouter) createProviderClient(ctx context.Context, modelWithCreds *trustedai.ModelWithCredentials) (gai.ProviderClient, error) {
	switch modelWithCreds.CredentialType {
	case trustedaiv1.CredentialType_CREDENTIAL_TYPE_OPENROUTER:
		creds, err := mr.credentialRepo.GetOpenRouterCredential(ctx, modelWithCreds.CredentialID)
		if err != nil {
			return nil, fmt.Errorf("failed to get OpenRouter credentials: %w", err)
		}

		opts := []openrouter.ProviderOption{
			openrouter.WithAPIKey(creds.APIKey),
		}
		if mr.logger != nil {
			opts = append(opts, openrouter.WithLogger(mr.logger))
		}
		if creds.SiteName != "" {
			opts = append(opts, openrouter.WithSiteName(creds.SiteName))
		}
		if creds.HTTPReferer != "" {
			opts = append(opts, openrouter.WithHTTPReferer(creds.HTTPReferer))
		}

		return openrouter.New(opts...), nil
	default:
		return nil, fmt.Errorf("unsupported credential type: %s", modelWithCreds.CredentialType.String())
	}
}

func (mr *ModelRouter) RegisterProvider(ctx context.Context, provider gai.ProviderClient) error {
	return errors.New("RegisterProvider is deprecated - providers are now managed through the database")
}

func (mr *ModelRouter) RouteModel(ctx context.Context, modelID string) (gai.ProviderClient, error) {
	if mr.modelRepo == nil || mr.credentialRepo == nil {
		return nil, errors.New("database repositories not configured")
	}

	modelWithCreds, err := mr.modelRepo.GetModelWithCredentials(ctx, modelID)
	if err != nil {
		return nil, err
	}

	providerClient, err := mr.createProviderClient(ctx, modelWithCreds)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider client: %w", err)
	}

	// Extract the actual provider model ID from the model reference in metadata
	modelReference := ""
	if modelWithCreds.Model.Metadata != nil {
		if ref, ok := modelWithCreds.Model.Metadata["model_reference"].(string); ok {
			modelReference = ref
		}
	}

	if modelReference == "" {
		return nil, fmt.Errorf("model reference not found in metadata for model %s", modelID)
	}

	actualModelID, err := extractActualModelID(modelReference)
	if err != nil {
		return nil, fmt.Errorf("failed to extract actual model ID from reference %s: %w", modelReference, err)
	}

	// Create a transforming wrapper that replaces the aliased model ID with the actual provider model ID
	transformingClient := &modelIDTransformingClient{
		ProviderClient: providerClient,
		aliasedModelID: modelID,
		actualModelID:  actualModelID,
		logger:         mr.logger,
	}

	return transformingClient, nil
}

func (mr *ModelRouter) ListModels(ctx context.Context) ([]gai.Model, error) {
	if mr.modelRepo == nil {
		return nil, errors.New("model repository not configured")
	}

	return mr.modelRepo.GetAllModels(ctx)
}

func (mr *ModelRouter) ListProviders() []gai.ProviderClient {
	if mr.logger != nil {
		mr.logger.Debug("ListProviders called - providers are now created dynamically per model")
	}
	return []gai.ProviderClient{}
}

// CacheStatsProvider interface for repositories that support cache statistics
type CacheStatsProvider interface {
	CacheStats() map[string]any
}

// GetCacheStats returns cache statistics if cached repositories are being used
func (mr *ModelRouter) GetCacheStats() map[string]any {
	stats := make(map[string]any)

	// Check if we're using cached repositories
	if cacheProvider, ok := mr.modelRepo.(CacheStatsProvider); ok {
		modelStats := cacheProvider.CacheStats()
		for k, v := range modelStats {
			stats["model_"+k] = v
		}
	}

	if cacheProvider, ok := mr.credentialRepo.(CacheStatsProvider); ok {
		credStats := cacheProvider.CacheStats()
		for k, v := range credStats {
			stats["credential_"+k] = v
		}
	}

	if len(stats) == 0 {
		stats["message"] = "caching not enabled"
	}

	return stats
}
