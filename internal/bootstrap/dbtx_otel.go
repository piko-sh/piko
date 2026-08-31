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
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// maxOperationTokens bounds how much of a statement the operation label reads.
	maxOperationTokens = 16

	// maxCommentScan bounds the search for a leading comment's terminator, so an
	// unterminated comment cannot turn the label into a scan of the whole statement.
	maxCommentScan = 512
)

var (
	_ DBTX = (*otelDBTX)(nil)
)

const (
	// maxOperationLabels bounds how many distinct operation labels one wrapper emits.
	maxOperationLabels = 256

	// maxOperationIdentifierLen bounds the table identifier used in a label.
	maxOperationIdentifierLen = 64

	// maxLeadingComments bounds how many stacked comments are skipped before the verb.
	maxLeadingComments = 8
)

var (
	// nonTableKeywords are words that can appear where a table name is expected but never
	// name one: a sub-query, a values list, or a Postgres inheritance qualifier.
	nonTableKeywords = map[string]struct{}{
		"SELECT": {}, "VALUES": {}, "ONLY": {}, "LATERAL": {}, "UNNEST": {},
	}

	// operationTableKeyword maps a SQL verb to the keyword that precedes its primary table,
	// so deriveOperationName can find the table without parsing the statement.
	operationTableKeyword = map[string]string{
		"WITH":    "FROM",
		"EXPLAIN": "FROM",
		"SELECT":  "FROM",
		"DELETE":  "FROM",
		"INSERT":  "INTO",
		"UPDATE":  "UPDATE",
		"REPLACE": "INTO",
	}
)

// otelDBTX wraps a DBTX with OpenTelemetry tracing and metrics, creating a span and
// recording duration metrics for each database call. When the OTel SDK is not configured,
// no-op providers ensure zero overhead.
type otelDBTX struct {
	// inner is the underlying database connection being instrumented.
	inner DBTX

	// tracer is the OTel tracer used to create spans for database operations.
	tracer trace.Tracer

	// observer receives a QueryObservation after each call (nil when WithQueryObserver was
	// not used). It is the per-query telemetry forwarding seam.
	observer monitoring_domain.QueryObserver

	// resolver maps a SQL query string to a human-readable operation name.
	resolver func(string) string

	// operationLabels bounds and caches the operation labels this wrapper emits.
	operationLabels *operationLabelSet

	// reportOnce keeps the "instrumentation not applied" warning to one line per database.
	reportOnce *sync.Once

	// databaseSystem is the OTel db.system attribute value (e.g. "postgresql").
	databaseSystem string

	// databaseNamespace is the registered database name used in span names.
	databaseNamespace string
}

// operationLabelSet caches the label derived from each statement and bounds how many
// distinct labels a wrapper will ever emit.
type operationLabelSet struct {
	// labels maps a statement to its derived label.
	labels sync.Map

	// count tracks how many distinct labels are held, so the bound can be enforced without
	// walking the map.
	count atomic.Int64
}

// labelFor returns the cached label for a statement, deriving it when the set has room.
//
// Takes query (string) which is the statement to label.
//
// Returns string which is the label to report, reduced to the bare verb once the set is
// full.
func (set *operationLabelSet) labelFor(query string) string {
	if cached, ok := set.labels.Load(query); ok {
		label, _ := cached.(string)

		return label
	}

	label := deriveOperationName(query)

	if set.count.Load() >= maxOperationLabels {
		return operationVerb(label)
	}

	if _, loaded := set.labels.LoadOrStore(query, label); !loaded {
		set.count.Add(1)
	}

	return label
}

// newOTelDBTX creates an instrumented DBTX wrapper.
//
// Takes inner (DBTX) which is the underlying database connection.
// Takes databaseSystem (string) which is the OTel db.system value (e.g. "postgresql",
// "mysql", "sqlite").
// Takes databaseNamespace (string) which is the registered database name (e.g. "tasks").
// Takes resolver (func(string) string) which maps a SQL query string to a human-readable
// operation name. May be nil, in which case operations are reported as "UNKNOWN".
//
// Returns *otelDBTX which implements DBTX with instrumentation.
func newOTelDBTX(
	inner DBTX,
	databaseSystem string,
	databaseNamespace string,
	resolver func(string) string,
	observer monitoring_domain.QueryObserver,
) *otelDBTX {
	return &otelDBTX{
		inner:             inner,
		databaseSystem:    databaseSystem,
		databaseNamespace: databaseNamespace,
		resolver:          resolver,
		observer:          observer,
		tracer:            otel.Tracer("piko/db/" + databaseNamespace),
		reportOnce:        new(sync.Once),
		operationLabels:   new(operationLabelSet),
	}
}

// ExecContext executes a query without returning rows, wrapped with a span and metric
// recording.
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

	rows := int64(0)
	if result != nil {
		if n, rerr := result.RowsAffected(); rerr == nil {
			rows = n
		}
	}
	o.observe(ctx, operation, query, start, rows, err)

	return result, err
}

// QueryContext executes a query that returns rows, wrapped with a span and metric
// recording.
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

	o.observe(ctx, operation, query, start, 0, err)

	return rows, err
}

// QueryRowContext executes a query that returns at most one row, wrapped with a span and
// duration metric. Errors are deferred to row.Scan and cannot be captured at the DBTX
// level.
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

	o.observe(ctx, operation, query, start, 0, nil)

	return row
}

// WrapDBTX returns an instrumented view of a different connection, carrying this
// wrapper's tracer, observer, resolver and identifying attributes onto it.
//
// Takes inner (any) which is the connection to instrument; anything that is not a DBTX is
// returned unchanged rather than being replaced with something unusable.
//
// Returns any which is the instrumented connection, or inner when it cannot be wrapped.
func (o *otelDBTX) WrapDBTX(inner any) any {
	target, ok := inner.(DBTX)
	if !ok {

		o.reportOnce.Do(func() {
			_, l := logger_domain.From(context.Background(), log)
			l.Warn("database instrumentation not applied: connection is not a DBTX; "+
				"statements run inside a transaction will not be traced or measured",
				logger_domain.String("db.namespace", o.databaseNamespace),
				logger_domain.String("type", fmt.Sprintf("%T", inner)),
			)
		})
		return inner
	}

	wrapped := *o
	wrapped.inner = target
	return &wrapped
}

// observe forwards one completed statement to the registered QueryObserver (a no-op when
// none is set). It runs on the hottest path, so a panicking observer is recovered here
// rather than allowed to crash the DB call it instruments.
//
// Takes ctx (context.Context) for cancellation and panic-recovery context.
// Takes operation (string) which is the resolved operation name.
// Takes statement (string) which is the SQL text.
// Takes start (time.Time) which is when the call began, for the duration.
// Takes rows (int64) which is the affected or returned row count when known.
// Takes err (error) which is the call's error, or nil on success.
func (o *otelDBTX) observe(ctx context.Context, operation, statement string, start time.Time, rows int64, err error) {
	if o.observer == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_ = goroutine.HandlePanicRecovery(ctx, "bootstrap.dbtx_otel.observe", r)
		}
	}()
	o.observer.ObserveQuery(ctx, &monitoring_domain.QueryObservation{
		Connection: o.databaseNamespace,
		Operation:  operation,
		Statement:  statement,
		System:     o.databaseSystem,
		DurationMs: time.Since(start).Milliseconds(),
		Rows:       rows,
		Err:        err,
	})
}

// resolveOperation maps a SQL query string to a human-readable operation name using the
// configured resolver.
//
// Takes query (string) which is the SQL query to resolve.
//
// Returns string which is the operation name, or "UNKNOWN" when the resolver is nil or
// returns an empty string.
func (o *otelDBTX) resolveOperation(query string) string {
	if o.resolver != nil {
		if name := o.resolver(query); name != "" {
			return name
		}
	}

	return o.operationLabels.labelFor(query)
}

// operationVerb reduces a derived label to its verb, the part that is bounded by
// definition.
//
// Takes label (string) which is a derived "VERB table" label.
//
// Returns string which is the verb alone.
func operationVerb(label string) string {
	if verb, _, found := strings.Cut(label, " "); found {
		return verb
	}

	return label
}

// deriveOperationName produces a low-cardinality operation label from a SQL statement
// when no QueryNameResolver is configured.
//
// Takes query (string) which is the SQL statement.
//
// Returns string which is a label like "SELECT artefacts", or "UNKNOWN" when the
// statement does not parse into one.
func deriveOperationName(query string) string {
	fields := leadingFields(stripLeadingComment(query), maxOperationTokens)
	if len(fields) == 0 {
		return "UNKNOWN"
	}

	verb := strings.ToUpper(fields[0])
	keyword, ok := operationTableKeyword[verb]
	if !ok {
		return "UNKNOWN"
	}

	if keyword == verb {
		if len(fields) > 1 {
			if table := trimSQLIdentifier(fields[1]); table != "" {
				return verb + " " + table
			}
		}
		return verb
	}

	for index := 1; index < len(fields)-1; index++ {
		if !strings.EqualFold(fields[index], keyword) {
			continue
		}

		candidate := fields[index+1]

		if strings.HasPrefix(candidate, "(") {
			continue
		}

		if isSQLKeyword(trimSQLIdentifier(candidate)) && index+2 < len(fields) {
			candidate = fields[index+2]
		}

		if table := trimSQLIdentifier(candidate); table != "" && !isSQLKeyword(table) {
			return verb + " " + table
		}
	}

	return verb
}

// stripLeadingComment removes a block comment from the front of a statement.
//
// Takes query (string) which is the SQL statement.
//
// Returns string which is the statement without a leading block comment.
func stripLeadingComment(query string) string {
	trimmed := strings.TrimLeft(query, " \t\r\n")

	for range maxLeadingComments {
		scanLimit := min(len(trimmed), maxCommentScan)

		switch {

		case strings.HasPrefix(trimmed, "--"):
			end := strings.IndexByte(trimmed[:scanLimit], '\n')
			if end < 0 {
				return query
			}
			trimmed = strings.TrimLeft(trimmed[end+1:], " \t\r\n")

		case strings.HasPrefix(trimmed, "/*"):
			end := strings.Index(trimmed[:scanLimit], "*/")
			if end < 0 {
				return query
			}
			trimmed = strings.TrimLeft(trimmed[end+len("*/"):], " \t\r\n")

		default:
			return trimmed
		}
	}

	return trimmed
}

// leadingFields splits the first limit whitespace-separated tokens out of a statement.
//
// Takes query (string) which is the SQL statement.
// Takes limit (int) which is the most tokens to return.
//
// Returns []string which holds at most limit tokens, in order.
func leadingFields(query string, limit int) []string {
	fields := make([]string, 0, limit)
	for token := range strings.FieldsSeq(query) {
		fields = append(fields, token)
		if len(fields) == limit {
			break
		}
	}
	return fields
}

// isSQLKeyword reports whether a token is a word the label must never mistake for a
// table.
//
// Takes token (string) which is a candidate table name.
//
// Returns bool which is true when the token is a keyword rather than an identifier.
func isSQLKeyword(token string) bool {
	_, keyword := nonTableKeywords[strings.ToUpper(token)]
	return keyword
}

// trimSQLIdentifier strips quoting, schema qualification and trailing punctuation from a
// table reference so the label stays stable across equivalent spellings.
//
// Takes token (string) which is the raw token following the table keyword.
//
// Returns string which is the bare identifier, empty when the token is not one.
func trimSQLIdentifier(token string) string {
	token = strings.Trim(token, "`\"'[]();,")
	if index := strings.LastIndex(token, "."); index >= 0 {
		token = token[index+1:]
	}
	if strings.ContainsFunc(token, func(character rune) bool { return !isSQLIdentifierRune(character) }) {
		return ""
	}
	if len(token) > maxOperationIdentifierLen {
		return ""
	}

	return token
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
// Takes start (time.Time) which is the operation start time for duration calculation.
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

// isSQLIdentifierRune reports whether a rune may appear in a bare SQL identifier.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for the characters an unquoted identifier may contain.
func isSQLIdentifierRune(character rune) bool {
	return character == '_' ||
		(character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		(character >= '0' && character <= '9')
}
