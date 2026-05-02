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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

var (
	// hypertableReloptionLifts maps a lower-cased `tsdb.*` or `timescaledb.*` reloption key
	// to the TIMESCALE_* EngineSpecific key it should populate.
	//
	// The key is found in a plain CREATE TABLE WITH body. The map is the modern (TS 2.18+)
	// counterpart to timescaleReloptionMap; the keys differ because the modern form sits on
	// a regular CREATE TABLE while the legacy compression keys appear on ALTER TABLE /
	// CREATE MATERIALIZED VIEW. The boolean `tsdb.hypertable` / `timescaledb.hypertable`
	// toggle key is omitted here because it is the discriminator that activates the lift;
	// the hook surfaces it as TIMESCALE_HYPERTABLE separately.
	hypertableReloptionLifts = map[string]string{
		"tsdb.partition_column":          "TIMESCALE_TIME_COLUMN",
		"tsdb.chunk_interval":            "TIMESCALE_CHUNK_TIME_INTERVAL",
		"tsdb.segmentby":                 "TIMESCALE_SEGMENTBY",
		"tsdb.orderby":                   "TIMESCALE_ORDERBY",
		"tsdb.enable_columnstore":        "TIMESCALE_COLUMNSTORE_ENABLED",
		"timescaledb.partition_column":   "TIMESCALE_TIME_COLUMN",
		"timescaledb.chunk_interval":     "TIMESCALE_CHUNK_TIME_INTERVAL",
		"timescaledb.segmentby":          "TIMESCALE_SEGMENTBY",
		"timescaledb.orderby":            "TIMESCALE_ORDERBY",
		"timescaledb.enable_columnstore": "TIMESCALE_COLUMNSTORE_ENABLED",
	}

	// hypertableActivatorKeys lists the lower-cased reloption keys whose presence in a
	// CREATE TABLE WITH body marks the table as a hypertable. Either of the two namespaces
	// is accepted because the timescaledb extension registers both as synonyms.
	hypertableActivatorKeys = map[string]struct{}{
		"tsdb.hypertable":        {},
		"timescaledb.hypertable": {},
	}
)

// createTableWithBody locates the position of a trailing `WITH (...)` body on a CREATE
// TABLE statement. The struct decouples the search from the lift logic so the hook can
// short-circuit before touching the mutation when no body is present.
type createTableWithBody struct {
	// openParenIndex is the token index of the body's opening `(`.
	openParenIndex int

	// closeParenIndex is the token index of the body's matching `)`.
	closeParenIndex int
}

// timescaleCreateTableHypertableHook lifts TS 2.18+ hypertable reloptions from a plain
// `CREATE TABLE x (...) WITH (tsdb.hypertable, ...)` statement into structured
// EngineSpecific keys on the mutation. The postgres CREATE TABLE handler treats the
// trailing WITH body as opaque, so without this hook the statement would silently lose
// its hypertable semantics.
//
// The hook only fires for a successful CREATE TABLE mutation; other kinds (extension
// statements, DML, DROP, etc.) short-circuit at the top. When the WITH body does not
// include a hypertable activator (`tsdb.hypertable` or `timescaledb.hypertable`) the
// mutation is left untouched, so a plain `CREATE TABLE x (...) WITH (fillfactor=80)`
// passes through unmodified.
//
// Takes p (db_engine_postgres.ParserContext) which exposes the statement's token slice
// via Tokens(). The hook does not mutate parser state.
// Takes _ (db_engine_postgres.StatementKind) which classifies the statement; the
// parameter is accepted via blank identifier because the mutation kind is the
// load-bearing check below, but the PostParseHook contract still requires the slot.
// Takes mutation (*querier_dto.CatalogueMutation) which receives the lifted
// EngineSpecific entries when the body activates the hypertable.
//
// Returns error only when the underlying scan signals a structural problem; the current
// implementation never returns an error, but the signature is kept aligned with the
// PostParseHook contract so extensions can later add diagnostics without changing the
// wiring.
func timescaleCreateTableHypertableHook(
	p db_engine_postgres.ParserContext,
	_ db_engine_postgres.StatementKind,
	mutation *querier_dto.CatalogueMutation,
) error {
	if mutation == nil || mutation.Kind != querier_dto.MutationCreateTable {
		return nil
	}
	tokens := p.Tokens()
	body, ok := findCreateTableWithBody(tokens)
	if !ok {
		return nil
	}
	reloptions := extractCreateTableReloptions(tokens, body.openParenIndex, body.closeParenIndex)
	if !containsHypertableActivator(reloptions) {
		return nil
	}
	applyHypertableReloptions(mutation, reloptions)
	return nil
}

// findCreateTableWithBody scans the statement tokens for the trailing `WITH (` body of a
// CREATE TABLE.
//
// The scan walks the token stream at paren depth zero and remembers the LAST `WITH (`
// pair so the body always corresponds to the reloption tail rather than any earlier
// `WITH` keyword that may appear inside a column expression, a typed `CREATE TABLE foo OF
// type_name (...)` body, or an `AS SELECT` body. Tracking the last top-level pair avoids
// the trap of stepping past the first `(` after CREATE TABLE: that approach
// mis-identified the body for typed-table and partition-of forms whose surface includes
// auxiliary parenthesised constructs before the reloption tail.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
//
// Returns createTableWithBody which is the body's open and close paren indices when one
// is found.
// Returns bool which is true on success.
func findCreateTableWithBody(tokens []db_engine_postgres.Token) (createTableWithBody, bool) {
	var (
		lastBody  createTableWithBody
		lastFound bool
	)
	depth := 0
	for index := 0; index < len(tokens); index++ {
		switch tokens[index].Kind() {
		case db_engine_postgres.TokenLeftParen:
			depth++
			continue
		case db_engine_postgres.TokenRightParen:
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth != 0 {
			continue
		}
		if !strings.EqualFold(tokens[index].Value(), "WITH") {
			continue
		}
		if index+1 >= len(tokens) || tokens[index+1].Kind() != db_engine_postgres.TokenLeftParen {
			continue
		}
		closeIndex, ok := matchClosingParen(tokens, index+1)
		if !ok {
			return createTableWithBody{}, false
		}
		lastBody = createTableWithBody{openParenIndex: index + 1, closeParenIndex: closeIndex}
		lastFound = true

		index = closeIndex
	}
	return lastBody, lastFound
}

// matchClosingParen returns the index of the close paren that pairs with the open paren
// at openIndex.
//
// The scan honours nested parens and bails out with a false return when the nesting
// exceeds maxParenDepth, mirroring the bound enforced by the sibling capture helpers so
// adversarial inputs are bounded uniformly. A false return also signals an unmatched open
// paren, typically a malformed statement that the built-in parser already rejected, where
// the hook declines to act.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes openIndex (int) which is the index of the `(` token.
//
// Returns int which is the matching `)` index.
// Returns bool which is true when a match was found.
func matchClosingParen(tokens []db_engine_postgres.Token, openIndex int) (int, bool) {
	depth := 0
	for index := openIndex; index < len(tokens); index++ {
		switch tokens[index].Kind() {
		case db_engine_postgres.TokenLeftParen:
			if depth >= maxParenDepth {
				return 0, false
			}
			depth++
		case db_engine_postgres.TokenRightParen:
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

// extractCreateTableReloptions parses the reloption body between the supplied paren
// indices into a lower-cased key/value map.
//
// The body shape is `key1 [= value1], key2 [= value2], ...` where each key may be
// schema-qualified (`tsdb.hypertable`). A flag-style key with no `=` (the bare
// `tsdb.hypertable` discriminator) maps to the literal "true" so callers can treat it
// uniformly with explicit booleans.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes openParenIndex (int) which is the index of the `(` token.
// Takes closeParenIndex (int) which is the index of the matching `)`.
//
// Returns map[string]string keyed on the lower-cased option name.
func extractCreateTableReloptions(
	tokens []db_engine_postgres.Token,
	openParenIndex int,
	closeParenIndex int,
) map[string]string {
	reloptions := map[string]string{}
	index := openParenIndex + 1
	for index < closeParenIndex {
		key, nextIndex := readReloptionKey(tokens, index, closeParenIndex)
		if key == "" {
			return reloptions
		}
		value, advanced := readReloptionValue(tokens, nextIndex, closeParenIndex)
		reloptions[strings.ToLower(key)] = value
		index = skipReloptionSeparator(tokens, advanced, closeParenIndex)
	}
	return reloptions
}

// readReloptionKey reads a dotted identifier key starting at index and returns the
// assembled key plus the index of the next token to inspect (after any optional `=`). An
// empty return signals that the cursor did not point at a valid key, which terminates the
// parse.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes index (int) which is the cursor position.
// Takes limit (int) which is the close-paren index that bounds the scan.
//
// Returns string which is the captured key text.
// Returns int which is the next cursor position.
func readReloptionKey(tokens []db_engine_postgres.Token, index int, limit int) (string, int) {
	if index >= limit || tokens[index].Kind() != db_engine_postgres.TokenIdentifier {
		return "", index
	}
	var builder strings.Builder
	builder.WriteString(tokens[index].Value())
	cursor := index + 1
	for cursor < limit && tokens[cursor].Kind() == db_engine_postgres.TokenDot {
		builder.WriteByte('.')
		cursor++
		if cursor >= limit || tokens[cursor].Kind() != db_engine_postgres.TokenIdentifier {
			return "", cursor
		}
		builder.WriteString(tokens[cursor].Value())
		cursor++
	}
	return builder.String(), cursor
}

// readReloptionValue reads the value half of a reloption pair.
//
// When the cursor sits on `=` the value follows; otherwise the key was a flag and the
// value is the literal "true". String values have their surrounding quotes stripped
// because the tokeniser already removes them, leaving the captured text directly usable
// as the EngineSpecific value.
//
// The scan tracks parenthesis depth so a value containing a nested expression (for
// example `tsdb.foo = my_helper(a, b)`) is not truncated at the first inner comma. Only
// commas observed at depth zero terminate the value; commas inside nested parens are
// emitted verbatim. Nested parens beyond maxParenDepth fall back to the flag-style "true"
// capture so adversarial inputs do not provoke unbounded work; the bound mirrors the
// other capture helpers in db_engine_timescaledb.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes index (int) which is the cursor position.
// Takes limit (int) which is the close-paren index that bounds the scan.
//
// Returns string which is the captured value.
// Returns int which is the cursor position after the value.
func readReloptionValue(tokens []db_engine_postgres.Token, index int, limit int) (string, int) {
	if index >= limit {
		return literalTrue, index
	}
	if tokens[index].Kind() != db_engine_postgres.TokenOperator || tokens[index].Value() != "=" {
		return literalTrue, index
	}
	cursor := index + 1
	if cursor >= limit {
		return "", cursor
	}
	return scanReloptionValueBody(tokens, cursor, limit)
}

// scanReloptionValueBody walks the value tokens from start up to limit, stopping at the
// first top-level comma.
//
// It returns the captured value text and the cursor position one past the last consumed
// token. When nested parens exceed maxParenDepth the scan bails with literalTrue so
// adversarial inputs do not provoke unbounded work; the cursor at bail-out points at the
// offending `(`.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes start (int) which is the cursor position at the first value token (one past the
// `=`).
// Takes limit (int) which is the close-paren index bounding the scan.
//
// Returns string which is the captured value (trimmed).
// Returns int which is the cursor position after the value.
func scanReloptionValueBody(tokens []db_engine_postgres.Token, start int, limit int) (string, int) {
	var builder strings.Builder
	depth := 0
	cursor := start
	for cursor < limit {
		tok := tokens[cursor]
		if depth == 0 && tok.Kind() == db_engine_postgres.TokenComma {
			break
		}
		nextDepth, overflow := advanceReloptionDepth(tok, depth)
		if overflow {
			return literalTrue, cursor
		}
		depth = nextDepth
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		appendCapturedToken(&builder, tok)
		cursor++
	}
	return strings.TrimSpace(builder.String()), cursor
}

// advanceReloptionDepth returns the updated paren-depth after observing tok, plus an
// overflow flag set when a `(` would push depth above maxParenDepth.
//
// Tokens other than parens leave depth unchanged. The helper keeps the value-scan loop
// free of nested switch / if logic so the host stays under the cognitive-complexity
// bound.
//
// Takes tok (db_engine_postgres.Token) which is the current token.
// Takes depth (int) which is the running paren depth.
//
// Returns int which is the depth after tok is consumed.
// Returns bool which is true when the depth would overflow.
func advanceReloptionDepth(tok db_engine_postgres.Token, depth int) (int, bool) {
	switch tok.Kind() {
	case db_engine_postgres.TokenLeftParen:
		if depth >= maxParenDepth {
			return depth, true
		}
		return depth + 1, false
	case db_engine_postgres.TokenRightParen:
		if depth > 0 {
			return depth - 1, false
		}
	}
	return depth, false
}

// skipReloptionSeparator advances the cursor past a comma when one is present so the
// following key is reached.
//
// Takes tokens ([]db_engine_postgres.Token) which is the statement's token slice.
// Takes index (int) which is the cursor position.
// Takes limit (int) which is the close-paren index that bounds the scan.
//
// Returns int which is the next cursor position.
func skipReloptionSeparator(tokens []db_engine_postgres.Token, index int, limit int) int {
	if index < limit && tokens[index].Kind() == db_engine_postgres.TokenComma {
		return index + 1
	}
	return index
}

// containsHypertableActivator reports whether the parsed reloptions include a
// `tsdb.hypertable` or `timescaledb.hypertable` key. The value is ignored; the
// activator's presence alone toggles the lift.
//
// Takes reloptions (map[string]string) which is the parsed body.
//
// Returns bool which is true when the activator is present.
func containsHypertableActivator(reloptions map[string]string) bool {
	for key := range reloptions {
		if _, ok := hypertableActivatorKeys[key]; ok {
			return true
		}
	}
	return false
}

// applyHypertableReloptions copies recognised TS 2.18+ reloption keys onto the mutation's
// EngineSpecific map.
//
// The TIMESCALE_HYPERTABLE marker is always set when the activator was present;
// recognised keys map to their structured EngineSpecific names via
// hypertableReloptionLifts. Unrecognised entries are preserved verbatim under
// TIMESCALE_RELOPTION_<UPPER_KEY> so nothing is lost silently.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the lifted entries.
// Takes reloptions (map[string]string) which is the parsed body.
func applyHypertableReloptions(mutation *querier_dto.CatalogueMutation, reloptions map[string]string) {
	if mutation.EngineSpecific == nil {
		mutation.EngineSpecific = map[string]string{}
	}
	mutation.EngineSpecific["TIMESCALE_HYPERTABLE"] = literalTrue
	for key, value := range reloptions {
		if _, isActivator := hypertableActivatorKeys[key]; isActivator {
			continue
		}
		if engineSpecificKey, ok := hypertableReloptionLifts[key]; ok {
			mutation.EngineSpecific[engineSpecificKey] = unquoteReloptionValue(value)
			continue
		}
		mutation.EngineSpecific["TIMESCALE_RELOPTION_"+strings.ToUpper(key)] = value
	}
}

// unquoteReloptionValue strips a single layer of surrounding single quotes from a
// reloption value (un-doubling embedded quotes), normalising the WITH-body form ('ts') to
// the bare form (ts) the call/host parser paths produce for the same timescale key.
// Values without surrounding quotes (numbers, INTERVAL expressions, identifiers) are
// returned unchanged.
//
// Takes value (string) which is the captured reloption value.
//
// Returns string which is the value with one surrounding quote pair removed when present.
func unquoteReloptionValue(value string) string {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return value
}
