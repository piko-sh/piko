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

// parseWhereClause consumes the predicate of a WHERE or HAVING clause.
func (p *parser) parseWhereClause() {
	p.parseExpressionUntilTerminator()
}

var (
	// expressionTerminatorKeywords lists keywords that terminate a free-form SQL expression.
	//
	// The expressions are WHERE, ON, and HAVING. JOIN-introducing keywords are included so
	// that a JOIN's ON expression stops at the next JOIN rather than swallowing the
	// subsequent JOIN clause (which would prevent the next joined table from being added to
	// the scope chain and leave its projected columns typed as `any`).
	expressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		"LOCK": {}, "PROCEDURE": {},
		keywordJOIN: {}, "INNER": {}, "LEFT": {}, "RIGHT": {},
		"FULL": {}, "CROSS": {}, "NATURAL": {},
	}

	// joinKeywordTerminators is the subset of expressionTerminatorKeywords whose tokens are
	// also valid SQL function names.
	//
	// Examples include LEFT() and RIGHT(). When such a token is immediately followed by an
	// opening parenthesis it is a function call rather than a JOIN starter, so the
	// terminator check must skip it to avoid prematurely ending the surrounding expression
	// and dropping any parameters that follow.
	joinKeywordTerminators = map[string]struct{}{
		"INNER": {}, "LEFT": {}, "RIGHT": {}, "FULL": {}, "CROSS": {}, "NATURAL": {},
	}

	// whereExpressionTerminatorKeywords is the terminator set used for WHERE and HAVING
	// expressions.
	//
	// It deliberately omits the JOIN-introducing keywords because those are legal unquoted
	// column names in MySQL and a WHERE or HAVING predicate never starts a JOIN at top
	// level. Including them treated a column named `left`, `right`, or `inner` as a clause
	// boundary and dropped every parameter that followed it. The ON-clause scan keeps the
	// JOIN keywords via expressionTerminatorKeywords.
	whereExpressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		"LOCK": {}, "PROCEDURE": {},
	}
)

// parseExpressionUntilTerminator skips a WHERE or HAVING predicate up to its terminator
// set.
func (p *parser) parseExpressionUntilTerminator() {
	p.skipTokensUntilTerminatorSet(whereExpressionTerminatorKeywords)
}

// parseJoinConditionExpression skips a JOIN ON predicate, keeping the JOIN-introducing
// keywords as terminators (via expressionTerminatorKeywords) so the condition stops at
// the next JOIN.
func (p *parser) parseJoinConditionExpression() {
	p.skipTokensUntilTerminatorSet(expressionTerminatorKeywords)
}

// skipTokensUntilTerminatorSet walks tokens, balancing parens, until a member of
// terminators appears at depth zero or the stream ends.
//
// Takes terminators (map[string]struct{}) which lists upper-case keyword terminators to
// stop at.
func (p *parser) skipTokensUntilTerminatorSet(terminators map[string]struct{}) {
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
		if depth == 0 && p.isKeywordTerminator(tok, terminators) {
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
	if _, isJoinKeyword := joinKeywordTerminators[strings.ToUpper(tok.value)]; isJoinKeyword && p.peek().kind == tokenLeftParen {
		p.advance()
		return false
	}
	return true
}

// isKeywordTerminator reports whether tok is an identifier whose keyword is in
// terminators.
//
// Takes tok (token) which is the token to inspect.
// Takes terminators (map[string]struct{}) which lists the upper-case keyword terminators.
//
// Returns bool which is true when the token's keyword is a terminator.
func (*parser) isKeywordTerminator(tok token, terminators map[string]struct{}) bool {
	if tok.kind != tokenIdentifier {
		return false
	}
	_, ok := terminators[strings.ToUpper(tok.value)]
	return ok
}

// parseGroupByClause parses a GROUP BY column list with optional ROLLUP.
//
// Returns []querier_dto.ColumnReference which lists the grouped columns.
func (p *parser) parseGroupByClause() []querier_dto.ColumnReference {
	var columns []querier_dto.ColumnReference

	for {
		column, ok := p.parseGroupByColumn()
		if ok {
			columns = append(columns, column)
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	p.matchKeyword(keywordWITH)
	p.matchKeyword(keywordROLLUP)

	return columns
}

// parseGroupByColumn parses a single GROUP BY column reference.
//
// Returns querier_dto.ColumnReference which is the parsed column.
// Returns bool which is true when a column was parsed.
func (p *parser) parseGroupByColumn() (querier_dto.ColumnReference, bool) {
	if p.current().kind != tokenIdentifier {
		p.advance()
		return querier_dto.ColumnReference{}, false
	}
	first := p.advance().value
	if p.current().kind != tokenDot {
		return querier_dto.ColumnReference{ColumnName: first}, true
	}
	p.advance()
	if p.current().kind != tokenIdentifier {
		return querier_dto.ColumnReference{ColumnName: first}, true
	}
	second := p.advance().value
	return querier_dto.ColumnReference{TableAlias: first, ColumnName: second}, true
}

var (
	// orderByTerminators lists keywords that end an ORDER BY clause.
	orderByTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, "WINDOW": {}, "LOCK": {}, "PROCEDURE": {},
	}
)

// parseOrderByList consumes the body of an ORDER BY clause.
func (p *parser) parseOrderByList() {
	p.skipTokensUntilTerminatorSet(orderByTerminators)
}

// consumeParameterOrAdvance registers a parameter token or advances past the current
// token when it is not a parameter.
//
// A single-token read is sufficient here. Unlike PostgreSQL, DuckDB, and SQLite, MySQL
// forbids an expression in LIMIT or OFFSET (only an integer literal or a ? placeholder is
// accepted), so a compound value such as COALESCE(?, 100) cannot occur and there is no
// nested parameter to scan for.
//
// Takes context (querier_dto.ParameterContext) which annotates the registered parameter
// when one is consumed.
func (p *parser) consumeParameterOrAdvance(context querier_dto.ParameterContext) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, context, nil, nil)
		return
	}
	p.advance()
}

// parseLimitOffset parses LIMIT and OFFSET clauses in either order.
//
// Supports the MySQL `LIMIT a, b` shorthand by swapping the resulting parameter contexts.
func (p *parser) parseLimitOffset() {
	if p.matchKeyword(keywordOFFSET) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	}

	if !p.matchKeyword(keywordLIMIT) {
		return
	}

	p.consumeParameterOrAdvance(querier_dto.ParameterContextLimit)

	if p.current().kind == tokenComma {
		p.advance()
		p.parameterRefs = p.swapLastTwoLimitParameters(p.parameterRefs)
		p.consumeParameterOrAdvance(querier_dto.ParameterContextLimit)
		return
	}

	if p.matchKeyword(keywordOFFSET) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	}
}

// swapLastTwoLimitParameters retags the trailing LIMIT parameter as an OFFSET to match
// the `LIMIT offset, count` MySQL form.
//
// Takes refs ([]querier_dto.RawParameterReference) which is the running list of parameter
// references.
//
// Returns []querier_dto.RawParameterReference which is the modified list.
func (*parser) swapLastTwoLimitParameters(refs []querier_dto.RawParameterReference) []querier_dto.RawParameterReference {
	if len(refs) == 0 {
		return refs
	}
	lastIndex := len(refs) - 1
	if refs[lastIndex].Context == querier_dto.ParameterContextLimit {
		refs[lastIndex].Context = querier_dto.ParameterContextOffset
	}
	return refs
}

// parseCompoundQuery matches a compound query operator if present.
//
// Returns querier_dto.CompoundOperator which is the matched operator, or zero when no
// compound keyword follows.
func (p *parser) parseCompoundQuery() querier_dto.CompoundOperator {
	if p.matchKeyword(keywordUNION) {
		if p.matchKeyword(keywordALL) {
			return querier_dto.CompoundUnionAll
		}
		return querier_dto.CompoundUnion
	}
	if p.matchKeyword(keywordINTERSECT) {
		p.matchKeyword(keywordALL)
		return querier_dto.CompoundIntersect
	}
	if p.matchKeyword(keywordEXCEPT) {
		p.matchKeyword(keywordALL)
		return querier_dto.CompoundExcept
	}
	return 0
}

// skipForUpdateClause consumes a trailing FOR UPDATE or LOCK IN SHARE MODE clause,
// recording whether row locking was requested.
func (p *parser) skipForUpdateClause() {
	if p.matchKeyword(keywordFOR) {
		p.hasForUpdate = true
		p.matchKeyword("UPDATE")
		p.matchKeyword("SHARE")
		if p.matchKeyword("OF") {
			p.parseForUpdateTableList()
		}
		p.matchKeyword("NOWAIT")
		p.matchKeyword("SKIP")
		p.matchKeyword("LOCKED")
		return
	}

	if p.matchKeyword("LOCK") {
		if p.matchKeyword("IN") {
			p.matchKeyword("SHARE")
			p.matchKeyword("MODE")
		}
	}
}

// parseForUpdateTableList consumes a comma-separated list of table names following a FOR
// UPDATE OF clause.
func (p *parser) parseForUpdateTableList() {
	for {
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// parseValuesClause parses an INSERT ... VALUES row list.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which are the target column names.
func (p *parser) parseValuesClause(tableName string, columnNames []string) {
	for p.current().kind == tokenLeftParen {
		p.advance()
		p.parseValuesRow(tableName, columnNames)

		if p.current().kind == tokenRightParen {
			p.advance()
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// columnRefForIndex builds a column reference for an index into columnNames.
//
// Takes tableName (string) which is the alias for the reference.
// Takes columnNames ([]string) which is the list of available column names.
// Takes index (int) which is the position within columnNames.
//
// Returns *querier_dto.ColumnReference which is the built reference, or nil when index is
// out of range.
func (*parser) columnRefForIndex(tableName string, columnNames []string, index int) *querier_dto.ColumnReference {
	if index >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[index],
	}
}

// parseValuesRow parses a single parenthesised VALUES row.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which are the target column names.
func (p *parser) parseValuesRow(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseValuesRowElement(tableName, columnNames, columnIndex)

		if p.current().kind == tokenComma {
			p.advance()
			columnIndex++
		}
	}
}

// parseValuesRowElement parses one element within a VALUES row.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which are the target column names.
// Takes columnIndex (int) which is the column position in the row.
func (p *parser) parseValuesRowElement(tableName string, columnNames []string, columnIndex int) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
		return
	}
	if p.current().kind == tokenLeftParen {
		columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
		p.scanInsertValueExpression(columnRef)
		return
	}
	if p.current().kind != tokenComma {
		p.advance()
	}
}

// scanInsertValueExpression walks the parenthesised expression starting at the current
// token, registering every parameter it encounters with the supplied columnRef.
//
// Non-parameter tokens are advanced over so the cursor lands just past the matching
// closing paren. parseValuesRowElement uses it to keep INSERT parameters inside function
// calls properly typed via assignment context. Without it those parameters would be
// dropped and the emitter would default them to *any.
//
// Takes columnRef (*querier_dto.ColumnReference) which is the INSERT column the
// parenthesised expression is the value for.
func (p *parser) scanInsertValueExpression(columnRef *querier_dto.ColumnReference) {
	if p.current().kind != tokenLeftParen {
		return
	}
	p.advance()
	depth := 1
	for !p.atEnd() && depth > 0 {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
			p.advance()
		case tokenRightParen:
			depth--
			p.advance()
		default:
			if isParameterToken(tok.kind) {
				parameterToken := tok
				p.advance()
				p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
				continue
			}
			p.advance()
		}
	}
}

// skipValuesTrailingRows walks past the additional rows of a VALUES list, registering any
// parameters they contain.
func (p *parser) skipValuesTrailingRows() {
	for p.current().kind == tokenComma {
		p.advance()
		if p.current().kind == tokenLeftParen {
			p.advance()
		}
		for !p.atEnd() && p.current().kind != tokenRightParen {
			if isParameterToken(p.current().kind) {
				p.handleParameterInExpression()
				continue
			}
			p.advance()
			if p.current().kind == tokenComma {
				p.advance()
			}
		}
		if p.current().kind == tokenRightParen {
			p.advance()
		}
	}
}

var (
	// insertSourceTerminators lists keywords that end the source clause of an INSERT ...
	// SELECT statement.
	insertSourceTerminators = map[string]struct{}{
		keywordON: {}, keywordRETURNING: {},
	}
)

// parseInsertSource consumes the SELECT source body of an INSERT.
func (p *parser) parseInsertSource() {
	p.skipTokensUntilTerminatorSet(insertSourceTerminators)
}

// parseOnDuplicateKeyUpdate parses an ON DUPLICATE KEY UPDATE clause.
//
// Takes tableName (string) which is the target table name for column reference resolution
// within the SET clause.
func (p *parser) parseOnDuplicateKeyUpdate(tableName string) {
	p.matchKeyword(keywordDUPLICATE)
	p.matchKeyword(keywordKEY)

	if p.matchKeyword("UPDATE") {
		p.parseSetClause(tableName)
	}
}

// parseSetClause parses the assignment list of an UPDATE SET clause.
//
// Takes tableName (string) which is the default table for unqualified column references.
func (p *parser) parseSetClause(tableName string) {
	for {
		if p.current().kind == tokenLeftParen {
			p.parseMultiColumnSetClause(tableName)
		} else {
			p.parseSingleColumnSetClause(tableName)
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// parseSingleColumnSetClause parses a `column = value` assignment.
//
// Takes tableName (string) which is the default table when the column is not qualified.
func (p *parser) parseSingleColumnSetClause(tableName string) {
	columnName := ""
	if p.current().kind == tokenIdentifier {
		columnName = p.advance().value
	}

	if p.current().kind == tokenDot && p.peek().kind == tokenIdentifier {
		tableName = columnName
		p.advance()
		columnName = p.advance().value
	}

	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}

	var columnRef *querier_dto.ColumnReference
	if columnName != "" {
		columnRef = &querier_dto.ColumnReference{
			TableAlias: tableName,
			ColumnName: columnName,
		}
	}

	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
	}

	p.skipSetExpressionWithColumn(columnRef)
}

// parseMultiColumnSetClause parses a `(c1, c2) = (v1, v2)` assignment.
//
// Takes tableName (string) which is the default table for unqualified column references.
func (p *parser) parseMultiColumnSetClause(tableName string) {
	columnNames, _ := p.parseColumnList()

	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}

	if p.current().kind != tokenLeftParen {
		p.skipSetExpression()
		return
	}

	p.advance()
	p.parseMultiColumnSetValues(tableName, columnNames)
	if p.current().kind == tokenRightParen {
		p.advance()
	}
}

// parseMultiColumnSetValues parses the value tuple in a multi-column SET.
//
// Takes tableName (string) which is the default table for unqualified column references.
// Takes columnNames ([]string) which is the parsed LHS column list.
func (p *parser) parseMultiColumnSetValues(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if isParameterToken(p.current().kind) {
			parameterToken := p.current()
			columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
			p.advance()
			p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
		} else {
			p.skipSetExpression()
		}
		if p.current().kind == tokenComma {
			p.advance()
			columnIndex++
		}
	}
}

var (
	// setExpressionTerminators lists keywords that end a value expression in an UPDATE SET
	// clause.
	setExpressionTerminators = map[string]struct{}{
		keywordWHERE: {}, keywordFROM: {}, keywordRETURNING: {},
		keywordORDER: {}, keywordLIMIT: {},
	}
)

// skipSetExpression skips the value expression of a SET assignment.
//
// Registers any parameter tokens encountered with an unknown context.
func (p *parser) skipSetExpression() {
	p.skipSetExpressionWithColumn(nil)
}

// skipSetExpressionWithColumn walks the RHS of a single SET assignment, stopping at a
// comma or a SET-clause terminator at depth 0.
//
// Terminators include WHERE, FROM, and RETURNING. Parameters encountered inside the RHS
// are registered with the supplied setTargetRef as assignment context so they pick up the
// column's Go type. Pass nil for setTargetRef to leave value parameters with no context.
//
// A parameter that is the operand of a comparison inside the expression (for example the
// `?` in `SET col = CASE WHEN other >= ? THEN ... END`) takes its type from the compared
// column, not the assignment target: the scanner remembers the most recent `<column>
// <comparison-operator>` pair and binds the following parameter to that column. Any other
// parameter (one that contributes to the assigned value, such as `COALESCE(?, col)`)
// keeps the assignment-target column reference so it still picks up that column's Go
// type. Without this the comparison operand was wrongly typed from the SET target,
// producing a Params field whose Go type did not match the value the caller binds.
//
// Takes setTargetRef (*querier_dto.ColumnReference) which is the column on the left of
// the SET assignment; nil leaves value parameters with ParameterContextUnknown.
func (p *parser) skipSetExpressionWithColumn(setTargetRef *querier_dto.ColumnReference) {
	tableAlias := ""
	if setTargetRef != nil {
		tableAlias = setTargetRef.TableAlias
	}
	depth := 0
	var comparisonColumn *querier_dto.ColumnReference
	lastIdentifier := ""
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

		if depth == 0 && p.isSetExpressionTerminator(tok) {
			break
		}
		if isParameterToken(tok.kind) {
			p.registerSetExpressionParameter(tok, setTargetRef, comparisonColumn)
			comparisonColumn = nil
			lastIdentifier = ""
			continue
		}

		lastIdentifier, comparisonColumn = comparisonColumnForSetParameter(tok, tableAlias, lastIdentifier)
		p.advance()
	}
}

// comparisonColumnForSetParameter advances the comparison scan over a SET expression.
//
// When the previous identifier is immediately followed by a comparison operator, that
// column is reported so a following parameter is typed from it; for example the parameter
// in "WHEN attempt >= ?" is typed from attempt. Any other token clears the pending state.
//
// Takes tok (token) which is the token being observed.
// Takes tableAlias (string) which qualifies the reported column reference.
// Takes lastIdentifier (string) which is the identifier from the previous step.
//
// Returns nextIdentifier (string) carried into the next step.
// Returns comparisonColumn (*querier_dto.ColumnReference) which is the compared column,
// or nil when no identifier is followed by a comparison operator.
func comparisonColumnForSetParameter(
	tok token,
	tableAlias, lastIdentifier string,
) (nextIdentifier string, comparisonColumn *querier_dto.ColumnReference) {
	switch {
	case tok.kind == tokenIdentifier:
		return tok.value, nil
	case tok.kind == tokenOperator && isComparisonOperator(tok.value) && lastIdentifier != "":
		return "", &querier_dto.ColumnReference{TableAlias: tableAlias, ColumnName: lastIdentifier}
	default:
		return "", nil
	}
}

// registerSetExpressionParameter consumes a parameter token within a SET assignment RHS
// and attaches the column reference and context that match its position in the
// expression.
//
// When comparisonColumn is non-nil the parameter is an operand of a comparison (for
// example `WHEN attempt >= ?`), so it is recorded against that column with comparison
// context. Otherwise it is treated as contributing to the assigned value and recorded
// against the SET-target column with assignment context (or unknown context when no
// target is known).
//
// Takes parameterToken (token) which is the placeholder token to record.
// Takes setTargetRef (*querier_dto.ColumnReference) which is the SET-target column the
// assigned value is for, when known.
// Takes comparisonColumn (*querier_dto.ColumnReference) which is the column the parameter
// is compared against, or nil when it is not a comparison operand.
func (p *parser) registerSetExpressionParameter(
	parameterToken token,
	setTargetRef *querier_dto.ColumnReference,
	comparisonColumn *querier_dto.ColumnReference,
) {
	p.advance()
	if comparisonColumn != nil {
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextComparison, comparisonColumn, nil)
		return
	}
	context := querier_dto.ParameterContextUnknown
	if setTargetRef != nil {
		context = querier_dto.ParameterContextAssignment
	}
	p.registerParameterFromToken(parameterToken, context, setTargetRef, nil)
}

// isSetExpressionTerminator reports whether tok ends a SET expression.
//
// Takes tok (token) which is the candidate token.
//
// Returns bool which is true for commas or terminator keywords.
func (*parser) isSetExpressionTerminator(tok token) bool {
	if tok.kind == tokenComma {
		return true
	}
	if tok.kind == tokenIdentifier {
		_, ok := setExpressionTerminators[strings.ToUpper(tok.value)]
		return ok
	}
	return false
}
