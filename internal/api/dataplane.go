// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type DataPlaneServer struct {
	options    *dataPlaneOptions
	mux        *http.ServeMux
	httpServer *http.Server
}

type dataPlaneOptions struct {
	Logger       *slog.Logger
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Addr         string
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
		options: opts,
		mux:     mux,
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
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /hello", s.handleHello)
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
	if _, err := fmt.Fprint(w, `{"message":"Hello, World!","server":"llmgw-dataplane"}`); err != nil {
		s.options.Logger.Error("Failed to write hello response", "error", err)
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
