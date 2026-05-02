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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseCTEList parses the comma-separated list of common-table-expression definitions
// following a WITH keyword.
//
// Returns []querier_dto.RawCTEDefinition which is the parsed CTE list.
// Returns error when any CTE definition fails to parse.
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

// parseSingleCTEDefinition parses one CTE definition including its optional column list
// and parenthesised body.
//
// Takes isRecursive (bool) which is true when the surrounding WITH clause carried the
// RECURSIVE keyword.
//
// Returns querier_dto.RawCTEDefinition which is the parsed definition.
// Returns error when the CTE name, body, or AS keyword cannot be parsed.
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

// analyseCTEBody analyses the inner query of a CTE definition.
//
// Takes cteTokens ([]token) which is the inner CTE token slice.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed query, or nil when analysis
// failed.
// Returns *parser which is the child parser used for the CTE body so the caller can
// splice its parameter state.
func (p *parser) analyseCTEBody(cteTokens []token) (*querier_dto.RawQueryAnalysis, *parser) {
	cteParser := newParser(cteTokens)
	cteParser.analysisDepth = p.analysisDepth
	cteParser.expressionDepth = p.expressionDepth
	cteParser.maxParseDepth = p.maxParseDepth
	var cteAnalysis *querier_dto.RawQueryAnalysis
	var analyseErr error

	switch {
	case cteParser.isKeyword(keywordVALUES):
		cteAnalysis, analyseErr = cteParser.analyseValues()
	case cteParser.isAnyKeyword("INSERT", "UPDATE", "DELETE", keywordREPLACE):
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

// analyseCTEBodyDML dispatches to the relevant DML analyser for the body of a
// data-modifying CTE.
//
// Takes cteParser (*parser) which is the parser positioned at the CTE body tokens.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed query.
// Returns error when the DML body cannot be analysed.
func (*parser) analyseCTEBodyDML(cteParser *parser) (*querier_dto.RawQueryAnalysis, error) {
	switch {
	case cteParser.isKeyword("INSERT"), cteParser.isKeyword(keywordREPLACE):
		return cteParser.analyseInsert()
	case cteParser.isKeyword("UPDATE"):
		return cteParser.analyseUpdate()
	default:
		return cteParser.analyseDelete()
	}
}

// parseCTEColumnNames parses the optional parenthesised column list that precedes the AS
// keyword in a CTE definition.
//
// Returns []string which is the parsed column list, or nil when absent.
// Returns error when the column list is malformed.
func (p *parser) parseCTEColumnNames() ([]string, error) {
	if p.current().kind != tokenLeftParen || p.isKeyword(keywordAS) {
		return nil, nil
	}
	if p.peekForAS() {
		return nil, nil
	}
	return p.parseColumnList()
}

// populateCTEDefinition fills in the structural fields of a CTE definition from the
// analysed body.
//
// Takes definition (*querier_dto.RawCTEDefinition) which receives the populated fields.
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysed CTE body.
// Takes columnNames ([]string) which is the explicit column list when present.
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

	definition.ParameterReferences = analysis.ParameterReferences
}

// peekForAS reports whether the parenthesised group at the current position is followed
// by an AS keyword.
//
// Returns bool which is true when no trailing AS keyword was found, used to disambiguate
// column lists from subqueries.
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

// parseOutputColumns parses the comma-separated SELECT projection list.
//
// Returns []querier_dto.RawOutputColumn which is the parsed projection.
// Returns error when an individual column fails to parse.
func (p *parser) parseOutputColumns() ([]querier_dto.RawOutputColumn, error) {
	var columns []querier_dto.RawOutputColumn

	if p.insertProjectionColumns != nil {
		p.insertProjectionIndex = 0
	}

	for {
		column, err := p.parseOneOutputColumn()
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)

		if p.insertProjectionColumns != nil {
			p.insertProjectionIndex++
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns, nil
}

// parseOneOutputColumn parses a single projection entry, handling star, qualified star
// and expression-with-alias forms.
//
// Returns querier_dto.RawOutputColumn which is the parsed column.
// Returns error when the alias clause is malformed.
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

// parseOutputColumnAlias parses the optional alias clause on a projection entry.
//
// Takes column (*querier_dto.RawOutputColumn) which is updated with the parsed alias when
// one is present.
//
// Returns error when the AS keyword is followed by an unparseable identifier.
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
// Returns []querier_dto.RawOutputColumn which is the parsed projection.
// Returns error when an individual column fails to parse.
func (p *parser) parseReturningClause() ([]querier_dto.RawOutputColumn, error) {
	return p.parseOutputColumns()
}

// expressionToOutputColumn unpacks a parsed expression into the relevant
// projection-column fields based on its concrete kind.
//
// Takes expression (querier_dto.Expression) which is the parsed projection expression.
// Takes column (*querier_dto.RawOutputColumn) which receives the unpacked fields.
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

// isSelectTerminator reports whether the current token starts a clause that ends a SELECT
// projection list.
//
// Returns bool reporting whether the current keyword terminates the projection.
func (p *parser) isSelectTerminator() bool {
	return p.isAnyKeyword(keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING, keywordORDER, keywordLIMIT, keywordOFFSET,
		keywordFOR, keywordUNION, keywordINTERSECT, keywordEXCEPT, keywordON, keywordRETURNING, "INTO", "WINDOW",
		"LOCK", "PROCEDURE")
}

// parseFromClause parses the FROM clause including comma-joined tables and explicit JOIN
// clauses.
//
// Returns []querier_dto.TableReference which is the list of base tables.
// Returns []querier_dto.JoinClause which is the list of explicit joins.
// Returns error when a table source or join clause fails to parse.
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

// appendCommaJoinedTable parses an additional comma-joined table source and appends it to
// the supplied slice.
//
// Takes tables (*[]querier_dto.TableReference) which is the slice to append to.
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

// appendExplicitJoin parses an explicit JOIN clause including its optional join
// condition.
//
// Takes joinKind (querier_dto.JoinKind) which is the kind of join being parsed.
// Takes joins (*[]querier_dto.JoinClause) which is the slice to append the parsed join
// to.
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

// parseTableSource parses a single source appearing in a FROM clause, covering
// subqueries, table-valued functions and base tables.
//
// Takes joinKind (querier_dto.JoinKind) which is the join kind to record on any derived
// or table-valued source.
//
// Returns *querier_dto.TableReference which is the parsed base-table reference, or nil
// when the source was captured as a derived table or table-valued function.
// Returns error when parsing the table source fails.
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
	p.skipIndexHints()
	return new(p.parseTableReference()), nil
}

// skipIndexHints consumes MySQL index hint clauses such as USE INDEX and FORCE INDEX
// following a table reference.
func (p *parser) skipIndexHints() {
	for p.isAnyKeyword("USE", "FORCE", keywordIGNORE) {
		p.advance()
		p.matchKeyword("INDEX")
		p.matchKeyword(keywordKEY)
		p.matchKeyword(keywordFOR)
		p.matchKeyword(keywordJOIN)
		p.matchKeyword(keywordORDER)
		p.matchKeyword(keywordBY)
		p.matchKeyword(keywordGROUP)
		p.matchKeyword(keywordBY)
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
	}
}

// parseJoinCondition consumes an optional ON or USING clause following a JOIN.
func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseJoinConditionExpression()
		return
	}
	if p.matchKeyword(keywordUSING) && p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

// parseTableReference parses a schema-qualified table name with an optional alias.
//
// Returns querier_dto.TableReference which is the parsed reference, or the zero value
// when the current token is not an identifier.
func (p *parser) parseTableReference() querier_dto.TableReference {
	if p.current().kind != tokenIdentifier {
		return querier_dto.TableReference{}
	}

	schema, name := p.mustSchemaQualifiedName()

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() &&
		!p.isAnyKeyword(keywordSET, keywordVALUES, keywordDEFAULT, keywordWHERE, "INNER", "LEFT", "RIGHT",
			"CROSS", "NATURAL", keywordJOIN, keywordON, keywordUSING, keywordLATERAL,
			"STRAIGHT_JOIN", "USE", "FORCE", keywordIGNORE) {
		alias = p.advance().value
	}

	return querier_dto.TableReference{Schema: schema, Name: name, Alias: alias}
}

// isSubqueryStart reports whether the current parenthesised group opens a subquery rather
// than a column list.
//
// Returns bool reporting whether a subquery starts at the current position.
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

// isTableValuedFunctionStart reports whether the current position starts a table-valued
// function invocation.
//
// Returns bool reporting whether the lookahead matches a function call.
func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!p.isAnyKeyword(keywordSELECT, keywordWITH, keywordVALUES)
}

// parseDerivedTable parses a parenthesised subquery used as a derived table, including
// its optional alias and column list.
//
// Takes joinKind (querier_dto.JoinKind) which is the join kind to record on the derived
// table.
//
// Returns error when the inner query fails to analyse.
func (p *parser) parseDerivedTable(joinKind querier_dto.JoinKind) error {
	innerTokens, collectError := p.collectParenthesised()
	if collectError != nil {
		return collectError
	}

	childParser := newParser(innerTokens)
	childParser.parameterCount = p.parameterCount
	childParser.analysisDepth = p.analysisDepth
	childParser.expressionDepth = p.expressionDepth
	childParser.maxParseDepth = p.maxParseDepth
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

// parseTableValuedFunction parses a table-valued function invocation in a FROM clause
// along with any alias and column-definition list.
//
// Takes joinKind (querier_dto.JoinKind) which is the join kind to record on the function
// reference.
func (p *parser) parseTableValuedFunction(joinKind querier_dto.JoinKind) {
	functionName := strings.ToLower(p.advance().value)
	p.advance()

	parameterCountBefore := p.parameterCount
	var argumentBoundaries []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseExpression()
		argumentBoundaries = append(argumentBoundaries, p.parameterCount)
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	p.markParametersAsFunctionArguments(parameterCountBefore, functionName, argumentBoundaries)

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

// parseTVFColumnDefinitions parses the parenthesised column-definition list that may
// follow a table-valued function alias.
//
// Returns []querier_dto.TVFColumnDefinition which is the parsed list.
func (p *parser) parseTVFColumnDefinitions() []querier_dto.TVFColumnDefinition {
	p.advance()
	var definitions []querier_dto.TVFColumnDefinition
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind != tokenIdentifier {
			break
		}
		name := p.advance().value
		var typeName string
		if p.current().kind == tokenIdentifier {
			typeName = p.parseCastTypeName()
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

// joinKeywordEntry captures how a leading join keyword maps to a join kind and whether
// the OUTER token may follow.
type joinKeywordEntry struct {
	// kind is the join kind selected when the keyword matches.
	kind querier_dto.JoinKind

	// hasOuter records whether OUTER may follow the leading keyword.
	hasOuter bool
}

var (
	// joinKeywordDispatch maps the leading join-keyword token to its joinKeywordEntry.
	joinKeywordDispatch = map[string]joinKeywordEntry{
		"INNER": {kind: querier_dto.JoinInner},
		"LEFT":  {kind: querier_dto.JoinLeft, hasOuter: true},
		"RIGHT": {kind: querier_dto.JoinRight, hasOuter: true},
		"CROSS": {kind: querier_dto.JoinCross},
	}
)

// parseJoinKeyword consumes the keyword sequence that opens a JOIN clause.
//
// Returns querier_dto.JoinKind which is the kind of join recognised.
// Returns bool which reports whether a join keyword sequence was consumed.
func (p *parser) parseJoinKeyword() (querier_dto.JoinKind, bool) {
	p.matchKeyword("NATURAL")

	if p.matchKeyword("STRAIGHT_JOIN") {
		return querier_dto.JoinInner, true
	}

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

// isJoinKeyword reports whether the current token introduces a JOIN clause.
//
// Returns bool reporting whether the current token is a join keyword.
func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "CROSS", "NATURAL", "STRAIGHT_JOIN")
}

// parseValuesFirstRow parses the first row of a VALUES statement, naming the resulting
// columns column1, column2, and so on.
//
// Returns []querier_dto.RawOutputColumn which is the synthesised projection inferred from
// the row.
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
