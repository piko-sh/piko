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

package db_engine_timescaledb

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

var (
	// timescaleReloptionMap maps a lower-cased `timescaledb.*` reloption key to its
	// TIMESCALE_* EngineSpecific key.
	//
	// Centralising the mapping as a package-level table keeps the registration declarative;
	// adding a new reloption is a single map entry rather than another case arm. Keys are
	// stored lower-cased so the lookup helper can compare without allocating a normalised
	// key on every call.
	//
	// The map covers both legacy compression keys (`timescaledb.compress`,
	// `timescaledb.compress_segmentby`, and similar) and the TS 2.18+ canonical columnstore
	// keys (`timescaledb.enable_columnstore`, `timescaledb.segmentby`,
	// `timescaledb.orderby`) so reloption bodies produced by either generation of the
	// toolkit lift into the same EngineSpecific surface. Downstream consumers should treat
	// the columnstore and compression keys as semantically equivalent; the legacy keys are
	// preserved verbatim where they appeared in the SQL.
	timescaleReloptionMap = map[string]string{
		"timescaledb.continuous":                   "TIMESCALE_CONTINUOUS_FLAG",
		"timescaledb.materialized_only":            "TIMESCALE_MATERIALIZED_ONLY",
		"timescaledb.create_group_indexes":         "TIMESCALE_CREATE_GROUP_INDEXES",
		"timescaledb.finalized":                    "TIMESCALE_FINALIZED",
		"timescaledb.compress":                     "TIMESCALE_COMPRESSION_ENABLED",
		"timescaledb.compress_segmentby":           "TIMESCALE_COMPRESSION_SEGMENTBY",
		"timescaledb.compress_orderby":             "TIMESCALE_COMPRESSION_ORDERBY",
		"timescaledb.compress_chunk_time_interval": "TIMESCALE_COMPRESSION_CHUNK_TIME_INTERVAL",
		"timescaledb.enable_columnstore":           "TIMESCALE_COLUMNSTORE_ENABLED",
		"timescaledb.segmentby":                    "TIMESCALE_SEGMENTBY",
		"timescaledb.orderby":                      "TIMESCALE_ORDERBY",
	}
)

// parseCreateContinuousAggregate parses a CREATE MATERIALIZED VIEW continuous-aggregate
// statement into a catalogue mutation.
//
// The recognised shape is CREATE MATERIALIZED VIEW [IF NOT EXISTS] [schema.]name WITH
// (timescaledb.continuous = true [, timescaledb.materialized_only = ...]) AS SELECT ...
// optionally followed by WITH [NO] DATA. The reloption body is parsed structurally so the
// timescaledb keys flow into EngineSpecific. The SELECT body is captured as opaque text;
// downstream consumers can re-analyse if they need typed columns.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser context positioned at
// the CREATE keyword.
//
// Returns *querier_dto.CatalogueMutation which is a MutationCreateView carrying the
// continuous-aggregate marker in EngineSpecific.
// Returns error when the statement cannot be parsed.
func parseCreateContinuousAggregate(p db_engine_postgres.ParserContext) (*querier_dto.CatalogueMutation, error) {
	p.MustKeyword("CREATE")
	if p.MatchKeyword("OR") {
		p.MustKeyword("REPLACE")
	}
	p.MustKeyword("MATERIALIZED")
	p.MustKeyword("VIEW")

	ifNotExists := p.MatchIfNotExists()

	schema, name, err := p.ParseQualifiedName()
	if err != nil {
		return nil, fmt.Errorf("continuous aggregate name: %w", err)
	}

	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateView,
		SchemaName: schema,
		TableName:  name,
		EngineSpecific: map[string]string{
			"TIMESCALE_CONTINUOUS_AGGREGATE": literalTrue,
		},
	}
	if ifNotExists {
		mutation.EngineSpecific["TIMESCALE_IF_NOT_EXISTS"] = literalTrue
	}

	if applyErr := applyContinuousAggregateReloptions(p, mutation, name); applyErr != nil {
		return nil, applyErr
	}

	if !p.MatchKeyword("AS") {
		return nil, fmt.Errorf("continuous aggregate %q: missing AS clause at position %d", name, p.CurrentToken().Position())
	}

	mutation.ViewDefinition = p.AnalyseViewBody(nil)

	body, bodyErr := captureViewBody(p)
	if bodyErr != nil {
		return nil, fmt.Errorf("continuous aggregate %q: %w", name, bodyErr)
	}
	if body != "" {
		mutation.EngineSpecific["TIMESCALE_VIEW_BODY"] = body
	}

	mutation.EngineSpecific["TIMESCALE_WITH_DATA"] = parseTrailingDataModifier(p)
	return mutation, nil
}

// applyContinuousAggregateReloptions consumes the mandatory WITH body of a continuous
// aggregate.
//
// It maps known timescaledb keys onto the TIMESCALE_* EngineSpecific entries and
// preserves unknown keys verbatim under TIMESCALE_RELOPTION_<KEY>. The fallback mirrors
// the compression-alter parser so downstream consumers can still inspect keys this parser
// does not recognise.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser context positioned just
// after the view name.
// Takes mutation (*querier_dto.CatalogueMutation) which has the outgoing EngineSpecific
// map populated.
// Takes name (string) which is the view name; used in error wrapping so callers see which
// statement failed.
//
// Returns error when the WITH clause is missing or the reloption body cannot be parsed.
func applyContinuousAggregateReloptions(p db_engine_postgres.ParserContext, mutation *querier_dto.CatalogueMutation, name string) error {
	if !p.MatchKeyword("WITH") {
		return fmt.Errorf("continuous aggregate %q: missing WITH clause at position %d", name, p.CurrentToken().Position())
	}
	reloptions, reloptionErr := p.ParseReloptionList()
	if reloptionErr != nil {
		return fmt.Errorf("continuous aggregate %q: WITH body: %w", name, reloptionErr)
	}
	for key, value := range reloptions {
		if matched := timescaleReloptionToEngineSpecific(key); matched != "" {
			mutation.EngineSpecific[matched] = value
			continue
		}
		mutation.EngineSpecific["TIMESCALE_RELOPTION_"+strings.ToUpper(key)] = value
	}
	return nil
}

// parseTrailingDataModifier consumes the optional `WITH [NO] DATA` modifier and returns
// the canonical boolean string the caller should store in TIMESCALE_WITH_DATA. The
// default when the modifier is absent is `true` because that matches TimescaleDB's
// documented behaviour for a freshly created continuous aggregate.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser context positioned at
// the optional modifier.
//
// Returns string which is literalTrue or literalFalse.
func parseTrailingDataModifier(p db_engine_postgres.ParserContext) string {
	if !p.MatchKeyword("WITH") {
		return literalTrue
	}
	if p.MatchKeyword("NO") {
		p.MatchKeyword("DATA")
		return literalFalse
	}
	p.MatchKeyword("DATA")
	return literalTrue
}

// timescaleReloptionToEngineSpecific maps a `timescaledb.*` reloption key to its
// TIMESCALE_* EngineSpecific key.
//
// Takes key (string) which is the raw reloption key from the WITH body.
//
// Returns string which is the matching TIMESCALE_* key, or "" when the key is unknown and
// is dropped by the caller's fallback path.
func timescaleReloptionToEngineSpecific(key string) string {
	return timescaleReloptionMap[strings.ToLower(key)]
}

// captureViewBody collects the AS SELECT body up to an optional trailing `WITH [NO] DATA`
// modifier. Returns the body as raw text.
//
// The body may itself begin with a CTE (`AS WITH cte AS (...) SELECT ...`), so a bare
// top-level `WITH` is only treated as the terminator when it is followed by `NO DATA` or
// `DATA <EOF>`. String literals are re-wrapped with single quotes so the captured text
// round-trips as valid SQL. The scan bails with errParenDepthExceeded when nested parens
// exceed maxParenDepth so adversarial inputs do not provoke unbounded work. An
// unterminated body (open parens still pending at EOF) is reported as an error so callers
// see a clear failure rather than a silently truncated capture.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the first token of
// the view body (just after `AS`).
//
// Returns the trimmed body text.
// Returns error on depth overflow or unterminated body.
func captureViewBody(p db_engine_postgres.ParserContext) (string, error) {
	var builder strings.Builder
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		if isViewBodyTerminator(p, &builder, depth) {
			return strings.TrimSpace(builder.String()), nil
		}
		newDepth, err := updateViewBodyParenDepth(tok, depth)
		if err != nil {
			return "", err
		}
		depth = newDepth
		appendViewBodyToken(&builder, tok)
		p.Advance()
	}
	if depth != 0 {
		return "", fmt.Errorf("continuous aggregate view body: unterminated parenthesis (depth %d) at end of statement", depth)
	}
	return strings.TrimSpace(builder.String()), nil
}

// isViewBodyTerminator reports whether the cursor sits on the trailing `WITH [NO] DATA`
// modifier that ends the captured view body.
//
// The modifier appears after the SELECT body, so it only terminates capture once some
// body text has been collected; a leading `WITH data AS (...)` or `WITH no AS (...)` CTE
// (builder still empty) must not be mistaken for it, otherwise the whole view body is
// dropped.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at tok.
// Takes builder (*strings.Builder) holding the body captured so far.
// Takes depth (int) which is the current paren nesting depth.
//
// Returns bool which is true when the cursor sits on the trailing data modifier.
func isViewBodyTerminator(p db_engine_postgres.ParserContext, builder *strings.Builder, depth int) bool {
	tok := p.CurrentToken()
	return depth == 0 && builder.Len() > 0 && tok.Kind() == db_engine_postgres.TokenIdentifier &&
		strings.EqualFold(tok.Value(), "WITH") && isTrailingDataModifier(p)
}

// updateViewBodyParenDepth adjusts the paren depth for tok and returns the new depth.
//
// A close paren floors at zero so an unbalanced extra close-paren does not drive depth
// negative and let a later depth-0 check misfire.
//
// Takes tok (db_engine_postgres.Token) which is the current token.
// Takes depth (int) which is the current paren nesting depth.
//
// Returns int which is the new paren nesting depth.
// Returns error when nesting overflows, wrapping errParenDepthExceeded with the position.
func updateViewBodyParenDepth(tok db_engine_postgres.Token, depth int) (int, error) {
	switch tok.Kind() {
	case db_engine_postgres.TokenLeftParen:
		if depth >= maxParenDepth {
			return 0, fmt.Errorf("continuous aggregate view body at position %d: %w", tok.Position(), errParenDepthExceeded)
		}
		return depth + 1, nil
	case db_engine_postgres.TokenRightParen:
		if depth > 0 {
			return depth - 1, nil
		}
		return depth, nil
	default:
		return depth, nil
	}
}

// appendViewBodyToken writes tok onto the captured body, prefixing a space separator when
// text has already been collected. Each token is re-wrapped through appendCapturedToken
// so escape-strings, bit-strings, dollar-quoted bodies and quoted identifiers keep their
// delimiters and the captured text round-trips as valid SQL.
//
// Takes builder (*strings.Builder) which accumulates the body text.
// Takes tok (db_engine_postgres.Token) which is the current token.
func appendViewBodyToken(builder *strings.Builder, tok db_engine_postgres.Token) {
	if builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	appendCapturedToken(builder, tok)
}

// isTrailingDataModifier reports whether the cursor sits on a `WITH` token that
// introduces the `WITH NO DATA` or `WITH DATA` trailing clause rather than a CTE inside
// the view body.
//
// The peek looks one token ahead: NO or DATA identifies the trailer.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the `WITH` token.
//
// Returns bool which is true when the `WITH` token introduces the trailing data modifier.
func isTrailingDataModifier(p db_engine_postgres.ParserContext) bool {
	next := p.Peek()
	if next.Kind() != db_engine_postgres.TokenIdentifier {
		return false
	}
	upper := strings.ToUpper(next.Value())
	return upper == "NO" || upper == "DATA"
}
