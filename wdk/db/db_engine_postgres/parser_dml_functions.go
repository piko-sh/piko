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

// parseFunctionCall parses a function-call expression after the opening `(` has been
// peeked, dispatching to the no-arg or with-args form.
//
// Takes name (string) which is the function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the function call expression.
func (p *parser) parseFunctionCall(name string, schema string) querier_dto.Expression {
	p.advance()
	loweredName := strings.ToLower(name)

	if p.current().kind == tokenStar || p.current().kind == tokenRightParen {
		return p.parseFunctionCallNoArgs(loweredName, schema)
	}

	return p.parseFunctionCallWithArgs(loweredName, schema)
}

// parseFunctionCallNoArgs consumes a function call whose argument list is empty or a
// single `*`.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the function call expression.
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

// parseFunctionCallWithArgs consumes the argument list of a function call including
// optional DISTINCT / ALL modifiers and ORDER BY clause.
//
// Takes loweredName (string) which is the lower-cased function name.
// Takes schema (string) which is the optional schema qualifier.
//
// Returns querier_dto.Expression which is the function call expression.
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

// parseFunctionArguments parses the comma-separated argument list of a function call,
// stopping at ORDER, LIMIT, or `)`.
//
// Returns []querier_dto.Expression which is the argument list.
func (p *parser) parseFunctionArguments() []querier_dto.Expression {
	var arguments []querier_dto.Expression
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isAnyKeyword(keywordORDER, keywordLIMIT) {
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

// parseFunctionOrderByClause consumes an ORDER BY clause that may appear inside an
// aggregate function call.
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

// markParametersAsFunctionArguments tags any parameter added since parameterCountBefore
// with the FunctionArgument context when it still has the Unknown context.
//
// Takes parameterCountBefore (int) which is the parameterCount snapshot taken before
// parsing the argument list.
func (p *parser) markParametersAsFunctionArguments(parameterCountBefore int) {
	for i := range p.parameterRefs {
		if p.parameterRefs[i].Number > parameterCountBefore &&
			p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown {
			p.parameterRefs[i].Context = querier_dto.ParameterContextFunctionArgument
		}
	}
}

// parseWithinGroupClause consumes a WITHIN GROUP (ORDER BY ...) clause that follows
// certain ordered-set aggregates.
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

// parseFilterClause consumes a FILTER (WHERE ...) clause that may follow an aggregate
// function call.
//
// Takes result (*querier_dto.FunctionCallExpression) which is the function call to attach
// the filter expression to.
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

// parseFunctionSuffix consumes optional WITHIN GROUP, FILTER, and OVER suffixes on a
// function call.
//
// Takes result (*querier_dto.FunctionCallExpression) which is the inner function call.
//
// Returns querier_dto.Expression which is the wrapped expression, or the original
// function call when no suffix applies.
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

// parseWindowSuffix consumes an OVER (window-spec) or OVER named-window clause that turns
// a function call into a window function expression.
//
// Takes innerFunction (*querier_dto.FunctionCallExpression) which is the inner aggregate
// or window function.
//
// Returns querier_dto.Expression which is the window function expression.
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

// parseWindowSpec consumes the body of a window specification, covering an optional
// reference name, PARTITION BY, ORDER BY, and frame clause.
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

// skipWindowFrame consumes a window frame clause including BETWEEN ... AND, frame bounds,
// and any EXCLUDE modifier.
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

// skipFrameBound consumes a single window frame bound such as CURRENT ROW, UNBOUNDED
// PRECEDING/FOLLOWING, or `expression PRECEDING`.
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

// parseCastFunctionExpression parses a CAST(expression AS type) form and annotates any
// parameter in the expression with the target type.
//
// Returns querier_dto.Expression which is the cast expression.
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

// parseCastTargetTypeName collects the type-name tokens that follow `CAST(... AS` until
// the closing paren or an unrelated keyword.
//
// Returns string which is the joined type name including any `[]` array suffixes.
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

// appendArrayBrackets appends the array subscript suffix to typeName once for each
// balanced `[]` pair at the current position.
//
// Takes typeName (string) which is the base type name.
//
// Returns string which is typeName with any array suffixes appended.
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

// annotateCastParameter tags the most recently registered parameter with the cast target
// type when the cast contained exactly one new parameter.
//
// Takes typeName (string) which is the normalised cast type name.
// Takes parameterCountBefore (int) which is the parameter count captured before parsing
// the cast expression.
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

// parseCoalesceExpression parses a COALESCE(...) call and propagates any column reference
// among its arguments to parameter siblings.
//
// Returns querier_dto.Expression which is the COALESCE expression.
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

// parseCoalesceArguments parses the comma-separated argument list of a COALESCE call,
// returning the first encountered column reference.
//
// Returns []querier_dto.Expression which is the argument list.
// Returns *querier_dto.ColumnReference which is the first column reference found in the
// arguments, or nil when none was present.
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

// annotateCoalesceParameters tags any parameter added inside the COALESCE call with the
// first column reference seen among its arguments.
//
// Takes firstColumnReference (*querier_dto.ColumnReference) which is the column inferred
// from the COALESCE arguments.
// Takes referenceCountBefore (int) which is the parameterRefs length captured before
// parsing the call.
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

// parseCaseExpression parses a CASE expression covering both the simple and searched
// forms with optional ELSE branch.
//
// Returns querier_dto.Expression which is the CASE expression.
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

// parseExistsSubquery parses an EXISTS (subquery) expression by recursing into a child
// parser and importing its parameter state.
//
// Returns querier_dto.Expression which is the EXISTS expression.
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

// parseScalarSubquery parses a parenthesised scalar subquery by recursing into a child
// parser and importing its parameter state.
//
// Returns querier_dto.Expression which is the scalar subquery expression.
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

// parseArrayExpression parses an ARRAY[...] literal or ARRAY(subquery) constructor.
//
// Returns querier_dto.Expression which is the array expression or a scalar subquery for
// the ARRAY(subquery) form.
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
