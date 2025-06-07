// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"log/slog"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
)

type SsoCallback struct {
	options *ssoCallbackOptions
}

// NewSsoCallback creates a new [SsoCallback].
func NewSsoCallback(options ...SsoCallbackOption) (*SsoCallback, error) {
	opts := defaultSsoCallbackOptions
	for _, opt := range GlobalSsoCallbackOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &SsoCallback{
		options: &opts,
	}, nil
}

type ssoCallbackOptions struct {
	Logger       *slog.Logger
	Providers    map[string]llmgw.SsoProvider
	SessionStore auth.SessionStore
}

var defaultSsoCallbackOptions = ssoCallbackOptions{
	Logger:       slog.Default(),
	Providers:    make(map[string]llmgw.SsoProvider),
	SessionStore: nil, // Will be set by WithSsoCallbackSessionStore
}

// GlobalSsoCallbackOptions is a list of [SsoCallbackOption]s that are applied to all [SsoCallback]s.
var GlobalSsoCallbackOptions []SsoCallbackOption

// SsoCallbackOption is an option for configuring a [SsoCallback].
type SsoCallbackOption interface {
	apply(*ssoCallbackOptions)
}

// funcSsoCallbackOption is a [SsoCallbackOption] that calls a function.
// It is used to wrap a function, so it satisfies the [SsoCallbackOption] interface.
type funcSsoCallbackOption struct {
	f func(*ssoCallbackOptions)
}

func (fdo *funcSsoCallbackOption) apply(opts *ssoCallbackOptions) {
	fdo.f(opts)
}

func newFuncSsoCallbackOption(f func(*ssoCallbackOptions)) *funcSsoCallbackOption {
	return &funcSsoCallbackOption{
		f: f,
	}
}

// WithSsoCallbackLogger returns a [SsoCallbackOption] that uses the provided logger.
func WithSsoCallbackLogger(logger *slog.Logger) SsoCallbackOption {
	return newFuncSsoCallbackOption(func(opts *ssoCallbackOptions) {
		opts.Logger = logger
	})
}

// WithSsoCallbackProvider returns a [SsoCallbackOption] that registers an SSO provider with the given prefix.
func WithSsoCallbackProvider(prefix string, provider llmgw.SsoProvider) SsoCallbackOption {
	return newFuncSsoCallbackOption(func(opts *ssoCallbackOptions) {
		opts.Providers[prefix] = provider
	})
}

// WithSsoCallbackSessionStore returns a [SsoCallbackOption] that sets the session store.
func WithSsoCallbackSessionStore(store auth.SessionStore) SsoCallbackOption {
	return newFuncSsoCallbackOption(func(opts *ssoCallbackOptions) {
		opts.SessionStore = store
	})
}
