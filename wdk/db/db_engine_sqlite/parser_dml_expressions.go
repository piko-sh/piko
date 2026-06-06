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

var (
	// expressionTerminators lists keywords that terminate a free-form SQL expression.
	//
	// JOIN-introducing keywords are included so that a JOIN's ON expression stops at the
	// next JOIN rather than swallowing the subsequent JOIN clause, which would prevent the
	// next joined table from being added to the scope chain and leave its projected columns
	// typed as `any`. Mirrors the same fix applied to the postgres / mysql / duckdb engines.
	expressionTerminators = map[string]bool{
		keywordGROUP: true, keywordHAVING: true, keywordORDER: true, keywordLIMIT: true,
		keywordUNION: true, keywordINTERSECT: true, keywordEXCEPT: true,
		keywordRETURNING: true, keywordSET: true, keywordON: true,
		keywordFROM: true, keywordWHERE: true,
		keywordJOIN: true, "INNER": true, "LEFT": true, "RIGHT": true,
		"FULL": true, "CROSS": true, "NATURAL": true,
	}

	// joinKeywordTerminators is the subset of expressionTerminators whose tokens are also
	// valid SQL function names.
	//
	// LEFT(), RIGHT() and similar tokens are functions when immediately followed by an
	// opening parenthesis rather than JOIN starters, so the terminator check must skip them
	// to avoid prematurely ending the surrounding expression and quietly dropping any
	// parameters that follow.
	joinKeywordTerminators = map[string]bool{
		"INNER": true, "LEFT": true, "RIGHT": true, "FULL": true, "CROSS": true, "NATURAL": true,
	}

	// whereExpressionTerminators is the terminator set used for WHERE / HAVING expressions.
	//
	// It deliberately omits the JOIN-introducing keywords (JOIN, INNER, LEFT, RIGHT, FULL,
	// CROSS, NATURAL) because those are all legal unquoted column names in SQLite, and a
	// WHERE/HAVING predicate never legitimately starts a JOIN at top level. Including them
	// treated a column named `left`/`right`/`inner` etc. as a clause boundary and quietly
	// dropped every parameter that followed it. The ON-clause scan keeps the JOIN keywords
	// via expressionTerminators.
	whereExpressionTerminators = map[string]bool{
		keywordGROUP: true, keywordHAVING: true, keywordORDER: true, keywordLIMIT: true,
		keywordUNION: true, keywordINTERSECT: true, keywordEXCEPT: true,
		keywordRETURNING: true, keywordSET: true, keywordON: true,
		keywordFROM: true, keywordWHERE: true,
	}
)

// parseWhereExpression skips a WHERE / HAVING predicate, stopping at the first
// clause-level terminator that does not double as a column name.
func (p *parser) parseWhereExpression() {
	p.parseExpressionUntilTerminator(whereExpressionTerminators)
}

// parseJoinConditionExpression skips a JOIN ON predicate. A join condition is genuinely
// followed by the next JOIN, so the JOIN-introducing keywords remain terminators here
// (via expressionTerminators); the handleExpressionTerminator function-call exemption
// still distinguishes LEFT()/RIGHT() calls from JOIN starters.
func (p *parser) parseJoinConditionExpression() {
	p.parseExpressionUntilTerminator(expressionTerminators)
}

// parseExpressionUntilTerminator advances over tokens until reaching a terminator from
// the supplied set at depth zero.
//
// Takes terminators (map[string]bool) which is the set of clause-level keywords that end
// the scan.
func (p *parser) parseExpressionUntilTerminator(terminators map[string]bool) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()

		nextDepth, stop, handled := p.handleExpressionParen(tok, depth)
		if stop {
			break
		}
		if handled {
			depth = nextDepth
			continue
		}

		if depth == 0 && isExpressionTerminator(tok, terminators) {
			if p.handleExpressionTerminator(tok) {
				break
			}
			continue
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// handleExpressionParen processes a parenthesis token within an expression scan. Returns
// the next depth value, whether to break the outer loop (unmatched right paren at depth
// zero), and whether the token was a parenthesis at all (and thus already consumed).
//
// Takes tok (token) which is the token under consideration.
// Takes depth (int) which is the current parenthesis nesting depth.
//
// Returns nextDepth (int) which is the updated nesting depth.
// Returns stop (bool) which is true when the outer loop should break.
// Returns handled (bool) which is true when the token was consumed here.
func (p *parser) handleExpressionParen(tok token, depth int) (nextDepth int, stop bool, handled bool) {
	if tok.kind == tokenLeftParen {
		if p.isSubqueryStart() {
			if innerAnalysis, ok := p.analyseSubqueryBody(); ok {
				p.predicateSubqueries = append(p.predicateSubqueries, innerAnalysis)
			}
			return depth, false, true
		}
		p.advance()
		return depth + 1, false, true
	}
	if tok.kind == tokenRightParen {
		if depth == 0 {
			return depth, true, true
		}
		p.advance()
		return depth - 1, false, true
	}
	return depth, false, false
}

// handleExpressionTerminator distinguishes a real JOIN keyword from a function call
// sharing the same name (LEFT(), RIGHT(), etc.). Consumes the identifier when it is
// followed by an opening parenthesis, returning false so the caller continues; otherwise
// returns true so the caller breaks out of the expression scan.
//
// Takes tok (token) which is the candidate terminator token.
//
// Returns bool which is true when the caller should break.
func (p *parser) handleExpressionTerminator(tok token) bool {
	if joinKeywordTerminators[strings.ToUpper(tok.value)] && p.peek().kind == tokenLeftParen {
		p.advance()
		return false
	}
	return true
}

// isExpressionTerminator reports whether the token is a clause-level keyword that ends an
// expression scan.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true when the token ends the expression.
func isExpressionTerminator(tok token, terminators map[string]bool) bool {
	return tok.kind == tokenIdentifier && terminators[strings.ToUpper(tok.value)]
}

// handleParameterInExpression registers a parameter token seen while scanning an
// expression, attaching context information when known.
//
// When the flat scan classifies the placeholder as a function argument it also records
// the enclosing function name and the 0-based argument ordinal on the freshly registered
// parameter, so the analyser can look the function up and back-propagate the declared
// argument type. The AST function-call path records the same metadata via
// markFunctionArgumentParameters; this branch covers the flat WHERE/ON scan path (for
// example LENGTH(?) > 0) which never descends into parseFunctionArguments.
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

	if context == querier_dto.ParameterContextFunctionArgument {
		if name, ordinal, ok := p.enclosingFunctionArgument(paramPosition); ok {
			lastIndex := len(p.parameterRefs) - 1
			if lastIndex >= 0 && p.parameterRefs[lastIndex].EnclosingFunctionName == "" {
				p.parameterRefs[lastIndex].EnclosingFunctionName = name
				p.parameterRefs[lastIndex].ArgumentOrdinal = ordinal
			}
		}
	}
}

// isComparisonOperator reports whether the supplied operator is a SQL comparison
// operator.
//
// Takes operator (string) which is the operator literal.
//
// Returns bool which is true for =, <>, !=, <, >, <=, and >=.
func isComparisonOperator(operator string) bool {
	switch operator {
	case "=", "<>", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// parseExpression parses a full SQL expression at the lowest precedence.
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

// parseOrExpression parses left-associative OR expressions.
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parseAndExpression parses left-associative AND expressions.
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

// parseNotExpression parses an optional NOT prefix followed by a comparison expression.
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parseComparisonExpression parses comparison operators including IS, IN, BETWEEN, LIKE,
// GLOB, REGEXP, and MATCH.
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parsePostfixComparisonSuffix attempts to parse a postfix comparison operator (IN,
// BETWEEN, LIKE, GLOB, REGEXP, MATCH) that follows the left operand. This is called after
// optionally consuming a NOT keyword so that constructs like NOT IN and NOT LIKE are
// handled.
//
// Takes left (querier_dto.Expression) which holds the already-parsed left operand of the
// comparison.
//
// Returns querier_dto.Expression which holds the parsed comparison expression, or nil if
// no postfix operator was found.
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

// maybeNegate wraps an expression in a NOT unary operator if the negated flag is set,
// otherwise returns the expression unchanged.
//
// Takes negated (bool) which indicates whether a NOT keyword was consumed before the
// expression.
// Takes expression (querier_dto.Expression) which holds the expression to optionally
// negate.
//
// Returns querier_dto.Expression which holds the original expression or a NOT-wrapped
// version.
func (*parser) maybeNegate(negated bool, expression querier_dto.Expression) querier_dto.Expression {
	if negated {
		return &querier_dto.UnaryOpExpression{Operator: keywordNOT, Operand: expression}
	}
	return expression
}

// parseComparisonOperator parses an infix comparison operator and its right-hand operand.
//
// Takes left (querier_dto.Expression) which holds the left operand already parsed.
//
// Returns querier_dto.Expression which is the parsed comparison.
func (p *parser) parseComparisonOperator(left querier_dto.Expression) querier_dto.Expression {
	operator := p.advance().value
	right := p.parseBitwiseExpression()
	return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: right}
}

// parseBitwiseExpression parses left-associative bitwise operators (&, |, <<, >>).
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parseJSONAccessExpression parses left-associative JSON arrow operators (-> and ->>).
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseJSONAccessExpression() querier_dto.Expression {
	left := p.parseAddExpression()

	for p.current().kind == tokenArrow || p.current().kind == tokenDoubleArrow {
		operator := p.advance().value
		right := p.parseAddExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// parseIsNullSuffix parses the IS [NOT] NULL suffix after a left operand, or the SQLite
// null-safe equality form col IS [NOT] <operand> when the operand is not NULL.
//
// IS NULL / IS NOT NULL return an IsNullExpression as before. For the null-safe equality
// form (for example col IS ?) the operand is parsed so the token stream stays
// synchronised, and any placeholder it introduces is tagged with the LHS column reference
// exactly as col = ? is, so a col IS ? nested inside a function argument, subquery or
// CASE is typed from the column rather than left untyped. This mirrors the flat-scan IS
// handling.
//
// Takes left (querier_dto.Expression) which holds the inner expression.
//
// Returns querier_dto.Expression which is the parsed IsNull or comparison expression.
func (p *parser) parseIsNullSuffix(left querier_dto.Expression) querier_dto.Expression {
	p.advance()
	negated := p.matchKeyword(keywordNOT)
	if p.matchKeyword("NULL") {
		return &querier_dto.IsNullExpression{Inner: left, Negated: negated}
	}

	parameterCountBefore := p.parameterCount
	right := p.parseBitwiseExpression()

	if columnExpression, ok := left.(*querier_dto.ColumnRefExpression); ok {
		columnReference := &querier_dto.ColumnReference{
			TableAlias: columnExpression.TableAlias,
			ColumnName: columnExpression.ColumnName,
		}
		for i := range p.parameterRefs {
			if p.parameterRefs[i].Number > parameterCountBefore && p.parameterRefs[i].ColumnReference == nil {
				p.parameterRefs[i].Context = querier_dto.ParameterContextComparison
				p.parameterRefs[i].ColumnReference = columnReference
			}
		}
	}

	operator := "IS"
	if negated {
		operator = "IS NOT"
	}
	return &querier_dto.ComparisonExpression{Operator: operator, Left: left, Right: right}
}

// parseInListSuffix parses the IN (...) suffix and tags any parameters inside it with the
// IN-list context.
//
// Takes left (querier_dto.Expression) which holds the inner expression.
//
// Returns querier_dto.Expression which is the parsed InList expression.
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

// parseBetweenSuffix parses a BETWEEN low AND high suffix and tags parameters inside it
// with the BETWEEN context.
//
// Takes left (querier_dto.Expression) which holds the inner expression.
//
// Returns querier_dto.Expression which is the parsed Between expression.
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

// parseParenthesisedExpressionList parses a parenthesised, comma separated list of
// expressions.
//
// Returns []querier_dto.Expression which holds the parsed list values.
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

// parseAddExpression parses left-associative additive operators (+, -, and the string
// concatenation operator ||).
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parseMulExpression parses left-associative multiplicative operators (*, /, %).
//
// Returns querier_dto.Expression which is the parsed expression tree.
func (p *parser) parseMulExpression() querier_dto.Expression {
	left := p.parseUnaryExpression()

	for p.isMulOperator() {
		operator := p.advance().value
		right := p.parseUnaryExpression()
		left = &querier_dto.BinaryOpExpression{Operator: operator, Left: left, Right: right}
	}

	return left
}

// isMulOperator reports whether the current token is a multiplicative operator.
//
// Returns bool which is true for *, /, and %.
func (p *parser) isMulOperator() bool {
	if p.current().kind == tokenStar {
		return true
	}
	return p.current().kind == tokenOperator &&
		(p.current().value == "/" || p.current().value == "%")
}

// parseUnaryExpression parses an optional unary prefix (+, -, ~) and the following
// primary expression.
//
// Returns querier_dto.Expression which is the parsed expression tree.
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

// parseCollateSuffix consumes an optional COLLATE clause after an expression.
//
// Takes expression (querier_dto.Expression) which is the prior expression to attach to.
//
// Returns querier_dto.Expression which is the original expression.
func (p *parser) parseCollateSuffix(expression querier_dto.Expression) querier_dto.Expression {
	if p.matchKeyword(keywordCOLLATE) {
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
	}
	return expression
}

// parsePrimaryExpression parses the highest-precedence expression atom: literals,
// parameters, identifiers, function calls, and subqueries.
//
// Returns querier_dto.Expression which is the parsed atom.
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

// parseIdentifierExpression parses an identifier-led primary expression covering keywords
// (NULL, TRUE, CAST, etc.) and column references.
//
// Returns querier_dto.Expression which is the parsed atom.
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

// parseFunctionCall parses a function-call expression including any FILTER clause and
// OVER window suffix.
//
// Takes functionName (string) which is the already-consumed function identifier.
//
// Returns querier_dto.Expression which is the parsed call or window expression.
func (p *parser) parseFunctionCall(functionName string) querier_dto.Expression {
	p.advance()

	loweredName := strings.ToLower(functionName)
	arguments := p.parseFunctionArguments(loweredName)

	result := &querier_dto.FunctionCallExpression{
		FunctionName: loweredName,
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

// parseFunctionArguments parses the parenthesised arguments of a function call and tags
// any newly-seen parameters as function arguments.
//
// Each top-level argument expression consumes one 0-based ordinal slot (a literal, a
// column or a sub-expression still advances the slot), so a placeholder sitting directly
// in the call's argument list is tagged with the enclosing function name and the slot it
// occupied. A placeholder nested inside an inner call was already tagged by that inner
// call's parseFunctionArguments, so the Context==ParameterContextUnknown guard leaves it
// untouched and the inner metadata is not clobbered.
//
// Takes loweredName (string) which is the lower-cased enclosing function name recorded on
// each placeholder so the analyser can look the function up and back-propagate the
// declared argument type.
//
// Returns []querier_dto.Expression which holds the parsed argument expressions.
func (p *parser) parseFunctionArguments(loweredName string) []querier_dto.Expression {
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

	argumentOrdinal := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isKeyword(keywordORDER) {
			break
		}
		refsCountBefore := len(p.parameterRefs)
		arguments = append(arguments, p.parseExpression())
		p.markFunctionArgumentParameters(refsCountBefore, loweredName, argumentOrdinal)
		argumentOrdinal++
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	p.parseFunctionOrderByClause()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return arguments
}

// markFunctionArgumentParameters tags every placeholder registered while parsing one
// top-level function argument with the function-argument context, the enclosing function
// name and the argument's 0-based ordinal slot. Placeholders that an inner call already
// tagged are skipped via the Context==ParameterContextUnknown guard, so a nested call's
// metadata is preserved.
//
// The boundary is the parameterRefs slice length rather than the parameter-count
// watermark: references are always appended, so refsCountBefore..end is exactly the set
// added by this argument, which is correct even for out-of-order numbered placeholders
// (for example ?3 then ?1) where the watermark would not advance.
//
// Takes refsCountBefore (int) which is len(parameterRefs) before the argument was parsed.
// Takes loweredName (string) which is the lower-cased enclosing function name.
// Takes argumentOrdinal (int) which is the 0-based slot the argument occupies.
func (p *parser) markFunctionArgumentParameters(refsCountBefore int, loweredName string, argumentOrdinal int) {
	for i := refsCountBefore; i < len(p.parameterRefs); i++ {
		if p.parameterRefs[i].Context == querier_dto.ParameterContextUnknown {
			p.parameterRefs[i].Context = querier_dto.ParameterContextFunctionArgument
			p.parameterRefs[i].EnclosingFunctionName = loweredName
			p.parameterRefs[i].ArgumentOrdinal = argumentOrdinal
		}
	}
}

// parseFunctionOrderByClause consumes the optional ORDER BY clause that may appear inside
// an aggregate function call.
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

// parseWindowSuffix parses the OVER (...) window specification that may follow an
// aggregate or window function call.
//
// Takes innerFunction (*querier_dto.FunctionCallExpression) which is the inner function
// to wrap.
//
// Returns querier_dto.Expression which is the window function expression.
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

// skipWindowFrame consumes a ROWS, RANGE, or GROUPS window frame clause.
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

// skipFrameBound consumes one bound of a window frame clause (CURRENT ROW, UNBOUNDED
// PRECEDING, expression PRECEDING, etc.).
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

// parseScalarSubquery parses a parenthesised SELECT used as a scalar expression, merging
// its parameter references back into the parent parser.
//
// Returns querier_dto.Expression which is the parsed scalar subquery, or
// UnknownExpression on failure.
func (p *parser) parseScalarSubquery() querier_dto.Expression {
	innerAnalysis, ok := p.analyseSubqueryBody()
	if !ok {
		return &querier_dto.UnknownExpression{}
	}
	return &querier_dto.ScalarSubqueryExpression{InnerQuery: innerAnalysis}
}

// parseExistsSubquery parses an EXISTS (SELECT ...) expression, merging its parameter
// references back into the parent parser.
//
// Returns querier_dto.Expression which is the parsed EXISTS expression, empty on failure.
func (p *parser) parseExistsSubquery() querier_dto.Expression {
	innerAnalysis, ok := p.analyseSubqueryBody()
	if !ok {
		return &querier_dto.ExistsExpression{}
	}
	return &querier_dto.ExistsExpression{InnerQuery: innerAnalysis}
}

// analyseSubqueryBody collects a parenthesised subquery, analyses it in a child parser
// that inherits the parameter and depth state, and splices the child's parameter results
// back.
//
// Shared by the scalar and EXISTS subquery parsers.
//
// Returns *querier_dto.RawQueryAnalysis which describes the inner subquery.
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

// parseCastExpression parses a CAST(expression AS type) expression and tags the inner
// parameter, when present, with the CAST target type.
//
// Returns querier_dto.Expression which is the parsed CAST expression, or
// UnknownExpression on failure.
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

// parseCoalesceExpression parses a COALESCE(...) expression.
//
// Returns querier_dto.Expression which is the parsed Coalesce expression, or
// UnknownExpression on failure.
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

// parseCaseExpression parses a CASE expression with branches and else.
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
