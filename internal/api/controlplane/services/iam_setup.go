// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"log/slog"

	"github.com/MadsRC/trustedai"
)

type Iam struct {
	options *iamOptions
}

// NewIam creates a new [Iam].
func NewIam(options ...IamOption) (*Iam, error) {
	opts := defaultIamOptions
	for _, opt := range GlobalIamOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &Iam{
		options: &opts,
	}, nil
}

type iamOptions struct {
	Logger                 *slog.Logger
	UserRepository         trustedai.UserRepository
	OrganizationRepository trustedai.OrganizationRepository
	TokenRepository        trustedai.TokenRepository
}

var defaultIamOptions = iamOptions{
	Logger: slog.Default(),
}

// GlobalIamOptions is a list of [IamOption]s that are applied to all [Iam]s.
var GlobalIamOptions []IamOption

// IamOption is an option for configuring a [Iam].
type IamOption interface {
	apply(*iamOptions)
}

// funcIamOption is a [IamOption] that calls a function.
// It is used to wrap a function, so it satisfies the [IamOption] interface.
type funcIamOption struct {
	f func(*iamOptions)
}

func (fdo *funcIamOption) apply(opts *iamOptions) {
	fdo.f(opts)
}

func newFuncIamOption(f func(*iamOptions)) *funcIamOption {
	return &funcIamOption{
		f: f,
	}
}

// WithIamLogger returns a [IamOption] that uses the provided logger.
func WithIamLogger(logger *slog.Logger) IamOption {
	return newFuncIamOption(func(opts *iamOptions) {
		opts.Logger = logger
	})
}

// WithUserRepository returns a [IamOption] that uses the provided user repository.
func WithUserRepository(repo trustedai.UserRepository) IamOption {
	return newFuncIamOption(func(opts *iamOptions) {
		opts.UserRepository = repo
	})
}

// WithOrganizationRepository returns a [IamOption] that uses the provided organization repository.
func WithOrganizationRepository(repo trustedai.OrganizationRepository) IamOption {
	return newFuncIamOption(func(opts *iamOptions) {
		opts.OrganizationRepository = repo
	})
}

// WithTokenRepository returns a [IamOption] that uses the provided token repository.
func WithTokenRepository(repo trustedai.TokenRepository) IamOption {
	return newFuncIamOption(func(opts *iamOptions) {
		opts.TokenRepository = repo
	})
}
