// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUsageAnalytics_Minimal(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockUserRepo := new(MockUserRepository)
	mockUsageRepo := new(MockUsageRepository)
	mockBillingRepo := new(MockBillingRepository)

	tests := []struct {
		name    string
		options []UsageAnalyticsOption
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []UsageAnalyticsOption{},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []UsageAnalyticsOption{WithUsageAnalyticsLogger(discardLogger)},
			wantErr: false,
		},
		{
			name: "Create with all repositories",
			options: []UsageAnalyticsOption{
				WithUsageAnalyticsLogger(discardLogger),
				WithUsageAnalyticsUserRepository(mockUserRepo),
				WithUsageRepository(mockUsageRepo),
				WithBillingRepository(mockBillingRepo),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUsageAnalytics(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUsageAnalytics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NotNil(t, got)
			assert.NotNil(t, got.options.Logger)

			// Verify repositories are set if provided
			if len(tt.options) > 1 {
				assert.NotNil(t, got.options.UserRepository)
				assert.NotNil(t, got.options.UsageRepository)
				assert.NotNil(t, got.options.BillingRepository)
			}
		})
	}
}

func TestUsageAnalyticsOptions(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockUserRepo := new(MockUserRepository)
	mockUsageRepo := new(MockUsageRepository)
	mockBillingRepo := new(MockBillingRepository)

	usageAnalytics, err := NewUsageAnalytics(
		WithUsageAnalyticsLogger(discardLogger),
		WithUsageAnalyticsUserRepository(mockUserRepo),
		WithUsageRepository(mockUsageRepo),
		WithBillingRepository(mockBillingRepo),
	)

	assert.NoError(t, err)
	assert.Equal(t, discardLogger, usageAnalytics.options.Logger)
	assert.Equal(t, mockUserRepo, usageAnalytics.options.UserRepository)
	assert.Equal(t, mockUsageRepo, usageAnalytics.options.UsageRepository)
	assert.Equal(t, mockBillingRepo, usageAnalytics.options.BillingRepository)
}
