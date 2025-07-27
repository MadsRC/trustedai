// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/MadsRC/trustedai"
	"github.com/MadsRC/trustedai/internal/api/dataplane/auth"
	"github.com/MadsRC/trustedai/internal/api/dataplane/interfaces"
	"github.com/MadsRC/trustedai/internal/monitoring"
	"github.com/google/uuid"
)

// Context keys for usage tracking data
type contextKey string

const (
	usageTrackingStartKey     contextKey = "usage_tracking_start"
	usageTrackingRequestIDKey contextKey = "usage_tracking_request_id"
	usageTrackingUserIDKey    contextKey = "usage_tracking_user_id"
)

// PendingEvent represents an event waiting for provider data
type PendingEvent struct {
	Event     *trustedai.UsageEvent
	CreatedAt time.Time
	Updated   bool
}

// UsageTrackingConfig holds configuration for the middleware
type UsageTrackingConfig struct {
	PendingEventTimeout time.Duration
	CleanupInterval     time.Duration
	MaxPendingEvents    int
	ChannelBuffer       int
}

// UsageTrackingOption is a functional option for the middleware
type UsageTrackingOption func(*UsageTrackingConfig)

// WithPendingEventTimeout sets the timeout for pending events
func WithPendingEventTimeout(timeout time.Duration) UsageTrackingOption {
	return func(c *UsageTrackingConfig) {
		c.PendingEventTimeout = timeout
	}
}

// WithCleanupInterval sets the cleanup interval for expired events
func WithCleanupInterval(interval time.Duration) UsageTrackingOption {
	return func(c *UsageTrackingConfig) {
		c.CleanupInterval = interval
	}
}

// WithMaxPendingEvents sets the maximum number of pending events
func WithMaxPendingEvents(max int) UsageTrackingOption {
	return func(c *UsageTrackingConfig) {
		c.MaxPendingEvents = max
	}
}

// WithChannelBuffer sets the buffer size for the events channel
func WithChannelBuffer(buffer int) UsageTrackingOption {
	return func(c *UsageTrackingConfig) {
		c.ChannelBuffer = buffer
	}
}

// UsageTrackingMiddleware captures usage events for all LLM requests
type UsageTrackingMiddleware struct {
	usageRepo     trustedai.UsageRepository
	logger        *slog.Logger
	eventsCh      chan *trustedai.UsageEvent
	done          chan struct{}
	metrics       *monitoring.UsageMetrics
	config        *UsageTrackingConfig
	pendingEvents sync.Map // map[string]*PendingEvent
	cleanupTicker *time.Ticker
}

// NewUsageTrackingMiddleware creates a new usage tracking middleware
func NewUsageTrackingMiddleware(usageRepo trustedai.UsageRepository, logger *slog.Logger, metrics *monitoring.UsageMetrics, opts ...UsageTrackingOption) *UsageTrackingMiddleware {
	// Default configuration
	config := &UsageTrackingConfig{
		PendingEventTimeout: 5 * time.Minute,
		CleanupInterval:     1 * time.Minute,
		MaxPendingEvents:    10000,
		ChannelBuffer:       1000,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	middleware := &UsageTrackingMiddleware{
		usageRepo:     usageRepo,
		logger:        logger,
		eventsCh:      make(chan *trustedai.UsageEvent, config.ChannelBuffer),
		done:          make(chan struct{}),
		metrics:       metrics,
		config:        config,
		cleanupTicker: time.NewTicker(config.CleanupInterval),
	}

	// Start background workers
	go middleware.processEvents()
	go middleware.cleanupExpiredEvents()

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
		requestID := func() string { id, _ := uuid.NewV7(); return id.String() }()

		// Create basic usage event (without token data) - will be updated later
		event := &trustedai.UsageEvent{
			ID:              func() string { id, _ := uuid.NewV7(); return id.String() }(),
			RequestID:       requestID,
			UserID:          userID,
			ModelID:         "", // Will be set by provider-specific logic if available
			Status:          "pending",
			UsageDataSource: "unavailable", // Default, can be updated by providers
			DataComplete:    false,         // Default, can be updated by providers
			Timestamp:       time.Now(),
			DurationMs:      nil, // Will be set after request completes
		}

		// Store as pending BEFORE calling next handler to avoid race condition
		pendingEvent := &PendingEvent{
			Event:     event,
			CreatedAt: time.Now(),
			Updated:   false,
		}

		// Check if we've exceeded max pending events
		storePending := true
		if m.countPendingEvents() >= m.config.MaxPendingEvents {
			m.logger.Warn("Max pending events exceeded, will persist immediately after request", "request_id", requestID)
			storePending = false
		} else {
			// Store as pending before calling handler
			m.pendingEvents.Store(requestID, pendingEvent)
		}

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

		// Handle post-request processing
		if status == "failed" {
			// For failed requests, remove from pending (if stored) and persist immediately
			if storePending {
				if pendingData, exists := m.pendingEvents.LoadAndDelete(requestID); exists {
					pending := pendingData.(*PendingEvent)
					event = pending.Event // Use the pre-created event
				}
			}

			// Update event with failure details
			event.Status = status
			event.DurationMs = intPtr(int(duration.Milliseconds()))

			select {
			case m.eventsCh <- event:
				if m.metrics != nil {
					m.metrics.RecordEventCaptured(r.Context())
					m.metrics.UpdateChannelSize(r.Context(), int64(len(m.eventsCh)))
				}
			default:
				m.logger.Warn("Usage tracking channel full, dropping failed event", "request_id", requestID)
				if m.metrics != nil {
					m.metrics.RecordEventDropped(r.Context())
				}
			}
		} else if !storePending {
			// For successful requests when pending storage was skipped (max events exceeded)
			event.Status = status
			event.UsageDataSource = "middleware_fallback"
			event.DurationMs = intPtr(int(duration.Milliseconds()))

			select {
			case m.eventsCh <- event:
				if m.metrics != nil {
					m.metrics.RecordEventCaptured(r.Context())
				}
			default:
				m.logger.Warn("Usage tracking channel full, dropping overflow event", "request_id", requestID)
				if m.metrics != nil {
					m.metrics.RecordEventDropped(r.Context())
				}
			}
		} else {
			// For successful requests with pending storage, update the stored event with duration
			if pendingData, exists := m.pendingEvents.Load(requestID); exists {
				pending := pendingData.(*PendingEvent)
				pending.Event.DurationMs = intPtr(int(duration.Milliseconds()))
			}

			if m.metrics != nil {
				m.metrics.RecordEventCaptured(r.Context())
			}
		}
	})
}

// countPendingEvents returns the number of pending events
func (m *UsageTrackingMiddleware) countPendingEvents() int {
	count := 0
	m.pendingEvents.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// UpdateEvent allows providers to update pending usage events with token data
func (m *UsageTrackingMiddleware) UpdateEvent(ctx context.Context, modelID string, usage *interfaces.TokenUsage, status string, duration time.Duration) {
	requestID, _ := ctx.Value(usageTrackingRequestIDKey).(string)
	if requestID == "" {
		m.logger.Warn("UpdateEvent called without request ID in context")
		return
	}

	m.logger.Debug("UpdateEvent called", "request_id", requestID, "model_id", modelID, "status", status)

	if pendingEventData, exists := m.pendingEvents.LoadAndDelete(requestID); exists {
		pending := pendingEventData.(*PendingEvent)

		// Update event with provider data
		pending.Event.ModelID = modelID
		pending.Event.Status = status
		pending.Event.UsageDataSource = "provider_response"
		pending.Event.DataComplete = usage != nil
		pending.Event.DurationMs = intPtr(int(duration.Milliseconds()))

		// Set token usage if available
		if usage != nil {
			pending.Event.InputTokens = &usage.PromptTokens
			pending.Event.OutputTokens = &usage.CompletionTokens
		}

		// Send completed event to background processor
		select {
		case m.eventsCh <- pending.Event:
			if m.metrics != nil {
				m.metrics.UpdateChannelSize(ctx, int64(len(m.eventsCh)))

				// Record business metrics
				if userID, _ := ctx.Value(usageTrackingUserIDKey).(string); userID != "" {
					m.metrics.RecordModelUsage(ctx, modelID, userID)

					// Record token usage metrics
					if usage != nil {
						m.metrics.RecordTokenUsage(ctx, int64(usage.PromptTokens), modelID, "input", userID)
						m.metrics.RecordTokenUsage(ctx, int64(usage.CompletionTokens), modelID, "output", userID)
					}
				}
			}
		default:
			m.logger.Warn("Usage tracking channel full, dropping updated event", "request_id", requestID)
			if m.metrics != nil {
				m.metrics.RecordEventDropped(ctx)
			}
		}

		m.logger.Debug("Usage event updated and queued", "request_id", requestID, "model_id", modelID, "status", status)
	} else {
		m.logger.Warn("UpdateEvent called for non-existent pending event", "request_id", requestID)
	}
}

// cleanupExpiredEvents runs periodically to clean up expired pending events
func (m *UsageTrackingMiddleware) cleanupExpiredEvents() {
	for {
		select {
		case <-m.cleanupTicker.C:
			cutoff := time.Now().Add(-m.config.PendingEventTimeout)
			toDelete := []string{}

			m.pendingEvents.Range(func(key, value any) bool {
				requestID := key.(string)
				pending := value.(*PendingEvent)

				if pending.CreatedAt.Before(cutoff) && !pending.Updated {
					toDelete = append(toDelete, requestID)
				}
				return true
			})

			// Process expired events
			for _, requestID := range toDelete {
				if pendingData, exists := m.pendingEvents.LoadAndDelete(requestID); exists {
					pending := pendingData.(*PendingEvent)

					// Finalize as timed out
					pending.Event.Status = "timeout"
					pending.Event.UsageDataSource = "middleware_timeout"
					pending.Event.ErrorType = stringPtr("timeout")
					pending.Event.FailureStage = stringPtr("during_generation")

					// Send to background processor
					select {
					case m.eventsCh <- pending.Event:
						m.logger.Debug("Expired pending event finalized",
							"request_id", requestID,
							"age", time.Since(pending.CreatedAt))
					default:
						m.logger.Warn("Failed to queue expired event, dropping", "request_id", requestID)
					}
				}
			}

			if len(toDelete) > 0 {
				m.logger.Info("Cleaned up expired pending events", "count", len(toDelete))
			}

		case <-m.done:
			m.cleanupTicker.Stop()
			return
		}
	}
}

// processEvents runs in a background goroutine to persist usage events
func (m *UsageTrackingMiddleware) processEvents() {
	for {
		select {
		case event := <-m.eventsCh:
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			if err := m.usageRepo.CreateUsageEvent(ctx, event); err != nil {
				m.logger.Error("Failed to persist usage event",
					"error", err,
					"request_id", event.RequestID,
					"user_id", event.UserID)

				if m.metrics != nil {
					m.metrics.RecordDatabaseError(ctx, "create_usage_event", "persistence_error")
				}
			} else {
				m.logger.Debug("Usage event persisted",
					"request_id", event.RequestID,
					"user_id", event.UserID,
					"model_id", event.ModelID,
					"status", event.Status)

				if m.metrics != nil {
					m.metrics.RecordProcessingLatency(ctx, time.Since(startTime))
				}
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

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
