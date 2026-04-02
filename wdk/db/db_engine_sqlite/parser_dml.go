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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func isParameterToken(kind tokenKind) bool {
	return kind == tokenQuestionMark || kind == tokenNumberedParam || kind == tokenNamedParam
}

func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if p.isKeyword(keywordWITH) {
		cteDefinitions, err := p.parseWithClause()
		if err != nil {
			return nil, err
		}
		analysis.CTEDefinitions = cteDefinitions
	}

	p.mustKeyword(keywordSELECT)
	p.matchKeyword("DISTINCT")
	p.matchKeyword("ALL")

	outputColumns, err := p.parseSelectList()
	if err != nil {
		return nil, err
	}
	analysis.OutputColumns = outputColumns

	if p.matchKeyword(keywordFROM) {
		fromTables, joinClauses, err := p.parseFromClause()
		if err != nil {
			return nil, err
		}
		analysis.FromTables = fromTables
		analysis.JoinClauses = joinClauses
	}

	if err := p.parseSelectBody(analysis); err != nil {
		return nil, err
	}

	if err := p.parseSelectCompoundBranches(analysis); err != nil {
		return nil, err
	}

	p.parseSelectTrailer()

	analysis.ReadOnly = true
	analysis.ParameterReferences = p.parameterRefs
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
	return analysis, nil
}

func (p *parser) parseSelectBody(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(keywordFROM) {
		fromTables, joinClauses, err := p.parseFromClause()
		if err != nil {
			return err
		}
		analysis.FromTables = fromTables
		analysis.JoinClauses = joinClauses
	}

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereExpression()
	}

	if p.matchKeyword(keywordGROUP) {
		p.matchKeyword("BY")
		groupByColumns := p.parseGroupByList()
		analysis.GroupByColumns = groupByColumns
	}

	if p.matchKeyword(keywordHAVING) {
		p.parseWhereExpression()
	}

	return nil
}

func (p *parser) parseSelectCompoundBranches(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		compoundOperator := p.matchCompoundOperator()
		if compoundOperator == 0 {
			break
		}
		branchAnalysis, branchError := p.analyseSelect()
		if branchError != nil {
			return branchError
		}
		analysis.CompoundBranches = append(analysis.CompoundBranches, querier_dto.RawCompoundBranch{
			Operator: compoundOperator,
			Query:    branchAnalysis,
		})
	}
	return nil
}

func (p *parser) parseSelectTrailer() {
	if p.matchKeyword(keywordORDER) {
		p.matchKeyword("BY")
		p.parseOrderByList()
	}

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitOffset()
	}
}

func (p *parser) analyseInsert() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if p.isKeyword(keywordWITH) {
		cteDefinitions, err := p.parseWithClause()
		if err != nil {
			return nil, err
		}
		analysis.CTEDefinitions = cteDefinitions
	}

	p.matchKeyword("INSERT")
	p.matchKeyword("REPLACE")

	if p.matchKeyword(keywordOR) {
		p.advance()
	}

	p.matchKeyword("INTO")

	tableName, alias := p.parseTableReference()
	analysis.FromTables = []querier_dto.TableReference{{Name: tableName, Alias: alias}}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, err := p.parseColumnList()
		if err != nil {
			return nil, err
		}
		columnNames = names
	}

	if p.matchKeyword(keywordVALUES) {
		p.parseValuesClause(tableName, columnNames)
	} else if p.matchKeyword("DEFAULT") {
		p.matchKeyword(keywordVALUES)
	} else {
		p.parseInsertSource()
	}

	if p.matchKeyword(keywordON) {
		p.skipOnConflict(tableName)
	}

	if p.matchKeyword(keywordRETURNING) {
		analysis.HasReturning = true
		outputColumns, err := p.parseSelectList()
		if err != nil {
			return nil, err
		}
		analysis.OutputColumns = outputColumns
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
}

func (p *parser) analyseUpdate() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if p.isKeyword(keywordWITH) {
		cteDefinitions, err := p.parseWithClause()
		if err != nil {
			return nil, err
		}
		analysis.CTEDefinitions = cteDefinitions
	}

	p.mustKeyword("UPDATE")

	if p.matchKeyword(keywordOR) {
		p.advance()
	}

	tableName, alias := p.parseTableReference()
	analysis.FromTables = []querier_dto.TableReference{{Name: tableName, Alias: alias}}

	p.mustKeyword(keywordSET)
	p.parseSetClause(tableName)

	if p.matchKeyword(keywordFROM) {
		fromTables, joinClauses, err := p.parseFromClause()
		if err != nil {
			return nil, err
		}
		analysis.FromTables = append(analysis.FromTables, fromTables...)
		analysis.JoinClauses = joinClauses
	}

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereExpression()
	}

	if p.matchKeyword(keywordRETURNING) {
		analysis.HasReturning = true
		outputColumns, err := p.parseSelectList()
		if err != nil {
			return nil, err
		}
		analysis.OutputColumns = outputColumns
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
}

func (p *parser) analyseDelete() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if p.isKeyword(keywordWITH) {
		cteDefinitions, err := p.parseWithClause()
		if err != nil {
			return nil, err
		}
		analysis.CTEDefinitions = cteDefinitions
	}

	p.mustKeyword("DELETE")
	p.mustKeyword(keywordFROM)

	tableName, alias := p.parseTableReference()
	analysis.FromTables = []querier_dto.TableReference{{Name: tableName, Alias: alias}}

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereExpression()
	}

	if p.matchKeyword(keywordORDER) {
		p.matchKeyword("BY")
		p.parseOrderByList()
	}

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitOffset()
	}

	if p.matchKeyword(keywordRETURNING) {
		analysis.HasReturning = true
		outputColumns, err := p.parseSelectList()
		if err != nil {
			return nil, err
		}
		analysis.OutputColumns = outputColumns
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
}

func (p *parser) analyseValues() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{ReadOnly: true}

	p.mustKeyword(keywordVALUES)

	if p.current().kind != tokenLeftParen {
		analysis.ParameterReferences = p.parameterRefs
		return analysis, nil
	}
	p.advance()

	analysis.OutputColumns = p.parseValuesFirstRow()

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.skipValuesTrailingRows()

	if p.matchKeyword(keywordORDER) {
		p.matchKeyword(keywordBY)
		p.parseOrderByList()
	}

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitOffset()
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
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

func (p *parser) parseWithClause() ([]querier_dto.RawCTEDefinition, error) {
	p.mustKeyword(keywordWITH)
	isRecursive := p.matchKeyword("RECURSIVE")

	var definitions []querier_dto.RawCTEDefinition

	for {
		definition, err := p.parseSingleCTE(isRecursive)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)

		if !p.matchKeyword(",") && p.current().kind != tokenComma {
			break
		}
		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	return definitions, nil
}

func (p *parser) parseSingleCTE(isRecursive bool) (querier_dto.RawCTEDefinition, error) {
	cteName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.RawCTEDefinition{}, err
	}

	columnNames, columnErr := p.parseCTEColumnNames()
	if columnErr != nil {
		return querier_dto.RawCTEDefinition{}, columnErr
	}

	if _, err := p.expectKeyword(keywordAS); err != nil {
		return querier_dto.RawCTEDefinition{}, err
	}

	cteTokens, err := p.collectParenthesised()
	if err != nil {
		return querier_dto.RawCTEDefinition{}, err
	}

	cteParser := newParser(cteTokens)
	cteAnalysis, analyseErr := cteParser.analyseSelect()

	definition := querier_dto.RawCTEDefinition{
		Name:        cteName,
		IsRecursive: isRecursive,
	}

	if analyseErr == nil {
		definition.OutputColumns = buildCTEOutputColumns(columnNames, cteAnalysis)
		definition.FromTables = cteAnalysis.FromTables
	}

	p.parameterCount += cteParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, cteParser.parameterRefs...)

	return definition, nil
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

func buildCTEOutputColumns(
	columnNames []string,
	analysis *querier_dto.RawQueryAnalysis,
) []querier_dto.RawOutputColumn {
	if len(columnNames) == 0 {
		return analysis.OutputColumns
	}
	columns := make([]querier_dto.RawOutputColumn, len(columnNames))
	for i, name := range columnNames {
		columns[i] = querier_dto.RawOutputColumn{Name: name}
	}
	return columns
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
		if p.isFollowedByAS(i) {
			return false
		}
		break
	}
	p.position = saved
	return true
}

func (p *parser) isFollowedByAS(position int) bool {
	return position+1 < len(p.tokens) &&
		p.tokens[position+1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[position+1].value, keywordAS)
}

func (p *parser) parseSelectList() ([]querier_dto.RawOutputColumn, error) {
	var columns []querier_dto.RawOutputColumn

	for {
		column, err := p.parseSelectItem()
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

func (p *parser) parseSelectItem() (querier_dto.RawOutputColumn, error) {
	if p.current().kind == tokenStar {
		p.advance()
		return querier_dto.RawOutputColumn{IsStar: true}, nil
	}

	if p.current().kind == tokenIdentifier && p.peek().kind == tokenDot {
		return p.parseQualifiedSelectItem()
	}

	expression := p.parseExpression()
	column := querier_dto.RawOutputColumn{}
	p.expressionToOutputColumn(expression, &column)

	if p.matchKeyword(keywordAS) {
		alias, aliasErr := p.parseIdentifierOrKeyword()
		if aliasErr != nil {
			return querier_dto.RawOutputColumn{}, aliasErr
		}
		column.Name = alias
	} else if column.Name == "" && p.current().kind == tokenIdentifier && !p.isSelectTerminator() {
		column.Name = p.advance().value
	}

	return column, nil
}

func (p *parser) parseQualifiedSelectItem() (querier_dto.RawOutputColumn, error) {
	tableAlias := p.advance().value
	p.advance()

	if p.current().kind == tokenStar {
		p.advance()
		return querier_dto.RawOutputColumn{
			IsStar:     true,
			TableAlias: tableAlias,
		}, nil
	}

	columnName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.RawOutputColumn{}, err
	}
	column := querier_dto.RawOutputColumn{
		Name:       columnName,
		TableAlias: tableAlias,
		ColumnName: columnName,
	}
	if p.matchKeyword(keywordAS) {
		alias, aliasErr := p.parseIdentifierOrKeyword()
		if aliasErr != nil {
			return querier_dto.RawOutputColumn{}, aliasErr
		}
		column.Name = alias
	} else if p.current().kind == tokenIdentifier && !p.isSelectTerminator() {
		column.Name = p.advance().value
	}

	return column, nil
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
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT,
		keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING)
}
