//go:build acceptance

// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1/trustedaiv1connect"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestConfig holds configuration for acceptance tests
type TestConfig struct {
	// ControlPlaneURL is the base URL for the ControlPlane API (ConnectRPC)
	ControlPlaneURL string
	// DataPlaneURL is the base URL for the DataPlane API (HTTP)
	DataPlaneURL string
	// BootstrapToken is the initial API token provided by the test runner
	BootstrapToken string
	// CreatedToken will store the new API token created during tests
	CreatedToken string
	// OpenRouterAPIKey is the API key for OpenRouter provider
	OpenRouterAPIKey string
	// CreatedCredentialID will store the OpenRouter credential ID created during tests
	CreatedCredentialID string
}

// setupTestConfig reads configuration from environment variables
func setupTestConfig(t *testing.T) *TestConfig {
	t.Helper()

	controlPlaneURL := os.Getenv("TRUSTEDAI_CONTROLPLANE_URL")
	if controlPlaneURL == "" {
		controlPlaneURL = "http://localhost:9999" // Default from docker-compose
	}

	dataPlaneURL := os.Getenv("TRUSTEDAI_DATAPLANE_URL")
	if dataPlaneURL == "" {
		dataPlaneURL = "http://localhost:8081" // Default from docker-compose
	}

	bootstrapToken := os.Getenv("TRUSTEDAI_BOOTSTRAP_TOKEN")
	require.NotEmpty(t, bootstrapToken, "TRUSTEDAI_BOOTSTRAP_TOKEN environment variable must be set")

	openRouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	require.NotEmpty(t, openRouterAPIKey, "OPENROUTER_API_KEY environment variable must be set")

	return &TestConfig{
		ControlPlaneURL:  controlPlaneURL,
		DataPlaneURL:     dataPlaneURL,
		BootstrapToken:   bootstrapToken,
		OpenRouterAPIKey: openRouterAPIKey,
	}
}

// createAuthenticatedClient creates a ConnectRPC client with bearer authentication
func createAuthenticatedClient(config *TestConfig, token string) trustedaiv1connect.IAMServiceClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Add bearer token interceptor
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
			return next(ctx, req)
		})
	}

	client := trustedaiv1connect.NewIAMServiceClient(
		httpClient,
		config.ControlPlaneURL,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(interceptor)),
	)

	return client
}

// createAuthenticatedModelManagementClient creates a ModelManagementService client with bearer authentication
func createAuthenticatedModelManagementClient(config *TestConfig, token string) trustedaiv1connect.ModelManagementServiceClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Add bearer token interceptor
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
			return next(ctx, req)
		})
	}

	client := trustedaiv1connect.NewModelManagementServiceClient(
		httpClient,
		config.ControlPlaneURL,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(interceptor)),
	)

	return client
}

// loadKeycloakConfig loads and parses the Keycloak realm configuration file
func loadKeycloakConfig(t *testing.T) map[string]interface{} {
	t.Helper()

	// Get the path to the Keycloak config file relative to the test
	configPath := filepath.Join("..", "keycloak", "testrealm02-realm.json")

	// Read the file
	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "Failed to read Keycloak config file")

	// Parse JSON
	var config map[string]interface{}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err, "Failed to parse Keycloak config JSON")

	return config
}

// getClientSecretByID extracts the client secret for a given client ID from Keycloak config
func getClientSecretByID(t *testing.T, config map[string]interface{}, clientID string) string {
	t.Helper()

	clients, ok := config["clients"].([]interface{})
	require.True(t, ok, "Failed to get clients array from Keycloak config")

	for _, clientInterface := range clients {
		client, ok := clientInterface.(map[string]interface{})
		if !ok {
			continue
		}

		if id, exists := client["id"]; exists && id == clientID {
			if secret, exists := client["secret"]; exists {
				secretStr, ok := secret.(string)
				require.True(t, ok, "Client secret is not a string")
				return secretStr
			}
		}
	}

	t.Fatalf("Client with ID %s not found in Keycloak config", clientID)
	return ""
}

// TestTrustedAIAcceptance is the main acceptance test suite
func TestTrustedAIAcceptance(t *testing.T) {
	config := setupTestConfig(t)

	t.Run("ControlPlane", func(t *testing.T) {
		t.Run("Authentication", func(t *testing.T) {
			testControlPlaneAuthentication(t, config)
		})

		t.Run("CreateAPIKey", func(t *testing.T) {
			testCreateAPIKey(t, config)
		})

		t.Run("UserManagement", func(t *testing.T) {
			testUserManagement(t, config)
		})

		t.Run("OrganizationManagement", func(t *testing.T) {
			testOrganizationManagement(t, config)
		})

		t.Run("TokenManagement", func(t *testing.T) {
			testTokenManagement(t, config)
		})

		t.Run("CreateOpenRouterCredential", func(t *testing.T) {
			testCreateOpenRouterCredential(t, config)
		})

		t.Run("CreateModel", func(t *testing.T) {
			testCreateModel(t, config)
		})

		t.Run("OIDCConfiguration", func(t *testing.T) {
			testOIDCConfiguration(t, config)
		})
	})

	t.Run("DataPlane", func(t *testing.T) {
		t.Run("Authentication", func(t *testing.T) {
			testDataPlaneAuthentication(t, config)
		})

		t.Run("OpenAICompatibility", func(t *testing.T) {
			testOpenAICompatibility(t, config)
		})

		t.Run("AnthropicCompatibility", func(t *testing.T) {
			testAnthropicCompatibility(t, config)
		})
	})
}

// testControlPlaneAuthentication verifies that the bootstrap token works for ControlPlane API
func testControlPlaneAuthentication(t *testing.T, config *TestConfig) {
	client := createAuthenticatedClient(config, config.BootstrapToken)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get current user to verify authentication
	resp, err := client.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
	require.NoError(t, err, "Failed to authenticate with bootstrap token")

	user := resp.Msg.User
	require.NotNil(t, user)
	assert.Equal(t, "admin@localhost", user.Email)
	assert.True(t, user.SystemAdmin)
	assert.Equal(t, "System Administrator", user.Name)
}

// testCreateAPIKey creates a new API key for the bootstrap user and stores it for later tests
func testCreateAPIKey(t *testing.T, config *TestConfig) {
	client := createAuthenticatedClient(config, config.BootstrapToken)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First get the current user to get their ID
	userResp, err := client.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
	require.NoError(t, err)

	userID := userResp.Msg.User.Id

	// Create a new API token
	expiresAt := time.Now().Add(1 * time.Hour) // 1 hour expiry for testing
	tokenReq := &trustedaiv1.IAMServiceCreateTokenRequest{
		UserId:      userID,
		Description: "Acceptance Test Token",
		ExpiresAt:   timestamppb.New(expiresAt),
	}

	tokenResp, err := client.CreateToken(ctx, connect.NewRequest(tokenReq))
	require.NoError(t, err, "Failed to create new API token")

	token := tokenResp.Msg.Token
	rawToken := tokenResp.Msg.RawToken

	// Verify token properties
	assert.Equal(t, userID, token.UserId)
	assert.Equal(t, "Acceptance Test Token", token.Description)
	assert.NotEmpty(t, token.Id)
	assert.NotEmpty(t, rawToken)
	assert.True(t, token.ExpiresAt.AsTime().After(time.Now()))

	// Store the created token for use in subsequent tests
	config.CreatedToken = rawToken

	// Verify the new token works by making a request with it
	newClient := createAuthenticatedClient(config, rawToken)
	verifyResp, err := newClient.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
	require.NoError(t, err, "New token should work for authentication")
	assert.Equal(t, userID, verifyResp.Msg.User.Id)
}

// testUserManagement tests user-related operations
func testUserManagement(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping user management tests - no created token available")
	}

	client := createAuthenticatedClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("GetCurrentUser", func(t *testing.T) {
		resp, err := client.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
		require.NoError(t, err)

		user := resp.Msg.User
		assert.NotEmpty(t, user.Id)
		assert.Equal(t, "admin@localhost", user.Email)
		assert.True(t, user.SystemAdmin)
	})

	t.Run("ListUsersByOrganization", func(t *testing.T) {
		// First get current user to get their organization ID
		userResp, err := client.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
		require.NoError(t, err)

		orgID := userResp.Msg.User.OrganizationId

		// List users in the organization
		listResp, err := client.ListUsersByOrganization(ctx, connect.NewRequest(&trustedaiv1.IAMServiceListUsersByOrganizationRequest{
			OrganizationId: orgID,
		}))
		require.NoError(t, err)

		users := listResp.Msg.Users
		assert.Len(t, users, 1, "Should have exactly one user (the bootstrap admin)")
		assert.Equal(t, "admin@localhost", users[0].Email)
	})
}

// testOrganizationManagement tests organization-related operations
func testOrganizationManagement(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping organization management tests - no created token available")
	}

	client := createAuthenticatedClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("ListOrganizations", func(t *testing.T) {
		resp, err := client.ListOrganizations(ctx, connect.NewRequest(&trustedaiv1.IAMServiceListOrganizationsRequest{}))
		require.NoError(t, err)

		orgs := resp.Msg.Organizations
		require.Len(t, orgs, 1, "Should have exactly one organization (system)")

		systemOrg := orgs[0]
		assert.Equal(t, "system", systemOrg.Name)
		assert.Equal(t, "System Administration", systemOrg.DisplayName)
		assert.True(t, systemOrg.IsSystem)
	})

	t.Run("GetOrganizationByName", func(t *testing.T) {
		resp, err := client.GetOrganizationByName(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetOrganizationByNameRequest{
			Name: "system",
		}))
		require.NoError(t, err)

		org := resp.Msg.Organization
		assert.Equal(t, "system", org.Name)
		assert.True(t, org.IsSystem)
	})
}

// testTokenManagement tests token-related operations
func testTokenManagement(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping token management tests - no created token available")
	}

	client := createAuthenticatedClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("ListUserTokens", func(t *testing.T) {
		// Get current user ID
		userResp, err := client.GetCurrentUser(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetCurrentUserRequest{}))
		require.NoError(t, err)

		userID := userResp.Msg.User.Id

		// List tokens for the user
		tokensResp, err := client.ListUserTokens(ctx, connect.NewRequest(&trustedaiv1.IAMServiceListUserTokensRequest{
			UserId: userID,
		}))
		require.NoError(t, err)

		tokens := tokensResp.Msg.Tokens
		assert.GreaterOrEqual(t, len(tokens), 1, "Should have at least one token")

		// Find our acceptance test token
		var testToken *trustedaiv1.APIToken
		for _, token := range tokens {
			if token.Description == "Acceptance Test Token" {
				testToken = token
				break
			}
		}

		require.NotNil(t, testToken, "Should find the acceptance test token")
		assert.Equal(t, userID, testToken.UserId)
	})
}

// testDataPlaneAuthentication verifies that API keys work with the DataPlane
func testDataPlaneAuthentication(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping DataPlane authentication tests - no created token available")
	}

	// Test health/status endpoint with authentication
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", config.DataPlaneURL+"/health", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.CreatedToken))

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Health endpoint should be accessible with valid token
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
		"Health endpoint should be accessible or return 404 if not implemented")
}

// testOpenAICompatibility tests OpenAI-compatible endpoints
func testOpenAICompatibility(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping OpenAI compatibility tests - no created token available")
	}

	t.Run("ChatCompletions", func(t *testing.T) {
		testOpenAIChatCompletions(t, config)
	})

	t.Run("Responses", func(t *testing.T) {
		testOpenAIResponses(t, config)
	})
}

// testOpenAIChatCompletions tests the OpenAI chat completions API endpoint using the official OpenAI SDK
func testOpenAIChatCompletions(t *testing.T, config *TestConfig) {
	// Create OpenAI client configured to use our DataPlane URL
	client := openai.NewClient(
		option.WithAPIKey(config.CreatedToken),
		option.WithBaseURL(config.DataPlaneURL+"/openai/v1"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chat completion request
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant."),
			openai.UserMessage("Who won the world series in 2020?"),
			openai.AssistantMessage("The Los Angeles Dodgers won the World Series in 2020."),
			openai.UserMessage("Where was it played?"),
		},
		Model: "gemini-2.5-flash-lite",
	})
	require.NoError(t, err, "Failed to create chat completion")

	// Verify response structure
	assert.NotEmpty(t, chatCompletion.ID, "Response should have an ID")
	assert.Equal(t, "chat.completion", string(chatCompletion.Object), "Object should be 'chat.completion'")
	assert.NotZero(t, chatCompletion.Created, "Response should have a created timestamp")
	assert.Equal(t, "gemini-2.5-flash-lite", chatCompletion.Model, "Model should match the requested model")
	assert.Greater(t, len(chatCompletion.Choices), 0, "Should have at least one choice")

	// Verify first choice
	choice := chatCompletion.Choices[0]
	assert.Equal(t, int64(0), choice.Index, "First choice should have index 0")
	assert.NotEmpty(t, choice.Message.Content, "Message content should not be empty")
	assert.Equal(t, "assistant", string(choice.Message.Role), "Message role should be assistant")
	assert.NotEmpty(t, choice.FinishReason, "Choice should have a finish reason")

	// Verify usage information if present
	if chatCompletion.Usage.TotalTokens > 0 {
		assert.GreaterOrEqual(t, chatCompletion.Usage.PromptTokens, int64(0), "Prompt tokens should be non-negative")
		assert.GreaterOrEqual(t, chatCompletion.Usage.CompletionTokens, int64(0), "Completion tokens should be non-negative")
		assert.GreaterOrEqual(t, chatCompletion.Usage.TotalTokens, int64(0), "Total tokens should be non-negative")
		assert.Equal(t, chatCompletion.Usage.PromptTokens+chatCompletion.Usage.CompletionTokens, chatCompletion.Usage.TotalTokens, "Total tokens should equal prompt + completion tokens")
	}
}

// testOpenAIResponses tests the OpenAI responses API endpoint using the official OpenAI SDK
func testOpenAIResponses(t *testing.T, config *TestConfig) {
	// Create OpenAI client configured to use our DataPlane URL
	client := openai.NewClient(
		option.WithAPIKey(config.CreatedToken),
		option.WithBaseURL(config.DataPlaneURL+"/openai/v1"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create responses request
	response, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: []responses.ResponseInputItemUnionParam{
				responses.ResponseInputItemParamOfMessage("Who won the world series in 2020?", responses.EasyInputMessageRoleUser),
			},
		},
		Model: "gemini-2.5-flash-lite",
	})
	require.NoError(t, err, "Failed to create response")

	// Verify response structure
	assert.NotEmpty(t, response.ID, "Response should have an ID")
	assert.Equal(t, "response", response.Object, "Object should be 'response'")
	assert.NotZero(t, response.CreatedAt, "Response should have a created timestamp")
	assert.Equal(t, "gemini-2.5-flash-lite", response.Model, "Model should match the requested model")
	assert.Greater(t, len(response.Output), 0, "Should have at least one output item")

	// Verify first output item is a message
	if len(response.Output) > 0 {
		outputItem := response.Output[0]
		if outputMsg := outputItem.AsMessage(); outputMsg.ID != "" {
			assert.NotEmpty(t, outputMsg.Content, "Message content should not be empty")
			assert.Equal(t, "assistant", string(outputMsg.Role), "Message role should be assistant")
			assert.Equal(t, responses.ResponseOutputMessageStatusCompleted, outputMsg.Status, "Message should be completed")
		}
	}

	// Verify usage information if present
	if response.Usage.TotalTokens > 0 {
		assert.GreaterOrEqual(t, response.Usage.InputTokens, int64(0), "Input tokens should be non-negative")
		assert.GreaterOrEqual(t, response.Usage.OutputTokens, int64(0), "Output tokens should be non-negative")
		assert.GreaterOrEqual(t, response.Usage.TotalTokens, int64(0), "Total tokens should be non-negative")
		assert.Equal(t, response.Usage.InputTokens+response.Usage.OutputTokens, response.Usage.TotalTokens, "Total tokens should equal input + output tokens")
	}
}

// testAnthropicCompatibility tests Anthropic-compatible endpoints
func testAnthropicCompatibility(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping Anthropic compatibility tests - no created token available")
	}
	t.Skip("Not yet implemented")
}

// testCreateOpenRouterCredential tests creating an OpenRouter credential
func testCreateOpenRouterCredential(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping OpenRouter credential tests - no created token available")
	}

	client := createAuthenticatedModelManagementClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new OpenRouter credential
	credentialReq := &trustedaiv1.ModelManagementServiceCreateOpenRouterCredentialRequest{
		Credential: &trustedaiv1.OpenRouterCredential{
			Name:        "Test OpenRouter Credential",
			Description: "Acceptance test OpenRouter credential",
			ApiKey:      config.OpenRouterAPIKey,
			SiteName:    "TrustedAI Test",
			HttpReferer: "https://trustedai-test.example.com",
			Enabled:     true,
		},
	}

	credentialResp, err := client.CreateOpenRouterCredential(ctx, connect.NewRequest(credentialReq))
	require.NoError(t, err, "Failed to create OpenRouter credential")

	credential := credentialResp.Msg.Credential
	require.NotNil(t, credential)

	// Verify credential properties
	assert.Equal(t, "Test OpenRouter Credential", credential.Name)
	assert.Equal(t, "Acceptance test OpenRouter credential", credential.Description)
	assert.Equal(t, config.OpenRouterAPIKey, credential.ApiKey)
	assert.Equal(t, "TrustedAI Test", credential.SiteName)
	assert.Equal(t, "https://trustedai-test.example.com", credential.HttpReferer)
	assert.True(t, credential.Enabled)
	assert.NotEmpty(t, credential.Id)
	assert.NotNil(t, credential.CreatedAt)
	assert.NotNil(t, credential.UpdatedAt)

	// Verify we can retrieve the credential
	getReq := &trustedaiv1.ModelManagementServiceGetOpenRouterCredentialRequest{
		Id: credential.Id,
	}
	getResp, err := client.GetOpenRouterCredential(ctx, connect.NewRequest(getReq))
	require.NoError(t, err, "Failed to retrieve OpenRouter credential")

	retrievedCredential := getResp.Msg.Credential
	assert.Equal(t, credential.Id, retrievedCredential.Id)
	assert.Equal(t, credential.Name, retrievedCredential.Name)
	assert.Equal(t, credential.Description, retrievedCredential.Description)
	assert.Equal(t, credential.ApiKey, retrievedCredential.ApiKey)

	// Verify the credential appears in the list
	listReq := &trustedaiv1.ModelManagementServiceListOpenRouterCredentialsRequest{
		IncludeDisabled: true,
	}
	listResp, err := client.ListOpenRouterCredentials(ctx, connect.NewRequest(listReq))
	require.NoError(t, err, "Failed to list OpenRouter credentials")

	credentials := listResp.Msg.Credentials
	assert.GreaterOrEqual(t, len(credentials), 1, "Should have at least one OpenRouter credential")

	// Find our test credential
	var foundCredential *trustedaiv1.OpenRouterCredential
	for _, cred := range credentials {
		if cred.Id == credential.Id {
			foundCredential = cred
			break
		}
	}

	require.NotNil(t, foundCredential, "Should find the test credential in the list")
	assert.Equal(t, credential.Name, foundCredential.Name)

	// Store the credential ID for use in subsequent tests
	config.CreatedCredentialID = credential.Id
}

// testCreateModel tests creating a model using the OpenRouter credential
func testCreateModel(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping model creation tests - no created token available")
	}
	if config.CreatedCredentialID == "" {
		t.Skip("Skipping model creation tests - no created credential ID available")
	}

	client := createAuthenticatedModelManagementClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, let's get the OpenRouter provider ID by listing supported providers
	providersResp, err := client.ListSupportedProviders(ctx, connect.NewRequest(&trustedaiv1.ModelManagementServiceListSupportedProvidersRequest{}))
	require.NoError(t, err, "Failed to list supported providers")

	var openRouterProviderID string
	for _, provider := range providersResp.Msg.Providers {
		if provider.ProviderType == "openrouter" {
			openRouterProviderID = provider.Id
			break
		}
	}
	require.NotEmpty(t, openRouterProviderID, "Should find OpenRouter provider")

	// Create the model metadata with the model_reference
	metadata := map[string]string{
		"model_reference": "openrouter:google/gemini-2.5-flash-lite",
	}

	// Create a new model
	modelReq := &trustedaiv1.ModelManagementServiceCreateModelRequest{
		Model: &trustedaiv1.Model{
			Id:             "gemini-2.5-flash-lite", // Set model ID
			Name:           "gemini-2.5-flash-lite",
			ProviderId:     openRouterProviderID,
			CredentialId:   config.CreatedCredentialID,
			CredentialType: trustedaiv1.CredentialType_CREDENTIAL_TYPE_OPENROUTER,
			Metadata:       metadata,
			Enabled:        true,
		},
	}

	modelResp, err := client.CreateModel(ctx, connect.NewRequest(modelReq))
	require.NoError(t, err, "Failed to create model")

	model := modelResp.Msg.Model
	require.NotNil(t, model)

	// Verify model properties
	assert.Equal(t, "gemini-2.5-flash-lite", model.Name)
	assert.Equal(t, openRouterProviderID, model.ProviderId)
	assert.Equal(t, config.CreatedCredentialID, model.CredentialId)
	assert.Equal(t, trustedaiv1.CredentialType_CREDENTIAL_TYPE_OPENROUTER, model.CredentialType)
	assert.True(t, model.Enabled)
	assert.NotEmpty(t, model.Id)
	assert.NotNil(t, model.CreatedAt)
	assert.NotNil(t, model.UpdatedAt)

	// Verify metadata contains our model_reference
	require.NotNil(t, model.Metadata)
	assert.Equal(t, "openrouter:google/gemini-2.5-flash-lite", model.Metadata["model_reference"])

	// Verify we can retrieve the model
	getReq := &trustedaiv1.ModelManagementServiceGetModelRequest{
		Id: model.Id,
	}
	getResp, err := client.GetModel(ctx, connect.NewRequest(getReq))
	require.NoError(t, err, "Failed to retrieve model")

	retrievedModel := getResp.Msg.Model
	assert.Equal(t, model.Id, retrievedModel.Id)
	assert.Equal(t, model.Name, retrievedModel.Name)
	assert.Equal(t, model.ProviderId, retrievedModel.ProviderId)
	assert.Equal(t, model.CredentialId, retrievedModel.CredentialId)

	// Verify the model appears in the list
	listReq := &trustedaiv1.ModelManagementServiceListModelsRequest{
		IncludeDisabled: true,
	}
	listResp, err := client.ListModels(ctx, connect.NewRequest(listReq))
	require.NoError(t, err, "Failed to list models")

	models := listResp.Msg.Models
	assert.GreaterOrEqual(t, len(models), 1, "Should have at least one model")

	// Find our test model
	var foundModel *trustedaiv1.Model
	for _, m := range models {
		if m.Id == model.Id {
			foundModel = m
			break
		}
	}

	require.NotNil(t, foundModel, "Should find the test model in the list")
	assert.Equal(t, model.Name, foundModel.Name)
}

// testOIDCConfiguration tests setting up OIDC for the system organization
func testOIDCConfiguration(t *testing.T, config *TestConfig) {
	if config.CreatedToken == "" {
		t.Skip("Skipping OIDC configuration tests - no created token available")
	}

	client := createAuthenticatedClient(config, config.CreatedToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get the system organization
	orgResp, err := client.GetOrganizationByName(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetOrganizationByNameRequest{
		Name: "system",
	}))
	require.NoError(t, err, "Failed to get system organization")

	systemOrg := orgResp.Msg.Organization
	require.NotNil(t, systemOrg)
	assert.Equal(t, "system", systemOrg.Name)

	// Load Keycloak configuration and extract client secret
	keycloakConfig := loadKeycloakConfig(t)
	clientSecret := getClientSecretByID(t, keycloakConfig, "39df7920-c02e-4d27-8d3f-018c290bc616")

	// Create OIDC configuration
	oidcConfig := map[string]any{
		"issuer":        "http://keycloak:8080/realms/testrealm02",
		"client_id":     "client01",
		"client_secret": clientSecret,
	}

	// Convert config to JSON string
	configJSON, err := json.Marshal(oidcConfig)
	require.NoError(t, err, "Failed to marshal OIDC config")

	// Update the organization with OIDC configuration
	updatedOrg := &trustedaiv1.Organization{
		Id:          systemOrg.Id,
		Name:        systemOrg.Name,
		DisplayName: systemOrg.DisplayName,
		IsSystem:    systemOrg.IsSystem,
		CreatedAt:   systemOrg.CreatedAt,
		SsoType:     "oidc",
		SsoConfig:   string(configJSON),
	}

	updateResp, err := client.UpdateOrganization(ctx, connect.NewRequest(&trustedaiv1.IAMServiceUpdateOrganizationRequest{
		Organization: updatedOrg,
		HasIsSystem:  false, // Don't change is_system flag
	}))
	require.NoError(t, err, "Failed to update organization with OIDC config")

	// Verify the organization was updated correctly
	updatedOrgResult := updateResp.Msg.Organization
	assert.Equal(t, "oidc", updatedOrgResult.SsoType)
	assert.Equal(t, string(configJSON), updatedOrgResult.SsoConfig)

	// Verify we can retrieve the organization and it has the correct config
	verifyResp, err := client.GetOrganizationByName(ctx, connect.NewRequest(&trustedaiv1.IAMServiceGetOrganizationByNameRequest{
		Name: "system",
	}))
	require.NoError(t, err, "Failed to retrieve updated organization")

	verifiedOrg := verifyResp.Msg.Organization
	assert.Equal(t, "oidc", verifiedOrg.SsoType)
	assert.Equal(t, string(configJSON), verifiedOrg.SsoConfig)

	// Parse and verify the SSO config contains expected values
	var parsedConfig map[string]any
	err = json.Unmarshal([]byte(verifiedOrg.SsoConfig), &parsedConfig)
	require.NoError(t, err, "Failed to parse SSO config JSON")

	assert.Equal(t, "http://keycloak:8080/realms/testrealm02", parsedConfig["issuer"])
	assert.Equal(t, "client01", parsedConfig["client_id"])
	assert.Equal(t, clientSecret, parsedConfig["client_secret"])
}
