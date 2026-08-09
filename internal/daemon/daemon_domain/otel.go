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

package daemon_domain

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the daemon_domain package.
	log = logger_domain.GetLogger("piko/internal/daemon/daemon_domain")

	// meter is the OpenTelemetry meter for the daemon_domain package.
	meter = otel.Meter("piko/internal/daemon/daemon_domain")

	// fileEventsProcessed counts the number of file system events processed by the watcher.
	fileEventsProcessed metric.Int64Counter

	// http2ProtocolErrors counts HTTP/2 protocol-level errors by type.
	http2ProtocolErrors metric.Int64Counter
)

// RecordHTTP2ProtocolError counts one HTTP/2 protocol-level error.
//
// The server adapter owns the HTTP/2 transport configuration but not the daemon's
// telemetry, so it reports errors through this entry point rather than reaching for the
// counter directly.
//
// Takes errType (string) which names the protocol error, in lowercase with underscores.
func RecordHTTP2ProtocolError(ctx context.Context, errType string) {
	recordProtocolError(ctx, http2ProtocolErrors, errType)
}

// recordProtocolError adds one to counter, tolerating an instrument that was never
// created.
//
// Instrument creation reports to the OpenTelemetry error handler rather than being fatal,
// so a counter can be left unset. The caller here is the HTTP/2 transport, where a nil
// dereference would take down a live connection over a metric.
//
// Takes counter (metric.Int64Counter) which receives the increment, or nil.
// Takes errType (string) which names the protocol error, in lowercase with underscores.
func recordProtocolError(ctx context.Context, counter metric.Int64Counter, errType string) {
	if counter == nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", errType),
	))
}

func init() {
	var err error

	fileEventsProcessed, err = meter.Int64Counter(
		"daemon.file_events_processed",
		metric.WithDescription("Number of file events processed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	http2ProtocolErrors, err = meter.Int64Counter(
		"daemon.http2_protocol_errors",
		metric.WithDescription("HTTP/2 protocol-level errors by type"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
