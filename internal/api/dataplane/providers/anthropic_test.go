// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codeberg.org/gai-org/gai"
	"github.com/MadsRC/trustedai/internal/api/dataplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProvider()
	assert.Equal(t, "anthropic", provider.Name())
}

func TestAnthropicProvider_Messages(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": "Hello, Claude!"}
		]
	}`

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response AnthropicResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "message", response.Type)
	assert.Equal(t, "assistant", response.Role)
	assert.Equal(t, "claude-sonnet-4-20250514", response.Model)
	assert.Equal(t, "end_turn", response.StopReason)
	assert.Len(t, response.Content, 1)
	assert.Equal(t, "text", response.Content[0].Type)
	assert.Equal(t, "Hello! This is a test response.", response.Content[0].Text)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 10, response.Usage.InputTokens)
	assert.Equal(t, 15, response.Usage.OutputTokens)
}

func TestAnthropicProvider_Messages_WithSystem(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"system": "You are a helpful assistant.",
		"messages": [
			{"role": "user", "content": "Hello!"}
		]
	}`

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response AnthropicResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "message", response.Type)
	assert.Equal(t, "assistant", response.Role)
}

func TestAnthropicProvider_Messages_MultipleContentBlocks(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"messages": [
			{
				"role": "user", 
				"content": [
					{"type": "text", "text": "What's in this image?"},
					{
						"type": "image",
						"source": {
							"type": "base64",
							"media_type": "image/jpeg",
							"data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
						}
					}
				]
			}
		]
	}`

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response AnthropicResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "message", response.Type)
	assert.Equal(t, "assistant", response.Role)
}

func TestAnthropicProvider_Messages_Streaming(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		mockClient         *mockLLMClient
		expectedStatusCode int
		expectedChunks     int
	}{
		{
			name: "successful streaming",
			requestBody: `{
				"model": "claude-sonnet-4-20250514",
				"max_tokens": 1024,
				"stream": true,
				"messages": [
					{"role": "user", "content": "Hello!"}
				]
			}`,
			mockClient:         &mockLLMClient{},
			expectedStatusCode: http.StatusOK,
			expectedChunks:     3,
		},
		{
			name: "streaming with custom chunks",
			requestBody: `{
				"model": "claude-sonnet-4-20250514",
				"max_tokens": 1024,
				"stream": true,
				"messages": [
					{"role": "user", "content": "Test streaming"}
				]
			}`,
			mockClient: &mockLLMClient{
				streamChunks: []*gai.ResponseChunk{
					{
						ID:       "custom-123",
						Delta:    gai.OutputDelta{Text: "Custom"},
						Finished: false,
						Status:   "generating",
					},
					{
						ID:       "custom-123",
						Delta:    gai.OutputDelta{Text: " response"},
						Finished: true,
						Status:   "completed",
						Usage:    &gai.TokenUsage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
					},
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedChunks:     2,
		},
		{
			name: "streaming error from LLM client",
			requestBody: `{
				"model": "claude-sonnet-4-20250514",
				"max_tokens": 1024,
				"stream": true,
				"messages": [
					{"role": "user", "content": "Hello!"}
				]
			}`,
			mockClient:         &mockLLMClient{shouldStreamError: true},
			expectedStatusCode: http.StatusInternalServerError,
			expectedChunks:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAnthropicProvider()
			provider.SetLLMClient(tt.mockClient)

			mux := http.NewServeMux()
			provider.SetupRoutes(mux, nil)

			req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedStatusCode != http.StatusOK {
				return
			}

			assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
			assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
			assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

			body := w.Body.String()
			lines := strings.Split(body, "\n")

			var dataLines []string
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLines = append(dataLines, line)
				}
			}

			assert.Equal(t, tt.expectedChunks, len(dataLines))

			for _, dataLine := range dataLines {
				dataContent := strings.TrimPrefix(dataLine, "data: ")
				var event AnthropicStreamEvent
				err := json.Unmarshal([]byte(dataContent), &event)
				require.NoError(t, err)

				assert.Contains(t, []string{"content_block_delta", "message_stop"}, event.Type)
			}
		})
	}
}

func TestAnthropicProvider_convertChunkToAnthropicEvent(t *testing.T) {
	provider := NewAnthropicProvider()

	tests := []struct {
		name      string
		chunk     *gai.ResponseChunk
		modelID   string
		wantType  string
		wantIndex int
	}{
		{
			name: "regular chunk",
			chunk: &gai.ResponseChunk{
				ID:       "test-123",
				Delta:    gai.OutputDelta{Text: "Hello"},
				Finished: false,
				Status:   "generating",
			},
			modelID:   "claude-sonnet-4-20250514",
			wantType:  "content_block_delta",
			wantIndex: 0,
		},
		{
			name: "final chunk",
			chunk: &gai.ResponseChunk{
				ID:       "test-123",
				Delta:    gai.OutputDelta{Text: "!"},
				Finished: true,
				Status:   "completed",
				Usage:    &gai.TokenUsage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
			},
			modelID:   "claude-sonnet-4-20250514",
			wantType:  "message_stop",
			wantIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertChunkToAnthropicEvent(tt.chunk, tt.modelID, "2023-06-01")

			assert.Equal(t, tt.wantType, result.Type)

			if tt.chunk.Finished {
				assert.Equal(t, "message_stop", result.Type)
				if tt.chunk.Usage != nil {
					assert.NotNil(t, result.Usage)
					assert.Equal(t, tt.chunk.Usage.PromptTokens, result.Usage.InputTokens)
					assert.Equal(t, tt.chunk.Usage.CompletionTokens, result.Usage.OutputTokens)
				}
			} else {
				assert.Equal(t, "content_block_delta", result.Type)
				assert.Equal(t, tt.wantIndex, result.Index)
				assert.NotNil(t, result.Delta)

				delta := result.Delta.(AnthropicStreamDelta)
				assert.Equal(t, "text_delta", delta.Type)
				assert.Equal(t, tt.chunk.Delta.Text, delta.Text)
			}
		})
	}
}

func TestAnthropicProvider_convertToGaiRequest(t *testing.T) {
	provider := NewAnthropicProvider()

	tests := []struct {
		name                 string
		req                  AnthropicRequest
		expectedModelID      string
		expectedInstructions string
		expectedMessagesLen  int
	}{
		{
			name: "simple text message",
			req: AnthropicRequest{
				Model:     "claude-sonnet-4-20250514",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			expectedModelID:      "claude-sonnet-4-20250514",
			expectedInstructions: "",
			expectedMessagesLen:  1,
		},
		{
			name: "with system message",
			req: AnthropicRequest{
				Model:     "claude-sonnet-4-20250514",
				MaxTokens: 1024,
				System:    "You are a helpful assistant.",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			expectedModelID:      "claude-sonnet-4-20250514",
			expectedInstructions: "You are a helpful assistant.",
			expectedMessagesLen:  1,
		},
		{
			name: "conversation with multiple messages",
			req: AnthropicRequest{
				Model:     "claude-sonnet-4-20250514",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
					{Role: "user", Content: "How are you?"},
				},
			},
			expectedModelID:      "claude-sonnet-4-20250514",
			expectedInstructions: "",
			expectedMessagesLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertToGaiRequest(tt.req)

			assert.Equal(t, tt.expectedModelID, result.ModelID)
			assert.Equal(t, tt.expectedInstructions, result.Instructions)
			assert.Equal(t, tt.req.MaxTokens, result.MaxOutputTokens)
			assert.Equal(t, tt.req.Stream, result.Stream)

			if conversation, ok := result.Input.(gai.Conversation); ok {
				assert.Equal(t, tt.expectedMessagesLen, len(conversation.Messages))
			}
		})
	}
}

func TestAnthropicProvider_extractTextFromAnthropicMessage(t *testing.T) {
	provider := NewAnthropicProvider()

	tests := []struct {
		name     string
		message  AnthropicMessage
		expected string
	}{
		{
			name:     "string content",
			message:  AnthropicMessage{Role: "user", Content: "Hello world"},
			expected: "Hello world",
		},
		{
			name: "array content with text blocks",
			message: AnthropicMessage{
				Role: "user",
				Content: []any{
					map[string]any{
						"type": "text",
						"text": "Hello",
					},
					map[string]any{
						"type": "text",
						"text": "world",
					},
				},
			},
			expected: "Hello world",
		},
		{
			name: "array content with mixed types",
			message: AnthropicMessage{
				Role: "user",
				Content: []any{
					map[string]any{
						"type": "text",
						"text": "Look at this image:",
					},
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/jpeg",
							"data":       "base64data",
						},
					},
				},
			},
			expected: "Look at this image:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.extractTextFromAnthropicMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnthropicProvider_Shutdown(t *testing.T) {
	provider := NewAnthropicProvider()
	ctx := context.Background()

	err := provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestAnthropicProvider_WithLogger(t *testing.T) {
	logger := dataplane.WithProviderLogger(nil)
	provider := NewAnthropicProvider(logger)

	assert.Nil(t, provider.options.Logger)
}

func TestAnthropicProvider_InvalidRequestBody(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnthropicProvider_NoLLMClient(t *testing.T) {
	provider := NewAnthropicProvider()

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": "Hello"}
		]
	}`

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnthropicProvider_VersionValidation(t *testing.T) {
	provider := NewAnthropicProvider()

	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{
			name:    "valid version 2023-01-01",
			version: "2023-01-01",
			valid:   true,
		},
		{
			name:    "valid version 2023-06-01",
			version: "2023-06-01",
			valid:   true,
		},
		{
			name:    "invalid version",
			version: "2022-01-01",
			valid:   false,
		},
		{
			name:    "invalid version format",
			version: "invalid",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.isValidVersion(tt.version)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestAnthropicProvider_VersionHeader(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": "Hello"}
		]
	}`

	tests := []struct {
		name               string
		version            string
		expectedStatusCode int
	}{
		{
			name:               "with valid version 2023-06-01",
			version:            "2023-06-01",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "with valid version 2023-01-01",
			version:            "2023-01-01",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "with invalid version",
			version:            "2022-01-01",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "without version header (should use default)",
			version:            "",
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.version != "" {
				req.Header.Set("anthropic-version", tt.version)
			}

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
		})
	}
}

// Tool use tests
func TestAnthropicProvider_ToolUse(t *testing.T) {
	provider := NewAnthropicProvider()

	mockClient := &mockLLMClient{
		toolCallResponse: &gai.Response{
			ID:      "test-tool-123",
			ModelID: "claude-sonnet-4-20250514",
			Status:  "completed",
			Output: []gai.OutputItem{
				gai.ToolCallOutput{
					ID:        "tool-call-123",
					Name:      "get_weather",
					Arguments: `{"location": "San Francisco"}`,
					Status:    "pending",
				},
			},
			Usage:     &gai.TokenUsage{PromptTokens: 20, CompletionTokens: 25, TotalTokens: 45},
			CreatedAt: time.Now(),
		},
	}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	requestBody := `{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"tools": [{
			"name": "get_weather",
			"description": "Get current weather for a location",
			"input_schema": {
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"description": "The city and state"
					}
				},
				"required": ["location"]
			}
		}],
		"messages": [
			{"role": "user", "content": "What's the weather like in San Francisco?"}
		]
	}`

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response AnthropicResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "message", response.Type)
	assert.Equal(t, "assistant", response.Role)
	assert.Len(t, response.Content, 1)
	assert.Equal(t, "tool_use", response.Content[0].Type)
	assert.Equal(t, "tool-call-123", response.Content[0].ID)
	assert.Equal(t, "get_weather", response.Content[0].Name)
	assert.NotNil(t, response.Content[0].Input)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 20, response.Usage.InputTokens)
	assert.Equal(t, 25, response.Usage.OutputTokens)
}

func TestAnthropicProvider_ListModels(t *testing.T) {
	mockRouter := &mockModelRouter{
		models: []gai.Model{
			{
				ID:       "claude-3-5-sonnet-20241022",
				Name:     "Claude 3.5 Sonnet",
				Provider: "anthropic",
				Metadata: map[string]any{
					"created_at": "2024-10-22T00:00:00Z",
				},
			},
			{
				ID:       "claude-3-haiku-20240307",
				Name:     "Claude 3 Haiku",
				Provider: "anthropic",
				Metadata: map[string]any{
					"created_at": "2024-03-07T00:00:00Z",
				},
			},
		},
	}

	provider := NewAnthropicProvider(dataplane.WithModelRouter(mockRouter))

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/anthropic/v1/models", nil)
	req.Header.Set("anthropic-version", "2023-06-01")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response map[string]any
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")
	assert.Contains(t, response, "has_more")
	assert.Equal(t, false, response["has_more"])

	data, ok := response["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 2)

	// Check first model
	model1, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "claude-3-5-sonnet-20241022", model1["id"])
	assert.Equal(t, "model", model1["type"])
	assert.Equal(t, "Claude 3.5 Sonnet", model1["display_name"])
	assert.Equal(t, "2024-10-22T00:00:00Z", model1["created_at"])
}

func TestAnthropicProvider_ListModels_NoModelRouter(t *testing.T) {
	provider := NewAnthropicProvider()

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/anthropic/v1/models", nil)
	req.Header.Set("anthropic-version", "2023-06-01")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnthropicProvider_ListModels_InvalidVersion(t *testing.T) {
	mockRouter := &mockModelRouter{}
	provider := NewAnthropicProvider(dataplane.WithModelRouter(mockRouter))

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/anthropic/v1/models", nil)
	req.Header.Set("anthropic-version", "invalid-version")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
