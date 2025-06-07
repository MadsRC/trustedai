// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"log/slog"
)

type UserRepository struct {
	options *userRepositoryOptions
}

// NewUserRepository creates a new [UserRepository].
func NewUserRepository(options ...UserRepositoryOption) (*UserRepository, error) {
	opts := defaultUserRepositoryOptions
	for _, opt := range GlobalUserRepositoryOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &UserRepository{
		options: &opts,
	}, nil
}

type userRepositoryOptions struct {
	Logger *slog.Logger
	Db     PgxPoolInterface
}

var defaultUserRepositoryOptions = userRepositoryOptions{
	Logger: slog.Default(),
}

// GlobalUserRepositoryOptions is a list of [UserRepositoryOption]s that are applied to all [UserRepository]s.
var GlobalUserRepositoryOptions []UserRepositoryOption

// UserRepositoryOption is an option for configuring a [UserRepository].
type UserRepositoryOption interface {
	apply(*userRepositoryOptions)
}

// funcUserRepositoryOption is a [UserRepositoryOption] that calls a function.
// It is used to wrap a function, so it satisfies the [UserRepositoryOption] interface.
type funcUserRepositoryOption struct {
	f func(*userRepositoryOptions)
}

func (fdo *funcUserRepositoryOption) apply(opts *userRepositoryOptions) {
	fdo.f(opts)
}

func newFuncUserRepositoryOption(f func(*userRepositoryOptions)) *funcUserRepositoryOption {
	return &funcUserRepositoryOption{
		f: f,
	}
}

// WithUserRepositoryLogger returns a [UserRepositoryOption] that uses the provided logger.
func WithUserRepositoryLogger(logger *slog.Logger) UserRepositoryOption {
	return newFuncUserRepositoryOption(func(opts *userRepositoryOptions) {
		opts.Logger = logger
	})
}

// WithUserRepositoryDb returns a [UserRepositoryOption] that uses the provided database connection.
func WithUserRepositoryDb(db PgxPoolInterface) UserRepositoryOption {
	return newFuncUserRepositoryOption(func(opts *userRepositoryOptions) {
		opts.Db = db
	})
}
