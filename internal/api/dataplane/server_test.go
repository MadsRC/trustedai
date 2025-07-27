// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MadsRC/trustedai/internal/api/dataplane/interfaces"
)

func TestNewDataPlaneServer(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []DataPlaneOption
		want    *DataPlaneServer
		wantErr bool
	}{
		{
			name:    "Create with default options",
			options: []DataPlaneOption{},
			want: &DataPlaneServer{
				options: &dataPlaneOptions{
					Logger:       slog.Default(),
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
					Addr:         ":8081",
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []DataPlaneOption{WithDataPlaneLogger(discardLogger)},
			want: &DataPlaneServer{
				options: &dataPlaneOptions{
					Logger:       discardLogger,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
					Addr:         ":8081",
				},
			},
		},
		{
			name:    "Create with custom address",
			options: []DataPlaneOption{WithDataPlaneAddr(":9090")},
			want: &DataPlaneServer{
				options: &dataPlaneOptions{
					Logger:       slog.Default(),
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
					Addr:         ":9090",
				},
			},
		},
		{
			name: "Create with custom timeouts",
			options: []DataPlaneOption{
				WithDataPlaneReadTimeout(10 * time.Second),
				WithDataPlaneWriteTimeout(15 * time.Second),
				WithDataPlaneIdleTimeout(60 * time.Second),
			},
			want: &DataPlaneServer{
				options: &dataPlaneOptions{
					Logger:       slog.Default(),
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 15 * time.Second,
					IdleTimeout:  60 * time.Second,
					Addr:         ":8081",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDataPlaneServer(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDataPlaneServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Error("NewDataPlaneServer() returned nil server")
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewDataPlaneServer() logger = %v, want %v", got.options.Logger, tt.want.options.Logger)
			}
			if got.options.ReadTimeout != tt.want.options.ReadTimeout {
				t.Errorf("NewDataPlaneServer() readTimeout = %v, want %v", got.options.ReadTimeout, tt.want.options.ReadTimeout)
			}
			if got.options.WriteTimeout != tt.want.options.WriteTimeout {
				t.Errorf("NewDataPlaneServer() writeTimeout = %v, want %v", got.options.WriteTimeout, tt.want.options.WriteTimeout)
			}
			if got.options.IdleTimeout != tt.want.options.IdleTimeout {
				t.Errorf("NewDataPlaneServer() idleTimeout = %v, want %v", got.options.IdleTimeout, tt.want.options.IdleTimeout)
			}
			if got.options.Addr != tt.want.options.Addr {
				t.Errorf("NewDataPlaneServer() addr = %v, want %v", got.options.Addr, tt.want.options.Addr)
			}
			if got.mux == nil {
				t.Error("NewDataPlaneServer() mux is nil")
			}
			if got.httpServer == nil {
				t.Error("NewDataPlaneServer() httpServer is nil")
			}
		})
	}
}

func TestDataPlaneServer_HandleHealth(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedContentType := "application/json"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedBody := `{"status":"ok"}`
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}
}

func TestDataPlaneServer_HandleHello(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	server.handleHello(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedContentType := "application/json"
	if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedBody := `{"message":"Hello, World!","server":"trustedai-dataplane"}`
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}
}

func TestDataPlaneServer_Routes(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Health endpoint",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Hello endpoint",
			method:         "GET",
			path:           "/hello",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			server.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDataPlaneServer_GetMux(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	mux := server.GetMux()
	if mux == nil {
		t.Error("GetMux() returned nil")
	}
	if mux != server.mux {
		t.Error("GetMux() returned different mux than expected")
	}
}

func TestDataPlaneServer_GetServer(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	httpServer := server.GetServer()
	if httpServer == nil {
		t.Error("GetServer() returned nil")
	}
	if httpServer != server.httpServer {
		t.Error("GetServer() returned different server than expected")
	}
}

func TestDataPlaneServer_Shutdown(t *testing.T) {
	server, err := NewDataPlaneServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() returned error: %v", err)
	}
}

func TestDataPlaneServer_WithProviders(t *testing.T) {
	// Create a mock provider
	mockProvider := &mockProvider{name: "test"}

	server, err := NewDataPlaneServer(WithDataPlaneProviders(mockProvider))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if len(server.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(server.providers))
	}

	if server.providers[0].Name() != "test" {
		t.Errorf("Expected provider name 'test', got %s", server.providers[0].Name())
	}
}

// mockProvider is a simple mock implementation of Provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler) {
	mux.HandleFunc("GET /"+m.name, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("mock response")); err != nil {
			// Handle error in test (logging would be overkill for test)
			return
		}
	})
}

func (m *mockProvider) SetLLMClient(client LLMClient) {
	// Mock implementation - does nothing
}

func (m *mockProvider) SetUsageMiddleware(middleware interfaces.UsageMiddleware) {
	// Mock implementation - does nothing
}

func (m *mockProvider) Shutdown(ctx context.Context) error {
	return nil
}
