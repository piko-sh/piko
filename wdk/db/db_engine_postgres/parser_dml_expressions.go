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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseExpression parses a full SQL expression at the current position.
//
// Returns querier_dto.Expression which is the parsed expression.
func (p *parser) parseExpression() querier_dto.Expression {
	return p.parseOrExpression()
}

// parseOrExpression parses a left-associative OR chain.
//
// Returns querier_dto.Expression which is the OR expression, or the single operand when
// no OR follows.
func (p *parser) parseOrExpression() querier_dto.Expression {
	left := p.parseAndExpression()
	for p.matchKeyword("OR") {
		right := p.parseAndExpression()
		left = &querier_dto.LogicalOpExpression{
			Operator: "OR",
			Operands: []querier_dto.Expression{left, right},
		}
	}
	return left
}

// parseAndExpression parses a left-associative AND chain.
//
// Returns querier_dto.Expression which is the AND expression, or the single operand when
// no AND follows.
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

// parseNotExpression parses an optional leading NOT followed by a comparison expression.
//
// Returns querier_dto.Expression which is the negated or bare comparison expression.
func (p *parser) parseNotExpression() querier_dto.Expression {
	if p.matchKeyword(keywordNOT) {
		operand := p.parseComparisonExpression()
		return &querier_dto.UnaryOpExpression{
			Operator: keywordNOT,
			Operand:  operand,
		}
	}
	return p.parseComparisonExpression()
}

// parseComparisonExpression parses a comparison, IS, IN, BETWEEN, LIKE, or SIMILAR TO
// expression and any leading NOT.
//
// Returns querier_dto.Expression which is the comparison expression.
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

// parsePostfixComparisonSuffix dispatches to the IN, BETWEEN, LIKE, or SIMILAR TO suffix
// parser when one of those keywords follows.
//
// Takes left (querier_dto.Expression) which is the left-hand side expression.
//
// Returns querier_dto.Expression which is the wrapped expression, or nil when no postfix
// keyword follows.
func (p *parser) parsePostfixComparisonSuffix(left querier_dto.Expression) querier_dto.Expression {
	if p.isKeyword("IN") {
		return p.parseInListSuffix(left)
	}
	if p.isKeyword("BETWEEN") {
		return p.parseBetweenSuffix(left)
	}
	if p.isKeyword("LIKE") || p.isKeyword("ILIKE") {
		return p.parseLikeSuffix(left)
	}
	if p.isKeyword("SIMILAR") {
		return p.parseSimilarToSuffix(left)
	}
	return nil
}

// maybeNegate wraps expression in a NOT when negated is true.
//
// Takes negated (bool) which signals whether negation applies.
// Takes expression (querier_dto.Expression) which is the inner expression.
//
// Returns querier_dto.Expression which is the negated or original expression.
func (*parser) maybeNegate(negated bool, expression querier_dto.Expression) querier_dto.Expression {
	if negated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: expression}
	}
	return expression
}

// parseLikeSuffix parses the right-hand side of a LIKE or ILIKE comparison, including any
// ESCAPE clause.
//
// Takes left (querier_dto.Expression) which is the LIKE expression's left-hand side.
//
// Returns querier_dto.Expression which is the LIKE comparison.
func (p *parser) parseLikeSuffix(left querier_dto.Expression) querier_dto.Expression {
	keyword := strings.ToUpper(p.advance().value)
	right := p.parseBitwiseExpression()
	if p.matchKeyword("ESCAPE") {
		p.parseBitwiseExpression()
	}
	return &querier_dto.ComparisonExpression{Operator: keyword, Left: left, Right: right}
}

// parseSimilarToSuffix parses the right-hand side of a SIMILAR TO comparison, including
// any ESCAPE clause.
//
// Takes left (querier_dto.Expression) which is the LHS of the comparison.
//
// Returns querier_dto.Expression which is the SIMILAR TO comparison.
func (p *parser) parseSimilarToSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	p.matchKeyword("TO")
	right := p.parseBitwiseExpression()
	if p.matchKeyword("ESCAPE") {
		p.parseBitwiseExpression()
	}
	return &querier_dto.ComparisonExpression{Operator: "SIMILAR TO", Left: left, Right: right}
}

// parseComparisonOperator parses a binary comparison expression and any ANY / ALL / SOME
// quantifier subquery suffix.
//
// Takes left (querier_dto.Expression) which is the comparison's LHS.
//
// Returns querier_dto.Expression which is the comparison expression.
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

// parseBitwiseExpression parses a left-associative chain of bitwise operators (`&`, `|`,
// `#`, `<<`, `>>`).
//
// Returns querier_dto.Expression which is the bitwise expression.
func (p *parser) parseBitwiseExpression() querier_dto.Expression {
	left := p.parseJSONExpression()

	for p.current().kind == tokenOperator &&
		(p.current().value == "&" || p.current().value == "|" || p.current().value == "#" ||
			p.current().value == "<<" || p.current().value == ">>") {
		operator := p.advance().value
		right := p.parseJSONExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseJSONExpression parses a left-associative chain of JSON access operators (`->`,
// `->>`, `#>`, `#>>`).
//
// Returns querier_dto.Expression which is the JSON expression.
func (p *parser) parseJSONExpression() querier_dto.Expression {
	left := p.parseAddExpression()

	for p.current().kind == tokenArrow || p.current().kind == tokenDoubleArrow ||
		p.current().kind == tokenHashArrow || p.current().kind == tokenHashDoubleArrow {
		operator := p.advance().value
		right := p.parseAddExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseAddExpression parses a left-associative chain of `+` and `-` binary operators.
//
// Returns querier_dto.Expression which is the additive expression.
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

// parseMulExpression parses a left-associative chain of `*`, `/`, and `%` operators.
//
// Returns querier_dto.Expression which is the multiplicative expression.
func (p *parser) parseMulExpression() querier_dto.Expression {
	left := p.parseConcatExpression()

	for p.isMulOperator() {
		operator := p.advance().value
		right := p.parseConcatExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// isMulOperator reports whether the current token is a multiplicative operator.
//
// Returns bool which is true for `*`, `/`, or `%`.
func (p *parser) isMulOperator() bool {
	if p.current().kind == tokenStar {
		return true
	}
	return p.current().kind == tokenOperator &&
		(p.current().value == "/" || p.current().value == "%")
}

// parseConcatExpression parses a left-associative chain of `||` concatenation operators.
//
// Returns querier_dto.Expression which is the concatenation expression.
func (p *parser) parseConcatExpression() querier_dto.Expression {
	left := p.parseUnaryExpression()

	for p.current().kind == tokenOperator && p.current().value == "||" {
		p.advance()
		right := p.parseUnaryExpression()
		left = &querier_dto.BinaryOpExpression{Operator: "||", Left: left, Right: right}
	}

	return left
}

// parseUnaryExpression parses a unary `-`, `+`, `~`, or NOT prefix and the following
// expression.
//
// Returns querier_dto.Expression which is the unary expression, or the underlying
// expression when no unary prefix appears.
func (p *parser) parseUnaryExpression() querier_dto.Expression {
	if p.current().kind == tokenOperator &&
		(p.current().value == "-" || p.current().value == "+" || p.current().value == "~") {
		operator := p.advance().value
		inner := p.parseCastExpression()
		if _, ok := inner.(*querier_dto.LiteralExpression); ok {
			return inner
		}
		return &querier_dto.UnaryOpExpression{Operator: operator, Operand: inner}
	}
	if p.matchKeyword(keywordNOT) {
		inner := p.parseCastExpression()
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: inner}
	}
	return p.parseCastExpression()
}

// parseCastExpression parses any trailing `::type` casts after a subscript expression.
//
// Returns querier_dto.Expression which is the cast expression, or the underlying
// expression when no cast follows.
func (p *parser) parseCastExpression() querier_dto.Expression {
	left := p.parseSubscriptExpression()

	for p.current().kind == tokenCast {
		p.advance()
		typeName := p.parseCastTypeName()

		if len(p.parameterRefs) > 0 {
			lastIndex := len(p.parameterRefs) - 1
			if p.parameterRefs[lastIndex].Context == querier_dto.ParameterContextUnknown {
				p.parameterRefs[lastIndex].Context = querier_dto.ParameterContextCast
				p.parameterRefs[lastIndex].CastType = new(normaliseTypeName(typeName, nil))
			}
		}

		left = &querier_dto.CastExpression{
			TypeName: strings.ToLower(typeName),
			Inner:    left,
		}
	}

	return left
}

// parseSubscriptExpression parses array subscript or slice expressions after a primary
// expression.
//
// Returns querier_dto.Expression which is the subscript expression, or the underlying
// primary when no subscript follows.
func (p *parser) parseSubscriptExpression() querier_dto.Expression {
	left := p.parsePrimaryExpression()

	for p.current().kind == tokenLeftBracket {
		p.advance()
		indexExpression := p.parseExpression()
		isSlice := false
		if p.current().kind == tokenOperator && p.current().value == ":" {
			p.advance()
			p.parseExpression()
			isSlice = true
		}
		if p.current().kind == tokenRightBracket {
			p.advance()
		}
		if isSlice {
			left = &querier_dto.UnknownExpression{}
		} else {
			left = &querier_dto.ArraySubscriptExpression{Array: left, Index: indexExpression}
		}
	}

	return left
}

// parsePrimaryExpression parses a literal, parameter, parenthesised expression, or
// identifier expression.
//
// Returns querier_dto.Expression which is the parsed primary expression, or
// UnknownExpression when the token is unrecognised.
func (p *parser) parsePrimaryExpression() querier_dto.Expression {
	tok := p.current()

	switch tok.kind {
	case tokenNumber:
		return p.parseNumberLiteral(tok)
	case tokenString, tokenEscapeString, tokenDollarString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "text"}
	case tokenBitString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "bit"}
	case tokenDollarParam, tokenNamedParam:
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

// parseNumberLiteral consumes a number literal token and classifies it as integer or
// double precision.
//
// Takes tok (token) which is the number token.
//
// Returns querier_dto.Expression which is the literal expression.
func (p *parser) parseNumberLiteral(tok token) querier_dto.Expression {
	p.advance()
	if strings.Contains(tok.value, ".") || strings.Contains(tok.value, "e") || strings.Contains(tok.value, "E") {
		return &querier_dto.LiteralExpression{TypeName: "double precision"}
	}
	return &querier_dto.LiteralExpression{TypeName: "integer"}
}

// parseParameterExpression consumes a parameter token and records it with an unknown
// context.
//
// Returns querier_dto.Expression which is an UnknownExpression standing in for the
// parameter.
func (p *parser) parseParameterExpression() querier_dto.Expression {
	parameterToken := p.current()
	p.advance()
	p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextUnknown, nil, nil)
	return &querier_dto.UnknownExpression{}
}

// parseParenthesisedExpression parses a parenthesised expression, dispatching to
// scalar-subquery handling when a SELECT follows the opening paren.
//
// Returns querier_dto.Expression which is the inner expression or scalar subquery.
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
	// implicitFunctionIdentifiers names SQL identifiers that act as zero-argument function
	// calls without trailing parentheses.
	implicitFunctionIdentifiers = map[string]struct{}{
		"CURRENT_TIMESTAMP": {}, "CURRENT_DATE": {}, "CURRENT_TIME": {},
		"CURRENT_USER": {}, "SESSION_USER": {}, "LOCALTIME": {},
		"LOCALTIMESTAMP": {}, "CURRENT_ROLE": {}, "CURRENT_CATALOG": {},
		"CURRENT_SCHEMA": {},
	}
)

// parseIdentifierExpression parses an identifier-rooted expression, dispatching to
// specialised handlers for keywords such as NULL, CAST, and CASE.
//
// Returns querier_dto.Expression which is the parsed expression.
func (p *parser) parseIdentifierExpression() querier_dto.Expression {
	upper := strings.ToUpper(p.current().value)

	if handler := p.identifierExpressionHandler(upper); handler != nil {
		return handler()
	}

	if _, isImplicit := implicitFunctionIdentifiers[upper]; isImplicit {
		p.advance()
		return &querier_dto.FunctionCallExpression{FunctionName: strings.ToLower(upper)}
	}

	return p.parseColumnOrFunctionReference()
}

// identifierExpressionHandler returns the parsing closure for a recognised identifier
// keyword such as NULL, TRUE, CAST, or CASE.
//
// Takes upper (string) which is the upper-case identifier value.
//
// Returns func() querier_dto.Expression which is the matching sub-parser, or nil when the
// identifier is not recognised.
func (p *parser) identifierExpressionHandler(upper string) func() querier_dto.Expression {
	switch upper {
	case keywordNULL:
		return p.parseNullLiteral
	case "TRUE", "FALSE":
		return p.parseBooleanLiteral
	case "CAST":
		return p.parseCastFunctionExpression
	case "COALESCE":
		return p.parseCoalesceExpression
	case "CASE":
		return p.parseCaseExpression
	case keywordEXISTS:
		return func() querier_dto.Expression {
			p.advance()
			return p.parseExistsSubquery()
		}
	case "ARRAY":
		return p.parseArrayExpression
	case keywordNOT:
		return func() querier_dto.Expression { return &querier_dto.UnknownExpression{} }
	case "INTERVAL":
		return p.parseIntervalLiteral
	case keywordROW:
		return p.parseRowConstructorExpression
	default:
		return nil
	}
}

// parseNullLiteral consumes the NULL keyword.
//
// Returns querier_dto.Expression which is nil to represent SQL NULL.
func (p *parser) parseNullLiteral() querier_dto.Expression {
	p.advance()
	return nil
}

// parseBooleanLiteral consumes a TRUE or FALSE keyword.
//
// Returns querier_dto.Expression which is the boolean literal.
func (p *parser) parseBooleanLiteral() querier_dto.Expression {
	p.advance()
	return &querier_dto.LiteralExpression{TypeName: "boolean"}
}

var (
	// intervalFieldKeywords lists the field keywords that may follow an INTERVAL literal's
	// value.
	intervalFieldKeywords = map[string]struct{}{
		"YEAR": {}, "MONTH": {}, "DAY": {},
		"HOUR": {}, "MINUTE": {}, "SECOND": {},
		"TO": {},
	}
)

// parseIntervalLiteral consumes an INTERVAL literal, its value, and any trailing field
// qualifiers.
//
// Returns querier_dto.Expression which is the interval literal expression.
func (p *parser) parseIntervalLiteral() querier_dto.Expression {
	p.advance()
	if p.current().kind == tokenString {
		p.advance()
	}
	for p.current().kind == tokenIdentifier {
		if _, ok := intervalFieldKeywords[strings.ToUpper(p.current().value)]; !ok {
			break
		}
		p.advance()
	}
	return &querier_dto.LiteralExpression{TypeName: "interval"}
}

// parseRowConstructorExpression consumes the ROW keyword and the trailing parenthesised
// list.
//
// Returns querier_dto.Expression which is an UnknownExpression representing the row.
func (p *parser) parseRowConstructorExpression() querier_dto.Expression {
	p.advance()
	p.skipRowConstructor()
	return &querier_dto.UnknownExpression{}
}

// parseColumnOrFunctionReference parses a bare identifier as a column reference or
// function call depending on the following token.
//
// Returns querier_dto.Expression which is the column reference, function call, or
// qualified reference.
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

// skipRowConstructor consumes the parenthesised expression list of a ROW constructor.
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

// parseDotQualifiedIdentifier parses the tail of a dot-qualified reference such as
// `alias.column`, `alias.*`, or `schema.function(...)`.
//
// Takes name (string) which is the identifier before the dot.
//
// Returns querier_dto.Expression which is the qualified column or function expression.
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

// parseSchemaQualifiedRef parses the tail of a three-part reference such as
// `schema.table.column` or `schema.table.function(...)`.
//
// Takes schema (string) which is the leading schema name.
// Takes table (string) which is the middle identifier.
//
// Returns querier_dto.Expression which is the qualified column or function expression.
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

// parseIsSuffix parses the tail of an IS predicate including IS NULL, IS DISTINCT FROM,
// and IS TRUE/FALSE/UNKNOWN.
//
// Takes left (querier_dto.Expression) which is the IS predicate's LHS.
//
// Returns querier_dto.Expression which is the IS expression.
func (p *parser) parseIsSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	negated := p.matchKeyword(keywordNOT)

	if p.matchKeyword(keywordNULL) {
		return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
	}

	if p.matchKeyword("DISTINCT") {
		p.matchKeyword(keywordFROM)
		right := p.parseBitwiseExpression()
		operator := "IS DISTINCT FROM"
		if negated {
			operator = "IS NOT DISTINCT FROM"
		}
		return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: right}
	}

	if p.matchKeyword("TRUE") || p.matchKeyword("FALSE") || p.matchKeyword("UNKNOWN") {
		return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
	}

	return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
}

// parseInListSuffix parses an IN clause containing either a subquery or an explicit list
// of values and tags any contained parameters with the inferred column reference.
//
// Takes left (querier_dto.Expression) which is the IN predicate's LHS.
//
// Returns querier_dto.Expression which is the IN-list expression.
func (p *parser) parseInListSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()

	if p.isSubqueryStart() {
		innerTokens, collectError := p.collectParenthesised()
		if collectError != nil {
			return &querier_dto.UnknownExpression{}
		}
		childParser := newParser(innerTokens)
		childParser.parameterCount = p.parameterCount
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

// parseBetweenSuffix parses the low and high operands of a BETWEEN predicate and tags any
// contained parameters with the inferred column reference.
//
// Takes left (querier_dto.Expression) which is the BETWEEN predicate's LHS.
//
// Returns querier_dto.Expression which is the BETWEEN expression.
func (p *parser) parseBetweenSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	p.matchKeyword("SYMMETRIC")

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

// parseParenthesisedExpressionList parses a comma-separated list of expressions enclosed
// in parentheses.
//
// Returns []querier_dto.Expression which is the list of parsed expressions, or nil when
// the current token is not '('.
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
