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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseExpression() querier_dto.Expression {
	return p.parseOrExpression()
}

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

func (*parser) maybeNegate(negated bool, expression querier_dto.Expression) querier_dto.Expression {
	if negated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: expression}
	}
	return expression
}

func (p *parser) parseLikeSuffix(left querier_dto.Expression) querier_dto.Expression {
	keyword := strings.ToUpper(p.advance().value)
	right := p.parseBitwiseExpression()
	if p.matchKeyword("ESCAPE") {
		p.parseBitwiseExpression()
	}
	return &querier_dto.ComparisonExpression{Operator: keyword, Left: left, Right: right}
}

func (p *parser) parseSimilarToSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	p.matchKeyword("TO")
	right := p.parseBitwiseExpression()
	if p.matchKeyword("ESCAPE") {
		p.parseBitwiseExpression()
	}
	return &querier_dto.ComparisonExpression{Operator: "SIMILAR TO", Left: left, Right: right}
}

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

func (p *parser) parseJSONExpression() querier_dto.Expression {
	left := p.parseAddExpression()

	for p.current().kind == tokenArrow || p.current().kind == tokenDoubleArrow {
		operator := p.advance().value
		right := p.parseAddExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

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

func (p *parser) parseMulExpression() querier_dto.Expression {
	left := p.parseConcatExpression()

	for p.isMulOperator() {
		operator := p.advance().value
		right := p.parseConcatExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

func (p *parser) isMulOperator() bool {
	if p.current().kind == tokenStar {
		return true
	}
	return p.current().kind == tokenOperator &&
		(p.current().value == "/" || p.current().value == "%")
}

func (p *parser) parseConcatExpression() querier_dto.Expression {
	left := p.parseUnaryExpression()

	for p.current().kind == tokenOperator && p.current().value == "||" {
		p.advance()
		right := p.parseUnaryExpression()
		left = &querier_dto.BinaryOpExpression{Operator: "||", Left: left, Right: right}
	}

	return left
}

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

func (p *parser) parsePrimaryExpression() querier_dto.Expression {
	tok := p.current()

	switch tok.kind {
	case tokenNumber:
		return p.parseNumberLiteral(tok)
	case tokenString:
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

func (p *parser) parseNumberLiteral(tok token) querier_dto.Expression {
	p.advance()
	if strings.Contains(tok.value, ".") || strings.Contains(tok.value, "e") || strings.Contains(tok.value, "E") {
		return &querier_dto.LiteralExpression{TypeName: "double precision"}
	}
	return &querier_dto.LiteralExpression{TypeName: "integer"}
}

func (p *parser) parseParameterExpression() querier_dto.Expression {
	parameterToken := p.current()
	p.advance()
	p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextUnknown, nil, nil)
	return &querier_dto.UnknownExpression{}
}

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

var implicitFunctionIdentifiers = map[string]struct{}{
	"CURRENT_TIMESTAMP": {}, "CURRENT_DATE": {}, "CURRENT_TIME": {},
	"CURRENT_USER": {}, "SESSION_USER": {}, "LOCALTIME": {},
	"LOCALTIMESTAMP": {}, "CURRENT_ROLE": {}, "CURRENT_CATALOG": {},
	"CURRENT_SCHEMA": {},
}

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

func (p *parser) parseNullLiteral() querier_dto.Expression {
	p.advance()
	return nil
}

func (p *parser) parseBooleanLiteral() querier_dto.Expression {
	p.advance()
	return &querier_dto.LiteralExpression{TypeName: "boolean"}
}

var intervalFieldKeywords = map[string]struct{}{
	"YEAR": {}, "MONTH": {}, "DAY": {},
	"HOUR": {}, "MINUTE": {}, "SECOND": {},
	"TO": {},
}

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

func (p *parser) parseRowConstructorExpression() querier_dto.Expression {
	p.advance()
	p.skipRowConstructor()
	return &querier_dto.UnknownExpression{}
}

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
