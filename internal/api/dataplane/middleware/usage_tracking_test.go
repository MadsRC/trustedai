// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

	// With the new pending event system, successful requests don't immediately persist
	// They are stored as pending events. We'll simulate a provider update to trigger persistence.
	// No immediate CreateUsageEvent call is expected from Track for successful requests.

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

	// With pending events system, no immediate persistence occurs for successful requests
	// The event is stored in memory waiting for provider update
}

func TestUsageTrackingMiddleware_UpdateEvent(t *testing.T) {
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

	// First, we need to create a pending event, then update it
	// The issue is timing: the pending event is created AFTER the handler completes
	// Let's call UpdateEvent directly with proper context

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler with middleware to create pending event
	wrappedHandler := middleware.Track(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	recorder := httptest.NewRecorder()

	// Execute request - this creates pending event
	wrappedHandler.ServeHTTP(recorder, req)

	// Now get the request context from the request and call UpdateEvent
	// We need to extract the request ID from the middleware's context handling
	// Let's create a proper context with the tracking values

	// Create context with tracking data (simulating what the middleware sets)
	var capturedRequestID string
	capturedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value(usageTrackingRequestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	wrappedCaptureHandler := middleware.Track(capturedHandler)
	req2 := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	recorder2 := httptest.NewRecorder()
	wrappedCaptureHandler.ServeHTTP(recorder2, req2)

	// Now create context for the GAI response call
	ctx := context.WithValue(context.Background(), usageTrackingRequestIDKey, capturedRequestID)
	ctx = context.WithValue(ctx, usageTrackingUserIDKey, "")

	// Create token usage
	usage := &interfaces.TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	// Call UpdateEvent with the captured context
	middleware.UpdateEvent(ctx, "test-model", usage, "success", 2*time.Second)

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

func TestUsageTrackingMiddleware_UpdateEventRaceCondition(t *testing.T) {
	mockRepo := &MockUsageRepository{}
	logger := slog.Default()
	middleware := NewUsageTrackingMiddleware(mockRepo, logger, nil)

	// Set up expectation that the event should be successfully persisted
	// This verifies the race condition is FIXED - the event gets updated and persisted
	var persistedEvent *llmgw.UsageEvent
	mockRepo.On("CreateUsageEvent", mock.Anything, mock.MatchedBy(func(event *llmgw.UsageEvent) bool {
		// Verify the event has the expected data from UpdateEvent
		return event.Status == "success" &&
			event.ModelID == "test-model" &&
			event.UsageDataSource == "provider_response" &&
			event.DataComplete == true &&
			event.InputTokens != nil && *event.InputTokens == 10 &&
			event.OutputTokens != nil && *event.OutputTokens == 20
	})).Run(func(args mock.Arguments) {
		persistedEvent = args.Get(1).(*llmgw.UsageEvent)
	}).Return(nil).Once()

	// Track if UpdateEvent was called
	var updateEventCalled bool

	// Create a handler that simulates a provider calling UpdateEvent during request processing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate what a provider does - call UpdateEvent during request processing
		usage := &interfaces.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		}

		// This should successfully find and update the pending event (race condition fixed)
		middleware.UpdateEvent(r.Context(), "test-model", usage, "success", 100*time.Millisecond)
		updateEventCalled = true

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the handler with the middleware
	wrappedHandler := middleware.Track(testHandler)

	// Create a test request
	req := httptest.NewRequest("POST", "/test", nil)
	recorder := httptest.NewRecorder()

	// Execute the request
	wrappedHandler.ServeHTTP(recorder, req)

	// Give some time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify the actual behavior - not logs
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, updateEventCalled, "UpdateEvent should have been called")

	// The critical test: verify the event was actually persisted to the database
	// This proves the race condition is FIXED because:
	// 1. Pending event was stored before handler
	// 2. UpdateEvent found the pending event during handler execution
	// 3. Event was successfully updated and persisted
	mockRepo.AssertExpectations(t)

	// Additional verification that the persisted event has correct data
	assert.NotNil(t, persistedEvent, "Event should have been persisted")
	if persistedEvent != nil {
		assert.Equal(t, "success", persistedEvent.Status)
		assert.Equal(t, "test-model", persistedEvent.ModelID)
		assert.Equal(t, "provider_response", persistedEvent.UsageDataSource)
	}

	// Clean up
	middleware.Shutdown()
}
