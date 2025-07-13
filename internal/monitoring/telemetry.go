// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package monitoring

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
}

type TelemetryManager struct {
	meterProvider *sdkmetric.MeterProvider
	config        TelemetryConfig
}

func NewTelemetryManager(config TelemetryConfig) (*TelemetryManager, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	if config.OTLPEndpoint == "" {
		return nil, fmt.Errorf("OTLP endpoint is required")
	}

	otlpExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(config.OTLPEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	periodicReader := sdkmetric.NewPeriodicReader(otlpExporter)
	log.Printf("OTLP metrics enabled, endpoint: %s", config.OTLPEndpoint)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(periodicReader),
	)

	otel.SetMeterProvider(meterProvider)

	return &TelemetryManager{
		meterProvider: meterProvider,
		config:        config,
	}, nil
}

func (tm *TelemetryManager) GetMeter(instrumentationName string) metric.Meter {
	return tm.meterProvider.Meter(instrumentationName)
}

func (tm *TelemetryManager) Shutdown(ctx context.Context) error {
	return tm.meterProvider.Shutdown(ctx)
}
