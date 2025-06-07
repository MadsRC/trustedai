// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"io"
	"log/slog"
	"testing"
)

func TestNewUserRepository(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []UserRepositoryOption
		want    *UserRepository
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []UserRepositoryOption{},
			want: &UserRepository{
				options: &userRepositoryOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []UserRepositoryOption{WithUserRepositoryLogger(discardLogger)},
			want: &UserRepository{
				options: &userRepositoryOptions{
					Logger: discardLogger,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUserRepository(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUserRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewUserRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUserRepository_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []UserRepositoryOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []UserRepositoryOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalUserRepositoryOptions = []UserRepositoryOption{
				WithUserRepositoryLogger(tt.inputLogger),
			}
			got1, _ := NewUserRepository(tt.options...)
			got2, _ := NewUserRepository(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewUserRepository() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewUserRepository() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewUserRepository() = %v, want %v", got1, got2)
			}
			GlobalUserRepositoryOptions = []UserRepositoryOption{}
			got3, _ := NewUserRepository(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewUserRepository() = %v, want %v", got3, slog.Default())
			}
		})
	}
}
