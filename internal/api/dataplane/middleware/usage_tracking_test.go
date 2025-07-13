// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUsageRepository is a mock implementation of UsageRepository
type MockUsageRepository struct {
	mock.Mock
}

func (m *MockUsageRepository) CreateUsageEvent(ctx context.Context, event *llmgw.UsageEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockUsageRepository) GetUsageEvent(ctx context.Context, id string) (*llmgw.UsageEvent, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) ListUsageEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) ListUsageEventsForCostCalculation(ctx context.Context, limit int) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

func (m *MockUsageRepository) UpdateUsageEventCost(ctx context.Context, eventID string, cost llmgw.CostResult) error {
	args := m.Called(ctx, eventID, cost)
	return args.Error(0)
}

func (m *MockUsageRepository) ListUsageEventsByPeriod(ctx context.Context, userID string, start, end time.Time) ([]*llmgw.UsageEvent, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Get(0).([]*llmgw.UsageEvent), args.Error(1)
}

func TestUsageTrackingMiddleware_Track(t *testing.T) {
	// Create mock repository
	mockRepo := &MockUsageRepository{}
	logger := slog.Default()

	// Create middleware
	middleware := NewUsageTrackingMiddleware(mockRepo, logger, nil)
	defer middleware.Shutdown()

	// Set up expectation for CreateUsageEvent (without user context, userID will be empty)
	mockRepo.On("CreateUsageEvent", mock.Anything, mock.MatchedBy(func(event *llmgw.UsageEvent) bool {
		return event.Status == "success" &&
			event.UsageDataSource == "unavailable" &&
			event.DataComplete == false &&
			event.RequestID != "" &&
			event.UserID == "" && // No auth context, so userID should be empty
			event.DurationMs != nil
	})).Return(nil)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context values are set
		assert.NotNil(t, r.Context().Value(usageTrackingStartKey))
		assert.NotNil(t, r.Context().Value(usageTrackingRequestIDKey))
		assert.NotNil(t, r.Context().Value(usageTrackingUserIDKey))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Wrap handler with middleware
	wrappedHandler := middleware.Track(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", nil)
	recorder := httptest.NewRecorder()

	// Execute request (without auth context for simplicity)
	wrappedHandler.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "success", recorder.Body.String())

	// Give some time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}

func TestUsageTrackingMiddleware_CreateEventFromGAIResponse(t *testing.T) {
	// Create mock repository
	mockRepo := &MockUsageRepository{}
	logger := slog.Default()

	// Create middleware
	middleware := NewUsageTrackingMiddleware(mockRepo, logger, nil)
	defer middleware.Shutdown()

	// Set up expectation for CreateUsageEvent
	mockRepo.On("CreateUsageEvent", mock.Anything, mock.MatchedBy(func(event *llmgw.UsageEvent) bool {
		return event.Status == "success" &&
			event.UsageDataSource == "provider_response" &&
			event.DataComplete == true &&
			event.ModelID == "test-model" &&
			event.InputTokens != nil && *event.InputTokens == 100 &&
			event.OutputTokens != nil && *event.OutputTokens == 50
	})).Return(nil)

	// Create context with tracking data
	ctx := context.WithValue(context.Background(), usageTrackingRequestIDKey, "test-request-id")
	ctx = context.WithValue(ctx, usageTrackingUserIDKey, "test-user-id")

	// Create token usage
	usage := &interfaces.TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	// Call method
	middleware.CreateEventFromGAIResponse(ctx, "test-model", usage, "success", 2*time.Second)

	// Give some time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}

func TestUsageTrackingMiddleware_ErrorHandling(t *testing.T) {
	// Create mock repository
	mockRepo := &MockUsageRepository{}
	logger := slog.Default()

	// Create middleware
	middleware := NewUsageTrackingMiddleware(mockRepo, logger, nil)
	defer middleware.Shutdown()

	// Set up expectation for CreateUsageEvent with error status
	mockRepo.On("CreateUsageEvent", mock.Anything, mock.MatchedBy(func(event *llmgw.UsageEvent) bool {
		return event.Status == "failed" &&
			event.ErrorType != nil && *event.ErrorType == "auth_error" &&
			event.FailureStage != nil && *event.FailureStage == "pre_generation"
	})).Return(nil)

	// Create test handler that returns 401
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	})

	// Wrap handler with middleware
	wrappedHandler := middleware.Track(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", nil)
	recorder := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// Give some time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}
