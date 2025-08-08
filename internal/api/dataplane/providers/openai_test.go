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

	"codeberg.org/gai-org/gai"
	"github.com/MadsRC/trustedai/internal/api/dataplane"
	"github.com/openai/openai-go/responses"
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

func TestOpenAIProvider_ConvertResponseToGaiRequest(t *testing.T) {
	provider := NewOpenAIProvider()

	testCases := []struct {
		name             string
		inputJSON        string
		isStream         bool
		expectedGaiReq   gai.GenerateRequest
		expectError      bool
		errorDescription string
	}{
		{
			name:      "simple string input",
			inputJSON: `{"model": "gpt-4o", "input": "Hello world"}`,
			isStream:  false,
			expectedGaiReq: gai.GenerateRequest{
				ModelID: "gpt-4o",
				Input:   gai.TextInput{Text: "Hello world"},
				Stream:  false,
			},
			expectError: false,
		},
		{
			name: "message array input with string",
			inputJSON: `{
				"model": "gpt-4o",
				"input": [
					{
						"type": "message",
						"role": "user",
						"content": "Hello"
					}
				]
			}`,
			isStream: false,
			expectedGaiReq: gai.GenerateRequest{
				ModelID: "gpt-4o",
				Input: gai.Conversation{
					Messages: []gai.Message{
						{
							Role:    gai.RoleUser,
							Content: gai.TextInput{Text: "Hello"},
						},
					},
				},
				Stream: false,
			},
			expectError: false,
		},
		{
			name: "message array input",
			inputJSON: `{
				"model": "gpt-4o",
				"input": [
					{
						"type": "message",
						"role": "user",
						"content": [{"type": "input_text", "text": "Hello"}]
					}
				]
			}`,
			isStream: false,
			expectedGaiReq: gai.GenerateRequest{
				ModelID: "gpt-4o",
				Input: gai.Conversation{
					Messages: []gai.Message{
						{
							Role:    gai.RoleUser,
							Content: gai.TextInput{Text: "Hello"},
						},
					},
				},
				Stream: false,
			},
			expectError: false,
		},
		{
			name: "with instructions and parameters",
			inputJSON: `{
				"model": "gpt-4o",
				"input": "Test",
				"instructions": "You are a helpful assistant",
				"temperature": 0.7,
				"max_output_tokens": 100
			}`,
			isStream: true,
			expectedGaiReq: gai.GenerateRequest{
				ModelID:         "gpt-4o",
				Input:           gai.TextInput{Text: "Test"},
				Instructions:    "You are a helpful assistant",
				Temperature:     0.7,
				MaxOutputTokens: 100,
				Stream:          true,
			},
			expectError: false,
		},
		{
			name: "image input",
			inputJSON: `{
				"model": "gpt-4o",
				"input": [
					{
						"type": "message",
						"role": "user", 
						"content": [
							{"type": "input_text", "text": "What's in this image?"},
							{"type": "input_image", "image_url": "https://example.com/image.jpg", "detail": "auto"}
						]
					}
				]
			}`,
			isStream: false,
			expectedGaiReq: gai.GenerateRequest{
				ModelID: "gpt-4o",
				Input: gai.Conversation{
					Messages: []gai.Message{
						{
							Role:    gai.RoleUser,
							Content: gai.TextInput{Text: "What's in this image?"},
						},
						{
							Role:    gai.RoleUser,
							Content: gai.ImageInput{URL: "https://example.com/image.jpg", Detail: "auto"},
						},
					},
				},
				Stream: false,
			},
			expectError: false,
		},
		{
			name:      "text_input_no_message_type",
			inputJSON: `{"input":[{"content":"Who won the world series in 2020?","role":"user"}],"model":"gpt-4o"}`,
			isStream:  false,
			expectedGaiReq: gai.GenerateRequest{
				ModelID: "gpt-4o",
				Input: gai.Conversation{
					Messages: []gai.Message{
						{
							Role:    gai.RoleUser,
							Content: gai.TextInput{Text: "Who won the world series in 2020?"},
						},
					},
				},
				Stream: false,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Preprocess JSON to handle shorthand message format (like the real handler does)
			var rawReq map[string]any
			err := json.Unmarshal([]byte(tc.inputJSON), &rawReq)
			require.NoError(t, err, "Failed to unmarshal raw JSON")

			provider.preprocessResponseInput(rawReq)

			processedJSON, err := json.Marshal(rawReq)
			require.NoError(t, err, "Failed to marshal preprocessed JSON")

			// Parse JSON into ResponseNewParams
			var req responses.ResponseNewParams
			err = json.Unmarshal(processedJSON, &req)
			if tc.expectError {
				require.Error(t, err, tc.errorDescription)
				return
			}
			require.NoError(t, err, "Failed to unmarshal test JSON")

			// Make sure data isn't lost during round-trip (using processed JSON)
			d, err := json.Marshal(req)
			require.NoError(t, err)
			require.JSONEq(t, string(processedJSON), string(d))

			// Convert to GAI request
			gaiReq := provider.convertResponseToGaiRequest(req, tc.isStream)

			// Compare with expected result
			assert.Equal(t, tc.expectedGaiReq.ModelID, gaiReq.ModelID, "ModelID mismatch")
			assert.Equal(t, tc.expectedGaiReq.Instructions, gaiReq.Instructions, "Instructions mismatch")
			assert.Equal(t, tc.expectedGaiReq.Stream, gaiReq.Stream, "Stream mismatch")
			assert.Equal(t, tc.expectedGaiReq.Temperature, gaiReq.Temperature, "Temperature mismatch")
			assert.Equal(t, tc.expectedGaiReq.TopP, gaiReq.TopP, "TopP mismatch")
			assert.Equal(t, tc.expectedGaiReq.MaxOutputTokens, gaiReq.MaxOutputTokens, "MaxOutputTokens mismatch")

			// Compare Input (needs special handling for different types)
			assert.Equal(t, tc.expectedGaiReq.Input, gaiReq.Input, "Input mismatch")

			// Compare Tools if present
			if len(tc.expectedGaiReq.Tools) > 0 || len(gaiReq.Tools) > 0 {
				assert.Equal(t, tc.expectedGaiReq.Tools, gaiReq.Tools, "Tools mismatch")
			}

			// Compare ToolChoice if present
			if tc.expectedGaiReq.ToolChoice != nil || gaiReq.ToolChoice != nil {
				assert.Equal(t, tc.expectedGaiReq.ToolChoice, gaiReq.ToolChoice, "ToolChoice mismatch")
			}
		})
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
