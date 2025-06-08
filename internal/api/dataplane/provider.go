// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"log/slog"
	"net/http"
)

type Provider interface {
	Name() string
	SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler)
	Shutdown(ctx context.Context) error
}

type ProviderOptions struct {
	Logger *slog.Logger
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
