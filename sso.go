// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package llmgw

import (
	"context"
	"time"
)

type DeviceAuthResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresIn       time.Duration
	Interval        time.Duration
}

type SsoProvider interface {
	GetAuthURL(ctx context.Context, state string) (string, error)
	HandleCallback(ctx context.Context, code string) (*User, error)
	ValidateToken(ctx context.Context, token string) (bool, map[string]any, error)

	// Device Flow additions
	StartDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error)
	CheckDeviceAuth(ctx context.Context, deviceCode string) (*User, error)
}
