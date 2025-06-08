// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package controlplane

import (
	"io"
	"log/slog"
	"testing"
)

func TestNewControlPlaneServer(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []ControlPlaneOption
		want    *ControlPlaneServer
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []ControlPlaneOption{},
			want: &ControlPlaneServer{
				options: &controlPlaneOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []ControlPlaneOption{WithControlPlaneLogger(discardLogger)},
			want: &ControlPlaneServer{
				options: &controlPlaneOptions{
					Logger: discardLogger,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewControlPlaneServer(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewControlPlaneServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewControlPlaneServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewControlPlaneServer_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []ControlPlaneOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []ControlPlaneOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalControlPlaneOptions = []ControlPlaneOption{
				WithControlPlaneLogger(tt.inputLogger),
			}
			got1, _ := NewControlPlaneServer(tt.options...)
			got2, _ := NewControlPlaneServer(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewControlPlaneServer() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewControlPlaneServer() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewControlPlaneServer() = %v, want %v", got1, got2)
			}
			GlobalControlPlaneOptions = []ControlPlaneOption{}
			got3, _ := NewControlPlaneServer(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewControlPlaneServer() = %v, want %v", got3, slog.Default())
			}
		})
	}
}
