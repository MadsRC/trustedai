// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package trustedai

import (
	"context"
	"time"
)

// UsageEvent represents a single usage tracking event
type UsageEvent struct {
	ID              string    `json:"id"`
	RequestID       string    `json:"requestId"`
	UserID          string    `json:"userId"`
	ModelID         string    `json:"modelId"`
	InputTokens     *int      `json:"inputTokens,omitempty"`
	OutputTokens    *int      `json:"outputTokens,omitempty"`
	CachedTokens    *int      `json:"cachedTokens,omitempty"`
	ReasoningTokens *int      `json:"reasoningTokens,omitempty"`
	Status          string    `json:"status"`
	FailureStage    *string   `json:"failureStage,omitempty"`
	ErrorType       *string   `json:"errorType,omitempty"`
	ErrorMessage    *string   `json:"errorMessage,omitempty"`
	UsageDataSource string    `json:"usageDataSource"`
	DataComplete    bool      `json:"dataComplete"`
	Timestamp       time.Time `json:"timestamp"`
	DurationMs      *int      `json:"durationMs,omitempty"`
	InputCostCents  *float64  `json:"inputCostCents,omitempty"`
	OutputCostCents *float64  `json:"outputCostCents,omitempty"`
	TotalCostCents  *float64  `json:"totalCostCents,omitempty"`
}

// BillingSummary represents pre-aggregated billing data for a user and period
type BillingSummary struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	PeriodStart       time.Time `json:"periodStart"`
	PeriodEnd         time.Time `json:"periodEnd"`
	TotalRequests     int       `json:"totalRequests"`
	TotalInputTokens  int64     `json:"totalInputTokens"`
	TotalOutputTokens int64     `json:"totalOutputTokens"`
	TotalCostCents    float64   `json:"totalCostCents"`
	CreatedAt         time.Time `json:"createdAt"`
}

// CostResult represents the result of a cost calculation
type CostResult struct {
	InputCostCents  float64 `json:"inputCostCents"`
	OutputCostCents float64 `json:"outputCostCents"`
	TotalCostCents  float64 `json:"totalCostCents"`
}

// UsageRepository defines persistence operations for usage events
type UsageRepository interface {
	// CreateUsageEvent stores a new usage event
	CreateUsageEvent(ctx context.Context, event *UsageEvent) error

	// GetUsageEvent retrieves a usage event by ID
	GetUsageEvent(ctx context.Context, id string) (*UsageEvent, error)

	// ListUsageEventsByUser retrieves usage events for a specific user with pagination
	ListUsageEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*UsageEvent, error)

	// ListUsageEventsForCostCalculation retrieves uncalculated usage events that are ready for cost calculation
	ListUsageEventsForCostCalculation(ctx context.Context, limit int) ([]*UsageEvent, error)

	// UpdateUsageEventCost updates the cost fields for a usage event
	UpdateUsageEventCost(ctx context.Context, eventID string, cost CostResult) error

	// ListUsageEventsByPeriod retrieves usage events for a specific period
	ListUsageEventsByPeriod(ctx context.Context, userID string, start, end time.Time) ([]*UsageEvent, error)
}

// BillingRepository defines persistence operations for billing summaries
type BillingRepository interface {
	// CreateBillingSummary stores a new billing summary
	CreateBillingSummary(ctx context.Context, summary *BillingSummary) error

	// GetBillingSummary retrieves a billing summary by ID
	GetBillingSummary(ctx context.Context, id string) (*BillingSummary, error)

	// ListBillingSummariesByUser retrieves billing summaries for a specific user
	ListBillingSummariesByUser(ctx context.Context, userID string, limit, offset int) ([]*BillingSummary, error)

	// ListBillingSummariesByPeriod retrieves billing summaries for a specific period
	ListBillingSummariesByPeriod(ctx context.Context, start, end time.Time) ([]*BillingSummary, error)

	// GetBillingSummaryForUserPeriod retrieves existing billing summary for a user and period
	GetBillingSummaryForUserPeriod(ctx context.Context, userID string, start, end time.Time) (*BillingSummary, error)

	// UpdateBillingSummary updates an existing billing summary
	UpdateBillingSummary(ctx context.Context, summary *BillingSummary) error
}
