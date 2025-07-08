// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codeberg.org/gai-org/gai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_CreateResponse_NonStreaming(t *testing.T) {
	provider := NewOpenAIProvider()
	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	reqBody := CreateResponseRequest{
		Model:           "gpt-4",
		Input:           "Hello, how are you?",
		Instructions:    "Be helpful and friendly",
		Temperature:     toPtr(0.7),
		MaxOutputTokens: toPtr(100),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/openai/v1/responses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response Response
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "response", response.Object)
	assert.Equal(t, ResponseStatusCompleted, response.Status)
	assert.Equal(t, "gpt-4", response.Model)
	assert.NotEmpty(t, response.ID)
	assert.NotZero(t, response.CreatedAt)
	assert.Len(t, response.Output, 1)
	assert.NotNil(t, response.OutputText)
	assert.Equal(t, "Hello! This is a test response.", *response.OutputText)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 10, response.Usage.InputTokens)
	assert.Equal(t, 15, response.Usage.OutputTokens)
	assert.Equal(t, 25, response.Usage.TotalTokens)
}

func TestOpenAIProvider_CreateResponse_WithInputItems(t *testing.T) {
	provider := NewOpenAIProvider()
	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	messageContent, _ := json.Marshal(map[string]string{
		"role":    "user",
		"content": "What is the weather like?",
	})

	reqBody := CreateResponseRequest{
		Model: "gpt-4",
		InputItems: []InputItem{
			{
				Type:    InputItemTypeMessage,
				Content: messageContent,
			},
		},
		Instructions: "Be helpful",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/openai/v1/responses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, ResponseStatusCompleted, response.Status)
	assert.Equal(t, "gpt-4", response.Model)
}

func TestOpenAIProvider_CreateResponse_Streaming(t *testing.T) {
	provider := NewOpenAIProvider()
	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	reqBody := CreateResponseRequest{
		Model:  "gpt-4",
		Input:  "Tell me a story",
		Stream: toPtr(true),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/openai/v1/responses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "data:")
	assert.Contains(t, responseBody, "[DONE]")

	lines := strings.Split(responseBody, "\n")
	var events []string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") && !strings.Contains(line, "[DONE]") {
			events = append(events, strings.TrimPrefix(line, "data: "))
		}
	}

	assert.True(t, len(events) > 0, "Should have streaming events")
}

func TestOpenAIProvider_CreateResponse_InvalidJSON(t *testing.T) {
	provider := NewOpenAIProvider()
	mockClient := &mockLLMClient{}
	provider.SetLLMClient(mockClient)

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("POST", "/openai/v1/responses", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOpenAIProvider_CreateResponse_NoLLMClient(t *testing.T) {
	provider := NewOpenAIProvider()
	// Intentionally not setting LLM client

	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	reqBody := CreateResponseRequest{
		Model: "gpt-4",
		Input: "Hello",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/openai/v1/responses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConvertCreateResponseToGaiRequest(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name     string
		request  CreateResponseRequest
		expected gai.GenerateRequest
	}{
		{
			name: "simple text input",
			request: CreateResponseRequest{
				Model:           "gpt-4",
				Input:           "Hello world",
				Instructions:    "Be helpful",
				Temperature:     toPtr(0.8),
				TopP:            toPtr(0.9),
				MaxOutputTokens: toPtr(150),
			},
			expected: gai.GenerateRequest{
				ModelID:         "gpt-4",
				Instructions:    "Be helpful",
				Input:           gai.TextInput{Text: "Hello world"},
				Temperature:     0.8,
				TopP:            0.9,
				MaxOutputTokens: 150,
				Stream:          false,
			},
		},
		{
			name: "with streaming",
			request: CreateResponseRequest{
				Model:  "gpt-4",
				Input:  "Tell me a joke",
				Stream: toPtr(true),
			},
			expected: gai.GenerateRequest{
				ModelID: "gpt-4",
				Input:   gai.TextInput{Text: "Tell me a joke"},
				Stream:  true,
			},
		},
		{
			name: "with input items - user message",
			request: CreateResponseRequest{
				Model: "gpt-4",
				InputItems: []InputItem{
					{
						Type:    InputItemTypeMessage,
						Content: mustMarshal(map[string]string{"role": "user", "content": "What's the capital of France?"}),
					},
				},
			},
			expected: gai.GenerateRequest{
				ModelID: "gpt-4",
				Input: gai.Conversation{
					Messages: []gai.Message{
						{
							Role:    gai.RoleUser,
							Content: gai.TextInput{Text: "What's the capital of France?"},
						},
					},
				},
				Stream: false,
			},
		},
		{
			name: "with system message in input items",
			request: CreateResponseRequest{
				Model: "gpt-4",
				InputItems: []InputItem{
					{
						Type:    InputItemTypeMessage,
						Content: mustMarshal(map[string]string{"role": "system", "content": "You are a helpful assistant"}),
					},
					{
						Type:    InputItemTypeMessage,
						Content: mustMarshal(map[string]string{"role": "user", "content": "Hello"}),
					},
				},
			},
			expected: gai.GenerateRequest{
				ModelID:      "gpt-4",
				Instructions: "You are a helpful assistant",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertCreateResponseToGaiRequest(tt.request)
			assert.Equal(t, tt.expected.ModelID, result.ModelID)
			assert.Equal(t, tt.expected.Instructions, result.Instructions)
			assert.Equal(t, tt.expected.Temperature, result.Temperature)
			assert.Equal(t, tt.expected.TopP, result.TopP)
			assert.Equal(t, tt.expected.MaxOutputTokens, result.MaxOutputTokens)
			assert.Equal(t, tt.expected.Stream, result.Stream)

			// For input comparison, we need to check type and content
			if textInput, ok := tt.expected.Input.(gai.TextInput); ok {
				resultTextInput, ok := result.Input.(gai.TextInput)
				assert.True(t, ok, "Expected TextInput")
				assert.Equal(t, textInput.Text, resultTextInput.Text)
			} else if conv, ok := tt.expected.Input.(gai.Conversation); ok {
				resultConv, ok := result.Input.(gai.Conversation)
				assert.True(t, ok, "Expected Conversation")
				assert.Len(t, resultConv.Messages, len(conv.Messages))
				for i, msg := range conv.Messages {
					assert.Equal(t, msg.Role, resultConv.Messages[i].Role)
					if textContent, ok := msg.Content.(gai.TextInput); ok {
						resultTextContent, ok := resultConv.Messages[i].Content.(gai.TextInput)
						assert.True(t, ok, "Expected TextInput content")
						assert.Equal(t, textContent.Text, resultTextContent.Text)
					}
				}
			}
		})
	}
}

func TestConvertGaiResponseToResponse(t *testing.T) {
	provider := NewOpenAIProvider()

	gaiResp := &gai.Response{
		ID:      "test-response-456",
		ModelID: "gpt-4",
		Status:  "completed",
		Output: []gai.OutputItem{
			gai.TextOutput{Text: "Paris is the capital of France."},
		},
		Usage: &gai.TokenUsage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
		CreatedAt: time.Now(),
	}

	result := provider.convertGaiResponseToResponse(gaiResp, "gpt-4")

	assert.Equal(t, "test-response-456", result.ID)
	assert.Equal(t, "response", result.Object)
	assert.Equal(t, "gpt-4", result.Model)
	assert.Equal(t, ResponseStatusCompleted, result.Status)
	assert.Len(t, result.Output, 1)
	assert.Equal(t, OutputItemTypeMessage, result.Output[0].Type)
	assert.NotNil(t, result.OutputText)
	assert.Equal(t, "Paris is the capital of France.", *result.OutputText)
	assert.NotNil(t, result.Usage)
	assert.Equal(t, 20, result.Usage.InputTokens)
	assert.Equal(t, 10, result.Usage.OutputTokens)
	assert.Equal(t, 30, result.Usage.TotalTokens)
}

func TestResponseStructMethods(t *testing.T) {
	t.Run("NewResponse", func(t *testing.T) {
		response := NewResponse("test-id-123")

		assert.Equal(t, "test-id-123", response.ID)
		assert.Equal(t, "response", response.Object)
		assert.Equal(t, ResponseStatusInProgress, response.Status)
		assert.NotZero(t, response.CreatedAt)
		assert.Nil(t, response.Error)
		assert.Nil(t, response.IncompleteDetails)
		assert.Empty(t, response.Output)
		assert.True(t, response.ParallelToolCalls)
		assert.False(t, response.Background)
	})

	t.Run("SetCompleted", func(t *testing.T) {
		response := NewResponse("test-id")
		response.SetCompleted()

		assert.Equal(t, ResponseStatusCompleted, response.Status)
	})

	t.Run("SetFailed", func(t *testing.T) {
		response := NewResponse("test-id")
		response.SetFailed(ResponseErrorServerError, "Something went wrong")

		assert.Equal(t, ResponseStatusFailed, response.Status)
		assert.NotNil(t, response.Error)
		assert.Equal(t, ResponseErrorServerError, response.Error.Code)
		assert.Equal(t, "Something went wrong", response.Error.Message)
	})

	t.Run("SetCancelled", func(t *testing.T) {
		response := NewResponse("test-id")
		response.SetCancelled()

		assert.Equal(t, ResponseStatusCancelled, response.Status)
	})

	t.Run("AddOutputItem", func(t *testing.T) {
		response := NewResponse("test-id")

		item := OutputItem{
			Type:    OutputItemTypeMessage,
			Content: mustMarshal(map[string]string{"role": "assistant", "content": "Hello"}),
		}

		response.AddOutputItem(item)

		assert.Len(t, response.Output, 1)
		assert.Equal(t, OutputItemTypeMessage, response.Output[0].Type)
	})
}

func TestConvertChunkToResponseStreamEvent(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name       string
		chunk      *gai.ResponseChunk
		responseID string
		wantType   string
	}{
		{
			name: "text delta chunk",
			chunk: &gai.ResponseChunk{
				ID:    "chunk-1",
				Delta: gai.OutputDelta{Text: "Hello"},
			},
			responseID: "resp-123",
			wantType:   "response.text.delta",
		},
		{
			name: "completed chunk",
			chunk: &gai.ResponseChunk{
				ID:       "chunk-2",
				Finished: true,
				Usage:    &gai.TokenUsage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
			},
			responseID: "resp-123",
			wantType:   "response.completed",
		},
		{
			name: "created chunk",
			chunk: &gai.ResponseChunk{
				ID: "chunk-3",
			},
			responseID: "resp-123",
			wantType:   "response.created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertChunkToResponseStreamEvent(tt.chunk, tt.responseID)

			assert.Equal(t, tt.wantType, result.Type)
			assert.NotNil(t, result.Data)

			var eventData map[string]any
			err := json.Unmarshal(result.Data, &eventData)
			require.NoError(t, err)

			assert.Equal(t, tt.responseID, eventData["id"])
			assert.Equal(t, "response.stream_event", eventData["object"])
			assert.NotZero(t, eventData["created"])
			assert.Equal(t, tt.wantType, eventData["type"])
		})
	}
}

// Helper function to create pointer to value
func toPtr[T any](v T) *T {
	return &v
}

// Helper function to marshal JSON that panics on error (for tests only)
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
