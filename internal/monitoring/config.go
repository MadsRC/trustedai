// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package monitoring

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
}

type Manager struct {
	telemetry    *TelemetryManager
	usageMetrics *UsageMetrics
	config       Config
}

func NewManager(config Config) (*Manager, error) {
	telemetryConfig := TelemetryConfig(config)

	telemetry, err := NewTelemetryManager(telemetryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry manager: %w", err)
	}

	meter := telemetry.GetMeter("codeberg.org/MadsRC/llmgw/usage")
	usageMetrics, err := NewUsageMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage metrics: %w", err)
	}

	return &Manager{
		telemetry:    telemetry,
		usageMetrics: usageMetrics,
		config:       config,
	}, nil
}

func (m *Manager) GetUsageMetrics() *UsageMetrics {
	return m.usageMetrics
}

func (m *Manager) GetMeter(instrumentationName string) metric.Meter {
	return m.telemetry.GetMeter(instrumentationName)
}

func (m *Manager) Shutdown(ctx context.Context) error {
	return m.telemetry.Shutdown(ctx)
}
