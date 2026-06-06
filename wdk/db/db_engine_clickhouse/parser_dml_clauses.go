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

package db_engine_clickhouse

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseOrderByList reads the body of an ORDER BY clause and records one OrderByColumn
// entry per comma-separated column expression.
//
// Each entry captures the expression text plus its trailing direction (ASC / DESC), nulls
// placement (NULLS FIRST / NULLS LAST), and any WITH FILL [FROM .. TO .. STEP ..] body.
//
// The expression body itself is consumed via the parameter-tracking capture so
// `{name:Type}` placeholders inside the body still register against
// analysis.ParameterReferences. Modifiers stop the expression scan via
// stopAfterOrderByColumn so the per-column direction and fill bodies attach to the right
// column even when an expression contains nested commas (e.g. tuple syntax `(a, b)`).
//
// Takes analysis (*querier_dto.RawQueryAnalysis) whose OrderByColumns and parameter
// references are populated.
func (p *parser) parseOrderByList(analysis *querier_dto.RawQueryAnalysis) {
	for {
		if p.atEnd() {
			return
		}
		column := p.parseOrderByColumn(analysis)
		if column.Expression == "" && column.Direction == querier_dto.OrderDirectionUnspecified &&
			column.Nulls == querier_dto.OrderNullsUnspecified && !column.HasFill {
			return
		}
		analysis.OrderByColumns = append(analysis.OrderByColumns, column)
		if p.current().kind != tokenComma {
			return
		}
		p.advance()
	}
}

// parseOrderByColumn reads a single ORDER BY entry.
//
// The expression body is captured verbatim; ASC / DESC, NULLS FIRST / NULLS LAST, and
// WITH FILL [FROM .. TO .. STEP ..] modifiers are recognised and recorded on the returned
// OrderByColumn.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register any
// captured parameter references against.
//
// Returns querier_dto.OrderByColumn which is the populated entry.
func (p *parser) parseOrderByColumn(analysis *querier_dto.RawQueryAnalysis) querier_dto.OrderByColumn {
	expression := p.captureExpressionTrackingParams(
		analysis, querier_dto.ParameterContextComparison, stopAfterOrderByColumn...,
	)
	column := querier_dto.OrderByColumn{Expression: strings.TrimSpace(expression)}
	column.Direction = p.matchOrderDirection()
	column.Nulls = p.matchOrderNullsPlacement()
	if p.matchKeyword(kwWith) && p.matchKeyword(kwFill) {
		column.HasFill = true
		p.parseOrderByFillBoundaries(analysis, &column)
	}
	return column
}

// matchOrderDirection consumes an optional ASC or DESC keyword.
//
// Returns querier_dto.OrderDirection which is the matched direction, or
// OrderDirectionUnspecified when no direction keyword appears at the cursor.
func (p *parser) matchOrderDirection() querier_dto.OrderDirection {
	if p.matchKeyword(kwAsc) {
		return querier_dto.OrderDirectionAsc
	}
	if p.matchKeyword(kwDesc) {
		return querier_dto.OrderDirectionDesc
	}
	return querier_dto.OrderDirectionUnspecified
}

// matchOrderNullsPlacement consumes an optional `NULLS FIRST` or `NULLS LAST` clause.
//
// When NULLS is followed by an unexpected token the parser rewinds so the caller can
// re-interpret it as part of the next modifier.
//
// Returns querier_dto.OrderNullsPlacement which is the matched placement, or
// OrderNullsUnspecified when no placement clause appears at the cursor.
func (p *parser) matchOrderNullsPlacement() querier_dto.OrderNullsPlacement {
	if !p.isKeyword(kwNulls) {
		return querier_dto.OrderNullsUnspecified
	}
	saved := p.position
	p.advance()
	if p.matchKeyword(kwFirst) {
		return querier_dto.OrderNullsFirst
	}
	if p.matchKeyword(kwLast) {
		return querier_dto.OrderNullsLast
	}
	p.position = saved
	return querier_dto.OrderNullsUnspecified
}

// parseOrderByFillBoundaries consumes any combination of `FROM expr`, `TO expr`, and
// `STEP expr` clauses that may follow `WITH FILL` on an ORDER BY column. Each boundary
// body is captured verbatim onto the OrderByColumn so downstream consumers can re-emit or
// rewrite the clause.
//
// Takes analysis (*querier_dto.RawQueryAnalysis), the analysis to register any captured
// parameter references against.
// Takes column (*querier_dto.OrderByColumn), the column whose Fill fields should be
// populated.
func (p *parser) parseOrderByFillBoundaries(
	analysis *querier_dto.RawQueryAnalysis, column *querier_dto.OrderByColumn,
) {
	for p.isAnyKeyword(kwFrom, kwTo, kwStep) {
		keyword := strings.ToUpper(p.current().value)
		p.advance()
		body := p.captureExpressionTrackingParams(
			analysis, querier_dto.ParameterContextComparison, stopAfterOrderByColumn...,
		)
		body = strings.TrimSpace(body)
		switch keyword {
		case kwFrom:
			column.FillFrom = body
		case kwTo:
			column.FillTo = body
		case kwStep:
			column.FillStep = body
		}
	}
}

// parseLimitWithModifier handles the LIMIT ... WITH TIES / WITH FILL trailing modifier
// and the FROM / TO / STEP fields that may follow WITH FILL.
//
// Pulled out of parseLimitClause to keep the max-control-nesting linter satisfied.
//
// FROM / TO / STEP bodies are captured as textual expressions under the
// engineKeyLimitFillFrom / engineKeyLimitFillTo / engineKeyLimitFillStep keys on the
// analysis EngineSpecific map. Capturing the boundary expressions lets downstream
// consumers (codegen / dynamic-runtime) re-emit or rewrite the LIMIT clause without
// re-parsing the source SQL. Parameter placeholders inside the bodies remain registered
// through consumeExpressionTrackingParams.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) whose EngineSpecific map and parameter
// references are populated.
func (p *parser) parseLimitWithModifier(analysis *querier_dto.RawQueryAnalysis) {
	if p.matchKeyword(kwTies) {
		return
	}
	if !p.matchKeyword(kwFill) {
		return
	}
	fillStops := []string{kwFrom, kwTo, kwStep, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept}
	for p.isAnyKeyword(kwFrom, kwTo, kwStep) {
		boundaryKey := limitFillKey(p.current().value)
		p.advance()
		body := p.captureExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, fillStops...)
		if boundaryKey != "" {
			ensureEngineSpecific(analysis)[boundaryKey] = body
		}
	}
}

// limitFillKey maps a LIMIT FILL boundary keyword (FROM / TO / STEP) to the
// EngineSpecific key under which the boundary expression is recorded. Returns the empty
// string for unrecognised keywords so the caller can skip the capture.
//
// Takes keyword (string), the FROM / TO / STEP token text from the LIMIT clause body.
//
// Returns the EngineSpecific key, or "" for unrecognised keywords.
func limitFillKey(keyword string) string {
	switch strings.ToUpper(keyword) {
	case kwFrom:
		return engineKeyLimitFillFrom
	case kwTo:
		return engineKeyLimitFillTo
	case kwStep:
		return engineKeyLimitFillStep
	}
	return ""
}

// ensureEngineSpecific returns the analysis EngineSpecific map, allocating it lazily on
// first access.
//
// Centralising the lazy initialisation keeps callers free of nil checks and keeps the
// allocation off the hot path for analyses that never attach engine metadata.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) whose map should be returned.
//
// Returns map[string]string which is the (possibly newly allocated) EngineSpecific map.
func ensureEngineSpecific(analysis *querier_dto.RawQueryAnalysis) map[string]string {
	if analysis.EngineSpecific == nil {
		analysis.EngineSpecific = map[string]string{}
	}
	return analysis.EngineSpecific
}

// captureExpressionTrackingParams consumes an expression body just like
// consumeExpressionTrackingParams but additionally returns the concatenated textual body
// of the consumed tokens.
//
// Used by clauses that need both the parameter-tracking side effect and the literal SQL
// text of the captured expression (e.g. LIMIT WITH FILL bodies). The returned text joins
// tokens with a single space because the tokeniser strips inter-token whitespace;
// consumers that care about formatting should normalise further. Parameter placeholders
// render as `{name:Type}` exactly as in the source.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// parameters against.
// Takes context (querier_dto.ParameterContext) which is the context tag applied to any
// captured parameter references.
// Takes stopKeywords (...string) which is the list of top-level keywords that terminate
// the consume loop.
//
// Returns string which is the joined textual body of the consumed tokens.
func (p *parser) captureExpressionTrackingParams(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
	stopKeywords ...string,
) string {
	var builder strings.Builder
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier && identifierMatchesAny(tok.value, stopKeywords) {
			return builder.String()
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return builder.String()
		}
		depth = newDepth
		appendTokenText(&builder, tok)
		p.advance()
	}
	return builder.String()
}

// appendTokenText writes a token's textual form to the supplied builder, separated from
// prior tokens by a single space. Used by captureExpressionTrackingParams to reconstruct
// the source body of a clause expression from the lexed token stream.
//
// String tokens are wrapped in single quotes and parameter tokens are wrapped in `{}` to
// recover the original SQL surface syntax that the tokeniser strips during lexing. All
// other token kinds emit their value verbatim.
//
// The identifier round-trip is lossy for delimited identifiers: the tokeniser unwraps
// backtick and double-quoted identifiers to their bare value and the token struct carries
// no quoting flag, so a name that originally required quoting (a keyword, or one
// containing spaces or special characters) is re-emitted bare here. A consumer that
// re-runs this text through ClickHouse must not assume the output preserves the original
// delimited form.
//
// Takes builder (*strings.Builder), the target output buffer.
// Takes tok (token), the token whose text should be appended.
func appendTokenText(builder *strings.Builder, tok token) {
	if builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	switch tok.kind {
	case tokenString:

		builder.WriteByte('\'')
		builder.WriteString(escapeClickHouseStringBody(tok.value))
		builder.WriteByte('\'')
	case tokenClickHouseParam:
		builder.WriteByte('{')
		builder.WriteString(tok.value)
		builder.WriteByte('}')
	default:
		builder.WriteString(tok.value)
	}
}
