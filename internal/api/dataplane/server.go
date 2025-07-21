// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
	dauth "codeberg.org/MadsRC/llmgw/internal/api/dataplane/auth"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/middleware"
	"codeberg.org/MadsRC/llmgw/internal/monitoring"
)

type DataPlaneServer struct {
	options         *dataPlaneOptions
	mux             *http.ServeMux
	httpServer      *http.Server
	authMiddleware  *dauth.CombinedAuthMiddleware
	usageMiddleware *middleware.UsageTrackingMiddleware
	providers       []Provider
	LLMClient       LLMClient
}

type dataPlaneOptions struct {
	Logger             *slog.Logger
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	Addr               string
	TokenAuthenticator *auth.TokenAuthenticator
	UsageRepository    llmgw.UsageRepository
	UsageMetrics       *monitoring.UsageMetrics
	Providers          []Provider
	LLMClient          LLMClient
}

type DataPlaneOption interface {
	apply(*dataPlaneOptions)
}

type dataPlaneOptionFunc func(*dataPlaneOptions)

func (f dataPlaneOptionFunc) apply(opts *dataPlaneOptions) {
	f(opts)
}

func WithDataPlaneLogger(logger *slog.Logger) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.Logger = logger
	})
}

func WithDataPlaneAddr(addr string) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.Addr = addr
	})
}

func WithDataPlaneReadTimeout(timeout time.Duration) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.ReadTimeout = timeout
	})
}

func WithDataPlaneWriteTimeout(timeout time.Duration) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.WriteTimeout = timeout
	})
}

func WithDataPlaneIdleTimeout(timeout time.Duration) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.IdleTimeout = timeout
	})
}

func WithDataPlaneTokenAuthenticator(authenticator *auth.TokenAuthenticator) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.TokenAuthenticator = authenticator
	})
}

func WithDataPlaneProviders(providers ...Provider) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.Providers = append(opts.Providers, providers...)
	})
}

func WithDataPlaneLLMClient(client LLMClient) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.LLMClient = client
	})
}

func WithDataPlaneUsageRepository(repo llmgw.UsageRepository) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.UsageRepository = repo
	})
}

func WithDataPlaneUsageMetrics(metrics *monitoring.UsageMetrics) DataPlaneOption {
	return dataPlaneOptionFunc(func(opts *dataPlaneOptions) {
		opts.UsageMetrics = metrics
	})
}

func NewDataPlaneServer(options ...DataPlaneOption) (*DataPlaneServer, error) {
	opts := &dataPlaneOptions{
		Logger:       slog.Default(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":8081",
	}

	for _, option := range options {
		option.apply(opts)
	}

	mux := http.NewServeMux()

	server := &DataPlaneServer{
		options:   opts,
		mux:       mux,
		providers: opts.Providers,
		LLMClient: opts.LLMClient,
	}

	// Initialize Combined authentication middleware if TokenAuthenticator is provided
	if opts.TokenAuthenticator != nil {
		server.authMiddleware = dauth.NewCombinedAuthMiddleware(opts.TokenAuthenticator, opts.Logger)
	}

	// Initialize usage tracking middleware if UsageRepository is provided
	if opts.UsageRepository != nil {
		server.usageMiddleware = middleware.NewUsageTrackingMiddleware(opts.UsageRepository, opts.Logger, opts.UsageMetrics)
	}

	// Set LLMClient on all providers if available
	if server.LLMClient != nil {
		for _, provider := range server.providers {
			provider.SetLLMClient(server.LLMClient)
		}
	}

	// Set UsageMiddleware on all providers if available
	if server.usageMiddleware != nil {
		for _, provider := range server.providers {
			provider.SetUsageMiddleware(server.usageMiddleware)
		}
	}

	server.setupRoutes()

	server.httpServer = &http.Server{
		Addr:         opts.Addr,
		Handler:      mux,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}

	return server, nil
}

func (s *DataPlaneServer) setupRoutes() {
	// Health endpoint - no authentication required
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Build middleware chain for protected endpoints
	var baseAuth func(http.Handler) http.Handler

	// Start with usage tracking middleware (innermost)
	if s.usageMiddleware != nil {
		baseAuth = s.usageMiddleware.Track
	}

	// Add authentication middleware (outermost for provider routes)
	if s.authMiddleware != nil {
		if baseAuth != nil {
			// Chain: auth -> usage tracking
			authHandler := s.authMiddleware.Authenticate
			prevBaseAuth := baseAuth // Capture current value before redefining
			baseAuth = func(next http.Handler) http.Handler {
				return authHandler(prevBaseAuth(next))
			}
		} else {
			baseAuth = s.authMiddleware.Authenticate
		}

		// Apply to hello endpoint (auth only, no usage tracking needed)
		s.mux.Handle("GET /hello", s.authMiddleware.Authenticate(http.HandlerFunc(s.handleHello)))
	} else {
		// Fallback if no authentication is configured (for backward compatibility)
		s.mux.HandleFunc("GET /hello", s.handleHello)
	}

	// Setup provider routes with full middleware chain
	for _, provider := range s.providers {
		s.options.Logger.Info("Setting up provider routes", "provider", provider.Name())
		provider.SetupRoutes(s.mux, baseAuth)
	}
}

func (s *DataPlaneServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, `{"status":"ok"}`); err != nil {
		s.options.Logger.Error("Failed to write health response", "error", err)
	}
}

func (s *DataPlaneServer) handleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Check if user is authenticated
	if user := dauth.UserFromHTTPContext(r); user != nil {
		response := fmt.Sprintf(`{"message":"Hello, %s!","server":"llmgw-dataplane","user_id":"%s"}`,
			user.Name, user.ID)
		if _, err := fmt.Fprint(w, response); err != nil {
			s.options.Logger.Error("Failed to write hello response", "error", err)
		}
	} else {
		if _, err := fmt.Fprint(w, `{"message":"Hello, World!","server":"llmgw-dataplane"}`); err != nil {
			s.options.Logger.Error("Failed to write hello response", "error", err)
		}
	}
}

func (s *DataPlaneServer) Start(ctx context.Context) error {
	s.options.Logger.Info("Starting data plane server", "addr", s.options.Addr)

	listener, err := net.Listen("tcp", s.options.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.options.Addr, err)
	}

	serverErrors := make(chan error, 1)

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("server failed: %w", err)
		}
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		return s.Shutdown(ctx)
	}
}

func (s *DataPlaneServer) Shutdown(ctx context.Context) error {
	s.options.Logger.Info("Shutting down data plane server")

	// Shutdown providers first
	for _, provider := range s.providers {
		if err := provider.Shutdown(ctx); err != nil {
			s.options.Logger.Error("Failed to shutdown provider", "provider", provider.Name(), "error", err)
		}
	}

	// Shutdown usage middleware
	if s.usageMiddleware != nil {
		s.usageMiddleware.Shutdown()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.options.Logger.Error("Failed to gracefully shutdown server", "error", err)
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.options.Logger.Info("Data plane server stopped")
	return nil
}

func (s *DataPlaneServer) GetMux() *http.ServeMux {
	return s.mux
}

func (s *DataPlaneServer) GetServer() *http.Server {
	return s.httpServer
}
