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

package db_engine_sqlite

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

var expressionTerminators = map[string]bool{
	keywordGROUP: true, keywordHAVING: true, keywordORDER: true, keywordLIMIT: true,
	keywordUNION: true, keywordINTERSECT: true, keywordEXCEPT: true,
	keywordRETURNING: true, keywordSET: true, keywordON: true,
	keywordFROM: true, keywordWHERE: true,
}

func (p *parser) parseWhereExpression() {
	p.parseExpressionUntilTerminator()
}

func (p *parser) parseExpressionUntilTerminator() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()

		if tok.kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if tok.kind == tokenRightParen {
			if depth == 0 {
				break
			}
			depth--
			p.advance()
			continue
		}

		if depth == 0 && isExpressionTerminator(tok) {
			break
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

func isExpressionTerminator(tok token) bool {
	return tok.kind == tokenIdentifier && expressionTerminators[strings.ToUpper(tok.value)]
}

func (p *parser) handleParameterInExpression() {
	paramPosition := p.position
	parameterToken := p.current()

	context, columnRef := p.resolveComparisonContext(paramPosition)

	if context == querier_dto.ParameterContextUnknown {
		if likeContext, likeColumn := p.resolveLikeContext(paramPosition); likeContext != querier_dto.ParameterContextUnknown {
			context, columnRef = likeContext, likeColumn
		}
	}

	var castType *querier_dto.SQLType
	if context == querier_dto.ParameterContextUnknown {
		context, columnRef, castType = p.detectParameterContext(paramPosition)
	}

	p.advance()
	p.registerParameterFromToken(parameterToken, context, columnRef, castType)
}

func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

func (p *parser) parseExpression() querier_dto.Expression {
	return p.parseOrExpression()
}

func (p *parser) parseOrExpression() querier_dto.Expression {
	left := p.parseAndExpression()
	for p.matchKeyword(keywordOR) {
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
			Operator: "NOT",
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
		return p.parseIsNullSuffix(left)
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

// parsePostfixComparisonSuffix attempts to parse a
// postfix comparison operator (IN, BETWEEN, LIKE, GLOB,
// REGEXP, MATCH) that follows the left operand. This is
// called after optionally consuming a NOT keyword so
// that constructs like NOT IN and NOT LIKE are handled.
//
// Takes left (querier_dto.Expression) which holds the
// already-parsed left operand of the comparison.
//
// Returns querier_dto.Expression which holds the parsed
// comparison expression, or nil if no postfix operator
// was found.
func (p *parser) parsePostfixComparisonSuffix(left querier_dto.Expression) querier_dto.Expression {
	if p.isKeyword("IN") {
		return p.parseInListSuffix(left)
	}
	if p.isKeyword("BETWEEN") {
		return p.parseBetweenSuffix(left)
	}
	for _, keyword := range []string{"LIKE", "GLOB", "REGEXP", "MATCH"} {
		if p.matchKeyword(keyword) {
			right := p.parseBitwiseExpression()
			return &querier_dto.ComparisonExpression{Operator: keyword, Left: left, Right: right}
		}
	}
	return nil
}

// maybeNegate wraps an expression in a NOT unary
// operator if the negated flag is set, otherwise returns
// the expression unchanged.
//
// Takes negated (bool) which indicates whether a NOT
// keyword was consumed before the expression.
// Takes expression (querier_dto.Expression) which holds
// the expression to optionally negate.
//
// Returns querier_dto.Expression which holds the
// original expression or a NOT-wrapped version.
func (*parser) maybeNegate(negated bool, expression querier_dto.Expression) querier_dto.Expression {
	if negated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: expression}
	}
	return expression
}

func (p *parser) parseComparisonOperator(left querier_dto.Expression) querier_dto.Expression {
	operator := p.advance().value
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: right}
}

func (p *parser) parseBitwiseExpression() querier_dto.Expression {
	left := p.parseJSONAccessExpression()

	for p.current().kind == tokenOperator &&
		(p.current().value == "&" || p.current().value == "|" ||
			p.current().value == "<<" || p.current().value == ">>") {
		operator := p.advance().value
		right := p.parseJSONAccessExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

func (p *parser) parseJSONAccessExpression() querier_dto.Expression {
	left := p.parseAddExpression()

	for p.current().kind == tokenArrow || p.current().kind == tokenDoubleArrow {
		operator := p.advance().value
		right := p.parseAddExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

func (p *parser) parseIsNullSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	negated := p.matchKeyword(keywordNOT)
	p.matchKeyword("NULL")
	return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
}

func (p *parser) parseInListSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
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

func (p *parser) parseAddExpression() querier_dto.Expression {
	left := p.parseMulExpression()

	for p.current().kind == tokenOperator &&
		(p.current().value == "+" || p.current().value == "-" || p.current().value == "||") {
		operator := p.advance().value
		right := p.parseMulExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

func (p *parser) parseMulExpression() querier_dto.Expression {
	left := p.parseUnaryExpression()

	for p.isMulOperator() {
		operator := p.advance().value
		right := p.parseUnaryExpression()
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

func (p *parser) parseUnaryExpression() querier_dto.Expression {
	if p.current().kind == tokenOperator && (p.current().value == "-" || p.current().value == "+" || p.current().value == "~") {
		operator := p.advance().value
		inner := p.parsePrimaryExpression()
		if _, ok := inner.(*querier_dto.LiteralExpression); ok {
			return p.parseCollateSuffix(inner)
		}
		return p.parseCollateSuffix(&querier_dto.UnaryOpExpression{Operator: operator, Operand: inner})
	}
	return p.parseCollateSuffix(p.parsePrimaryExpression())
}

func (p *parser) parseCollateSuffix(expression querier_dto.Expression) querier_dto.Expression {
	if p.matchKeyword(keywordCOLLATE) {
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
	}
	return expression
}

func (p *parser) parsePrimaryExpression() querier_dto.Expression {
	tok := p.current()

	switch tok.kind {
	case tokenNumber:
		p.advance()
		if strings.Contains(tok.value, ".") || strings.Contains(tok.value, "e") || strings.Contains(tok.value, "E") {
			return &querier_dto.LiteralExpression{TypeName: "real"}
		}
		return &querier_dto.LiteralExpression{TypeName: "integer"}

	case tokenString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "text"}

	case tokenBlobLiteral:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "blob"}

	case tokenQuestionMark, tokenNumberedParam, tokenNamedParam:
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextUnknown, nil, nil)
		return &querier_dto.UnknownExpression{}

	case tokenLeftParen:
		if p.isSubqueryStart() {
			return p.parseScalarSubquery()
		}
		p.advance()
		inner := p.parseExpression()
		if p.current().kind == tokenRightParen {
			p.advance()
		}
		return inner

	case tokenIdentifier:
		return p.parseIdentifierExpression()

	default:
		p.advance()
		return &querier_dto.UnknownExpression{}
	}
}

func (p *parser) parseIdentifierExpression() querier_dto.Expression {
	upper := strings.ToUpper(p.current().value)

	if upper == "NULL" {
		p.advance()
		return nil
	}

	if upper == "TRUE" || upper == "FALSE" {
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "boolean"}
	}

	if upper == "CAST" {
		return p.parseCastExpression()
	}

	if upper == "COALESCE" {
		return p.parseCoalesceExpression()
	}

	if upper == "CASE" {
		return p.parseCaseExpression()
	}

	if upper == keywordEXISTS {
		p.advance()
		return p.parseExistsSubquery()
	}

	name := p.advance().value

	if p.current().kind == tokenDot {
		p.advance()
		if p.current().kind == tokenStar {
			p.advance()
			return &querier_dto.UnknownExpression{}
		}
		if p.current().kind == tokenIdentifier {
			columnName := p.advance().value
			return &querier_dto.ColumnRefExpression{
				TableAlias: name,
				ColumnName: columnName,
			}
		}
	}

	if p.current().kind == tokenLeftParen {
		return p.parseFunctionCall(name)
	}

	return &querier_dto.ColumnRefExpression{
		TableAlias: "",
		ColumnName: name,
	}
}

func (p *parser) parseFunctionCall(functionName string) querier_dto.Expression {
	p.advance()

	arguments := p.parseFunctionArguments()

	result := &querier_dto.FunctionCallExpression{
		FunctionName: strings.ToLower(functionName),
		Schema:       "",
		Arguments:    arguments,
	}

	if p.matchKeyword("FILTER") {
		if p.current().kind == tokenLeftParen {
			p.advance()
			p.matchKeyword(keywordWHERE)
			result.FilterExpression = p.parseExpression()
			if p.current().kind == tokenRightParen {
				p.advance()
			}
		}
	}

	if p.isKeyword("OVER") {
		return p.parseWindowSuffix(result)
	}

	return result
}

func (p *parser) parseFunctionArguments() []querier_dto.Expression {
	var arguments []querier_dto.Expression

	if p.current().kind == tokenStar {
		p.advance()
		if p.current().kind == tokenRightParen {
			p.advance()
		}
		return arguments
	}

	p.matchKeyword("DISTINCT")
	p.matchKeyword("ALL")

	if p.current().kind == tokenRightParen {
		p.advance()
		return arguments
	}

	parameterCountBefore := p.parameterCount

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isKeyword(keywordORDER) {
			break
		}
		arguments = append(arguments, p.parseExpression())
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	p.parseFunctionOrderByClause()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore &&
			p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown {
			p.parameterRefs[i].Context = querier_dto.ParameterContextFunctionArgument
		}
	}

	return arguments
}

func (p *parser) parseFunctionOrderByClause() {
	if !p.matchKeyword(keywordORDER) {
		return
	}
	p.matchKeyword(keywordBY)
	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseExpression()
		p.matchKeyword("ASC")
		p.matchKeyword("DESC")
		p.matchKeyword("NULLS")
		p.matchKeyword("FIRST")
		p.matchKeyword("LAST")
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

func (p *parser) parseWindowSuffix(innerFunction *querier_dto.FunctionCallExpression) querier_dto.Expression {
	p.advance()

	if p.current().kind != tokenLeftParen {
		return &querier_dto.WindowFunctionExpression{Function: innerFunction}
	}
	p.advance()

	if p.matchKeyword("PARTITION") {
		p.matchKeyword("BY")
		p.parseExpression()
		for p.current().kind == tokenComma {
			p.advance()
			p.parseExpression()
		}
	}

	if p.matchKeyword(keywordORDER) {
		p.matchKeyword("BY")
		p.parseExpression()
		p.matchKeyword("ASC")
		p.matchKeyword("DESC")
		p.matchKeyword("NULLS")
		p.matchKeyword("FIRST")
		p.matchKeyword("LAST")
		for p.current().kind == tokenComma {
			p.advance()
			p.parseExpression()
			p.matchKeyword("ASC")
			p.matchKeyword("DESC")
			p.matchKeyword("NULLS")
			p.matchKeyword("FIRST")
			p.matchKeyword("LAST")
		}
	}

	if p.isAnyKeyword("ROWS", "RANGE", "GROUPS") {
		p.skipWindowFrame()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return &querier_dto.WindowFunctionExpression{Function: innerFunction}
}

func (p *parser) skipWindowFrame() {
	p.advance()
	if p.matchKeyword("BETWEEN") {
		p.skipFrameBound()
		p.matchKeyword(keywordAND)
		p.skipFrameBound()
	} else {
		p.skipFrameBound()
	}
	p.matchKeyword("EXCLUDE")
	p.matchKeyword("CURRENT")
	p.matchKeyword("ROW")
	p.matchKeyword(keywordGROUP)
	p.matchKeyword("TIES")
	p.isAnyKeyword("NO", "OTHERS")
	if p.matchKeyword("NO") {
		p.matchKeyword("OTHERS")
	}
}

func (p *parser) skipFrameBound() {
	if p.matchKeyword("CURRENT") {
		p.matchKeyword("ROW")
		return
	}
	if p.matchKeyword("UNBOUNDED") {
		p.matchKeyword("PRECEDING")
		p.matchKeyword("FOLLOWING")
		return
	}
	p.parseExpression()
	p.matchKeyword("PRECEDING")
	p.matchKeyword("FOLLOWING")
}

func (p *parser) parseScalarSubquery() querier_dto.Expression {
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

	return &querier_dto.ScalarSubqueryExpression{InnerQuery: innerAnalysis}
}

func (p *parser) parseExistsSubquery() querier_dto.Expression {
	innerTokens, collectError := p.collectParenthesised()
	if collectError != nil {
		return &querier_dto.ExistsExpression{}
	}

	childParser := newParser(innerTokens)
	childParser.parameterCount = p.parameterCount
	innerAnalysis, analyseError := childParser.analyseSelect()
	if analyseError != nil {
		return &querier_dto.ExistsExpression{}
	}
	p.parameterCount = childParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, childParser.parameterRefs...)

	return &querier_dto.ExistsExpression{InnerQuery: innerAnalysis}
}

func (p *parser) parseCastExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()

	parameterCountBefore := p.parameterCount
	inner := p.parseExpression()

	p.matchKeyword(keywordAS)

	typeName := ""
	if p.current().kind == tokenIdentifier {
		typeName = p.advance().value
		for p.current().kind == tokenIdentifier {
			typeName += " " + p.advance().value
		}
	}

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	if typeName != "" && p.parameterCount == parameterCountBefore+1 {
		lastIndex := len(p.parameterRefs) - 1
		if lastIndex >= 0 {
			p.parameterRefs[lastIndex].Context = querier_dto.ParameterContextCast
			p.parameterRefs[lastIndex].CastType = new(normaliseTypeName(typeName))
		}
	}

	return &querier_dto.CastExpression{
		TypeName: strings.ToLower(typeName),
		Inner:    inner,
	}
}

func (p *parser) parseCoalesceExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()

	var arguments []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		arguments = append(arguments, p.parseExpression())
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return &querier_dto.CoalesceExpression{
		Arguments: arguments,
	}
}

func (p *parser) parseCaseExpression() querier_dto.Expression {
	p.advance()

	if !p.isKeyword("WHEN") {
		p.parseExpression()
	}

	var branches []querier_dto.CaseWhenBranch
	for p.matchKeyword("WHEN") {
		condition := p.parseExpression()
		p.matchKeyword("THEN")
		result := p.parseExpression()
		branches = append(branches, querier_dto.CaseWhenBranch{Condition: condition, Result: result})
	}

	expression := &querier_dto.CaseWhenExpression{Branches: branches}

	if p.matchKeyword("ELSE") {
		expression.ElseResult = p.parseExpression()
	}

	p.matchKeyword("END")
	return expression
}
