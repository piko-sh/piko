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
	// expressionTerminatorKeywords lists keywords that end a WHERE or HAVING expression and
	// return control to the outer query parser.
	expressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		"LOCK": {}, "PROCEDURE": {},
	}
)

// parseExpressionUntilTerminator skips tokens until a clause terminator.
func (p *parser) parseExpressionUntilTerminator() {
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

		if tok.kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if p.handleRightParenInSkip(tok, &depth) {
			break
		}
		if depth == 0 && p.isKeywordTerminator(tok, terminators) {
			break
		}
		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// handleRightParenInSkip adjusts depth for a right paren during a skip.
//
// Takes tok (token) which is the current token.
// Takes depth (*int) which is the running paren depth.
//
// Returns bool which is true when the skip loop must stop because depth is zero and the
// paren closes the enclosing expression.
func (p *parser) handleRightParenInSkip(tok token, depth *int) bool {
	if tok.kind != tokenRightParen {
		return false
	}
	if *depth == 0 {
		return true
	}
	*depth--
	p.advance()
	return false
}

// isKeywordTerminator reports whether tok matches a terminator keyword.
//
// Takes tok (token) which is the candidate token.
// Takes terminators (map[string]struct{}) which lists upper-case keyword terminators.
//
// Returns bool which is true when tok is a terminator keyword.
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
		p.mustSkipParenthesised()
		return
	}
	if p.current().kind != tokenComma {
		p.advance()
	}
}

// skipValuesTrailingRows discards rows after the first VALUES row.
//
// Registers any parameter tokens encountered with an unknown context.
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

	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		var columnRef *querier_dto.ColumnReference
		if columnName != "" {
			columnRef = &querier_dto.ColumnReference{
				TableAlias: tableName,
				ColumnName: columnName,
			}
		}
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
	} else {
		p.skipSetExpression()
	}
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

		if depth == 0 && p.isSetExpressionTerminator(tok) {
			break
		}
		if isParameterToken(tok.kind) {
			p.advance()
			p.registerParameterFromToken(tok, querier_dto.ParameterContextUnknown, nil, nil)
			continue
		}

		p.advance()
	}
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
