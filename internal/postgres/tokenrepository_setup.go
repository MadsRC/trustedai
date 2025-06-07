// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	options *tokenRepositoryOptions
}

// NewTokenRepository creates a new [TokenRepository].
func NewTokenRepository(options ...TokenRepositoryOption) (*TokenRepository, error) {
	opts := defaultTokenRepositoryOptions
	for _, opt := range GlobalTokenRepositoryOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &TokenRepository{
		options: &opts,
	}, nil
}

type tokenRepositoryOptions struct {
	Logger *slog.Logger
	Db     *pgxpool.Pool
}

var defaultTokenRepositoryOptions = tokenRepositoryOptions{
	Logger: slog.Default(),
}

// GlobalTokenRepositoryOptions is a list of [TokenRepositoryOption]s that are applied to all [TokenRepository]s.
var GlobalTokenRepositoryOptions []TokenRepositoryOption

// TokenRepositoryOption is an option for configuring a [TokenRepository].
type TokenRepositoryOption interface {
	apply(*tokenRepositoryOptions)
}

// funcTokenRepositoryOption is a [TokenRepositoryOption] that calls a function.
// It is used to wrap a function, so it satisfies the [TokenRepositoryOption] interface.
type funcTokenRepositoryOption struct {
	f func(*tokenRepositoryOptions)
}

func (fdo *funcTokenRepositoryOption) apply(opts *tokenRepositoryOptions) {
	fdo.f(opts)
}

func newFuncTokenRepositoryOption(f func(*tokenRepositoryOptions)) *funcTokenRepositoryOption {
	return &funcTokenRepositoryOption{
		f: f,
	}
}

// WithTokenRepositoryLogger returns a [TokenRepositoryOption] that uses the provided logger.
func WithTokenRepositoryLogger(logger *slog.Logger) TokenRepositoryOption {
	return newFuncTokenRepositoryOption(func(opts *tokenRepositoryOptions) {
		opts.Logger = logger
	})
}

// WithTokenRepositoryDb returns a [TokenRepositoryOption] that uses the provided database connection.
func WithTokenRepositoryDb(db *pgxpool.Pool) TokenRepositoryOption {
	return newFuncTokenRepositoryOption(func(opts *tokenRepositoryOptions) {
		opts.Db = db
	})
}
