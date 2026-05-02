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

package db_engine_mysql

import (
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// parseExpression parses a top-level SQL expression.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseExpression() querier_dto.Expression {
	if p.expressionDepth >= p.maxParseDepth {
		return &querier_dto.UnknownExpression{}
	}
	p.expressionDepth++
	defer func() { p.expressionDepth-- }()
	return p.parseOrExpression()
}

// parseOrExpression parses left-associative OR or `||` chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseOrExpression() querier_dto.Expression {
	left := p.parseAndExpression()
	for p.matchKeyword("OR") || (p.current().kind == tokenOperator && p.current().value == "||") {
		if p.current().kind == tokenOperator && p.current().value == "||" {
			p.advance()
		}
		right := p.parseAndExpression()
		left = &querier_dto.LogicalOpExpression{
			Operator: "OR",
			Operands: []querier_dto.Expression{left, right},
		}
	}
	return left
}

// parseAndExpression parses left-associative AND chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseAndExpression() querier_dto.Expression {
	left := p.parseNotExpression()
	for p.matchKeyword(keywordAND) {
		right := p.parseNotExpression()
		left = &querier_dto.LogicalOpExpression{
			Operator: "AND",
			Operands: []querier_dto.Expression{left, right},
		}
	}
	return left
}

// parseNotExpression parses a unary NOT or `!` prefix.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseNotExpression() querier_dto.Expression {
	if p.matchKeyword(keywordNOT) {
		operand := p.parseComparisonExpression()
		return &querier_dto.UnaryOpExpression{
			Operator: keywordNOT,
			Operand:  operand,
		}
	}
	if p.current().kind == tokenOperator && p.current().value == "!" {
		p.advance()
		operand := p.parseComparisonExpression()
		return &querier_dto.UnaryOpExpression{
			Operator: keywordNOT,
			Operand:  operand,
		}
	}
	return p.parseComparisonExpression()
}

// parseComparisonExpression parses comparisons and postfix predicates.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseComparisonExpression() querier_dto.Expression {
	left := p.parseBitwiseExpression()

	if p.current().kind == tokenOperator && isComparisonOperator(p.current().value) {
		return p.parseComparisonOperator(left)
	}

	if p.isKeyword("IS") {
		return p.parseIsSuffix(left)
	}

	notNegated := p.matchKeyword(keywordNOT)

	expression := p.parsePostfixComparisonSuffix(left)
	if expression != nil {
		return p.maybeNegate(notNegated, expression)
	}

	if notNegated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: left}
	}

	return left
}

// parsePostfixComparisonSuffix matches postfix comparison predicates.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed predicate, or nil when no postfix
// predicate keyword follows.
func (p *parser) parsePostfixComparisonSuffix(left querier_dto.Expression) querier_dto.Expression {
	if p.isKeyword("IN") {
		return p.parseInListSuffix(left)
	}
	if p.isKeyword("BETWEEN") {
		return p.parseBetweenSuffix(left)
	}
	if p.isKeyword("LIKE") {
		return p.parseLikeSuffix(left)
	}
	if p.isKeyword("REGEXP") || p.isKeyword("RLIKE") {
		return p.parseRegexpSuffix(left)
	}
	if p.isKeyword("SOUNDS") {
		return p.parseSoundsLikeSuffix(left)
	}
	return nil
}

// maybeNegate wraps expression in a NOT when negated is true.
//
// Takes negated (bool) which indicates whether to negate.
// Takes expression (querier_dto.Expression) which is the inner expression.
//
// Returns querier_dto.Expression which is the negated or original expression.
func (*parser) maybeNegate(negated bool, expression querier_dto.Expression) querier_dto.Expression {
	if negated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: expression}
	}
	return expression
}

// parseLikeSuffix parses a LIKE predicate with an optional ESCAPE clause.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed comparison.
func (p *parser) parseLikeSuffix(left querier_dto.Expression) querier_dto.Expression {
	keyword := strings.ToUpper(p.advance().value)
	right := p.parseBitwiseExpression()
	if p.matchKeyword("ESCAPE") {
		p.parseBitwiseExpression()
	}
	return &querier_dto.ComparisonExpression{Operator: keyword, Left: left, Right: right}
}

// parseRegexpSuffix parses a REGEXP or RLIKE predicate.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed comparison.
func (p *parser) parseRegexpSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: "REGEXP", Left: left, Right: right}
}

// parseSoundsLikeSuffix parses a SOUNDS LIKE predicate.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed comparison.
func (p *parser) parseSoundsLikeSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	p.matchKeyword("LIKE")
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: "SOUNDS LIKE", Left: left, Right: right}
}

// parseComparisonOperator parses a comparison with optional ANY/ALL/SOME.
//
// Takes left (querier_dto.Expression) which is the LHS of the comparison.
//
// Returns querier_dto.Expression which is the parsed comparison.
func (p *parser) parseComparisonOperator(left querier_dto.Expression) querier_dto.Expression {
	operator := p.advance().value
	if p.matchKeyword("ANY") || p.matchKeyword(keywordALL) || p.matchKeyword("SOME") {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
		return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: &querier_dto.UnknownExpression{}}
	}
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: right}
}

// parseBitwiseExpression parses bitwise &, |, ^, <<, and >> chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseBitwiseExpression() querier_dto.Expression {
	left := p.parseJSONExpression()

	for p.current().kind == tokenOperator &&
		(p.current().value == "&" || p.current().value == "|" || p.current().value == "^" ||
			p.current().value == "<<" || p.current().value == ">>") {
		operator := p.advance().value
		right := p.parseJSONExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseJSONExpression parses JSON `->` and `->>` operator chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseJSONExpression() querier_dto.Expression {
	left := p.parseAddExpression()

	for p.current().kind == tokenArrow || p.current().kind == tokenDoubleArrow {
		operator := p.advance().value
		right := p.parseAddExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseAddExpression parses additive `+` and `-` chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseAddExpression() querier_dto.Expression {
	left := p.parseMulExpression()

	for p.current().kind == tokenOperator &&
		(p.current().value == "+" || p.current().value == "-") {
		operator := p.advance().value
		right := p.parseMulExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseMulExpression parses multiplicative *, /, %, DIV, and MOD chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseMulExpression() querier_dto.Expression {
	left := p.parseUnaryExpression()

	for p.isMulOperator() {
		operator := ""
		if p.current().kind == tokenStar {
			operator = "*"
		} else {
			operator = p.current().value
		}
		p.advance()
		right := p.parseUnaryExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// isMulOperator reports whether the current token is a `*` family operator.
//
// Returns bool which is true for *, /, %, DIV, or MOD.
func (p *parser) isMulOperator() bool {
	if p.current().kind == tokenStar {
		return true
	}
	return p.current().kind == tokenOperator &&
		(p.current().value == "/" || p.current().value == "%" || p.current().value == "DIV" || p.current().value == "MOD")
}

// parseUnaryExpression parses unary -, +, ~, NOT, or BINARY prefixes.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseUnaryExpression() querier_dto.Expression {
	if p.current().kind == tokenOperator &&
		(p.current().value == "-" || p.current().value == "+" || p.current().value == "~") {
		operator := p.advance().value
		inner := p.parseCollateExpression()
		if _, ok := inner.(*querier_dto.LiteralExpression); ok {
			return inner
		}
		return &querier_dto.UnaryOpExpression{Operator: operator, Operand: inner}
	}
	if p.matchKeyword(keywordNOT) {
		inner := p.parseCollateExpression()
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: inner}
	}
	if p.matchKeyword("BINARY") {
		inner := p.parseCollateExpression()
		return &querier_dto.CastExpression{TypeName: "binary", Inner: inner}
	}
	return p.parseCollateExpression()
}

// parseCollateExpression parses an optional COLLATE suffix.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseCollateExpression() querier_dto.Expression {
	left := p.parseSubscriptExpression()

	if p.matchKeyword(keywordCOLLATE) {
		if p.current().kind == tokenIdentifier || p.current().kind == tokenString {
			p.advance()
		}
	}

	return left
}

// parseSubscriptExpression parses `[index]` array subscript chains.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseSubscriptExpression() querier_dto.Expression {
	left := p.parsePrimaryExpression()

	for p.current().kind == tokenLeftBracket {
		p.advance()
		indexExpression := p.parseExpression()
		if p.current().kind == tokenRightBracket {
			p.advance()
		}
		left = &querier_dto.ArraySubscriptExpression{Array: left, Index: indexExpression}
	}

	return left
}

// parsePrimaryExpression parses a primary expression atom.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parsePrimaryExpression() querier_dto.Expression {
	tok := p.current()

	switch tok.kind {
	case tokenNumber:
		return p.parseNumberLiteral(tok)
	case tokenString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "text"}
	case tokenHexString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "varbinary"}
	case tokenBitString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "bit"}
	case tokenQuestionMark, tokenNamedParam:
		return p.parseParameterExpression()
	case tokenLeftParen:
		return p.parseParenthesisedExpression()
	case tokenIdentifier:
		return p.parseIdentifierExpression()
	default:
		p.advance()
		return &querier_dto.UnknownExpression{}
	}
}

// parseNumberLiteral classifies a numeric literal as int or double.
//
// Takes tok (token) which is the number token being consumed.
//
// Returns querier_dto.Expression which is the typed literal expression.
func (p *parser) parseNumberLiteral(tok token) querier_dto.Expression {
	p.advance()
	if strings.Contains(tok.value, ".") || strings.Contains(tok.value, "e") || strings.Contains(tok.value, "E") {
		return &querier_dto.LiteralExpression{TypeName: "double"}
	}
	return &querier_dto.LiteralExpression{TypeName: "int"}
}

// parseParameterExpression registers a bare parameter token.
//
// When an INSERT ... SELECT projection context is active the placeholder is assigned the
// matching INSERT target column (by the projection ordinal) with
// ParameterContextAssignment, mirroring the VALUES path; this lets the analyser type the
// placeholder from the target column even though the SELECT body itself has no scope
// column for it. Otherwise the placeholder is registered as context-unknown and a later
// marker may retag it.
//
// Returns querier_dto.Expression which is an UnknownExpression placeholder.
func (p *parser) parseParameterExpression() querier_dto.Expression {
	parameterToken := p.current()
	p.advance()

	context := querier_dto.ParameterContextUnknown
	var columnRef *querier_dto.ColumnReference
	if ref := p.insertProjectionColumnRef(); ref != nil {
		context = querier_dto.ParameterContextAssignment
		columnRef = ref
	}

	p.registerParameterFromToken(parameterToken, context, columnRef, nil)
	return &querier_dto.UnknownExpression{}
}

// insertProjectionColumnRef returns the INSERT target column reference for the projection
// item currently being parsed, or nil when no INSERT ... SELECT projection context is
// active or the ordinal is beyond the declared target column list.
//
// Returns *querier_dto.ColumnReference which is the target column reference or nil.
func (p *parser) insertProjectionColumnRef() *querier_dto.ColumnReference {
	if p.insertProjectionColumns == nil {
		return nil
	}
	return p.columnRefForIndex(
		p.insertProjectionTable, p.insertProjectionColumns, p.insertProjectionIndex)
}

// parseParenthesisedExpression parses a `(expr)` or scalar subquery.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseParenthesisedExpression() querier_dto.Expression {
	if p.isSubqueryStart() {
		return p.parseScalarSubquery()
	}
	p.advance()
	inner := p.parseExpression()
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return inner
}

var (
	// implicitFunctionIdentifiers lists identifiers that name a function even when written
	// without a trailing parenthesis pair.
	implicitFunctionIdentifiers = map[string]struct{}{
		"CURRENT_TIMESTAMP": {}, "CURRENT_DATE": {}, "CURRENT_TIME": {},
		"CURRENT_USER": {}, "LOCALTIME": {}, "LOCALTIMESTAMP": {},
		"UTC_TIMESTAMP": {}, "UTC_DATE": {}, "UTC_TIME": {},
	}
)

// parseIdentifierExpression dispatches an identifier to its handler.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseIdentifierExpression() querier_dto.Expression {
	upper := strings.ToUpper(p.current().value)

	if handler := p.identifierExpressionHandler(upper); handler != nil {
		return handler()
	}

	if _, isImplicit := implicitFunctionIdentifiers[upper]; isImplicit {
		p.advance()
		if p.current().kind == tokenLeftParen {
			p.advance()
			if p.current().kind == tokenRightParen {
				p.advance()
			}
		}
		return &querier_dto.FunctionCallExpression{FunctionName: strings.ToLower(upper)}
	}

	return p.parseColumnOrFunctionReference()
}

// identifierExpressionHandler returns the parser for a special keyword.
//
// Takes upper (string) which is the upper-cased identifier value.
//
// Returns func() querier_dto.Expression which is the matched parser, or nil when the
// identifier has no special handling.
func (p *parser) identifierExpressionHandler(upper string) func() querier_dto.Expression {
	switch upper {
	case keywordNULL:
		return p.parseNullLiteral
	case "TRUE", "FALSE":
		return p.parseBooleanLiteral
	case "CAST":
		return p.parseCastFunctionExpression
	case "CONVERT":
		return p.parseConvertExpression
	case "COALESCE":
		return p.parseCoalesceExpression
	case "CASE":
		return p.parseCaseExpression
	case keywordEXISTS:
		return func() querier_dto.Expression {
			p.advance()
			return p.parseExistsSubquery()
		}
	case keywordNOT:
		return func() querier_dto.Expression { return &querier_dto.UnknownExpression{} }
	case "INTERVAL":
		return p.parseIntervalLiteral
	case "ROW":
		return p.parseRowConstructorExpression
	case "IF":
		return p.parseIfExpression
	default:
		return nil
	}
}

// parseNullLiteral consumes a NULL token.
//
// Returns querier_dto.Expression which is nil to represent SQL NULL.
func (p *parser) parseNullLiteral() querier_dto.Expression {
	p.advance()
	return nil
}

// parseBooleanLiteral consumes TRUE or FALSE as a tinyint literal.
//
// Returns querier_dto.Expression which is the typed literal expression.
func (p *parser) parseBooleanLiteral() querier_dto.Expression {
	p.advance()
	return &querier_dto.LiteralExpression{TypeName: "tinyint"}
}

var (
	// intervalFieldKeywords lists the unit identifiers accepted after the value expression
	// in a MySQL INTERVAL literal.
	intervalFieldKeywords = map[string]struct{}{
		"YEAR": {}, "MONTH": {}, "DAY": {},
		"HOUR": {}, "MINUTE": {}, "SECOND": {},
		"MICROSECOND": {}, "WEEK": {}, "QUARTER": {},
		"YEAR_MONTH": {}, "DAY_HOUR": {}, "DAY_MINUTE": {},
		"DAY_SECOND": {}, "DAY_MICROSECOND": {},
		"HOUR_MINUTE": {}, "HOUR_SECOND": {}, "HOUR_MICROSECOND": {},
		"MINUTE_SECOND": {}, "MINUTE_MICROSECOND": {},
		"SECOND_MICROSECOND": {},
	}
)

// parseIntervalLiteral parses an INTERVAL value unit expression.
//
// Returns querier_dto.Expression which is the typed literal expression.
func (p *parser) parseIntervalLiteral() querier_dto.Expression {
	p.advance()
	p.parseExpression()
	if p.current().kind == tokenIdentifier {
		if _, ok := intervalFieldKeywords[strings.ToUpper(p.current().value)]; ok {
			p.advance()
		}
	}
	return &querier_dto.LiteralExpression{TypeName: "interval"}
}

// parseRowConstructorExpression skips a ROW(...) constructor.
//
// Returns querier_dto.Expression which is an UnknownExpression placeholder.
func (p *parser) parseRowConstructorExpression() querier_dto.Expression {
	p.advance()
	p.skipRowConstructor()
	return &querier_dto.UnknownExpression{}
}

// parseIfExpression parses IF(cond, then, else) as a function call.
//
// Returns querier_dto.Expression which is the parsed function call.
func (p *parser) parseIfExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.FunctionCallExpression{FunctionName: "if"}
	}
	return p.parseFunctionCall("if", "")
}

// parseColumnOrFunctionReference parses an unqualified column or call.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseColumnOrFunctionReference() querier_dto.Expression {
	name := p.advance().value

	if p.current().kind == tokenDot {
		return p.parseDotQualifiedIdentifier(name)
	}

	if p.current().kind == tokenLeftParen {
		return p.parseFunctionCall(name, "")
	}

	return &querier_dto.ColumnRefExpression{
		TableAlias: "",
		ColumnName: name,
	}
}

// skipRowConstructor consumes a parenthesised ROW value list.
func (p *parser) skipRowConstructor() {
	if p.current().kind != tokenLeftParen {
		return
	}
	p.advance()
	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseExpression()
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
}

// parseDotQualifiedIdentifier parses `<alias>.<column>` or related forms.
//
// Takes name (string) which is the identifier preceding the dot.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseDotQualifiedIdentifier(name string) querier_dto.Expression {
	p.advance()
	if p.current().kind == tokenStar {
		p.advance()
		return &querier_dto.UnknownExpression{}
	}
	if p.current().kind != tokenIdentifier {
		return &querier_dto.ColumnRefExpression{TableAlias: "", ColumnName: name}
	}
	second := p.advance().value

	if p.current().kind == tokenDot {
		return p.parseSchemaQualifiedRef(name, second)
	}
	if p.current().kind == tokenLeftParen {
		return p.parseFunctionCall(second, name)
	}
	return &querier_dto.ColumnRefExpression{TableAlias: name, ColumnName: second}
}

// parseSchemaQualifiedRef parses `<schema>.<table>.<column>` references.
//
// Takes schema (string) which is the leading schema identifier.
// Takes table (string) which is the middle table identifier.
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseSchemaQualifiedRef(schema string, table string) querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenIdentifier {
		return &querier_dto.ColumnRefExpression{TableAlias: schema, ColumnName: table}
	}
	third := p.advance().value
	if p.current().kind == tokenLeftParen {
		return p.parseFunctionCall(third, schema+"."+table)
	}
	return &querier_dto.ColumnRefExpression{TableAlias: table, ColumnName: third}
}

// parseIsSuffix parses an IS NULL, IS TRUE, or related predicate.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed IS predicate.
func (p *parser) parseIsSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	negated := p.matchKeyword(keywordNOT)

	if p.matchKeyword(keywordNULL) {
		return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
	}

	if p.matchKeyword("TRUE") || p.matchKeyword("FALSE") || p.matchKeyword("UNKNOWN") {
		return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
	}

	return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
}

// parseInListSuffix parses an IN(...) predicate, value list or subquery.
//
// Tags any parameters parsed inside the IN list with the LHS column when the LHS is a
// bare column reference.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed IN expression.
func (p *parser) parseInListSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()

	if p.isSubqueryStart() {
		innerTokens, collectError := p.collectParenthesised()
		if collectError != nil {
			return &querier_dto.UnknownExpression{}
		}
		childParser := newParser(innerTokens)
		childParser.parameterCount = p.parameterCount
		childParser.analysisDepth = p.analysisDepth
		childParser.expressionDepth = p.expressionDepth
		childParser.maxParseDepth = p.maxParseDepth
		innerAnalysis, analyseError := childParser.analyseSelect()
		if analyseError != nil {
			return &querier_dto.UnknownExpression{}
		}
		p.parameterCount = childParser.parameterCount
		p.parameterRefs = append(p.parameterRefs, childParser.parameterRefs...)
		return &querier_dto.InListExpression{
			Inner:  left,
			Values: []querier_dto.Expression{&querier_dto.ScalarSubqueryExpression{InnerQuery: innerAnalysis}},
		}
	}

	parameterCountBefore := p.parameterCount
	values := p.parseParenthesisedExpressionList()

	var columnReference *querier_dto.ColumnReference
	if columnExpression, ok := left.(*querier_dto.ColumnRefExpression); ok {
		columnReference = &querier_dto.ColumnReference{
			TableAlias: columnExpression.TableAlias,
			ColumnName: columnExpression.ColumnName,
		}
	}
	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore {
			p.parameterRefs[i].Context = querier_dto.ParameterContextInList
			if columnReference != nil {
				p.parameterRefs[i].ColumnReference = columnReference
			}
		}
	}

	return &querier_dto.InListExpression{Inner: left, Values: values}
}

// parseBetweenSuffix parses a BETWEEN low AND high predicate.
//
// Tags any parameters parsed inside the range with the LHS column when the LHS is a bare
// column reference.
//
// Takes left (querier_dto.Expression) which is the LHS of the predicate.
//
// Returns querier_dto.Expression which is the parsed BETWEEN expression.
func (p *parser) parseBetweenSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()

	parameterCountBefore := p.parameterCount
	low := p.parseAddExpression()
	p.matchKeyword(keywordAND)
	high := p.parseAddExpression()

	var columnReference *querier_dto.ColumnReference
	if columnExpression, ok := left.(*querier_dto.ColumnRefExpression); ok {
		columnReference = &querier_dto.ColumnReference{
			TableAlias: columnExpression.TableAlias,
			ColumnName: columnExpression.ColumnName,
		}
	}
	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore {
			p.parameterRefs[i].Context = querier_dto.ParameterContextBetween
			if columnReference != nil {
				p.parameterRefs[i].ColumnReference = columnReference
			}
		}
	}

	return &querier_dto.BetweenExpression{Inner: left, Low: low, High: high}
}

// parseParenthesisedExpressionList parses a `(expr, expr, ...)` tuple.
//
// Returns []querier_dto.Expression which is the parsed expression list, or nil when no
// opening parenthesis is present.
func (p *parser) parseParenthesisedExpressionList() []querier_dto.Expression {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var values []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		values = append(values, p.parseExpression())
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return values
}

// handleParameterInExpression registers a parameter found mid-expression.
//
// Inspects surrounding tokens to infer the parameter's context, column reference, and any
// cast type, then advances past the token. When the placeholder sits in a function
// argument list the enclosing function name and argument ordinal are stamped onto the
// reference so the analyser can type it from the matched function signature.
func (p *parser) handleParameterInExpression() {
	paramPosition := p.position
	parameterToken := p.current()

	context, columnRef, castType, functionArgument := p.resolveParameterContext(paramPosition)

	p.advance()

	p.registerParameterFromTokenWithFunctionArgument(
		parameterToken, context, columnRef, castType, functionArgument)
}

// resolveParameterContext infers the context for a parameter token.
//
// Tries comparison/arithmetic operator detection first, then LIKE pattern resolution,
// then a wider context scan.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred LHS column when one was
// identified.
// Returns *querier_dto.SQLType which is the cast target type when the parameter sits
// inside a CAST.
// Returns functionArgumentMetadata which carries the enclosing function name and argument
// ordinal when the parameter sits in a function argument list, empty otherwise.
func (p *parser) resolveParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType, functionArgumentMetadata) {
	context, columnRef := p.resolveContextFromPrecedingOperator(paramPosition)
	if context != querier_dto.ParameterContextUnknown {
		return context, columnRef, nil, functionArgumentMetadata{}
	}
	if likeContext, likeColumn := p.resolveLikeContext(paramPosition); likeContext != querier_dto.ParameterContextUnknown {
		return likeContext, likeColumn, nil, functionArgumentMetadata{}
	}
	return p.detectParameterContext(paramPosition)
}

// resolveLikeContext infers the column behind a LIKE-family pattern.
//
// Walks back from the parameter position through balanced parens to find an enclosing
// LIKE, REGEXP, or RLIKE operator, stopping at boolean or clause boundaries. Falls
// through to a wider LHS scan when the immediate-left token is not itself a column. MATCH
// (...) AGAINST (...) full-text syntax is not detected here.
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
	case "AS", "CAST", "CONVERT", "COLLATE", "DISTINCT", "ALL", "NULL", "TRUE", "FALSE",
		"INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT",
		"FLOAT", "DOUBLE", "DECIMAL", "NUMERIC",
		"CHAR", "VARCHAR", "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"BINARY", "VARBINARY", "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
		"BOOLEAN", "BOOL", "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
		"JSON", "ENUM", "SET",
		"ESCAPE", "BETWEEN", "IS", "IN", "EXISTS", "NOT", "ASC", "DESC":
		return true
	}
	return false
}

// isLikePatternKeyword reports whether a keyword introduces a pattern match.
//
// Matches MySQL's LIKE, REGEXP, and RLIKE. MySQL has no ILIKE or GLOB; case-insensitivity
// is collation-driven. MATCH is part of MATCH (...) AGAINST (...) full-text syntax and is
// not detected here.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true for LIKE, REGEXP, or RLIKE.
func isLikePatternKeyword(keyword string) bool {
	switch keyword {
	case "LIKE", "REGEXP", "RLIKE":
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
	return engine_shared.IsClauseBoundaryKeyword(keyword)
}

// resolveContextFromPrecedingOperator infers context from the preceding comparison or
// arithmetic operator and its left operand.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred LHS column when one was
// identified.
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

// extractColumnReferenceOrParenthesised returns the column at position or scans inside a
// `(...)` group when position closes a paren.
//
// Takes position (int) which is the token index to inspect.
//
// Returns *querier_dto.ColumnReference which is the resolved column or nil when none is
// identified.
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

// detectParameterContext infers context from the enclosing parenthesis.
//
// Recognises IN lists, CAST expressions, and function argument lists by inspecting the
// identifier immediately before the opening paren. For a function argument it
// additionally records the enclosing function name and the placeholder's zero-based
// argument ordinal so the analyser can type the placeholder from the matched function
// signature.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is the inferred context.
// Returns *querier_dto.ColumnReference which is the inferred column for IN lists, or nil
// otherwise.
// Returns *querier_dto.SQLType which is the cast target type for CAST contexts, or nil
// otherwise.
// Returns functionArgumentMetadata which carries the enclosing function name and argument
// ordinal for a function argument, empty otherwise.
func (p *parser) detectParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType, functionArgumentMetadata) {
	enclosingParen := p.findEnclosingParen(paramPosition)
	if enclosingParen < 0 {
		return querier_dto.ParameterContextUnknown, nil, nil, functionArgumentMetadata{}
	}

	if enclosingParen >= 2 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, keywordIN) {
		columnRef := p.extractColumnReferenceBeforeIN(enclosingParen - 1)
		return querier_dto.ParameterContextInList, columnRef, nil, functionArgumentMetadata{}
	}

	if enclosingParen >= 1 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, keywordCAST) {
		castType := p.extractCastType(paramPosition)
		if castType != nil {
			return querier_dto.ParameterContextCast, nil, castType, functionArgumentMetadata{}
		}
	}

	if enclosingParen >= 1 && p.tokens[enclosingParen-1].kind == tokenIdentifier {
		functionName := strings.ToUpper(p.tokens[enclosingParen-1].value)
		if functionName != keywordIN && functionName != keywordCAST &&
			functionName != keywordSELECT && functionName != keywordWHERE {
			metadata := functionArgumentMetadata{
				enclosingFunctionName: p.enclosingFunctionName(enclosingParen - 1),
				argumentOrdinal:       p.argumentOrdinalAt(enclosingParen, paramPosition),
			}
			return querier_dto.ParameterContextFunctionArgument, nil, nil, metadata
		}
	}

	return querier_dto.ParameterContextUnknown, nil, nil, functionArgumentMetadata{}
}

// enclosingFunctionName reconstructs the lower-cased, optionally schema-qualified name of
// the function whose opening parenthesis the placeholder sits in.
//
// The name is read from the identifier at namePosition (the token immediately before the
// opening paren). A leading "schema ." qualifier is folded into the returned name so it
// matches the schema-qualified name the engine records on a function or TVF reference,
// which is how the analyser looks the function up.
//
// Takes namePosition (int) which is the token index of the function name identifier.
//
// Returns string which is the lower-cased function name, schema-qualified when
// applicable.
func (p *parser) enclosingFunctionName(namePosition int) string {
	if namePosition < 0 || namePosition >= len(p.tokens) ||
		p.tokens[namePosition].kind != tokenIdentifier {
		return ""
	}

	name := p.tokens[namePosition].value
	if namePosition >= 2 &&
		p.tokens[namePosition-1].kind == tokenDot &&
		p.tokens[namePosition-2].kind == tokenIdentifier {
		name = p.tokens[namePosition-2].value + "." + name
	}

	return strings.ToLower(name)
}

// argumentOrdinalAt computes the zero-based ordinal of the argument slot the placeholder
// occupies within the call whose opening parenthesis is openParen.
//
// The ordinal counts top-level commas (those at the call's own parenthesis depth) between
// the opening paren and the placeholder, so a literal, column, or nested expression
// argument still consumes a slot. The returned value is the literal argument slot and is
// NOT clamped for variadic functions; the analyser performs any variadic clamp.
//
// Takes openParen (int) which is the token index of the call's opening parenthesis.
// Takes paramPosition (int) which is the placeholder's token index.
//
// Returns int which is the zero-based argument ordinal.
func (p *parser) argumentOrdinalAt(openParen int, paramPosition int) int {
	ordinal := 0
	depth := 0
	for i := openParen + 1; i < paramPosition && i < len(p.tokens); i++ {
		switch p.tokens[i].kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenComma:
			if depth == 0 {
				ordinal++
			}
		}
	}
	return ordinal
}

// findEnclosingParen returns the index of the enclosing `(` at depth zero.
//
// Takes position (int) which is the inner token index to start from.
//
// Returns int which is the index of the enclosing `(`, or -1 when none.
func (p *parser) findEnclosingParen(position int) int {
	return engine_shared.FindEnclosingParen(position,
		func(index int) bool { return p.tokens[index].kind == tokenLeftParen },
		func(index int) bool { return p.tokens[index].kind == tokenRightParen },
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(p.tokens[index].value))
		},
	)
}

// extractColumnReferenceBeforeIN returns the column before an IN keyword.
//
// Takes inPosition (int) which is the IN keyword's token index.
//
// Returns *querier_dto.ColumnReference which is the LHS column or nil.
func (p *parser) extractColumnReferenceBeforeIN(inPosition int) *querier_dto.ColumnReference {
	if inPosition < 1 {
		return nil
	}
	return p.extractColumnReference(inPosition - 1)
}

// extractCastType reads the AS-prefixed type name following a parameter.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns *querier_dto.SQLType which is the normalised cast target type, or nil when no
// AS clause follows the parameter.
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

// findASKeywordAfter locates an AS keyword following paramPosition.
//
// Stops when the closing paren of the enclosing call is reached.
//
// Takes paramPosition (int) which is the starting token index.
//
// Returns int which is the AS keyword's index, or -1 when not found.
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

// collectCastTypeTokens joins consecutive identifier tokens for a type.
//
// Stops at a closing paren or any clause keyword that ends the type.
//
// Takes startPosition (int) which is the first identifier's index.
//
// Returns string which is the space-joined type name.
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

// isKeywordAt reports whether the token at position matches a keyword.
//
// Takes position (int) which is the token index to check.
// Takes keywords (...string) which are the candidate upper-case keywords.
//
// Returns bool which is true when the token matches any keyword.
func (p *parser) isKeywordAt(position int, keywords ...string) bool {
	if position >= len(p.tokens) || p.tokens[position].kind != tokenIdentifier {
		return false
	}
	return slices.Contains(keywords, strings.ToUpper(p.tokens[position].value))
}

// extractColumnReference reads a bare or qualified column at position.
//
// Takes position (int) which is the identifier's token index.
//
// Returns *querier_dto.ColumnReference which is the resolved column or nil when position
// does not name an identifier.
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

// isComparisonOperator reports whether operator is a SQL comparison.
//
// Takes operator (string) which is the operator text to test.
//
// Returns bool which is true for =, <>, !=, <, >, <=, >=, or <=>.
func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=", "<=>":
		return true
	}
	return false
}

// isArithmeticOperator reports whether operator is a SQL arithmetic op.
//
// Takes operator (string) which is the operator text to test.
//
// Returns bool which is true for +, -, /, or %.
func isArithmeticOperator(operator string) bool {
	switch operator {
	case "+", "-", "/", "%":
		return true
	}
	return false
}

// extractColumnReferenceFromParenthesised scans inside `(...)` for a column.
//
// Takes rightParenPosition (int) which is the closing paren's index.
//
// Returns *querier_dto.ColumnReference which is the first column found inside the group,
// or nil when none is present.
func (p *parser) extractColumnReferenceFromParenthesised(rightParenPosition int) *querier_dto.ColumnReference {
	leftParenPosition := p.findMatchingLeftParen(rightParenPosition)
	if leftParenPosition < 0 {
		return nil
	}
	return p.scanForColumnReference(leftParenPosition+1, rightParenPosition)
}

// findMatchingLeftParen pairs a `)` with its matching `(`.
//
// Takes rightParenPosition (int) which is the closing paren's index.
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

// scanForColumnReference returns the first column found in a token range.
//
// Takes startPosition (int) which is the inclusive start index.
// Takes endPosition (int) which is the exclusive end index.
//
// Returns *querier_dto.ColumnReference which is the first column found, or nil when the
// range contains none.
func (p *parser) scanForColumnReference(startPosition int, endPosition int) *querier_dto.ColumnReference {
	for j := startPosition; j < endPosition; j++ {
		reference := p.extractColumnReference(j)
		if reference != nil {
			return reference
		}
	}
	return nil
}

// parseCastTypeName parses the target type of a CAST or CONVERT.
//
// Handles schema-qualified names and multi-word type keywords such as CHARACTER VARYING
// and DOUBLE PRECISION.
//
// Returns string which is the assembled type name.
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

	return builder.String()
}

// appendSchemaQualifier appends `.identifier` when a dot follows.
//
// Takes builder (*strings.Builder) which receives the appended text.
func (p *parser) appendSchemaQualifier(builder *strings.Builder) {
	if p.current().kind == tokenDot && p.peek().kind == tokenIdentifier {
		p.advance()
		builder.WriteByte('.')
		builder.WriteString(p.advance().value)
	}
}

// appendMultiWordTypeKeywords appends trailing words of a compound type.
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

// isMultiWordTypeKeyword reports whether upper is a type continuation word.
//
// Takes upper (string) which is the upper-cased candidate keyword.
//
// Returns bool which is true when upper extends a compound type name.
func isMultiWordTypeKeyword(upper string) bool {
	switch upper {
	case "PRECISION", keywordUNSIGNED, "VARYING", "CHARACTER", "DOUBLE":
		return true
	}
	return false
}

// analyseSubqueryBody analyses a parenthesised subquery in a depth-inheriting child
// parser.
//
// It collects the parenthesised subquery, analyses it in a child parser that inherits the
// parameter and depth state, and splices the child's parameter results back into this
// parser. The EXISTS and scalar subquery parsers share it so the child setup lives in one
// place.
//
// Returns *querier_dto.RawQueryAnalysis which is the inner SELECT analysis, or nil on
// failure.
// Returns bool which is true when the subquery was collected and analysed successfully.
func (p *parser) analyseSubqueryBody() (*querier_dto.RawQueryAnalysis, bool) {
	innerTokens, collectError := p.collectParenthesised()
	if collectError != nil {
		return nil, false
	}

	childParser := newParser(innerTokens)
	childParser.parameterCount = p.parameterCount
	childParser.analysisDepth = p.analysisDepth
	childParser.expressionDepth = p.expressionDepth
	childParser.maxParseDepth = p.maxParseDepth
	innerAnalysis, analyseError := childParser.analyseSelect()
	if analyseError != nil {
		return nil, false
	}
	p.parameterCount = childParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, childParser.parameterRefs...)

	return innerAnalysis, true
}

// parseExistsSubquery parses the `(SELECT ...)` of an EXISTS predicate.
//
// Returns querier_dto.Expression which is the parsed EXISTS expression.
func (p *parser) parseExistsSubquery() querier_dto.Expression {
	innerAnalysis, ok := p.analyseSubqueryBody()
	if !ok {
		return &querier_dto.ExistsExpression{}
	}
	return &querier_dto.ExistsExpression{InnerQuery: innerAnalysis}
}

// parseScalarSubquery parses a parenthesised scalar SELECT subquery.
//
// Returns querier_dto.Expression which is the parsed scalar subquery.
func (p *parser) parseScalarSubquery() querier_dto.Expression {
	innerAnalysis, ok := p.analyseSubqueryBody()
	if !ok {
		return &querier_dto.UnknownExpression{}
	}
	return &querier_dto.ScalarSubqueryExpression{InnerQuery: innerAnalysis}
}
