// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package driver_handlers

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/logger/logger_domain"
)

// OtelSetupOptions provides additional configuration for OTEL setup.
type OtelSetupOptions struct {
	// AdditionalSpanProcessors contains span processors to register with the
	// tracer provider. Concrete types must satisfy sdktrace.SpanProcessor when
	// an SDK provider factory is registered.
	AdditionalSpanProcessors []any

	// AdditionalMetricReaders are extra metric readers to register with the
	// meter provider alongside any default readers. Concrete types must satisfy
	// sdkmetric.Reader when an SDK provider factory is registered.
	AdditionalMetricReaders []any
}

var (
	// otelInitialised tracks whether OTEL providers have been set up with
	// additional processors or readers. This prevents subsequent calls to
	// SetupOtel from overwriting the configured providers (e.g., when
	// logger.Apply is called after logger.Initialise).
	otelInitialised bool
)

// SetupOtel initialises OpenTelemetry tracing and metrics providers based on
// configuration. It returns a shutdown function to gracefully close all
// providers and any initialisation errors.
//
// The enabledIntegrations parameter contains the type names of integrations that
// are both enabled in config and have their adapter package imported. Allows
// collection of OTel components from all relevant integrations without knowing
// about them.
//
// If OTEL was already set up with additional processors/readers, skips
// re-initialisation to avoid overwriting the configured providers.
//
// When no OtelProviderFactory has been registered (i.e. the SDK module is not
// imported), noop providers are used even when config.Enabled is true.
//
// Takes config (OtelSetupConfig) which provides endpoint, protocol, and TLS
// settings.
// Takes enabledIntegrations ([]string) which lists integration type names
// that are both enabled and imported.
// Takes opts (*OtelSetupOptions) which provides additional span processors
// and metric readers; may be nil.
//
// Returns shutdown (func(context.Context) error) which gracefully closes all
// providers.
// Returns err (error) when provider creation fails.
func SetupOtel(ctx context.Context, config OtelSetupConfig, enabledIntegrations []string, opts *OtelSetupOptions) (shutdown func(context.Context) error, err error) {
	if otelInitialised && opts == nil {
		slog.Debug("SetupOtel: skipping re-initialisation, OTEL already configured with additional processors/readers")
		return func(context.Context) error { return nil }, nil
	}
	var tracerProvider trace.TracerProvider
	var meterProvider metric.MeterProvider

	var integrationProcessors []any
	var integrationPropagators []propagation.TextMapPropagator

	otelIntegrations := logger_domain.GetEnabledOtelIntegrations(enabledIntegrations)
	for _, integration := range otelIntegrations {
		processor, propagator := integration.OtelComponents()
		if processor != nil {
			integrationProcessors = append(integrationProcessors, processor)
			slog.Debug("Added OTel span processor from integration", slog.String("type", integration.Type()))
		}
		if propagator != nil {
			integrationPropagators = append(integrationPropagators, propagator)
			slog.Info("OTel propagator enabled from integration", slog.String("type", integration.Type()))
		}
	}

	var additionalProcessors []any
	additionalProcessors = append(additionalProcessors, integrationProcessors...)
	if opts != nil {
		additionalProcessors = append(additionalProcessors, opts.AdditionalSpanProcessors...)
	}

	var additionalReaders []any
	if opts != nil {
		additionalReaders = opts.AdditionalMetricReaders
	}

	needsRealProviders := config.Enabled || len(additionalProcessors) > 0 || len(additionalReaders) > 0

	var shutdownFunc func(context.Context) error

	if needsRealProviders {
		factory := getOtelProviderFactory()
		if factory != nil {
			result, err := factory(ctx, config, additionalProcessors, additionalReaders)
			if err != nil {
				return nil, fmt.Errorf("creating OTEL providers: %w", err)
			}
			tracerProvider = result.TracerProvider
			meterProvider = result.MeterProvider
			shutdownFunc = result.ShutdownFunc
		} else {
			slog.Warn("OTEL SDK not available: no provider factory registered. Import piko.sh/piko/wdk/logger/logger_otel_sdk to enable OTEL SDK support. Falling back to noop providers.")
			tracerProvider = tracenoop.NewTracerProvider()
			meterProvider = metricnoop.NewMeterProvider()
		}
	} else {
		tracerProvider = tracenoop.NewTracerProvider()
		meterProvider = metricnoop.NewMeterProvider()
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	propagators := make([]propagation.TextMapPropagator, 0, 2+len(integrationPropagators))
	propagators = append(propagators, propagation.TraceContext{}, propagation.Baggage{})
	propagators = append(propagators, integrationPropagators...)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	if shutdownFunc == nil {
		shutdownFunc = func(context.Context) error { return nil }
	}

	if opts != nil && (len(opts.AdditionalSpanProcessors) > 0 || len(opts.AdditionalMetricReaders) > 0) {
		otelInitialised = true
		slog.Debug("SetupOtel: marked as initialised with additional processors/readers")
	}

	return shutdownFunc, nil
}
