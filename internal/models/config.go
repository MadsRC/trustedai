// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package models

import (
	"codeberg.org/gai-org/gai"
)

var OpenRouterModels = map[string]gai.Model{
	"deepseek/deepseek-r1-0528-qwen3-8b:free": {
		ID:       "deepseek/deepseek-r1-0528-qwen3-8b:free",
		Name:     "DeepSeek-R1-0528-Qwen3-8B",
		Provider: PROVIDER_ID_OPENROUTER,
		Pricing: gai.ModelPricing{
			InputTokenPrice:  0.0,
			OutputTokenPrice: 0.0,
		},
		Capabilities: gai.ModelCapabilities{
			SupportsStreaming: true,
			SupportsJSON:      true,
			SupportsTools:     true,
			SupportsVision:    false,
			SupportsReasoning: true,
			MaxInputTokens:    32768,
			MaxOutputTokens:   8192,
		},
	},
	"meta-llama/llama-4-maverick-17b-128e-instruct:free": {
		ID:       "meta-llama/llama-4-maverick-17b-128e-instruct:free",
		Name:     "Llama-4-Maverick-17b-128e-instruct",
		Provider: PROVIDER_ID_OPENROUTER,
		Pricing: gai.ModelPricing{
			InputTokenPrice:  0.0,
			OutputTokenPrice: 0.0,
		},
		Capabilities: gai.ModelCapabilities{
			SupportsStreaming: true,
			SupportsJSON:      true,
			SupportsTools:     true,
			SupportsVision:    false,
			SupportsReasoning: true,
			MaxInputTokens:    124000,
			MaxOutputTokens:   4000,
		},
	},
}
