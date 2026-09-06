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
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// consumeExpressionTrackingParamsCommaAware consumes an expression body like
// consumeExpressionTrackingParams but also stops on a top-level comma.
//
// Clauses such as `LIMIT m, n` and `ORDER BY x, y, ...` use it where commas separate
// sibling expressions rather than continuing a single expression. The function never
// returns an error; the signature is kept void so the loop body stays free of an
// uninspected return value.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
// Takes stopKeywords (...string) which are the clause keywords that halt the scan.
func (p *parser) consumeExpressionTrackingParamsCommaAware(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
	stopKeywords ...string,
) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tokenIsTopLevelStop(tok, stopKeywords) {
			return
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// consumeOneExpression consumes a single expression, terminating at a comma or top-level
// clause keyword.
//
// The expression may be any combination of identifiers, numbers, strings, parameters,
// parenthesised groups, and infix operators. GROUP BY items use it. Any `{name:Type}`
// placeholder is registered against the analysis so a parameter inside a non-column GROUP
// BY expression (for example `GROUP BY toStartOfInterval(ts, {step:UInt32})`) is
// retained.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
func (p *parser) consumeOneExpression(analysis *querier_dto.RawQueryAnalysis, context querier_dto.ParameterContext) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 {
			if tok.kind == tokenComma {
				return
			}
			if tok.kind == tokenIdentifier && isTopClauseKeyword(tok.value) {
				return
			}
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// consumeExpressionTrackingParams consumes an expression body and records every
// `{name:Type}` parameter token it encounters as a parameter reference on the analysis.
//
// WHERE clauses use it where parameter context matters for downstream codegen.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
// Takes stopKeywords (...string) which are the clause keywords that halt the scan.
func (p *parser) consumeExpressionTrackingParams(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
	stopKeywords ...string,
) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier && identifierMatchesAny(tok.value, stopKeywords) {
			return
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// collectClickHouseParametersUntilEnd walks the rest of the statement and registers any
// `{name:Type}` placeholders with the analysis.
//
// INSERT VALUES paths use it where the rest of the statement is parsed opaquely, because
// the driver handles the values list at runtime, but parameter references must still be
// surfaced to the codegen layer so the generated method gets typed arguments.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
func (p *parser) collectClickHouseParametersUntilEnd(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
) {
	for !p.atEnd() {
		tok := p.current()
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		p.advance()
	}
}

// registerClickHouseParameter records a `{name:Type}` placeholder on the analysis. The
// token's value is `name:Type`; we split on `:` to extract the name and the type tag for
// downstream resolution.
//
// Type-parse errors are tracked on the parser so the surrounding analyser (analyseSelect
// / analyseInsert) can surface a warning via the engine's diagnostic channel. The
// parameter is still registered with a nil CastType so codegen falls back to the
// unknown-type path instead of dropping the binding entirely.
//
// A tag that parses cleanly but names no known type (for example {x:Strign}) resolves to
// the Unknown category rather than an error. That case is also recorded on the parser so
// the binding is no longer silently untyped; the parameter keeps its Unknown cast so
// codegen retains the binding.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register the
// placeholder against.
// Takes tok (token) which is the `{name:Type}` placeholder token.
// Takes context (querier_dto.ParameterContext) which is the context to record.
func (p *parser) registerClickHouseParameter(
	analysis *querier_dto.RawQueryAnalysis,
	tok token,
	context querier_dto.ParameterContext,
) {
	name, typeName := splitClickHouseParamBody(tok.value)
	number, exists := p.namedParameterMap[name]
	if !exists {
		p.parameterCount++
		number = p.parameterCount
		p.namedParameterMap[name] = number
	}
	var castType *querier_dto.SQLType
	if typeName != "" {
		t, err := parseClickHouseType(typeName, p.maxTypeParseDepth)

		if err == nil && t.Nullable {
			t.SQLType.Nullable = true
		}
		switch {
		case err != nil:
			if p.firstParameterTypeError == nil {
				p.firstParameterTypeError = fmt.Errorf("clickhouse: parameter %q has malformed type tag %q at position %d: %w", name, typeName, tok.position, err)
			}
		case t.SQLType.Category == querier_dto.TypeCategoryUnknown:

			castType = &t.SQLType
			if p.firstParameterTypeError == nil {
				p.firstParameterTypeError = fmt.Errorf("clickhouse: parameter %q has unrecognised type tag %q at position %d", name, typeName, tok.position)
			}
		default:
			castType = &t.SQLType
		}
	}
	analysis.ParameterReferences = append(analysis.ParameterReferences, querier_dto.RawParameterReference{
		Name:                  name,
		Number:                number,
		Context:               context,
		CastType:              castType,
		ColumnReference:       nil,
		EnclosingFunctionName: "",
		ArgumentOrdinal:       0,
	})
}

// mergeNestedParameterReferences folds a nested parser's parameter references into the
// parent analysis so they appear in the final parameter list.
//
// The nested parser is the one for a CTE body or a FROM-derived subquery. Each named
// placeholder is re-keyed through the parent parser's namedParameterMap so binding
// numbers stay consistent across the whole statement and the same {name:Type} used in
// several scopes collapses to one parameter. This mirrors the INSERT ... SELECT merge in
// analyseInsertBody; without it CTE and derived-subquery parameters are dropped because
// each nested parser numbers from one and never flattens into the outer list.
//
// The first malformed parameter-type tag seen inside a nested body is also lifted onto
// the parent parser so the surrounding analyser surfaces a diagnostic, matching the
// behaviour for top-level SELECT and INSERT placeholders.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the parent analysis to extend.
// Takes nested (*parser) which is the sub-parser that produced nestedAnalysis.
// Takes nestedAnalysis (*querier_dto.RawQueryAnalysis) which holds the nested references.
func (p *parser) mergeNestedParameterReferences(
	analysis *querier_dto.RawQueryAnalysis,
	nested *parser,
	nestedAnalysis *querier_dto.RawQueryAnalysis,
) {
	if nestedAnalysis != nil {
		for index := range nestedAnalysis.ParameterReferences {
			ref := nestedAnalysis.ParameterReferences[index]
			if ref.Name != "" {
				number, exists := p.namedParameterMap[ref.Name]
				if !exists {
					p.parameterCount++
					number = p.parameterCount
					p.namedParameterMap[ref.Name] = number
				}
				ref.Number = number
			}
			analysis.ParameterReferences = append(analysis.ParameterReferences, ref)
		}
	}
	if p.firstParameterTypeError == nil && nested != nil && nested.firstParameterTypeError != nil {
		p.firstParameterTypeError = nested.firstParameterTypeError
	}
}

// identifierMatchesAny reports whether name matches one of the supplied stop keywords
// case-insensitively.
//
// It is extracted to avoid inlining a nested loop in the consume helpers.
//
// Takes name (string) which is the identifier to test.
// Takes stopKeywords ([]string) which are the keywords to match against.
//
// Returns bool which is true when name matches a stop keyword.
func identifierMatchesAny(name string, stopKeywords []string) bool {
	return slices.ContainsFunc(stopKeywords, func(stop string) bool {
		return strings.EqualFold(name, stop)
	})
}

// splitClickHouseParamBody splits the placeholder body `name:Type` into its two halves.
//
// Type may be empty when malformed; the catalogue resolver surfaces a diagnostic in that
// case.
//
// Takes body (string) which is the placeholder body of shape `name:Type`.
//
// Returns name (string) which is the parameter identifier.
// Returns typeName (string) which is the type tag, possibly empty.
func splitClickHouseParamBody(body string) (name string, typeName string) {
	nameSegment, typeSegment, found := strings.Cut(body, ":")
	if !found {
		return body, ""
	}
	return strings.TrimSpace(nameSegment), strings.TrimSpace(typeSegment)
}
