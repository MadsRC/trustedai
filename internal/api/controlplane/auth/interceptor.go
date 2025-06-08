// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"strings"

	"codeberg.org/MadsRC/llmgw/internal/api/auth"
	"connectrpc.com/connect"
)

type Interceptor struct {
	sessionStore auth.SessionStore
}

func NewInterceptor(sessionStore auth.SessionStore) *Interceptor {
	return &Interceptor{
		sessionStore: sessionStore,
	}
}

func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		// Extract token from headers or cookies
		token := extractToken(req)
		if token != "" {
			session, err := i.sessionStore.Get(ctx, token)
			if err == nil {
				ctx = context.WithValue(ctx, sessionContextKey{}, session)
				return next(ctx, req)
			}
			// Fall through to token auth on session error
		}
		return next(ctx, req)
	}
}

// WrapStreamingClient implements the connect.Interceptor interface for client-side streaming
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements the connect.Interceptor interface for server-side streaming
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// Extract token from headers or cookies
		token := extractTokenFromStreamingConn(conn)
		if token != "" {
			session, err := i.sessionStore.Get(ctx, token)
			if err == nil {
				ctx = context.WithValue(ctx, sessionContextKey{}, session)
				return next(ctx, conn)
			}
		}
		return next(ctx, conn)
	}
}

func extractToken(req connect.AnyRequest) string {
	// Check Authorization header first
	if authHeader := req.Header().Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Fallback to cookie
	if cookieHeader := req.Header().Get("Cookie"); cookieHeader != "" {
		cookies := parseCookies(cookieHeader)
		return cookies["session_id"]
	}

	return ""
}

func extractTokenFromStreamingConn(conn connect.StreamingHandlerConn) string {
	// Check Authorization header first
	if authHeader := conn.RequestHeader().Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Fallback to cookie
	if cookieHeader := conn.RequestHeader().Get("Cookie"); cookieHeader != "" {
		cookies := parseCookies(cookieHeader)
		return cookies["session_id"]
	}

	return ""
}

func parseCookies(cookieHeader string) map[string]string {
	cookies := make(map[string]string)
	for c := range strings.SplitSeq(cookieHeader, ";") {
		parts := strings.SplitN(strings.TrimSpace(c), "=", 2)
		if len(parts) == 2 {
			cookies[parts[0]] = parts[1]
		}
	}
	return cookies
}

type sessionContextKey struct{}

func SessionFromContext(ctx context.Context) *auth.Session {
	session, _ := ctx.Value(sessionContextKey{}).(*auth.Session)
	return session
}
