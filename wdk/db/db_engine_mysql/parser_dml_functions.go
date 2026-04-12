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

// parseFunctionCall parses a function call beginning at the open paren.
//
// Takes name (string) which is the function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed function call.
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

// parseFunctionCallNoArgs handles functions called with `*` or no args.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed function call.
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

// parseFunctionCallWithArgs parses a function call with one or more arguments.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed function call.
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

// parseFunctionArguments parses a comma-separated argument list.
//
// Returns []querier_dto.Expression which is the parsed argument list.
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

// parseFunctionOrderByClause consumes an ORDER BY inside a function call.
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

// markParametersAsFunctionArguments retags parameters parsed since parameterCountBefore
// as function argument contexts.
//
// Takes parameterCountBefore (int) which is the parameter count recorded before the
// function arguments were parsed.
func (p *parser) markParametersAsFunctionArguments(parameterCountBefore int) {
	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore &&
			p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown {
			p.parameterRefs[i].Context = querier_dto.ParameterContextFunctionArgument
		}
	}
}

// parseFunctionSuffix wraps a function call in an OVER window if present.
//
// Takes result (*querier_dto.FunctionCallExpression) which is the inner function call.
//
// Returns querier_dto.Expression which is either result or a WindowFunctionExpression
// wrapping it.
func (p *parser) parseFunctionSuffix(result *querier_dto.FunctionCallExpression) querier_dto.Expression {
	if p.isKeyword("OVER") {
		return p.parseWindowSuffix(result)
	}

	return result
}

// parseWindowSuffix parses the OVER (...) clause of a window function.
//
// Takes innerFunction (*querier_dto.FunctionCallExpression) which is the function the
// window applies to.
//
// Returns querier_dto.Expression which is the wrapped window expression.
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

// parseWindowSpec parses PARTITION BY, ORDER BY, and frame clauses.
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

// skipWindowFrame consumes a ROWS or RANGE window frame definition.
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

// skipFrameBound consumes a single window frame bound expression.
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

// specialFunctionParser is the signature of a custom parser for a function whose argument
// grammar diverges from a plain comma list.
type specialFunctionParser func(string, string) querier_dto.Expression

// specialFunctionHandler returns the special parser for a function name.
//
// Takes loweredName (string) which is the lower-cased function name.
//
// Returns specialFunctionParser which is the matched parser, or nil when the function
// uses the default argument grammar.
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

// parseTrimFunction parses TRIM with its LEADING/TRAILING/BOTH grammar.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed TRIM call.
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

// parseExtractFunction parses EXTRACT(field FROM expression).
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed EXTRACT call.
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

// parseGroupConcatFunction parses GROUP_CONCAT with its SEPARATOR clause.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed GROUP_CONCAT call.
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

// parseConvertFunction parses CONVERT(expr, type) or CONVERT(expr USING).
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the parsed CONVERT call.
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

// parseCastFunctionExpression parses CAST(expr AS type).
//
// Returns querier_dto.Expression which is the parsed CAST expression, or an unknown
// expression when the opening paren is missing.
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

// parseConvertExpression dispatches to parseConvertFunction.
//
// Returns querier_dto.Expression which is the parsed CONVERT call.
func (p *parser) parseConvertExpression() querier_dto.Expression {
	p.advance()
	if p.current().kind != tokenLeftParen {
		return &querier_dto.UnknownExpression{}
	}
	p.advance()
	return p.parseConvertFunction("convert", "")
}

// parseCastTargetTypeName parses the target type of a CAST expression.
//
// Returns string which is the assembled type name.
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

// annotateCastParameter tags the most recent parameter with a cast type.
//
// Takes typeName (string) which is the raw type name parsed from the CAST.
// Takes parameterCountBefore (int) which is the parameter count recorded before the inner
// expression was parsed.
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

// parseCoalesceExpression parses a COALESCE(...) call with annotation.
//
// Annotates parameters with the first column reference seen so the expected column types
// can be inferred later.
//
// Returns querier_dto.Expression which is the parsed COALESCE expression.
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

// parseCoalesceArguments parses COALESCE arguments and tracks the first column reference
// encountered.
//
// Returns []querier_dto.Expression which is the parsed argument list.
// Returns *querier_dto.ColumnReference which is the first column seen, or nil when no
// argument is a bare column.
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

// annotateCoalesceParameters tags unannotated COALESCE parameters with the column
// reference of the first non-parameter argument.
//
// Takes firstColumnReference (*querier_dto.ColumnReference) which is the reference to
// apply to unannotated parameters.
// Takes referenceCountBefore (int) which is the parameterRefs length before COALESCE
// arguments were parsed.
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

// parseCaseExpression parses a CASE WHEN ... END expression.
//
// Returns querier_dto.Expression which is the parsed CASE expression.
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
