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
	"strings"
	"time"

	"github.com/MadsRC/trustedai"
	"golang.org/x/crypto/argon2"
)

// TokenAuthenticator handles API token authentication
type TokenAuthenticator struct {
	tokenRepository trustedai.TokenRepository
	userRepository  trustedai.UserRepository
}

// NewTokenAuthenticator creates a new token authenticator
func NewTokenAuthenticator(
	tokenRepository trustedai.TokenRepository,
	userRepository trustedai.UserRepository,
) *TokenAuthenticator {
	return &TokenAuthenticator{
		tokenRepository: tokenRepository,
		userRepository:  userRepository,
	}
}

// AuthenticateToken validates an API token and returns the associated user
func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*trustedai.User, error) {
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
		if errors.Is(err, trustedai.ErrNotFound) {
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
