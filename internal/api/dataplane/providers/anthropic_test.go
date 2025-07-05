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

	"codeberg.org/MadsRC/llmgw/internal/api/dataplane"
	"codeberg.org/gai-org/gai"
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
			result := provider.convertChunkToAnthropicEvent(tt.chunk, tt.modelID)

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
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello",
					},
					map[string]interface{}{
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
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Look at this image:",
					},
					map[string]interface{}{
						"type": "image",
						"source": map[string]interface{}{
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
