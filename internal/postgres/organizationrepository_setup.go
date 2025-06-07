// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"log/slog"
)

type OrganizationRepository struct {
	options *organizationRepositoryOptions
}

// NewOrganizationRepository creates a new [OrganizationRepository].
func NewOrganizationRepository(options ...OrganizationRepositoryOption) (*OrganizationRepository, error) {
	opts := defaultOrganizationRepositoryOptions
	for _, opt := range GlobalOrganizationRepositoryOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &OrganizationRepository{
		options: &opts,
	}, nil
}

type organizationRepositoryOptions struct {
	Logger *slog.Logger
	Db     PgxPoolInterface
}

var defaultOrganizationRepositoryOptions = organizationRepositoryOptions{
	Logger: slog.Default(),
}

// GlobalOrganizationRepositoryOptions is a list of [OrganizationRepositoryOption]s that are applied to all [OrganizationRepository]s.
var GlobalOrganizationRepositoryOptions []OrganizationRepositoryOption

// OrganizationRepositoryOption is an option for configuring a [OrganizationRepository].
type OrganizationRepositoryOption interface {
	apply(*organizationRepositoryOptions)
}

// funcOrganizationRepositoryOption is a [OrganizationRepositoryOption] that calls a function.
// It is used to wrap a function, so it satisfies the [OrganizationRepositoryOption] interface.
type funcOrganizationRepositoryOption struct {
	f func(*organizationRepositoryOptions)
}

func (fdo *funcOrganizationRepositoryOption) apply(opts *organizationRepositoryOptions) {
	fdo.f(opts)
}

func newFuncOrganizationRepositoryOption(f func(*organizationRepositoryOptions)) *funcOrganizationRepositoryOption {
	return &funcOrganizationRepositoryOption{
		f: f,
	}
}

// WithOrganizationRepositoryLogger returns a [OrganizationRepositoryOption] that uses the provided logger.
func WithOrganizationRepositoryLogger(logger *slog.Logger) OrganizationRepositoryOption {
	return newFuncOrganizationRepositoryOption(func(opts *organizationRepositoryOptions) {
		opts.Logger = logger
	})
}

// WithOrganizationRepositoryDb returns a [OrganizationRepositoryOption] that uses the provided database connection.
func WithOrganizationRepositoryDb(db PgxPoolInterface) OrganizationRepositoryOption {
	return newFuncOrganizationRepositoryOption(func(opts *organizationRepositoryOptions) {
		opts.Db = db
	})
}
