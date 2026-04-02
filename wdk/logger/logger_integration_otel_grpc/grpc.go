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

package logger_integration_otel_grpc

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
)

func init() {
	driver_handlers.RegisterOtlpProtocol("grpc", driver_handlers.OtlpProtocol{
		TraceExporterFactory:  createGrpcTraceExporter,
		MetricExporterFactory: createGrpcMetricExporter,
	})
}

// grpcMetadataCreds wraps gRPC metadata for use as per-RPC credentials.
type grpcMetadataCreds struct {
	// md holds the gRPC metadata key-value pairs to attach as per-RPC
	// credentials.
	md metadata.MD
}

// GetRequestMetadata returns the gRPC metadata as a map for per-RPC
// credentials. This implements the credentials.PerRPCCredentials interface.
//
// Returns map[string]string which contains the metadata headers.
// Returns error which is always nil.
func (c grpcMetadataCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	headers := make(map[string]string, len(c.md))
	for k, v := range c.md {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers, nil
}

// RequireTransportSecurity reports whether transport security is required.
// This implements the credentials.PerRPCCredentials interface.
//
// Returns bool which is always false.
func (grpcMetadataCreds) RequireTransportSecurity() bool {
	return false
}

// createGrpcExporter dials the configured endpoint and creates an OTLP gRPC
// exporter using the supplied constructor.
//
// Takes newExporter which receives the context and the established connection
// and returns the SDK-specific exporter.
//
// Returns any which is the constructed exporter.
// Returns io.Closer which is the gRPC connection to close on shutdown.
// Returns error when the gRPC connection or exporter cannot be created.
func createGrpcExporter(
	ctx context.Context,
	config driver_handlers.OtelSetupConfig,
	newExporter func(context.Context, *grpc.ClientConn) (any, error),
) (any, io.Closer, error) {
	grpcOpts, err := buildGrpcOptions(config)
	if err != nil {
		return nil, nil, err
	}
	conn, err := grpc.NewClient(config.Endpoint, grpcOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC connection to '%s': %w", config.Endpoint, err)
	}
	exporter, err := newExporter(ctx, conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return exporter, conn, nil
}

// createGrpcTraceExporter creates an OTLP trace exporter that uses gRPC.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and TLS settings.
//
// Returns any which is the trace exporter.
// Returns io.Closer which is the gRPC connection to close on shutdown.
// Returns error when the gRPC connection or exporter cannot be created.
func createGrpcTraceExporter(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
	return createGrpcExporter(ctx, config, func(ctx context.Context, conn *grpc.ClientConn) (any, error) {
		return otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	})
}

// createGrpcMetricExporter creates an OTLP metric exporter using gRPC.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies the OTLP
// endpoint and TLS settings.
//
// Returns any which is the metric exporter.
// Returns io.Closer which is the gRPC connection to close on shutdown.
// Returns error when the gRPC connection or exporter cannot be created.
func createGrpcMetricExporter(ctx context.Context, config driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
	return createGrpcExporter(ctx, config, func(ctx context.Context, conn *grpc.ClientConn) (any, error) {
		return otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	})
}

// buildGrpcOptions builds gRPC dial options from OTLP settings.
//
// Takes config (driver_handlers.OtelSetupConfig) which specifies TLS and
// header settings.
//
// Returns []grpc.DialOption which contains the configured dial options.
// Returns error which is always nil.
func buildGrpcOptions(config driver_handlers.OtelSetupConfig) ([]grpc.DialOption, error) {
	var grpcOpts []grpc.DialOption
	if config.TLSInsecure {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}
	if len(config.Headers) > 0 {
		grpcOpts = append(grpcOpts, grpc.WithPerRPCCredentials(grpcMetadataCreds{metadata.New(config.Headers)}))
	}
	return grpcOpts, nil
}
