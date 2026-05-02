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

// parseFunctionCall parses a function invocation after its name is consumed.
//
// Takes name (string) which is the original-case function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the resulting function call expression,
// optionally wrapped in a window or filter suffix.
func (p *parser) parseFunctionCall(name string, schema string) querier_dto.Expression {
	p.advance()
	loweredName := strings.ToLower(name)

	if p.current().kind == tokenStar || p.current().kind == tokenRightParen {
		return p.parseFunctionCallNoArgs(loweredName, schema)
	}

	return p.parseFunctionCallWithArgs(loweredName, schema)
}

// parseFunctionCallNoArgs handles a function call with no arguments or star.
//
// Takes loweredName (string) which is the lowercase function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the resulting function call expression with any
// trailing suffix applied.
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

// parseFunctionCallWithArgs parses a function call that has arguments.
//
// Takes loweredName (string) which is the lowercase function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the resulting function call expression with any
// trailing suffix applied.
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

// parseFunctionArguments parses a comma-separated list of function arguments.
//
// Returns []querier_dto.Expression which contains the parsed arguments including any
// lambda expressions.
func (p *parser) parseFunctionArguments() []querier_dto.Expression {
	var arguments []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isAnyKeyword(keywordORDER, keywordLIMIT) {
			break
		}
		if lambda := p.tryParseLambda(); lambda != nil {
			arguments = append(arguments, lambda)
		} else {
			arguments = append(arguments, p.parseExpression())
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	return arguments
}

// tryParseLambda parses a single or multi-parameter lambda when present.
//
// Returns querier_dto.Expression which is the parsed lambda, or nil when the current
// tokens do not form a lambda.
func (p *parser) tryParseLambda() querier_dto.Expression {
	if p.current().kind == tokenIdentifier && p.peek().kind == tokenArrow {
		parameterName := p.advance().value
		p.advance()
		body := p.parseFunctionArgumentExpression()
		return &querier_dto.LambdaExpression{
			Parameters: []string{parameterName},
			Body:       body,
		}
	}

	if p.current().kind == tokenLeftParen && p.looksLikeMultiParamLambda() {
		return p.parseMultiParamLambda()
	}

	return nil
}

// looksLikeMultiParamLambda peeks for a parenthesised lambda parameter list.
//
// Returns bool which is true when the upcoming tokens form (a, b, ...) ->.
func (p *parser) looksLikeMultiParamLambda() bool {
	savedPosition := p.position
	defer func() { p.position = savedPosition }()

	p.advance()
	for !p.atEnd() {
		if p.current().kind != tokenIdentifier {
			return false
		}
		p.advance()
		if p.current().kind == tokenRightParen {
			p.advance()
			return p.current().kind == tokenArrow
		}
		if p.current().kind != tokenComma {
			return false
		}
		p.advance()
	}
	return false
}

// parseMultiParamLambda parses a multi-parameter lambda expression.
//
// Returns querier_dto.Expression which is the parsed lambda expression.
func (p *parser) parseMultiParamLambda() querier_dto.Expression {
	p.advance()
	var parameters []string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenIdentifier {
			parameters = append(parameters, p.advance().value)
		}
		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	p.advance()
	body := p.parseFunctionArgumentExpression()
	return &querier_dto.LambdaExpression{
		Parameters: parameters,
		Body:       body,
	}
}

// parseFunctionArgumentExpression parses one expression used as a function arg.
//
// Returns querier_dto.Expression which is the parsed argument expression.
func (p *parser) parseFunctionArgumentExpression() querier_dto.Expression {
	return p.parseExpression()
}

// parseFunctionOrderByClause consumes a trailing ORDER BY clause in arguments.
func (p *parser) parseFunctionOrderByClause() {
	if !p.matchKeyword(keywordORDER) {
		return
	}
	p.matchKeyword(keywordBY)
	for !p.atEnd() && p.current().kind != tokenRightParen {
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

// markParametersAsFunctionArguments tags new bind parameters as function args.
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

// parseWithinGroupClause parses a WITHIN GROUP (ORDER BY ...) clause.
func (p *parser) parseWithinGroupClause() {
	p.matchKeyword(keywordGROUP)
	if p.current().kind != tokenLeftParen {
		return
	}
	p.advance()
	if p.matchKeyword(keywordORDER) {
		p.matchKeyword(keywordBY)
		for !p.atEnd() && p.current().kind != tokenRightParen {
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
	if p.current().kind == tokenRightParen {
		p.advance()
	}
}

// parseFilterClause parses a FILTER (WHERE ...) aggregate suffix.
//
// Takes result (*querier_dto.FunctionCallExpression) which receives the parsed filter
// expression.
func (p *parser) parseFilterClause(result *querier_dto.FunctionCallExpression) {
	if p.current().kind != tokenLeftParen {
		return
	}
	p.advance()
	p.matchKeyword(keywordWHERE)
	result.FilterExpression = p.parseExpression()
	if p.current().kind == tokenRightParen {
		p.advance()
	}
}

// parseFunctionSuffix parses WITHIN GROUP, FILTER and OVER suffixes.
//
// Takes result (*querier_dto.FunctionCallExpression) which is the function call to
// decorate with suffix clauses.
//
// Returns querier_dto.Expression which is the original call or a window function wrapper.
func (p *parser) parseFunctionSuffix(result *querier_dto.FunctionCallExpression) querier_dto.Expression {
	if p.matchKeyword("WITHIN") {
		p.parseWithinGroupClause()
	}

	if p.matchKeyword("FILTER") {
		p.parseFilterClause(result)
	}

	if p.isKeyword("OVER") {
		return p.parseWindowSuffix(result)
	}

	return result
}

// parseWindowSuffix parses an OVER named window or inline window spec.
//
// Takes innerFunction (*querier_dto.FunctionCallExpression) which is the function call
// being windowed.
//
// Returns querier_dto.Expression which is the resulting window function expression.
func (p *parser) parseWindowSuffix(innerFunction *querier_dto.FunctionCallExpression) querier_dto.Expression {
	p.advance()

	if p.current().kind == tokenIdentifier && p.peek().kind != tokenLeftParen &&
		!p.isAnyKeyword("PARTITION", keywordORDER, keywordROWS, "RANGE", "GROUPS") {
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

// parseWindowSpec parses PARTITION BY, ORDER BY and frame clauses.
func (p *parser) parseWindowSpec() {
	if p.current().kind == tokenIdentifier &&
		!p.isAnyKeyword("PARTITION", keywordORDER, keywordROWS, "RANGE", "GROUPS") &&
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

	if p.isAnyKeyword(keywordROWS, "RANGE", "GROUPS") {
		p.skipWindowFrame()
	}
}

// skipWindowFrame consumes a ROWS, RANGE or GROUPS frame clause.
func (p *parser) skipWindowFrame() {
	p.advance()
	if p.matchKeyword("BETWEEN") {
		p.skipFrameBound()
		p.matchKeyword(keywordAND)
		p.skipFrameBound()
	} else {
		p.skipFrameBound()
	}
	if p.matchKeyword("EXCLUDE") {
		p.matchKeyword(keywordCURRENT)
		p.matchKeyword(keywordROW)
		p.matchKeyword(keywordGROUP)
		p.matchKeyword("TIES")
		if p.matchKeyword("NO") {
			p.matchKeyword("OTHERS")
		}
	}
}

// skipFrameBound consumes a single frame boundary keyword and operand.
func (p *parser) skipFrameBound() {
	if p.matchKeyword(keywordCURRENT) {
		p.matchKeyword(keywordROW)
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

// parseCastFunctionExpression parses a CAST(expr AS type) function form.
//
// Returns querier_dto.Expression which is the resulting cast expression.
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

// parseCastTargetTypeName parses the type name on the right side of CAST.
//
// Returns string which is the assembled type name including array suffixes.
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

	typeName = p.appendArrayBrackets(typeName)
	return typeName
}

// appendArrayBrackets appends [] suffixes for each empty bracket pair seen.
//
// Takes typeName (string) which is the base type name.
//
// Returns string which is the type name with array subscript suffixes appended.
func (p *parser) appendArrayBrackets(typeName string) string {
	for p.current().kind == tokenLeftBracket {
		p.advance()
		if p.current().kind == tokenRightBracket {
			typeName += arraySubscriptSuffix
			p.advance()
		}
	}
	return typeName
}

// annotateCastParameter records the cast target type on the inner parameter.
//
// Takes typeName (string) which is the parsed cast target type name.
// Takes parameterCountBefore (int) which is the parameter count recorded before the cast
// operand was parsed.
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

// parseCoalesceExpression parses a COALESCE(...) function call.
//
// Returns querier_dto.Expression which is the resulting coalesce expression or an unknown
// expression on syntax mismatch.
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

// parseCoalesceArguments parses comma-separated COALESCE arguments.
//
// Returns []querier_dto.Expression which is the parsed argument list.
// Returns *querier_dto.ColumnReference which is the first column reference argument, or
// nil when no column reference was found.
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

// annotateCoalesceParameters tags COALESCE parameters with the first column.
//
// Takes firstColumnReference (*querier_dto.ColumnReference) which is the first column
// reference seen among the arguments.
// Takes referenceCountBefore (int) which is the parameter reference count recorded before
// COALESCE arguments were parsed.
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

// parseCaseExpression parses a CASE expression with branches and else.
//
// Returns querier_dto.Expression which is the resulting CASE expression.
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

// parseExistsSubquery parses an EXISTS (...) subquery predicate.
//
// Returns querier_dto.Expression which is the resulting EXISTS expression.
func (p *parser) parseExistsSubquery() querier_dto.Expression {
	innerTokens, collectError := p.collectParenthesised()
	if collectError != nil {
		return &querier_dto.ExistsExpression{}
	}

	childParser := newParser(innerTokens)
	childParser.parameterCount = p.parameterCount
	childParser.analysisDepth = p.analysisDepth
	childParser.expressionDepth = p.expressionDepth
	childParser.maxParseDepth = p.maxParseDepth
	innerAnalysis, analyseError := childParser.analyseSelect()
	if analyseError != nil {
		return &querier_dto.ExistsExpression{}
	}
	p.parameterCount = childParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, childParser.parameterRefs...)

	return &querier_dto.ExistsExpression{InnerQuery: innerAnalysis}
}

// parseScalarSubquery parses a parenthesised scalar subquery.
//
// Returns querier_dto.Expression which is the resulting scalar subquery expression or an
// unknown expression on parse failure.
func (p *parser) parseScalarSubquery() querier_dto.Expression {
	innerAnalysis, ok := p.analyseSubqueryBody()
	if !ok {
		return &querier_dto.UnknownExpression{}
	}
	return &querier_dto.ScalarSubqueryExpression{InnerQuery: innerAnalysis}
}

// analyseSubqueryBody collects a parenthesised subquery, analyses it in a child parser
// that inherits the parameter and depth state, and splices the child's parameter results
// back into this parser. Shared by the scalar-subquery expression parser and the
// WHERE/HAVING predicate scan.
//
// Returns *querier_dto.RawQueryAnalysis which is the inner SELECT analysis (nil on
// failure).
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

// parseArrayExpression parses ARRAY[...] or ARRAY(subquery) constructors.
//
// Returns querier_dto.Expression which is an UnknownExpression placeholder or a scalar
// subquery.
func (p *parser) parseArrayExpression() querier_dto.Expression {
	p.advance()

	if p.current().kind == tokenLeftBracket {
		p.advance()
		for !p.atEnd() && p.current().kind != tokenRightBracket {
			p.parseExpression()
			if p.current().kind != tokenComma {
				break
			}
			p.advance()
		}
		if p.current().kind == tokenRightBracket {
			p.advance()
		}
		return &querier_dto.UnknownExpression{}
	}

	if p.current().kind == tokenLeftParen {
		return p.parseScalarSubquery()
	}

	return &querier_dto.UnknownExpression{}
}
