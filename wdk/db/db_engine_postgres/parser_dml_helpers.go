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
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// handleParameterInExpression registers the current parameter token, resolving its
// context from preceding tokens and consuming any inline cast suffix.
func (p *parser) handleParameterInExpression() {
	paramPosition := p.position
	parameterToken := p.current()

	context, columnRef, castType := p.resolveParameterContext(paramPosition)

	p.advance()

	if p.current().kind == tokenCast {
		castType = p.consumeInlineCast()
		if context == querier_dto.ParameterContextUnknown {
			context = querier_dto.ParameterContextCast
		}
	}

	p.registerParameterFromToken(parameterToken, context, columnRef, castType)
}

// resolveParameterContext resolves the syntactic context of a parameter at paramPosition
// by consulting preceding operators, LIKE scope, and enclosing parens.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred column, or nil when none was
// found.
// Returns *querier_dto.SQLType which is the inferred cast type, or nil when none was
// found.
func (p *parser) resolveParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType) {
	context, columnRef := p.resolveContextFromPrecedingOperator(paramPosition)
	if context != querier_dto.ParameterContextUnknown {
		return context, columnRef, nil
	}
	if likeContext, likeColumn := p.resolveLikeContext(paramPosition); likeContext != querier_dto.ParameterContextUnknown {
		return likeContext, likeColumn, nil
	}
	return p.detectParameterContext(paramPosition)
}

// resolveLikeContext resolves a parameter sitting inside a LIKE or ILIKE pattern,
// returning its LHS column reference.
//
// Walks back from a parameter position through balanced parens (stopping at boolean or
// clause boundaries) to find an enclosing LIKE or ILIKE operator and returns its LHS
// column reference, falling through to a wider scan of the LHS expression when the
// immediate-left token is not itself a column. Postgres' regex-match operators (~, ~*,
// !~, !~*) are handled via the comparison-operator path; SIMILAR TO (two tokens) is not
// detected here.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is ParameterContextLike when the parameter
// sits inside a pattern operator's right-hand side, else ParameterContextUnknown.
// Returns *querier_dto.ColumnReference which holds the LHS column when one can be
// identified, else nil.
func (p *parser) resolveLikeContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	likePosition, found := p.findEnclosingLikeOperator(paramPosition)
	if !found {
		return querier_dto.ParameterContextUnknown, nil
	}
	return querier_dto.ParameterContextLike, p.resolveLikeOperatorColumn(likePosition)
}

// findEnclosingLikeOperator walks back from paramPosition through balanced parens looking
// for a pattern operator at depth 0, returning its token index when found. Boolean and
// clause keywords end the walk.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns int which is the LIKE-style operator's token index when found.
// Returns bool which is true when an operator was located.
func (p *parser) findEnclosingLikeOperator(paramPosition int) (int, bool) {
	return engine_shared.FindEnclosingLikeOperator(paramPosition,
		func(index int) bool { return p.tokens[index].kind == tokenLeftParen },
		func(index int) bool { return p.tokens[index].kind == tokenRightParen },
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(p.tokens[index].value))
		},
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikePatternKeyword(strings.ToUpper(p.tokens[index].value))
		},
	)
}

// resolveLikeOperatorColumn picks the column reference associated with a LIKE operator's
// left-hand side, first inspecting the immediately preceding token (skipping a single
// NOT) and falling back to a scan of the wider LHS expression for shapes like `(a || b)
// LIKE ?`.
//
// Takes likePosition (int) which is the LIKE-style operator's token index.
//
// Returns *querier_dto.ColumnReference which is the inferred LHS column or nil when none
// can be identified.
func (p *parser) resolveLikeOperatorColumn(likePosition int) *querier_dto.ColumnReference {
	columnPosition := likePosition - 1
	if columnPosition >= 0 && p.tokens[columnPosition].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[columnPosition].value, "NOT") {
		columnPosition--
	}
	if columnRef := p.extractColumnReference(columnPosition); columnRef != nil && !isKeywordColumnReference(columnRef) {
		return columnRef
	}
	return p.findColumnInExpressionRange(p.findLikeExpressionStart(likePosition), likePosition-1)
}

// isKeywordColumnReference reports whether a column reference's bare column name is
// actually a SQL keyword (LIKE, AND, etc.) rather than a column. Edge case for malformed
// input like "LIKE ?" at position 0 where the backward walk would otherwise treat the
// keyword itself as the LHS column.
//
// Takes columnRef (*querier_dto.ColumnReference) which is the candidate column reference.
//
// Returns bool which is true when the bare column name is a keyword.
func isKeywordColumnReference(columnRef *querier_dto.ColumnReference) bool {
	if columnRef == nil || columnRef.TableAlias != "" {
		return false
	}
	upper := strings.ToUpper(columnRef.ColumnName)
	return isLikePatternKeyword(upper) || isLikeBoundaryKeyword(upper) || isReservedNonColumnKeyword(upper)
}

// findLikeExpressionStart locates the first token of the LHS expression preceding a
// LIKE-style operator by walking back to the previous predicate boundary or the start of
// the token stream.
//
// Takes likePosition (int) which is the LIKE keyword's token index.
//
// Returns int which is the token index where the LHS starts (0 when no boundary precedes
// it).
func (p *parser) findLikeExpressionStart(likePosition int) int {
	parenDepth := 0
	for i := likePosition - 1; i >= 0; i-- {
		tok := p.tokens[i]
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenRightParen:
			parenDepth++
			continue
		case tokenLeftParen:
			if parenDepth > 0 {
				parenDepth--
				continue
			}
			return i + 1
		}
		if parenDepth > 0 {
			continue
		}
		if tok.kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(tok.value)) {
			return i + 1
		}
	}
	return 0
}

// findColumnInExpressionRange scans [start, end] forward for the first identifier that
// looks like a column, skipping function names, SQL types, aliases, and reserved keywords
// so complex LHS expressions still resolve to a meaningful column.
//
// Takes start (int) which is the inclusive start of the range.
// Takes end (int) which is the inclusive end of the range.
//
// Returns *querier_dto.ColumnReference which is the first column found, or nil when none
// is plausible.
func (p *parser) findColumnInExpressionRange(start, end int) *querier_dto.ColumnReference {
	if end < start || end >= len(p.tokens) {
		return nil
	}
	for i := start; i <= end; i++ {
		tok := p.tokens[i]
		if tok.kind != tokenIdentifier {
			continue
		}
		if isLikelyColumnIdentifier(p.tokens, i) {
			return p.extractColumnReference(i)
		}
		if isQualifiedColumnTail(p.tokens, i) {
			return p.extractColumnReference(i)
		}
	}
	return nil
}

// isLikelyColumnIdentifier checks whether the identifier at position looks like a column
// rather than a function, type, alias, or reserved keyword by inspecting neighbouring
// tokens.
//
// Takes tokens ([]token) which is the parser's token slice.
// Takes position (int) which is the candidate identifier's index.
//
// Returns bool which is true when the identifier looks like a column.
func isLikelyColumnIdentifier(tokens []token, position int) bool {
	if position < 0 || position >= len(tokens) {
		return false
	}
	tok := tokens[position]
	if tok.kind != tokenIdentifier {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenLeftParen {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenDot {
		return false
	}
	if position > 0 && tokens[position-1].kind == tokenIdentifier &&
		strings.EqualFold(tokens[position-1].value, "AS") {
		return false
	}
	upper := strings.ToUpper(tok.value)
	if isLikeBoundaryKeyword(upper) || isLikePatternKeyword(upper) {
		return false
	}
	if isReservedNonColumnKeyword(upper) {
		return false
	}
	return true
}

// isQualifiedColumnTail reports whether the identifier at position is the column half of
// a qualified `<alias>.<column>` reference, so the leftmost-column scan does not skip it.
//
// Takes tokens ([]token) which is the parser's token slice.
// Takes position (int) which is the candidate identifier's index.
//
// Returns bool which is true for the column half of `<alias>.<column>`.
func isQualifiedColumnTail(tokens []token, position int) bool {
	if position < 2 {
		return false
	}
	if tokens[position-1].kind != tokenDot {
		return false
	}
	if tokens[position-2].kind != tokenIdentifier {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenLeftParen {
		return false
	}
	return true
}

// isReservedNonColumnKeyword lists identifier values that must not be treated as column
// references when scanning a LIKE LHS: casts, type names, literals, and modifier keywords
// that lex as identifiers.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true when the value is never a column.
func isReservedNonColumnKeyword(keyword string) bool {
	switch keyword {
	case "AS", "CAST", "COLLATE", "DISTINCT", "ALL", "NULL", "TRUE", "FALSE",
		"INTEGER", "INT", "INT2", "INT4", "INT8", "BIGINT", "SMALLINT",
		"REAL", "FLOAT", "DOUBLE", "NUMERIC", "DECIMAL",
		"TEXT", "VARCHAR", "CHAR", "CHARACTER", "BYTEA",
		"BOOLEAN", "BOOL", "DATE", "TIME", "TIMESTAMP", "TIMESTAMPTZ",
		"INTERVAL", "JSON", "JSONB", "UUID",
		"ESCAPE", "BETWEEN", "IS", "IN", "EXISTS", "NOT", "ASC", "DESC":
		return true
	}
	return false
}

// isLikePatternKeyword reports whether a keyword introduces a Postgres string pattern
// match (LIKE or ILIKE). Postgres has no GLOB, RLIKE, or MATCH operator; its regex
// operators (~, ~*, !~, !~*) are tokenised as operators and handled elsewhere.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true for LIKE or ILIKE.
func isLikePatternKeyword(keyword string) bool {
	switch keyword {
	case "LIKE", "ILIKE":
		return true
	}
	return false
}

// isLikeBoundaryKeyword reports whether a keyword ends the predicate that could contain a
// LIKE pattern, so the backward walk should stop. It defers to the shared clause-boundary
// keyword set so every engine recognises the same boundaries.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true at any clause or boolean boundary.
func isLikeBoundaryKeyword(keyword string) bool {
	return engine_shared.IsClauseBoundaryKeyword(keyword)
}

// resolveContextFromPrecedingOperator inspects the operator immediately before
// paramPosition and infers a parameter context from a comparison or arithmetic operator.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context, or
// ParameterContextUnknown when nothing matched.
// Returns *querier_dto.ColumnReference which is the column appearing before the operator,
// or nil when none was found.
func (p *parser) resolveContextFromPrecedingOperator(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	if paramPosition < 2 {
		return querier_dto.ParameterContextUnknown, nil
	}

	prevToken := p.tokens[paramPosition-1]
	beforeOp := paramPosition - 2

	if prevToken.kind == tokenOperator && isComparisonOperator(prevToken.value) {
		columnRef := p.extractColumnReferenceOrParenthesised(beforeOp)
		if columnRef != nil {
			return querier_dto.ParameterContextComparison, columnRef
		}
	}

	if prevToken.kind == tokenStar || (prevToken.kind == tokenOperator && isArithmeticOperator(prevToken.value)) {
		columnRef := p.extractColumnReference(beforeOp)
		if columnRef != nil {
			return querier_dto.ParameterContextComparison, columnRef
		}
	}

	return querier_dto.ParameterContextUnknown, nil
}

// extractColumnReferenceOrParenthesised extracts a column reference at position, falling
// back to scanning the enclosing parenthesised expression when position points at a `)`.
//
// Takes position (int) which is the token index to inspect.
//
// Returns *querier_dto.ColumnReference which is the resolved column, or nil when none was
// found.
func (p *parser) extractColumnReferenceOrParenthesised(position int) *querier_dto.ColumnReference {
	columnRef := p.extractColumnReference(position)
	if columnRef != nil {
		return columnRef
	}
	if p.tokens[position].kind == tokenRightParen {
		return p.extractColumnReferenceFromParenthesised(position)
	}
	return nil
}

// consumeInlineCast consumes a `::type` cast suffix and returns the normalised SQL type.
//
// Returns *querier_dto.SQLType which is the normalised cast type.
func (p *parser) consumeInlineCast() *querier_dto.SQLType {
	p.advance()
	typeName := p.parseCastTypeName()
	return new(normaliseTypeName(typeName, nil))
}

// detectParameterContext infers a parameter context by inspecting the keyword that
// introduces the enclosing parenthesised group, such as IN, CAST, or a function name.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred column, or nil when none was
// found.
// Returns *querier_dto.SQLType which is the inferred cast type, or nil when none applies.
func (p *parser) detectParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType) {
	enclosingParen := p.findEnclosingParen(paramPosition)
	if enclosingParen < 0 {
		return querier_dto.ParameterContextUnknown, nil, nil
	}

	if enclosingParen >= 2 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, "IN") {
		columnRef := p.extractColumnReferenceBeforeIN(enclosingParen - 1)
		return querier_dto.ParameterContextInList, columnRef, nil
	}

	if enclosingParen >= 1 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, "CAST") {
		castType := p.extractCastType(paramPosition)
		if castType != nil {
			return querier_dto.ParameterContextCast, nil, castType
		}
	}

	if enclosingParen >= 1 && p.tokens[enclosingParen-1].kind == tokenIdentifier {
		functionName := strings.ToUpper(p.tokens[enclosingParen-1].value)
		if functionName != "IN" && functionName != "CAST" &&
			functionName != keywordSELECT && functionName != keywordWHERE {
			return querier_dto.ParameterContextFunctionArgument, nil, nil
		}
	}

	return querier_dto.ParameterContextUnknown, nil, nil
}

// findEnclosingParen finds the index of the `(` that opens the parenthesised group
// enclosing position.
//
// Takes position (int) which is the token index to inspect.
//
// Returns int which is the enclosing `(` index, or -1 when position is not inside any
// group.
func (p *parser) findEnclosingParen(position int) int {
	return engine_shared.FindEnclosingParen(position,
		func(index int) bool { return p.tokens[index].kind == tokenLeftParen },
		func(index int) bool { return p.tokens[index].kind == tokenRightParen },
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(p.tokens[index].value))
		},
	)
}

// extractColumnReferenceBeforeIN extracts the column reference one token before an IN
// keyword, which is the LHS of the IN predicate.
//
// Takes inPosition (int) which is the IN keyword's token index.
//
// Returns *querier_dto.ColumnReference which is the LHS column, or nil when none was
// found.
func (p *parser) extractColumnReferenceBeforeIN(inPosition int) *querier_dto.ColumnReference {
	if inPosition < 1 {
		return nil
	}
	return p.extractColumnReference(inPosition - 1)
}

// extractCastType locates an AS keyword following a CAST parameter and returns the
// normalised target type.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns *querier_dto.SQLType which is the cast target type, or nil when no AS or type
// tokens were found.
func (p *parser) extractCastType(paramPosition int) *querier_dto.SQLType {
	asPosition := p.findASKeywordAfter(paramPosition)
	if asPosition < 0 {
		return nil
	}
	typeNameStart := asPosition + 1
	if typeNameStart >= len(p.tokens) || p.tokens[typeNameStart].kind != tokenIdentifier {
		return nil
	}
	typeName := p.collectCastTypeTokens(typeNameStart)
	return new(normaliseTypeName(typeName, nil))
}

// findASKeywordAfter locates the index of an AS keyword that follows paramPosition
// without crossing a `)`.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns int which is the AS token index, or -1 when none was found.
func (p *parser) findASKeywordAfter(paramPosition int) int {
	for i := paramPosition + 1; i < len(p.tokens); i++ {
		if p.tokens[i].kind == tokenIdentifier && strings.EqualFold(p.tokens[i].value, keywordAS) {
			return i
		}
		if p.tokens[i].kind == tokenRightParen {
			break
		}
	}
	return -1
}

// collectCastTypeTokens joins identifier tokens starting at startPosition into a
// multi-word type name, stopping at a closing paren or major clause keyword.
//
// Takes startPosition (int) which is the first type token's index.
//
// Returns string which is the joined type name.
func (p *parser) collectCastTypeTokens(startPosition int) string {
	var builder strings.Builder
	builder.WriteString(p.tokens[startPosition].value)
	for j := startPosition + 1; j < len(p.tokens); j++ {
		if p.tokens[j].kind != tokenIdentifier ||
			p.isKeywordAt(j, keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT) {
			break
		}
		if p.tokens[j].kind == tokenRightParen {
			break
		}
		builder.WriteByte(' ')
		builder.WriteString(p.tokens[j].value)
	}
	return builder.String()
}

// isKeywordAt reports whether the token at position is an identifier whose upper-case
// value matches one of keywords.
//
// Takes position (int) which is the token index to inspect.
// Takes keywords (...string) which is the list of keywords to test.
//
// Returns bool which is true when the token matches one of them.
func (p *parser) isKeywordAt(position int, keywords ...string) bool {
	if position >= len(p.tokens) || p.tokens[position].kind != tokenIdentifier {
		return false
	}
	return slices.Contains(keywords, strings.ToUpper(p.tokens[position].value))
}

// extractColumnReference reads a bare or `alias.column` column reference rooted at
// position.
//
// Takes position (int) which is the column identifier's token index.
//
// Returns *querier_dto.ColumnReference which is the parsed reference, or nil when
// position does not point at an identifier.
func (p *parser) extractColumnReference(position int) *querier_dto.ColumnReference {
	if position < 0 || position >= len(p.tokens) {
		return nil
	}

	tok := p.tokens[position]
	if tok.kind != tokenIdentifier {
		return nil
	}

	if position >= 2 && p.tokens[position-1].kind == tokenDot && p.tokens[position-2].kind == tokenIdentifier {
		return &querier_dto.ColumnReference{
			TableAlias: p.tokens[position-2].value,
			ColumnName: tok.value,
		}
	}

	return &querier_dto.ColumnReference{
		ColumnName: tok.value,
	}
}

// isComparisonOperator reports whether operator is one of the SQL comparison operators.
//
// Takes operator (string) which is the operator value to test.
//
// Returns bool which is true for `=`, `<>`, `!=`, `<`, `>`, `<=`, or `>=`.
func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// isArithmeticOperator reports whether operator is a basic arithmetic operator.
//
// Takes operator (string) which is the operator value to test.
//
// Returns bool which is true for `+`, `-`, `/`, or `%`.
func isArithmeticOperator(operator string) bool {
	switch operator {
	case "+", "-", "/", "%":
		return true
	}
	return false
}

// extractColumnReferenceFromParenthesised resolves a column reference inside the
// parenthesised group whose closing token is at rightParenPosition.
//
// Takes rightParenPosition (int) which is the `)` token index.
//
// Returns *querier_dto.ColumnReference which is the resolved column, or nil when none was
// found.
func (p *parser) extractColumnReferenceFromParenthesised(rightParenPosition int) *querier_dto.ColumnReference {
	leftParenPosition := p.findMatchingLeftParen(rightParenPosition)
	if leftParenPosition < 0 {
		return nil
	}
	return p.scanForColumnReference(leftParenPosition+1, rightParenPosition)
}

// findMatchingLeftParen locates the matching `(` for the `)` at rightParenPosition.
//
// Takes rightParenPosition (int) which is the `)` token index.
//
// Returns int which is the matching `(` index, or -1 when unmatched.
func (p *parser) findMatchingLeftParen(rightParenPosition int) int {
	depth := 0
	for i := rightParenPosition; i >= 0; i-- {
		switch p.tokens[i].kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenRightParen:
			depth++
		case tokenLeftParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// scanForColumnReference scans forward from startPosition for the first column reference
// before endPosition.
//
// Takes startPosition (int) which is the inclusive start of the scan.
// Takes endPosition (int) which is the exclusive end of the scan.
//
// Returns *querier_dto.ColumnReference which is the first column reference, or nil when
// none was found.
func (p *parser) scanForColumnReference(startPosition int, endPosition int) *querier_dto.ColumnReference {
	for j := startPosition; j < endPosition; j++ {
		reference := p.extractColumnReference(j)
		if reference != nil {
			return reference
		}
	}
	return nil
}

// parseCastTypeName parses a `::type` cast target consisting of an identifier, an
// optional schema qualifier, multi-word type keywords, any size parenthesisation, and
// array brackets.
//
// Returns string which is the joined type name.
func (p *parser) parseCastTypeName() string {
	if p.current().kind != tokenIdentifier {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(p.advance().value)

	p.appendSchemaQualifier(&builder)
	p.appendMultiWordTypeKeywords(&builder)

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	p.appendTypeArrayBrackets(&builder)

	return builder.String()
}

// appendSchemaQualifier consumes a `.identifier` schema qualifier after a type name and
// appends it to builder.
//
// Takes builder (*strings.Builder) which is the type-name accumulator.
func (p *parser) appendSchemaQualifier(builder *strings.Builder) {
	if p.current().kind == tokenDot && p.peek().kind == tokenIdentifier {
		p.advance()
		builder.WriteByte('.')
		builder.WriteString(p.advance().value)
	}
}

// appendMultiWordTypeKeywords consumes trailing identifiers that form a multi-word type
// name (e.g. `DOUBLE PRECISION`) and appends them to builder.
//
// Takes builder (*strings.Builder) which is the type-name accumulator.
func (p *parser) appendMultiWordTypeKeywords(builder *strings.Builder) {
	for p.current().kind == tokenIdentifier {
		if !isMultiWordTypeKeyword(strings.ToUpper(p.current().value)) {
			break
		}
		builder.WriteByte(' ')
		builder.WriteString(p.advance().value)
	}
}

// appendTypeArrayBrackets consumes any `[N]` or `[]` array suffixes following a type name
// and appends the array marker to builder.
//
// Takes builder (*strings.Builder) which is the type-name accumulator.
func (p *parser) appendTypeArrayBrackets(builder *strings.Builder) {
	for p.current().kind == tokenLeftBracket {
		p.advance()
		if p.current().kind == tokenNumber {
			p.advance()
		}
		if p.current().kind == tokenRightBracket {
			builder.WriteString(arraySubscriptSuffix)
			p.advance()
		}
	}
}

// isMultiWordTypeKeyword reports whether upper is the second or later word in a
// multi-word SQL type name.
//
// Takes upper (string) which is the upper-case identifier value.
//
// Returns bool which is true when the value belongs to a multi-word type form.
func isMultiWordTypeKeyword(upper string) bool {
	switch upper {
	case "VARYING", "PRECISION", "WITHOUT", keywordWITH, keywordTIME, keywordZONE,
		"CHARACTER", "DOUBLE", "TIMESTAMP", "INTERVAL":
		return true
	}
	return false
}
