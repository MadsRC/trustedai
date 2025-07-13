// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateUsageEvent stores a new usage event
func (r *UsageRepository) CreateUsageEvent(ctx context.Context, event *llmgw.UsageEvent) error {
	query := `
		INSERT INTO usage_events (
			id, request_id, user_id, model_id,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			status, failure_stage, error_type, error_message,
			usage_data_source, data_complete, timestamp, duration_ms,
			input_cost_cents, output_cost_cents, total_cost_cents
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err := r.options.Db.Exec(ctx, query,
		event.ID,
		event.RequestID,
		event.UserID,
		event.ModelID,
		event.InputTokens,
		event.OutputTokens,
		event.CachedTokens,
		event.ReasoningTokens,
		event.Status,
		event.FailureStage,
		event.ErrorType,
		event.ErrorMessage,
		event.UsageDataSource,
		event.DataComplete,
		event.Timestamp,
		event.DurationMs,
		event.InputCostCents,
		event.OutputCostCents,
		event.TotalCostCents,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return llmgw.ErrDuplicateEntry
		}
		r.options.Logger.Error("Failed to create usage event", "error", err)
		return err
	}
	return nil
}

// GetUsageEvent retrieves a usage event by ID
func (r *UsageRepository) GetUsageEvent(ctx context.Context, id string) (*llmgw.UsageEvent, error) {
	query := `
		SELECT id, request_id, user_id, model_id,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			status, failure_stage, error_type, error_message,
			usage_data_source, data_complete, timestamp, duration_ms,
			input_cost_cents, output_cost_cents, total_cost_cents
		FROM usage_events
		WHERE id = $1`

	row := r.options.Db.QueryRow(ctx, query, id)

	var event llmgw.UsageEvent
	err := row.Scan(
		&event.ID,
		&event.RequestID,
		&event.UserID,
		&event.ModelID,
		&event.InputTokens,
		&event.OutputTokens,
		&event.CachedTokens,
		&event.ReasoningTokens,
		&event.Status,
		&event.FailureStage,
		&event.ErrorType,
		&event.ErrorMessage,
		&event.UsageDataSource,
		&event.DataComplete,
		&event.Timestamp,
		&event.DurationMs,
		&event.InputCostCents,
		&event.OutputCostCents,
		&event.TotalCostCents,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get usage event", "error", err, "id", id)
		return nil, err
	}
	return &event, nil
}

// ListUsageEventsByUser retrieves usage events for a specific user with pagination
func (r *UsageRepository) ListUsageEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*llmgw.UsageEvent, error) {
	query := `
		SELECT id, request_id, user_id, model_id,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			status, failure_stage, error_type, error_message,
			usage_data_source, data_complete, timestamp, duration_ms,
			input_cost_cents, output_cost_cents, total_cost_cents
		FROM usage_events
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.options.Db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.options.Logger.Error("Failed to list usage events by user", "error", err, "userID", userID)
		return nil, err
	}
	defer rows.Close()

	var events []*llmgw.UsageEvent
	for rows.Next() {
		var event llmgw.UsageEvent
		err := rows.Scan(
			&event.ID,
			&event.RequestID,
			&event.UserID,
			&event.ModelID,
			&event.InputTokens,
			&event.OutputTokens,
			&event.CachedTokens,
			&event.ReasoningTokens,
			&event.Status,
			&event.FailureStage,
			&event.ErrorType,
			&event.ErrorMessage,
			&event.UsageDataSource,
			&event.DataComplete,
			&event.Timestamp,
			&event.DurationMs,
			&event.InputCostCents,
			&event.OutputCostCents,
			&event.TotalCostCents,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan usage event row", "error", err)
			return nil, err
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating usage event rows", "error", err)
		return nil, err
	}

	return events, nil
}

// ListUsageEventsForCostCalculation retrieves uncalculated usage events that are ready for cost calculation
func (r *UsageRepository) ListUsageEventsForCostCalculation(ctx context.Context, limit int) ([]*llmgw.UsageEvent, error) {
	query := `
		SELECT id, request_id, user_id, model_id,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			status, failure_stage, error_type, error_message,
			usage_data_source, data_complete, timestamp, duration_ms,
			input_cost_cents, output_cost_cents, total_cost_cents
		FROM usage_events
		WHERE data_complete = true AND total_cost_cents IS NULL
		ORDER BY timestamp ASC
		LIMIT $1`

	rows, err := r.options.Db.Query(ctx, query, limit)
	if err != nil {
		r.options.Logger.Error("Failed to list usage events for cost calculation", "error", err)
		return nil, err
	}
	defer rows.Close()

	var events []*llmgw.UsageEvent
	for rows.Next() {
		var event llmgw.UsageEvent
		err := rows.Scan(
			&event.ID,
			&event.RequestID,
			&event.UserID,
			&event.ModelID,
			&event.InputTokens,
			&event.OutputTokens,
			&event.CachedTokens,
			&event.ReasoningTokens,
			&event.Status,
			&event.FailureStage,
			&event.ErrorType,
			&event.ErrorMessage,
			&event.UsageDataSource,
			&event.DataComplete,
			&event.Timestamp,
			&event.DurationMs,
			&event.InputCostCents,
			&event.OutputCostCents,
			&event.TotalCostCents,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan usage event row", "error", err)
			return nil, err
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating usage event rows", "error", err)
		return nil, err
	}

	return events, nil
}

// UpdateUsageEventCost updates the cost fields for a usage event
func (r *UsageRepository) UpdateUsageEventCost(ctx context.Context, eventID string, cost llmgw.CostResult) error {
	query := `
		UPDATE usage_events SET
			input_cost_cents = $2,
			output_cost_cents = $3,
			total_cost_cents = $4
		WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query,
		eventID,
		cost.InputCostCents,
		cost.OutputCostCents,
		cost.TotalCostCents,
	)

	if err != nil {
		r.options.Logger.Error("Failed to update usage event cost", "error", err, "eventID", eventID)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}

// ListUsageEventsByPeriod retrieves usage events for a specific period
func (r *UsageRepository) ListUsageEventsByPeriod(ctx context.Context, userID string, start, end time.Time) ([]*llmgw.UsageEvent, error) {
	query := `
		SELECT id, request_id, user_id, model_id,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			status, failure_stage, error_type, error_message,
			usage_data_source, data_complete, timestamp, duration_ms,
			input_cost_cents, output_cost_cents, total_cost_cents
		FROM usage_events
		WHERE user_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC`

	rows, err := r.options.Db.Query(ctx, query, userID, start, end)
	if err != nil {
		r.options.Logger.Error("Failed to list usage events by period", "error", err, "userID", userID)
		return nil, err
	}
	defer rows.Close()

	var events []*llmgw.UsageEvent
	for rows.Next() {
		var event llmgw.UsageEvent
		err := rows.Scan(
			&event.ID,
			&event.RequestID,
			&event.UserID,
			&event.ModelID,
			&event.InputTokens,
			&event.OutputTokens,
			&event.CachedTokens,
			&event.ReasoningTokens,
			&event.Status,
			&event.FailureStage,
			&event.ErrorType,
			&event.ErrorMessage,
			&event.UsageDataSource,
			&event.DataComplete,
			&event.Timestamp,
			&event.DurationMs,
			&event.InputCostCents,
			&event.OutputCostCents,
			&event.TotalCostCents,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan usage event row", "error", err)
			return nil, err
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating usage event rows", "error", err)
		return nil, err
	}

	return events, nil
}
