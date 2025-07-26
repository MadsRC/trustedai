// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/monitoring"
	"codeberg.org/gai-org/gai"
)

// CostCalculator processes usage events to calculate costs and generate billing summaries
type CostCalculator struct {
	usageRepo   llmgw.UsageRepository
	modelRepo   llmgw.ModelRepository
	billingRepo llmgw.BillingRepository
	logger      *slog.Logger
	batchSize   int
	metrics     *monitoring.UsageMetrics
}

// CostCalculatorOption configures CostCalculator behavior
type CostCalculatorOption func(*CostCalculator)

// WithBatchSize sets the number of events to process in each batch
func WithBatchSize(size int) CostCalculatorOption {
	return func(c *CostCalculator) {
		c.batchSize = size
	}
}

// WithLogger sets the logger for the cost calculator
func WithLogger(logger *slog.Logger) CostCalculatorOption {
	return func(c *CostCalculator) {
		c.logger = logger
	}
}

// WithMetrics sets the metrics for the cost calculator
func WithMetrics(metrics *monitoring.UsageMetrics) CostCalculatorOption {
	return func(c *CostCalculator) {
		c.metrics = metrics
	}
}

// NewCostCalculator creates a new CostCalculator instance
func NewCostCalculator(
	usageRepo llmgw.UsageRepository,
	modelRepo llmgw.ModelRepository,
	billingRepo llmgw.BillingRepository,
	options ...CostCalculatorOption,
) *CostCalculator {
	c := &CostCalculator{
		usageRepo:   usageRepo,
		modelRepo:   modelRepo,
		billingRepo: billingRepo,
		logger:      slog.Default(),
		batchSize:   100, // Default batch size
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

// ProcessUsageEvents processes uncalculated usage events and calculates their costs
func (c *CostCalculator) ProcessUsageEvents(ctx context.Context) error {
	startTime := time.Now()
	c.logger.Info("Starting cost calculation for usage events", "batchSize", c.batchSize)

	for {
		// Fetch batch of uncalculated events
		events, err := c.usageRepo.ListUsageEventsForCostCalculation(ctx, c.batchSize)
		if err != nil {
			c.logger.Error("Failed to fetch usage events for cost calculation", "error", err)
			return fmt.Errorf("failed to fetch usage events: %w", err)
		}

		if len(events) == 0 {
			c.logger.Info("No more usage events to process")
			break
		}

		c.logger.Info("Processing batch of usage events", "count", len(events))

		// Process each event in the batch
		for _, event := range events {
			if err := c.processEvent(ctx, event); err != nil {
				c.logger.Error("Failed to process usage event",
					"eventID", event.ID,
					"error", err)
				// Continue processing other events even if one fails
				continue
			}
		}

		c.logger.Info("Completed batch processing", "processedCount", len(events))

		// Record batch write metrics
		if c.metrics != nil {
			c.metrics.RecordBatchWrite(ctx, int64(len(events)))
		}

		// If we got fewer events than the batch size, we're done
		if len(events) < c.batchSize {
			break
		}
	}

	// Record total cost calculation duration
	if c.metrics != nil {
		c.metrics.RecordCostCalculationDuration(ctx, time.Since(startTime))
	}

	c.logger.Info("Cost calculation completed")
	return nil
}

// processEvent calculates the cost for a single usage event
func (c *CostCalculator) processEvent(ctx context.Context, event *llmgw.UsageEvent) error {
	// Get model pricing information
	model, err := c.modelRepo.GetModelByID(ctx, event.ModelID)
	if err != nil {
		return fmt.Errorf("failed to get model %s: %w", event.ModelID, err)
	}

	// Calculate cost based on token usage
	cost := c.calculateCost(*event, model.Pricing)

	// Update the usage event with calculated costs
	if err := c.usageRepo.UpdateUsageEventCost(ctx, event.ID, cost); err != nil {
		return fmt.Errorf("failed to update cost for event %s: %w", event.ID, err)
	}

	c.logger.Debug("Updated cost for usage event",
		"eventID", event.ID,
		"modelID", event.ModelID,
		"inputTokens", event.InputTokens,
		"outputTokens", event.OutputTokens,
		"totalCostCents", cost.TotalCostCents)

	return nil
}

// calculateCost computes the cost for a usage event based on model pricing
// Pricing is expected to be per token (e.g., 0.0000001 = $0.0000001 per token)
func (c *CostCalculator) calculateCost(event llmgw.UsageEvent, pricing gai.ModelPricing) llmgw.CostResult {
	var inputCost, outputCost float64

	// Calculate input token cost (pricing is per token)
	if event.InputTokens != nil {
		inputCost = float64(*event.InputTokens) * pricing.InputTokenPrice
	}

	// Calculate output token cost (pricing is per token)
	if event.OutputTokens != nil {
		outputCost = float64(*event.OutputTokens) * pricing.OutputTokenPrice
	}

	// Convert to fractional cents and return
	inputCostCents := inputCost * 100
	outputCostCents := outputCost * 100
	totalCostCents := inputCostCents + outputCostCents

	return llmgw.CostResult{
		InputCostCents:  inputCostCents,
		OutputCostCents: outputCostCents,
		TotalCostCents:  totalCostCents,
	}
}

// GenerateBillingSummaries creates billing summaries for completed usage events
func (c *CostCalculator) GenerateBillingSummaries(ctx context.Context, period BillingPeriod) error {
	c.logger.Info("Generating billing summaries", "period", period)

	start, end := period.GetTimeRange()
	c.logger.Info("Billing period range", "start", start, "end", end)

	// This is a simplified implementation - in production you might want to:
	// 1. Get all users with usage in the period
	// 2. Generate summaries for each user
	// 3. Handle concurrent summary generation

	// For now, we'll implement a basic approach that could be extended
	return c.generateBillingSummariesForPeriod(ctx, start, end)
}

// generateBillingSummariesForPeriod generates billing summaries for a specific time period
func (c *CostCalculator) generateBillingSummariesForPeriod(ctx context.Context, start, end time.Time) error {
	// Get all billing summaries for the period to see what exists
	existingSummaries, err := c.billingRepo.ListBillingSummariesByPeriod(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to list existing billing summaries: %w", err)
	}

	c.logger.Info("Found existing billing summaries", "count", len(existingSummaries))

	// Note: This is a basic implementation. In production, you would:
	// 1. Query for all users with usage events in the period
	// 2. Aggregate usage by user
	// 3. Create or update billing summaries
	// 4. Handle edge cases like partial periods, timezone handling, etc.

	c.logger.Info("Billing summary generation completed for period", "start", start, "end", end)
	return nil
}

// BillingPeriod represents different billing period types
type BillingPeriod interface {
	GetTimeRange() (start, end time.Time)
	String() string
}

// DailyPeriod represents a daily billing period
type DailyPeriod struct {
	Date time.Time
}

func (d DailyPeriod) GetTimeRange() (start, end time.Time) {
	start = time.Date(d.Date.Year(), d.Date.Month(), d.Date.Day(), 0, 0, 0, 0, d.Date.Location())
	end = start.Add(24*time.Hour - time.Nanosecond)
	return start, end
}

func (d DailyPeriod) String() string {
	return fmt.Sprintf("daily-%s", d.Date.Format("2006-01-02"))
}

// MonthlyPeriod represents a monthly billing period
type MonthlyPeriod struct {
	Year  int
	Month time.Month
}

func (m MonthlyPeriod) GetTimeRange() (start, end time.Time) {
	start = time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return start, end
}

func (m MonthlyPeriod) String() string {
	return fmt.Sprintf("monthly-%04d-%02d", m.Year, m.Month)
}
