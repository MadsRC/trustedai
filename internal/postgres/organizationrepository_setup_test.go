// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"io"
	"log/slog"
	"testing"
)

func TestNewOrganizationRepository(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name    string
		options []OrganizationRepositoryOption
		want    *OrganizationRepository
		wantErr bool
	}{
		{
			name:    "Create with default logger",
			options: []OrganizationRepositoryOption{},
			want: &OrganizationRepository{
				options: &organizationRepositoryOptions{
					Logger: slog.Default(),
				},
			},
			wantErr: false,
		},
		{
			name:    "Create with custom logger",
			options: []OrganizationRepositoryOption{WithOrganizationRepositoryLogger(discardLogger)},
			want: &OrganizationRepository{
				options: &organizationRepositoryOptions{
					Logger: discardLogger,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrganizationRepository(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOrganizationRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.options.Logger != tt.want.options.Logger {
				t.Errorf("NewOrganizationRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOrganizationRepository_GlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []OrganizationRepositoryOption
		inputLogger *slog.Logger
	}{
		{
			name:        "Global options are applied",
			options:     []OrganizationRepositoryOption{},
			inputLogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalOrganizationRepositoryOptions = []OrganizationRepositoryOption{
				WithOrganizationRepositoryLogger(tt.inputLogger),
			}
			got1, _ := NewOrganizationRepository(tt.options...)
			got2, _ := NewOrganizationRepository(tt.options...)
			if got1.options.Logger != tt.inputLogger {
				t.Errorf("NewOrganizationRepository() = %v, want %v", got1, tt.inputLogger)
			}
			if got2.options.Logger != tt.inputLogger {
				t.Errorf("NewOrganizationRepository() = %v, want %v", got2, tt.inputLogger)
			}
			if got1.options.Logger != got2.options.Logger {
				t.Errorf("NewOrganizationRepository() = %v, want %v", got1, got2)
			}
			GlobalOrganizationRepositoryOptions = []OrganizationRepositoryOption{}
			got3, _ := NewOrganizationRepository(tt.options...)
			if got3.options.Logger == tt.inputLogger {
				t.Errorf("NewOrganizationRepository() = %v, want %v", got3, slog.Default())
			}
		})
	}
}
