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

	"codeberg.org/MadsRC/llmgw/internal/api"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
	"codeberg.org/MadsRC/llmgw/internal/api/providers"
	"codeberg.org/MadsRC/llmgw/internal/api/services"
	"codeberg.org/MadsRC/llmgw/internal/bootstrap"
	"codeberg.org/MadsRC/llmgw/internal/oidc"
	"codeberg.org/MadsRC/llmgw/internal/postgres"
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

	// Check and perform bootstrap if needed
	logger.Info("Checking system bootstrap status...")
	if err := bootstrap.CheckAndBootstrap(ctx, logger, orgRepo, userRepo, tokenRepo); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

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
	ssoHandler, err := services.NewSsoCallback(
		services.WithSsoCallbackLogger(logger),
		services.WithSsoCallbackProvider("oidc", oidcProvider),
		services.WithSsoCallbackSessionStore(sessionStore),
	)
	if err != nil {
		return fmt.Errorf("failed to create SSO handler: %w", err)
	}

	// Create auth interceptors
	authInterceptor := auth.NewInterceptor(sessionStore)
	tokenAuthenticator := auth.NewTokenAuthenticator(tokenRepo, userRepo)
	tokenInterceptor := auth.NewTokenInterceptor(tokenAuthenticator)

	// Create server
	server, err := api.NewControlPlaneServer(
		api.WithControlPlaneLogger(logger),
		api.WithControlPlaneUserRepository(userRepo),
		api.WithControlPlaneOrganizationRepository(orgRepo),
		api.WithControlPlaneTokenRepository(tokenRepo),
		api.WithSSOHandler(ssoHandler),
		api.WithSessionStore(sessionStore),
		api.WithAuthInterceptor(authInterceptor),
		api.WithTokenInterceptor(tokenInterceptor),
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create OpenAI provider
	openaiProvider := providers.NewOpenAIProvider(
		api.WithProviderLogger(logger),
	)

	// Create DataPlane server
	dataPlaneServer, err := api.NewDataPlaneServer(
		api.WithDataPlaneLogger(logger),
		api.WithDataPlaneAddr(c.String("data-plane-listen")),
		api.WithDataPlaneTokenAuthenticator(tokenAuthenticator),
		api.WithDataPlaneProviders(openaiProvider),
	)
	if err != nil {
		return fmt.Errorf("failed to create data plane server: %w", err)
	}

	// Setup ControlPlane HTTP server
	controlPlaneAddr := c.String("control-plane-listen")
	logger.Info("Starting control plane server", "address", controlPlaneAddr)

	controlPlaneServer := &http.Server{
		Addr:         controlPlaneAddr,
		Handler:      h2c.NewHandler(server.GetMux(), &http2.Server{}),
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
