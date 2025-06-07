// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"io"
	"log/slog"
	"testing"
)

func TestNewTokenRepository(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []TokenRepositoryOption
		want    *TokenRepository
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []TokenRepositoryOption{},
			want: &TokenRepository{
				options: &tokenRepositoryOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []TokenRepositoryOption{WithTokenRepositoryLogger(discardLogger)},
			want: &TokenRepository{
				options: &tokenRepositoryOptions{
					Logger: discardLogger,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTokenRepository(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTokenRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewTokenRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTokenRepository_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []TokenRepositoryOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []TokenRepositoryOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalTokenRepositoryOptions = []TokenRepositoryOption{
				WithTokenRepositoryLogger(tt.inputLogger),
			}
			got1, _ := NewTokenRepository(tt.options...)
			got2, _ := NewTokenRepository(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewTokenRepository() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewTokenRepository() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewTokenRepository() = %v, want %v", got1, got2)
			}
			GlobalTokenRepositoryOptions = []TokenRepositoryOption{}
			got3, _ := NewTokenRepository(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewTokenRepository() = %v, want %v", got3, slog.Default())
			}
		})
	}
}
