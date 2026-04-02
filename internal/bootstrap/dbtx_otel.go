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

package bootstrap

import (
	"context"
	"database/sql"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var _ DBTX = (*otelDBTX)(nil)

// otelDBTX wraps a DBTX with OpenTelemetry tracing and metrics, creating a
// span and recording duration metrics for each database call. When the OTel
// SDK is not configured, no-op providers ensure zero overhead.
type otelDBTX struct {
	// inner is the underlying database connection being instrumented.
	inner DBTX

	// tracer is the OTel tracer used to create spans for database operations.
	tracer trace.Tracer

	// resolver maps a SQL query string to a human-readable operation name.
	resolver func(string) string

	// databaseSystem is the OTel db.system attribute value (e.g. "postgresql").
	databaseSystem string

	// databaseNamespace is the registered database name used in span names.
	databaseNamespace string
}

// newOTelDBTX creates an instrumented DBTX wrapper.
//
// Takes inner (DBTX) which is the underlying database connection.
// Takes databaseSystem (string) which is the OTel db.system value
// (e.g. "postgresql", "mysql", "sqlite").
// Takes databaseNamespace (string) which is the registered database name
// (e.g. "tasks").
// Takes resolver (func(string) string) which maps a SQL query string to a
// human-readable operation name. May be nil, in which case operations are
// reported as "UNKNOWN".
//
// Returns *otelDBTX which implements DBTX with instrumentation.
func newOTelDBTX(
	inner DBTX,
	databaseSystem string,
	databaseNamespace string,
	resolver func(string) string,
) *otelDBTX {
	return &otelDBTX{
		inner:             inner,
		databaseSystem:    databaseSystem,
		databaseNamespace: databaseNamespace,
		resolver:          resolver,
		tracer:            otel.Tracer("piko/db/" + databaseNamespace),
	}
}

// ExecContext executes a query without returning rows, wrapped with a span
// and metric recording.
//
// Takes ctx (context.Context) for cancellation and span propagation.
// Takes query (string) which is the SQL query.
// Takes arguments (...any) which are the query parameters.
//
// Returns sql.Result from the underlying DBTX.
// Returns error from the underlying DBTX, also recorded on the span.
func (o *otelDBTX) ExecContext(ctx context.Context, query string, arguments ...any) (sql.Result, error) {
	operation := o.resolveOperation(query)
	ctx, span := o.startSpan(ctx, operation)
	defer span.End()

	start := time.Now()
	result, err := o.inner.ExecContext(ctx, query, arguments...)
	o.recordMetrics(ctx, operation, start, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return result, err
}

// QueryContext executes a query that returns rows, wrapped with a span and
// metric recording.
//
// Takes ctx (context.Context) for cancellation and span propagation.
// Takes query (string) which is the SQL query.
// Takes arguments (...any) which are the query parameters.
//
// Returns *sql.Rows from the underlying DBTX.
// Returns error from the underlying DBTX, also recorded on the span.
func (o *otelDBTX) QueryContext(ctx context.Context, query string, arguments ...any) (*sql.Rows, error) {
	operation := o.resolveOperation(query)
	ctx, span := o.startSpan(ctx, operation)
	defer span.End()

	start := time.Now()
	rows, err := o.inner.QueryContext(ctx, query, arguments...)
	o.recordMetrics(ctx, operation, start, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return rows, err
}

// QueryRowContext executes a query that returns at most one row, wrapped with
// a span and duration metric. Errors are deferred to row.Scan and cannot be
// captured at the DBTX level.
//
// Takes ctx (context.Context) for cancellation and span propagation.
// Takes query (string) which is the SQL query.
// Takes arguments (...any) which are the query parameters.
//
// Returns *sql.Row from the underlying DBTX.
func (o *otelDBTX) QueryRowContext(ctx context.Context, query string, arguments ...any) *sql.Row {
	operation := o.resolveOperation(query)
	ctx, span := o.startSpan(ctx, operation)
	defer span.End()

	start := time.Now()
	row := o.inner.QueryRowContext(ctx, query, arguments...)
	o.recordMetrics(ctx, operation, start, nil)

	return row
}

// resolveOperation maps a SQL query string to a human-readable operation name
// using the configured resolver.
//
// Takes query (string) which is the SQL query to resolve.
//
// Returns string which is the operation name, or "UNKNOWN" when the resolver
// is nil or returns an empty string.
func (o *otelDBTX) resolveOperation(query string) string {
	if o.resolver != nil {
		if name := o.resolver(query); name != "" {
			return name
		}
	}
	return "UNKNOWN"
}

// startSpan creates a new trace span with standard database attributes.
//
// Takes ctx (context.Context) which carries the parent span context.
// Takes operation (string) which is the operation name for the span.
//
// Returns context.Context which carries the new span.
// Returns trace.Span which is the created span.
func (o *otelDBTX) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	return o.tracer.Start(ctx, o.databaseNamespace+" "+operation,
		trace.WithAttributes(
			attribute.String("db.system", o.databaseSystem),
			attribute.String("db.namespace", o.databaseNamespace),
			attribute.String("db.operation.name", operation),
		),
	)
}

// recordMetrics records duration and count metrics for a database operation.
//
// Takes ctx (context.Context) which carries the metric context.
// Takes operation (string) which is the operation name for metric attributes.
// Takes start (time.Time) which is the operation start time for duration
// calculation.
// Takes err (error) which, when non-nil, increments the error counter.
func (o *otelDBTX) recordMetrics(ctx context.Context, operation string, start time.Time, err error) {
	duration := float64(time.Since(start).Milliseconds())
	attributes := metric.WithAttributeSet(attribute.NewSet(
		attribute.String("db.system", o.databaseSystem),
		attribute.String("db.namespace", o.databaseNamespace),
		attribute.String("db.operation.name", operation),
	))

	dbOperationDuration.Record(ctx, duration, attributes)
	dbOperationCount.Add(ctx, 1, attributes)

	if err != nil {
		dbOperationErrorCount.Add(ctx, 1, attributes)
	}
}
