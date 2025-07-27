// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/MadsRC/trustedai"
	"github.com/MadsRC/trustedai/internal/api/auth"
)

// TokenInterceptor is a connect interceptor that handles API token authentication
type TokenInterceptor struct {
	authenticator *auth.TokenAuthenticator
}

// NewTokenInterceptor creates a new token interceptor
func NewTokenInterceptor(authenticator *auth.TokenAuthenticator) *TokenInterceptor {
	return &TokenInterceptor{
		authenticator: authenticator,
	}
}

// WrapUnary implements the connect.Interceptor interface
func (i *TokenInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// If session is already authenticated, proceed
		if SessionFromContext(ctx) != nil {
			return next(ctx, req)
		}

		// Extract token from Authorization header
		token := extractTokenFromHeader(req.Header())
		if token == "" {
			return nil, connect.NewError(
				connect.CodeUnauthenticated,
				errors.New("missing credentials"),
			)
		}

		// Authenticate token
		user, err := i.authenticator.AuthenticateToken(ctx, token)
		if err != nil {
			return nil, connect.NewError(
				connect.CodeUnauthenticated,
				fmt.Errorf("invalid credentials: %w", err),
			)
		}

		// Add user to context
		ctx = context.WithValue(ctx, userContextKey{}, user)
		return next(ctx, req)
	}
}

// WrapStreamingClient implements the connect.Interceptor interface
func (i *TokenInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next // Client-side not implemented
}

// WrapStreamingHandler implements the connect.Interceptor interface
func (i *TokenInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// If session is already authenticated, proceed
		if SessionFromContext(ctx) != nil {
			return next(ctx, conn)
		}

		// Extract token from Authorization header
		token := extractTokenFromHeader(conn.RequestHeader())
		if token == "" {
			return connect.NewError(
				connect.CodeUnauthenticated,
				errors.New("missing credentials"),
			)
		}

		// Authenticate token
		user, err := i.authenticator.AuthenticateToken(ctx, token)
		if err != nil {
			return connect.NewError(
				connect.CodeUnauthenticated,
				fmt.Errorf("invalid credentials: %w", err),
			)
		}

		// Add user to context
		ctx = context.WithValue(ctx, userContextKey{}, user)
		return next(ctx, conn)
	}
}

// extractTokenFromHeader extracts a token from the Authorization header
func extractTokenFromHeader(header http.Header) string {
	auth := header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
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
