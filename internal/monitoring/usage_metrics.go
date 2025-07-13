// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type UsageMetrics struct {
	eventsCapturedTotal     metric.Int64Counter
	eventsDroppedTotal      metric.Int64Counter
	batchWritesTotal        metric.Int64Counter
	processingLatency       metric.Float64Histogram
	costCalculationDuration metric.Float64Histogram
	tokenUsageByModel       metric.Int64Counter
	costPerUserCents        metric.Int64Gauge
	requestsPerOrg          metric.Int64Counter
	modelPopularity         metric.Int64Counter
	channelSize             metric.Int64Gauge
	workerQueueSize         metric.Int64Gauge
	dbWriteErrorsTotal      metric.Int64Counter
}

func NewUsageMetrics(meter metric.Meter) (*UsageMetrics, error) {
	eventsCapturedTotal, err := meter.Int64Counter(
		"usage_events_captured_total",
		metric.WithDescription("Total usage events captured"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events_captured_total counter: %w", err)
	}

	eventsDroppedTotal, err := meter.Int64Counter(
		"usage_events_dropped_total",
		metric.WithDescription("Events dropped due to channel full"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events_dropped_total counter: %w", err)
	}

	batchWritesTotal, err := meter.Int64Counter(
		"usage_batch_writes_total",
		metric.WithDescription("Database batch write operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch_writes_total counter: %w", err)
	}

	processingLatency, err := meter.Float64Histogram(
		"usage_processing_latency_seconds",
		metric.WithDescription("Time from capture to database write"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing_latency histogram: %w", err)
	}

	costCalculationDuration, err := meter.Float64Histogram(
		"cost_calculation_duration_seconds",
		metric.WithDescription("Time to process cost calculations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost_calculation_duration histogram: %w", err)
	}

	tokenUsageByModel, err := meter.Int64Counter(
		"token_usage_by_model",
		metric.WithDescription("Token usage breakdown by model"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create token_usage_by_model counter: %w", err)
	}

	costPerUserCents, err := meter.Int64Gauge(
		"cost_per_user_cents",
		metric.WithDescription("Cost tracking by user"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost_per_user_cents gauge: %w", err)
	}

	requestsPerOrg, err := meter.Int64Counter(
		"requests_per_organization",
		metric.WithDescription("Request volume by organization"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requests_per_organization counter: %w", err)
	}

	modelPopularity, err := meter.Int64Counter(
		"model_popularity",
		metric.WithDescription("Usage distribution across models"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model_popularity counter: %w", err)
	}

	channelSize, err := meter.Int64Gauge(
		"usage_channel_size",
		metric.WithDescription("Current buffered events in channel"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage_channel_size gauge: %w", err)
	}

	workerQueueSize, err := meter.Int64Gauge(
		"usage_worker_queue_size",
		metric.WithDescription("Background processing queue depth"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage_worker_queue_size gauge: %w", err)
	}

	dbWriteErrorsTotal, err := meter.Int64Counter(
		"database_write_errors_total",
		metric.WithDescription("Failed database operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create database_write_errors_total counter: %w", err)
	}

	return &UsageMetrics{
		eventsCapturedTotal:     eventsCapturedTotal,
		eventsDroppedTotal:      eventsDroppedTotal,
		batchWritesTotal:        batchWritesTotal,
		processingLatency:       processingLatency,
		costCalculationDuration: costCalculationDuration,
		tokenUsageByModel:       tokenUsageByModel,
		costPerUserCents:        costPerUserCents,
		requestsPerOrg:          requestsPerOrg,
		modelPopularity:         modelPopularity,
		channelSize:             channelSize,
		workerQueueSize:         workerQueueSize,
		dbWriteErrorsTotal:      dbWriteErrorsTotal,
	}, nil
}

func (um *UsageMetrics) RecordEventCaptured(ctx context.Context, options ...metric.AddOption) {
	um.eventsCapturedTotal.Add(ctx, 1, options...)
}

func (um *UsageMetrics) RecordEventDropped(ctx context.Context, options ...metric.AddOption) {
	um.eventsDroppedTotal.Add(ctx, 1, options...)
}

func (um *UsageMetrics) RecordBatchWrite(ctx context.Context, count int64, options ...metric.AddOption) {
	um.batchWritesTotal.Add(ctx, count, options...)
}

func (um *UsageMetrics) RecordProcessingLatency(ctx context.Context, duration time.Duration, options ...metric.RecordOption) {
	um.processingLatency.Record(ctx, duration.Seconds(), options...)
}

func (um *UsageMetrics) RecordCostCalculationDuration(ctx context.Context, duration time.Duration, options ...metric.RecordOption) {
	um.costCalculationDuration.Record(ctx, duration.Seconds(), options...)
}

func (um *UsageMetrics) RecordTokenUsage(ctx context.Context, tokens int64, model string, tokenType string, userID string) {
	um.tokenUsageByModel.Add(ctx, tokens,
		metric.WithAttributes(
			attribute.String("model", model),
			attribute.String("token_type", tokenType),
			attribute.String("user_id", userID),
		),
	)
}

func (um *UsageMetrics) UpdateCostPerUser(ctx context.Context, userID string, costCents int64) {
	um.costPerUserCents.Record(ctx, costCents,
		metric.WithAttributes(
			attribute.String("user_id", userID),
		),
	)
}

func (um *UsageMetrics) RecordRequest(ctx context.Context, orgID string, userID string) {
	um.requestsPerOrg.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("organization_id", orgID),
			attribute.String("user_id", userID),
		),
	)
}

func (um *UsageMetrics) RecordModelUsage(ctx context.Context, model string, userID string) {
	um.modelPopularity.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("model", model),
			attribute.String("user_id", userID),
		),
	)
}

func (um *UsageMetrics) UpdateChannelSize(ctx context.Context, size int64) {
	um.channelSize.Record(ctx, size)
}

func (um *UsageMetrics) UpdateWorkerQueueSize(ctx context.Context, size int64) {
	um.workerQueueSize.Record(ctx, size)
}

func (um *UsageMetrics) RecordDatabaseError(ctx context.Context, operation string, errorType string) {
	um.dbWriteErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("error_type", errorType),
		),
	)
}
