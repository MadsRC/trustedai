// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/gai-org/gai"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
	"codeberg.org/MadsRC/llmgw/internal/api/controlplane"
	cauth "codeberg.org/MadsRC/llmgw/internal/api/controlplane/auth"
	controlplaneservices "codeberg.org/MadsRC/llmgw/internal/api/controlplane/services"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane"
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/providers"
	"codeberg.org/MadsRC/llmgw/internal/bootstrap"
	"codeberg.org/MadsRC/llmgw/internal/modelrouter"
	"codeberg.org/MadsRC/llmgw/internal/monitoring"
	"codeberg.org/MadsRC/llmgw/internal/oidc"
	"codeberg.org/MadsRC/llmgw/internal/postgres"
	"codeberg.org/MadsRC/llmgw/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	cmd := &cli.Command{
		Name:    "llmgw",
		Usage:   "LLM Gateway Control Plane Server",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "control-plane-listen",
				Value:   "localhost:9999",
				Usage:   "Address for control plane to listen on",
				Sources: cli.EnvVars("LLMGW_CONTROL_PLANE_LISTEN"),
			},
			&cli.StringFlag{
				Name:    "data-plane-listen",
				Value:   "localhost:8081",
				Usage:   "Address for data plane to listen on",
				Sources: cli.EnvVars("LLMGW_DATA_PLANE_LISTEN"),
			},
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "PostgreSQL database connection URL",
				Sources:  cli.EnvVars("DATABASE_URL"),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "base-url",
				Value:   "http://localhost:9999",
				Usage:   "Base URL for the server (used for OIDC redirects)",
				Sources: cli.EnvVars("LLMGW_BASE_URL"),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("LLMGW_DEBUG"),
			},
			&cli.StringFlag{
				Name:     "otlp-endpoint",
				Usage:    "OTLP endpoint for metrics export (required)",
				Sources:  cli.EnvVars("LLMGW_OTLP_ENDPOINT"),
				Required: true,
			},
		},
		Action: runServer,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("Failed to run command", "error", err)
		os.Exit(1)
	}
}

func runServer(ctx context.Context, c *cli.Command) error {
	// Setup logger
	logLevel := slog.LevelInfo
	if c.Bool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Connect to database
	dbURL := c.String("database-url")
	logger.Info("Connecting to database", "url", dbURL)

	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info("Database connection established")

	// Run database migrations
	if err := postgres.RunMigrations(logger, dbURL); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	// Create repositories
	userRepo, err := postgres.NewUserRepository(
		postgres.WithUserRepositoryLogger(logger),
		postgres.WithUserRepositoryDb(dbPool),
	)
	if err != nil {
		return fmt.Errorf("failed to create user repository: %w", err)
	}

	orgRepo, err := postgres.NewOrganizationRepository(
		postgres.WithOrganizationRepositoryLogger(logger),
		postgres.WithOrganizationRepositoryDb(dbPool),
	)
	if err != nil {
		return fmt.Errorf("failed to create organization repository: %w", err)
	}

	tokenRepo, err := postgres.NewTokenRepository(
		postgres.WithTokenRepositoryLogger(logger),
		postgres.WithTokenRepositoryDb(dbPool),
	)
	if err != nil {
		return fmt.Errorf("failed to create token repository: %w", err)
	}

	// Create credential repository
	credentialRepo := postgres.NewCredentialRepository(dbPool)

	// Create model repository
	modelRepo := postgres.NewModelRepository(dbPool)

	// Create usage repository
	usageRepo, err := postgres.NewUsageRepository(
		postgres.WithUsageRepositoryLogger(logger),
		postgres.WithUsageRepositoryDb(dbPool),
	)
	if err != nil {
		return fmt.Errorf("failed to create usage repository: %w", err)
	}

	// Create billing repository
	billingRepo, err := postgres.NewBillingRepository(
		postgres.WithBillingRepositoryLogger(logger),
		postgres.WithBillingRepositoryDb(dbPool),
	)
	if err != nil {
		return fmt.Errorf("failed to create billing repository: %w", err)
	}

	// Check and perform bootstrap if needed
	logger.Info("Checking system bootstrap status...")
	if err := bootstrap.CheckAndBootstrap(ctx, logger, orgRepo, userRepo, tokenRepo); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Set up monitoring system
	logger.Info("Setting up monitoring system...")
	monitoringConfig := monitoring.Config{
		ServiceName:    "llmgw",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   c.String("otlp-endpoint"),
	}

	monitoringManager, err := monitoring.NewManager(monitoringConfig)
	if err != nil {
		return fmt.Errorf("failed to create monitoring manager: %w", err)
	}

	// Get usage metrics
	usageMetrics := monitoringManager.GetUsageMetrics()
	logger.Info("Monitoring system initialized with OTLP export", "endpoint", c.String("otlp-endpoint"))

	// Create session store
	sessionStore := auth.NewMemorySessionStore()

	// Create OIDC provider
	oidcProvider, err := oidc.NewProvider(
		oidc.WithProviderLogger(logger),
		oidc.WithProviderUserRepo(userRepo),
		oidc.WithProviderOrgRepo(orgRepo),
		oidc.WithProviderBaseURL(c.String("base-url")),
	)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Create SSO callback service
	logger.Info("Creating SSO callback service...")
	ssoHandler, err := controlplaneservices.NewSsoCallback(
		controlplaneservices.WithSsoCallbackLogger(logger),
		controlplaneservices.WithSsoCallbackProvider("oidc", oidcProvider),
		controlplaneservices.WithSsoCallbackSessionStore(sessionStore),
	)
	if err != nil {
		return fmt.Errorf("failed to create SSO handler: %w", err)
	}
	logger.Info("SSO callback service created successfully")

	// Create auth interceptors
	authInterceptor := cauth.NewInterceptor(sessionStore)
	tokenAuthenticator := auth.NewTokenAuthenticator(tokenRepo, userRepo)
	tokenInterceptor := cauth.NewTokenInterceptor(tokenAuthenticator)

	// Create server
	server, err := controlplane.NewControlPlaneServer(
		controlplane.WithControlPlaneLogger(logger),
		controlplane.WithControlPlaneUserRepository(userRepo),
		controlplane.WithControlPlaneOrganizationRepository(orgRepo),
		controlplane.WithControlPlaneTokenRepository(tokenRepo),
		controlplane.WithControlPlaneCredentialRepository(credentialRepo),
		controlplane.WithControlPlaneModelRepository(modelRepo),
		controlplane.WithSSOHandler(ssoHandler),
		controlplane.WithSessionStore(sessionStore),
		controlplane.WithAuthInterceptor(authInterceptor),
		controlplane.WithTokenInterceptor(tokenInterceptor),
		controlplane.WithFrontendFS(llmgw.GetFrontendFS()),
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create custom model router with database integration
	customModelRouter := modelrouter.New(
		modelrouter.WithDatabase(dbPool),
		modelrouter.WithLogger(logger.WithGroup("modelrouter")),
	)

	// Create LLM client with custom router
	llmClient := gai.New(
		gai.WithClientLogger(logger.WithGroup("llmclient")),
		gai.WithModelRouter(customModelRouter),
	)

	logger.Info("Model router configured with database integration - providers are managed through database")

	// Create cost calculator
	costCalculator := services.NewCostCalculator(
		usageRepo,
		modelRepo,
		billingRepo,
		services.WithLogger(logger.WithGroup("cost-calculator")),
		services.WithBatchSize(50),
		services.WithMetrics(usageMetrics),
	)

	// Create scheduler for background jobs
	scheduler := services.NewScheduler(
		costCalculator,
		services.WithSchedulerLogger(logger.WithGroup("scheduler")),
	)

	// Create OpenAI provider
	openaiProvider := providers.NewOpenAIProvider(
		dataplane.WithProviderLogger(logger),
	)

	// Create Anthropic provider
	anthropicProvider := providers.NewAnthropicProvider(
		dataplane.WithProviderLogger(logger),
	)

	// Create DataPlane server
	dataPlaneServer, err := dataplane.NewDataPlaneServer(
		dataplane.WithDataPlaneLogger(logger),
		dataplane.WithDataPlaneAddr(c.String("data-plane-listen")),
		dataplane.WithDataPlaneTokenAuthenticator(tokenAuthenticator),
		dataplane.WithDataPlaneUsageRepository(usageRepo),
		dataplane.WithDataPlaneUsageMetrics(usageMetrics),
		dataplane.WithDataPlaneProviders(openaiProvider, anthropicProvider),
		dataplane.WithDataPlaneLLMClient(llmClient),
	)
	if err != nil {
		return fmt.Errorf("failed to create data plane server: %w", err)
	}

	// Setup ControlPlane HTTP server with CORS
	controlPlaneAddr := c.String("control-plane-listen")
	logger.Info("Starting control plane server", "address", controlPlaneAddr)

	// Create CORS middleware wrapper
	corsWrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("CORS Middleware", "method", r.Method, "path", r.URL.Path, "origin", r.Header.Get("Origin"))

		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			logger.Debug("CORS Middleware: Handling OPTIONS preflight request")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Authorization, X-User-Agent, User-Agent, Accept-Encoding")
			w.Header().Set("Access-Control-Max-Age", "7200")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass through to original handler
		server.GetMux().ServeHTTP(w, r)
	})

	controlPlaneServer := &http.Server{
		Addr:         controlPlaneAddr,
		Handler:      h2c.NewHandler(corsWrapper, &http2.Server{}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start both servers in background
	serverChan := make(chan error, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start ControlPlane server
	go func() {
		if err := controlPlaneServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverChan <- fmt.Errorf("control plane server failed: %w", err)
		}
	}()

	// Start DataPlane server
	go func() {
		if err := dataPlaneServer.Start(ctx); err != nil {
			serverChan <- fmt.Errorf("data plane server failed: %w", err)
		}
	}()

	// Start background scheduler
	scheduler.Start(ctx)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverChan:
		return err
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down", "signal", sig)

		// Cancel context to stop DataPlane server
		cancel()

		// Stop background scheduler
		scheduler.Stop()

		// Shutdown monitoring system
		if err := monitoringManager.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown monitoring system", "error", err)
		}

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Shutdown ControlPlane server gracefully
		if err := controlPlaneServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown control plane server gracefully", "error", err)
			return err
		}

		// DataPlane server shutdown is handled by context cancellation

		logger.Info("Server shutdown complete")
		return nil
	}
}
