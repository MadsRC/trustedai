// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"log/slog"

	"github.com/MadsRC/trustedai"
)

type ModelManagement struct {
	options *modelManagementOptions
}

// NewModelManagement creates a new [ModelManagement].
func NewModelManagement(options ...ModelManagementOption) (*ModelManagement, error) {
	opts := defaultModelManagementOptions
	for _, opt := range GlobalModelManagementOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &ModelManagement{
		options: &opts,
	}, nil
}

type modelManagementOptions struct {
	Logger               *slog.Logger
	CredentialRepository trustedai.CredentialRepository
	ModelRepository      trustedai.ModelRepository
}

var defaultModelManagementOptions = modelManagementOptions{
	Logger: slog.Default(),
}

// GlobalModelManagementOptions is a list of [ModelManagementOption]s that are applied to all [ModelManagement]s.
var GlobalModelManagementOptions []ModelManagementOption

// ModelManagementOption is an option for configuring a [ModelManagement].
type ModelManagementOption interface {
	apply(*modelManagementOptions)
}

// funcModelManagementOption is a [ModelManagementOption] that calls a function.
// It is used to wrap a function, so it satisfies the [ModelManagementOption] interface.
type funcModelManagementOption struct {
	f func(*modelManagementOptions)
}

func (fdo *funcModelManagementOption) apply(opts *modelManagementOptions) {
	fdo.f(opts)
}

func newFuncModelManagementOption(f func(*modelManagementOptions)) *funcModelManagementOption {
	return &funcModelManagementOption{
		f: f,
	}
}

// WithModelManagementLogger returns a [ModelManagementOption] that uses the provided logger.
func WithModelManagementLogger(logger *slog.Logger) ModelManagementOption {
	return newFuncModelManagementOption(func(opts *modelManagementOptions) {
		opts.Logger = logger
	})
}

// WithCredentialRepository returns a [ModelManagementOption] that uses the provided credential repository.
func WithCredentialRepository(repo trustedai.CredentialRepository) ModelManagementOption {
	return newFuncModelManagementOption(func(opts *modelManagementOptions) {
		opts.CredentialRepository = repo
	})
}

// WithModelRepository returns a [ModelManagementOption] that uses the provided model repository.
func WithModelRepository(repo trustedai.ModelRepository) ModelManagementOption {
	return newFuncModelManagementOption(func(opts *modelManagementOptions) {
		opts.ModelRepository = repo
	})
}
