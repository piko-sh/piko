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

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseExpression() querier_dto.Expression {
	return p.parseOrExpression()
}

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

func (p *parser) parseRegexpSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: "REGEXP", Left: left, Right: right}
}

func (p *parser) parseSoundsLikeSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	p.matchKeyword("LIKE")
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: "SOUNDS LIKE", Left: left, Right: right}
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
		(p.current().value == "&" || p.current().value == "|" || p.current().value == "^" ||
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

func (p *parser) isMulOperator() bool {
	if p.current().kind == tokenStar {
		return true
	}
	return p.current().kind == tokenOperator &&
		(p.current().value == "/" || p.current().value == "%" || p.current().value == "DIV" || p.current().value == "MOD")
}

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

func (p *parser) parseCollateExpression() querier_dto.Expression {
	left := p.parseSubscriptExpression()

	if p.matchKeyword(keywordCOLLATE) {
		if p.current().kind == tokenIdentifier || p.current().kind == tokenString {
			p.advance()
		}
	}

	return left
}

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

func (p *parser) parseNumberLiteral(tok token) querier_dto.Expression {
	p.advance()
	if strings.Contains(tok.value, ".") || strings.Contains(tok.value, "e") || strings.Contains(tok.value, "E") {
		return &querier_dto.LiteralExpression{TypeName: "double"}
	}
	return &querier_dto.LiteralExpression{TypeName: "int"}
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
	"CURRENT_USER": {}, "LOCALTIME": {}, "LOCALTIMESTAMP": {},
	"UTC_TIMESTAMP": {}, "UTC_DATE": {}, "UTC_TIME": {},
}

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

func (p *parser) parseNullLiteral() querier_dto.Expression {
	p.advance()
	return nil
}

func (p *parser) parseBooleanLiteral() querier_dto.Expression {
	p.advance()
	return &querier_dto.LiteralExpression{TypeName: "tinyint"}
}

var intervalFieldKeywords = map[string]struct{}{
	"YEAR": {}, "MONTH": {}, "DAY": {},
	"HOUR": {}, "MINUTE": {}, "SECOND": {},
	"MICROSECOND": {}, "WEEK": {}, "QUARTER": {},
	"YEAR_MONTH": {}, "DAY_HOUR": {}, "DAY_MINUTE": {},
	"DAY_SECOND": {}, "DAY_MICROSECOND": {},
	"HOUR_MINUTE": {}, "HOUR_SECOND": {}, "HOUR_MICROSECOND": {},
	"MINUTE_SECOND": {}, "MINUTE_MICROSECOND": {},
	"SECOND_MICROSECOND": {},
}

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

func (p *parser) parseRowConstructorExpression() querier_dto.Expression {
	p.advance()
	p.skipRowConstructor()
	return &querier_dto.UnknownExpression{}
}

func (p *parser) parseIfExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.FunctionCallExpression{FunctionName: "if"}
	}
	return p.parseFunctionCall("if", "")
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

func (p *parser) handleParameterInExpression() {
	paramPosition := p.position
	parameterToken := p.current()

	context, columnRef, castType := p.resolveParameterContext(paramPosition)

	p.advance()

	p.registerParameterFromToken(parameterToken, context, columnRef, castType)
}

func (p *parser) resolveParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType) {
	context, columnRef := p.resolveContextFromPrecedingOperator(paramPosition)
	if context != querier_dto.ParameterContextUnknown {
		return context, columnRef, nil
	}
	return p.detectParameterContext(paramPosition)
}

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

func (p *parser) findEnclosingParen(position int) int {
	depth := 0
	for i := position - 1; i >= 0; i-- {
		switch p.tokens[i].kind {
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

func (p *parser) extractColumnReferenceBeforeIN(inPosition int) *querier_dto.ColumnReference {
	if inPosition < 1 {
		return nil
	}
	return p.extractColumnReference(inPosition - 1)
}

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

func (p *parser) isKeywordAt(position int, keywords ...string) bool {
	if position >= len(p.tokens) || p.tokens[position].kind != tokenIdentifier {
		return false
	}
	return slices.Contains(keywords, strings.ToUpper(p.tokens[position].value))
}

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

func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=", "<=>":
		return true
	}
	return false
}

func isArithmeticOperator(operator string) bool {
	switch operator {
	case "+", "-", "/", "%":
		return true
	}
	return false
}

func (p *parser) extractColumnReferenceFromParenthesised(rightParenPosition int) *querier_dto.ColumnReference {
	leftParenPosition := p.findMatchingLeftParen(rightParenPosition)
	if leftParenPosition < 0 {
		return nil
	}
	return p.scanForColumnReference(leftParenPosition+1, rightParenPosition)
}

func (p *parser) findMatchingLeftParen(rightParenPosition int) int {
	depth := 0
	for i := rightParenPosition; i >= 0; i-- {
		switch p.tokens[i].kind {
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

func (p *parser) scanForColumnReference(startPosition int, endPosition int) *querier_dto.ColumnReference {
	for j := startPosition; j < endPosition; j++ {
		reference := p.extractColumnReference(j)
		if reference != nil {
			return reference
		}
	}
	return nil
}

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

func (p *parser) appendSchemaQualifier(builder *strings.Builder) {
	if p.current().kind == tokenDot && p.peek().kind == tokenIdentifier {
		p.advance()
		builder.WriteByte('.')
		builder.WriteString(p.advance().value)
	}
}

func (p *parser) appendMultiWordTypeKeywords(builder *strings.Builder) {
	for p.current().kind == tokenIdentifier {
		if !isMultiWordTypeKeyword(strings.ToUpper(p.current().value)) {
			break
		}
		builder.WriteByte(' ')
		builder.WriteString(p.advance().value)
	}
}

func isMultiWordTypeKeyword(upper string) bool {
	switch upper {
	case "PRECISION", keywordUNSIGNED, "VARYING", "CHARACTER", "DOUBLE":
		return true
	}
	return false
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
