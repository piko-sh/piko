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
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safeconv"
)

func (p *parser) parseFromClause() ([]querier_dto.TableReference, []querier_dto.JoinClause, error) {
	var tables []querier_dto.TableReference
	var joins []querier_dto.JoinClause

	table, err := p.parseFromTableSource(querier_dto.JoinInner)
	if err != nil {
		return nil, nil, err
	}
	if table != nil {
		tables = append(tables, *table)
	}

	for {
		joinKind := p.parseJoinKeyword()
		if joinKind < 0 {
			commaTable, commaErr := p.parseCommaJoin()
			if commaErr != nil {
				return nil, nil, commaErr
			}
			if commaTable == nil {
				break
			}
			tables = append(tables, *commaTable)
			continue
		}

		join, joinErr := p.parseJoinTarget(joinKind)
		if joinErr != nil {
			return nil, nil, joinErr
		}
		if join != nil {
			joins = append(joins, *join)
		}

		p.parseJoinCondition()
	}

	return tables, joins, nil
}

func (p *parser) parseCommaJoin() (*querier_dto.TableReference, error) {
	if p.current().kind != tokenComma {
		return nil, nil
	}
	p.advance()
	return p.parseFromTableSource(querier_dto.JoinInner)
}

func (p *parser) parseFromTableSource(joinKind querier_dto.JoinKind) (*querier_dto.TableReference, error) {
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(joinKind)
		return nil, nil
	}
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(joinKind); err != nil {
			return nil, err
		}
		return nil, nil
	}
	tableName, alias := p.parseTableReference()
	return &querier_dto.TableReference{Name: tableName, Alias: alias}, nil
}

func (p *parser) parseJoinTarget(joinKind int) (*querier_dto.JoinClause, error) {
	joinKindValue := safeconv.IntToUint8(joinKind)
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(querier_dto.JoinKind(joinKindValue))
		return nil, nil
	}
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(querier_dto.JoinKind(joinKindValue)); err != nil {
			return nil, err
		}
		return nil, nil
	}
	joinTable, joinAlias := p.parseTableReference()
	return &querier_dto.JoinClause{
		Kind: querier_dto.JoinKind(joinKindValue),
		Table: querier_dto.TableReference{
			Name:  joinTable,
			Alias: joinAlias,
		},
	}, nil
}

func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseWhereExpression()
	} else if p.matchKeyword("USING") {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
	}
}

func (p *parser) isSubqueryStart() bool {
	return p.current().kind == tokenLeftParen && p.peek().kind == tokenIdentifier &&
		strings.EqualFold(p.peek().value, keywordSELECT)
}

func (p *parser) parseDerivedTable(joinKind querier_dto.JoinKind) error {
	innerTokens, collectError := p.collectParenthesised()
	if collectError != nil {
		return collectError
	}

	childParser := newParser(innerTokens)
	childParser.parameterCount = p.parameterCount
	innerAnalysis, analyseError := childParser.analyseSelect()
	if analyseError != nil {
		return analyseError
	}
	p.parameterCount = childParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, childParser.parameterRefs...)

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() {
		alias = p.advance().value
	}

	p.rawDerivedTables = append(p.rawDerivedTables, querier_dto.RawDerivedTableReference{
		Alias:      alias,
		InnerQuery: innerAnalysis,
		JoinKind:   joinKind,
	})

	return nil
}

func (p *parser) parseTableReference() (tableName string, tableAlias string) {
	if p.current().kind != tokenIdentifier {
		return "", ""
	}

	name := p.advance().value

	if p.current().kind == tokenDot {
		p.advance()
		if p.current().kind == tokenIdentifier {
			name = p.advance().value
		}
	}

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() &&
		!p.isAnyKeyword(keywordSET, keywordVALUES, "DEFAULT", keywordWHERE, "INNER", "LEFT", "RIGHT",
			"FULL", "CROSS", "NATURAL", keywordJOIN, keywordON, "USING") {
		alias = p.advance().value
	}

	return name, alias
}

func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!strings.EqualFold(p.current().value, keywordSELECT)
}

func (p *parser) parseTableValuedFunction(joinKind querier_dto.JoinKind) {
	functionName := strings.ToLower(p.advance().value)
	p.advance()

	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseExpression()
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	alias := functionName
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() {
		alias = p.advance().value
	}

	p.rawTableValuedFunctions = append(p.rawTableValuedFunctions, querier_dto.RawTableValuedFunctionReference{
		FunctionName: functionName,
		Alias:        alias,
		JoinKind:     joinKind,
	})
}

func (p *parser) parseJoinKeyword() int {
	p.matchKeyword("NATURAL")

	if p.matchKeyword("INNER") {
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinInner)
	}
	if p.matchKeyword("LEFT") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinLeft)
	}
	if p.matchKeyword("RIGHT") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinRight)
	}
	if p.matchKeyword("FULL") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinFull)
	}
	if p.matchKeyword("CROSS") {
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinCross)
	}
	if p.matchKeyword(keywordJOIN) {
		return int(querier_dto.JoinInner)
	}

	return -1
}

func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "NATURAL")
}

func (p *parser) matchCompoundOperator() querier_dto.CompoundOperator {
	if p.matchKeyword(keywordUNION) {
		if p.matchKeyword("ALL") {
			return querier_dto.CompoundUnionAll
		}
		return querier_dto.CompoundUnion
	}
	if p.matchKeyword(keywordINTERSECT) {
		return querier_dto.CompoundIntersect
	}
	if p.matchKeyword(keywordEXCEPT) {
		return querier_dto.CompoundExcept
	}
	return 0
}

func (p *parser) resolveComparisonContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	if paramPosition < 2 {
		return querier_dto.ParameterContextUnknown, nil
	}
	prevToken := p.tokens[paramPosition-1]
	if prevToken.kind != tokenOperator || !isComparisonOperator(prevToken.value) {
		return querier_dto.ParameterContextUnknown, nil
	}
	columnRef := p.extractColumnReference(paramPosition - 2)
	if columnRef == nil {
		return querier_dto.ParameterContextUnknown, nil
	}
	return querier_dto.ParameterContextComparison, columnRef
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
	typeStart := asPosition + 1
	if typeStart >= len(p.tokens) || p.tokens[typeStart].kind != tokenIdentifier {
		return nil
	}
	typeName := p.collectCastTypeName(typeStart)
	return new(normaliseTypeName(typeName))
}

func (p *parser) findASKeywordAfter(paramPosition int) int {
	for i := paramPosition + 1; i < len(p.tokens); i++ {
		if p.tokens[i].kind == tokenRightParen {
			return -1
		}
		if p.tokens[i].kind == tokenIdentifier && strings.EqualFold(p.tokens[i].value, keywordAS) {
			return i
		}
	}
	return -1
}

func (p *parser) collectCastTypeName(start int) string {
	var builder strings.Builder
	builder.WriteString(p.tokens[start].value)
	for j := start + 1; j < len(p.tokens); j++ {
		if p.tokens[j].kind != tokenIdentifier || p.isCastTypeTerminator(j) {
			break
		}
		builder.WriteString(" ")
		builder.WriteString(p.tokens[j].value)
	}
	return builder.String()
}

func (p *parser) isCastTypeTerminator(position int) bool {
	if p.tokens[position].kind == tokenRightParen {
		return true
	}
	return p.isKeywordAt(
		position,
		keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING,
		keywordORDER, keywordLIMIT,
	)
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

func (p *parser) parseGroupByList() []querier_dto.ColumnReference {
	var columns []querier_dto.ColumnReference

	for {
		columns = append(columns, p.parseGroupByItem()...)

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns
}

func (p *parser) parseGroupByItem() []querier_dto.ColumnReference {
	if p.current().kind != tokenIdentifier {
		p.advance()
		return nil
	}

	first := p.advance().value
	if p.current().kind != tokenDot {
		return []querier_dto.ColumnReference{{ColumnName: first}}
	}

	p.advance()
	if p.current().kind != tokenIdentifier {
		return nil
	}

	second := p.advance().value
	return []querier_dto.ColumnReference{{TableAlias: first, ColumnName: second}}
}

var orderByTerminators = map[string]bool{
	keywordLIMIT: true, keywordUNION: true, keywordINTERSECT: true,
	keywordEXCEPT: true, keywordRETURNING: true,
}

func (p *parser) parseOrderByList() {
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

		if depth == 0 && isOrderByTerminator(tok) {
			break
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

func isOrderByTerminator(tok token) bool {
	return tok.kind == tokenIdentifier && orderByTerminators[strings.ToUpper(tok.value)]
}

func (p *parser) parseLimitOffset() {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextLimit, nil, nil)
	} else {
		p.advance()
	}

	if p.matchKeyword("OFFSET") {
		if isParameterToken(p.current().kind) {
			parameterToken := p.current()
			p.advance()
			p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextOffset, nil, nil)
		} else {
			p.advance()
		}
	} else if p.current().kind == tokenComma {
		p.advance()
		if isParameterToken(p.current().kind) {
			parameterToken := p.current()
			p.advance()
			p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextOffset, nil, nil)
		} else {
			p.advance()
		}
	}
}

func isInsertSourceTerminator(tok token, depth int) bool {
	if tok.kind == tokenRightParen && depth == 0 {
		return true
	}
	if depth == 0 && tok.kind == tokenIdentifier {
		upper := strings.ToUpper(tok.value)
		return upper == keywordON || upper == keywordRETURNING
	}
	return false
}

func (p *parser) parseInsertSource() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if isInsertSourceTerminator(tok, depth) {
			break
		}

		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

func (p *parser) parseValuesClause(tableName string, columnNames []string) {
	for p.current().kind == tokenLeftParen {
		p.advance()
		p.parseOneValueRow(tableName, columnNames)

		if p.current().kind == tokenRightParen {
			p.advance()
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

func (p *parser) parseOneValueRow(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		columnIndex += p.parseOneValueElement(tableName, columnNames, columnIndex)
	}
}

func (p *parser) parseOneValueElement(tableName string, columnNames []string, columnIndex int) int {
	if isParameterToken(p.current().kind) {
		columnReference := columnRefForIndex(tableName, columnNames, columnIndex)
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnReference, nil)
	} else if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	} else if p.current().kind != tokenComma {
		p.advance()
	}

	if p.current().kind == tokenComma {
		p.advance()
		return 1
	}
	return 0
}

func columnRefForIndex(tableName string, columnNames []string, columnIndex int) *querier_dto.ColumnReference {
	if columnIndex >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[columnIndex],
	}
}

func (p *parser) parseSetClause(tableName string) {
	for {
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

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

var setExpressionTerminators = map[string]bool{
	keywordWHERE: true, keywordFROM: true, keywordRETURNING: true,
	keywordORDER: true, keywordLIMIT: true,
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
		if depth == 0 && isSetExpressionTerminator(tok) {
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

func isSetExpressionTerminator(tok token) bool {
	if tok.kind == tokenComma {
		return true
	}
	return tok.kind == tokenIdentifier && setExpressionTerminators[strings.ToUpper(tok.value)]
}

func (p *parser) skipOnConflict(tableName string) {
	p.matchKeyword("CONFLICT")

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
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
				p.parseWhereExpression()
			}
		}
	}
}
