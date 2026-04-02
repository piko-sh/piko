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

package logger_integration_otel_http

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
)

func init() {
	httpTraceFactory := func(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
		exp, err := createOtlpHTTPTraceExporter(ctx, config)
		if err != nil {
			return nil, nil, err
		}
		return exp, nil, nil
	}
	httpMetricFactory := func(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
		exp, err := createOtlpHTTPMetricExporter(ctx, config)
		if err != nil {
			return nil, nil, err
		}
		return exp, nil, nil
	}
	driver_handlers.RegisterOtlpProtocol("http", driver_handlers.OtlpProtocol{
		TraceExporterFactory:  httpTraceFactory,
		MetricExporterFactory: httpMetricFactory,
	})
	driver_handlers.RegisterOtlpProtocol("https", driver_handlers.OtlpProtocol{
		TraceExporterFactory:  httpTraceFactory,
		MetricExporterFactory: httpMetricFactory,
	})
}

// otlpHTTPOptionBuilders bundles the four SDK-specific option constructor
// functions, reducing the parameter count of createOtlpHTTPExporter.
type otlpHTTPOptionBuilders[O any] struct {
	// withEndpoint constructs the endpoint option.
	withEndpoint func(string) O

	// withURLPath constructs the URL-path option.
	withURLPath func(string) O

	// withInsecure constructs the TLS-insecure option.
	withInsecure func() O

	// withHeaders constructs the headers option.
	withHeaders func(map[string]string) O
}

// createOtlpHTTPExporter builds an OTLP HTTP exporter using SDK-specific option
// constructors supplied by the caller.
//
// Takes pathSuffix (string) which is appended to the URL path (e.g. "/v1/metrics").
// Takes builders (otlpHTTPOptionBuilders[O]) which bundles the
// SDK-specific option constructors.
// Takes newExporter which constructs the final exporter from the collected options.
//
// Returns any which is the constructed exporter.
// Returns error when the endpoint is invalid or the exporter cannot be created.
func createOtlpHTTPExporter[O any](
	ctx context.Context,
	config driver_handlers.OtelSetupConfig,
	pathSuffix string,
	builders otlpHTTPOptionBuilders[O],
	newExporter func(context.Context, ...O) (any, error),
) (any, error) {
	endpoint, path, err := normaliseHTTPEndpoint(config)
	if err != nil {
		return nil, fmt.Errorf("normalising HTTP endpoint: %w", err)
	}

	opts := []O{
		builders.withEndpoint(endpoint),
		builders.withURLPath(path + pathSuffix),
	}

	if config.TLSInsecure {
		opts = append(opts, builders.withInsecure())
	}
	if len(config.Headers) > 0 {
		opts = append(opts, builders.withHeaders(config.Headers))
	}

	return newExporter(ctx, opts...)
}

// createOtlpHTTPMetricExporter creates an OTLP HTTP exporter for metrics.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and protocol.
//
// Returns any which is the metric exporter.
// Returns error when the endpoint is invalid or the exporter cannot be created.
func createOtlpHTTPMetricExporter(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, error) {
	return createOtlpHTTPExporter(
		ctx, config, "/v1/metrics",
		otlpHTTPOptionBuilders[otlpmetrichttp.Option]{
			withEndpoint: otlpmetrichttp.WithEndpoint,
			withURLPath:  otlpmetrichttp.WithURLPath,
			withInsecure: otlpmetrichttp.WithInsecure,
			withHeaders:  otlpmetrichttp.WithHeaders,
		},
		func(ctx context.Context, opts ...otlpmetrichttp.Option) (any, error) {
			return otlpmetrichttp.New(ctx, opts...)
		},
	)
}

// createOtlpHTTPTraceExporter creates an OTLP HTTP trace exporter.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and protocol.
//
// Returns any which is the trace exporter.
// Returns error when the endpoint is invalid or the exporter cannot be created.
func createOtlpHTTPTraceExporter(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, error) {
	return createOtlpHTTPExporter(
		ctx, config, "/v1/traces",
		otlpHTTPOptionBuilders[otlptracehttp.Option]{
			withEndpoint: otlptracehttp.WithEndpoint,
			withURLPath:  otlptracehttp.WithURLPath,
			withInsecure: otlptracehttp.WithInsecure,
			withHeaders:  otlptracehttp.WithHeaders,
		},
		func(ctx context.Context, opts ...otlptracehttp.Option) (any, error) {
			return otlptracehttp.New(ctx, opts...)
		},
	)
}

// normaliseHTTPEndpoint extracts the host and path from the OTLP endpoint,
// adding a scheme prefix if missing.
//
// Takes config (driver_handlers.OtelSetupConfig) which provides the endpoint
// and protocol.
//
// Returns endpoint (string) which is the host portion.
// Returns path (string) which is the URL path portion.
// Returns err (error) when the URL cannot be parsed.
func normaliseHTTPEndpoint(config driver_handlers.OtelSetupConfig) (endpoint, path string, err error) {
	fullURL := config.Endpoint
	if !strings.HasPrefix(fullURL, "http") {
		if strings.EqualFold(config.Protocol, "https") {
			fullURL = "https://" + fullURL
		} else {
			fullURL = "http://" + fullURL
		}
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid OTLP http endpoint URL: %w", err)
	}

	return u.Host, u.Path, nil
}
