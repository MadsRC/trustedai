// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/gai-org/gai"
	"github.com/stretchr/testify/assert"
)

func TestCostCalculator_calculateCost(t *testing.T) {
	// Create a mock cost calculator
	calculator := &CostCalculator{}

	tests := []struct {
		name           string
		event          llmgw.UsageEvent
		pricing        gai.ModelPricing
		expectedResult llmgw.CostResult
	}{
		{
			name: "Basic cost calculation",
			event: llmgw.UsageEvent{
				InputTokens:  intPtr(1000),
				OutputTokens: intPtr(500),
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.001, // $0.001 per token
				OutputTokenPrice: 0.002, // $0.002 per token
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  100, // 1000 * 0.001 * 100
				OutputCostCents: 100, // 500 * 0.002 * 100
				TotalCostCents:  200, // 100 + 100
			},
		},
		{
			name: "Free model pricing",
			event: llmgw.UsageEvent{
				InputTokens:  intPtr(1000),
				OutputTokens: intPtr(500),
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.0,
				OutputTokenPrice: 0.0,
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  0,
				OutputCostCents: 0,
				TotalCostCents:  0,
			},
		},
		{
			name: "Missing input tokens",
			event: llmgw.UsageEvent{
				InputTokens:  nil,
				OutputTokens: intPtr(500),
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.001,
				OutputTokenPrice: 0.002,
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  0,   // No input tokens
				OutputCostCents: 100, // 500 * 0.002 * 100
				TotalCostCents:  100,
			},
		},
		{
			name: "Missing output tokens",
			event: llmgw.UsageEvent{
				InputTokens:  intPtr(1000),
				OutputTokens: nil,
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.001,
				OutputTokenPrice: 0.002,
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  100, // 1000 * 0.001 * 100
				OutputCostCents: 0,   // No output tokens
				TotalCostCents:  100,
			},
		},
		{
			name: "High precision pricing",
			event: llmgw.UsageEvent{
				InputTokens:  intPtr(1),
				OutputTokens: intPtr(1),
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.00001, // Very small price
				OutputTokenPrice: 0.00001,
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  0, // Rounds down to 0 cents
				OutputCostCents: 0, // Rounds down to 0 cents
				TotalCostCents:  0,
			},
		},
		{
			name: "Large token counts",
			event: llmgw.UsageEvent{
				InputTokens:  intPtr(100000),
				OutputTokens: intPtr(50000),
			},
			pricing: gai.ModelPricing{
				InputTokenPrice:  0.03,
				OutputTokenPrice: 0.06,
			},
			expectedResult: llmgw.CostResult{
				InputCostCents:  300000, // 100000 * 0.03 * 100
				OutputCostCents: 300000, // 50000 * 0.06 * 100
				TotalCostCents:  600000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.calculateCost(tt.event, tt.pricing)

			assert.Equal(t, tt.expectedResult.InputCostCents, result.InputCostCents, "Input cost mismatch")
			assert.Equal(t, tt.expectedResult.OutputCostCents, result.OutputCostCents, "Output cost mismatch")
			assert.Equal(t, tt.expectedResult.TotalCostCents, result.TotalCostCents, "Total cost mismatch")
		})
	}
}

func TestBillingPeriod_DailyPeriod(t *testing.T) {
	// Test daily period time range calculation
	date := mustParseTime("2025-01-15T10:30:00Z")
	period := DailyPeriod{Date: date}

	start, end := period.GetTimeRange()

	expectedStart := mustParseTime("2025-01-15T00:00:00Z")
	expectedEnd := mustParseTime("2025-01-15T23:59:59.999999999Z")

	assert.Equal(t, expectedStart, start, "Daily period start time mismatch")
	assert.Equal(t, expectedEnd, end, "Daily period end time mismatch")
	assert.Equal(t, "daily-2025-01-15", period.String(), "Daily period string representation mismatch")
}

func TestBillingPeriod_MonthlyPeriod(t *testing.T) {
	// Test monthly period time range calculation
	period := MonthlyPeriod{Year: 2025, Month: 1}

	start, end := period.GetTimeRange()

	expectedStart := mustParseTime("2025-01-01T00:00:00Z")
	expectedEnd := mustParseTime("2025-01-31T23:59:59.999999999Z")

	assert.Equal(t, expectedStart, start, "Monthly period start time mismatch")
	assert.Equal(t, expectedEnd, end, "Monthly period end time mismatch")
	assert.Equal(t, "monthly-2025-01", period.String(), "Monthly period string representation mismatch")
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func mustParseTime(layout string) time.Time {
	t, err := time.Parse(time.RFC3339, layout)
	if err != nil {
		panic(err)
	}
	return t
}
