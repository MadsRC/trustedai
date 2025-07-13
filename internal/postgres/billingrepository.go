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

// CreateBillingSummary stores a new billing summary
func (r *BillingRepository) CreateBillingSummary(ctx context.Context, summary *llmgw.BillingSummary) error {
	query := `
		INSERT INTO billing_summaries (
			id, user_id, period_start, period_end,
			total_requests, total_input_tokens, total_output_tokens,
			total_cost_cents, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.options.Db.Exec(ctx, query,
		summary.ID,
		summary.UserID,
		summary.PeriodStart,
		summary.PeriodEnd,
		summary.TotalRequests,
		summary.TotalInputTokens,
		summary.TotalOutputTokens,
		summary.TotalCostCents,
		summary.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return llmgw.ErrDuplicateEntry
		}
		r.options.Logger.Error("Failed to create billing summary", "error", err)
		return err
	}
	return nil
}

// GetBillingSummary retrieves a billing summary by ID
func (r *BillingRepository) GetBillingSummary(ctx context.Context, id string) (*llmgw.BillingSummary, error) {
	query := `
		SELECT id, user_id, period_start, period_end,
			total_requests, total_input_tokens, total_output_tokens,
			total_cost_cents, created_at
		FROM billing_summaries
		WHERE id = $1`

	row := r.options.Db.QueryRow(ctx, query, id)

	var summary llmgw.BillingSummary
	err := row.Scan(
		&summary.ID,
		&summary.UserID,
		&summary.PeriodStart,
		&summary.PeriodEnd,
		&summary.TotalRequests,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
		&summary.TotalCostCents,
		&summary.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get billing summary", "error", err, "id", id)
		return nil, err
	}
	return &summary, nil
}

// ListBillingSummariesByUser retrieves billing summaries for a specific user
func (r *BillingRepository) ListBillingSummariesByUser(ctx context.Context, userID string, limit, offset int) ([]*llmgw.BillingSummary, error) {
	query := `
		SELECT id, user_id, period_start, period_end,
			total_requests, total_input_tokens, total_output_tokens,
			total_cost_cents, created_at
		FROM billing_summaries
		WHERE user_id = $1
		ORDER BY period_start DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.options.Db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.options.Logger.Error("Failed to list billing summaries by user", "error", err, "userID", userID)
		return nil, err
	}
	defer rows.Close()

	var summaries []*llmgw.BillingSummary
	for rows.Next() {
		var summary llmgw.BillingSummary
		err := rows.Scan(
			&summary.ID,
			&summary.UserID,
			&summary.PeriodStart,
			&summary.PeriodEnd,
			&summary.TotalRequests,
			&summary.TotalInputTokens,
			&summary.TotalOutputTokens,
			&summary.TotalCostCents,
			&summary.CreatedAt,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan billing summary row", "error", err)
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating billing summary rows", "error", err)
		return nil, err
	}

	return summaries, nil
}

// ListBillingSummariesByPeriod retrieves billing summaries for a specific period
func (r *BillingRepository) ListBillingSummariesByPeriod(ctx context.Context, start, end time.Time) ([]*llmgw.BillingSummary, error) {
	query := `
		SELECT id, user_id, period_start, period_end,
			total_requests, total_input_tokens, total_output_tokens,
			total_cost_cents, created_at
		FROM billing_summaries
		WHERE period_start >= $1 AND period_end <= $2
		ORDER BY period_start DESC`

	rows, err := r.options.Db.Query(ctx, query, start, end)
	if err != nil {
		r.options.Logger.Error("Failed to list billing summaries by period", "error", err)
		return nil, err
	}
	defer rows.Close()

	var summaries []*llmgw.BillingSummary
	for rows.Next() {
		var summary llmgw.BillingSummary
		err := rows.Scan(
			&summary.ID,
			&summary.UserID,
			&summary.PeriodStart,
			&summary.PeriodEnd,
			&summary.TotalRequests,
			&summary.TotalInputTokens,
			&summary.TotalOutputTokens,
			&summary.TotalCostCents,
			&summary.CreatedAt,
		)
		if err != nil {
			r.options.Logger.Error("Failed to scan billing summary row", "error", err)
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		r.options.Logger.Error("Error iterating billing summary rows", "error", err)
		return nil, err
	}

	return summaries, nil
}

// GetBillingSummaryForUserPeriod retrieves existing billing summary for a user and period
func (r *BillingRepository) GetBillingSummaryForUserPeriod(ctx context.Context, userID string, start, end time.Time) (*llmgw.BillingSummary, error) {
	query := `
		SELECT id, user_id, period_start, period_end,
			total_requests, total_input_tokens, total_output_tokens,
			total_cost_cents, created_at
		FROM billing_summaries
		WHERE user_id = $1 AND period_start = $2 AND period_end = $3`

	row := r.options.Db.QueryRow(ctx, query, userID, start, end)

	var summary llmgw.BillingSummary
	err := row.Scan(
		&summary.ID,
		&summary.UserID,
		&summary.PeriodStart,
		&summary.PeriodEnd,
		&summary.TotalRequests,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
		&summary.TotalCostCents,
		&summary.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, llmgw.ErrNotFound
	}
	if err != nil {
		r.options.Logger.Error("Failed to get billing summary for user period", "error", err, "userID", userID)
		return nil, err
	}
	return &summary, nil
}

// UpdateBillingSummary updates an existing billing summary
func (r *BillingRepository) UpdateBillingSummary(ctx context.Context, summary *llmgw.BillingSummary) error {
	query := `
		UPDATE billing_summaries SET
			total_requests = $2,
			total_input_tokens = $3,
			total_output_tokens = $4,
			total_cost_cents = $5
		WHERE id = $1`

	result, err := r.options.Db.Exec(ctx, query,
		summary.ID,
		summary.TotalRequests,
		summary.TotalInputTokens,
		summary.TotalOutputTokens,
		summary.TotalCostCents,
	)

	if err != nil {
		r.options.Logger.Error("Failed to update billing summary", "error", err, "id", summary.ID)
		return err
	}

	if result.RowsAffected() == 0 {
		return llmgw.ErrNotFound
	}

	return nil
}
