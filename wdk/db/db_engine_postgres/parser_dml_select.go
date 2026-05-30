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

// parseCTEList parses the WITH list of common table expressions.
//
// Returns []querier_dto.RawCTEDefinition which holds the parsed CTE entries.
// Returns error when a CTE definition fails to parse.
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

// parseSingleCTEDefinition parses one CTE definition within a WITH list.
//
// Takes isRecursive (bool) which indicates whether WITH RECURSIVE was given.
//
// Returns querier_dto.RawCTEDefinition which describes the parsed CTE.
// Returns error when the CTE name, column list, or body fail to parse.
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

// skipMaterialisationHint consumes an optional MATERIALIZED / NOT MATERIALIZED CTE hint.
func (p *parser) skipMaterialisationHint() {
	p.matchKeyword("MATERIALIZED")
	if p.matchKeyword(keywordNOT) {
		p.matchKeyword("MATERIALIZED")
	}
}

// analyseCTEBody runs the appropriate analyser over a CTE token slice.
//
// Takes cteTokens ([]token) which is the body of the CTE between parentheses.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysis result, or nil when parsing
// fails.
// Returns *parser which is the child parser used so callers can inherit parameter state.
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

// analyseCTEBodyDML dispatches the CTE body to the matching DML analyser.
//
// Takes cteParser (*parser) which holds the CTE body tokens.
//
// Returns *querier_dto.RawQueryAnalysis which is the DML analysis result.
// Returns error when the underlying analyser fails.
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

// parseCTEColumnNames parses the optional column-name list before AS.
//
// Returns []string which is the parsed column-name list, or nil when absent.
// Returns error when the column list fails to parse.
func (p *parser) parseCTEColumnNames() ([]string, error) {
	if p.current().kind != tokenLeftParen || p.isKeyword(keywordAS) {
		return nil, nil
	}
	if p.peekForAS() {
		return nil, nil
	}
	return p.parseColumnList()
}

// populateCTEDefinition fills the CTE definition fields from its analysis.
//
// Takes definition (*querier_dto.RawCTEDefinition) which is the target to populate.
// Takes analysis (*querier_dto.RawQueryAnalysis) which provides the source columns and
// clauses.
// Takes columnNames ([]string) which optionally overrides the inferred output column
// names.
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

// peekForAS reports whether the matching ')' is followed by an AS keyword.
//
// Returns bool which is true when AS does not follow the matched ')'.
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

// parseColumnList parses a parenthesised comma-separated list of names.
//
// Returns []string which is the parsed name list.
// Returns error when an opening '(' is missing or a name fails to parse.
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

// parseOutputColumns parses the SELECT or RETURNING output column list.
//
// Returns []querier_dto.RawOutputColumn which holds the parsed columns.
// Returns error when a column fails to parse.
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

// parseOneOutputColumn parses a single output column including aliases.
//
// Returns querier_dto.RawOutputColumn which describes the parsed column.
// Returns error when alias parsing fails.
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

// parseOutputColumnAlias attaches an explicit or implicit alias to column.
//
// Takes column (*querier_dto.RawOutputColumn) which is the column to mutate.
//
// Returns error when the explicit alias identifier fails to parse.
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

// parseReturningClause parses a RETURNING column list.
//
// Returns []querier_dto.RawOutputColumn which holds the returned columns.
// Returns error when an output column fails to parse.
func (p *parser) parseReturningClause() ([]querier_dto.RawOutputColumn, error) {
	return p.parseOutputColumns()
}

// expressionToOutputColumn copies expression metadata onto column.
//
// Takes expression (querier_dto.Expression) which is the parsed expression.
// Takes column (*querier_dto.RawOutputColumn) which is the column to populate.
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

// isSelectTerminator reports whether the current token ends a SELECT list.
//
// Returns bool which is true when the current keyword terminates the list.
func (p *parser) isSelectTerminator() bool {
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT, keywordOFFSET,
		keywordFETCH, keywordFOR, keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING, "INTO", "WINDOW")
}

// parseFromClause parses a FROM clause including comma joins and explicit joins.
//
// Returns []querier_dto.TableReference which holds the parsed table sources.
// Returns []querier_dto.JoinClause which holds the parsed explicit joins.
// Returns error when a table source or join fails to parse.
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

// appendCommaJoinedTable parses a comma-joined table source and appends it.
//
// Takes tables (*[]querier_dto.TableReference) which is the slice to grow.
//
// Returns error when the table source fails to parse.
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

// appendExplicitJoin parses an explicit JOIN and appends it to joins.
//
// Takes joinKind (querier_dto.JoinKind) which classifies the join.
// Takes joins (*[]querier_dto.JoinClause) which is the slice to grow.
//
// Returns error when the joined table source fails to parse.
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

// parseTableSource parses a derived table, table-valued function, or table.
//
// Takes joinKind (querier_dto.JoinKind) which classifies the join context.
//
// Returns *querier_dto.TableReference which is the parsed table reference, or nil when
// the source was a derived table or TVF and was tracked elsewhere.
// Returns error when the source fails to parse.
func (p *parser) parseTableSource(joinKind querier_dto.JoinKind) (*querier_dto.TableReference, error) {
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(joinKind); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(joinKind)
		return nil, nil
	}
	return new(p.parseTableReference()), nil
}

// parseJoinCondition consumes an ON predicate or a USING column list.
func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseWhereClause()
		return
	}
	if p.matchKeyword(keywordUSING) && p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

// parseTableReference parses a schema-qualified table name with optional alias.
//
// Returns querier_dto.TableReference which describes the parsed reference.
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

// isSubqueryStart reports whether the current position begins a subquery.
//
// Returns bool which is true when '(' is followed by SELECT, WITH, or VALUES.
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

// isTableValuedFunctionStart reports whether the current position begins a table-valued
// function call.
//
// Returns bool which is true when an identifier is followed by '('.
func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!p.isAnyKeyword(keywordSELECT, keywordWITH, keywordVALUES)
}

// parseDerivedTable parses a parenthesised subquery used as a table source.
//
// Takes joinKind (querier_dto.JoinKind) which classifies the join context.
//
// Returns error when the inner query fails to parse or analyse.
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

// parseTableValuedFunction parses a table-valued function reference.
//
// Takes joinKind (querier_dto.JoinKind) which classifies the join context.
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

// parseTVFColumnDefinitions parses the column definition list of a TVF reference.
//
// Returns []querier_dto.TVFColumnDefinition which holds the parsed columns.
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

// joinKeywordEntry maps a join keyword to its join kind and OUTER flag.
type joinKeywordEntry struct {
	// kind classifies the join produced by the keyword.
	kind querier_dto.JoinKind

	// hasOuter indicates whether the keyword may be followed by OUTER.
	hasOuter bool
}

var (
	// joinKeywordDispatch maps JOIN-introducer keywords to their join entries.
	joinKeywordDispatch = map[string]joinKeywordEntry{
		"INNER": {kind: querier_dto.JoinInner},
		"LEFT":  {kind: querier_dto.JoinLeft, hasOuter: true},
		"RIGHT": {kind: querier_dto.JoinRight, hasOuter: true},
		"FULL":  {kind: querier_dto.JoinFull, hasOuter: true},
		"CROSS": {kind: querier_dto.JoinCross},
	}
)

// parseJoinKeyword consumes a join-introducing keyword sequence.
//
// Returns querier_dto.JoinKind which classifies the consumed join.
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

// isJoinKeyword reports whether the current token is a join-introducer.
//
// Returns bool which is true for JOIN, INNER, LEFT, RIGHT, FULL, CROSS, or NATURAL.
func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "NATURAL")
}
