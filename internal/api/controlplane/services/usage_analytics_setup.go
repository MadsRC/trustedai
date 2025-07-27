// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"log/slog"

	"github.com/MadsRC/trustedai"
)

// UsageAnalyticsOptions holds the configuration options for the usage analytics service
type UsageAnalyticsOptions struct {
	Logger            *slog.Logger
	UserRepository    trustedai.UserRepository
	UsageRepository   trustedai.UsageRepository
	BillingRepository trustedai.BillingRepository
}

// NewUsageAnalytics creates a new UsageAnalytics service with the provided options
func NewUsageAnalytics(options ...UsageAnalyticsOption) (*UsageAnalytics, error) {
	opts := defaultUsageAnalyticsOptions
	for _, opt := range GlobalUsageAnalyticsOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	return &UsageAnalytics{
		options: opts,
	}, nil
}

var defaultUsageAnalyticsOptions = UsageAnalyticsOptions{
	Logger: slog.Default(),
}

// GlobalUsageAnalyticsOptions is a list of UsageAnalyticsOption that are applied to all UsageAnalytics services
var GlobalUsageAnalyticsOptions []UsageAnalyticsOption

// UsageAnalyticsOption is an option for configuring a UsageAnalytics service
type UsageAnalyticsOption interface {
	apply(*UsageAnalyticsOptions)
}

// funcUsageAnalyticsOption is a UsageAnalyticsOption that calls a function
type funcUsageAnalyticsOption struct {
	f func(*UsageAnalyticsOptions)
}

func (fdo *funcUsageAnalyticsOption) apply(opts *UsageAnalyticsOptions) {
	fdo.f(opts)
}

func newFuncUsageAnalyticsOption(f func(*UsageAnalyticsOptions)) *funcUsageAnalyticsOption {
	return &funcUsageAnalyticsOption{
		f: f,
	}
}

// WithUsageAnalyticsLogger returns a UsageAnalyticsOption that uses the provided logger
func WithUsageAnalyticsLogger(logger *slog.Logger) UsageAnalyticsOption {
	return newFuncUsageAnalyticsOption(func(opts *UsageAnalyticsOptions) {
		opts.Logger = logger
	})
}

// WithUsageAnalyticsUserRepository returns a UsageAnalyticsOption that uses the provided user repository
func WithUsageAnalyticsUserRepository(repo trustedai.UserRepository) UsageAnalyticsOption {
	return newFuncUsageAnalyticsOption(func(opts *UsageAnalyticsOptions) {
		opts.UserRepository = repo
	})
}

// WithUsageRepository returns a UsageAnalyticsOption that uses the provided usage repository
func WithUsageRepository(repo trustedai.UsageRepository) UsageAnalyticsOption {
	return newFuncUsageAnalyticsOption(func(opts *UsageAnalyticsOptions) {
		opts.UsageRepository = repo
	})
}

// WithBillingRepository returns a UsageAnalyticsOption that uses the provided billing repository
func WithBillingRepository(repo trustedai.BillingRepository) UsageAnalyticsOption {
	return newFuncUsageAnalyticsOption(func(opts *UsageAnalyticsOptions) {
		opts.BillingRepository = repo
	})
}
