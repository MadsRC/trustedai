// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"io"
	"log/slog"
	"testing"
)

func TestNewSsoCallback(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []SsoCallbackOption
		want    *SsoCallback
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []SsoCallbackOption{},
			want: &SsoCallback{
				options: &ssoCallbackOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []SsoCallbackOption{WithSsoCallbackLogger(discardLogger)},
			want: &SsoCallback{
				options: &ssoCallbackOptions{
					Logger: discardLogger,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSsoCallback(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSsoCallback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewSsoCallback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSsoCallback_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []SsoCallbackOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []SsoCallbackOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalSsoCallbackOptions = []SsoCallbackOption{
				WithSsoCallbackLogger(tt.inputLogger),
			}
			got1, _ := NewSsoCallback(tt.options...)
			got2, _ := NewSsoCallback(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewSsoCallback() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewSsoCallback() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewSsoCallback() = %v, want %v", got1, got2)
			}
			GlobalSsoCallbackOptions = []SsoCallbackOption{}
			got3, _ := NewSsoCallback(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewSsoCallback() = %v, want %v", got3, slog.Default())
			}
		})
	}
}
