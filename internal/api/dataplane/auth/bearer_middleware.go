// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MadsRC/trustedai"
	sharedauth "github.com/MadsRC/trustedai/internal/api/auth"
)

// BearerMiddleware provides Bearer token authentication for HTTP handlers
type BearerMiddleware struct {
	authenticator *sharedauth.TokenAuthenticator
	logger        *slog.Logger
}

// NewBearerMiddleware creates a new Bearer token middleware
func NewBearerMiddleware(authenticator *sharedauth.TokenAuthenticator, logger *slog.Logger) *BearerMiddleware {
	return &BearerMiddleware{
		authenticator: authenticator,
		logger:        logger,
	}
}

// Authenticate wraps an HTTP handler with Bearer token authentication
func (m *BearerMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Bearer token from Authorization header
		token := extractBearerToken(r)
		if token == "" {
			m.logger.Debug("Missing Authorization header", "path", r.URL.Path)
			http.Error(w, "Unauthorized: missing Bearer token", http.StatusUnauthorized)
			return
		}

		// Authenticate the token
		user, err := m.authenticator.AuthenticateToken(r.Context(), token)
		if err != nil {
			m.logger.Debug("Token authentication failed",
				"error", err.Error(),
				"path", r.URL.Path)
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		r = r.WithContext(ctx)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// AuthenticateFunc wraps an HTTP handler function with Bearer token authentication
func (m *BearerMiddleware) AuthenticateFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Authenticate(next).ServeHTTP
}

// extractBearerToken extracts the Bearer token from the Authorization header
func extractBearerToken(r *http.Request) string {
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

type userContextKey struct{}

// UserFromContext extracts the user from the context
func UserFromContext(ctx context.Context) *trustedai.User {
	user, _ := ctx.Value(userContextKey{}).(*trustedai.User)
	return user
}

// UserFromHTTPContext extracts the authenticated user from an HTTP request context
func UserFromHTTPContext(r *http.Request) *trustedai.User {
	return UserFromContext(r.Context())
}
