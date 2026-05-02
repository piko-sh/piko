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

package db_engine_postgres

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// StatementKindExtensionBase is the lowest StatementKind value that an extension may
	// claim. Built-in kinds occupy [0, statementKindCount) (currently well under 50
	// entries); extensions use the reserved range [StatementKindExtensionBase,
	// StatementKindExtensionBase + 1000).
	//
	// The 1000-value budget leaves comfortable headroom above the largest built-in count so
	// the reservation boundary does not shift if new built-in kinds land. When two
	// extensions claim the same statement shape (Classify returns the same non-zero kind),
	// the first one registered with WithStatementExtensions wins; subsequent extensions for
	// the same shape are silently ignored. Extensions returning a kind below this base value
	// are rejected at classification time as a defensive guard against accidental collisions
	// with built-in handlers.
	StatementKindExtensionBase StatementKind = 1000

	// TokenIdentifier is a bare or quoted identifier (column name, table name, keyword
	// token).
	TokenIdentifier = tokenIdentifier

	// TokenString is a single-quoted string literal.
	TokenString = tokenString

	// TokenEscapeString is an E'...' escape string literal. Exposed so postgres-derived
	// extensions that re-emit captured bodies can re-wrap it (the token Value has the E'
	// prefix and quotes stripped, so emitting it bare would corrupt the literal).
	TokenEscapeString = tokenEscapeString

	// TokenBitString is a B'...' bit string literal. Exposed for the same re-emission reason
	// as TokenEscapeString.
	TokenBitString = tokenBitString

	// TokenDollarString is a dollar-quoted ($tag$ ... $tag$) string literal.
	//
	// The token Value holds only the inner content, so re-emitting extensions must restore
	// the dollar delimiters.
	TokenDollarString = tokenDollarString

	// TokenNumber is a numeric literal.
	TokenNumber = tokenNumber

	// TokenLeftParen is `(`.
	TokenLeftParen = tokenLeftParen

	// TokenRightParen is `)`.
	TokenRightParen = tokenRightParen

	// TokenComma is `,`.
	TokenComma = tokenComma

	// TokenDot is `.`.
	TokenDot = tokenDot

	// TokenCast is the `::` cast operator.
	//
	// It is exposed so postgres-derived extension engines (e.g. TimescaleDB) can recognise
	// and skip the `'table'::regclass` / `'col'::name` casts that appear in their directive
	// calls.
	TokenCast = tokenCast

	// TokenOperator covers operator symbols such as `=`, `<`, `>`, `:`, `||`, `!~*`, etc.
	TokenOperator = tokenOperator

	// TokenEOF is the end-of-input sentinel.
	TokenEOF = tokenEOF
)

// StatementKind is the exported alias of the private statementKind so extension packages
// can name their own kinds in the reserved range [StatementKindExtensionBase,
// StatementKindExtensionBase + 1000).
//
// Extension-owned kinds are dispatched through the extension registry rather than the
// built-in ddlHandlers array.
type StatementKind = statementKind

// Token is the exported alias of the private token so extensions can inspect the token
// slice handed to their Classify method without importing the unexported lexer types. Use
// Value / Position / Kind accessors below for read-only inspection.
type Token = token

// TokenKind is the exported alias of the private tokenKind so extensions can switch on
// token classifications without importing the unexported lexer types.
type TokenKind = tokenKind

// Value returns the lexed text of the token (identifier text, literal value, operator
// symbol, etc.).
//
// Returns string which is the token's lexed text.
func (t Token) Value() string { return t.value }

// Position returns the byte offset of the token in the source SQL.
//
// Returns int which is the byte offset of the token.
func (t Token) Position() int { return t.position }

// Kind returns the token's TokenKind classification.
//
// Returns TokenKind which is the token's classification.
func (t Token) Kind() TokenKind { return t.kind }

// StatementExtension lets postgres-derived engines (cockroach, timescaledb, yugabyte,
// etc.) recognise and parse SQL their host postgres does not natively support. Each
// extension owns one or more StatementKind values in the [StatementKindExtensionBase,
// StatementKindExtensionBase + 1000) range.
//
// Registered via WithStatementExtensions. The engine's classifier consults built-in
// classifiers first, then walks the extension list in registration order; the first
// non-zero return wins. The dispatcher then looks up the owning extension by re-running
// Classify when ApplyDDL fires.
type StatementExtension interface {
	// Classify inspects the leading tokens of a candidate statement and returns a non-zero
	// StatementKind in the extension's reserved range when this extension claims the
	// statement, or 0 to decline. Must not mutate the input slice.
	Classify(tokens []Token) StatementKind

	// Parse runs after classification, driving the supplied parser context to consume tokens
	// and producing a CatalogueMutation. The kind argument is the value previously returned
	// by Classify, so a single extension can serve multiple statement shapes.
	Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error)
}

// PostParseHook runs after a built-in DDL handler produces a CatalogueMutation.
//
// Hooks may mutate the result in place to attach engine-specific metadata (e.g. lifting
// `timescaledb.compress` keys from an ALTER TABLE SET body into mutation.EngineSpecific).
// Hooks run in registration order; a returned error aborts the chain and propagates from
// ApplyDDL.
//
// The mutation argument is nil when the built-in handler declined to produce one (e.g.
// statementKindUnknown). Hooks should tolerate the nil case and return early.
type PostParseHook func(p ParserContext, kind StatementKind, mutation *querier_dto.CatalogueMutation) error

// ParserContext is the curated surface a StatementExtension or PostParseHook uses to
// drive postgres parsing without re-implementing the tokeniser-aware helpers.
//
// The set is intentionally minimal and grows as concrete extensions need it. Stability is
// best-effort while the API is experimental.
//
// PostParseHook implementations MUST NOT advance the cursor; only read-only methods
// (CurrentToken, Peek, AtEnd, EngineSpecificFromTokens) are safe to call from a hook. The
// mutating methods (Advance, MustKeyword, MatchKeyword, MatchIfNotExists, MatchIfExists,
// ParseQualifiedName, ParseColumnList, ParseReloptionList, ConsumeRemainder,
// ConsumeRemainderAsText) are exposed for StatementExtension.Parse only; a hook that
// advances them after the built-in handler has consumed its tokens corrupts subsequent
// hooks in the chain.
type ParserContext interface {
	// MustKeyword consumes the next token, requiring it to be the named keyword. Panics (via
	// the parser's internal mechanism) if the token does not match; the engine's ApplyDDL
	// recovers panics into a wrapped error.
	MustKeyword(name string)

	// MatchKeyword consumes the current token only if it matches the supplied keyword
	// (case-insensitive). Returns true on consume.
	MatchKeyword(name string) bool

	// MatchIfNotExists consumes a trailing `IF NOT EXISTS` clause if present. Returns true
	// on consume.
	MatchIfNotExists() bool

	// MatchIfExists consumes a trailing `IF EXISTS` clause if present. Returns true on
	// consume.
	MatchIfExists() bool

	// CurrentToken returns the token at the cursor without consuming it. Returns an EOF
	// sentinel when the cursor is past the end.
	CurrentToken() Token

	// Peek returns the token one past the cursor without consuming it. Returns an EOF
	// sentinel when out of range.
	Peek() Token

	// Advance consumes and returns the current token.
	Advance() Token

	// AtEnd reports whether the cursor has reached end of input.
	AtEnd() bool

	// ParseQualifiedName parses an optionally schema-qualified name (`schema.name` or
	// `name`). Returns the parsed parts.
	ParseQualifiedName() (schema string, name string, err error)

	// ParseColumnList parses a `(col1, col2, ...)` identifier list. The opening paren must
	// be at the cursor; the closing paren is consumed.
	ParseColumnList() ([]string, error)

	// ParseColumnType parses and normalises the column type at the cursor, consuming the
	// type tokens.
	//
	// It handles multi-word names, precision/scale/length modifiers, and array suffixes,
	// normalising the result through the engine's type map. Extensions defining their own
	// column syntax (for example TimescaleDB's CREATE HYPERTABLE) call this so column types
	// resolve identically to the base PostgreSQL CREATE TABLE handler instead of degrading
	// to an Unknown `any` type.
	//
	// Returns the normalised SQLType and the array dimension count (0 for a non-array type).
	ParseColumnType() (querier_dto.SQLType, int)

	// ParseReloptionList parses a `( key = value, key = value, ... )` reloption body.
	//
	// The opening paren must be at the cursor; the closing paren is consumed. Values are
	// returned as raw text (single-quoted strings are unwrapped) and keys are
	// case-preserved.
	ParseReloptionList() (map[string]string, error)

	// ConsumeRemainder advances the cursor to end of input. Used by extensions that capture
	// the rest of the statement opaquely.
	ConsumeRemainder()

	// ConsumeRemainderAsText returns the concatenated source text of the remaining tokens
	// and advances to end of input.
	//
	// It is useful for capturing opaque statement bodies for replay (e.g. async data
	// mutations or function bodies). Identifier quoting is preserved when needed.
	ConsumeRemainderAsText() string

	// EngineSpecificFromTokens returns the raw textual reconstruction of the tokens starting
	// at startIndex and ending one past endIndex. Convenience for hooks that captured a body
	// and want to round-trip it into EngineSpecific metadata.
	EngineSpecificFromTokens(startIndex, endIndex int) string

	// Tokens returns the read-only token slice that backs the parser context.
	//
	// Implementations return the underlying slice; callers MUST treat the result as
	// read-only. It is intended for post-parse hooks that need to inspect statement shape
	// after the built-in handler has run skipToStatementEnd, when the cursor sits at EOF and
	// Advance is unsafe to call. Hooks that want to lift trailing reloption bodies (for
	// example TimescaleDB's `CREATE TABLE foo (...) WITH (tsdb.hypertable, ...)` form) walk
	// the slice without mutating parser state.
	Tokens() []Token

	// AnalyseViewBody analyses the SELECT body at the cursor and returns its parsed
	// RawQueryAnalysis without consuming any tokens, so a caller can set the resulting
	// definition on a CreateView mutation (giving the catalogue typed columns) and then
	// still consume the body itself. It returns nil when the body cannot be parsed.
	//
	// The optional columnNames overlay the inferred projection names with an explicit column
	// list (the `CREATE VIEW v (a, b) AS ...` form); pass nil to keep the inferred names. It
	// is used by extensions such as TimescaleDB continuous aggregates, whose `AS SELECT`
	// body must be typed exactly like a plain view.
	AnalyseViewBody(columnNames []string) *querier_dto.RawQueryAnalysis
}

// parserContext is the concrete ParserContext implementation; it wraps a *parser and
// exposes only the methods declared on the interface.
type parserContext struct {
	// p is the wrapped parser whose cursor the context drives.
	p *parser

	// engine carries the PostgresEngine so extensions can reuse its type normalisation.
	engine *PostgresEngine
}

// newParserContext wraps a *parser for handing to extension code.
//
// The wrapper is single-use; do not retain references across the ApplyDDL boundary. The
// engine is carried so extensions can reuse the engine's type normalisation
// (ParseColumnType) instead of reimplementing it.
//
// Takes p (*parser) which is the parser to wrap.
// Takes engine (*PostgresEngine) which supplies type normalisation.
//
// Returns *parserContext which is the single-use wrapper.
func newParserContext(p *parser, engine *PostgresEngine) *parserContext {
	return &parserContext{p: p, engine: engine}
}

// ParseColumnType parses and normalises the column type at the cursor, consuming the type
// tokens. See the ParserContext interface for details.
//
// Returns querier_dto.SQLType which is the normalised column type.
// Returns int which is the array dimension count, 0 for a non-array type.
func (c *parserContext) ParseColumnType() (querier_dto.SQLType, int) {
	return c.p.parseColumnType(c.engine)
}

// MustKeyword consumes the next token as the named keyword or panics.
//
// Takes name (string) which is the required keyword.
func (c *parserContext) MustKeyword(name string) { c.p.mustKeyword(name) }

// MatchKeyword consumes the current token when it matches the named keyword and returns
// whether a match occurred.
//
// Takes name (string) which is the keyword to match.
//
// Returns bool which is true when the token was consumed.
func (c *parserContext) MatchKeyword(name string) bool { return c.p.matchKeyword(name) }

// MatchIfNotExists consumes a trailing `IF NOT EXISTS` clause if present.
//
// The NOT token is required; callers that need `IF EXISTS` semantics must use
// MatchIfExists.
//
// Returns bool which is true when the clause was consumed.
func (c *parserContext) MatchIfNotExists() bool {
	startPosition := c.p.position
	if !c.p.matchKeyword("IF") {
		return false
	}
	if !c.p.matchKeyword("NOT") {
		c.p.position = startPosition
		return false
	}
	if !c.p.matchKeyword("EXISTS") {
		c.p.position = startPosition
		return false
	}
	return true
}

// MatchIfExists consumes a trailing `IF EXISTS` clause if present.
//
// A partial match (IF present but EXISTS missing) restores the cursor to the position
// before IF so subsequent matchers see the original token stream; without the restore an
// `IF NOT EXISTS` lookahead would leave the cursor on NOT and break later keyword
// matching.
//
// Returns bool which is true when the clause was consumed.
func (c *parserContext) MatchIfExists() bool {
	startPosition := c.p.position
	if !c.p.matchKeyword("IF") {
		return false
	}
	if !c.p.matchKeyword("EXISTS") {
		c.p.position = startPosition
		return false
	}
	return true
}

// CurrentToken returns the token at the cursor without consuming it.
//
// Returns Token which is the cursor token, or an EOF sentinel past the end.
func (c *parserContext) CurrentToken() Token { return c.p.current() }

// Peek returns the token one past the cursor without consuming it.
//
// Returns Token which is the lookahead token, or an EOF sentinel out of range.
func (c *parserContext) Peek() Token { return c.p.peek() }

// Advance consumes and returns the current token.
//
// Returns Token which is the consumed token.
func (c *parserContext) Advance() Token { return c.p.advance() }

// AtEnd reports whether the cursor has reached end of input.
//
// Returns bool which is true when the cursor is at end of input.
func (c *parserContext) AtEnd() bool { return c.p.atEnd() }

// ParseQualifiedName parses an optionally schema-qualified name.
//
// Returns schema (string) which is the qualifying schema, empty when absent.
// Returns name (string) which is the object name.
// Returns err (error) when the name cannot be parsed.
func (c *parserContext) ParseQualifiedName() (schema string, name string, err error) {
	return c.p.parseSchemaQualifiedName()
}

// ParseColumnList parses a `(col1, col2, ...)` identifier list.
//
// It accepts bare identifiers and double-quoted identifiers (which the tokeniser emits as
// TokenString).
//
// Returns []string which is the parsed column names.
// Returns error when the opening or closing paren is missing.
func (c *parserContext) ParseColumnList() ([]string, error) {
	if c.p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", c.p.current().position)
	}
	c.p.advance()
	var names []string
	for !c.p.atEnd() {
		current := c.p.current()
		if current.kind != tokenIdentifier && current.kind != tokenString {
			break
		}
		names = append(names, current.value)
		c.p.advance()
		if c.p.current().kind == tokenComma {
			c.p.advance()
			continue
		}
		break
	}
	if c.p.current().kind != tokenRightParen {
		return nil, fmt.Errorf("expected ')' at position %d", c.p.current().position)
	}
	c.p.advance()
	return names, nil
}

// ParseReloptionList parses `( key = value, ... )` and returns the captured key-value
// pairs.
//
// Values that are single-quoted strings have their quotes stripped; other values are
// captured as raw text.
//
// Returns map[string]string which is the captured key-value pairs.
// Returns error when the body is malformed or a value cannot be parsed.
func (c *parserContext) ParseReloptionList() (map[string]string, error) {
	if c.p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", c.p.current().position)
	}
	c.p.advance()
	options := map[string]string{}
	for !c.p.atEnd() && c.p.current().kind != tokenRightParen {
		key, keyErr := c.parseReloptionKey()
		if keyErr != nil {
			return nil, keyErr
		}
		if !c.matchOperator("=") {
			return nil, fmt.Errorf("expected '=' after reloption key %q at position %d", key, c.p.current().position)
		}
		value, valueErr := c.parseReloptionValue()
		if valueErr != nil {
			return nil, valueErr
		}
		options[key] = value
		if c.p.current().kind == tokenComma {
			c.p.advance()
			continue
		}
		break
	}
	if c.p.current().kind != tokenRightParen {
		return nil, fmt.Errorf("expected ')' at position %d", c.p.current().position)
	}
	c.p.advance()
	return options, nil
}

// ConsumeRemainder advances the cursor to end of input in O(1) by jumping straight to the
// end of the token slice. Used by extensions that capture the rest of the statement
// opaquely after their own header parsing has completed.
func (c *parserContext) ConsumeRemainder() {
	c.p.position = len(c.p.tokens)
}

// ConsumeRemainderAsText returns the concatenated source text of the remaining tokens.
// Single-quoted strings are re-wrapped and backtick-style identifiers are preserved.
//
// Returns string which is the trimmed source text of the remaining tokens.
func (c *parserContext) ConsumeRemainderAsText() string {
	var builder strings.Builder
	for !c.p.atEnd() {
		appendTokenText(&builder, c.p.current())
		builder.WriteByte(' ')
		c.p.advance()
	}
	return strings.TrimSpace(builder.String())
}

// Tokens returns the read-only token slice that backs the parser context.
//
// Callers MUST NOT mutate the result; the slice is shared with the active parser. It is
// intended for post-parse hooks that need to inspect statement shape after the built-in
// handler has run skipToStatementEnd. Hooks walk the slice without mutating parser state;
// mutating extensions should still drive parsing through the dedicated MatchKeyword /
// Advance helpers.
//
// Returns []Token which is the read-only token slice backing the context.
func (c *parserContext) Tokens() []Token {
	return c.p.tokens
}

// AnalyseViewBody analyses the SELECT body at the cursor without consuming any tokens,
// returning its parsed RawQueryAnalysis (or nil when it cannot be parsed).
//
// It delegates to the same view-body analyser the built-in CREATE VIEW handler uses, so
// an extension's `AS SELECT` body is typed identically to a plain view.
//
// Takes columnNames ([]string) which optionally overlay the inferred projection names.
//
// Returns *querier_dto.RawQueryAnalysis which is the parsed body, or nil on failure.
func (c *parserContext) AnalyseViewBody(columnNames []string) *querier_dto.RawQueryAnalysis {
	return c.p.analyseViewBody(columnNames)
}

// EngineSpecificFromTokens reconstructs source text from a token range.
//
// It is used by post-parse hooks that captured offsets during the built-in parse and want
// to round-trip the body into metadata.
//
// Takes startIndex (int) which is the first token index, inclusive.
// Takes endIndex (int) which is the index one past the last token.
//
// Returns string which is the reconstructed source text, empty when the range is invalid.
func (c *parserContext) EngineSpecificFromTokens(startIndex, endIndex int) string {
	if startIndex < 0 || endIndex > len(c.p.tokens) || startIndex >= endIndex {
		return ""
	}
	var builder strings.Builder
	for index := startIndex; index < endIndex; index++ {
		appendTokenText(&builder, c.p.tokens[index])
		builder.WriteByte(' ')
	}
	return strings.TrimSpace(builder.String())
}

// parseReloptionKey reads a key, which may be dotted (`namespace.key`).
//
// Returns string which is the parsed key text.
// Returns error when no identifier is present or a dotted segment is missing.
func (c *parserContext) parseReloptionKey() (string, error) {
	if c.p.current().kind != tokenIdentifier {
		return "", fmt.Errorf("expected reloption key at position %d", c.p.current().position)
	}
	var builder strings.Builder
	builder.WriteString(c.p.current().value)
	c.p.advance()
	for c.p.current().kind == tokenDot {
		builder.WriteByte('.')
		c.p.advance()
		if c.p.current().kind != tokenIdentifier {
			return "", fmt.Errorf("expected identifier after '.' at position %d", c.p.current().position)
		}
		builder.WriteString(c.p.current().value)
		c.p.advance()
	}
	return builder.String(), nil
}

// parseReloptionValue reads a value, stopping at the next top-level comma or close paren.
//
// A single literal or identifier is returned verbatim. A multi-token value (e.g.
// `INTERVAL '1 day'`) is captured into a space-separated string with string literals
// re-wrapped.
//
// Returns string which is the captured value text.
// Returns error when the multi-token capture exits with unbalanced parentheses, because
// the outer ParseReloptionList only verifies that the closing paren of the body is
// present and would otherwise accept malformed shapes like `(key = (val,)` where the
// value never closes its own nested paren; the error is errDDLDepthExceeded when the
// captured expression nests parens beyond maxDDLDepth so a maliciously crafted reloption
// body cannot exhaust the goroutine stack.
func (c *parserContext) parseReloptionValue() (string, error) {
	first := c.p.current()
	if isSimpleReloptionValueToken(first.kind) {
		c.p.advance()
		return first.value, nil
	}

	var builder strings.Builder
	depth := 0
	for !c.p.atEnd() {
		tok := c.p.current()
		if depth == 0 && (tok.kind == tokenComma || tok.kind == tokenRightParen) {
			break
		}
		depth = adjustReloptionDepth(tok.kind, depth)
		if depth > maxDDLDepth {
			return "", errDDLDepthExceeded
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		appendTokenText(&builder, tok)
		c.p.advance()
	}
	if depth != 0 {
		return "", fmt.Errorf("unbalanced parentheses in reloption value at position %d", first.position)
	}
	return builder.String(), nil
}

// matchOperator consumes the current token when it is the named operator token (e.g. `=`)
// and returns whether a match occurred.
//
// It is used by reloption parsing where the postgres lexer emits operator tokens for `=`
// rather than treating it as part of an identifier.
//
// Takes value (string) which is the operator symbol to match.
//
// Returns bool which is true when the operator token was consumed.
func (c *parserContext) matchOperator(value string) bool {
	tok := c.p.current()
	if tok.kind == tokenOperator && tok.value == value {
		c.p.advance()
		return true
	}
	return false
}

// isSimpleReloptionValueToken reports whether the supplied token kind represents a
// single-token reloption value that can be returned verbatim without any multi-token
// capture. Strings and numbers qualify; everything else triggers the wider capture loop.
//
// Takes kind (tokenKind) which is the cursor token's classification.
//
// Returns bool which is true when the value is a simple literal.
func isSimpleReloptionValueToken(kind tokenKind) bool {
	return kind == tokenString || kind == tokenNumber
}

// adjustReloptionDepth updates the paren nesting counter for the multi-token reloption
// value capture.
//
// Other token kinds leave the depth untouched. It is extracted so the surrounding loop
// does not need a switch with side-effecting cases and can keep its body shallow.
//
// Takes kind (tokenKind) which is the cursor token's classification.
// Takes depth (int) which is the current paren nesting depth.
//
// Returns int which is the updated depth.
func adjustReloptionDepth(kind tokenKind, depth int) int {
	switch kind {
	case tokenLeftParen:
		return depth + 1
	case tokenRightParen:
		return depth - 1
	default:
		return depth
	}
}

// appendTokenText writes the source-faithful representation of tok to builder,
// re-wrapping every quoted-literal token kind back into its original syntax and emitting
// all other tokens verbatim.
//
// It is used by the text-capture helpers (ConsumeRemainderAsText,
// EngineSpecificFromTokens, parseReloptionValue) so the quoting policy stays consistent
// across them.
//
// The tokeniser stores only the decoded inner content of each literal (quotes, escape
// sequences and dollar tags stripped), so each kind needs its own re-wrapping rule to
// round-trip faithfully. A tokenString re-doubles embedded single quotes inside '...'. A
// tokenEscapeString re-escapes backslashes and re-doubles single quotes inside E'...'. A
// tokenBitString wraps the raw bit digits inside B'...'. A tokenDollarString restores
// $$...$$ delimiters, choosing a non-colliding tag when the body already contains the
// delimiter.
//
// Takes builder (*strings.Builder) which receives the re-wrapped token text.
// Takes tok (token) which is the token to render.
func appendTokenText(builder *strings.Builder, tok token) {
	switch tok.kind {
	case tokenString:
		builder.WriteByte('\'')
		builder.WriteString(strings.ReplaceAll(tok.value, "'", "''"))
		builder.WriteByte('\'')
	case tokenEscapeString:
		builder.WriteString("E'")
		escaped := strings.ReplaceAll(tok.value, "\\", "\\\\")
		builder.WriteString(strings.ReplaceAll(escaped, "'", "''"))
		builder.WriteByte('\'')
	case tokenBitString:
		builder.WriteString("B'")
		builder.WriteString(tok.value)
		builder.WriteByte('\'')
	case tokenDollarString:
		delimiter := dollarStringDelimiter(tok.value)
		builder.WriteString(delimiter)
		builder.WriteString(tok.value)
		builder.WriteString(delimiter)
	default:
		builder.WriteString(tok.value)
	}
}

// dollarStringDelimiter chooses a dollar-quote delimiter that does not appear inside
// body, so re-wrapping a dollar-quoted literal cannot be terminated early by content that
// happens to contain the delimiter. The empty-tag form ($$) is preferred; when the body
// contains it a numbered tag ($pkN$) is selected instead.
//
// Takes body (string) which is the decoded inner content of the literal.
//
// Returns string which is the delimiter to place on both sides of body.
func dollarStringDelimiter(body string) string {
	if !strings.Contains(body, "$$") {
		return "$$"
	}
	for counter := 0; ; counter++ {
		candidate := fmt.Sprintf("$pk%d$", counter)
		if !strings.Contains(body, candidate) {
			return candidate
		}
	}
}
