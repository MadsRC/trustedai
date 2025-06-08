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

	"codeberg.org/MadsRC/llmgw/internal/api"
)

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider()
	if got := provider.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestOpenAIProvider_ChatCompletions(t *testing.T) {
	provider := NewOpenAIProvider()
	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("POST", "/openai/chat/completions", strings.NewReader(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}`))
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
	provider := NewOpenAIProvider()
	mux := http.NewServeMux()
	provider.SetupRoutes(mux, nil)

	req := httptest.NewRequest("GET", "/openai/models", nil)
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

	if response["object"] != "list" {
		t.Errorf("Expected object to be 'list', got %v", response["object"])
	}

	data, ok := response["data"].([]any)
	if !ok || len(data) == 0 {
		t.Errorf("Expected data to be a non-empty array")
	}
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
	logger := api.WithProviderLogger(nil)
	provider := NewOpenAIProvider(logger)

	if provider.options.Logger != nil {
		t.Errorf("Expected logger to be nil, got %v", provider.options.Logger)
	}
}
