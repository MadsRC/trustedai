// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"codeberg.org/gai-org/gai"
	"connectrpc.com/connect"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestModelManagement_ListSupportedCredentialTypes(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service, err := NewModelManagement(
		WithModelManagementLogger(discardLogger),
	)
	require.NoError(t, err)

	req := connect.NewRequest(&trustedaiv1.ModelManagementServiceListSupportedCredentialTypesRequest{})

	resp, err := service.ListSupportedCredentialTypes(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	credentialTypes := resp.Msg.GetCredentialTypes()
	require.Len(t, credentialTypes, 1)

	openRouterType := credentialTypes[0]
	assert.Equal(t, trustedaiv1.CredentialType_CREDENTIAL_TYPE_OPENROUTER, openRouterType.GetType())
	assert.Equal(t, "OpenRouter", openRouterType.GetDisplayName())
	assert.Contains(t, openRouterType.GetDescription(), "OpenRouter API credentials")
}

func TestModelManagement_ListSupportedProviders(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service, err := NewModelManagement(
		WithModelManagementLogger(discardLogger),
	)
	require.NoError(t, err)

	req := connect.NewRequest(&trustedaiv1.ModelManagementServiceListSupportedProvidersRequest{})

	resp, err := service.ListSupportedProviders(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	providers := resp.Msg.GetProviders()
	require.Len(t, providers, 1)

	openRouterProvider := providers[0]
	assert.Equal(t, models.PROVIDER_ID_OPENROUTER, openRouterProvider.GetId())
	assert.Equal(t, "OpenRouter", openRouterProvider.GetName())
	assert.Equal(t, models.PROVIDER_ID_OPENROUTER, openRouterProvider.GetProviderType())
	assert.True(t, openRouterProvider.GetEnabled())
	assert.NotNil(t, openRouterProvider.GetCreatedAt())
	assert.NotNil(t, openRouterProvider.GetUpdatedAt())
}

func TestModelManagement_ListSupportedModelsForProvider(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service, err := NewModelManagement(
		WithModelManagementLogger(discardLogger),
	)
	require.NoError(t, err)

	tests := []struct {
		name        string
		providerId  trustedaiv1.ProviderId
		wantErr     bool
		expectedErr string
	}{
		{
			name:       "OpenRouter provider",
			providerId: trustedaiv1.ProviderId_PROVIDER_ID_OPENROUTER,
			wantErr:    false,
		},
		{
			name:        "Unspecified provider",
			providerId:  trustedaiv1.ProviderId_PROVIDER_ID_UNSPECIFIED,
			wantErr:     true,
			expectedErr: "provider ID must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := connect.NewRequest(&trustedaiv1.ModelManagementServiceListSupportedModelsForProviderRequest{
				ProviderId: tt.providerId,
			})

			resp, err := service.ListSupportedModelsForProvider(context.Background(), req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			supportedModels := resp.Msg.GetModels()
			require.NotEmpty(t, supportedModels, "Expected at least one model for OpenRouter provider")

			// Verify model structure
			for _, model := range supportedModels {
				assert.NotEmpty(t, model.GetId())
				assert.NotEmpty(t, model.GetName())
				assert.Equal(t, models.PROVIDER_ID_OPENROUTER, model.GetProviderId())
				assert.True(t, model.GetEnabled())
				assert.NotNil(t, model.GetCreatedAt())
				assert.NotNil(t, model.GetUpdatedAt())
				assert.NotNil(t, model.GetCapabilities())

				// Metadata should be present
				assert.NotNil(t, model.GetMetadata())

				// Capabilities should have reasonable defaults
				caps := model.GetCapabilities()
				assert.GreaterOrEqual(t, caps.GetMaxInputTokens(), int32(0))
				assert.GreaterOrEqual(t, caps.GetMaxOutputTokens(), int32(0))
			}
		})
	}
}

func TestModelManagement_ListSupportedModelsForProvider_ValidatesModels(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service, err := NewModelManagement(
		WithModelManagementLogger(discardLogger),
	)
	require.NoError(t, err)

	req := connect.NewRequest(&trustedaiv1.ModelManagementServiceListSupportedModelsForProviderRequest{
		ProviderId: trustedaiv1.ProviderId_PROVIDER_ID_OPENROUTER,
	})

	resp, err := service.ListSupportedModelsForProvider(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	supportedModels := resp.Msg.GetModels()
	require.NotEmpty(t, supportedModels)

	// Check that we get the expected number of models from hardcoded data
	expectedModelCount := len(models.OpenRouterModels)
	assert.Equal(t, expectedModelCount, len(supportedModels))

	// Verify each model has required fields
	for _, model := range supportedModels {
		t.Run("Model_"+model.GetId(), func(t *testing.T) {
			assert.NotEmpty(t, model.GetId())
			assert.NotEmpty(t, model.GetName())
			assert.Equal(t, models.PROVIDER_ID_OPENROUTER, model.GetProviderId())

			// Check capabilities are properly converted
			caps := model.GetCapabilities()
			require.NotNil(t, caps)

			// Check that pricing exists if the original model has pricing
			if originalModel, exists := models.OpenRouterModels[model.GetId()]; exists {
				if originalModel.Pricing.InputTokenPrice > 0 || originalModel.Pricing.OutputTokenPrice > 0 {
					assert.NotNil(t, model.GetPricing())
					assert.Equal(t, originalModel.Pricing.InputTokenPrice, model.GetPricing().GetInputTokenPrice())
					assert.Equal(t, originalModel.Pricing.OutputTokenPrice, model.GetPricing().GetOutputTokenPrice())
				}
			}
		})
	}
}

func TestNewModelManagement_Minimal(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockCredentialRepo := new(MockCredentialRepository)
	mockModelRepo := new(MockModelRepository)

	tests := []struct {
		name    string
		options []ModelManagementOption
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []ModelManagementOption{},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []ModelManagementOption{WithModelManagementLogger(discardLogger)},
			wantErr: false,
		},
		{
			name: "Create with all repositories",
			options: []ModelManagementOption{
				WithModelManagementLogger(discardLogger),
				WithCredentialRepository(mockCredentialRepo),
				WithModelRepository(mockModelRepo),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewModelManagement(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewModelManagement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NotNil(t, got)
			assert.NotNil(t, got.options.Logger)

			// Verify repositories are set if provided
			if len(tt.options) > 1 {
				assert.NotNil(t, got.options.CredentialRepository)
				assert.NotNil(t, got.options.ModelRepository)
			}
		})
	}
}

// Mock repositories for testing
type MockCredentialRepository struct {
	mock.Mock
}

func (m *MockCredentialRepository) CreateOpenRouterCredential(ctx context.Context, credential *trustedai.OpenRouterCredential) error {
	args := m.Called(ctx, credential)
	return args.Error(0)
}

func (m *MockCredentialRepository) GetOpenRouterCredential(ctx context.Context, id uuid.UUID) (*trustedai.OpenRouterCredential, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trustedai.OpenRouterCredential), args.Error(1)
}

func (m *MockCredentialRepository) ListOpenRouterCredentials(ctx context.Context) ([]trustedai.OpenRouterCredential, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trustedai.OpenRouterCredential), args.Error(1)
}

func (m *MockCredentialRepository) UpdateOpenRouterCredential(ctx context.Context, credential *trustedai.OpenRouterCredential) error {
	args := m.Called(ctx, credential)
	return args.Error(0)
}

func (m *MockCredentialRepository) DeleteOpenRouterCredential(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockModelRepository struct {
	mock.Mock
}

func (m *MockModelRepository) GetAllModels(ctx context.Context) ([]gai.Model, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]gai.Model), args.Error(1)
}

func (m *MockModelRepository) GetAllModelsWithReference(ctx context.Context) ([]trustedai.ModelWithReference, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trustedai.ModelWithReference), args.Error(1)
}

func (m *MockModelRepository) GetModelByID(ctx context.Context, modelID string) (*gai.Model, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gai.Model), args.Error(1)
}

func (m *MockModelRepository) GetModelByIDWithReference(ctx context.Context, modelID string) (*trustedai.ModelWithReference, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trustedai.ModelWithReference), args.Error(1)
}

func (m *MockModelRepository) GetModelWithCredentials(ctx context.Context, modelID string) (*trustedai.ModelWithCredentials, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trustedai.ModelWithCredentials), args.Error(1)
}

func (m *MockModelRepository) CreateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	args := m.Called(ctx, model, credentialID, credentialType)
	return args.Error(0)
}

func (m *MockModelRepository) UpdateModel(ctx context.Context, model *gai.Model, credentialID uuid.UUID, credentialType trustedaiv1.CredentialType) error {
	args := m.Called(ctx, model, credentialID, credentialType)
	return args.Error(0)
}

func (m *MockModelRepository) DeleteModel(ctx context.Context, modelID string) error {
	args := m.Called(ctx, modelID)
	return args.Error(0)
}
