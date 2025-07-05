// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"log/slog"
	"net/http"

	sharedauth "codeberg.org/MadsRC/llmgw/internal/api/auth"
)

// XAPIKeyMiddleware provides x-api-key header authentication for HTTP handlers
type XAPIKeyMiddleware struct {
	authenticator *sharedauth.TokenAuthenticator
	logger        *slog.Logger
}

// NewXAPIKeyMiddleware creates a new x-api-key middleware
func NewXAPIKeyMiddleware(authenticator *sharedauth.TokenAuthenticator, logger *slog.Logger) *XAPIKeyMiddleware {
	return &XAPIKeyMiddleware{
		authenticator: authenticator,
		logger:        logger,
	}
}

// Authenticate wraps an HTTP handler with x-api-key authentication
func (m *XAPIKeyMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from x-api-key header
		apiKey := r.Header.Get("x-api-key")
		if apiKey == "" {
			m.logger.Debug("Missing x-api-key header", "path", r.URL.Path)
			http.Error(w, "Unauthorized: missing x-api-key", http.StatusUnauthorized)
			return
		}

		// Authenticate the API key (reuse existing token authentication)
		user, err := m.authenticator.AuthenticateToken(r.Context(), apiKey)
		if err != nil {
			m.logger.Debug("API key authentication failed",
				"error", err.Error(),
				"path", r.URL.Path)
			http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		r = r.WithContext(ctx)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// AuthenticateFunc wraps an HTTP handler function with x-api-key authentication
func (m *XAPIKeyMiddleware) AuthenticateFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Authenticate(next).ServeHTTP
}
