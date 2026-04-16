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
	"io"
	"strings"
	"sync"
)

// OtlpTraceExporterFactory creates an OTLP trace exporter for the given protocol.
//
// Returns the exporter (as any, concrete type is sdktrace.SpanExporter) and an
// optional closer (e.g. a gRPC connection) that should be closed during shutdown.
type OtlpTraceExporterFactory func(ctx context.Context, config OtelSetupConfig) (exporter any, closer io.Closer, err error)

// OtlpMetricExporterFactory creates an OTLP metric exporter for the given
// protocol. Returns the exporter (as any, concrete type is
// sdkmetric.Exporter) and an optional closer that should be closed during
// shutdown.
type OtlpMetricExporterFactory func(ctx context.Context, config OtelSetupConfig) (exporter any, closer io.Closer, err error)

// OtlpProtocol bundles trace and metric exporter factories for a named
// protocol (e.g. "grpc", "http").
type OtlpProtocol struct {
	// TraceExporterFactory holds the factory for creating trace exporters.
	TraceExporterFactory OtlpTraceExporterFactory

	// MetricExporterFactory holds the factory for creating metric exporters.
	MetricExporterFactory OtlpMetricExporterFactory
}

var (
	// protocolsMu guards concurrent access to protocols.
	protocolsMu sync.RWMutex

	// protocols holds the registered OTLP protocol factories keyed by name.
	protocols = map[string]OtlpProtocol{}
)

// RegisterOtlpProtocol registers an OTLP protocol by name.
//
// Typically called from init() in a wdk module (e.g. logger_integration_otel_grpc).
//
// Takes name (string) which identifies the protocol (e.g. "grpc", "http").
// Takes protocol (OtlpProtocol) which holds the exporter factories.
//
// Safe for concurrent use. Protected by a package-level mutex.
func RegisterOtlpProtocol(name string, protocol OtlpProtocol) {
	protocolsMu.Lock()
	defer protocolsMu.Unlock()
	protocols[strings.ToLower(name)] = protocol
}

// GetOtlpProtocol looks up a registered protocol by name.
//
// Takes name (string) which identifies the protocol to look up.
//
// Returns OtlpProtocol which holds the exporter factories.
// Returns bool which indicates whether the protocol was found.
//
// Safe for concurrent use. Protected by a package-level mutex.
func GetOtlpProtocol(name string) (OtlpProtocol, bool) {
	protocolsMu.RLock()
	defer protocolsMu.RUnlock()
	p, ok := protocols[strings.ToLower(name)]
	return p, ok
}
