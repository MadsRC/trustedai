// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"codeberg.org/gai-org/gai"
	"connectrpc.com/connect"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1/trustedaiv1connect"
	"github.com/MadsRC/trustedai/internal/models"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ensure ModelManagement implements the required interfaces
var _ trustedaiv1connect.ModelManagementServiceHandler = (*ModelManagement)(nil)

// Provider Service Methods

// GetProvider retrieves a provider by ID from hardcoded providers
func (s *ModelManagement) GetProvider(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceGetProviderRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceGetProviderResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] GetProvider invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: provider ID is required"))
	}

	// Check hardcoded providers from models package
	var protoProvider *trustedaiv1.Provider
	switch req.Msg.GetId() {
	case "openrouter":
		protoProvider = &trustedaiv1.Provider{
			Id:           "openrouter",
			Name:         "OpenRouter",
			ProviderType: "openrouter",
			Enabled:      true,
			CreatedAt:    timestamppb.New(time.Now()),
			UpdatedAt:    timestamppb.New(time.Now()),
		}
	default:
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model management service: provider not found"))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceGetProviderResponse{
		Provider: protoProvider,
	}

	return connect.NewResponse(response), nil
}

// ListProviders retrieves all hardcoded providers
func (s *ModelManagement) ListProviders(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListProvidersRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListProvidersResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListProviders invoked", "includeDisabled", req.Msg.GetIncludeDisabled())

	// Return hardcoded providers
	protoProviders := []*trustedaiv1.Provider{
		{
			Id:           "openrouter",
			Name:         "OpenRouter",
			ProviderType: "openrouter",
			Enabled:      true,
			CreatedAt:    timestamppb.New(time.Now()),
			UpdatedAt:    timestamppb.New(time.Now()),
		},
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceListProvidersResponse{
		Providers: protoProviders,
	}

	return connect.NewResponse(response), nil
}

// OpenRouter Credential Service Methods

// CreateOpenRouterCredential creates a new OpenRouter credential
func (s *ModelManagement) CreateOpenRouterCredential(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceCreateOpenRouterCredentialRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceCreateOpenRouterCredentialResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] CreateOpenRouterCredential invoked", "name", req.Msg.GetCredential().GetName())

	// Validate request
	if req.Msg.GetCredential() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential is required"))
	}

	if req.Msg.GetCredential().GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential name is required"))
	}

	if req.Msg.GetCredential().GetApiKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: API key is required"))
	}

	// Generate ID if not provided
	var credentialID uuid.UUID
	if req.Msg.GetCredential().GetId() != "" {
		parsedID, err := uuid.Parse(req.Msg.GetCredential().GetId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
		}
		credentialID = parsedID
	}
	// If ID is empty, the repository will generate one

	// Create credential domain object
	credential := &trustedai.OpenRouterCredential{
		ID:          credentialID,
		Name:        req.Msg.GetCredential().GetName(),
		Description: req.Msg.GetCredential().GetDescription(),
		APIKey:      req.Msg.GetCredential().GetApiKey(),
		SiteName:    req.Msg.GetCredential().GetSiteName(),
		HTTPReferer: req.Msg.GetCredential().GetHttpReferer(),
		Enabled:     req.Msg.GetCredential().GetEnabled(),
	}

	// Create credential in repository
	err := s.options.CredentialRepository.CreateOpenRouterCredential(ctx, credential)
	if err != nil {
		s.options.Logger.Error("Failed to create OpenRouter credential", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to create OpenRouter credential: %w", err))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceCreateOpenRouterCredentialResponse{
		Credential: openRouterCredentialToProto(credential),
	}

	return connect.NewResponse(response), nil
}

// GetOpenRouterCredential retrieves an OpenRouter credential by ID
func (s *ModelManagement) GetOpenRouterCredential(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceGetOpenRouterCredentialRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceGetOpenRouterCredentialResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] GetOpenRouterCredential invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential ID is required"))
	}

	// Parse UUID
	credentialID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
	}

	// Get credential from repository
	credential, err := s.options.CredentialRepository.GetOpenRouterCredential(ctx, credentialID)
	if err != nil {
		s.options.Logger.Error("Failed to get OpenRouter credential", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("model management service: credential not found: %w", err))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceGetOpenRouterCredentialResponse{
		Credential: openRouterCredentialToProto(credential),
	}

	return connect.NewResponse(response), nil
}

// ListOpenRouterCredentials retrieves all OpenRouter credentials
func (s *ModelManagement) ListOpenRouterCredentials(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListOpenRouterCredentialsRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListOpenRouterCredentialsResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListOpenRouterCredentials invoked", "includeDisabled", req.Msg.GetIncludeDisabled())

	// Get credentials from repository (existing repo only returns enabled credentials)
	credentials, err := s.options.CredentialRepository.ListOpenRouterCredentials(ctx)
	if err != nil {
		s.options.Logger.Error("Failed to list OpenRouter credentials", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to list OpenRouter credentials: %w", err))
	}

	// Convert credentials to proto
	protoCredentials := make([]*trustedaiv1.OpenRouterCredential, 0, len(credentials))
	for _, credential := range credentials {
		protoCredentials = append(protoCredentials, openRouterCredentialToProto(&credential))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceListOpenRouterCredentialsResponse{
		Credentials: protoCredentials,
	}

	return connect.NewResponse(response), nil
}

// UpdateOpenRouterCredential updates an existing OpenRouter credential
func (s *ModelManagement) UpdateOpenRouterCredential(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceUpdateOpenRouterCredentialRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceUpdateOpenRouterCredentialResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] UpdateOpenRouterCredential invoked", "id", req.Msg.GetCredential().GetId())

	// Validate request
	if req.Msg.GetCredential() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential is required"))
	}

	if req.Msg.GetCredential().GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential ID is required"))
	}

	// Parse UUID
	credentialID, err := uuid.Parse(req.Msg.GetCredential().GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
	}

	// Get existing credential
	existingCredential, err := s.options.CredentialRepository.GetOpenRouterCredential(ctx, credentialID)
	if err != nil {
		s.options.Logger.Error("Failed to get OpenRouter credential for update", "error", err, "id", req.Msg.GetCredential().GetId())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("model management service: credential not found: %w", err))
	}

	// Update fields if provided
	if req.Msg.GetCredential().GetName() != "" {
		existingCredential.Name = req.Msg.GetCredential().GetName()
	}

	if req.Msg.GetCredential().GetDescription() != "" {
		existingCredential.Description = req.Msg.GetCredential().GetDescription()
	}

	if req.Msg.GetCredential().GetApiKey() != "" {
		existingCredential.APIKey = req.Msg.GetCredential().GetApiKey()
	}

	if req.Msg.GetCredential().GetSiteName() != "" {
		existingCredential.SiteName = req.Msg.GetCredential().GetSiteName()
	}

	if req.Msg.GetCredential().GetHttpReferer() != "" {
		existingCredential.HTTPReferer = req.Msg.GetCredential().GetHttpReferer()
	}

	// Update enabled status if explicitly provided
	if req.Msg.GetHasEnabled() {
		existingCredential.Enabled = req.Msg.GetCredential().GetEnabled()
	}

	// Update credential in repository
	err = s.options.CredentialRepository.UpdateOpenRouterCredential(ctx, existingCredential)
	if err != nil {
		s.options.Logger.Error("Failed to update OpenRouter credential", "error", err, "id", req.Msg.GetCredential().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to update OpenRouter credential: %w", err))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceUpdateOpenRouterCredentialResponse{
		Credential: openRouterCredentialToProto(existingCredential),
	}

	return connect.NewResponse(response), nil
}

// DeleteOpenRouterCredential removes an OpenRouter credential
func (s *ModelManagement) DeleteOpenRouterCredential(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceDeleteOpenRouterCredentialRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceDeleteOpenRouterCredentialResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] DeleteOpenRouterCredential invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential ID is required"))
	}

	// Parse UUID
	credentialID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
	}

	// Check if credential exists
	_, err = s.options.CredentialRepository.GetOpenRouterCredential(ctx, credentialID)
	if err != nil {
		s.options.Logger.Error("Failed to get OpenRouter credential for deletion", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("model management service: credential not found: %w", err))
	}

	// Delete credential (soft delete by setting enabled = false)
	err = s.options.CredentialRepository.DeleteOpenRouterCredential(ctx, credentialID)
	if err != nil {
		s.options.Logger.Error("Failed to delete OpenRouter credential", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to delete OpenRouter credential: %w", err))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceDeleteOpenRouterCredentialResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// CreateModel creates a new model with automatic inference from hardcoded models
func (s *ModelManagement) CreateModel(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceCreateModelRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceCreateModelResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] CreateModel invoked", "id", req.Msg.GetModel().GetId())

	// Validate request
	if req.Msg.GetModel() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model is required"))
	}

	if req.Msg.GetModel().GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model ID is required"))
	}

	// Extract model reference from metadata
	modelReference := ""
	if req.Msg.GetModel().GetMetadata() != nil {
		modelReference = req.Msg.GetModel().GetMetadata()["model_reference"]
	}

	if modelReference == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model_reference must be provided in metadata"))
	}

	if req.Msg.GetModel().GetProviderId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: provider ID is required"))
	}

	if req.Msg.GetModel().GetCredentialId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential ID is required"))
	}

	if req.Msg.GetModel().GetCredentialType() == trustedaiv1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential type is required"))
	}

	// Look up hardcoded model for inference
	hardcodedModel, err := models.GetModelByReference(modelReference)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid model reference: %w", err))
	}

	// Parse credential ID
	credentialID, err := uuid.Parse(req.Msg.GetModel().GetCredentialId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
	}

	// Create enhanced model with inference from hardcoded model
	gaiModel := createModelWithInference(req.Msg.GetModel(), hardcodedModel, modelReference)

	// Create model in repository
	err = s.options.ModelRepository.CreateModel(ctx, gaiModel, credentialID, req.Msg.GetModel().GetCredentialType())
	if err != nil {
		s.options.Logger.Error("Failed to create model", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to create model: %w", err))
	}

	// Convert back to proto for response
	protoModel := gaiModelToProto(gaiModel, req.Msg.GetModel().GetCredentialId(), req.Msg.GetModel().GetCredentialType())

	// Return response
	response := &trustedaiv1.ModelManagementServiceCreateModelResponse{
		Model: protoModel,
	}

	return connect.NewResponse(response), nil
}

// GetModel retrieves a model by ID with model reference
func (s *ModelManagement) GetModel(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceGetModelRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceGetModelResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] GetModel invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model ID is required"))
	}

	// Get model with credentials from repository
	modelWithCreds, err := s.options.ModelRepository.GetModelByID(ctx, req.Msg.GetId())
	if err != nil {
		s.options.Logger.Error("Failed to get model", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("model management service: model not found: %w", err))
	}

	// Convert to proto model
	protoModel := &trustedaiv1.Model{
		Id:             modelWithCreds.Model.ID,
		Name:           modelWithCreds.Model.Name,
		ProviderId:     modelWithCreds.Model.Provider,
		CredentialId:   modelWithCreds.CredentialID.String(),
		CredentialType: modelWithCreds.CredentialType,
		Metadata:       convertGaiMetadataToProto(modelWithCreds.Model.Metadata),
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Now()),
		UpdatedAt:      timestamppb.New(time.Now()),
	}

	// Add pricing if available
	if modelWithCreds.Model.Pricing.InputTokenPrice > 0 || modelWithCreds.Model.Pricing.OutputTokenPrice > 0 {
		protoModel.Pricing = &trustedaiv1.ModelPricing{
			InputTokenPrice:  modelWithCreds.Model.Pricing.InputTokenPrice,
			OutputTokenPrice: modelWithCreds.Model.Pricing.OutputTokenPrice,
		}
	}

	// Add capabilities
	protoModel.Capabilities = &trustedaiv1.ModelCapabilities{
		SupportsStreaming: modelWithCreds.Model.Capabilities.SupportsStreaming,
		SupportsJson:      modelWithCreds.Model.Capabilities.SupportsJSON,
		SupportsTools:     modelWithCreds.Model.Capabilities.SupportsTools,
		SupportsVision:    modelWithCreds.Model.Capabilities.SupportsVision,
		SupportsReasoning: modelWithCreds.Model.Capabilities.SupportsReasoning,
		MaxInputTokens:    int32(modelWithCreds.Model.Capabilities.MaxInputTokens),
		MaxOutputTokens:   int32(modelWithCreds.Model.Capabilities.MaxOutputTokens),
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceGetModelResponse{
		Model: protoModel,
	}

	return connect.NewResponse(response), nil
}

// ListModels retrieves models based on filters with model references
func (s *ModelManagement) ListModels(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListModelsRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListModelsResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListModels invoked", "includeDisabled", req.Msg.GetIncludeDisabled())

	// Get models with credentials from repository (existing repo only returns enabled models)
	modelsWithCreds, err := s.options.ModelRepository.GetAllModels(ctx)
	if err != nil {
		s.options.Logger.Error("Failed to list models", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to list models: %w", err))
	}

	// Convert models to proto
	protoModels := make([]*trustedaiv1.Model, 0, len(modelsWithCreds))
	for _, modelWithCreds := range modelsWithCreds {
		protoModel := &trustedaiv1.Model{
			Id:             modelWithCreds.Model.ID,
			Name:           modelWithCreds.Model.Name,
			ProviderId:     modelWithCreds.Model.Provider,
			CredentialId:   modelWithCreds.CredentialID.String(),
			CredentialType: modelWithCreds.CredentialType,
			Metadata:       convertGaiMetadataToProto(modelWithCreds.Model.Metadata),
			Enabled:        true,
			CreatedAt:      timestamppb.New(time.Now()),
			UpdatedAt:      timestamppb.New(time.Now()),
		}

		// Add pricing if available
		if modelWithCreds.Model.Pricing.InputTokenPrice > 0 || modelWithCreds.Model.Pricing.OutputTokenPrice > 0 {
			protoModel.Pricing = &trustedaiv1.ModelPricing{
				InputTokenPrice:  modelWithCreds.Model.Pricing.InputTokenPrice,
				OutputTokenPrice: modelWithCreds.Model.Pricing.OutputTokenPrice,
			}
		}

		// Add capabilities
		protoModel.Capabilities = &trustedaiv1.ModelCapabilities{
			SupportsStreaming: modelWithCreds.Model.Capabilities.SupportsStreaming,
			SupportsJson:      modelWithCreds.Model.Capabilities.SupportsJSON,
			SupportsTools:     modelWithCreds.Model.Capabilities.SupportsTools,
			SupportsVision:    modelWithCreds.Model.Capabilities.SupportsVision,
			SupportsReasoning: modelWithCreds.Model.Capabilities.SupportsReasoning,
			MaxInputTokens:    int32(modelWithCreds.Model.Capabilities.MaxInputTokens),
			MaxOutputTokens:   int32(modelWithCreds.Model.Capabilities.MaxOutputTokens),
		}

		protoModels = append(protoModels, protoModel)
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceListModelsResponse{
		Models: protoModels,
	}

	return connect.NewResponse(response), nil
}

// UpdateModel updates an existing model
func (s *ModelManagement) UpdateModel(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceUpdateModelRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceUpdateModelResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] UpdateModel invoked", "id", req.Msg.GetModel().GetId())

	// Validate request
	if req.Msg.GetModel() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model is required"))
	}

	if req.Msg.GetModel().GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model ID is required"))
	}

	if req.Msg.GetModel().GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model name is required"))
	}

	if req.Msg.GetModel().GetProviderId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: provider ID is required"))
	}

	if req.Msg.GetModel().GetCredentialId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential ID is required"))
	}

	if req.Msg.GetModel().GetCredentialType() == trustedaiv1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: credential type is required"))
	}

	// Parse credential ID
	credentialID, err := uuid.Parse(req.Msg.GetModel().GetCredentialId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model management service: invalid credential ID format: %w", err))
	}

	// Convert protobuf model to gai.Model
	gaiModel := protoModelToGaiModel(req.Msg.GetModel())

	// Update model in repository
	err = s.options.ModelRepository.UpdateModel(ctx, gaiModel, credentialID, req.Msg.GetModel().GetCredentialType())
	if err != nil {
		s.options.Logger.Error("Failed to update model", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to update model: %w", err))
	}

	// Convert back to proto for response
	protoModel := gaiModelToProto(gaiModel, req.Msg.GetModel().GetCredentialId(), req.Msg.GetModel().GetCredentialType())

	// Return response
	response := &trustedaiv1.ModelManagementServiceUpdateModelResponse{
		Model: protoModel,
	}

	return connect.NewResponse(response), nil
}

// DeleteModel removes a model
func (s *ModelManagement) DeleteModel(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceDeleteModelRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceDeleteModelResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] DeleteModel invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: model ID is required"))
	}

	// Delete model in repository (soft delete)
	err := s.options.ModelRepository.DeleteModel(ctx, req.Msg.GetId())
	if err != nil {
		s.options.Logger.Error("Failed to delete model", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("model management service: failed to delete model: %w", err))
	}

	// Return response
	response := &trustedaiv1.ModelManagementServiceDeleteModelResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Supported types service methods

// ListSupportedCredentialTypes returns the credential types supported by the system
func (s *ModelManagement) ListSupportedCredentialTypes(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListSupportedCredentialTypesRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListSupportedCredentialTypesResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListSupportedCredentialTypes invoked")

	supportedTypes := []*trustedaiv1.SupportedCredentialType{
		{
			Type:        trustedaiv1.CredentialType_CREDENTIAL_TYPE_OPENROUTER,
			DisplayName: "OpenRouter",
			Description: "OpenRouter API credentials for accessing various LLM providers through OpenRouter's unified API",
		},
	}

	response := &trustedaiv1.ModelManagementServiceListSupportedCredentialTypesResponse{
		CredentialTypes: supportedTypes,
	}

	return connect.NewResponse(response), nil
}

// ListSupportedProviders returns the providers supported by the system
func (s *ModelManagement) ListSupportedProviders(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListSupportedProvidersRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListSupportedProvidersResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListSupportedProviders invoked")

	supportedProviders := []*trustedaiv1.Provider{
		{
			Id:           models.PROVIDER_ID_OPENROUTER,
			Name:         "OpenRouter",
			ProviderType: models.PROVIDER_ID_OPENROUTER,
			Enabled:      true,
			CreatedAt:    timestamppb.New(time.Now()),
			UpdatedAt:    timestamppb.New(time.Now()),
		},
	}

	response := &trustedaiv1.ModelManagementServiceListSupportedProvidersResponse{
		Providers: supportedProviders,
	}

	return connect.NewResponse(response), nil
}

// ListSupportedModelsForProvider returns the models supported for a specific provider
func (s *ModelManagement) ListSupportedModelsForProvider(
	ctx context.Context,
	req *connect.Request[trustedaiv1.ModelManagementServiceListSupportedModelsForProviderRequest],
) (*connect.Response[trustedaiv1.ModelManagementServiceListSupportedModelsForProviderResponse], error) {
	s.options.Logger.Debug("[ModelManagementService] ListSupportedModelsForProvider invoked", "providerId", req.Msg.GetProviderId())

	var supportedModels []*trustedaiv1.Model

	switch req.Msg.GetProviderId() {
	case trustedaiv1.ProviderId_PROVIDER_ID_OPENROUTER:
		for modelID, gaiModel := range models.OpenRouterModels {
			protoModel := &trustedaiv1.Model{
				Id:         modelID,
				Name:       gaiModel.Name,
				ProviderId: gaiModel.Provider,
				Metadata:   convertGaiMetadataToProto(gaiModel.Metadata),
				Enabled:    true,
				CreatedAt:  timestamppb.New(time.Now()),
				UpdatedAt:  timestamppb.New(time.Now()),
			}

			if gaiModel.Pricing.InputTokenPrice > 0 || gaiModel.Pricing.OutputTokenPrice > 0 {
				protoModel.Pricing = &trustedaiv1.ModelPricing{
					InputTokenPrice:  gaiModel.Pricing.InputTokenPrice,
					OutputTokenPrice: gaiModel.Pricing.OutputTokenPrice,
				}
			}

			protoModel.Capabilities = &trustedaiv1.ModelCapabilities{
				SupportsStreaming: gaiModel.Capabilities.SupportsStreaming,
				SupportsJson:      gaiModel.Capabilities.SupportsJSON,
				SupportsTools:     gaiModel.Capabilities.SupportsTools,
				SupportsVision:    gaiModel.Capabilities.SupportsVision,
				SupportsReasoning: gaiModel.Capabilities.SupportsReasoning,
				MaxInputTokens:    int32(gaiModel.Capabilities.MaxInputTokens),
				MaxOutputTokens:   int32(gaiModel.Capabilities.MaxOutputTokens),
			}

			supportedModels = append(supportedModels, protoModel)
		}
	case trustedaiv1.ProviderId_PROVIDER_ID_UNSPECIFIED:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: provider ID must be specified"))
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model management service: unsupported provider ID"))
	}

	response := &trustedaiv1.ModelManagementServiceListSupportedModelsForProviderResponse{
		Models: supportedModels,
	}

	return connect.NewResponse(response), nil
}

// Helper functions

// createModelWithInference creates a gai.Model with automatic inference from hardcoded models
func createModelWithInference(protoModel *trustedaiv1.Model, hardcodedModel *gai.Model, modelReference string) *gai.Model {
	gaiModel := &gai.Model{
		ID:       protoModel.GetId(),
		Provider: protoModel.GetProviderId(),
		Metadata: make(map[string]any),
	}

	// Store model reference in metadata
	gaiModel.Metadata["model_reference"] = modelReference

	// Infer name if not provided
	if protoModel.GetName() != "" {
		gaiModel.Name = protoModel.GetName()
	} else {
		gaiModel.Name = hardcodedModel.Name
	}

	// Infer pricing with partial support
	if protoModel.GetPricing() != nil {
		// Start with hardcoded pricing as base
		gaiModel.Pricing = hardcodedModel.Pricing
		// Override only the fields that were explicitly provided
		if protoModel.GetPricing().GetInputTokenPrice() > 0 {
			gaiModel.Pricing.InputTokenPrice = protoModel.GetPricing().GetInputTokenPrice()
		}
		if protoModel.GetPricing().GetOutputTokenPrice() > 0 {
			gaiModel.Pricing.OutputTokenPrice = protoModel.GetPricing().GetOutputTokenPrice()
		}
	} else {
		gaiModel.Pricing = hardcodedModel.Pricing
	}

	// Infer capabilities with partial support
	if protoModel.GetCapabilities() != nil {
		// Start with hardcoded capabilities as base
		gaiModel.Capabilities = hardcodedModel.Capabilities
		// Override only the fields that were explicitly provided
		if protoModel.GetCapabilities().GetMaxInputTokens() > 0 {
			gaiModel.Capabilities.MaxInputTokens = int(protoModel.GetCapabilities().GetMaxInputTokens())
		}
		if protoModel.GetCapabilities().GetMaxOutputTokens() > 0 {
			gaiModel.Capabilities.MaxOutputTokens = int(protoModel.GetCapabilities().GetMaxOutputTokens())
		}
		// For boolean fields, we need to check if they were explicitly set to true
		// Note: protobuf doesn't distinguish between false and unset for booleans
		if hasExplicitBooleanCapabilities(protoModel.GetCapabilities()) {
			gaiModel.Capabilities.SupportsStreaming = protoModel.GetCapabilities().GetSupportsStreaming()
			gaiModel.Capabilities.SupportsJSON = protoModel.GetCapabilities().GetSupportsJson()
			gaiModel.Capabilities.SupportsTools = protoModel.GetCapabilities().GetSupportsTools()
			gaiModel.Capabilities.SupportsVision = protoModel.GetCapabilities().GetSupportsVision()
			gaiModel.Capabilities.SupportsReasoning = protoModel.GetCapabilities().GetSupportsReasoning()
		}
	} else {
		gaiModel.Capabilities = hardcodedModel.Capabilities
	}

	return gaiModel
}

// hasExplicitBooleanCapabilities checks if any boolean capability fields are explicitly set
// Note: This is a simplified approach - in reality, protobuf doesn't distinguish between
// false and unset for booleans. For proper field presence detection, we'd need to use
// protobuf reflection or optional fields.
func hasExplicitBooleanCapabilities(caps *trustedaiv1.ModelCapabilities) bool {
	// For now, we assume if any boolean is true, they were explicitly set
	// This could be enhanced with field presence detection if needed
	return caps.GetSupportsStreaming() || caps.GetSupportsJson() ||
		caps.GetSupportsTools() || caps.GetSupportsVision() || caps.GetSupportsReasoning()
}

// openRouterCredentialToProto converts a domain OpenRouter credential to a protobuf credential
func openRouterCredentialToProto(credential *trustedai.OpenRouterCredential) *trustedaiv1.OpenRouterCredential {
	return &trustedaiv1.OpenRouterCredential{
		Id:          credential.ID.String(),
		Name:        credential.Name,
		Description: credential.Description,
		ApiKey:      credential.APIKey,
		SiteName:    credential.SiteName,
		HttpReferer: credential.HTTPReferer,
		Enabled:     credential.Enabled,
		CreatedAt:   timestamppb.New(time.Now()), // No created_at in existing struct
		UpdatedAt:   timestamppb.New(time.Now()), // No updated_at in existing struct
	}
}

// protoModelToGaiModel converts a protobuf model to a gai.Model
func protoModelToGaiModel(protoModel *trustedaiv1.Model) *gai.Model {
	model := &gai.Model{
		ID:       protoModel.GetId(),
		Name:     protoModel.GetName(),
		Provider: protoModel.GetProviderId(),
		Metadata: convertProtoMetadataToGai(protoModel.GetMetadata()),
	}

	// Convert pricing
	if protoModel.GetPricing() != nil {
		model.Pricing = gai.ModelPricing{
			InputTokenPrice:  protoModel.GetPricing().GetInputTokenPrice(),
			OutputTokenPrice: protoModel.GetPricing().GetOutputTokenPrice(),
		}
	}

	// Convert capabilities
	if protoModel.GetCapabilities() != nil {
		model.Capabilities = gai.ModelCapabilities{
			SupportsStreaming: protoModel.GetCapabilities().GetSupportsStreaming(),
			SupportsJSON:      protoModel.GetCapabilities().GetSupportsJson(),
			SupportsTools:     protoModel.GetCapabilities().GetSupportsTools(),
			SupportsVision:    protoModel.GetCapabilities().GetSupportsVision(),
			SupportsReasoning: protoModel.GetCapabilities().GetSupportsReasoning(),
			MaxInputTokens:    int(protoModel.GetCapabilities().GetMaxInputTokens()),
			MaxOutputTokens:   int(protoModel.GetCapabilities().GetMaxOutputTokens()),
		}
	}

	return model
}

// gaiModelToProto converts a gai.Model to a protobuf model
func gaiModelToProto(gaiModel *gai.Model, credentialID string, credentialType trustedaiv1.CredentialType) *trustedaiv1.Model {
	protoModel := &trustedaiv1.Model{
		Id:             gaiModel.ID,
		Name:           gaiModel.Name,
		ProviderId:     gaiModel.Provider,
		CredentialId:   credentialID,
		CredentialType: credentialType,
		Metadata:       convertGaiMetadataToProto(gaiModel.Metadata),
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Now()),
		UpdatedAt:      timestamppb.New(time.Now()),
	}

	// Convert pricing
	if gaiModel.Pricing.InputTokenPrice > 0 || gaiModel.Pricing.OutputTokenPrice > 0 {
		protoModel.Pricing = &trustedaiv1.ModelPricing{
			InputTokenPrice:  gaiModel.Pricing.InputTokenPrice,
			OutputTokenPrice: gaiModel.Pricing.OutputTokenPrice,
		}
	}

	// Convert capabilities
	protoModel.Capabilities = &trustedaiv1.ModelCapabilities{
		SupportsStreaming: gaiModel.Capabilities.SupportsStreaming,
		SupportsJson:      gaiModel.Capabilities.SupportsJSON,
		SupportsTools:     gaiModel.Capabilities.SupportsTools,
		SupportsVision:    gaiModel.Capabilities.SupportsVision,
		SupportsReasoning: gaiModel.Capabilities.SupportsReasoning,
		MaxInputTokens:    int32(gaiModel.Capabilities.MaxInputTokens),
		MaxOutputTokens:   int32(gaiModel.Capabilities.MaxOutputTokens),
	}

	return protoModel
}

// convertProtoMetadataToGai converts proto metadata (map[string]string) to gai metadata (map[string]any)
func convertProtoMetadataToGai(protoMetadata map[string]string) map[string]any {
	if protoMetadata == nil {
		return make(map[string]any)
	}

	gaiMetadata := make(map[string]any, len(protoMetadata))
	for key, value := range protoMetadata {
		gaiMetadata[key] = value
	}
	return gaiMetadata
}

// convertGaiMetadataToProto converts gai metadata (map[string]any) to proto metadata (map[string]string)
func convertGaiMetadataToProto(gaiMetadata map[string]any) map[string]string {
	if gaiMetadata == nil {
		return make(map[string]string)
	}

	protoMetadata := make(map[string]string, len(gaiMetadata))
	for key, value := range gaiMetadata {
		// Convert any value to string for proto compatibility
		if strValue, ok := value.(string); ok {
			protoMetadata[key] = strValue
		} else {
			// For non-string values, convert to string representation
			protoMetadata[key] = fmt.Sprintf("%v", value)
		}
	}
	return protoMetadata
}
