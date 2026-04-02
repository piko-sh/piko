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

package logger_otel_sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// batchSpanQueueSize is the maximum number of spans buffered before export.
	batchSpanQueueSize = 1024

	// batchSpanExportMaxBatch is the maximum number of spans exported in a
	// single batch.
	batchSpanExportMaxBatch = 256
)

func init() {
	driver_handlers.RegisterOtelProviderFactory(createProviders)
}

// createProviders creates real OTEL SDK trace and metric providers. It is
// registered as the OtelProviderFactory during init().
//
// The additionalProcessors and additionalReaders slices contain values whose
// concrete types satisfy sdktrace.SpanProcessor and sdkmetric.Reader
// respectively. Elements that do not match these types are silently skipped.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies OTLP
// endpoint, protocol, and sampling settings.
// Takes additionalProcessors ([]any) which holds optional span processors.
// Takes additionalReaders ([]any) which holds optional metric readers.
//
// Returns driver_handlers.OtelProviderResult which contains the configured
// tracer and meter providers.
// Returns error when provider creation fails.
func createProviders(
	ctx context.Context,
	config driver_handlers.OtelSetupConfig,
	additionalProcessors []any,
	additionalReaders []any,
) (driver_handlers.OtelProviderResult, error) {
	sdkProcessors := collectTyped[sdktrace.SpanProcessor](additionalProcessors)
	sdkReaders := collectTyped[sdkmetric.Reader](additionalReaders)

	sdkTP, closers, err := buildTracerProvider(ctx, config, sdkProcessors)
	if err != nil {
		return driver_handlers.OtelProviderResult{}, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	sdkMP, meterClosers, err := buildMeterProvider(ctx, config, sdkReaders)
	if err != nil {
		return driver_handlers.OtelProviderResult{}, fmt.Errorf("failed to create meter provider: %w", err)
	}
	closers = append(closers, meterClosers...)

	return driver_handlers.OtelProviderResult{
		TracerProvider: sdkTP,
		MeterProvider:  sdkMP,
		Closers:        closers,
		ShutdownFunc:   buildShutdownFunc(sdkTP, sdkMP, closers),
	}, nil
}

// collectTyped extracts items of type T from a slice of
// any, skipping non-matching elements.
//
// Takes items ([]any) which holds the source items to
// filter.
//
// Returns []T which holds the matched items.
func collectTyped[T any](items []any) []T {
	var result []T
	for _, item := range items {
		if typed, ok := item.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

// buildTracerProvider creates either an OTLP or local
// tracer provider depending on the config.
//
// Takes config (driver_handlers.OtelSetupConfig) which
// specifies OTLP endpoint, protocol, and sampling settings.
// Takes processors ([]sdktrace.SpanProcessor) which holds
// additional span processors to register.
//
// Returns *sdktrace.TracerProvider which is the configured
// tracer provider.
// Returns []io.Closer which holds optional closers for OTLP
// connections.
// Returns error when provider creation fails.
func buildTracerProvider(ctx context.Context, config driver_handlers.OtelSetupConfig, processors []sdktrace.SpanProcessor) (*sdktrace.TracerProvider, []io.Closer, error) {
	if config.Enabled {
		tp, closer, err := createOtlpTracerProvider(ctx, config, processors)
		if err != nil {
			return nil, nil, err
		}
		var closers []io.Closer
		if closer != nil {
			closers = append(closers, closer)
		}
		return tp, closers, nil
	}
	tp, err := createLocalTracerProvider(config.TraceSampleRate, processors)
	return tp, nil, err
}

// buildMeterProvider creates either an OTLP or local meter
// provider depending on the config.
//
// Takes config (driver_handlers.OtelSetupConfig) which
// specifies OTLP endpoint, protocol, and sampling settings.
// Takes readers ([]sdkmetric.Reader) which holds additional
// metric readers to register.
//
// Returns *sdkmetric.MeterProvider which is the configured
// meter provider.
// Returns []io.Closer which holds optional closers for OTLP
// connections.
// Returns error when provider creation fails.
func buildMeterProvider(ctx context.Context, config driver_handlers.OtelSetupConfig, readers []sdkmetric.Reader) (*sdkmetric.MeterProvider, []io.Closer, error) {
	if config.Enabled {
		mp, closer, err := createOtlpMeterProvider(ctx, config, readers)
		if err != nil {
			return nil, nil, err
		}
		var closers []io.Closer
		if closer != nil {
			closers = append(closers, closer)
		}
		return mp, closers, nil
	}
	mp, err := createLocalMeterProvider(readers)
	return mp, nil, err
}

// buildShutdownFunc creates a shutdown function that
// flushes the tracer and meter providers and closes all
// OTLP connections.
//
// Takes sdkTP (*sdktrace.TracerProvider) which is the tracer
// provider to shut down.
// Takes sdkMP (*sdkmetric.MeterProvider) which is the meter
// provider to shut down.
// Takes closers ([]io.Closer) which holds OTLP connections
// to close.
//
// Returns func(context.Context) error which performs the
// ordered shutdown.
func buildShutdownFunc(sdkTP *sdktrace.TracerProvider, sdkMP *sdkmetric.MeterProvider, closers []io.Closer) func(context.Context) error {
	return func(ctx context.Context) error {
		var allErrors []error
		if sdkTP != nil {
			if err := sdkTP.Shutdown(ctx); err != nil {
				allErrors = append(allErrors, fmt.Errorf("tracerProvider shutdown: %w", err))
			}
		}
		if sdkMP != nil {
			if err := sdkMP.Shutdown(ctx); err != nil {
				allErrors = append(allErrors, fmt.Errorf("meterProvider shutdown: %w", err))
			}
		}
		for _, c := range closers {
			if err := c.Close(); err != nil {
				allErrors = append(allErrors, fmt.Errorf("OTLP connection close: %w", err))
			}
		}
		return errors.Join(allErrors...)
	}
}

// createOtlpTracerProvider creates an OpenTelemetry tracer provider with OTLP
// export configured based on the specified protocol.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and protocol.
// Takes additionalProcessors ([]sdktrace.SpanProcessor) which provides
// additional span processors to register (e.g., Sentry, monitoring service).
//
// Returns *sdktrace.TracerProvider which is the configured tracer provider.
// Returns io.Closer which is an optional closer (e.g. a gRPC connection) that
// should be closed during shutdown, or nil when not applicable.
// Returns error when the protocol is unsupported or the exporter cannot be
// created.
func createOtlpTracerProvider(ctx context.Context, config driver_handlers.OtelSetupConfig, additionalProcessors []sdktrace.SpanProcessor) (*sdktrace.TracerProvider, io.Closer, error) {
	protocol, ok := driver_handlers.GetOtlpProtocol(config.Protocol)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported OTLP protocol: '%s'. Import the appropriate module (e.g. logger_integration_otel_grpc for gRPC)", config.Protocol)
	}
	exporterAny, closer, err := protocol.TraceExporterFactory(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	exporter, ok := exporterAny.(sdktrace.SpanExporter)
	if !ok {
		if closer != nil {
			_ = closer.Close()
		}
		return nil, nil, fmt.Errorf("trace exporter factory returned %T, expected sdktrace.SpanExporter", exporterAny)
	}

	otelResource, err := createResource()
	if err != nil {
		_ = exporter.Shutdown(ctx)
		if closer != nil {
			_ = closer.Close()
		}
		return nil, nil, err
	}

	providerOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(otelResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.TraceSampleRate))),
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithBatchTimeout(1*time.Second),
		sdktrace.WithExportTimeout(10*time.Second),
		sdktrace.WithMaxQueueSize(batchSpanQueueSize),
		sdktrace.WithMaxExportBatchSize(batchSpanExportMaxBatch),
	)
	providerOptions = append(providerOptions, sdktrace.WithSpanProcessor(bsp))

	for _, processor := range additionalProcessors {
		providerOptions = append(providerOptions, sdktrace.WithSpanProcessor(processor))
	}

	return sdktrace.NewTracerProvider(providerOptions...), closer, nil
}

// createLocalTracerProvider creates a tracer provider without OTLP export,
// using only the provided span processors. This is used when the monitoring
// service is enabled but OTLP is not configured.
//
// Takes sampleRate (float64) which is the fraction of traces to sample (0.0 to
// 1.0).
// Takes processors ([]sdktrace.SpanProcessor) which are the processors to
// register.
//
// Returns *sdktrace.TracerProvider which is the configured tracer provider.
// Returns error when resource creation fails.
func createLocalTracerProvider(sampleRate float64, processors []sdktrace.SpanProcessor) (*sdktrace.TracerProvider, error) {
	otelResource, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource for local tracer provider: %w", err)
	}

	providerOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(otelResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
	}

	for _, processor := range processors {
		providerOptions = append(providerOptions, sdktrace.WithSpanProcessor(processor))
	}

	return sdktrace.NewTracerProvider(providerOptions...), nil
}

// createOtlpMeterProvider creates an OpenTelemetry meter provider using the
// specified OTLP configuration.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and protocol.
// Takes additionalReaders ([]sdkmetric.Reader) which provides additional
// metric readers to register (e.g., monitoring service).
//
// Returns *sdkmetric.MeterProvider which is configured with a periodic reader.
// Returns io.Closer which is an optional closer (e.g. a gRPC connection) that
// should be closed during shutdown, or nil when not applicable.
// Returns error when the protocol is unsupported or the exporter cannot be
// created.
func createOtlpMeterProvider(ctx context.Context, config driver_handlers.OtelSetupConfig, additionalReaders []sdkmetric.Reader) (*sdkmetric.MeterProvider, io.Closer, error) {
	protocol, ok := driver_handlers.GetOtlpProtocol(config.Protocol)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported OTLP protocol: '%s'. Import the appropriate module (e.g. logger_integration_otel_grpc for gRPC)", config.Protocol)
	}
	exporterAny, closer, err := protocol.MetricExporterFactory(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	exporter, ok := exporterAny.(sdkmetric.Exporter)
	if !ok {
		if closer != nil {
			_ = closer.Close()
		}
		return nil, nil, fmt.Errorf("metric exporter factory returned %T, expected sdkmetric.Exporter", exporterAny)
	}

	otelResource, err := createResource()
	if err != nil {
		_ = exporter.Shutdown(ctx)
		if closer != nil {
			_ = closer.Close()
		}
		return nil, nil, err
	}

	metricReader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(30*time.Second),
	)

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(otelResource),
		sdkmetric.WithReader(metricReader),
	}

	for _, reader := range additionalReaders {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	return sdkmetric.NewMeterProvider(opts...), closer, nil
}

// createLocalMeterProvider creates a meter provider without OTLP export,
// using only the provided metric readers. This is used when the monitoring
// service is enabled but OTLP is not configured.
//
// Takes readers ([]sdkmetric.Reader) which are the readers to register.
//
// Returns *sdkmetric.MeterProvider which is the configured meter provider.
// Returns error when resource creation fails.
func createLocalMeterProvider(readers []sdkmetric.Reader) (*sdkmetric.MeterProvider, error) {
	otelResource, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource for local meter provider: %w", err)
	}

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(otelResource),
	}

	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	return sdkmetric.NewMeterProvider(opts...), nil
}

// createResource builds an OpenTelemetry resource with service metadata and
// any environment attributes detected at startup (K8s, Lambda, Cloud Run,
// Azure Container Apps, Piko-specific overrides).
//
// Returns *resource.Resource which contains the merged default resource and
// service attributes.
// Returns error when the resource merge fails.
func createResource() (*resource.Resource, error) {
	envAttrs := logger_domain.EnvironmentOtelAttrs()
	attrs := make([]attribute.KeyValue, 0, len(envAttrs)+2)

	if !logger_domain.EnvironmentOverridesServiceName() {
		attrs = append(attrs, semconv.ServiceName("piko"))
	}
	if !logger_domain.EnvironmentOverridesServiceVersion() {
		attrs = append(attrs, semconv.ServiceVersion("0.1.0"))
	}
	attrs = append(attrs, envAttrs...)

	return resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	))
}
