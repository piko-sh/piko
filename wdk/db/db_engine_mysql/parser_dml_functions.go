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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseFunctionCall(name string, schema string) querier_dto.Expression {
	p.advance()
	loweredName := strings.ToLower(name)

	if p.current().kind == tokenStar || p.current().kind == tokenRightParen {
		return p.parseFunctionCallNoArgs(loweredName, schema)
	}

	if handler := p.specialFunctionHandler(loweredName); handler != nil {
		return handler(loweredName, schema)
	}

	return p.parseFunctionCallWithArgs(loweredName, schema)
}

func (p *parser) parseFunctionCallNoArgs(loweredName string, schema string) querier_dto.Expression {
	if p.current().kind == tokenStar {
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	result := &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
	}
	return p.parseFunctionSuffix(result)
}

func (p *parser) parseFunctionCallWithArgs(loweredName string, schema string) querier_dto.Expression {
	p.matchKeyword("DISTINCT")
	p.matchKeyword(keywordALL)

	parameterCountBefore := p.parameterCount
	arguments := p.parseFunctionArguments()
	p.parseFunctionOrderByClause()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.markParametersAsFunctionArguments(parameterCountBefore)

	result := &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
		Arguments:    arguments,
	}

	return p.parseFunctionSuffix(result)
}

func (p *parser) parseFunctionArguments() []querier_dto.Expression {
	var arguments []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isAnyKeyword(keywordORDER, keywordLIMIT, keywordSEPARATOR) {
			break
		}
		arguments = append(arguments, p.parseExpression())
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	return arguments
}

func (p *parser) parseFunctionOrderByClause() {
	if !p.matchKeyword(keywordORDER) {
		return
	}
	p.matchKeyword(keywordBY)
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isKeyword(keywordSEPARATOR) {
			break
		}
		p.parseExpression()
		p.matchKeyword(keywordASC)
		p.matchKeyword(keywordDESC)
		p.matchKeyword(keywordNULLS)
		p.matchKeyword(keywordFIRST)
		p.matchKeyword(keywordLAST)
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

func (p *parser) markParametersAsFunctionArguments(parameterCountBefore int) {
	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore &&
			p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown {
			p.parameterRefs[i].Context = querier_dto.ParameterContextFunctionArgument
		}
	}
}

func (p *parser) parseFunctionSuffix(result *querier_dto.FunctionCallExpression) querier_dto.Expression {
	if p.isKeyword("OVER") {
		return p.parseWindowSuffix(result)
	}

	return result
}

func (p *parser) parseWindowSuffix(innerFunction *querier_dto.FunctionCallExpression) querier_dto.Expression {
	p.advance()

	if p.current().kind == tokenIdentifier && p.peek().kind != tokenLeftParen &&
		!p.isAnyKeyword("PARTITION", keywordORDER, "ROWS", "RANGE") {
		p.advance()
		return &querier_dto.WindowFunctionExpression{Function: innerFunction}
	}

	if p.current().kind != tokenLeftParen {
		return &querier_dto.WindowFunctionExpression{Function: innerFunction}
	}
	p.advance()

	p.parseWindowSpec()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return &querier_dto.WindowFunctionExpression{Function: innerFunction}
}

func (p *parser) parseWindowSpec() {
	if p.current().kind == tokenIdentifier &&
		!p.isAnyKeyword("PARTITION", keywordORDER, "ROWS", "RANGE") &&
		p.current().kind != tokenRightParen {
		p.advance()
	}

	if p.matchKeyword("PARTITION") {
		p.matchKeyword(keywordBY)
		p.parseExpression()
		for p.current().kind == tokenComma {
			p.advance()
			p.parseExpression()
		}
	}

	if p.matchKeyword(keywordORDER) {
		p.matchKeyword(keywordBY)
		p.parseExpression()
		p.matchKeyword(keywordASC)
		p.matchKeyword(keywordDESC)
		p.matchKeyword(keywordNULLS)
		p.matchKeyword(keywordFIRST)
		p.matchKeyword(keywordLAST)
		for p.current().kind == tokenComma {
			p.advance()
			p.parseExpression()
			p.matchKeyword(keywordASC)
			p.matchKeyword(keywordDESC)
			p.matchKeyword(keywordNULLS)
			p.matchKeyword(keywordFIRST)
			p.matchKeyword(keywordLAST)
		}
	}

	if p.isAnyKeyword("ROWS", "RANGE") {
		p.skipWindowFrame()
	}
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
}

func (p *parser) skipFrameBound() {
	if p.matchKeyword(keywordCURRENT) {
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

type specialFunctionParser func(string, string) querier_dto.Expression

func (p *parser) specialFunctionHandler(loweredName string) specialFunctionParser {
	switch loweredName {
	case "trim":
		return p.parseTrimFunction
	case "extract":
		return p.parseExtractFunction
	case "group_concat":
		return p.parseGroupConcatFunction
	case "convert":
		return p.parseConvertFunction
	default:
		return nil
	}
}

func (p *parser) parseTrimFunction(loweredName string, schema string) querier_dto.Expression {
	parameterCountBefore := p.parameterCount

	if p.isAnyKeyword("LEADING", "TRAILING", "BOTH") {
		p.advance()
		if !p.isKeyword(keywordFROM) {
			p.parseExpression()
		}
		p.matchKeyword(keywordFROM)
	}

	arguments := p.parseFunctionArguments()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.markParametersAsFunctionArguments(parameterCountBefore)

	return &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
		Arguments:    arguments,
	}
}

func (p *parser) parseExtractFunction(loweredName string, schema string) querier_dto.Expression {
	if p.current().kind == tokenIdentifier {
		p.advance()
	}
	p.matchKeyword(keywordFROM)

	parameterCountBefore := p.parameterCount
	arguments := p.parseFunctionArguments()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.markParametersAsFunctionArguments(parameterCountBefore)

	return &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
		Arguments:    arguments,
	}
}

func (p *parser) parseGroupConcatFunction(loweredName string, schema string) querier_dto.Expression {
	p.matchKeyword("DISTINCT")

	parameterCountBefore := p.parameterCount
	var arguments []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isAnyKeyword(keywordORDER, keywordSEPARATOR) {
			break
		}
		arguments = append(arguments, p.parseExpression())
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	p.parseFunctionOrderByClause()

	if p.matchKeyword(keywordSEPARATOR) {
		p.parseExpression()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.markParametersAsFunctionArguments(parameterCountBefore)

	result := &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
		Arguments:    arguments,
	}

	return p.parseFunctionSuffix(result)
}

func (p *parser) parseConvertFunction(loweredName string, schema string) querier_dto.Expression {
	parameterCountBefore := p.parameterCount
	inner := p.parseExpression()

	if p.matchKeyword(keywordAS) || p.matchKeyword(keywordUSING) {
		typeName := ""
		if p.current().kind == tokenIdentifier {
			typeName = p.parseCastTypeName()
		}

		if p.current().kind == tokenRightParen {
			p.advance()
		}

		p.annotateCastParameter(typeName, parameterCountBefore)

		return &querier_dto.CastExpression{
			TypeName: strings.ToLower(typeName),
			Inner:    inner,
		}
	}

	if p.current().kind == tokenComma {
		p.advance()
		typeName := ""
		if p.current().kind == tokenIdentifier {
			typeName = p.parseCastTypeName()
		}

		if p.current().kind == tokenRightParen {
			p.advance()
		}

		p.annotateCastParameter(typeName, parameterCountBefore)

		return &querier_dto.CastExpression{
			TypeName: strings.ToLower(typeName),
			Inner:    inner,
		}
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
		Schema:       schema,
		Arguments:    []querier_dto.Expression{inner},
	}
}

func (p *parser) parseCastFunctionExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()

	parameterCountBefore := p.parameterCount
	inner := p.parseExpression()

	p.matchKeyword(keywordAS)

	typeName := p.parseCastTargetTypeName()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.annotateCastParameter(typeName, parameterCountBefore)

	return &querier_dto.CastExpression{
		TypeName: strings.ToLower(typeName),
		Inner:    inner,
	}
}

func (p *parser) parseConvertExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()
	return p.parseConvertFunction("convert", "")
}

func (p *parser) parseCastTargetTypeName() string {
	typeName := ""
	if p.current().kind == tokenIdentifier {
		typeName = p.advance().value
		for p.current().kind == tokenIdentifier &&
			!p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT) {
			if p.current().kind == tokenRightParen {
				break
			}
			typeName += " " + p.advance().value
		}
	}

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	return typeName
}

func (p *parser) annotateCastParameter(typeName string, parameterCountBefore int) {
	if typeName == "" || p.parameterCount != parameterCountBefore+1 {
		return
	}
	lastIndex := len(p.parameterRefs) - 1
	if lastIndex < 0 {
		return
	}
	p.parameterRefs[lastIndex].Context = querier_dto.ParameterContextCast
	p.parameterRefs[lastIndex].CastType = new(normaliseTypeName(typeName, nil))
}

func (p *parser) parseCoalesceExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()

	referenceCountBefore := len(p.parameterRefs)
	arguments, firstColumnReference := p.parseCoalesceArguments()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.annotateCoalesceParameters(firstColumnReference, referenceCountBefore)

	return &querier_dto.CoalesceExpression{
		Arguments: arguments,
	}
}

func (p *parser) parseCoalesceArguments() ([]querier_dto.Expression, *querier_dto.ColumnReference) {
	var arguments []querier_dto.Expression
	var firstColumnReference *querier_dto.ColumnReference
	for !p.atEnd() && p.current().kind != tokenRightParen {
		argument := p.parseExpression()
		arguments = append(arguments, argument)
		if columnRefExpression, ok := argument.(*querier_dto.ColumnRefExpression); ok && firstColumnReference == nil {
			firstColumnReference = &querier_dto.ColumnReference{
				TableAlias: columnRefExpression.TableAlias,
				ColumnName: columnRefExpression.ColumnName,
			}
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	return arguments, firstColumnReference
}

func (p *parser) annotateCoalesceParameters(firstColumnReference *querier_dto.ColumnReference, referenceCountBefore int) {
	if firstColumnReference == nil {
		return
	}
	for i := referenceCountBefore; i < len(p.parameterRefs); i++ {
		if p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown &&
			p.parameterRefs[i].ColumnReference == nil &&
			p.parameterRefs[i].CastType == nil {
			p.parameterRefs[i].ColumnReference = firstColumnReference
			p.parameterRefs[i].Context = querier_dto.ParameterContextComparison
		}
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
