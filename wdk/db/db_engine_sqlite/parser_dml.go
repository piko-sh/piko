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

// isParameterToken reports whether a token kind is one of the placeholder forms.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool which is true for question marks, numbered, and named parameters.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenQuestionMark || kind == tokenNumberedParam || kind == tokenNamedParam
}

// analyseSelect parses a SELECT statement and produces the raw query analysis with output
// columns, parameters, and table references.
//
// Returns *querier_dto.RawQueryAnalysis which describes the SELECT.
// Returns error when the statement fails to parse.
func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	if p.analysisDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.analysisDepth++
	defer func() { p.analysisDepth-- }()

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
	analysis.PredicateSubqueries = p.predicateSubqueries
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
	return analysis, nil
}

// parseSelectBody parses FROM, WHERE, GROUP BY, and HAVING clauses into the analysis
// record.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed clause data.
//
// Returns error when a clause fails to parse.
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
		analysis.HasWhereClause = true
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

// parseSelectCompoundBranches parses UNION, INTERSECT, and EXCEPT branches into the
// analysis record.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives any parsed branch
// entries.
//
// Returns error when a branch fails to parse.
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

// parseSelectTrailer consumes the trailing ORDER BY and LIMIT clauses of a SELECT
// statement.
func (p *parser) parseSelectTrailer() {
	if p.matchKeyword(keywordORDER) {
		p.matchKeyword("BY")
		p.parseOrderByList()
	}

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitOffset()
	}
}

// analyseInsert parses an INSERT or REPLACE statement and produces the raw query
// analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the INSERT.
// Returns error when the statement fails to parse.
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

	insertSelect, valuesError := p.parseInsertValues(tableName, columnNames)
	if valuesError != nil {
		return nil, valuesError
	}
	analysis.InsertSelect = insertSelect

	for p.matchKeyword(keywordON) {
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
	analysis.PredicateSubqueries = p.predicateSubqueries
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
	return analysis, nil
}

// analyseUpdate parses an UPDATE statement and produces the raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the UPDATE.
// Returns error when the statement fails to parse.
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
		analysis.HasWhereClause = true
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
	analysis.PredicateSubqueries = p.predicateSubqueries
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
	return analysis, nil
}

// analyseDelete parses a DELETE statement and produces the raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the DELETE.
// Returns error when the statement fails to parse.
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
		analysis.HasWhereClause = true
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
	analysis.PredicateSubqueries = p.predicateSubqueries
	analysis.RawDerivedTables = p.rawDerivedTables
	analysis.RawTableValuedFunctions = p.rawTableValuedFunctions
	return analysis, nil
}

// analyseValues parses a bare VALUES statement and produces the raw query analysis.
//
// Returns *querier_dto.RawQueryAnalysis which describes the VALUES statement, marked
// read-only.
// Returns error when the statement fails to parse.
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

// parseValuesFirstRow parses the first row of a VALUES expression and returns synthetic
// column names for each expression.
//
// Returns []querier_dto.RawOutputColumn which describes the first row's columns.
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

// skipValuesTrailingRows consumes any rows beyond the first in a VALUES expression while
// still registering placeholder parameters.
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

// parseWithClause parses the WITH clause and returns one definition per common table
// expression.
//
// Returns []querier_dto.RawCTEDefinition which holds the parsed definitions in order.
// Returns error when a CTE fails to parse.
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

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return definitions, nil
}

// parseSingleCTE parses one common table expression definition.
//
// Takes isRecursive (bool) which is true when the WITH clause is RECURSIVE.
//
// Returns querier_dto.RawCTEDefinition which describes the CTE.
// Returns error when the definition fails to parse.
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
	cteParser.parameterCount = p.parameterCount
	cteParser.analysisDepth = p.analysisDepth
	cteParser.expressionDepth = p.expressionDepth
	cteParser.maxParseDepth = p.maxParseDepth
	cteAnalysis, analyseErr := cteParser.analyseSelect()

	definition := querier_dto.RawCTEDefinition{
		Name:        cteName,
		IsRecursive: isRecursive,
	}

	if analyseErr == nil {
		definition.OutputColumns = buildCTEOutputColumns(columnNames, cteAnalysis)
		definition.FromTables = cteAnalysis.FromTables
		definition.JoinClauses = cteAnalysis.JoinClauses

		definition.ParameterReferences = cteAnalysis.ParameterReferences
	}

	p.parameterCount = cteParser.parameterCount
	p.parameterRefs = append(p.parameterRefs, cteParser.parameterRefs...)

	return definition, nil
}

// parseCTEColumnNames parses the optional column-name list that may follow a CTE name
// before the AS keyword.
//
// Returns []string which is the column-name list, or nil when absent.
// Returns error when the list is malformed.
func (p *parser) parseCTEColumnNames() ([]string, error) {
	if p.current().kind != tokenLeftParen || p.isKeyword(keywordAS) {
		return nil, nil
	}
	if p.peekForAS() {
		return nil, nil
	}
	return p.parseColumnList()
}

// buildCTEOutputColumns chooses the output columns advertised by a CTE.
//
// Takes columnNames ([]string) which is the explicit CTE column list, if any.
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds the inner query's analysis.
//
// Returns []querier_dto.RawOutputColumn which is the explicit list when provided,
// otherwise the inner query's columns.
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

// peekForAS reports whether a parenthesised group followed by AS lies ahead, used to
// distinguish CTE column lists from subquery clauses.
//
// Returns bool which is true when the lookahead does not find AS after a matching close
// parenthesis.
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

// isFollowedByAS reports whether the token after the supplied position is the AS keyword.
//
// Takes position (int) which is the index whose successor is checked.
//
// Returns bool which is true when the next token is AS.
func (p *parser) isFollowedByAS(position int) bool {
	return position+1 < len(p.tokens) &&
		p.tokens[position+1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[position+1].value, keywordAS)
}

// parseSelectList parses the comma-separated select list.
//
// When this select list is the body of an INSERT ... SELECT, its top-level projection
// placeholders are bound positionally to the INSERT target columns (the same binding the
// VALUES path applies), so an INSERT ... SELECT $1, $2 types $1/$2 from the target
// columns rather than leaving them untyped. The INSERT target is consumed and cleared up
// front so nested subqueries and compound branches parsed inside the items do not inherit
// it.
//
// Returns []querier_dto.RawOutputColumn which is the parsed output column list.
// Returns error when an item fails to parse.
func (p *parser) parseSelectList() ([]querier_dto.RawOutputColumn, error) {
	insertTargetTable := p.insertTargetTable
	insertTargetColumns := p.insertTargetColumns
	p.insertTargetTable = ""
	p.insertTargetColumns = nil

	var columns []querier_dto.RawOutputColumn

	projectionOrdinal := 0
	for {
		refsCountBefore := len(p.parameterRefs)
		column, err := p.parseSelectItem()
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
		if insertTargetTable != "" {
			p.bindInsertSelectProjection(refsCountBefore, insertTargetTable, insertTargetColumns, projectionOrdinal)
		}
		projectionOrdinal++

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns, nil
}

// bindInsertSelectProjection binds the placeholders registered while parsing one
// top-level INSERT ... SELECT projection item to the matching INSERT target column.
//
// Only placeholders that do not already carry a column reference are bound, so a deeper
// comparison/LIKE column reference is preserved; the assignment context and the qualified
// target column reference mirror the VALUES path so the analyser types the placeholder
// from the target column. The boundary is the parameterRefs length (references are
// appended) so out-of-order numbered placeholders are handled correctly.
//
// Takes refsCountBefore (int) which is len(parameterRefs) before the item was parsed.
// Takes tableName (string) which is the INSERT target table name (the column ref alias).
// Takes columnNames ([]string) which are the INSERT target column names in positional
// order.
// Takes projectionOrdinal (int) which is the 0-based projection slot of the item.
func (p *parser) bindInsertSelectProjection(refsCountBefore int, tableName string, columnNames []string, projectionOrdinal int) {
	columnReference := columnRefForIndex(tableName, columnNames, projectionOrdinal)
	if columnReference == nil {
		return
	}
	for i := refsCountBefore; i < len(p.parameterRefs); i++ {
		if p.parameterRefs[i].ColumnReference == nil {
			p.parameterRefs[i].Context = querier_dto.ParameterContextAssignment
			p.parameterRefs[i].ColumnReference = columnReference
		}
	}
}

// parseSelectItem parses a single select-list item: star, qualified reference, or
// expression with optional alias.
//
// Returns querier_dto.RawOutputColumn which describes the parsed item.
// Returns error when the item fails to parse.
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

// parseQualifiedSelectItem parses an alias-qualified select item such as table.* or
// table.column.
//
// Returns querier_dto.RawOutputColumn which describes the qualified item.
// Returns error when the item fails to parse.
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

// expressionToOutputColumn copies the relevant fields from an expression into an
// output-column record.
//
// Takes expression (querier_dto.Expression) which is the parsed select expression.
// Takes column (*querier_dto.RawOutputColumn) which receives the derived column metadata.
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

// isSelectTerminator reports whether the current token marks the end of the select list.
//
// Returns bool which is true when the token starts a clause that follows the select list.
func (p *parser) isSelectTerminator() bool {
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT,
		keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING)
}
