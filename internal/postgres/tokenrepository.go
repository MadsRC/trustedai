// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/MadsRC/trustedai"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

// Ensure TokenRepository implements trustedai.TokenRepository
var _ trustedai.TokenRepository = (*TokenRepository)(nil)

const (
	tokenLength       = 32
	tokenPrefixLength = 8
)

// CreateToken generates and stores a new API token
func (r *TokenRepository) CreateToken(
	ctx context.Context,
	userID string,
	description string,
	expiresAt time.Time,
) (*trustedai.APIToken, string, error) {
	r.options.Logger.Debug("Creating new API token", "userID", userID)

	// Generate random token
	tokenBytes := make([]byte, tokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	prefix := rawToken[:tokenPrefixLength]

	// Generate hashes
	prefixHash := sha256.Sum256([]byte(prefix))
	prefixHashStr := base64.RawURLEncoding.EncodeToString(prefixHash[:])

	tokenHash, err := generateArgon2idHash(rawToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash token: %w", err)
	}

	// Create token record
	apiToken := &trustedai.APIToken{
		ID:          generateUUID(),
		UserID:      userID,
		Description: description,
		PrefixHash:  prefixHashStr,
		TokenHash:   tokenHash,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt.UTC(),
	}

	// Insert into database
	const query = `INSERT INTO tokens 
		(id, user_id, description, prefix_hash, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.options.Db.Exec(ctx, query,
		apiToken.ID,
		apiToken.UserID,
		apiToken.Description,
		apiToken.PrefixHash,
		apiToken.TokenHash,
		apiToken.CreatedAt,
		apiToken.ExpiresAt,
	)

	if err != nil {
		return nil, "", fmt.Errorf("failed to store token: %w", err)
	}

	return apiToken, rawToken, nil
}

// GetTokenByPrefixHash retrieves token by hashed prefix
func (r *TokenRepository) GetTokenByPrefixHash(
	ctx context.Context,
	prefixHash string,
) (*trustedai.APIToken, error) {
	r.options.Logger.Debug("Looking up token by prefix hash")

	const query = `SELECT 
		id, user_id, description, prefix_hash, token_hash, 
		created_at, expires_at, last_used_at 
		FROM tokens WHERE prefix_hash = $1`

	var token trustedai.APIToken
	var lastUsedAt *time.Time // Nullable

	err := r.options.Db.QueryRow(ctx, query, prefixHash).Scan(
		&token.ID,
		&token.UserID,
		&token.Description,
		&token.PrefixHash,
		&token.TokenHash,
		&token.CreatedAt,
		&token.ExpiresAt,
		&lastUsedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("token not found: %w", trustedai.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if lastUsedAt != nil {
		token.LastUsedAt = lastUsedAt
	}

	return &token, nil
}

// RevokeToken permanently invalidates a token
func (r *TokenRepository) RevokeToken(ctx context.Context, tokenID string) error {
	r.options.Logger.Debug("Revoking token", "tokenID", tokenID)

	const query = `DELETE FROM tokens WHERE id = $1`
	cmdTag, err := r.options.Db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("token not found: %w", trustedai.ErrNotFound)
	}

	return nil
}

// ListUserTokens returns all active tokens for a user
func (r *TokenRepository) ListUserTokens(
	ctx context.Context,
	userID string,
) ([]*trustedai.APIToken, error) {
	r.options.Logger.Debug("Listing tokens for user", "userID", userID)

	const query = `SELECT 
		id, user_id, description, prefix_hash, token_hash, 
		created_at, expires_at, last_used_at 
		FROM tokens WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.options.Db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*trustedai.APIToken
	for rows.Next() {
		var token trustedai.APIToken
		var lastUsedAt *time.Time // Nullable

		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Description,
			&token.PrefixHash,
			&token.TokenHash,
			&token.CreatedAt,
			&token.ExpiresAt,
			&lastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		if lastUsedAt != nil {
			token.LastUsedAt = lastUsedAt
		}

		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tokens: %w", err)
	}

	return tokens, nil
}

// UpdateTokenUsage records when a token was last used
func (r *TokenRepository) UpdateTokenUsage(
	ctx context.Context,
	tokenID string,
) error {
	r.options.Logger.Debug("Updating token usage", "tokenID", tokenID)

	const query = `UPDATE tokens SET last_used_at = $1 WHERE id = $2`
	cmdTag, err := r.options.Db.Exec(ctx, query, time.Now().UTC(), tokenID)
	if err != nil {
		return fmt.Errorf("failed to update token usage: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("token not found: %w", trustedai.ErrNotFound)
	}

	return nil
}

// generateArgon2idHash creates a secure hash using Argon2id with recommended parameters
func generateArgon2idHash(token string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Argon2id parameters (19MiB memory, 2 iterations, 1 parallelism)
	memory := uint32(19 * 1024)
	iterations := uint32(2)
	parallelism := uint8(1)
	keyLength := uint32(32)

	// Generate hash
	hash := argon2.IDKey(
		[]byte(token),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	// Format as standard Argon2id encoded string
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encodedHash, nil
}

// ListUserTokensForUser returns tokens visible to the requesting user
// Users can only see their own tokens, system admins can see any user's tokens
func (r *TokenRepository) ListUserTokensForUser(
	ctx context.Context,
	requestingUser *trustedai.User,
	targetUserID string,
) ([]*trustedai.APIToken, error) {
	// Authorization check: users can only list their own tokens
	// unless they are system admins
	if !requestingUser.IsSystemAdmin() && requestingUser.ID != targetUserID {
		return nil, trustedai.ErrUnauthorized
	}

	return r.ListUserTokens(ctx, targetUserID)
}

// ListAllTokensForUser returns all tokens visible to the requesting user
// System admins see all tokens, regular users see only their own tokens
func (r *TokenRepository) ListAllTokensForUser(
	ctx context.Context,
	requestingUser *trustedai.User,
) ([]*trustedai.APIToken, error) {
	if requestingUser.IsSystemAdmin() {
		// System admins see all tokens across all users
		const query = `SELECT 
			id, user_id, description, prefix_hash, token_hash, 
			created_at, expires_at, last_used_at 
			FROM tokens
			ORDER BY created_at DESC`

		rows, err := r.options.Db.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to list all tokens: %w", err)
		}
		defer rows.Close()

		var tokens []*trustedai.APIToken
		for rows.Next() {
			var token trustedai.APIToken
			var lastUsedAt *time.Time

			if err := rows.Scan(
				&token.ID,
				&token.UserID,
				&token.Description,
				&token.PrefixHash,
				&token.TokenHash,
				&token.CreatedAt,
				&token.ExpiresAt,
				&lastUsedAt,
			); err != nil {
				return nil, fmt.Errorf("failed to scan token: %w", err)
			}

			if lastUsedAt != nil {
				token.LastUsedAt = lastUsedAt
			}

			tokens = append(tokens, &token)
		}

		return tokens, rows.Err()
	}

	// Regular users see only their own tokens
	return r.ListUserTokens(ctx, requestingUser.ID)
}

// RevokeTokenForUser revokes a token if the requesting user has permission
func (r *TokenRepository) RevokeTokenForUser(
	ctx context.Context,
	requestingUser *trustedai.User,
	tokenID string,
) error {
	// If not a system admin, verify the token belongs to the requesting user
	if !requestingUser.IsSystemAdmin() {
		const checkQuery = `SELECT user_id FROM tokens WHERE id = $1`
		var tokenUserID string
		err := r.options.Db.QueryRow(ctx, checkQuery, tokenID).Scan(&tokenUserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return trustedai.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to check token ownership: %w", err)
		}

		if tokenUserID != requestingUser.ID {
			return trustedai.ErrUnauthorized
		}
	}

	return r.RevokeToken(ctx, tokenID)
}

// generateUUID creates a random UUID for token IDs
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		panic(err) // This should never happen with a properly functioning system
	}

	// Format as UUID string
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
