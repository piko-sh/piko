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
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword(keywordSELECT)
	p.skipSelectModifiers()

	outputColumns, err := p.parseOutputColumns()
	if err != nil {
		return nil, err
	}
	analysis.OutputColumns = outputColumns

	if err := p.parseFromClauseIfPresent(analysis); err != nil {
		return nil, err
	}

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereClause()
	}

	if p.matchKeyword(keywordGROUP) {
		p.matchKeyword(keywordBY)
		analysis.GroupByColumns = p.parseGroupByClause()
	}

	if p.matchKeyword(keywordHAVING) {
		p.parseWhereClause()
	}

	if err := p.parseCompoundBranches(analysis); err != nil {
		return nil, err
	}

	p.parseOrderByIfPresent()
	p.parseLimitOffsetIfPresent()
	p.skipForUpdateClause()

	analysis.ReadOnly = !p.hasForUpdate && !p.hasDataModifyingCTE
	p.finaliseAnalysis(analysis)
	return analysis, nil
}

func (p *parser) skipSelectModifiers() {
	for p.isAnyKeyword(keywordALL, "DISTINCT", "HIGH_PRIORITY", "STRAIGHT_JOIN",
		"SQL_SMALL_RESULT", "SQL_BIG_RESULT", "SQL_BUFFER_RESULT",
		"SQL_NO_CACHE", "SQL_CALC_FOUND_ROWS") {
		p.advance()
	}
}

func (p *parser) parseCTEListIfPresent(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.isKeyword(keywordWITH) {
		return nil
	}
	cteDefinitions, err := p.parseCTEList()
	if err != nil {
		return err
	}
	analysis.CTEDefinitions = cteDefinitions
	return nil
}

func (p *parser) parseFromClauseIfPresent(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.matchKeyword(keywordFROM) {
		return nil
	}
	fromTables, joinClauses, err := p.parseFromClause()
	if err != nil {
		return err
	}
	analysis.FromTables = fromTables
	analysis.JoinClauses = joinClauses
	return nil
}

func (p *parser) parseCompoundBranches(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		compoundOperator := p.parseCompoundQuery()
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

func (p *parser) parseOrderByIfPresent() {
	if p.matchKeyword(keywordORDER) {
		p.matchKeyword(keywordBY)
		p.parseOrderByList()
	}
}

func (p *parser) parseLimitOffsetIfPresent() {
	if p.isKeyword(keywordLIMIT) || p.isKeyword(keywordOFFSET) {
		p.parseLimitOffset()
	}
}

func (p *parser) finaliseAnalysis(analysis *querier_dto.RawQueryAnalysis) {
	analysis.ParameterReferences = p.parameterRefs
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
}

func (p *parser) analyseInsert() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword("INSERT", keywordREPLACE)
	p.matchKeyword(keywordIGNORE)
	p.matchKeyword("INTO")

	schema, tableName := p.mustSchemaQualifiedName()
	alias := p.parseOptionalAlias()
	analysis.FromTables = []querier_dto.TableReference{{Schema: schema, Name: tableName, Alias: alias}}

	columnNames, err := p.parseInsertColumnList()
	if err != nil {
		return nil, err
	}

	p.parseInsertValues(tableName, columnNames)

	if p.matchKeyword(keywordON) {
		p.parseOnDuplicateKeyUpdate(tableName)
	}

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	analysis.InsertTable = tableName
	analysis.InsertColumns = columnNames
	return analysis, nil
}

func (p *parser) parseOptionalAlias() string {
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			return p.advance().value
		}
	}
	return ""
}

func (p *parser) parseInsertColumnList() ([]string, error) {
	if p.current().kind == tokenLeftParen && !p.isKeyword(keywordSELECT) && !p.isKeyword(keywordVALUES) {
		return p.parseColumnList()
	}
	return nil, nil
}

func (p *parser) parseInsertValues(tableName string, columnNames []string) {
	if p.matchKeyword(keywordVALUES) || p.matchKeyword("VALUE") {
		p.parseValuesClause(tableName, columnNames)
	} else if p.matchKeyword(keywordSET) {
		p.parseSetClause(tableName)
	} else if p.isKeyword(keywordSELECT) || p.isKeyword(keywordWITH) || p.current().kind == tokenLeftParen {
		p.parseInsertSource()
	}
}

func (p *parser) parseReturningIfPresent(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.matchKeyword(keywordRETURNING) {
		return nil
	}
	analysis.HasReturning = true
	outputColumns, err := p.parseReturningClause()
	if err != nil {
		return err
	}
	analysis.OutputColumns = outputColumns
	return nil
}

func (p *parser) analyseUpdate() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword("UPDATE")

	fromTables, joinClauses, err := p.parseUpdateTableReferences()
	if err != nil {
		return nil, err
	}
	analysis.FromTables = fromTables
	analysis.JoinClauses = joinClauses

	p.mustKeyword(keywordSET)

	tableName := ""
	if len(fromTables) > 0 {
		tableName = fromTables[0].Name
	}
	p.parseSetClause(tableName)

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereClause()
	}

	p.parseOrderByIfPresent()
	p.parseLimitOffsetIfPresent()

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

func (p *parser) parseUpdateTableReferences() ([]querier_dto.TableReference, []querier_dto.JoinClause, error) {
	var tables []querier_dto.TableReference
	var joins []querier_dto.JoinClause

	tableRef := p.parseTableReference()
	tables = append(tables, tableRef)

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
		p.advance()
		additionalTable := p.parseTableReference()
		tables = append(tables, additionalTable)
	}

	return tables, joins, nil
}

func (p *parser) analyseDelete() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword("DELETE")

	if p.isKeyword(keywordFROM) {
		return p.parseSimpleDelete(analysis)
	}
	return p.parseMultiTableDelete(analysis)
}

func (p *parser) parseSimpleDelete(analysis *querier_dto.RawQueryAnalysis) (*querier_dto.RawQueryAnalysis, error) {
	p.mustKeyword(keywordFROM)

	schema, tableName := p.mustSchemaQualifiedName()
	alias := p.parseDeleteAlias()
	analysis.FromTables = []querier_dto.TableReference{{Schema: schema, Name: tableName, Alias: alias}}

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereClause()
	}

	p.parseOrderByIfPresent()
	p.parseLimitOffsetIfPresent()

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

func (p *parser) parseMultiTableDelete(analysis *querier_dto.RawQueryAnalysis) (*querier_dto.RawQueryAnalysis, error) {
	for p.current().kind == tokenIdentifier {
		_, name := p.mustSchemaQualifiedName()
		if p.current().kind == tokenDot && p.peek().kind == tokenStar {
			p.advance()
			p.advance()
		}
		_ = name
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	p.mustKeyword(keywordFROM)

	fromTables, joinClauses, err := p.parseFromClause()
	if err != nil {
		return nil, err
	}
	analysis.FromTables = fromTables
	analysis.JoinClauses = joinClauses

	if p.matchKeyword(keywordWHERE) {
		p.parseWhereClause()
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

func (p *parser) parseDeleteAlias() string {
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			return p.advance().value
		}
		return ""
	}
	if p.current().kind == tokenIdentifier && !p.isAnyKeyword(keywordUSING, keywordWHERE, keywordRETURNING, keywordORDER, keywordLIMIT) {
		return p.advance().value
	}
	return ""
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

	if p.isKeyword(keywordLIMIT) || p.isKeyword(keywordOFFSET) {
		p.parseLimitOffset()
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
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
