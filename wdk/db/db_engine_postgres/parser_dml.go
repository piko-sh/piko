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
	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword(keywordSELECT)

	if p.matchKeyword("DISTINCT") {
		p.parseDistinctOn()
	}
	p.matchKeyword(keywordALL)

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
	if p.isKeyword(keywordLIMIT) || p.isKeyword(keywordFETCH) || p.isKeyword(keywordOFFSET) {
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

	p.mustKeyword("INSERT")
	p.matchKeyword("INTO")

	schema, tableName := p.mustSchemaQualifiedName()
	alias := p.parseOptionalAlias()
	analysis.FromTables = []querier_dto.TableReference{{Schema: schema, Name: tableName, Alias: alias}}

	columnNames, err := p.parseInsertColumnList()
	if err != nil {
		return nil, err
	}

	p.skipOverridingClause()
	p.parseInsertValues(tableName, columnNames)

	if p.matchKeyword(keywordON) {
		p.parseOnConflict(tableName)
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

func (p *parser) skipOverridingClause() {
	if p.matchKeyword("OVERRIDING") {
		p.matchKeyword("SYSTEM")
		p.matchKeyword("USER")
		p.matchKeyword("VALUE")
	}
}

func (p *parser) parseInsertValues(tableName string, columnNames []string) {
	if p.matchKeyword(keywordVALUES) {
		p.parseValuesClause(tableName, columnNames)
	} else if p.matchKeyword(keywordDEFAULT) {
		p.matchKeyword(keywordVALUES)
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
	p.matchKeyword("ONLY")

	schema, tableName := p.mustSchemaQualifiedName()
	alias := p.parseUpdateAlias()
	analysis.FromTables = []querier_dto.TableReference{{Schema: schema, Name: tableName, Alias: alias}}

	p.mustKeyword(keywordSET)
	p.parseSetClause(tableName)

	if err := p.parseAdditionalFromClause(analysis); err != nil {
		return nil, err
	}

	p.parseWhereOrCurrentOf()

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

func (p *parser) parseUpdateAlias() string {
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			return p.advance().value
		}
		return ""
	}
	if p.current().kind == tokenIdentifier && !p.isKeyword(keywordSET) {
		return p.advance().value
	}
	return ""
}

func (p *parser) parseAdditionalFromClause(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.matchKeyword(keywordFROM) {
		return nil
	}
	fromTables, joinClauses, err := p.parseFromClause()
	if err != nil {
		return err
	}
	analysis.FromTables = append(analysis.FromTables, fromTables...)
	analysis.JoinClauses = joinClauses
	return nil
}

func (p *parser) parseWhereOrCurrentOf() {
	if !p.matchKeyword(keywordWHERE) {
		return
	}
	if p.matchKeyword(keywordCURRENT) {
		p.matchKeyword("OF")
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
		return
	}
	p.parseWhereClause()
}

func (p *parser) analyseDelete() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{}

	if err := p.parseCTEListIfPresent(analysis); err != nil {
		return nil, err
	}

	p.mustKeyword("DELETE")
	p.mustKeyword(keywordFROM)
	p.matchKeyword("ONLY")

	schema, tableName := p.mustSchemaQualifiedName()
	alias := p.parseDeleteAlias()
	analysis.FromTables = []querier_dto.TableReference{{Schema: schema, Name: tableName, Alias: alias}}

	if err := p.parseUsingClause(analysis); err != nil {
		return nil, err
	}

	p.parseWhereOrCurrentOf()

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
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
	if p.current().kind == tokenIdentifier && !p.isAnyKeyword(keywordUSING, keywordWHERE, keywordRETURNING) {
		return p.advance().value
	}
	return ""
}

func (p *parser) parseUsingClause(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.matchKeyword(keywordUSING) {
		return nil
	}
	fromTables, joinClauses, err := p.parseFromClause()
	if err != nil {
		return err
	}
	analysis.FromTables = append(analysis.FromTables, fromTables...)
	analysis.JoinClauses = joinClauses
	return nil
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

	if p.isKeyword(keywordLIMIT) || p.isKeyword(keywordFETCH) || p.isKeyword(keywordOFFSET) {
		p.parseLimitOffset()
	}

	analysis.ParameterReferences = p.parameterRefs
	return analysis, nil
}

func (p *parser) parseDistinctOn() {
	if !p.matchKeyword(keywordON) {
		return
	}
	if p.current().kind != tokenLeftParen {
		return
	}
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
}
