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

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

// captureExpressionUntilBoundary reads tokens up to the next top-level comma or close
// paren and returns the concatenated, round-trip-safe source text.
//
// Single-quoted strings are re-wrapped so the captured fragment stays valid SQL. The
// per-call parsers use it when an argument is itself a multi-token expression (for
// example `INTERVAL '1 day'`, `NOW()`, or a function call with its own paren nesting).
//
// The capture honours maxParenDepth, returning errParenDepthExceeded when the nesting
// overflows so adversarial inputs do not provoke unbounded work. It returns an empty
// string when the cursor already sits on a comma or close paren; callers that require at
// least one token should validate before invoking.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the first token of
// the expression.
//
// Returns string which is the captured text (trimmed of surrounding whitespace).
// Returns error when the paren-depth limit is exceeded.
func captureExpressionUntilBoundary(p db_engine_postgres.ParserContext) (string, error) {
	var builder strings.Builder
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		if depth == 0 && (tok.Kind() == db_engine_postgres.TokenComma || tok.Kind() == db_engine_postgres.TokenRightParen) {
			break
		}
		switch tok.Kind() {
		case db_engine_postgres.TokenLeftParen:
			if depth >= maxParenDepth {
				return "", fmt.Errorf("expression at position %d: %w", tok.Position(), errParenDepthExceeded)
			}
			depth++
		case db_engine_postgres.TokenRightParen:
			depth--
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		appendCapturedToken(&builder, tok)
		p.Advance()
	}
	return strings.TrimSpace(builder.String()), nil
}

// appendCapturedToken writes a single token's source-faithful text to builder.
//
// Writing the faithful text keeps the captured fragment valid SQL when replayed verbatim.
// The lexer strips the delimiters and prefixes from string and dollar-quoted literals and
// the quotes from double-quoted identifiers, so each kind is re-wrapped here; emitting
// their bare Value would corrupt the replayed statement.
//
// Takes builder (*strings.Builder) which receives the text.
// Takes tok (db_engine_postgres.Token) which is the token to emit.
func appendCapturedToken(builder *strings.Builder, tok db_engine_postgres.Token) {
	switch tok.Kind() {
	case db_engine_postgres.TokenString, db_engine_postgres.TokenEscapeString:

		builder.WriteByte('\'')
		builder.WriteString(strings.ReplaceAll(tok.Value(), "'", "''"))
		builder.WriteByte('\'')
	case db_engine_postgres.TokenBitString:
		builder.WriteString("B'")
		builder.WriteString(strings.ReplaceAll(tok.Value(), "'", "''"))
		builder.WriteByte('\'')
	case db_engine_postgres.TokenDollarString:
		builder.WriteString(dollarQuote(tok.Value()))
	case db_engine_postgres.TokenIdentifier:
		builder.WriteString(requoteIdentifier(tok.Value()))
	default:
		builder.WriteString(tok.Value())
	}
}

// requoteIdentifier re-wraps an identifier in double quotes when its text could not stand
// as a bare identifier.
//
// Text that contains whitespace or punctuation or starts with a digit can only have come
// from a quoted identifier, so it is re-quoted. Plain identifiers are emitted verbatim.
// Mixed-case identifiers are left bare: the lexer preserves case for quoted and unquoted
// forms, so they are indistinguishable here and re-quoting could change folding.
//
// Takes value (string) which is the identifier text.
//
// Returns string which is the (possibly double-quoted) identifier.
func requoteIdentifier(value string) string {
	if !identifierNeedsQuoting(value) {
		return value
	}
	return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
}

// identifierNeedsQuoting reports whether value contains a rune that may not begin or
// appear within a bare SQL identifier (so it must have been double-quoted), or is empty.
//
// Runes are classified with engine_shared.IsIdentStart / IsIdentPart, which accept the
// Unicode letters and digits the tokeniser already permits, so a multi-byte Unicode
// identifier is left bare rather than needlessly re-quoted.
//
// Takes value (string) which is the identifier text to inspect.
//
// Returns bool which is true when the identifier must be double-quoted.
func identifierNeedsQuoting(value string) bool {
	if value == "" {
		return true
	}
	for index, character := range value {
		if index == 0 {
			if !engine_shared.IsIdentStart(character) {
				return true
			}
			continue
		}
		if !engine_shared.IsIdentPart(character) {
			return true
		}
	}
	return false
}

// dollarQuote wraps content in a dollar-quote ($tag$ ... $tag$) using a tag whose
// delimiter does not appear in content.
//
// Choosing a tag absent from content lets the captured body replay without the delimiter
// being terminated early. The probe appends a trailing '$' to content so a tag that would
// be completed across the content and closing-delimiter boundary is also rejected.
//
// Takes content (string) which is the inner dollar-quoted body.
//
// Returns string which is the safely re-wrapped dollar-quoted literal.
func dollarQuote(content string) string {
	probe := content + "$"
	for _, tag := range []string{"$$", "$piko$", "$pikobody$"} {
		if !strings.Contains(probe, tag) {
			return tag + content + tag
		}
	}
	for index := 0; ; index++ {
		tag := fmt.Sprintf("$piko%d$", index)
		if !strings.Contains(probe, tag) {
			return tag + content + tag
		}
	}
}
