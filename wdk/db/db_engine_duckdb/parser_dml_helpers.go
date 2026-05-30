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

package db_engine_duckdb

import (
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// handleParameterInExpression registers the current parameter token, inferring its
// context, target column, and inline cast type.
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

// resolveParameterContext picks the best context, target column, and cast for the
// parameter at paramPosition.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred target column, or nil when
// none is known.
// Returns *querier_dto.SQLType which is the inferred cast type, or nil when no inline
// cast applies.
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

// resolveLikeContext infers the LIKE-style context for a parameter.
//
// Walks back from paramPosition through balanced parens, stopping at boolean or clause
// boundaries, to find an enclosing LIKE, ILIKE, or GLOB pattern operator and returns its
// LHS column reference, falling through to a wider scan of the LHS expression when the
// immediate-left token is not itself a column. DuckDB's regex operators (the ~ family)
// are handled via the comparison-operator path; SIMILAR TO (two tokens) is not detected
// here.
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
	parenDepth := 0
	for i := paramPosition - 1; i >= 0; i-- {
		tok := p.tokens[i]
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenRightParen:
			parenDepth++
			continue
		case tokenLeftParen:
			parenDepth--
			continue
		}
		if parenDepth > 0 || tok.kind != tokenIdentifier {
			continue
		}
		keyword := strings.ToUpper(tok.value)
		if isLikeBoundaryKeyword(keyword) {
			return 0, false
		}
		if isLikePatternKeyword(keyword) {
			return i, true
		}
	}
	return 0, false
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
	case "AS", "CAST", "TRY_CAST", "COLLATE", "DISTINCT", "ALL", "NULL", "TRUE", "FALSE",
		"INTEGER", "INT", "TINYINT", "SMALLINT", "BIGINT", "HUGEINT",
		"REAL", "FLOAT", "DOUBLE", "NUMERIC", "DECIMAL",
		"VARCHAR", "CHAR", "TEXT", "STRING", "BLOB",
		"BOOLEAN", "BOOL", "DATE", "TIME", "TIMESTAMP", "TIMESTAMPTZ",
		"INTERVAL", "JSON", "UUID",
		"ESCAPE", "BETWEEN", "IS", "IN", "EXISTS", "NOT", "ASC", "DESC":
		return true
	}
	return false
}

// isLikePatternKeyword reports whether a keyword introduces a DuckDB string pattern match
// (LIKE, ILIKE, or GLOB). DuckDB has no RLIKE or MATCH operator; SIMILAR TO is two tokens
// and is not detected here, and regex matching uses the ~ family of operators handled
// elsewhere.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true for LIKE, ILIKE, or GLOB.
func isLikePatternKeyword(keyword string) bool {
	switch keyword {
	case "LIKE", "ILIKE", "GLOB":
		return true
	}
	return false
}

// isLikeBoundaryKeyword reports whether a keyword ends the predicate that could contain a
// LIKE pattern, so the backward walk should stop.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true at any clause or boolean boundary.
func isLikeBoundaryKeyword(keyword string) bool {
	switch keyword {
	case "AND", "OR", "WHERE", "HAVING", "ON", "WHEN", "THEN", "ELSE", "CASE",
		"FROM", "GROUP", "ORDER", "LIMIT", "OFFSET", "RETURNING",
		"UNION", "INTERSECT", "EXCEPT", "BY",
		"SELECT", "INSERT", "UPDATE", "DELETE", "VALUES", "SET",
		"ESCAPE":
		return true
	}
	return false
}

// resolveContextFromPrecedingOperator infers a parameter's context from a comparison or
// arithmetic operator that precedes it.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context or
// ParameterContextUnknown when no operator pattern matches.
// Returns *querier_dto.ColumnReference which is the LHS column when identified, else nil.
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

// extractColumnReferenceOrParenthesised resolves a column reference from position,
// peeking inside a balanced parenthesised group when position is a closing parenthesis.
//
// Takes position (int) which is the candidate column position.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when none
// could be identified.
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

// consumeInlineCast reads a "::type" cast following a parameter and returns the
// normalised type.
//
// Returns *querier_dto.SQLType which is the normalised cast type.
func (p *parser) consumeInlineCast() *querier_dto.SQLType {
	p.advance()
	typeName := p.parseCastTypeName()
	return new(normaliseTypeName(typeName, nil))
}

// detectParameterContext infers a parameter's context from the keyword or call that
// encloses it.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context or
// ParameterContextUnknown.
// Returns *querier_dto.ColumnReference which is the target column for IN-list patterns,
// else nil.
// Returns *querier_dto.SQLType which is the cast type for CAST(...) usage, else nil.
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

// findEnclosingParen walks back from position looking for the opening parenthesis that
// encloses it at the same nesting depth.
//
// Takes position (int) which is the search starting index.
//
// Returns int which is the index of the enclosing '(' or -1 when none is found.
func (p *parser) findEnclosingParen(position int) int {
	depth := 0
	for i := position - 1; i >= 0; i-- {
		switch p.tokens[i].kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenRightParen:
			depth++
		case tokenLeftParen:
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// extractColumnReferenceBeforeIN reads the column reference that precedes an IN keyword.
//
// Takes inPosition (int) which is the IN keyword's token index.
//
// Returns *querier_dto.ColumnReference which is the LHS column, or nil when inPosition
// leaves no room for one.
func (p *parser) extractColumnReferenceBeforeIN(inPosition int) *querier_dto.ColumnReference {
	if inPosition < 1 {
		return nil
	}
	return p.extractColumnReference(inPosition - 1)
}

// extractCastType finds the AS keyword after a CAST(...) parameter and returns the
// normalised target type.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns *querier_dto.SQLType which is the cast type, or nil when no type follows AS.
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

// findASKeywordAfter searches forward from paramPosition for an AS keyword, stopping at
// the next closing parenthesis.
//
// Takes paramPosition (int) which is the search starting index.
//
// Returns int which is the AS keyword's token index, or -1 when none is found before the
// enclosing ')'.
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

// collectCastTypeTokens joins consecutive identifier tokens of a CAST target type into a
// single space-separated string.
//
// Takes startPosition (int) which is the index of the first identifier of the type name.
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

// isKeywordAt reports whether the token at position is an identifier matching any of the
// given keywords.
//
// Takes position (int) which is the token index to test.
// Takes keywords (...string) which is the upper-case keyword set to match against.
//
// Returns bool which is true when the token matches one of keywords.
func (p *parser) isKeywordAt(position int, keywords ...string) bool {
	if position >= len(p.tokens) || p.tokens[position].kind != tokenIdentifier {
		return false
	}
	return slices.Contains(keywords, strings.ToUpper(p.tokens[position].value))
}

// extractColumnReference reads a bare or schema-qualified column reference at position.
//
// Takes position (int) which is the identifier's token index.
//
// Returns *querier_dto.ColumnReference which is the parsed reference, or nil when
// position is out of range or not an identifier.
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

// isComparisonOperator reports whether operator is a SQL comparison operator spelling.
//
// Takes operator (string) which is the operator literal.
//
// Returns bool which is true for =, <>, !=, <, >, <=, or >=.
func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// isArithmeticOperator reports whether operator is a SQL arithmetic operator spelling
// other than multiplication.
//
// Takes operator (string) which is the operator literal.
//
// Returns bool which is true for +, -, /, or %.
func isArithmeticOperator(operator string) bool {
	switch operator {
	case "+", "-", "/", "%":
		return true
	}
	return false
}

// extractColumnReferenceFromParenthesised resolves a column reference from within a
// parenthesised group ending at rightParenPosition.
//
// Takes rightParenPosition (int) which is the closing-parenthesis token index.
//
// Returns *querier_dto.ColumnReference which is the first column found within the group,
// or nil when none can be identified.
func (p *parser) extractColumnReferenceFromParenthesised(rightParenPosition int) *querier_dto.ColumnReference {
	leftParenPosition := p.findMatchingLeftParen(rightParenPosition)
	if leftParenPosition < 0 {
		return nil
	}
	return p.scanForColumnReference(leftParenPosition+1, rightParenPosition)
}

// findMatchingLeftParen walks back from rightParenPosition to find the matching opening
// parenthesis.
//
// Takes rightParenPosition (int) which is the closing-parenthesis token index.
//
// Returns int which is the matching '(' index, or -1 when nothing matches.
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

// scanForColumnReference scans tokens in [startPosition, endPosition) for the first
// plausible column reference.
//
// Takes startPosition (int) which is the inclusive start index.
// Takes endPosition (int) which is the exclusive end index.
//
// Returns *querier_dto.ColumnReference which is the first column found, or nil when none
// is identified.
func (p *parser) scanForColumnReference(startPosition int, endPosition int) *querier_dto.ColumnReference {
	for j := startPosition; j < endPosition; j++ {
		reference := p.extractColumnReference(j)
		if reference != nil {
			return reference
		}
	}
	return nil
}

// parseCastTypeName consumes a target type name following an inline cast, including
// schema qualifier, multi-word built-ins, parenthesised modifiers, and trailing array
// brackets.
//
// Returns string which is the canonical type spelling, empty when the current token is
// not an identifier.
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

// appendSchemaQualifier appends a ".name" suffix to builder when a schema-qualified
// continuation follows the current token.
//
// Takes builder (*strings.Builder) which receives the appended text.
func (p *parser) appendSchemaQualifier(builder *strings.Builder) {
	if p.current().kind == tokenDot && p.peek().kind == tokenIdentifier {
		p.advance()
		builder.WriteByte('.')
		builder.WriteString(p.advance().value)
	}
}

// appendMultiWordTypeKeywords appends each consecutive multi-word type keyword to
// builder, separated by spaces.
//
// Takes builder (*strings.Builder) which receives the appended text.
func (p *parser) appendMultiWordTypeKeywords(builder *strings.Builder) {
	for p.current().kind == tokenIdentifier {
		if !isMultiWordTypeKeyword(strings.ToUpper(p.current().value)) {
			break
		}
		builder.WriteByte(' ')
		builder.WriteString(p.advance().value)
	}
}

// appendTypeArrayBrackets appends one array subscript suffix to builder per consumed []
// or [N] pair.
//
// Takes builder (*strings.Builder) which receives the appended text.
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

// isMultiWordTypeKeyword reports whether upper is one of the keyword continuations
// recognised in multi-word built-in type names.
//
// Takes upper (string) which is the candidate keyword upper-cased.
//
// Returns bool which is true for keywords such as VARYING, PRECISION, or ZONE.
func isMultiWordTypeKeyword(upper string) bool {
	switch upper {
	case "VARYING", "PRECISION", "WITHOUT", keywordWITH, keywordTIME, keywordZONE,
		"CHARACTER", "DOUBLE", "TIMESTAMP", "INTERVAL":
		return true
	}
	return false
}
