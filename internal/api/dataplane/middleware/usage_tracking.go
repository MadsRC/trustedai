// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/auth"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/interfaces"
	"github.com/google/uuid"
)

// Context keys for usage tracking data
type contextKey string

const (
	usageTrackingStartKey     contextKey = "usage_tracking_start"
	usageTrackingRequestIDKey contextKey = "usage_tracking_request_id"
	usageTrackingUserIDKey    contextKey = "usage_tracking_user_id"
)

// UsageTrackingMiddleware captures usage events for all LLM requests
type UsageTrackingMiddleware struct {
	usageRepo llmgw.UsageRepository
	logger    *slog.Logger
	eventsCh  chan *llmgw.UsageEvent
	done      chan struct{}
}

// NewUsageTrackingMiddleware creates a new usage tracking middleware
func NewUsageTrackingMiddleware(usageRepo llmgw.UsageRepository, logger *slog.Logger) *UsageTrackingMiddleware {
	middleware := &UsageTrackingMiddleware{
		usageRepo: usageRepo,
		logger:    logger,
		eventsCh:  make(chan *llmgw.UsageEvent, 1000), // Buffered channel for non-blocking operation
		done:      make(chan struct{}),
	}

	// Start background worker to process events
	go middleware.processEvents()

	return middleware
}

// Track wraps HTTP handlers to capture usage events
func (m *UsageTrackingMiddleware) Track(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start timing
		startTime := time.Now()

		// Create a custom response writer to capture response data
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Extract user ID from context
		userID := ""
		if user := auth.UserFromHTTPContext(r); user != nil {
			userID = user.ID
		}

		// Generate request ID if not present
		requestID := uuid.New().String()

		// Store tracking context in request
		ctx := context.WithValue(r.Context(), usageTrackingStartKey, startTime)
		ctx = context.WithValue(ctx, usageTrackingRequestIDKey, requestID)
		ctx = context.WithValue(ctx, usageTrackingUserIDKey, userID)

		// Call next handler
		next.ServeHTTP(recorder, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(startTime)

		// Determine status based on HTTP response
		status := "success"
		if recorder.statusCode >= 400 {
			status = "failed"
		}

		// Create basic usage event (without token data)
		event := &llmgw.UsageEvent{
			ID:              uuid.New().String(),
			RequestID:       requestID,
			UserID:          userID,
			ModelID:         "", // Will be set by provider-specific logic if available
			Status:          status,
			UsageDataSource: "unavailable", // Default, can be updated by providers
			DataComplete:    false,         // Default, can be updated by providers
			Timestamp:       time.Now(),
			DurationMs:      intPtr(int(duration.Milliseconds())),
		}

		// Set error information for failed requests
		if status == "failed" {
			errorType := "http_error"
			failureStage := "pre_generation"
			if recorder.statusCode >= 500 {
				errorType = "server_error"
				failureStage = "during_generation"
			} else if recorder.statusCode == 401 || recorder.statusCode == 403 {
				errorType = "auth_error"
				failureStage = "pre_generation"
			} else if recorder.statusCode == 429 {
				errorType = "rate_limit"
				failureStage = "pre_generation"
			}

			event.ErrorType = &errorType
			event.FailureStage = &failureStage
		}

		// Send event to background processor (non-blocking)
		select {
		case m.eventsCh <- event:
			// Event queued successfully
		default:
			// Channel is full, drop event to prevent blocking
			m.logger.Warn("Usage tracking channel full, dropping event", "request_id", requestID)
		}
	})
}

// UpdateEvent allows providers to update usage events with token data
func (m *UsageTrackingMiddleware) UpdateEvent(ctx context.Context, requestID string, update func(*llmgw.UsageEvent)) {
	// This is a placeholder for provider integration
	// In a full implementation, you might want to:
	// 1. Store pending events in a map with requestID as key
	// 2. Allow providers to update events before they're persisted
	// 3. Or use a callback mechanism

	// For now, providers can create their own events with complete data
	m.logger.Debug("Event update requested", "request_id", requestID)
}

// CreateEventFromGAIResponse creates a usage event from GAI response data
func (m *UsageTrackingMiddleware) CreateEventFromGAIResponse(ctx context.Context, modelID string, usage *interfaces.TokenUsage, status string, duration time.Duration) {
	// Extract tracking context
	requestID, _ := ctx.Value(usageTrackingRequestIDKey).(string)
	userID, _ := ctx.Value(usageTrackingUserIDKey).(string)

	if requestID == "" {
		requestID = uuid.New().String()
	}

	event := &llmgw.UsageEvent{
		ID:              uuid.New().String(),
		RequestID:       requestID,
		UserID:          userID,
		ModelID:         modelID,
		Status:          status,
		UsageDataSource: "provider_response",
		DataComplete:    usage != nil,
		Timestamp:       time.Now(),
		DurationMs:      intPtr(int(duration.Milliseconds())),
	}

	// Set token usage if available
	if usage != nil {
		event.InputTokens = &usage.PromptTokens
		event.OutputTokens = &usage.CompletionTokens
	}

	// Send event to background processor
	select {
	case m.eventsCh <- event:
		// Event queued successfully
	default:
		// Channel is full, drop event
		m.logger.Warn("Usage tracking channel full, dropping GAI event", "request_id", requestID)
	}
}

// processEvents runs in a background goroutine to persist usage events
func (m *UsageTrackingMiddleware) processEvents() {
	for {
		select {
		case event := <-m.eventsCh:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			if err := m.usageRepo.CreateUsageEvent(ctx, event); err != nil {
				m.logger.Error("Failed to persist usage event",
					"error", err,
					"request_id", event.RequestID,
					"user_id", event.UserID)
			} else {
				m.logger.Debug("Usage event persisted",
					"request_id", event.RequestID,
					"user_id", event.UserID,
					"model_id", event.ModelID,
					"status", event.Status)
			}

			cancel()

		case <-m.done:
			// Shutdown signal received
			m.logger.Info("Usage tracking middleware shutting down")

			// Process remaining events
			for {
				select {
				case event := <-m.eventsCh:
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					if err := m.usageRepo.CreateUsageEvent(ctx, event); err != nil {
						m.logger.Error("Failed to persist final usage event", "error", err)
					}
					cancel()
				default:
					return
				}
			}
		}
	}
}

// Shutdown gracefully stops the middleware
func (m *UsageTrackingMiddleware) Shutdown() {
	close(m.done)
}

// responseRecorder captures HTTP response data
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
