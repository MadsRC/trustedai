// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"connectrpc.com/connect"
	"golang.org/x/crypto/argon2"
)

// TokenAuthenticator handles API token authentication
type TokenAuthenticator struct {
	tokenRepository llmgw.TokenRepository
	userRepository  llmgw.UserRepository
}

// NewTokenAuthenticator creates a new token authenticator
func NewTokenAuthenticator(
	tokenRepository llmgw.TokenRepository,
	userRepository llmgw.UserRepository,
) *TokenAuthenticator {
	return &TokenAuthenticator{
		tokenRepository: tokenRepository,
		userRepository:  userRepository,
	}
}

// AuthenticateToken validates an API token and returns the associated user
func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*llmgw.User, error) {
	// Token must be at least prefix length
	if len(token) <= 8 {
		return nil, errors.New("invalid token format")
	}

	// Extract prefix (first 8 chars)
	prefix := token[:8]

	// Hash the prefix for lookup
	prefixHash := sha256.Sum256([]byte(prefix))
	prefixHashStr := base64.RawURLEncoding.EncodeToString(prefixHash[:])

	// Look up token by prefix hash
	tokenRecord, err := a.tokenRepository.GetTokenByPrefixHash(ctx, prefixHashStr)
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, errors.New("invalid token")
		}
		return nil, fmt.Errorf("token lookup failed: %w", err)
	}

	// Check if token is expired
	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	// Verify full token hash
	if !verifyArgon2idHash(tokenRecord.TokenHash, token) {
		return nil, errors.New("invalid token")
	}

	// Update last used timestamp
	go func() {
		// Use background context since this is non-critical
		bgCtx := context.Background()
		_ = a.tokenRepository.UpdateTokenUsage(bgCtx, tokenRecord.ID)
	}()

	// Get associated user
	user, err := a.userRepository.Get(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// verifyArgon2idHash checks if a token matches the stored hash
func verifyArgon2idHash(encodedHash, token string) bool {
	// Parse the hash string
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return false
	}

	// Extract parameters
	var version int
	var memory uint32
	var iterations uint32
	var parallelism uint8

	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return false
	}

	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	keyLength := uint32(len(decodedHash))
	computedHash := argon2.IDKey(
		[]byte(token),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	// Compare in constant time using crypto/subtle
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1
}

// TokenInterceptor is a connect interceptor that handles API token authentication
type TokenInterceptor struct {
	authenticator *TokenAuthenticator
}

// NewTokenInterceptor creates a new token interceptor
func NewTokenInterceptor(authenticator *TokenAuthenticator) *TokenInterceptor {
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
func UserFromContext(ctx context.Context) *llmgw.User {
	user, _ := ctx.Value(userContextKey{}).(*llmgw.User)
	return user
}
