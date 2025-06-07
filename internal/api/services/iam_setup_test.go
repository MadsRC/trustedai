// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *llmgw.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Get(ctx context.Context, id string) (*llmgw.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*llmgw.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.User), args.Error(1)
}

func (m *MockUserRepository) GetByExternalID(ctx context.Context, provider, externalID string) (*llmgw.User, error) {
	args := m.Called(ctx, provider, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.User), args.Error(1)
}

func (m *MockUserRepository) ListByOrganization(ctx context.Context, orgID string) ([]*llmgw.User, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *llmgw.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockOrganizationRepository struct {
	mock.Mock
}

func (m *MockOrganizationRepository) Create(ctx context.Context, org *llmgw.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Get(ctx context.Context, id string) (*llmgw.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetByName(ctx context.Context, name string) (*llmgw.Organization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) List(ctx context.Context) ([]*llmgw.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Update(ctx context.Context, org *llmgw.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) CreateToken(ctx context.Context, userID string, description string, expiresAt time.Time) (*llmgw.APIToken, string, error) {
	args := m.Called(ctx, userID, description, expiresAt)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*llmgw.APIToken), args.String(1), args.Error(2)
}

func (m *MockTokenRepository) GetTokenByPrefixHash(ctx context.Context, prefixHash string) (*llmgw.APIToken, error) {
	args := m.Called(ctx, prefixHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmgw.APIToken), args.Error(1)
}

func (m *MockTokenRepository) RevokeToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockTokenRepository) ListUserTokens(ctx context.Context, userID string) ([]*llmgw.APIToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*llmgw.APIToken), args.Error(1)
}

func (m *MockTokenRepository) UpdateTokenUsage(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func TestNewIam(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockUserRepo := new(MockUserRepository)
	mockOrgRepo := new(MockOrganizationRepository)
	mockTokenRepo := new(MockTokenRepository)

	tests := []struct {
		name    string
		options []IamOption
		want    *Iam
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []IamOption{},
			want: &Iam{
				options: &iamOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []IamOption{WithIamLogger(discardLogger)},
			want: &Iam{
				options: &iamOptions{
					Logger: discardLogger,
				},
			},
		},
		{
			name: "Create with all repositories",
			options: []IamOption{
				WithIamLogger(discardLogger),
				WithUserRepository(mockUserRepo),
				WithOrganizationRepository(mockOrgRepo),
				WithTokenRepository(mockTokenRepo),
			},
			want: &Iam{
				options: &iamOptions{
					Logger:                 discardLogger,
					UserRepository:         mockUserRepo,
					OrganizationRepository: mockOrgRepo,
					TokenRepository:        mockTokenRepo,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIam(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want.options.Logger, got.options.Logger)

			if tt.want.options.UserRepository != nil {
				assert.Equal(t, tt.want.options.UserRepository, got.options.UserRepository)
			}

			if tt.want.options.OrganizationRepository != nil {
				assert.Equal(t, tt.want.options.OrganizationRepository, got.options.OrganizationRepository)
			}

			if tt.want.options.TokenRepository != nil {
				assert.Equal(t, tt.want.options.TokenRepository, got.options.TokenRepository)
			}
		})
	}
}

func TestNewIam_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []IamOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []IamOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalIamOptions = []IamOption{
				WithIamLogger(tt.inputLogger),
			}
			got1, _ := NewIam(tt.options...)
			got2, _ := NewIam(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewIam() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewIam() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewIam() = %v, want %v", got1, got2)
			}
			GlobalIamOptions = []IamOption{}
			got3, _ := NewIam(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewIam() = %v, want %v", got3, slog.Default())
			}
		})
	}
}

func TestWithRepositories(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockOrgRepo := new(MockOrganizationRepository)
	mockTokenRepo := new(MockTokenRepository)

	iam, _ := NewIam(
		WithUserRepository(mockUserRepo),
		WithOrganizationRepository(mockOrgRepo),
		WithTokenRepository(mockTokenRepo),
	)

	assert.Equal(t, mockUserRepo, iam.options.UserRepository)
	assert.Equal(t, mockOrgRepo, iam.options.OrganizationRepository)
	assert.Equal(t, mockTokenRepo, iam.options.TokenRepository)
}
