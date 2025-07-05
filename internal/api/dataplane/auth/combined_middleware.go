// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	sharedauth "codeberg.org/MadsRC/llmgw/internal/api/auth"
)

// CombinedAuthMiddleware provides authentication via Bearer token or x-api-key header
type CombinedAuthMiddleware struct {
	authenticator *sharedauth.TokenAuthenticator
	logger        *slog.Logger
}

// NewCombinedAuthMiddleware creates a new combined authentication middleware
func NewCombinedAuthMiddleware(authenticator *sharedauth.TokenAuthenticator, logger *slog.Logger) *CombinedAuthMiddleware {
	return &CombinedAuthMiddleware{
		authenticator: authenticator,
		logger:        logger,
	}
}

// Authenticate wraps an HTTP handler with combined authentication (Bearer token or x-api-key)
func (m *CombinedAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		var authType string

		// Try Bearer token first
		bearerToken := extractBearerTokenFromRequest(r)
		if bearerToken != "" {
			token = bearerToken
			authType = "Bearer"
		} else {
			// Try x-api-key header
			apiKey := r.Header.Get("x-api-key")
			if apiKey != "" {
				token = apiKey
				authType = "x-api-key"
			}
		}

		if token == "" {
			m.logger.Debug("Missing authentication credentials", "path", r.URL.Path)
			http.Error(w, "Unauthorized: missing Bearer token or x-api-key", http.StatusUnauthorized)
			return
		}

		// Authenticate the token
		user, err := m.authenticator.AuthenticateToken(r.Context(), token)
		if err != nil {
			m.logger.Debug("Authentication failed",
				"error", err.Error(),
				"authType", authType,
				"path", r.URL.Path)
			http.Error(w, "Unauthorized: invalid credentials", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		r = r.WithContext(ctx)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// AuthenticateFunc wraps an HTTP handler function with combined authentication
func (m *CombinedAuthMiddleware) AuthenticateFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Authenticate(next).ServeHTTP
}

// extractBearerTokenFromRequest extracts the Bearer token from the Authorization header
func extractBearerTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}
