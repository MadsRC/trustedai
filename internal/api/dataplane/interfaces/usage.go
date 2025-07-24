// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package interfaces

import (
	"context"
	"time"
)

// UsageMiddleware interface for tracking usage events
type UsageMiddleware interface {
	UpdateEvent(ctx context.Context, modelID string, usage *TokenUsage, status string, duration time.Duration)
}

// TokenUsage represents token usage data from GAI responses
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
