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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseWhereClause skips tokens of a WHERE expression up to the next clause boundary,
// registering parameters along the way.
func (p *parser) parseWhereClause() {
	p.parseExpressionUntilTerminator()
}

var (
	// expressionTerminatorKeywords lists keywords that end a general SQL expression at depth
	// zero, used by parseExpressionUntilTerminator.
	expressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordQUALIFY: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		keywordPIVOT: {}, keywordUNPIVOT: {},
	}
)

// parseExpressionUntilTerminator skips tokens of a top-level expression up to the next
// general clause boundary keyword.
func (p *parser) parseExpressionUntilTerminator() {
	p.skipTokensUntilTerminatorSet(expressionTerminatorKeywords)
}

// skipTokensUntilTerminatorSet advances tokens until a top-level keyword from terminators
// or an unmatched closing parenthesis.
//
// Takes terminators (map[string]struct{}) which is the upper-case keyword set that ends
// the walk.
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

// handleRightParenInSkip processes a closing parenthesis encountered during a token skip.
//
// Takes tok (token) which is the candidate closing-parenthesis token.
// Takes depth (*int) which is the current nesting depth, decremented when tok is
// consumed.
//
// Returns bool which is true when the skip should stop because depth reached zero.
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

// isKeywordTerminator reports whether tok is an identifier present in terminators.
//
// Takes tok (token) which is the candidate token.
// Takes terminators (map[string]struct{}) which is the upper-case keyword set to test
// against.
//
// Returns bool which is true when tok is a matching keyword.
func (*parser) isKeywordTerminator(tok token, terminators map[string]struct{}) bool {
	if tok.kind != tokenIdentifier {
		return false
	}
	_, ok := terminators[strings.ToUpper(tok.value)]
	return ok
}

// parseGroupByClause reads a comma-separated GROUP BY column list.
//
// Returns []querier_dto.ColumnReference which is the parsed column reference list.
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

	return columns
}

// parseGroupByColumn parses a single GROUP BY entry, accepting bare or schema-qualified
// identifiers.
//
// Returns querier_dto.ColumnReference which is the parsed reference.
// Returns bool which is true when a column reference was produced.
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
	// orderByTerminators lists keywords that end an ORDER BY list at depth zero.
	orderByTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, "WINDOW": {},
	}
)

// parseOrderByList skips ORDER BY expressions up to the next clause boundary.
func (p *parser) parseOrderByList() {
	p.skipTokensUntilTerminatorSet(orderByTerminators)
}

// consumeParameterOrAdvance consumes the current token, registering it as a bind
// parameter with context when it is a parameter token.
//
// Takes context (querier_dto.ParameterContext) which tags any registered parameter.
func (p *parser) consumeParameterOrAdvance(context querier_dto.ParameterContext) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, context, nil, nil)
		return
	}
	p.advance()
}

// parseLimitOffset dispatches over the LIMIT, OFFSET, and FETCH clause shapes accepted by
// DuckDB.
func (p *parser) parseLimitOffset() {
	p.parseLeadingOffset()

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitClause()
		return
	}

	p.parseFetchClause()
	p.parseTrailingOffset()
}

// parseLeadingOffset consumes an OFFSET clause that appears before any LIMIT or FETCH
// clause.
func (p *parser) parseLeadingOffset() {
	if !p.isKeyword(keywordOFFSET) {
		return
	}
	p.advance()
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseLimitClause consumes a LIMIT expression and any trailing OFFSET or "limit, offset"
// comma form.
func (p *parser) parseLimitClause() {
	if !p.matchKeyword(keywordALL) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextLimit)
	}

	if p.matchKeyword(keywordOFFSET) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
		p.matchKeyword(keywordROW)
		p.matchKeyword(keywordROWS)
	} else if p.current().kind == tokenComma {
		p.advance()
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	}
}

// parseFetchClause consumes a FETCH FIRST/NEXT N ROWS [ONLY|WITH TIES] clause when one is
// present.
func (p *parser) parseFetchClause() {
	if !p.matchKeyword(keywordFETCH) {
		return
	}
	p.matchKeyword(keywordFIRST)
	p.matchKeyword("NEXT")

	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextLimit, nil, nil)
	} else if p.current().kind == tokenNumber {
		p.advance()
	}

	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
	p.matchKeyword("ONLY")
	p.matchKeyword(keywordWITH)
	p.matchKeyword("TIES")
}

// parseTrailingOffset consumes an OFFSET clause that follows a FETCH clause, unless a
// LIMIT is about to start instead.
func (p *parser) parseTrailingOffset() {
	if p.isKeyword(keywordLIMIT) || !p.matchKeyword(keywordOFFSET) {
		return
	}
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseCompoundQuery consumes an optional UNION, INTERSECT, or EXCEPT operator, honouring
// the ALL modifier.
//
// Returns querier_dto.CompoundOperator which is the operator that was consumed, or zero
// when none was present.
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

// skipForUpdateClause consumes a FOR UPDATE/SHARE locking clause and records its presence
// on the parser.
func (p *parser) skipForUpdateClause() {
	if !p.matchKeyword(keywordFOR) {
		return
	}
	p.hasForUpdate = true
	p.matchKeyword("UPDATE")
	p.matchKeyword("NO")
	p.matchKeyword(keywordKEY)
	p.matchKeyword("SHARE")
	if p.matchKeyword("OF") {
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
	p.matchKeyword("NOWAIT")
	p.matchKeyword("SKIP")
	p.matchKeyword("LOCKED")
}

// parseValuesClause walks a VALUES (...), (...), ... source attached to an INSERT,
// binding parameters to their target columns.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the explicit column list, or nil when none was
// provided.
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

// columnRefForIndex builds the column reference associated with the index-th positional
// value in a VALUES row.
//
// Takes tableName (string) which is the target table for the reference.
// Takes columnNames ([]string) which is the declared column list.
// Takes index (int) which is the position within the row.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when index
// is past the declared list.
func (*parser) columnRefForIndex(tableName string, columnNames []string, index int) *querier_dto.ColumnReference {
	if index >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[index],
	}
}

// parseValuesRow walks one parenthesised VALUES row's elements.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the declared column list.
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

// parseValuesRowElement handles one element of a VALUES row, binding a parameter to its
// target column when present.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the declared column list.
// Takes columnIndex (int) which is the element's position in the row.
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

// parseValuesFirstRow parses the first row of a top-level VALUES statement and emits one
// synthetic output column per element.
//
// Returns []querier_dto.RawOutputColumn which is the row's output columns labelled
// column1, column2, and so on.
func (p *parser) parseValuesFirstRow() []querier_dto.RawOutputColumn {
	var outputColumns []querier_dto.RawOutputColumn
	var columnIndex int

	for !p.atEnd() && p.current().kind != tokenRightParen {
		expression := p.parseExpression()
		columnIndex++
		outputColumns = append(outputColumns, querier_dto.RawOutputColumn{
			Name:       fmt.Sprintf("column%d", columnIndex),
			Expression: expression,
		})
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return outputColumns
}

// skipValuesTrailingRows walks the rows after the first in a top-level VALUES statement,
// registering parameters without emitting columns.
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
	// insertSourceTerminators lists the keywords that end an INSERT source expression at
	// depth zero.
	insertSourceTerminators = map[string]struct{}{
		keywordON: {}, keywordRETURNING: {},
	}
)

// parseInsertSource skips the SELECT or query expression supplying an INSERT, stopping at
// ON CONFLICT or RETURNING.
func (p *parser) parseInsertSource() {
	p.skipTokensUntilTerminatorSet(insertSourceTerminators)
}

// parseOnConflict consumes an ON CONFLICT clause and analyses any DO UPDATE body's SET
// and WHERE expressions.
//
// Takes tableName (string) which is the target table for assignment column references.
func (p *parser) parseOnConflict(tableName string) {
	p.matchKeyword("CONFLICT")

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	if p.matchKeyword(keywordON) {
		p.matchKeyword(keywordCONSTRAINT)
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
	}

	if p.matchKeyword("DO") {
		if p.matchKeyword("NOTHING") {
			return
		}
		if p.matchKeyword("UPDATE") {
			if p.matchKeyword(keywordSET) {
				p.parseSetClause(tableName)
			}
			if p.matchKeyword(keywordWHERE) {
				p.parseWhereClause()
			}
		}
	}
}

// parseSetClause walks an UPDATE SET list, dispatching to single- or multi-column
// assignment parsers.
//
// Takes tableName (string) which is the target table for assignment column references.
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

// parseSingleColumnSetClause parses one "column = expression" assignment, binding
// parameter targets to the column when possible.
//
// Takes tableName (string) which is the target table.
func (p *parser) parseSingleColumnSetClause(tableName string) {
	columnName := ""
	if p.current().kind == tokenIdentifier {
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

// parseMultiColumnSetClause parses a "(c1, c2, ...) = (v1, v2, ...)" SET assignment.
//
// Takes tableName (string) which is the target table.
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

// parseMultiColumnSetValues walks the value list inside a multi-column SET assignment,
// binding parameter targets by position.
//
// Takes tableName (string) which is the target table.
// Takes columnNames ([]string) which is the LHS column list.
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
	// setExpressionTerminators lists keywords that end a SET expression at depth zero.
	setExpressionTerminators = map[string]struct{}{
		keywordWHERE: {}, keywordFROM: {}, keywordRETURNING: {},
		keywordORDER: {}, keywordLIMIT: {},
	}
)

// skipSetExpression advances past a single SET assignment expression up to the next comma
// or terminator keyword.
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

// isSetExpressionTerminator reports whether tok ends one SET assignment expression.
//
// Takes tok (token) which is the candidate token.
//
// Returns bool which is true at a comma or one of the terminator keywords.
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
