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

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider()
	if got := provider.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestOpenAIProvider_ChatCompletions(t *testing.T) {
	provider := NewOpenAIProvider()

	// Set up a mock LLM client
	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("POST", "/openai/v1/chat/completions", strings.NewReader(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["object"] != "chat.completion" {
		t.Errorf("Expected object to be 'chat.completion', got %v", response["object"])
	}

	choices, ok := response["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Errorf("Expected choices to be a non-empty array")
	}
}

func TestOpenAIProvider_ListModels(t *testing.T) {
	mockRouter := &mockModelRouter{
		models: []gai.Model{
			{
				ID:       "gpt-4o",
				Name:     "GPT-4o",
				Provider: "openai",
				Metadata: map[string]any{
					"created_at": "2024-05-13T00:00:00Z",
				},
			},
			{
				ID:       "gpt-3.5-turbo",
				Name:     "GPT-3.5 Turbo",
				Provider: "openai",
				Metadata: map[string]any{
					"created_at": "2023-03-01T00:00:00Z",
				},
			},
		},
	}

	provider := NewOpenAIProvider(dataplane.WithModelRouter(mockRouter))
	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/openai/v1/models", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)

	var response map[string]any
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "list", response["object"])
	assert.Contains(t, response, "data")

	data, ok := response["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 2)

	// Check first model
	model1, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", model1["id"])
	assert.Equal(t, "model", model1["object"])
	assert.Equal(t, "openai", model1["owned_by"])
	assert.Contains(t, model1, "created")
}

func TestOpenAIProvider_Shutdown(t *testing.T) {
	provider := NewOpenAIProvider()
	ctx := context.Background()

	err := provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Unexpected error during shutdown: %v", err)
	}
}

func TestOpenAIProvider_WithLogger(t *testing.T) {
	logger := dataplane.WithProviderLogger(nil)
	provider := NewOpenAIProvider(logger)

	if provider.options.Logger != nil {
		t.Errorf("Expected logger to be nil, got %v", provider.options.Logger)
	}
}

func TestOpenAIProvider_ChatCompletions_Streaming(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		mockClient         *mockLLMClient
		expectedStatusCode int
		expectedChunks     int
		shouldHaveDone     bool
	}{
		{
			name:               "successful streaming",
			requestBody:        `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}],"stream":true}`,
			mockClient:         &mockLLMClient{},
			expectedStatusCode: http.StatusOK,
			expectedChunks:     3,
			shouldHaveDone:     true,
		},
		{
			name:        "streaming with custom chunks",
			requestBody: `{"model":"gpt-4","messages":[{"role":"user","content":"Test"}],"stream":true}`,
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
			shouldHaveDone:     true,
		},
		{
			name:               "streaming error from LLM client",
			requestBody:        `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}],"stream":true}`,
			mockClient:         &mockLLMClient{shouldStreamError: true},
			expectedStatusCode: http.StatusInternalServerError,
			expectedChunks:     0,
			shouldHaveDone:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProvider()
			provider.SetLLMClient(tt.mockClient)

			mux := http.NewServeMux()
			provider.SetupRoutes(mux, nil)

			req := httptest.NewRequest("POST", "/openai/v1/chat/completions", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedStatusCode != http.StatusOK {
				return
			}

			// Check headers for streaming
			assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
			assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
			assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

			// Parse SSE response
			body := w.Body.String()
			lines := strings.Split(body, "\n")

			var dataLines []string
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLines = append(dataLines, line)
				}
			}

			// Should have expected number of chunks plus [DONE]
			expectedDataLines := tt.expectedChunks
			if tt.shouldHaveDone {
				expectedDataLines++
			}
			assert.Equal(t, expectedDataLines, len(dataLines))

			// Check [DONE] message if expected
			if tt.shouldHaveDone {
				assert.Equal(t, "data: [DONE]", dataLines[len(dataLines)-1])
			}

			// Validate chunk format (exclude [DONE])
			for i := range len(dataLines) - 1 {
				dataContent := strings.TrimPrefix(dataLines[i], "data: ")
				var chunk map[string]any
				err := json.Unmarshal([]byte(dataContent), &chunk)
				require.NoError(t, err)

				assert.Equal(t, "chat.completion.chunk", chunk["object"])
				assert.Contains(t, chunk["id"], "chatcmpl-")
				assert.NotNil(t, chunk["choices"])

				choices := chunk["choices"].([]any)
				assert.Len(t, choices, 1)

				choice := choices[0].(map[string]any)
				assert.Equal(t, float64(0), choice["index"])
				assert.NotNil(t, choice["delta"])
			}
		})
	}
}

func TestOpenAIProvider_convertChunkToOpenAI(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name       string
		chunk      *gai.ResponseChunk
		modelID    string
		wantObject string
		wantFinish any
	}{
		{
			name: "regular chunk",
			chunk: &gai.ResponseChunk{
				ID:       "test-123",
				Delta:    gai.OutputDelta{Text: "Hello"},
				Finished: false,
				Status:   "generating",
			},
			modelID:    "gpt-3.5-turbo",
			wantObject: "chat.completion.chunk",
			wantFinish: nil,
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
			modelID:    "gpt-4",
			wantObject: "chat.completion.chunk",
			wantFinish: "stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertChunkToOpenAI(tt.chunk, tt.modelID)

			assert.Equal(t, tt.wantObject, result["object"])
			assert.Equal(t, "chatcmpl-"+tt.chunk.ID, result["id"])
			assert.Equal(t, tt.modelID, result["model"])
			assert.NotNil(t, result["created"])

			choices := result["choices"].([]map[string]any)
			require.Len(t, choices, 1)

			choice := choices[0]
			assert.Equal(t, 0, choice["index"])
			assert.Equal(t, tt.wantFinish, choice["finish_reason"])

			delta := choice["delta"].(map[string]any)
			if tt.chunk.Finished {
				// Final chunk should have empty delta
				assert.Empty(t, delta)
			} else {
				assert.Equal(t, tt.chunk.Delta.Text, delta["content"])
			}

			// Check usage information for final chunk
			if tt.chunk.Usage != nil {
				usage := result["usage"].(map[string]any)
				assert.Equal(t, tt.chunk.Usage.PromptTokens, usage["prompt_tokens"])
				assert.Equal(t, tt.chunk.Usage.CompletionTokens, usage["completion_tokens"])
				assert.Equal(t, tt.chunk.Usage.TotalTokens, usage["total_tokens"])
			}
		})
	}
}

func TestOpenAIProvider_ListModels_NoModelRouter(t *testing.T) {
	provider := NewOpenAIProvider()

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/openai/v1/models", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
