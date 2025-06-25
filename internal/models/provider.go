// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package models

import (
	"fmt"
	"strings"

	"codeberg.org/gai-org/gai"
)

const (
	PROVIDER_ID_UNKNOWN    = ""
	PROVIDER_ID_OPENROUTER = "openrouter"
)

type Provider struct {
	ID     string
	Name   string
	Models map[string]gai.Model
}

var OpenRouterProvider = Provider{
	ID:     PROVIDER_ID_OPENROUTER,
	Name:   "OpenRouter",
	Models: OpenRouterModels,
}

// GetModelByReference looks up a hardcoded model by composite reference (provider:modelID)
func GetModelByReference(modelReference string) (*gai.Model, error) {
	parts := strings.SplitN(modelReference, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid model reference format: %s (expected provider:modelID)", modelReference)
	}

	providerID := parts[0]
	modelID := parts[1]

	switch providerID {
	case PROVIDER_ID_OPENROUTER:
		if model, exists := OpenRouterModels[modelID]; exists {
			return &model, nil
		}
		return nil, fmt.Errorf("model %s not found for provider %s", modelID, providerID)
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}
