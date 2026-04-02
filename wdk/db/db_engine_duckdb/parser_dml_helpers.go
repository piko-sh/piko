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

func (p *parser) consumeInlineCast() *querier_dto.SQLType {
	p.advance()
	typeName := p.parseCastTypeName()
	return new(normaliseTypeName(typeName, nil))
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
	case "=", "<>", "!=", "<", ">", "<=", ">=":
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

	p.appendTypeArrayBrackets(&builder)

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

func isMultiWordTypeKeyword(upper string) bool {
	switch upper {
	case "VARYING", "PRECISION", "WITHOUT", keywordWITH, keywordTIME, keywordZONE,
		"CHARACTER", "DOUBLE", "TIMESTAMP", "INTERVAL":
		return true
	}
	return false
}
