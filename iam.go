// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package llmgw

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// User represents a system user with SSO integration capabilities
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	OrganizationID string    `json:"organizationId"`
	ExternalID     string    `json:"externalId"` // ID from identity provider
	Provider       string    `json:"provider"`   // "github", "okta", etc
	SystemAdmin    bool      `json:"systemAdmin"`
	CreatedAt      time.Time `json:"createdAt"`
	LastLogin      time.Time `json:"lastLogin"`
}

// GetOrganizationID implements tenant association for authorization
func (u *User) GetOrganizationID() string {
	return u.OrganizationID
}

// IsSystemAdmin checks if user has platform-level privileges
func (u *User) IsSystemAdmin() bool {
	return u.SystemAdmin
}

// SSOConfig is a custom type for SSO configuration with redaction capabilities
type SSOConfig map[string]any

// MarshalJSON redacts sensitive fields during JSON serialization
func (s SSOConfig) MarshalJSON() ([]byte, error) {
	redacted := s.redactSecrets()
	return json.Marshal(redacted)
}

// String redacts sensitive fields in string representations
func (s SSOConfig) String() string {
	redacted := s.redactSecrets()
	return fmt.Sprintf("%v", redacted)
}

// redactSecrets creates a safe copy for serialization
func (s SSOConfig) redactSecrets() map[string]any {
	copy := make(map[string]any, len(s))
	for k, v := range s {
		switch k {
		case "oidc_client_secret", "private_key", "api_key":
			copy[k] = "********"
		default:
			copy[k] = v
		}
	}
	return copy
}

// Organization represents a tenant in the system
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	IsSystem    bool      `json:"isSystem"` // Marks the platform's own organization
	CreatedAt   time.Time `json:"createdAt"`
	SSOType     string    `json:"ssoType"`   // "oidc", "saml", "github", etc
	SSOConfig   SSOConfig `json:"ssoConfig"` // Flexible provider configuration with redaction
}

// IsSSOEnabled checks if organization has SSO configured
func (o *Organization) IsSSOEnabled() bool {
	return o.SSOType != "" // SSO is always enabled if a type is specified
}

// IsSystemTenant identifies the platform management tenant
func (o *Organization) IsSystemTenant() bool {
	return o.IsSystem
}

// UserRepository defines persistence operations for Users
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Get(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByExternalID(ctx context.Context, provider, externalID string) (*User, error)
	ListByOrganization(ctx context.Context, orgID string) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// OrganizationRepository defines persistence operations for Organizations
type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	Get(ctx context.Context, id string) (*Organization, error)
	GetByName(ctx context.Context, name string) (*Organization, error)
	List(ctx context.Context) ([]*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id string) error
}

// APIToken represents an API access credential
type APIToken struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Description string     `json:"description"`
	PrefixHash  string     `json:"-"` // SHA256 of token prefix for lookups
	TokenHash   string     `json:"-"` // Argon2id hash of full token
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// TokenRepository defines persistence operations for API tokens
type TokenRepository interface {
	// CreateToken generates and stores a new API token
	CreateToken(
		ctx context.Context,
		userID string,
		description string,
		expiresAt time.Time,
	) (*APIToken, string, error) // Returns token record and raw token

	// GetTokenByPrefixHash retrieves token by hashed prefix
	GetTokenByPrefixHash(ctx context.Context, prefixHash string) (*APIToken, error)

	// RevokeToken permanently invalidates a token
	RevokeToken(ctx context.Context, tokenID string) error

	// ListUserTokens returns all active tokens for a user
	ListUserTokens(ctx context.Context, userID string) ([]*APIToken, error)

	// UpdateTokenUsage records when a token was last used
	UpdateTokenUsage(ctx context.Context, tokenID string) error
}
