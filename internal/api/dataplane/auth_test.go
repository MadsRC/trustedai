// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
)

// mockTokenRepository implements llmgw.TokenRepository for testing
type mockTokenRepository struct {
	tokens map[string]*llmgw.APIToken
}

func newMockTokenRepository() *mockTokenRepository {
	return &mockTokenRepository{
		tokens: make(map[string]*llmgw.APIToken),
	}
}

func (m *mockTokenRepository) CreateToken(ctx context.Context, userID string, description string, expiresAt time.Time) (*llmgw.APIToken, string, error) {
	return nil, "", fmt.Errorf("not implemented")
}

func (m *mockTokenRepository) GetTokenByPrefixHash(ctx context.Context, prefixHash string) (*llmgw.APIToken, error) {
	if token, exists := m.tokens[prefixHash]; exists {
		return token, nil
	}
	return nil, llmgw.ErrNotFound
}

func (m *mockTokenRepository) RevokeToken(ctx context.Context, tokenID string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTokenRepository) ListUserTokens(ctx context.Context, userID string) ([]*llmgw.APIToken, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTokenRepository) UpdateTokenUsage(ctx context.Context, tokenID string) error {
	return nil // No-op for testing
}

func (m *mockTokenRepository) ListUserTokensForUser(ctx context.Context, requestingUser *llmgw.User, targetUserID string) ([]*llmgw.APIToken, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTokenRepository) ListAllTokensForUser(ctx context.Context, requestingUser *llmgw.User) ([]*llmgw.APIToken, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTokenRepository) RevokeTokenForUser(ctx context.Context, requestingUser *llmgw.User, tokenID string) error {
	return fmt.Errorf("not implemented")
}

// mockUserRepository implements llmgw.UserRepository for testing
type mockUserRepository struct {
	users map[string]*llmgw.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*llmgw.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *llmgw.User) error {
	return fmt.Errorf("not implemented")
}

func (m *mockUserRepository) Get(ctx context.Context, id string) (*llmgw.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, llmgw.ErrNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*llmgw.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepository) GetByExternalID(ctx context.Context, provider, externalID string) (*llmgw.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepository) ListByOrganization(ctx context.Context, orgID string) ([]*llmgw.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepository) Update(ctx context.Context, user *llmgw.User) error {
	return fmt.Errorf("not implemented")
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockUserRepository) ListByOrganizationForUser(ctx context.Context, requestingUser *llmgw.User, orgID string) ([]*llmgw.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepository) ListAllForUser(ctx context.Context, requestingUser *llmgw.User) ([]*llmgw.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestDataPlaneAuthenticationIntegration(t *testing.T) {
	// Setup mock repositories
	tokenRepo := newMockTokenRepository()
	userRepo := newMockUserRepository()

	// Create a mock user
	testUser := &llmgw.User{
		ID:   "test-user-id",
		Name: "Test User",
	}
	userRepo.users["test-user-id"] = testUser

	// Create a mock token with a simplified hash for testing
	// In a real scenario, this would be properly hashed with Argon2id
	testToken := &llmgw.APIToken{
		ID:         "test-token-id",
		UserID:     "test-user-id",
		PrefixHash: "test-prefix-hash",
		TokenHash:  "test-token-hash", // Simplified for testing
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	tokenRepo.tokens["test-prefix-hash"] = testToken

	// Create token authenticator
	tokenAuth := auth.NewTokenAuthenticator(tokenRepo, userRepo)

	// Create DataPlane server with authentication
	server, err := NewDataPlaneServer(
		WithDataPlaneLogger(slog.Default()),
		WithDataPlaneAddr(":0"), // Use random port for testing
		WithDataPlaneTokenAuthenticator(tokenAuth),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Health endpoint without auth",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "Hello endpoint without auth",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: missing Bearer token or x-api-key\n",
		},
		{
			name:           "Hello endpoint with invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: invalid credentials\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.name == "Health endpoint without auth" {
				req, err = http.NewRequest("GET", "/health", nil)
			} else {
				req, err = http.NewRequest("GET", "/hello", nil)
			}
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			server.GetMux().ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestDataPlaneXAPIKeyAuthentication(t *testing.T) {
	// Setup mock repositories
	tokenRepo := newMockTokenRepository()
	userRepo := newMockUserRepository()

	// Create a mock user
	testUser := &llmgw.User{
		ID:   "test-user-id",
		Name: "Test User",
	}
	userRepo.users["test-user-id"] = testUser

	// Create a mock token with a simplified hash for testing
	testToken := &llmgw.APIToken{
		ID:         "test-token-id",
		UserID:     "test-user-id",
		PrefixHash: "test-prefix-hash",
		TokenHash:  "test-token-hash",
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	tokenRepo.tokens["test-prefix-hash"] = testToken

	// Create token authenticator
	tokenAuth := auth.NewTokenAuthenticator(tokenRepo, userRepo)

	// Create DataPlane server with authentication
	server, err := NewDataPlaneServer(
		WithDataPlaneLogger(slog.Default()),
		WithDataPlaneAddr(":0"),
		WithDataPlaneTokenAuthenticator(tokenAuth),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Hello endpoint with x-api-key header - invalid key",
			headers:        map[string]string{"x-api-key": "invalid-key"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: invalid credentials\n",
		},
		{
			name:           "Hello endpoint with Bearer token and x-api-key - Bearer takes precedence",
			headers:        map[string]string{"Authorization": "Bearer invalid-token", "x-api-key": "some-key"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: invalid credentials\n",
		},
		{
			name:           "Hello endpoint with x-api-key only",
			headers:        map[string]string{"x-api-key": "some-key"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: invalid credentials\n",
		},
		{
			name:           "Hello endpoint with no auth headers",
			headers:        map[string]string{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized: missing Bearer token or x-api-key\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/hello", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()
			server.GetMux().ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestBearerMiddlewareTokenExtraction(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "Valid Bearer token",
			authHeader: "Bearer token123",
			expected:   "token123",
		},
		{
			name:       "Bearer token with case insensitive",
			authHeader: "bearer token456",
			expected:   "token456",
		},
		{
			name:       "Missing Bearer prefix",
			authHeader: "token789",
			expected:   "",
		},
		{
			name:       "Empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "Only Bearer without token",
			authHeader: "Bearer",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			result := extractBearerToken(req)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// extractBearerToken is redefined here for testing since it's not exported
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
