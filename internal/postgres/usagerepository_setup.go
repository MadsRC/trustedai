// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"log/slog"
)

type UsageRepository struct {
	options *usageRepositoryOptions
}

// NewUsageRepository creates a new [UsageRepository].
func NewUsageRepository(options ...UsageRepositoryOption) (*UsageRepository, error) {
	opts := defaultUsageRepositoryOptions
	for _, opt := range GlobalUsageRepositoryOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &UsageRepository{
		options: &opts,
	}, nil
}

type usageRepositoryOptions struct {
	Logger *slog.Logger
	Db     PgxPoolInterface
}

var defaultUsageRepositoryOptions = usageRepositoryOptions{
	Logger: slog.Default(),
}

// GlobalUsageRepositoryOptions is a list of [UsageRepositoryOption]s that are applied to all [UsageRepository]s.
var GlobalUsageRepositoryOptions []UsageRepositoryOption

// UsageRepositoryOption is an option for configuring a [UsageRepository].
type UsageRepositoryOption interface {
	apply(*usageRepositoryOptions)
}

// funcUsageRepositoryOption is a [UsageRepositoryOption] that calls a function.
// It is used to wrap a function, so it satisfies the [UsageRepositoryOption] interface.
type funcUsageRepositoryOption struct {
	f func(*usageRepositoryOptions)
}

func (fdo *funcUsageRepositoryOption) apply(opts *usageRepositoryOptions) {
	fdo.f(opts)
}

func newFuncUsageRepositoryOption(f func(*usageRepositoryOptions)) *funcUsageRepositoryOption {
	return &funcUsageRepositoryOption{
		f: f,
	}
}

// WithUsageRepositoryLogger returns a [UsageRepositoryOption] that uses the provided logger.
func WithUsageRepositoryLogger(logger *slog.Logger) UsageRepositoryOption {
	return newFuncUsageRepositoryOption(func(opts *usageRepositoryOptions) {
		opts.Logger = logger
	})
}

// WithUsageRepositoryDb returns a [UsageRepositoryOption] that uses the provided database connection.
func WithUsageRepositoryDb(db PgxPoolInterface) UsageRepositoryOption {
	return newFuncUsageRepositoryOption(func(opts *usageRepositoryOptions) {
		opts.Db = db
	})
}
