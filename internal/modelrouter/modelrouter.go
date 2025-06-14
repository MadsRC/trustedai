// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"maps"

	llm "codeberg.org/gai-org/gai"
)

type ModelRouter struct {
	providers          map[string]llm.ProviderClient
	modelToProvider    map[string]string
	modelAliases       map[string]string
	aliasOnlyMode      bool
	hardcodedModels    map[string]llm.Model
	hardcodedProviders map[string]ProviderConfig
}

type ProviderConfig struct {
	ID   string
	Name string
}

type ModelPricing struct {
	InputTokenPrice  float64
	OutputTokenPrice float64
}

type ModelConfig struct {
	ID                string
	Name              string
	Provider          string
	Pricing           ModelPricing
	SupportsImages    bool
	MaxInputTokens    int
	MaxOutputTokens   int
	SupportsReasoning bool
	SupportsTools     bool
	Capabilities      llm.ModelCapabilities
}

var (
	ErrModelNotFound    = errors.New("model not found")
	ErrProviderNotFound = errors.New("provider not found")
)

func New() *ModelRouter {
	mr := &ModelRouter{
		providers:       make(map[string]llm.ProviderClient),
		modelToProvider: make(map[string]string),
		modelAliases:    make(map[string]string),
		hardcodedModels: make(map[string]llm.Model),
		hardcodedProviders: map[string]ProviderConfig{
			"openrouter": {
				ID:   "openrouter",
				Name: "OpenRouter",
			},
		},
	}

	mr.initializeHardcodedModels()
	return mr
}

func (mr *ModelRouter) initializeHardcodedModels() {
	deepseekModel := llm.Model{
		ID:       "deepseek/deepseek-r1-0528-qwen3-8b:free",
		Name:     "DeepSeek-R1-0528-Qwen3-8B",
		Provider: "openrouter",
		Capabilities: llm.ModelCapabilities{
			SupportsStreaming: true,
			SupportsJSON:      true,
			SupportsFunctions: true,
			SupportsVision:    false,
		},
	}
	llama4MaverickModel := llm.Model{
		ID:       "meta-llama/llama-4-maverick-17b-128e-instruct:free",
		Name:     "LLama-4-Maverick-17b-128e-instruct",
		Provider: "openrouter",
		Capabilities: llm.ModelCapabilities{
			SupportsStreaming: true,
			SupportsJSON:      true,
			SupportsFunctions: true,
			SupportsVision:    false,
		},
	}
	mr.hardcodedModels[deepseekModel.ID] = deepseekModel
	mr.modelToProvider[deepseekModel.ID] = "openrouter"
	mr.hardcodedModels[llama4MaverickModel.ID] = llama4MaverickModel
	mr.modelToProvider[llama4MaverickModel.ID] = "openrouter"
}

func (mr *ModelRouter) GetModelConfig(modelID string) (*ModelConfig, error) {
	resolvedModelID := mr.resolveModelAlias(modelID)

	model, exists := mr.hardcodedModels[resolvedModelID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, resolvedModelID)
	}

	switch resolvedModelID {
	case "deepseek/deepseek-r1-0528-qwen3-8b:free":
		return &ModelConfig{
			ID:       model.ID,
			Name:     model.Name,
			Provider: model.Provider,
			Pricing: ModelPricing{
				InputTokenPrice:  0.000001,
				OutputTokenPrice: 0.000002,
			},
			SupportsImages:    false,
			MaxInputTokens:    32768,
			MaxOutputTokens:   8192,
			SupportsReasoning: true,
			SupportsTools:     true,
			Capabilities:      model.Capabilities,
		}, nil
	case "meta-llama/llama-4-maverick-17b-128e-instruct:free":
		return &ModelConfig{
			ID:       model.ID,
			Name:     model.Name,
			Provider: model.Provider,
			Pricing: ModelPricing{
				InputTokenPrice:  0.0,
				OutputTokenPrice: 0.0,
			},
			SupportsImages:    false,
			MaxInputTokens:    124000,
			MaxOutputTokens:   4000,
			SupportsReasoning: true,
			SupportsTools:     true,
			Capabilities:      model.Capabilities,
		}, nil
	default:
		return nil, fmt.Errorf("model configuration not found for %s", resolvedModelID)
	}
}

func (mr *ModelRouter) RegisterProvider(ctx context.Context, provider llm.ProviderClient) error {
	if provider == nil {
		return errors.New("provider cannot be nil")
	}

	providerID := provider.ID()
	if providerID == "" {
		return errors.New("provider ID cannot be empty")
	}

	mr.providers[providerID] = provider
	return nil
}

func (mr *ModelRouter) RouteModel(ctx context.Context, modelID string) (llm.ProviderClient, error) {
	resolvedModelID := mr.resolveModelAlias(modelID)

	if mr.aliasOnlyMode && !mr.isAliased(modelID) {
		return nil, fmt.Errorf("model %s not allowed in alias-only mode", modelID)
	}

	providerID, exists := mr.modelToProvider[resolvedModelID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, resolvedModelID)
	}

	provider, exists := mr.providers[providerID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerID)
	}

	return provider, nil
}

func (mr *ModelRouter) ListAvailableModels(ctx context.Context) ([]llm.Model, error) {
	var models []llm.Model

	for _, model := range mr.hardcodedModels {
		if mr.aliasOnlyMode {
			if mr.hasAlias(model.ID) {
				models = append(models, model)
			}
		} else {
			models = append(models, model)
		}
	}

	return models, nil
}

func (mr *ModelRouter) AddModelAlias(alias, actualModelID string) {
	if alias == "" || actualModelID == "" {
		return
	}
	mr.modelAliases[alias] = actualModelID
}

func (mr *ModelRouter) RemoveModelAlias(alias string) {
	delete(mr.modelAliases, alias)
}

func (mr *ModelRouter) ListModelAliases() map[string]string {
	aliases := make(map[string]string)
	maps.Copy(aliases, mr.modelAliases)
	return aliases
}

func (mr *ModelRouter) SetAliasOnlyMode(enabled bool) {
	mr.aliasOnlyMode = enabled
}

func (mr *ModelRouter) IsAliasOnlyMode() bool {
	return mr.aliasOnlyMode
}

func (mr *ModelRouter) resolveModelAlias(modelID string) string {
	if actualID, exists := mr.modelAliases[modelID]; exists {
		return actualID
	}
	return modelID
}

func (mr *ModelRouter) isAliased(modelID string) bool {
	_, exists := mr.modelAliases[modelID]
	return exists
}

func (mr *ModelRouter) hasAlias(actualModelID string) bool {
	for _, id := range mr.modelAliases {
		if id == actualModelID {
			return true
		}
	}
	return false
}
