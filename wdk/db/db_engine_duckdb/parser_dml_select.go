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

func (p *parser) parseCTEList() ([]querier_dto.RawCTEDefinition, error) {
	p.mustKeyword(keywordWITH)
	isRecursive := p.matchKeyword("RECURSIVE")

	var definitions []querier_dto.RawCTEDefinition

	for {
		definition, err := p.parseSingleCTEDefinition(isRecursive)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return definitions, nil
}

func (p *parser) parseSingleCTEDefinition(isRecursive bool) (querier_dto.RawCTEDefinition, error) {
	cteName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.RawCTEDefinition{}, err
	}

	columnNames, columnListErr := p.parseCTEColumnNames()
	if columnListErr != nil {
		return querier_dto.RawCTEDefinition{}, columnListErr
	}

	if _, err := p.expectKeyword(keywordAS); err != nil {
		return querier_dto.RawCTEDefinition{}, err
	}

	p.skipMaterialisationHint()

	cteTokens, collectErr := p.collectParenthesised()
	if collectErr != nil {
		return querier_dto.RawCTEDefinition{}, collectErr
	}

	cteAnalysis, cteParser := p.analyseCTEBody(cteTokens)

	definition := querier_dto.RawCTEDefinition{
		Name:        cteName,
		IsRecursive: isRecursive,
	}

	if cteAnalysis != nil {
		p.populateCTEDefinition(&definition, cteAnalysis, columnNames)
	}

	p.parameterCount += cteParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, cteParser.parameterRefs...)

	return definition, nil
}

func (p *parser) skipMaterialisationHint() {
	p.matchKeyword("MATERIALIZED")
	if p.matchKeyword(keywordNOT) {
		p.matchKeyword("MATERIALIZED")
	}
}

func (p *parser) analyseCTEBody(cteTokens []token) (*querier_dto.RawQueryAnalysis, *parser) {
	cteParser := newParser(cteTokens)
	var cteAnalysis *querier_dto.RawQueryAnalysis
	var analyseErr error

	switch {
	case cteParser.isKeyword(keywordVALUES):
		cteAnalysis, analyseErr = cteParser.analyseValues()
	case cteParser.isAnyKeyword("INSERT", "UPDATE", "DELETE"):
		p.hasDataModifyingCTE = true
		cteAnalysis, analyseErr = p.analyseCTEBodyDML(cteParser)
	default:
		cteAnalysis, analyseErr = cteParser.analyseSelect()
	}

	if analyseErr != nil {
		return nil, cteParser
	}
	return cteAnalysis, cteParser
}

func (*parser) analyseCTEBodyDML(cteParser *parser) (*querier_dto.RawQueryAnalysis, error) {
	switch {
	case cteParser.isKeyword("INSERT"):
		return cteParser.analyseInsert()
	case cteParser.isKeyword("UPDATE"):
		return cteParser.analyseUpdate()
	default:
		return cteParser.analyseDelete()
	}
}

func (p *parser) parseCTEColumnNames() ([]string, error) {
	if p.current().kind != tokenLeftParen || p.isKeyword(keywordAS) {
		return nil, nil
	}
	if p.peekForAS() {
		return nil, nil
	}
	return p.parseColumnList()
}

func (*parser) populateCTEDefinition(
	definition *querier_dto.RawCTEDefinition,
	analysis *querier_dto.RawQueryAnalysis,
	columnNames []string,
) {
	if len(columnNames) > 0 {
		for columnIndex, name := range columnNames {
			column := querier_dto.RawOutputColumn{Name: name}
			if columnIndex < len(analysis.OutputColumns) {
				column.Expression = analysis.OutputColumns[columnIndex].Expression
				column.ColumnName = analysis.OutputColumns[columnIndex].ColumnName
				column.TableAlias = analysis.OutputColumns[columnIndex].TableAlias
			}
			definition.OutputColumns = append(definition.OutputColumns, column)
		}
	} else {
		definition.OutputColumns = analysis.OutputColumns
	}
	definition.FromTables = analysis.FromTables
	definition.JoinClauses = analysis.JoinClauses
	definition.CompoundBranches = analysis.CompoundBranches
}

func (p *parser) peekForAS() bool {
	saved := p.position
	depth := 0
	for i := p.position; i < len(p.tokens); i++ {
		tok := p.tokens[i]
		if tok.kind == tokenLeftParen {
			depth++
			continue
		}
		if tok.kind != tokenRightParen {
			continue
		}
		depth--
		if depth != 0 {
			continue
		}
		if i+1 < len(p.tokens) && p.tokens[i+1].kind == tokenIdentifier &&
			strings.EqualFold(p.tokens[i+1].value, keywordAS) {
			return false
		}
		break
	}
	p.position = saved
	return true
}

func (p *parser) parseColumnList() ([]string, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()

	var names []string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		name, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, err
		}
		names = append(names, name)
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return names, nil
}

func (p *parser) parseOutputColumns() ([]querier_dto.RawOutputColumn, error) {
	var columns []querier_dto.RawOutputColumn

	for {
		column, err := p.parseOneOutputColumn()
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns, nil
}

func (p *parser) parseOneOutputColumn() (querier_dto.RawOutputColumn, error) {
	if p.current().kind == tokenStar {
		p.advance()
		return querier_dto.RawOutputColumn{IsStar: true}, nil
	}

	if p.current().kind == tokenIdentifier && p.peek().kind == tokenDot {
		tableAlias := p.advance().value
		p.advance()
		if p.current().kind == tokenStar {
			p.advance()
			return querier_dto.RawOutputColumn{IsStar: true, TableAlias: tableAlias}, nil
		}
		p.position -= 2
	}

	expression := p.parseExpression()
	column := querier_dto.RawOutputColumn{}
	p.expressionToOutputColumn(expression, &column)

	if err := p.parseOutputColumnAlias(&column); err != nil {
		return querier_dto.RawOutputColumn{}, err
	}

	return column, nil
}

func (p *parser) parseOutputColumnAlias(column *querier_dto.RawOutputColumn) error {
	if p.matchKeyword(keywordAS) {
		alias, aliasErr := p.parseIdentifierOrKeyword()
		if aliasErr != nil {
			return aliasErr
		}
		column.Name = alias
		return nil
	}
	if column.Name == "" && p.current().kind == tokenIdentifier && !p.isSelectTerminator() {
		column.Name = p.advance().value
	}
	return nil
}

func (p *parser) parseReturningClause() ([]querier_dto.RawOutputColumn, error) {
	return p.parseOutputColumns()
}

func (*parser) expressionToOutputColumn(expression querier_dto.Expression, column *querier_dto.RawOutputColumn) {
	switch expr := expression.(type) {
	case *querier_dto.ColumnRefExpression:
		column.Name = expr.ColumnName
		column.TableAlias = expr.TableAlias
		column.ColumnName = expr.ColumnName
	case *querier_dto.FunctionCallExpression:
		column.Name = expr.FunctionName
		column.Expression = expression
	default:
		column.Expression = expression
	}
}

func (p *parser) isSelectTerminator() bool {
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT, keywordOFFSET,
		keywordFETCH, keywordFOR, keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING, "INTO", "WINDOW")
}

func (p *parser) parseFromClause() ([]querier_dto.TableReference, []querier_dto.JoinClause, error) {
	var tables []querier_dto.TableReference
	var joins []querier_dto.JoinClause

	p.matchKeyword(keywordLATERAL)

	initialTableRef, initialErr := p.parseTableSource(querier_dto.JoinInner)
	if initialErr != nil {
		return nil, nil, initialErr
	}
	if initialTableRef != nil {
		tables = append(tables, *initialTableRef)
	}

	for {
		joinKind, isJoin := p.parseJoinKeyword()
		if isJoin {
			if err := p.appendExplicitJoin(joinKind, &joins); err != nil {
				return nil, nil, err
			}
			continue
		}

		if p.current().kind != tokenComma {
			break
		}
		if err := p.appendCommaJoinedTable(&tables); err != nil {
			return nil, nil, err
		}
	}

	return tables, joins, nil
}

func (p *parser) appendCommaJoinedTable(tables *[]querier_dto.TableReference) error {
	p.advance()
	p.matchKeyword(keywordLATERAL)
	tableRef, err := p.parseTableSource(querier_dto.JoinInner)
	if err != nil {
		return err
	}
	if tableRef != nil {
		*tables = append(*tables, *tableRef)
	}
	return nil
}

func (p *parser) appendExplicitJoin(joinKind querier_dto.JoinKind, joins *[]querier_dto.JoinClause) error {
	p.matchKeyword(keywordLATERAL)

	tableRef, err := p.parseTableSource(joinKind)
	if err != nil {
		return err
	}
	if tableRef != nil {
		*joins = append(*joins, querier_dto.JoinClause{Kind: joinKind, Table: *tableRef})
	}

	p.parseJoinCondition()
	return nil
}

func (p *parser) parseTableSource(joinKind querier_dto.JoinKind) (*querier_dto.TableReference, error) {
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(joinKind); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if p.isPivotOrUnpivot() {
		p.parsePivotOrUnpivot(joinKind)
		return nil, nil
	}
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(joinKind)
		return nil, nil
	}
	return new(p.parseTableReference()), nil
}

func (p *parser) isPivotOrUnpivot() bool {
	return p.isKeyword(keywordPIVOT) || p.isKeyword(keywordUNPIVOT)
}

func (p *parser) parsePivotOrUnpivot(joinKind querier_dto.JoinKind) {
	p.advance()

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinOrClauseKeyword() {
		alias = p.advance().value
	}

	if alias != "" {
		p.rawDerivedTables = append(p.rawDerivedTables, querier_dto.RawDerivedTableReference{
			Alias:    alias,
			JoinKind: joinKind,
		})
	}
}

func (p *parser) isJoinOrClauseKeyword() bool {
	return p.isAnyKeyword(keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT,
		keywordOFFSET, keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", keywordON,
		keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordQUALIFY, keywordPOSITIONAL)
}

func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseWhereClause()
		return
	}
	if p.matchKeyword(keywordUSING) && p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

func (p *parser) parseTableReference() querier_dto.TableReference {
	if p.current().kind != tokenIdentifier {
		return querier_dto.TableReference{}
	}

	schema, name := p.mustSchemaQualifiedName()

	if p.current().kind == tokenStar {
		p.advance()
	}

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() &&
		!p.isAnyKeyword(keywordSET, keywordVALUES, keywordDEFAULT, keywordWHERE, "INNER", "LEFT", "RIGHT",
			"FULL", "CROSS", "NATURAL", keywordJOIN, keywordON, keywordUSING, keywordLATERAL) {
		alias = p.advance().value
	}

	return querier_dto.TableReference{Schema: schema, Name: name, Alias: alias}
}

func (p *parser) isSubqueryStart() bool {
	if p.current().kind != tokenLeftParen {
		return false
	}

	saved := p.position
	p.advance()
	result := p.isKeyword(keywordSELECT) || p.isKeyword(keywordWITH) || p.isKeyword(keywordVALUES)
	p.position = saved
	return result
}

func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!p.isAnyKeyword(keywordSELECT, keywordWITH, keywordVALUES)
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

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	p.rawDerivedTables = append(p.rawDerivedTables, querier_dto.RawDerivedTableReference{
		Alias:      alias,
		InnerQuery: innerAnalysis,
		JoinKind:   joinKind,
	})

	return nil
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

	if p.matchKeyword(keywordWITH) {
		p.matchKeyword("ORDINALITY")
	}

	alias := functionName
	var columnDefinitions []querier_dto.TVFColumnDefinition
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
		if p.current().kind == tokenLeftParen {
			columnDefinitions = p.parseTVFColumnDefinitions()
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() {
		alias = p.advance().value
	}

	p.rawTableValuedFunctions = append(p.rawTableValuedFunctions, querier_dto.RawTableValuedFunctionReference{
		FunctionName:      functionName,
		Alias:             alias,
		ColumnDefinitions: columnDefinitions,
		JoinKind:          joinKind,
	})
}

func (p *parser) parseTVFColumnDefinitions() []querier_dto.TVFColumnDefinition {
	p.advance()
	var definitions []querier_dto.TVFColumnDefinition
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind != tokenIdentifier {
			break
		}
		name := p.advance().value
		var typeName string
		if p.current().kind == tokenIdentifier && p.current().kind != tokenRightParen {
			if !p.isAnyKeyword(",") && p.current().kind != tokenComma {
				typeName = p.parseCastTypeName()
			}
		}
		definitions = append(definitions, querier_dto.TVFColumnDefinition{
			Name:     name,
			TypeName: typeName,
		})
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return definitions
}

type joinKeywordEntry struct {
	kind querier_dto.JoinKind

	hasOuter bool
}

var joinKeywordDispatch = map[string]joinKeywordEntry{
	"INNER":           {kind: querier_dto.JoinInner},
	"LEFT":            {kind: querier_dto.JoinLeft, hasOuter: true},
	"RIGHT":           {kind: querier_dto.JoinRight, hasOuter: true},
	"FULL":            {kind: querier_dto.JoinFull, hasOuter: true},
	"CROSS":           {kind: querier_dto.JoinCross},
	keywordPOSITIONAL: {kind: querier_dto.JoinPositional},
}

func (p *parser) parseJoinKeyword() (querier_dto.JoinKind, bool) {
	p.matchKeyword("NATURAL")

	if p.current().kind == tokenIdentifier {
		if entry, exists := joinKeywordDispatch[strings.ToUpper(p.current().value)]; exists {
			p.advance()
			if entry.hasOuter {
				p.matchKeyword("OUTER")
			}
			p.matchKeyword(keywordJOIN)
			return entry.kind, true
		}
	}

	if p.matchKeyword(keywordJOIN) {
		return querier_dto.JoinInner, true
	}

	return 0, false
}

func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "NATURAL")
}
