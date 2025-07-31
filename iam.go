// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package trustedai

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"codeberg.org/gai-org/gai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/google/uuid"
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

// SSOConfig is a custom type for SSO configuration
type SSOConfig map[string]any

// Scan implements sql.Scanner interface for database scanning
func (s *SSOConfig) Scan(value any) error {
	if value == nil {
		*s = make(SSOConfig)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("cannot scan %T into SSOConfig", value)
	}
}

// Value implements driver.Valuer interface for database storage
func (s SSOConfig) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
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
	ListByOrganizationForUser(ctx context.Context, requestingUser *User, orgID string) ([]*User, error)
	ListAllForUser(ctx context.Context, requestingUser *User) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// OrganizationRepository defines persistence operations for Organizations
type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	Get(ctx context.Context, id string) (*Organization, error)
	GetByName(ctx context.Context, name string) (*Organization, error)
	List(ctx context.Context) ([]*Organization, error)
	ListForUser(ctx context.Context, user *User) ([]*Organization, error)
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

	// ListUserTokensForUser returns tokens visible to the requesting user
	ListUserTokensForUser(ctx context.Context, requestingUser *User, targetUserID string) ([]*APIToken, error)

	// ListAllTokensForUser returns all tokens visible to the requesting user
	ListAllTokensForUser(ctx context.Context, requestingUser *User) ([]*APIToken, error)

	// RevokeTokenForUser revokes a token if the requesting user has permission
	RevokeTokenForUser(ctx context.Context, requestingUser *User, tokenID string) error

	// UpdateTokenUsage records when a token was last used
	UpdateTokenUsage(ctx context.Context, tokenID string) error
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	Enabled      bool   `json:"enabled"`
}

// OpenRouterCredential represents an OpenRouter API credential
type OpenRouterCredential struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	APIKey      string    `json:"api_key"`
	SiteName    string    `json:"site_name"`
	HTTPReferer string    `json:"http_referer"`
	Enabled     bool      `json:"enabled"`
}

// ModelWithCredentials represents a model with its associated credentials
type ModelWithCredentials struct {
	Model          gai.Model
	CredentialID   uuid.UUID
	CredentialType trustedaiv1.CredentialType
}

// ProviderRepository defines persistence operations for Providers
type ProviderRepository interface {
	GetAllProviders(ctx context.Context) ([]ProviderConfig, error)
	GetProviderByID(ctx context.Context, providerID string) (*ProviderConfig, error)
	CreateProvider(ctx context.Context, provider *ProviderConfig) error
	UpdateProvider(ctx context.Context, provider *ProviderConfig) error
	DeleteProvider(ctx context.Context, providerID string) error
}

// CredentialRepository defines persistence operations for Credentials
type CredentialRepository interface {
	GetOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) (*OpenRouterCredential, error)
	ListOpenRouterCredentials(ctx context.Context) ([]OpenRouterCredential, error)
	CreateOpenRouterCredential(ctx context.Context, cred *OpenRouterCredential) error
	UpdateOpenRouterCredential(ctx context.Context, cred *OpenRouterCredential) error
	DeleteOpenRouterCredential(ctx context.Context, credentialID uuid.UUID) error
}

// ModelRepository defines persistence operations for Models
type ModelRepository interface {
	GetAllModels(ctx context.Context) ([]ModelWithCredentials, error)
	GetModelByID(ctx context.Context, modelID string) (*ModelWithCredentials, error)
	CreateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error
	UpdateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error
	DeleteModel(ctx context.Context, modelID string) error
}

// Context keys for passing data through request contexts
type ContextKey struct{}

var OrganizationContextKey = ContextKey{}
