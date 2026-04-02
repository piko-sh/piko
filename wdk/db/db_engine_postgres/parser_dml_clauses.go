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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseWhereClause() {
	p.parseExpressionUntilTerminator()
}

var expressionTerminatorKeywords = map[string]struct{}{
	keywordGROUP: {}, keywordHAVING: {}, keywordORDER: {}, keywordLIMIT: {},
	keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {}, "WINDOW": {},
	keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
	keywordRETURNING: {}, keywordSET: {}, keywordON: {},
	keywordFROM: {}, keywordWHERE: {}, "INTO": {},
}

func (p *parser) parseExpressionUntilTerminator() {
	p.skipTokensUntilTerminatorSet(expressionTerminatorKeywords)
}

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

func (*parser) isKeywordTerminator(tok token, terminators map[string]struct{}) bool {
	if tok.kind != tokenIdentifier {
		return false
	}
	_, ok := terminators[strings.ToUpper(tok.value)]
	return ok
}

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

var orderByTerminators = map[string]struct{}{
	keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
	keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
	keywordRETURNING: {}, "WINDOW": {},
}

func (p *parser) parseOrderByList() {
	p.skipTokensUntilTerminatorSet(orderByTerminators)
}

func (p *parser) consumeParameterOrAdvance(context querier_dto.ParameterContext) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, context, nil, nil)
		return
	}
	p.advance()
}

func (p *parser) parseLimitOffset() {
	p.parseLeadingOffset()

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitClause()
		return
	}

	p.parseFetchClause()
	p.parseTrailingOffset()
}

func (p *parser) parseLeadingOffset() {
	if !p.isKeyword(keywordOFFSET) {
		return
	}
	p.advance()
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

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

func (p *parser) parseTrailingOffset() {
	if p.isKeyword(keywordLIMIT) || !p.matchKeyword(keywordOFFSET) {
		return
	}
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

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

func (*parser) columnRefForIndex(tableName string, columnNames []string, index int) *querier_dto.ColumnReference {
	if index >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[index],
	}
}

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

var insertSourceTerminators = map[string]struct{}{
	keywordON: {}, keywordRETURNING: {},
}

func (p *parser) parseInsertSource() {
	p.skipTokensUntilTerminatorSet(insertSourceTerminators)
}

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

var setExpressionTerminators = map[string]struct{}{
	keywordWHERE: {}, keywordFROM: {}, keywordRETURNING: {},
	keywordORDER: {}, keywordLIMIT: {},
}

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
