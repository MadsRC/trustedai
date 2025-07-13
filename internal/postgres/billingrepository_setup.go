// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"log/slog"
)

type BillingRepository struct {
	options *billingRepositoryOptions
}

// NewBillingRepository creates a new [BillingRepository].
func NewBillingRepository(options ...BillingRepositoryOption) (*BillingRepository, error) {
	opts := defaultBillingRepositoryOptions
	for _, opt := range GlobalBillingRepositoryOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &BillingRepository{
		options: &opts,
	}, nil
}

type billingRepositoryOptions struct {
	Logger *slog.Logger
	Db     PgxPoolInterface
}

var defaultBillingRepositoryOptions = billingRepositoryOptions{
	Logger: slog.Default(),
}

// GlobalBillingRepositoryOptions is a list of [BillingRepositoryOption]s that are applied to all [BillingRepository]s.
var GlobalBillingRepositoryOptions []BillingRepositoryOption

// BillingRepositoryOption is an option for configuring a [BillingRepository].
type BillingRepositoryOption interface {
	apply(*billingRepositoryOptions)
}

// funcBillingRepositoryOption is a [BillingRepositoryOption] that calls a function.
// It is used to wrap a function, so it satisfies the [BillingRepositoryOption] interface.
type funcBillingRepositoryOption struct {
	f func(*billingRepositoryOptions)
}

func (fdo *funcBillingRepositoryOption) apply(opts *billingRepositoryOptions) {
	fdo.f(opts)
}

func newFuncBillingRepositoryOption(f func(*billingRepositoryOptions)) *funcBillingRepositoryOption {
	return &funcBillingRepositoryOption{
		f: f,
	}
}

// WithBillingRepositoryLogger returns a [BillingRepositoryOption] that uses the provided logger.
func WithBillingRepositoryLogger(logger *slog.Logger) BillingRepositoryOption {
	return newFuncBillingRepositoryOption(func(opts *billingRepositoryOptions) {
		opts.Logger = logger
	})
}

// WithBillingRepositoryDb returns a [BillingRepositoryOption] that uses the provided database connection.
func WithBillingRepositoryDb(db PgxPoolInterface) BillingRepositoryOption {
	return newFuncBillingRepositoryOption(func(opts *billingRepositoryOptions) {
		opts.Db = db
	})
}
