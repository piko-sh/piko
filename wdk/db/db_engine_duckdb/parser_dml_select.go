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

// parseCTEList parses a WITH ... clause and returns the CTE definitions.
//
// Returns []querier_dto.RawCTEDefinition which lists the parsed CTEs in declared order.
// Returns error when an individual CTE fails to parse.
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

// parseSingleCTEDefinition parses one CTE definition within a WITH clause.
//
// Takes isRecursive (bool) which records whether the enclosing WITH used the RECURSIVE
// keyword.
//
// Returns querier_dto.RawCTEDefinition which describes the parsed CTE.
// Returns error when parsing the name, columns, or body fails.
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

// skipMaterialisationHint consumes an optional MATERIALIZED or NOT MATERIALIZED hint
// between AS and the CTE body.
func (p *parser) skipMaterialisationHint() {
	p.matchKeyword("MATERIALIZED")
	if p.matchKeyword(keywordNOT) {
		p.matchKeyword("MATERIALIZED")
	}
}

// analyseCTEBody analyses a CTE body's tokens, dispatching to SELECT, VALUES, or DML
// analysis as appropriate.
//
// Takes cteTokens ([]token) which are the parenthesised tokens of the CTE body.
//
// Returns *querier_dto.RawQueryAnalysis which is the body's structural analysis, or nil
// when analysis failed.
// Returns *parser which is the child parser used so its parameter state can be merged by
// the caller.
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

// analyseCTEBodyDML analyses a CTE body that begins with INSERT, UPDATE, or DELETE.
//
// Takes cteParser (*parser) which is the parser positioned at the CTE body's first token.
//
// Returns *querier_dto.RawQueryAnalysis which is the body's analysis.
// Returns error when the underlying DML analysis fails.
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

// parseCTEColumnNames parses the optional column list preceding AS in a CTE definition.
//
// Returns []string which lists the parsed column names, or nil when no column list
// precedes AS.
// Returns error when parsing an individual identifier fails.
func (p *parser) parseCTEColumnNames() ([]string, error) {
	if p.current().kind != tokenLeftParen || p.isKeyword(keywordAS) {
		return nil, nil
	}
	if p.peekForAS() {
		return nil, nil
	}
	return p.parseColumnList()
}

// populateCTEDefinition copies the body analysis and explicit column names onto a CTE
// definition.
//
// Takes definition (*querier_dto.RawCTEDefinition) which is the CTE record to populate.
// Takes analysis (*querier_dto.RawQueryAnalysis) which provides the body's output
// columns, from-tables, and joins.
// Takes columnNames ([]string) which optionally overrides the body's output column names.
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

// peekForAS looks ahead past a balanced parenthesis run to detect whether a CTE column
// list precedes AS rather than the body.
//
// Returns bool which is true when the current parenthesis run is the CTE column list,
// false when it is the body.
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

// parseColumnList parses a parenthesised comma-separated list of column identifiers.
//
// Returns []string which lists the parsed names in declared order.
// Returns error when the opening parenthesis is missing or an identifier cannot be
// parsed.
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

// parseOutputColumns parses the SELECT or RETURNING projection list.
//
// Returns []querier_dto.RawOutputColumn which lists the parsed columns in declared order.
// Returns error when an individual column fails to parse.
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

// parseOneOutputColumn parses a single projection list entry, handling star, qualified
// star, expression, and alias forms.
//
// Returns querier_dto.RawOutputColumn which describes the parsed entry.
// Returns error when parsing the optional alias fails.
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

// parseOutputColumnAlias parses an AS alias or bare identifier alias for an output column
// and writes it onto column.
//
// Takes column (*querier_dto.RawOutputColumn) which is the column record to update with
// the alias.
//
// Returns error when parsing an AS alias identifier fails.
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

// parseReturningClause parses the projection list of a RETURNING clause.
//
// Returns []querier_dto.RawOutputColumn which lists the parsed columns in declared order.
// Returns error when an individual column fails to parse.
func (p *parser) parseReturningClause() ([]querier_dto.RawOutputColumn, error) {
	return p.parseOutputColumns()
}

// expressionToOutputColumn copies attributes from an expression into a raw output column,
// recognising column references and function calls.
//
// Takes expression (querier_dto.Expression) which is the parsed projection expression.
// Takes column (*querier_dto.RawOutputColumn) which receives the projected name,
// expression, and qualifiers.
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

// isSelectTerminator reports whether the current token is a keyword that ends a SELECT
// projection or table source.
//
// Returns bool which is true when the current keyword terminates a SELECT sub-clause.
func (p *parser) isSelectTerminator() bool {
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT, keywordOFFSET,
		keywordFETCH, keywordFOR, keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING, "INTO", "WINDOW")
}

// parseFromClause parses a FROM clause, including comma-separated tables and explicit
// JOIN clauses.
//
// Returns []querier_dto.TableReference which lists the comma-joined base tables in
// declared order.
// Returns []querier_dto.JoinClause which lists the explicit joins in declared order.
// Returns error when parsing a table source or join condition fails.
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

// appendCommaJoinedTable appends one comma-joined table source to tables.
//
// Takes tables (*[]querier_dto.TableReference) which is the slice to extend with the
// parsed table reference.
//
// Returns error when parsing the table source fails.
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

// appendExplicitJoin appends one explicit JOIN clause to joins, including its ON or USING
// condition.
//
// Takes joinKind (querier_dto.JoinKind) which is the parsed join kind.
// Takes joins (*[]querier_dto.JoinClause) which is the slice to extend with the parsed
// join.
//
// Returns error when parsing the table source fails.
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

// parseTableSource parses a table source as a base table, derived table, PIVOT or
// UNPIVOT, or table-valued function.
//
// Takes joinKind (querier_dto.JoinKind) which records how the source is joined into the
// FROM clause.
//
// Returns *querier_dto.TableReference which is the base table reference, or nil when the
// source was consumed via a derived-table, pivot, or table-valued function path.
// Returns error when parsing a derived table fails.
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

// isPivotOrUnpivot reports whether the current token introduces a PIVOT or UNPIVOT
// clause.
//
// Returns bool which is true for PIVOT or UNPIVOT keywords.
func (p *parser) isPivotOrUnpivot() bool {
	return p.isKeyword(keywordPIVOT) || p.isKeyword(keywordUNPIVOT)
}

// parsePivotOrUnpivot parses a PIVOT or UNPIVOT clause and records its alias on the
// derived-table list.
//
// Takes joinKind (querier_dto.JoinKind) which records how the pivot is joined into the
// FROM clause.
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

// isJoinOrClauseKeyword reports whether the current keyword starts a join or ends the
// current FROM table source.
//
// Returns bool which is true when the current keyword cannot be an alias.
func (p *parser) isJoinOrClauseKeyword() bool {
	return p.isAnyKeyword(keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT,
		keywordOFFSET, keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", keywordON,
		keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordQUALIFY, keywordPOSITIONAL)
}

// parseJoinCondition consumes an optional ON predicate or USING column list following a
// JOIN clause.
func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseWhereClause()
		return
	}
	if p.matchKeyword(keywordUSING) && p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

// parseTableReference parses a schema-qualified table reference with an optional alias.
//
// Returns querier_dto.TableReference which describes the parsed schema, name, and alias.
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

// isSubqueryStart reports whether the cursor sits on a "(" that begins a derived-table
// subquery.
//
// Returns bool which is true when the next tokens form a SELECT, WITH, or VALUES
// subquery.
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

// isTableValuedFunctionStart reports whether the cursor sits on an identifier followed by
// "(" that introduces a table-valued function.
//
// Returns bool which is true when the next tokens form a function call that is not
// SELECT, WITH, or VALUES.
func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!p.isAnyKeyword(keywordSELECT, keywordWITH, keywordVALUES)
}

// parseDerivedTable parses a parenthesised SELECT subquery and records it on the
// derived-table list with its alias and join kind.
//
// Takes joinKind (querier_dto.JoinKind) which records how the derived table is joined
// into the FROM clause.
//
// Returns error when the inner SELECT fails to parse or analyse.
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

// parseTableValuedFunction parses a table-valued function call along with its optional
// WITH ORDINALITY, alias, and column definition list.
//
// Takes joinKind (querier_dto.JoinKind) which records how the call is joined into the
// FROM clause.
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

// parseTVFColumnDefinitions parses an explicit column definition list attached to a
// table-valued function alias.
//
// Returns []querier_dto.TVFColumnDefinition which lists the parsed name and type pairs in
// declared order.
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

// joinKeywordEntry maps a join keyword to its kind and whether OUTER may follow.
type joinKeywordEntry struct {
	// kind is the join kind produced when this keyword is matched.
	kind querier_dto.JoinKind

	// hasOuter records whether an OUTER keyword may follow.
	hasOuter bool
}

var (
	// joinKeywordDispatch maps each recognised join-prefix keyword to its dispatch entry.
	joinKeywordDispatch = map[string]joinKeywordEntry{
		"INNER":           {kind: querier_dto.JoinInner},
		"LEFT":            {kind: querier_dto.JoinLeft, hasOuter: true},
		"RIGHT":           {kind: querier_dto.JoinRight, hasOuter: true},
		"FULL":            {kind: querier_dto.JoinFull, hasOuter: true},
		"CROSS":           {kind: querier_dto.JoinCross},
		keywordPOSITIONAL: {kind: querier_dto.JoinPositional},
	}
)

// parseJoinKeyword consumes a join prefix keyword along with optional NATURAL, OUTER, and
// JOIN tokens.
//
// Returns querier_dto.JoinKind which is the kind of the consumed join.
// Returns bool which is true when a join keyword was consumed.
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

// isJoinKeyword reports whether the current token introduces a join clause that would
// terminate an alias scan.
//
// Returns bool which is true for any join-prefix keyword.
func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "NATURAL")
}
