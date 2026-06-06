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

// analyseSelect parses a SELECT statement into a raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the parsed query.
// Returns error when the statement is malformed.
func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	if p.analysisDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.analysisDepth++
	defer func() { p.analysisDepth-- }()

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
		analysis.HasWhereClause = true
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

// parseCTEListIfPresent parses a WITH clause when present.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed CTE
// definitions.
//
// Returns error when the CTE list cannot be parsed.
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

// parseFromClauseIfPresent parses a FROM clause when present.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed tables and
// join clauses.
//
// Returns error when the FROM clause cannot be parsed.
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

// parseCompoundBranches parses UNION, INTERSECT, and EXCEPT branches.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives each parsed compound
// branch.
//
// Returns error when a branch cannot be parsed.
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

// parseOrderByIfPresent consumes an ORDER BY clause when present.
func (p *parser) parseOrderByIfPresent() {
	if p.matchKeyword(keywordORDER) {
		p.matchKeyword(keywordBY)
		p.parseOrderByList()
	}
}

// parseLimitOffsetIfPresent consumes a LIMIT, FETCH, or OFFSET clause.
func (p *parser) parseLimitOffsetIfPresent() {
	if p.isKeyword(keywordLIMIT) || p.isKeyword(keywordFETCH) || p.isKeyword(keywordOFFSET) {
		p.parseLimitOffset()
	}
}

// finaliseAnalysis copies parser-accumulated state into the analysis result.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives parameter references,
// derived tables, and table-valued function records.
func (p *parser) finaliseAnalysis(analysis *querier_dto.RawQueryAnalysis) {
	analysis.ParameterReferences = p.parameterRefs
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.PredicateSubqueries = p.predicateSubqueries
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
}

// analyseInsert parses an INSERT statement into a raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the parsed insert.
// Returns error when the statement is malformed.
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
	insertSelect, valuesError := p.parseInsertValues(tableName, columnNames)
	if valuesError != nil {
		return nil, valuesError
	}
	analysis.InsertSelect = insertSelect

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

// parseOptionalAlias consumes an optional AS alias for a table reference.
//
// Returns string which is the alias when present, else empty.
func (p *parser) parseOptionalAlias() string {
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			return p.advance().value
		}
	}
	return ""
}

// parseInsertColumnList parses the optional column list of an INSERT.
//
// Returns []string which lists the columns when a list is present, else nil.
// Returns error when the column list cannot be parsed.
func (p *parser) parseInsertColumnList() ([]string, error) {
	if p.current().kind == tokenLeftParen && !p.isKeyword(keywordSELECT) && !p.isKeyword(keywordVALUES) {
		return p.parseColumnList()
	}
	return nil, nil
}

// skipOverridingClause consumes an OVERRIDING SYSTEM/USER VALUE clause.
func (p *parser) skipOverridingClause() {
	if p.matchKeyword("OVERRIDING") {
		p.matchKeyword("SYSTEM")
		p.matchKeyword("USER")
		p.matchKeyword("VALUE")
	}
}

// parseInsertValues parses the VALUES, DEFAULT VALUES, or subquery payload.
//
// Takes tableName (string) which scopes column lookups in the values clause.
// Takes columnNames ([]string) which lists the target columns for parameter binding when
// present.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed SELECT body of an INSERT
// ... SELECT (nil for VALUES/DEFAULT VALUES/parenthesised sources).
// Returns error when the SELECT body fails to parse.
func (p *parser) parseInsertValues(tableName string, columnNames []string) (*querier_dto.RawQueryAnalysis, error) {
	switch {
	case p.matchKeyword(keywordVALUES):
		p.parseValuesClause(tableName, columnNames)
	case p.matchKeyword(keywordDEFAULT):
		p.matchKeyword(keywordVALUES)
	case p.isKeyword(keywordSELECT) || p.isKeyword(keywordWITH):

		p.insertProjectionTable = tableName
		p.insertProjectionColumns = columnNames
		p.insertProjectionIndex = 0
		analysis, selectError := p.analyseSelect()
		p.insertProjectionTable = ""
		p.insertProjectionColumns = nil
		p.insertProjectionIndex = 0
		return analysis, selectError
	case p.current().kind == tokenLeftParen:
		p.parseInsertSource()
	}
	return nil, nil
}

// parseReturningIfPresent parses a RETURNING clause when present.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed output columns
// and the HasReturning flag.
//
// Returns error when the RETURNING clause cannot be parsed.
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

// analyseUpdate parses an UPDATE statement into a raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the parsed update.
// Returns error when the statement is malformed.
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

	p.parseWhereOrCurrentOf(analysis)

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

// parseUpdateAlias parses the optional alias of an UPDATE target table.
//
// Returns string which is the alias when present, else empty.
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

// parseAdditionalFromClause parses the optional FROM clause of an UPDATE.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed additional
// tables and join clauses.
//
// Returns error when the FROM clause cannot be parsed.
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

// parseWhereOrCurrentOf consumes an optional WHERE clause, including the
// Postgres-specific WHERE CURRENT OF <cursor> form used in UPDATE and DELETE.
//
// It sets analysis.HasWhereClause when a WHERE keyword is matched so the runtime query
// builder appends additional predicates with " AND " rather than " WHERE ".
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the HasWhereClause flag.
func (p *parser) parseWhereOrCurrentOf(analysis *querier_dto.RawQueryAnalysis) {
	if !p.matchKeyword(keywordWHERE) {
		return
	}
	analysis.HasWhereClause = true
	if p.matchKeyword(keywordCURRENT) {
		p.matchKeyword("OF")
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
		return
	}
	p.parseWhereClause()
}

// analyseDelete parses a DELETE statement into a raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the parsed delete.
// Returns error when the statement is malformed.
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

	p.parseWhereOrCurrentOf(analysis)

	if err := p.parseReturningIfPresent(analysis); err != nil {
		return nil, err
	}

	p.finaliseAnalysis(analysis)
	return analysis, nil
}

// parseDeleteAlias parses the optional alias of a DELETE target table.
//
// Returns string which is the alias when present, else empty.
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

// parseUsingClause parses an optional USING clause of a DELETE.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed additional
// tables and join clauses.
//
// Returns error when the USING clause cannot be parsed.
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

// analyseValues parses a top-level VALUES statement into a query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the parsed values.
// Returns error which is always nil for VALUES analysis.
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

// parseDistinctOn consumes a DISTINCT ON (expr, ...) clause.
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
