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
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// This recursive-descent expression parser (entry point parseExpression) is unit-tested
// in parser_dml_expressions_test.go and resolves ClickHouse expression types and
// nullability.
//
// The live SELECT analyser consumes projection and predicate expressions opaquely via
// consumeExpressionUntilCommaOrKeyword (see parser_dml.go) rather than building a typed
// parsedExpression tree, so editing the parser does not by itself change analyser output.
// The precedence chain mirrors the postgres engine so the two engines analyse the same
// expression shapes identically. ClickHouse adds the lambda `->` operator at the lowest
// level and the postfix `[index]`, `.element` and `::cast` operators at the highest. Each
// level returns an opaque parsedExpression carrying the inferred SQLType (when statically
// determinable) and the kind tag the downstream analyser uses to decide nullability
// behaviour.

// parsedExpression carries the structured result of parsing an expression.
//
// The expression is not faithfully reconstructed at this level; instead it carries enough
// metadata for the downstream analyser to decide nullability, infer the result type, and
// emit diagnostics about untyped operations.
type parsedExpression struct {
	// referencedColumns lists the direct column references inside this expression, so the
	// analyser can wire up scope-based nullability propagation.
	referencedColumns []querier_dto.ColumnReference

	// resultType is the type of the expression's result. It is the empty SQLType when the
	// type cannot be determined statically, in which case the analyser falls back to
	// TypeCategoryUnknown.
	resultType querier_dto.SQLType

	// kind classifies the expression for downstream nullability and type-resolution
	// decisions.
	kind parsedExpressionKind

	// hasParameter is true when the expression contains a `{name:Type}` placeholder, so the
	// WHERE-clause analyser can decide whether to treat the expression as a constant for
	// nullability inference.
	hasParameter bool

	// hasAggregate is true when the expression contains an aggregate function call. The
	// HAVING clause uses this to validate that aggregate predicates exist.
	hasAggregate bool

	// hasWindow is true when the expression contains a window function call.
	hasWindow bool
}

// parsedExpressionKind tags the shape of a parsed expression. The analyser uses the kind
// to decide nullability and result-type inference.
type parsedExpressionKind uint8

const (
	// expressionKindUnknown is the catch-all for expressions whose shape is not yet
	// introspected.
	expressionKindUnknown parsedExpressionKind = iota

	// expressionKindLiteral is a literal value (number / string / bool / NULL).
	expressionKindLiteral

	// expressionKindColumn is a bare column reference (possibly table-qualified).
	expressionKindColumn

	// expressionKindParameter is a `{name:Type}` placeholder.
	expressionKindParameter

	// expressionKindFunction is a function call.
	expressionKindFunction

	// expressionKindLambda is a lambda expression like `x -> x + 1`.
	expressionKindLambda

	// expressionKindBinary is an infix binary operation.
	expressionKindBinary

	// expressionKindUnary is a prefix unary operation.
	expressionKindUnary

	// expressionKindCast is `expr::Type` or `CAST(expr AS Type)`.
	expressionKindCast

	// expressionKindCase is a `CASE WHEN cond THEN value ELSE other END` form.
	expressionKindCase

	// expressionKindBetween is `expr BETWEEN low AND high`.
	expressionKindBetween

	// expressionKindIn is `expr IN (...)` or `expr IN (SELECT ...)`.
	expressionKindIn

	// expressionKindIsNull is `expr IS NULL` / `expr IS NOT NULL`.
	expressionKindIsNull

	// expressionKindTupleElement is `tuple.fieldName` / `tuple.1`.
	expressionKindTupleElement

	// expressionKindArraySubscript is `arr[index]`.
	expressionKindArraySubscript
)

// parseExpression is the top-level entry point of the expression parser.
//
// It delegates to the lambda-arrow precedence layer which sits below boolean operations.
// The expressionDepth counter is incremented on entry and decremented on exit so deeply
// nested parenthesised expressions cannot drive the parser past maxParseDepth recursive
// frames. When the cap is reached a placeholder expressionKindUnknown result is returned
// so the caller can continue parsing the surrounding statement.
//
// Returns *parsedExpression which is the parsed expression tree.
func (p *parser) parseExpression() *parsedExpression {
	if p.expressionDepth >= p.maxParseDepth {
		return &parsedExpression{kind: expressionKindUnknown}
	}
	p.expressionDepth++
	defer func() { p.expressionDepth-- }()
	return p.parseLambdaExpression()
}

// parseLambdaExpression handles ClickHouse's lambda syntax `param -> body` or `(p1, p2)
// -> body`.
//
// Lambdas appear as arguments to higher-order functions like arrayMap, arrayFilter,
// arrayReduce and arraySort. When no lambda arrow is present the parser falls through to
// the OR-precedence parser.
//
// Returns *parsedExpression which is the parsed lambda or the underlying OR expression.
func (p *parser) parseLambdaExpression() *parsedExpression {
	saved := p.position
	if p.tryConsumeLambdaParameters() {
		if p.current().kind == tokenArrow {
			p.advance()
			body := p.parseOrExpression()
			result := &parsedExpression{kind: expressionKindLambda}
			if body != nil {
				result.resultType = body.resultType
				result.referencedColumns = body.referencedColumns
				result.hasParameter = body.hasParameter
				result.hasAggregate = body.hasAggregate
				result.hasWindow = body.hasWindow
			}
			return result
		}
		p.position = saved
	}

	left := p.parseOrExpression()
	if p.current().kind == tokenArrow {
		p.advance()
		body := p.parseOrExpression()
		merged := mergeFlags(left, body)
		merged.kind = expressionKindLambda
		merged.resultType = body.resultType
		return merged
	}
	return left
}

// tryConsumeLambdaParameters attempts to consume a lambda's parameter list.
//
// The parameter list is a single identifier or a parenthesised tuple. On success the
// cursor is left positioned at the `->` token; otherwise the cursor is left unchanged.
//
// Returns bool which is true when lambda parameters were consumed.
func (p *parser) tryConsumeLambdaParameters() bool {
	saved := p.position
	if p.current().kind == tokenIdentifier && p.peek().kind == tokenArrow {
		p.advance()
		return true
	}
	if p.current().kind != tokenLeftParen {
		return false
	}
	p.advance()
	for {
		if p.current().kind != tokenIdentifier {
			p.position = saved
			return false
		}
		p.advance()
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		if p.current().kind == tokenRightParen {
			p.advance()
			if p.current().kind == tokenArrow {
				return true
			}
			p.position = saved
			return false
		}
		p.position = saved
		return false
	}
}

// parseOrExpression handles `OR` at the lowest non-lambda precedence.
//
// Returns *parsedExpression which is the parsed OR expression or its left operand.
func (p *parser) parseOrExpression() *parsedExpression {
	left := p.parseAndExpression()
	for p.matchKeyword("OR") {
		right := p.parseAndExpression()
		left = mergeBinary(left, right, expressionKindBinary, boolReturnType())
	}
	return left
}

// parseAndExpression handles `AND` at the next-lowest precedence.
//
// Returns *parsedExpression which is the parsed AND expression or its left operand.
func (p *parser) parseAndExpression() *parsedExpression {
	left := p.parseNotExpression()
	for p.matchKeyword("AND") {
		right := p.parseNotExpression()
		left = mergeBinary(left, right, expressionKindBinary, boolReturnType())
	}
	return left
}

// parseNotExpression handles right-associative unary `NOT`.
//
// Column references and flags are propagated from the inner expression so consumers
// (HAVING aggregate checks, scope nullability) see the same dependencies as the original
// expression. A stacked `NOT NOT` chain recurses directly rather than through
// parseExpression, so the depth counter is incremented here too to keep the recursion
// within maxParseDepth frames.
//
// Returns *parsedExpression which is the parsed NOT expression or the underlying
// comparison.
func (p *parser) parseNotExpression() *parsedExpression {
	if p.matchKeyword("NOT") {
		if p.expressionDepth >= p.maxParseDepth {
			return &parsedExpression{kind: expressionKindUnknown}
		}
		p.expressionDepth++
		defer func() { p.expressionDepth-- }()
		inner := p.parseNotExpression()
		result := &parsedExpression{
			kind:       expressionKindUnary,
			resultType: boolReturnType(),
		}
		if inner != nil {
			result.referencedColumns = inner.referencedColumns
			result.hasParameter = inner.hasParameter
			result.hasAggregate = inner.hasAggregate
			result.hasWindow = inner.hasWindow
		}
		return result
	}
	return p.parseComparisonExpression()
}

// parseComparisonExpression handles the left-associative comparison and membership
// operators.
//
// It recognises `=`, `<>`, `!=`, `<`, `>`, `<=`, `>=`, `IN`, `NOT IN`, `BETWEEN`, `LIKE`,
// `ILIKE`, `IS NULL` and `IS NOT NULL`, emitting a binary expression that always produces
// Bool.
//
// Returns *parsedExpression which is the parsed comparison expression or its left
// operand.
func (p *parser) parseComparisonExpression() *parsedExpression {
	left := p.parseBitwiseOrExpression()

	if result := p.tryParseIsNullForm(left); result != nil {
		return result
	}
	if p.matchKeyword("BETWEEN") {
		return p.parseBetweenExpression(left)
	}
	if p.matchKeyword("NOT") {
		return p.parseNotComparisonForm(left)
	}
	if p.matchKeyword("LIKE") || p.matchKeyword("ILIKE") {
		right := p.parseBitwiseOrExpression()
		return mergeBinary(left, right, expressionKindBinary, boolReturnType())
	}
	if p.matchKeyword("IN") {
		return p.parseInExpression(left)
	}
	if p.isAnyComparisonOperator() {
		p.advance()
		right := p.parseBitwiseOrExpression()
		return mergeBinary(left, right, expressionKindBinary, boolReturnType())
	}
	return left
}

// tryParseIsNullForm consumes an `IS [NOT] NULL` suffix when present and returns the
// expression.
//
// When the cursor is not on an IS clause it returns nil so the caller can fall through to
// the other comparison forms. The `IS NOT NULL` versus `IS NULL` distinction is
// recognised but not preserved beyond the parser-local kind tag, because the downstream
// analyser rebuilds typed IsNullExpression DTOs from SQL text when codegen needs the
// distinction.
//
// Takes left (*parsedExpression) which is the operand to the left of the IS clause.
//
// Returns *parsedExpression which is the IS NULL expression, or nil when no IS clause
// follows.
func (p *parser) tryParseIsNullForm(left *parsedExpression) *parsedExpression {
	saved := p.position
	if !p.matchKeyword("IS") {
		return nil
	}
	p.matchKeyword("NOT")
	if !p.matchKeyword("NULL") {
		p.position = saved
		return nil
	}
	return &parsedExpression{
		kind:              expressionKindIsNull,
		resultType:        boolReturnType(),
		referencedColumns: left.referencedColumns,
		hasParameter:      left.hasParameter,
		hasAggregate:      left.hasAggregate,
		hasWindow:         left.hasWindow,
	}
}

// parseBetweenExpression handles the `BETWEEN lo AND hi` suffix.
//
// The cursor is expected to be positioned just past the BETWEEN keyword. If AND does not
// follow the lower expression the parser rewinds because the BETWEEN was not a complete
// clause, and the left expression is returned unchanged.
//
// Takes left (*parsedExpression) which is the operand to the left of the BETWEEN keyword.
//
// Returns *parsedExpression which is the BETWEEN expression, or left when the clause is
// partial.
func (p *parser) parseBetweenExpression(left *parsedExpression) *parsedExpression {
	savedBefore := p.position
	low := p.parseBitwiseOrExpression()
	if !p.matchKeyword("AND") {
		p.position = savedBefore
		return left
	}
	high := p.parseBitwiseOrExpression()
	columns := slices.Concat(left.referencedColumns, low.referencedColumns, high.referencedColumns)
	return &parsedExpression{
		kind:              expressionKindBetween,
		resultType:        boolReturnType(),
		referencedColumns: columns,
		hasParameter:      left.hasParameter || low.hasParameter || high.hasParameter,
		hasAggregate:      left.hasAggregate || low.hasAggregate || high.hasAggregate,
		hasWindow:         left.hasWindow || low.hasWindow || high.hasWindow,
	}
}

// parseNotComparisonForm handles the NOT-prefixed comparison variants.
//
// The recognised variants are NOT LIKE, NOT ILIKE, NOT IN and NOT BETWEEN. The cursor has
// already consumed the NOT keyword, and the unaltered left side is returned when no
// NOT-comparison suffix follows.
//
// Takes left (*parsedExpression) which is the operand to the left of the NOT keyword.
//
// Returns *parsedExpression which is the NOT-comparison expression, or left when none
// follows.
func (p *parser) parseNotComparisonForm(left *parsedExpression) *parsedExpression {
	if p.matchKeyword("LIKE") || p.matchKeyword("ILIKE") {
		right := p.parseBitwiseOrExpression()
		return mergeBinary(left, right, expressionKindBinary, boolReturnType())
	}
	if p.matchKeyword("IN") {
		return p.parseInExpression(left)
	}
	if p.matchKeyword("BETWEEN") {
		return p.parseBetweenExpression(left)
	}
	return left
}

// inBodyStartsWithSubquery reports whether the IN-body at the cursor opens a subquery.
//
// It accounts for arbitrary leading opening parens, such as `IN ((SELECT ...))` wrapping
// the subquery in an extra paren. The cursor is not advanced because this is pure
// lookahead.
//
// Returns bool which is true when the first non-paren token is the SELECT keyword.
func (p *parser) inBodyStartsWithSubquery() bool {
	offset := 0
	for {
		next := p.peekAt(p.position + offset)
		if next.kind != tokenLeftParen {
			break
		}
		offset++
	}
	candidate := p.peekAt(p.position + offset)
	return candidate.kind == tokenIdentifier && strings.EqualFold(candidate.value, "SELECT")
}

// parseInExpression parses the body of an `IN (...)` predicate after the IN keyword.
//
// The body may be a value list such as `IN (1, 2, 3)`, a subquery such as `IN (SELECT
// ...)`, or a tuple such as `IN ((a, b), (c, d))`. To preserve parameter and column
// references appearing inside the body, the parser walks each comma-separated element via
// parseExpression rather than skipping the balanced parens wholesale. Subquery bodies are
// re-parsed through a nested analyseSelect so that placeholder references inside `IN
// (SELECT ...)` lift onto the outer analysis, matching the behaviour of
// parseExistsExpression.
//
// Takes left (*parsedExpression) which is the operand to the left of the IN keyword.
//
// Returns *parsedExpression which is the IN expression, or left when no parenthesised
// body opens.
func (p *parser) parseInExpression(left *parsedExpression) *parsedExpression {
	if p.current().kind != tokenLeftParen {
		return left
	}
	p.advance()
	state := newInExpressionState(left)

	if p.inBodyStartsWithSubquery() {
		p.absorbInSubqueryBody(&state)
		return state.finalise(boolReturnType())
	}

	for !p.atEnd() && p.current().kind != tokenRightParen {
		element := p.parseExpression()
		state.absorb(element)
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return state.finalise(boolReturnType())
}

// inExpressionState accumulates the column-reference list and flag bits while
// parseInExpression walks its body. Pulled out so the IN parser body is short enough to
// satisfy the cognitive-complexity linter; finalise produces the resulting
// parsedExpression.
type inExpressionState struct {
	// columns accumulates the column references collected from the IN body.
	columns []querier_dto.ColumnReference

	// hasParameter is true when any absorbed element contains a parameter placeholder.
	hasParameter bool

	// hasAggregate is true when any absorbed element contains an aggregate call.
	hasAggregate bool

	// hasWindow is true when any absorbed element contains a window call.
	hasWindow bool
}

// newInExpressionState constructs the accumulator from the IN expression's left operand.
//
// Takes left (*parsedExpression) which is the operand to the left of the IN keyword.
//
// Returns inExpressionState which is the accumulator seeded from left.
func newInExpressionState(left *parsedExpression) inExpressionState {
	return inExpressionState{
		columns:      append([]querier_dto.ColumnReference{}, left.referencedColumns...),
		hasParameter: left.hasParameter,
		hasAggregate: left.hasAggregate,
		hasWindow:    left.hasWindow,
	}
}

// absorb merges a parsed element's references and flag bits into the accumulator.
//
// A nil element is ignored, mirroring the behaviour where parseExpression may return nil
// for empty input.
//
// Takes element (*parsedExpression) which is the parsed IN-body element to merge.
func (state *inExpressionState) absorb(element *parsedExpression) {
	if element == nil {
		return
	}
	state.columns = append(state.columns, element.referencedColumns...)
	state.hasParameter = state.hasParameter || element.hasParameter
	state.hasAggregate = state.hasAggregate || element.hasAggregate
	state.hasWindow = state.hasWindow || element.hasWindow
}

// finalise produces the IN expression from the accumulated state.
//
// Takes resultType (querier_dto.SQLType) which is the result type to assign to the
// expression.
//
// Returns *parsedExpression which is the IN expression carrying the accumulated
// references.
func (state *inExpressionState) finalise(resultType querier_dto.SQLType) *parsedExpression {
	return &parsedExpression{
		kind:              expressionKindIn,
		resultType:        resultType,
		referencedColumns: state.columns,
		hasParameter:      state.hasParameter,
		hasAggregate:      state.hasAggregate,
		hasWindow:         state.hasWindow,
	}
}

// absorbInSubqueryBody re-parses an `IN (SELECT ...)` body so its references lift onto
// the outer analysis.
//
// It collects the rest of the body up to and including the matching close paren, then
// re-parses the captured tokens through a nested analyseSelect so that any `{name:Type}`
// placeholder references and column references inside the subquery surface on the outer
// analysis. It mirrors the recovery behaviour of parseExistsExpression, so a malformed
// subquery body degrades to an opaque skip rather than failing the outer parse. The
// leading `(` has already been consumed by the caller, and this helper consumes the
// matching `)` before returning.
//
// Takes state (*inExpressionState) which is the accumulator to merge the subquery
// references into.
func (p *parser) absorbInSubqueryBody(state *inExpressionState) {
	body := p.collectInSubqueryBodyTokens()
	if len(body) == 0 {
		return
	}
	nested := newParser(body)
	nested.analysisDepth = p.analysisDepth
	nested.maxParseDepth = p.maxParseDepth
	nestedAnalysis, nestedErr := nested.analyseSelect()
	if nestedErr != nil || nestedAnalysis == nil {
		return
	}
	for index := range nestedAnalysis.ParameterReferences {
		ref := nestedAnalysis.ParameterReferences[index]
		if ref.Name == "" {
			continue
		}
		state.hasParameter = true
		break
	}
	for index := range nestedAnalysis.OutputColumns {
		column := nestedAnalysis.OutputColumns[index]
		if column.ColumnName == "" {
			continue
		}
		state.columns = append(state.columns, querier_dto.ColumnReference{
			TableAlias: column.TableAlias,
			ColumnName: column.ColumnName,
		})
	}
}

// collectInSubqueryBodyTokens drains the open `IN (SELECT ...)` body into a token slice.
//
// It collects up to and including the matching close paren. The returned slice excludes
// the trailing `)` so a fresh parser can run analyseSelect on the captured tokens without
// tripping the closing paren as a stray token.
//
// Returns []token which are the captured subquery body tokens without the trailing close
// paren.
func (p *parser) collectInSubqueryBodyTokens() []token {
	var body []token
	depth := 1
	for !p.atEnd() && depth > 0 {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
			if depth == 0 {
				p.advance()
				return body
			}
		default:
		}
		body = append(body, tok)
		p.advance()
	}
	return body
}

// isAnyComparisonOperator reports whether the current token is a comparison operator.
//
// The recognised operators are =, <>, !=, <, >, <= and >=.
//
// Returns bool which is true when the current token is one of those operators.
func (p *parser) isAnyComparisonOperator() bool {
	tok := p.current()
	if tok.kind != tokenOperator {
		return false
	}
	switch tok.value {
	case "=", "<", ">", "<=", ">=", "<>", "!=":
		return true
	}
	return false
}

// parseBitwiseOrExpression handles bitwise `|`.
//
// Returns *parsedExpression which is the parsed bitwise-OR expression or its left
// operand.
func (p *parser) parseBitwiseOrExpression() *parsedExpression {
	left := p.parseBitwiseAndExpression()
	for p.isOperator("|") {
		p.advance()
		right := p.parseBitwiseAndExpression()
		left = mergeBinary(left, right, expressionKindBinary, left.resultType)
	}
	return left
}

// parseBitwiseAndExpression handles bitwise `&`.
//
// Returns *parsedExpression which is the parsed bitwise-AND expression or its left
// operand.
func (p *parser) parseBitwiseAndExpression() *parsedExpression {
	left := p.parseShiftExpression()
	for p.isOperator("&") {
		p.advance()
		right := p.parseShiftExpression()
		left = mergeBinary(left, right, expressionKindBinary, left.resultType)
	}
	return left
}

// parseShiftExpression handles the `<<` and `>>` shift operators.
//
// Returns *parsedExpression which is the parsed shift expression or its left operand.
func (p *parser) parseShiftExpression() *parsedExpression {
	left := p.parseAdditiveExpression()
	for p.isOperator("<<") || p.isOperator(">>") {
		p.advance()
		right := p.parseAdditiveExpression()
		left = mergeBinary(left, right, expressionKindBinary, left.resultType)
	}
	return left
}

// parseAdditiveExpression handles the `+`, `-` and `||` (concat) operators.
//
// Returns *parsedExpression which is the parsed additive expression or its left operand.
func (p *parser) parseAdditiveExpression() *parsedExpression {
	left := p.parseMultiplicativeExpression()
	for p.isOperator("+") || p.isOperator("-") || p.isOperator("||") {
		p.advance()
		right := p.parseMultiplicativeExpression()
		left = mergeBinary(left, right, expressionKindBinary, left.resultType)
	}
	return left
}

// parseMultiplicativeExpression handles the `*`, `/` and `%` operators.
//
// Returns *parsedExpression which is the parsed multiplicative expression or its left
// operand.
func (p *parser) parseMultiplicativeExpression() *parsedExpression {
	left := p.parseUnaryExpression()
	for p.isOperator("*") || p.isOperator("/") || p.isOperator("%") || p.current().kind == tokenStar {
		p.advance()
		right := p.parseUnaryExpression()
		left = mergeBinary(left, right, expressionKindBinary, left.resultType)
	}
	return left
}

// parseUnaryExpression handles the unary `+` and `-` operators.
//
// A stacked sign chain (for example `- - - x`) recurses directly rather than through
// parseExpression, so the depth counter is incremented here too to keep the recursion
// within maxParseDepth frames. When the cap is reached a placeholder
// expressionKindUnknown result is returned so the caller can continue.
//
// Returns *parsedExpression which is the parsed unary expression or the underlying
// postfix term.
func (p *parser) parseUnaryExpression() *parsedExpression {
	if p.isOperator("-") || p.isOperator("+") {
		if p.expressionDepth >= p.maxParseDepth {
			return &parsedExpression{kind: expressionKindUnknown}
		}
		p.expressionDepth++
		defer func() { p.expressionDepth-- }()
		p.advance()
		inner := p.parseUnaryExpression()
		return &parsedExpression{
			kind:              expressionKindUnary,
			resultType:        inner.resultType,
			referencedColumns: inner.referencedColumns,
			hasParameter:      inner.hasParameter,
			hasAggregate:      inner.hasAggregate,
			hasWindow:         inner.hasWindow,
		}
	}
	return p.parsePostfixExpression()
}

// parsePostfixExpression handles the postfix operators on a primary expression.
//
// The handled operators are array subscript `arr[index]`, tuple element access
// `tuple.name` or `tuple.1` and cast `expr::Type`. Each wrap preserves the column
// references and flags of the underlying expression so downstream analysis (HAVING and
// nullability) can attribute usage correctly.
//
// Returns *parsedExpression which is the parsed postfix expression or the underlying
// primary.
func (p *parser) parsePostfixExpression() *parsedExpression {
	left := p.parsePrimaryExpression()
	for {
		switch p.current().kind {
		case tokenLeftBracket:
			left = p.consumePostfixArraySubscript(left)
		case tokenCast:
			left = p.consumePostfixCast(left)
		case tokenDot:
			next := p.peek()
			if next.kind != tokenIdentifier && next.kind != tokenNumber {
				return left
			}
			left = p.consumePostfixTupleElement(left)
		default:
			return left
		}
	}
}

// consumePostfixArraySubscript handles a `[N]` array-subscript suffix on the left
// operand.
//
// It advances the cursor past the bracketed body.
//
// Takes left (*parsedExpression) which is the operand being subscripted.
//
// Returns *parsedExpression which is the array-subscript expression wrapping left.
func (p *parser) consumePostfixArraySubscript(left *parsedExpression) *parsedExpression {
	p.advance()
	inner := p.parseExpression()
	if p.current().kind == tokenRightBracket {
		p.advance()
	}
	columns := append([]querier_dto.ColumnReference{}, left.referencedColumns...)
	if inner != nil {
		columns = append(columns, inner.referencedColumns...)
	}
	subscript := &parsedExpression{
		kind:              expressionKindArraySubscript,
		resultType:        elementTypeOrSelf(left.resultType),
		referencedColumns: columns,
		hasParameter:      left.hasParameter,
		hasAggregate:      left.hasAggregate,
		hasWindow:         left.hasWindow,
	}
	if inner != nil {
		subscript.hasParameter = subscript.hasParameter || inner.hasParameter
		subscript.hasAggregate = subscript.hasAggregate || inner.hasAggregate
		subscript.hasWindow = subscript.hasWindow || inner.hasWindow
	}
	return subscript
}

// consumePostfixCast handles a `::Type` cast suffix on the left operand.
//
// The cast target type is read but discarded because the analyser carries forward the
// left expression's type for compatibility with downstream nullability propagation.
//
// Takes left (*parsedExpression) which is the operand being cast.
//
// Returns *parsedExpression which is the cast expression wrapping left.
func (p *parser) consumePostfixCast(left *parsedExpression) *parsedExpression {
	p.advance()
	_ = p.readCastTargetType()
	return &parsedExpression{
		kind:              expressionKindCast,
		resultType:        left.resultType,
		referencedColumns: left.referencedColumns,
		hasParameter:      left.hasParameter,
		hasAggregate:      left.hasAggregate,
		hasWindow:         left.hasWindow,
	}
}

// consumePostfixTupleElement handles the `.field` or `.index` tuple element selector.
//
// The caller has already verified that the token after the dot is an identifier or
// number.
//
// Takes left (*parsedExpression) which is the tuple operand being selected from.
//
// Returns *parsedExpression which is the tuple-element expression wrapping left.
func (p *parser) consumePostfixTupleElement(left *parsedExpression) *parsedExpression {
	p.advance()
	selector := p.current()
	p.advance()
	return &parsedExpression{
		kind:              expressionKindTupleElement,
		resultType:        tupleFieldType(left.resultType, selector),
		referencedColumns: left.referencedColumns,
		hasParameter:      left.hasParameter,
		hasAggregate:      left.hasAggregate,
		hasWindow:         left.hasWindow,
	}
}

// readCastTargetType reads the type identifier after a `::` cast operator.
//
// The identifier may carry parameters such as `Decimal(18, 4)`. The cursor is left
// positioned just past the type body: when the type has parameter parens the closing `)`
// is consumed, and for bare type names the cursor sits on the token following the
// identifier. This unconditional consumption avoids a desync that would otherwise leave
// the caller's cursor on the closing paren.
//
// Returns string which is the raw type text for downstream type normalisation.
func (p *parser) readCastTargetType() string {
	var builder strings.Builder
	if p.current().kind == tokenIdentifier {
		builder.WriteString(p.current().value)
		p.advance()
	}
	if p.current().kind != tokenLeftParen {
		return builder.String()
	}
	builder.WriteByte('(')
	p.advance()
	depth := 1
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
			builder.WriteByte('(')
		case tokenRightParen:
			depth--
			builder.WriteByte(')')
		case tokenComma:
			builder.WriteString(", ")
		case tokenString:
			builder.WriteByte('\'')
			builder.WriteString(tok.value)
			builder.WriteByte('\'')
		default:

			builder.WriteString(tok.value)
		}
		p.advance()
	}
	return builder.String()
}

// parsePrimaryExpression handles literals, parenthesised expressions, identifiers,
// function calls, parameters and the CASE WHEN form.
//
// Returns *parsedExpression which is the parsed primary expression.
func (p *parser) parsePrimaryExpression() *parsedExpression {
	tok := p.current()
	switch tok.kind {
	case tokenNumber:
		p.advance()
		return &parsedExpression{
			kind:       expressionKindLiteral,
			resultType: literalNumberType(tok.value),
		}
	case tokenString:
		p.advance()
		return &parsedExpression{
			kind:       expressionKindLiteral,
			resultType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		}
	case tokenClickHouseParam:
		p.advance()
		return &parsedExpression{
			kind:         expressionKindParameter,
			hasParameter: true,
			resultType:   typeFromParamBody(tok.value),
		}
	case tokenLeftParen:
		return p.parseParenthesisedExpression()
	case tokenIdentifier:
		return p.parseIdentifierExpression()
	default:
	}

	p.advance()
	return &parsedExpression{kind: expressionKindUnknown}
}

// parseParenthesisedExpression parses `(expr)` or a tuple constructor.
//
// For the tuple form each comma-separated element's column references and flags are
// accumulated into the result so downstream analysis sees the full set of dependencies.
//
// Returns *parsedExpression which is the inner expression or the accumulated tuple
// expression.
func (p *parser) parseParenthesisedExpression() *parsedExpression {
	p.advance()
	inner := p.parseExpression()
	if p.current().kind != tokenComma {
		if p.current().kind == tokenRightParen {
			p.advance()
		}
		return inner
	}

	columns := append([]querier_dto.ColumnReference{}, inner.referencedColumns...)
	hasParameter := inner.hasParameter
	hasAggregate := inner.hasAggregate
	hasWindow := inner.hasWindow
	for p.current().kind == tokenComma {
		p.advance()
		element := p.parseExpression()
		if element == nil {
			continue
		}
		columns = append(columns, element.referencedColumns...)
		hasParameter = hasParameter || element.hasParameter
		hasAggregate = hasAggregate || element.hasAggregate
		hasWindow = hasWindow || element.hasWindow
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return &parsedExpression{
		kind:              expressionKindUnknown,
		resultType:        inner.resultType,
		referencedColumns: columns,
		hasParameter:      hasParameter,
		hasAggregate:      hasAggregate,
		hasWindow:         hasWindow,
	}
}

// parseCastKeywordExpression handles `CAST(expr AS Type)`.
//
// The CAST keyword has already been consumed and the cursor is on the opening paren. The
// inner expression is parsed recursively so its column references and flags propagate to
// the cast result.
//
// Returns *parsedExpression which is the cast expression carrying the inner type and
// references.
func (p *parser) parseCastKeywordExpression() *parsedExpression {
	if p.current().kind != tokenLeftParen {
		return &parsedExpression{kind: expressionKindCast}
	}
	p.advance()
	inner := p.parseExpression()

	if p.matchKeyword("AS") {
		_ = p.readCastTargetType()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	if inner == nil {
		return &parsedExpression{kind: expressionKindCast}
	}
	return &parsedExpression{
		kind:              expressionKindCast,
		resultType:        inner.resultType,
		referencedColumns: inner.referencedColumns,
		hasParameter:      inner.hasParameter,
		hasAggregate:      inner.hasAggregate,
		hasWindow:         inner.hasWindow,
	}
}

// parseIntervalLiteralExpression handles `INTERVAL <magnitude> <unit>` literals.
//
// The INTERVAL keyword has already been consumed. ClickHouse only accepts a single
// literal or identifier for the magnitude rather than a compound expression, so the
// parser uses the primary-expression level rather than the full expression chain to avoid
// swallowing the unit identifier into the magnitude.
//
// Returns *parsedExpression which is the interval literal carrying its magnitude
// references.
func (p *parser) parseIntervalLiteralExpression() *parsedExpression {
	magnitude := p.parsePrimaryExpression()
	if p.current().kind == tokenIdentifier {
		p.advance()
	}
	intervalType := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryTemporal,
		EngineName: "Interval",
	}
	if magnitude == nil {
		return &parsedExpression{kind: expressionKindLiteral, resultType: intervalType}
	}
	return &parsedExpression{
		kind:              expressionKindLiteral,
		resultType:        intervalType,
		referencedColumns: magnitude.referencedColumns,
		hasParameter:      magnitude.hasParameter,
		hasAggregate:      magnitude.hasAggregate,
		hasWindow:         magnitude.hasWindow,
	}
}

// parseIdentifierExpression handles bare identifiers, function calls and CASE
// expressions.
//
// Bare identifiers may be column references or keyword-like calls.
//
// Returns *parsedExpression which is the parsed identifier, function call or CASE
// expression.
func (p *parser) parseIdentifierExpression() *parsedExpression {
	tok := p.current()
	uppered := strings.ToUpper(tok.value)

	switch uppered {
	case "CASE":
		return p.parseCaseExpression()
	case "CAST":
		p.advance()
		return p.parseCastKeywordExpression()
	case "INTERVAL":
		p.advance()
		return p.parseIntervalLiteralExpression()
	case "EXISTS":
		return p.parseExistsExpression()
	case "TRUE", "FALSE":
		p.advance()
		return &parsedExpression{kind: expressionKindLiteral, resultType: querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"}}
	case "NULL":
		p.advance()
		return &parsedExpression{kind: expressionKindLiteral}
	}

	identifier := tok.value
	p.advance()

	if p.current().kind == tokenLeftParen {
		return p.parseFunctionCallExpression(identifier)
	}

	if p.current().kind == tokenDot {
		p.advance()
		second := p.current()
		if second.kind == tokenIdentifier {
			p.advance()
			if p.current().kind == tokenLeftParen {
				return p.parseFunctionCallExpression(identifier + "." + second.value)
			}
			return &parsedExpression{
				kind:       expressionKindColumn,
				resultType: querier_dto.SQLType{},
				referencedColumns: []querier_dto.ColumnReference{{
					TableAlias: identifier,
					ColumnName: second.value,
				}},
			}
		}
	}

	return &parsedExpression{
		kind:       expressionKindColumn,
		resultType: querier_dto.SQLType{},
		referencedColumns: []querier_dto.ColumnReference{{
			ColumnName: identifier,
		}},
	}
}

// parseExistsExpression handles the `EXISTS (SELECT ...)` predicate.
//
// The EXISTS keyword has already been consumed by the dispatch above. This helper
// consumes the parenthesised subquery body, re-parses it through analyseSelect so any
// column references and `{name:Type}` parameter placeholders inside the body surface on
// the surrounding analysis, and returns a Bool-typed parsedExpression. The engine cannot
// access the analysis pointer from inside parsePrimaryExpression, so re-parsing here is a
// self-contained nested parser whose output is discarded. Parameter and column references
// from the body still register against the outer parser via referencedColumns and
// hasParameter so the surrounding consumeExpression helpers attach them to the right
// slot. Subquery bodies that fail to parse fall back to a Bool-typed expression with no
// references so the surrounding statement still classifies correctly.
//
// Returns *parsedExpression which is the Bool-typed EXISTS expression carrying body
// references.
func (p *parser) parseExistsExpression() *parsedExpression {
	p.advance()
	boolType := querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"}
	if p.current().kind != tokenLeftParen {
		return &parsedExpression{kind: expressionKindUnknown, resultType: boolType}
	}
	body, bodyErr := p.collectParenthesised()
	if bodyErr != nil {
		return &parsedExpression{kind: expressionKindUnknown, resultType: boolType}
	}
	nested := newParser(body)
	nested.analysisDepth = p.analysisDepth
	nested.maxParseDepth = p.maxParseDepth
	nestedAnalysis, nestedErr := nested.analyseSelect()
	result := &parsedExpression{kind: expressionKindUnknown, resultType: boolType}
	if nestedErr != nil || nestedAnalysis == nil {
		return result
	}
	for index := range nestedAnalysis.ParameterReferences {
		ref := nestedAnalysis.ParameterReferences[index]
		if ref.Name == "" {
			continue
		}
		result.hasParameter = true
		break
	}
	for index := range nestedAnalysis.OutputColumns {
		column := nestedAnalysis.OutputColumns[index]
		if column.ColumnName == "" {
			continue
		}
		result.referencedColumns = append(result.referencedColumns, querier_dto.ColumnReference{
			TableAlias: column.TableAlias,
			ColumnName: column.ColumnName,
		})
	}
	return result
}

// parseCaseExpression handles the `CASE WHEN cond THEN value ELSE other END` shape.
//
// It accumulates column references and flags from every sub-expression (selector,
// predicates, results and else branch) so the resulting CASE expression reports the union
// of dependencies. The first result expression's type is used as the CASE result type
// when present, and downstream analysis can widen via the function resolver if branch
// types differ.
//
// Returns *parsedExpression which is the CASE expression carrying the accumulated
// dependencies.
func (p *parser) parseCaseExpression() *parsedExpression {
	p.advance()
	accumulator := newCaseAccumulator()

	if !p.isAnyKeyword("WHEN") && !p.atEnd() {
		accumulator.absorbReferences(p.parseExpression())
	}

	for p.matchKeyword("WHEN") {
		accumulator.absorbReferences(p.parseExpression())
		if !p.matchKeyword("THEN") {
			break
		}
		accumulator.absorbBranchResult(p.parseExpression())
	}
	if p.matchKeyword("ELSE") {
		accumulator.absorbBranchResult(p.parseExpression())
	}
	p.matchKeyword("END")
	return accumulator.finalise()
}

// caseExpressionAccumulator gathers the column references and flag bits collected from a
// CASE expression's selector / predicates / result branches.
type caseExpressionAccumulator struct {
	// columns accumulates the column references collected from every CASE sub-expression.
	columns []querier_dto.ColumnReference

	// resultType is the promoted CASE result type taken from the first concrete branch
	// result.
	resultType querier_dto.SQLType

	// hasParameter is true when any absorbed sub-expression contains a parameter
	// placeholder.
	hasParameter bool

	// hasAggregate is true when any absorbed sub-expression contains an aggregate call.
	hasAggregate bool

	// hasWindow is true when any absorbed sub-expression contains a window call.
	hasWindow bool
}

// newCaseAccumulator returns an empty accumulator.
//
// The result type starts at the zero SQLType, which carries TypeCategoryInteger as its
// category because that is the iota base for SQLTypeCategory. absorbBranchResult tightens
// its update condition so the same expressions produce the same downstream resolution.
//
// Returns *caseExpressionAccumulator which is the empty accumulator ready to absorb
// branches.
func newCaseAccumulator() *caseExpressionAccumulator {
	return &caseExpressionAccumulator{
		columns: []querier_dto.ColumnReference{},
	}
}

// absorbReferences merges an expression's references and flag bits into the accumulator.
//
// It is used for the selector and the WHEN predicate branches whose result type does not
// feed the CASE return type.
//
// Takes expr (*parsedExpression) which is the sub-expression whose references are merged.
func (a *caseExpressionAccumulator) absorbReferences(expr *parsedExpression) {
	if expr == nil {
		return
	}
	a.columns = append(a.columns, expr.referencedColumns...)
	a.hasParameter = a.hasParameter || expr.hasParameter
	a.hasAggregate = a.hasAggregate || expr.hasAggregate
	a.hasWindow = a.hasWindow || expr.hasWindow
}

// absorbBranchResult merges a THEN or ELSE result expression into the accumulator.
//
// The first concrete (non-Unknown) result type promotes to the CASE result type so
// downstream analysis carries a meaningful type.
//
// Takes expr (*parsedExpression) which is the branch result expression to merge.
func (a *caseExpressionAccumulator) absorbBranchResult(expr *parsedExpression) {
	if expr == nil {
		return
	}
	a.absorbReferences(expr)
	if a.resultType.Category == querier_dto.TypeCategoryUnknown && expr.resultType.Category != querier_dto.TypeCategoryUnknown {
		a.resultType = expr.resultType
	}
}

// finalise produces the case expression from the accumulated state.
//
// Returns *parsedExpression which is the CASE expression carrying the accumulated
// references.
func (a *caseExpressionAccumulator) finalise() *parsedExpression {
	return &parsedExpression{
		kind:              expressionKindCase,
		resultType:        a.resultType,
		referencedColumns: a.columns,
		hasParameter:      a.hasParameter,
		hasAggregate:      a.hasAggregate,
		hasWindow:         a.hasWindow,
	}
}

// parseFunctionCallExpression parses a function call after the function name.
//
// The function name has been consumed and the cursor is on the opening paren. Unlike the
// placeholder dialects (postgres, sqlite, mysql, duckdb), ClickHouse does not need the P1
// function-argument type back-propagation (EnclosingFunctionName plus ArgumentOrdinal on
// the placeholder). A ClickHouse parameter is `{name:Type}`, so the tokeniser already
// needs a non-empty type tag (validateClickHouseParamBody) and
// registerClickHouseParameter records it as the placeholder's CastType. The shared
// resolver types a parameter from its CastType first (type_resolver.go
// resolveParameterType), before it ever reaches the ParameterContextFunctionArgument
// branch, so a placeholder passed as a scalar function argument is already concretely
// typed and never falls to unknown. There is therefore no argument type to recover, and
// this parser deliberately does not tag ParameterContextFunctionArgument or set the
// enclosing function name on the placeholder.
//
// Takes name (string) which is the function name already consumed by the caller.
//
// Returns *parsedExpression which is the function-call expression carrying argument
// references.
func (p *parser) parseFunctionCallExpression(name string) *parsedExpression {
	if p.current().kind != tokenLeftParen {
		return &parsedExpression{kind: expressionKindFunction}
	}
	p.advance()
	state := newInExpressionState(&parsedExpression{})
	for !p.atEnd() && p.current().kind != tokenRightParen {
		state.absorb(p.parseExpression())
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	expr := &parsedExpression{
		kind:              expressionKindFunction,
		referencedColumns: state.columns,
		hasParameter:      state.hasParameter,
		hasAggregate:      isAggregateName(name) || state.hasAggregate,
		hasWindow:         state.hasWindow,
	}

	if p.matchKeyword("OVER") {
		expr.hasWindow = true
		expr.hasAggregate = false
		if p.current().kind == tokenLeftParen {
			_ = p.skipParenthesised()
		}
	}

	savedFilter := p.position
	if p.matchKeyword("FILTER") {
		if p.current().kind == tokenLeftParen {
			_ = p.skipParenthesised()
		} else {
			p.position = savedFilter
		}
	}

	return expr
}

// isOperator reports whether the current token is the supplied operator value.
//
// It does not consume the token.
//
// Takes value (string) which is the operator text to compare against.
//
// Returns bool which is true when the current token is that operator.
func (p *parser) isOperator(value string) bool {
	tok := p.current()
	return tok.kind == tokenOperator && tok.value == value
}

// boolReturnType returns the canonical Bool SQLType used by comparison and logical
// expressions.
//
// Returns querier_dto.SQLType which is the canonical Bool type.
func boolReturnType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"}
}

// mergeBinary constructs a binary expression by merging the flags from both operands.
//
// The parameter, aggregate and window flags are taken from the union of both operands.
//
// Takes left (*parsedExpression) which is the left operand.
// Takes right (*parsedExpression) which is the right operand.
// Takes kind (parsedExpressionKind) which is the kind tag for the result.
// Takes resultType (querier_dto.SQLType) which is the result type to assign.
//
// Returns *parsedExpression which is the merged binary expression.
func mergeBinary(left, right *parsedExpression, kind parsedExpressionKind, resultType querier_dto.SQLType) *parsedExpression {
	return &parsedExpression{
		kind:              kind,
		resultType:        resultType,
		referencedColumns: slices.Concat(left.referencedColumns, right.referencedColumns),
		hasParameter:      left.hasParameter || right.hasParameter,
		hasAggregate:      left.hasAggregate || right.hasAggregate,
		hasWindow:         left.hasWindow || right.hasWindow,
	}
}

// mergeFlags propagates flags from a nested expression to its containing expression.
//
// The parameter, aggregate and window flags propagate without changing the outer result
// type.
//
// Takes outer (*parsedExpression) which is the containing expression.
// Takes inner (*parsedExpression) which is the nested expression whose flags propagate.
//
// Returns *parsedExpression which is the outer expression with the merged flags.
func mergeFlags(outer, inner *parsedExpression) *parsedExpression {
	return &parsedExpression{
		kind:              outer.kind,
		resultType:        outer.resultType,
		referencedColumns: outer.referencedColumns,
		hasParameter:      outer.hasParameter || inner.hasParameter,
		hasAggregate:      outer.hasAggregate || inner.hasAggregate,
		hasWindow:         outer.hasWindow || inner.hasWindow,
	}
}

// elementTypeOrSelf returns the array's element type when present, otherwise the type
// itself.
//
// It is used for `arr[idx]` results.
//
// Takes t (querier_dto.SQLType) which is the type to inspect.
//
// Returns querier_dto.SQLType which is the element type or t when t is not an array.
func elementTypeOrSelf(t querier_dto.SQLType) querier_dto.SQLType {
	if t.Category == querier_dto.TypeCategoryArray && t.ElementType != nil {
		return *t.ElementType
	}
	return t
}

// tupleFieldType returns the type produced by a `base.selector` postfix access.
//
// Three distinct selector families are recognised. Tuple field selection
// (`tuple.fieldName` or `tuple.1`) walks the base's StructFields by name or by 1-based
// index per ClickHouse's tuple convention. Map subcolumn access (`map.keys` or
// `map.values`) produces Array(K) or Array(V) without unfolding the map itself. MergeTree
// array and nullable subcolumn paths (`array.size0` or `nullable.null`) produce the
// implicit sentinel column type ClickHouse exposes alongside the storage column. When the
// selector matches none of these families the result falls back to the base type so the
// caller still receives a valid SQLType.
//
// Takes base (querier_dto.SQLType) which is the type of the expression on the left of the
// dot.
// Takes selector (token) which is the postfix token, an identifier or number.
//
// Returns querier_dto.SQLType which is the resolved field, subcolumn or fallback base
// type.
func tupleFieldType(base querier_dto.SQLType, selector token) querier_dto.SQLType {
	if result, ok := subcolumnAccessType(base, selector); ok {
		return result
	}
	if base.Category != querier_dto.TypeCategoryStruct || len(base.StructFields) == 0 {
		return base
	}
	switch selector.kind {
	case tokenNumber:
		index := parseDecimalInt(selector.value)
		if index >= 1 && index <= len(base.StructFields) {
			return base.StructFields[index-1].SQLType
		}
	case tokenIdentifier:
		for index := range base.StructFields {
			if base.StructFields[index].Name == selector.value {
				return base.StructFields[index].SQLType
			}
		}
	default:
	}
	return base.StructFields[0].SQLType
}

// subcolumnAccessType resolves ClickHouse's implicit subcolumn accessors.
//
// These accessors sit alongside the storage column. `array.size0` resolves to UInt64 (the
// array's length), `nullable.null` resolves to UInt8 (the per-row NULL marker),
// `map.keys` resolves to Array(K) (the map's key list) and `map.values` resolves to
// Array(V) (the map's value list). On a miss the caller can fall through to the tuple
// field path.
//
// Takes base (querier_dto.SQLType) which is the type of the expression on the left of the
// dot.
// Takes selector (token) which is the postfix token, an identifier or number.
//
// Returns querier_dto.SQLType which is the resolved subcolumn type, or the empty type on
// a miss.
// Returns bool which is true when the selector matches a known subcolumn shape.
func subcolumnAccessType(base querier_dto.SQLType, selector token) (querier_dto.SQLType, bool) {
	if selector.kind != tokenIdentifier {
		return querier_dto.SQLType{}, false
	}
	switch strings.ToLower(selector.value) {
	case "size0":
		if base.Category == querier_dto.TypeCategoryArray {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}, true
		}
	case "null":
		if base.Nullable {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt8"}, true
		}
	case "keys":
		if base.Category == querier_dto.TypeCategoryMap && base.KeyType != nil {
			return querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  "Array",
				ElementType: new(*base.KeyType),
			}, true
		}
	case "values":
		if base.Category == querier_dto.TypeCategoryMap && base.ElementType != nil {
			return querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  "Array",
				ElementType: new(*base.ElementType),
			}, true
		}
	}
	return querier_dto.SQLType{}, false
}

// parseDecimalInt parses a decimal-integer literal into an int.
//
// Callers validate the kind beforehand so this only sees positive integers in practice.
// The magnitude is bounded by maxTypeModifierValue so an overlong run of digits cannot
// overflow the accumulator, and once the bound is reached the cap is returned, which
// fails every real index check.
//
// Takes literal (string) which is the decimal-integer text to parse.
//
// Returns int which is the parsed value, zero when malformed, or the cap when the bound
// is hit.
func parseDecimalInt(literal string) int {
	value := 0
	for index := range len(literal) {
		ch := literal[index]
		if ch < '0' || ch > '9' {
			return value
		}
		digit := int(ch - '0')
		if value > (maxTypeModifierValue-digit)/decimalRadix {
			return maxTypeModifierValue
		}
		value = value*decimalRadix + digit
	}
	return value
}

// literalNumberType classifies a numeric literal as integer or float.
//
// The classification depends on whether the literal contains a decimal point or exponent.
// Hex, binary and octal prefixed literals (0x, 0b, 0o) are always integers and are
// short-circuited first, because their digits may include e or E (a hex digit), which the
// decimal-point and exponent scan would otherwise misread as scientific notation. A
// genuine hex floating literal uses p, not e, for its exponent, so prefixed literals
// never classify as float.
//
// Takes value (string) which is the numeric literal text to classify.
//
// Returns querier_dto.SQLType which is the Int64 or Float64 type for the literal.
func literalNumberType(value string) querier_dto.SQLType {
	if hasIntegerRadixPrefix(value) {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
	}
	for index := range len(value) {
		if value[index] == '.' || value[index] == 'e' || value[index] == 'E' {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"}
		}
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
}

// hasIntegerRadixPrefix reports whether a numeric literal carries an integer radix
// prefix.
//
// The recognised prefixes are hex (0x), binary (0b) and octal (0o), allowing an optional
// leading sign. Such literals are always integers.
//
// Takes value (string) which is the numeric literal text to inspect.
//
// Returns bool which is true when value carries a hex, binary or octal radix prefix.
func hasIntegerRadixPrefix(value string) bool {
	body := value
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		body = body[1:]
	}
	if len(body) < 2 || body[0] != '0' {
		return false
	}
	switch body[1] {
	case 'x', 'X', 'b', 'B', 'o', 'O':
		return true
	default:
		return false
	}
}

// typeFromParamBody parses the `name:Type` body of a `{name:Type}` placeholder.
//
// It returns the empty SQLType when the body has no `:Type` portion or when the type
// fails to parse.
//
// Takes body (string) which is the placeholder body text between the braces.
//
// Returns querier_dto.SQLType which is the parsed type, or the empty type when absent.
func typeFromParamBody(body string) querier_dto.SQLType {
	_, typeSegment, found := strings.Cut(body, ":")
	if !found {
		return querier_dto.SQLType{}
	}
	result, err := parseClickHouseType(strings.TrimSpace(typeSegment))
	if err != nil {
		return querier_dto.SQLType{}
	}
	return result.SQLType
}
