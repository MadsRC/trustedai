// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories for usage analytics testing
type MockUsageRepository struct {
	mock.Mock
}

func (m *MockUsageRepository) CreateUsageEvent(ctx context.Context, event *llmgw.UsageEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockUsageRepository) GetUsageEvent(ctx context.Context, id string) (*llmgw.UsageEvent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) ListUsageEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) ListUsageEventsForCostCalculation(ctx context.Context, limit int) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) UpdateUsageEventCost(ctx context.Context, eventID string, cost llmgw.CostResult) error {
	args := m.Called(ctx, eventID, cost)
	return args.Error(0)
}

func (m *MockUsageRepository) ListUsageEventsByPeriod(ctx context.Context, userID string, start, end time.Time) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

type MockBillingRepository struct {
	mock.Mock
}

func (m *MockBillingRepository) CreateBillingSummary(ctx context.Context, summary *llmgw.BillingSummary) error {
	args := m.Called(ctx, summary)
	return args.Error(0)
}

func (m *MockBillingRepository) GetBillingSummary(ctx context.Context, id string) (*llmgw.BillingSummary, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.BillingSummary), args.Error(1)
}

func (m *MockBillingRepository) ListBillingSummariesByUser(ctx context.Context, userID string, limit, offset int) ([]*llmgw.BillingSummary, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.BillingSummary), args.Error(1)
}

func (m *MockBillingRepository) ListBillingSummariesByPeriod(ctx context.Context, start, end time.Time) ([]*llmgw.BillingSummary, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.BillingSummary), args.Error(1)
}

func (m *MockBillingRepository) GetBillingSummaryForUserPeriod(ctx context.Context, userID string, start, end time.Time) (*llmgw.BillingSummary, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.BillingSummary), args.Error(1)
}

func (m *MockBillingRepository) UpdateBillingSummary(ctx context.Context, summary *llmgw.BillingSummary) error {
	args := m.Called(ctx, summary)
	return args.Error(0)
}

func TestNewUsageAnalytics(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockUserRepo := new(MockUserRepository)
	mockUsageRepo := new(MockUsageRepository)
	mockBillingRepo := new(MockBillingRepository)

	tests := []struct {
		name    string
		options []UsageAnalyticsOption
		want    *UsageAnalytics
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []UsageAnalyticsOption{},
			want: &UsageAnalytics{
				options: UsageAnalyticsOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []UsageAnalyticsOption{WithUsageAnalyticsLogger(discardLogger)},
			want: &UsageAnalytics{
				options: UsageAnalyticsOptions{
					Logger: discardLogger,
				},
			},
		},
		{
			name: "Create with all repositories",
			options: []UsageAnalyticsOption{
				WithUsageAnalyticsLogger(discardLogger),
				WithUsageAnalyticsUserRepository(mockUserRepo),
				WithUsageRepository(mockUsageRepo),
				WithBillingRepository(mockBillingRepo),
			},
			want: &UsageAnalytics{
				options: UsageAnalyticsOptions{
					Logger:            discardLogger,
					UserRepository:    mockUserRepo,
					UsageRepository:   mockUsageRepo,
					BillingRepository: mockBillingRepo,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUsageAnalytics(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUsageAnalytics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want.options.Logger, got.options.Logger)

			if tt.want.options.UserRepository != nil {
				assert.Equal(t, tt.want.options.UserRepository, got.options.UserRepository)
			}

			if tt.want.options.UsageRepository != nil {
				assert.Equal(t, tt.want.options.UsageRepository, got.options.UsageRepository)
			}

			if tt.want.options.BillingRepository != nil {
				assert.Equal(t, tt.want.options.BillingRepository, got.options.BillingRepository)
			}
		})
	}
}

func TestNewUsageAnalytics_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []UsageAnalyticsOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []UsageAnalyticsOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalUsageAnalyticsOptions = []UsageAnalyticsOption{
				WithUsageAnalyticsLogger(tt.inputLogger),
			}
			got1, _ := NewUsageAnalytics(tt.options...)
			got2, _ := NewUsageAnalytics(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewUsageAnalytics() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewUsageAnalytics() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewUsageAnalytics() = %v, want %v", got1, got2)
			}
			GlobalUsageAnalyticsOptions = []UsageAnalyticsOption{}
			got3, _ := NewUsageAnalytics(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewUsageAnalytics() = %v, want %v", got3, slog.Default())
			}
		})
	}
}

func TestWithUsageAnalyticsRepositories(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockUsageRepo := new(MockUsageRepository)
	mockBillingRepo := new(MockBillingRepository)

	usageAnalytics, _ := NewUsageAnalytics(
		WithUsageAnalyticsUserRepository(mockUserRepo),
		WithUsageRepository(mockUsageRepo),
		WithBillingRepository(mockBillingRepo),
	)

	assert.Equal(t, mockUserRepo, usageAnalytics.options.UserRepository)
	assert.Equal(t, mockUsageRepo, usageAnalytics.options.UsageRepository)
	assert.Equal(t, mockBillingRepo, usageAnalytics.options.BillingRepository)
}
