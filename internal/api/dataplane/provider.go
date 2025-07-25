// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"log/slog"
	"net/http"

	"codeberg.org/gai-org/gai"
)

type LLMClient interface {
	Generate(ctx context.Context, req gai.GenerateRequest) (*gai.Response, error)
	GenerateStream(ctx context.Context, req gai.GenerateRequest) (gai.ResponseStream, error)
}

type ModelRouter interface {
	ListModels(ctx context.Context) ([]gai.Model, error)
}

type Provider interface {
	Name() string
	SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler)
	SetLLMClient(client LLMClient)
	Shutdown(ctx context.Context) error
}

type ProviderOptions struct {
	Logger      *slog.Logger
	ModelRouter ModelRouter
}

type ProviderOption interface {
	Apply(*ProviderOptions)
}

type providerOptionFunc func(*ProviderOptions)

func (f providerOptionFunc) Apply(opts *ProviderOptions) {
	f(opts)
}

func WithProviderLogger(logger *slog.Logger) ProviderOption {
	return providerOptionFunc(func(opts *ProviderOptions) {
		opts.Logger = logger
	})
}

func WithModelRouter(router ModelRouter) ProviderOption {
	return providerOptionFunc(func(opts *ProviderOptions) {
		opts.ModelRouter = router
	})
}
