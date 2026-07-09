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

package query_collector_grpcfb

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// maxStatementLen bounds the redacted SQL text per record so one pathological statement
	// cannot bloat a telemetry frame.
	maxStatementLen = 4096

	// maxErrorLen bounds the redacted error string per record so one pathological error
	// cannot bloat a telemetry frame.
	maxErrorLen = 512

	// rawInputCap bounds the raw statement/error text fed to the redaction regexps before
	// the final cap is applied.
	rawInputCap = 16 * maxStatementLen
)

var (
	// sqlDollarStringLit matches Postgres dollar-quoted string literals ($tag$...$tag$,
	// including the empty-tag $$...$$ form).
	sqlDollarStringLit = regexp.MustCompile(`\$[A-Za-z0-9_]*\$.*?\$[A-Za-z0-9_]*\$`)

	// sqlEStringLit matches Postgres escape-string literals (E'...'), whose bodies may
	// contain backslash escapes (\') as well as doubled single-quote escapes (two single
	// quotes).
	sqlEStringLit = regexp.MustCompile(`[eE]'(?:[^'\\]|''|\\.)*'`)
	// sqlStringLit matches single-quoted SQL string literals, handling doubled single-quote
	// escapes. Redacting these to '?' keeps PII in non-parameterized SQL or DB error strings
	// from reaching the remote sink, and tightens grouping.
	sqlStringLit = regexp.MustCompile(`'(?:[^']|'')*'`)

	// sqlHexLit matches hexadecimal integer literals (0x1A). Redacting these keeps PII
	// encoded in hex off-box; it runs before sqlNumLit so the leading 0 is not consumed.
	sqlHexLit = regexp.MustCompile(`\b0[xX][0-9A-Fa-f]+\b`)

	// sqlNumLit matches standalone numeric literals, including an optional leading sign, a
	// decimal part, and a scientific exponent.
	sqlNumLit = regexp.MustCompile(`(^|[^\w.])[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?\b`)
)

// redactSQL strips string + numeric literals from a SQL statement or DB error string.
//
// Redaction covers the ANSI single-quoted string form (with doubled-quote escapes), the
// Postgres dollar-quoted ($tag$...$tag$) and escape-string (E'...') forms, and numeric
// literals in decimal, signed, scientific, and hexadecimal (0x...) notation. Other
// dialect-specific forms (such as MySQL backtick identifiers or double-quoted string
// literals) are not redacted, so callers feeding those dialects should parameterise
// queries.
//
// Takes s (string) which is the SQL statement or error text to redact.
//
// Returns string which is s with its string and numeric literals collapsed to '?'.
func redactSQL(s string) string {
	s = sqlDollarStringLit.ReplaceAllString(s, "'?'")
	s = sqlEStringLit.ReplaceAllString(s, "'?'")
	s = sqlStringLit.ReplaceAllString(s, "'?'")
	s = sqlHexLit.ReplaceAllString(s, "?")
	s = sqlNumLit.ReplaceAllString(s, "${1}?")
	return s
}

// Collector implements piko.QueryObserver by translating each observed DB statement into
// a telemetry_grpcfb.QueryStat and enqueuing it on a (typically shared) telemetry client.
type Collector struct {
	// client is the telemetry transport that observed statements are enqueued onto.
	client *telemetry_grpcfb.Client
}

var (
	_ piko.QueryObserver = (*Collector)(nil)
)

// New wraps an existing, shared telemetry client. The caller owns the client's lifecycle;
// forwarding is non-blocking and lossy by design.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
//
// Returns *Collector which forwards observed statements over the shared client.
func New(client *telemetry_grpcfb.Client) *Collector {
	return &Collector{client: client}
}

// ObserveQuery enqueues one observed statement execution (non-blocking). The observation
// is not retained beyond this call.
//
// Takes obs (*piko.QueryObservation) which is the observed statement execution.
func (c *Collector) ObserveQuery(ctx context.Context, obs *piko.QueryObservation) {
	if c == nil || c.client == nil || obs == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			_, l := logger_domain.From(ctx, log)
			l.Error("recovered panic while observing query", logger_domain.String("panic", fmt.Sprint(r)))
		}
	}()

	var attrs []telemetry_grpcfb.KV

	rawStatement, statementTruncated := telemetry_grpcfb.TruncateUTF8(obs.Statement, rawInputCap)
	statement, statementCapped := telemetry_grpcfb.TruncateUTF8(redactSQL(rawStatement), maxStatementLen)
	if statementTruncated || statementCapped {
		attrs = append(attrs, telemetry_grpcfb.KV{Key: "truncated", Value: "statement"})
	}

	status, errorMessage := "ok", ""
	if obs.Err != nil {
		status = "error"
		rawError, errorTruncated := telemetry_grpcfb.TruncateUTF8(obs.Err.Error(), rawInputCap)
		var errorCapped bool
		errorMessage, errorCapped = telemetry_grpcfb.TruncateUTF8(redactSQL(rawError), maxErrorLen)
		if errorTruncated || errorCapped {
			attrs = append(attrs, telemetry_grpcfb.KV{Key: "truncated", Value: "error"})
		}
	}

	c.client.AddQueryStat(ctx, telemetry_grpcfb.QueryStat{
		Connection: obs.Connection,
		Statement:  statement,
		Operation:  obs.Operation,
		Status:     status,
		Error:      errorMessage,
		Attrs:      attrs,
		DurationMs: obs.DurationMs,
		Rows:       obs.Rows,
		Calls:      1,
		TsMs:       time.Now().UnixMilli(),
	})
}
