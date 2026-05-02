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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// parseCreateView parses a CREATE VIEW statement.
//
// Returns *querier_dto.CatalogueMutation which describes the new view.
// Returns error when the view name or column list cannot be parsed.
func (p *parser) parseCreateView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.mustKeyword("VIEW")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	viewName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, listError := p.parseColumnList()
		if listError != nil {
			return nil, listError
		}
		columnNames = names
	}

	p.mustKeyword(keywordAS)

	mutation := &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationCreateView,
		TableName: viewName,
	}

	bodyStart := p.position
	mutation.ViewDefinition = p.analyseViewBody(columnNames)

	if len(columnNames) == 0 {
		columnNames = inferViewColumnNames(p.tokens, bodyStart)
	}

	mutation.Columns = columnsFromNames(columnNames)

	return mutation, nil
}

// analyseViewBody analyses the SELECT body of a CREATE VIEW so the catalogue can store
// typed columns.
//
// Recovers from panics in the inner analyser so a malformed view body (e.g. CREATE VIEW v
// AS (SELECT ...) or CREATE VIEW v AS VALUES (...)) cannot crash the whole DDL apply.
//
// Takes columnNames ([]string) which is the declared column list overlaid onto the
// inferred projection names.
//
// Returns *querier_dto.RawQueryAnalysis which describes the view body, or nil when the
// body cannot be parsed and the catalogue falls back to the bare column-name list from
// inferViewColumnNames.
func (p *parser) analyseViewBody(columnNames []string) (result *querier_dto.RawQueryAnalysis) {
	remainingTokens := p.tokens[p.position:]
	if len(remainingTokens) == 0 {
		return nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil

			log.Warn("sqlite: view body analysis panic recovered",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
			)
		}
	}()

	viewParser := newParser(remainingTokens)
	viewParser.analysisDepth = p.analysisDepth
	viewParser.expressionDepth = p.expressionDepth
	viewParser.maxParseDepth = p.maxParseDepth
	if !viewParser.isKeyword(keywordSELECT) && !viewParser.isKeyword(keywordWITH) {
		return nil
	}
	viewAnalysis, analyseError := viewParser.analyseSelect()
	if analyseError != nil || viewAnalysis == nil {
		return nil
	}

	if len(columnNames) > 0 {
		overlayViewColumnNames(viewAnalysis, columnNames)
	}

	return viewAnalysis
}

// overlayViewColumnNames replaces the inferred names from the SELECT projection with the
// declared column list, preserving the resolved expression/column/table metadata so type
// resolution still works.
//
// Tolerates declared column lists that are longer than the SELECT projection (which is a
// degenerate but reachable shape via DDL) by appending name-only entries past the
// projection length rather than indexing out of bounds.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) whose output columns are renamed in
// place.
// Takes columnNames ([]string) which is the declared column list to overlay.
func overlayViewColumnNames(analysis *querier_dto.RawQueryAnalysis, columnNames []string) {
	for columnIndex, name := range columnNames {
		column := querier_dto.RawOutputColumn{Name: name}
		if columnIndex < len(analysis.OutputColumns) {
			column.Expression = analysis.OutputColumns[columnIndex].Expression
			column.ColumnName = analysis.OutputColumns[columnIndex].ColumnName
			column.TableAlias = analysis.OutputColumns[columnIndex].TableAlias
			analysis.OutputColumns[columnIndex] = column
			continue
		}
		analysis.OutputColumns = append(analysis.OutputColumns, column)
	}
	if len(columnNames) < len(analysis.OutputColumns) {
		analysis.OutputColumns = analysis.OutputColumns[:len(columnNames)]
	}
}

// columnsFromNames builds an unknown-typed Column slice from a name list.
//
// Used as the catalogue fallback when analyseViewBody fails or when individual SELECT
// items are not resolvable.
//
// Takes names ([]string) which are the column names to materialise.
//
// Returns []querier_dto.Column which holds nullable unknown-typed columns.
func columnsFromNames(names []string) []querier_dto.Column {
	if len(names) == 0 {
		return nil
	}
	columns := make([]querier_dto.Column, len(names))
	for index, name := range names {
		columns[index] = querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		}
	}
	return columns
}

// inferViewColumnNames extracts column names from the SELECT projection that follows
// `CREATE VIEW name AS`.
//
// It walks comma-separated expressions at parenthesis depth zero between the leading
// SELECT and the FROM keyword, returning the alias from `... AS alias` or the trailing
// identifier of a column reference. Expressions without a recoverable name produce an
// empty entry.
//
// Takes tokens ([]token) which is the full token stream of the containing statement.
// Takes start (int) which is the index of the first token after `AS`.
//
// Returns []string which is the inferred column name list in order.
func inferViewColumnNames(tokens []token, start int) []string {
	selectStart := findSelectStart(tokens, start)
	if selectStart < 0 {
		return nil
	}
	fromIndex := findTopLevelKeyword(tokens, selectStart, "FROM")
	if fromIndex < 0 {
		fromIndex = len(tokens)
	}
	return collectProjectionNames(tokens[selectStart:fromIndex])
}

// findSelectStart returns the index just past the outer-most SELECT keyword in the view
// body, skipping wrapping parentheses and any leading WITH ... AS (...)
// common-table-expression list.
//
// Recognised shapes are a bare `SELECT ...`, a parenthesised `(SELECT ...)`, a `WITH cte
// AS (...) SELECT ...`, a `WITH RECURSIVE cte AS (...) SELECT ...`, and any combination
// of these with redundant wrapping parens.
//
// Takes tokens ([]token) which is the token stream.
// Takes start (int) which is the starting search index (typically the token just past
// `AS`).
//
// Returns int which is the index of the first projection token, or -1 when no outer
// SELECT is found.
func findSelectStart(tokens []token, start int) int {
	for start < len(tokens) {
		switch tokens[start].kind {
		case tokenLeftParen:
			start++
			continue
		case tokenIdentifier:
			value := strings.ToUpper(tokens[start].value)
			if value == "SELECT" {
				return start + 1
			}
			if value == "WITH" {
				start = skipWithClause(tokens, start+1)
				continue
			}
			return -1
		default:
			return -1
		}
	}
	return -1
}

// skipWithClause walks past a WITH cte-list of the form `[RECURSIVE] name AS (...) [,
// name AS (...)] ...`.
//
// The first token after the last CTE body is returned. The function is conservative: any
// deviation from the expected structure leaves the cursor where it found the unexpected
// token, letting the caller fall through.
//
// Takes tokens ([]token) which is the token stream.
// Takes start (int) which is the index of the first token after WITH.
//
// Returns int which is the index of the token after the CTE list.
func skipWithClause(tokens []token, start int) int {
	if start < len(tokens) && tokens[start].kind == tokenIdentifier &&
		strings.EqualFold(tokens[start].value, "RECURSIVE") {
		start++
	}
	for {
		next, hasMore := skipSingleWithClauseEntry(tokens, start)
		if !hasMore {
			return next
		}
		start = next
	}
}

// skipSingleWithClauseEntry advances past one CTE entry within a WITH clause. Returns the
// next start index and a flag indicating whether the caller should continue iterating to
// consume more entries.
//
// Takes tokens ([]token) which is the token stream.
// Takes start (int) which is the index of the CTE name to consume.
//
// Returns int which is the index after this entry (or the position where parsing should
// stop on a malformed CTE list).
// Returns bool which is true when a trailing comma was consumed and another entry should
// follow.
func skipSingleWithClauseEntry(tokens []token, start int) (int, bool) {
	if start >= len(tokens) || tokens[start].kind != tokenIdentifier {
		return start, false
	}
	start++
	if start < len(tokens) && tokens[start].kind == tokenLeftParen {
		start = skipMatchingParens(tokens, start)
	}
	if start >= len(tokens) || tokens[start].kind != tokenIdentifier ||
		!strings.EqualFold(tokens[start].value, "AS") {
		return start, false
	}
	start++
	if start >= len(tokens) || tokens[start].kind != tokenLeftParen {
		return start, false
	}
	start = skipMatchingParens(tokens, start)
	if start < len(tokens) && tokens[start].kind == tokenComma {
		return start + 1, true
	}
	return start, false
}

// skipMatchingParens consumes a balanced parenthesised group beginning at start (which
// must reference a tokenLeftParen). Returns the index of the first token after the
// closing right paren.
//
// Takes tokens ([]token) which is the token stream.
// Takes start (int) which is the index of the opening `(`.
//
// Returns int which is the index just past the matching `)`, or len(tokens) when the
// input is unbalanced.
func skipMatchingParens(tokens []token, start int) int {
	depth := 0
	for index := start; index < len(tokens); index++ {
		switch tokens[index].kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
			if depth == 0 {
				return index + 1
			}
		default:
		}
	}
	return len(tokens)
}

// findTopLevelKeyword scans tokens for the first occurrence of the given keyword at
// parenthesis depth zero. The depth tracking lets the search skip past subqueries that
// may contain FROM, WHERE, etc.
//
// Takes tokens ([]token) which is the token stream.
// Takes start (int) which is the starting index.
// Takes keyword (string) which is the keyword to find (compared case-insensitively).
//
// Returns int which is the absolute index of the keyword, or -1 when not found at depth
// zero.
func findTopLevelKeyword(tokens []token, start int, keyword string) int {
	depth := 0
	for index := start; index < len(tokens); index++ {
		switch tokens[index].kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenIdentifier:
			if depth == 0 && strings.EqualFold(tokens[index].value, keyword) {
				return index
			}
		default:
		}
	}
	return -1
}

// collectProjectionNames splits the SELECT projection tokens on top- level commas and
// resolves each item to a column name via projectionItemName.
//
// Takes projection ([]token) which is the slice of tokens between the SELECT keyword
// (exclusive) and the FROM keyword (exclusive).
//
// Returns []string which is the per-item column name list.
func collectProjectionNames(projection []token) []string {
	var names []string
	depth := 0
	itemStart := 0
	for index := range projection {
		switch projection[index].kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenComma:
			if depth == 0 {
				names = append(names, projectionItemName(projection[itemStart:index]))
				itemStart = index + 1
			}
		default:
		}
	}
	if itemStart < len(projection) {
		names = append(names, projectionItemName(projection[itemStart:]))
	}
	return names
}

// projectionItemName resolves a single SELECT projection item to its column name.
//
// The rules mirror SQL's own naming behaviour. An `expr AS alias` returns `alias`, an
// `expr alias` implicit alias returns `alias`, a `table.column` returns `column`, a bare
// identifier returns the identifier, and anything else returns the empty string.
//
// Takes item ([]token) which is the projection item's token slice.
//
// Returns string which is the inferred column name, or empty when no sensible name can be
// derived.
func projectionItemName(item []token) string {
	if len(item) == 0 {
		return ""
	}

	for index := len(item) - 2; index >= 0; index-- {
		if item[index].kind != tokenIdentifier {
			continue
		}
		if !strings.EqualFold(item[index].value, "AS") {
			continue
		}
		if item[index+1].kind == tokenIdentifier {
			return item[index+1].value
		}
	}

	if len(item) >= 2 {
		last := item[len(item)-1]
		secondLast := item[len(item)-2]
		if last.kind == tokenIdentifier && secondLast.kind == tokenIdentifier &&
			!isReservedProjectionKeyword(last.value) && !isReservedProjectionKeyword(secondLast.value) {
			return last.value
		}
	}

	last := item[len(item)-1]
	if last.kind == tokenIdentifier {
		return last.value
	}
	return ""
}

// isReservedProjectionKeyword reports whether the identifier looks like a SQL keyword
// that should not be treated as a column alias.
//
// Takes value (string) which is the candidate identifier.
//
// Returns bool which is true for reserved keywords.
func isReservedProjectionKeyword(value string) bool {
	switch strings.ToUpper(value) {
	case "DISTINCT", "ALL", "AS", "DESC", "ASC", "NULL", "TRUE", "FALSE":
		return true
	}
	return false
}

// parseDropView parses a DROP VIEW statement.
//
// Returns *querier_dto.CatalogueMutation which describes the dropped view.
// Returns error when the view name cannot be parsed.
func (p *parser) parseDropView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("VIEW")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	viewName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationDropView,
		TableName: viewName,
	}, nil
}

// parseCreateIndex parses a CREATE [UNIQUE] INDEX statement.
//
// Returns *querier_dto.CatalogueMutation which records the indexed table.
// Returns error when an identifier cannot be parsed.
func (p *parser) parseCreateIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword(keywordUNIQUE)
	p.mustKeyword("INDEX")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	_, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	p.mustKeyword(keywordON)

	tableName, tableError := p.parseTableName()
	if tableError != nil {
		return nil, tableError
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationCreateIndex,
		TableName: tableName,
	}, nil
}

// parseDropIndex parses a DROP INDEX statement.
//
// Returns *querier_dto.CatalogueMutation which records the drop kind.
// Returns error when the index name cannot be parsed.
func (p *parser) parseDropIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("INDEX")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	_, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationDropIndex,
	}, nil
}

// parseCreateVirtualTable parses a CREATE VIRTUAL TABLE statement and extracts
// module-specific column shape information.
//
// Takes engine (*SQLiteEngine) which supplies type normalisation for inferred columns.
//
// Returns *querier_dto.CatalogueMutation which describes the virtual table, or nil when
// no USING clause is present.
// Returns error when the table name cannot be parsed.
func (p *parser) parseCreateVirtualTable(engine *SQLiteEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword("VIRTUAL")
	p.mustKeyword(keywordTABLE)

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	tableName, nameError := p.parseTableName()
	if nameError != nil {
		return nil, nameError
	}

	if !p.matchKeyword("USING") {
		return nil, nil
	}

	moduleName, moduleError := p.parseIdentifierOrKeyword()
	if moduleError != nil {
		return nil, nil
	}

	if p.current().kind != tokenLeftParen {
		return &querier_dto.CatalogueMutation{
			Kind:              querier_dto.MutationCreateTable,
			TableName:         tableName,
			IsVirtual:         true,
			VirtualModuleName: moduleName,
		}, nil
	}

	argumentTokens, _ := p.collectParenthesised()

	lowerModule := strings.ToLower(moduleName)

	var columns []querier_dto.Column
	var primaryKeyColumns []string

	switch lowerModule {
	case "fts5":
		columns = extractFTS5Columns(argumentTokens, engine)
	case "rtree", "rtree_i32":
		columns, primaryKeyColumns = extractRTreeColumns(argumentTokens, engine)
	default:
		columns = extractGenericVirtualColumns(argumentTokens, engine)
	}

	return &querier_dto.CatalogueMutation{
		Kind:              querier_dto.MutationCreateTable,
		TableName:         tableName,
		Columns:           columns,
		PrimaryKey:        primaryKeyColumns,
		IsVirtual:         true,
		VirtualModuleName: lowerModule,
	}, nil
}

// extractFTS5Columns derives the searchable columns of an FTS5 virtual table from its
// argument list and appends the synthetic `rank` column.
//
// Takes tokens ([]token) which are the argument tokens of the USING clause.
// Takes engine (*SQLiteEngine) which supplies type normalisation.
//
// Returns []querier_dto.Column which is the FTS5 column schema including `rank`.
func extractFTS5Columns(tokens []token, engine *SQLiteEngine) []querier_dto.Column {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column

	for _, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		if isFTS5Option(segment) {
			continue
		}

		columns = append(columns, querier_dto.Column{
			Name:     segment[0].value,
			SQLType:  engine.NormaliseTypeName("text"),
			Nullable: true,
		})
	}

	columns = append(columns, querier_dto.Column{
		Name:          "rank",
		SQLType:       engine.NormaliseTypeName("real"),
		Nullable:      true,
		IsGenerated:   true,
		GeneratedKind: querier_dto.GeneratedKindVirtual,
	})

	return columns
}

// isFTS5Option reports whether a comma-delimited segment of an FTS5 argument list is a
// configuration option rather than a column.
//
// Takes segment ([]token) which is one comma-separated argument segment.
//
// Returns bool which is true for `name = value` pairs and well-known FTS5 option names.
func isFTS5Option(segment []token) bool {
	if len(segment) < 2 {
		return false
	}

	if segment[1].kind == tokenOperator && segment[1].value == "=" {
		return true
	}

	optionName := strings.ToLower(segment[0].value)
	switch optionName {
	case "content", "content_rowid", "tokenize", "prefix", "detail", "columnsize":
		return true
	}

	return false
}

// extractRTreeColumns derives the column schema of an R-Tree virtual table, treating the
// first column as the integer primary key.
//
// Takes tokens ([]token) which are the argument tokens of the USING clause.
// Takes engine (*SQLiteEngine) which supplies type normalisation.
//
// Returns []querier_dto.Column which is the R-Tree column schema.
// Returns []string which lists the primary key column names.
func extractRTreeColumns(tokens []token, engine *SQLiteEngine) ([]querier_dto.Column, []string) {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column
	var primaryKeyColumns []string

	for columnIndex, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		name := segment[0].value

		if columnIndex == 0 {
			columns = append(columns, querier_dto.Column{
				Name:       name,
				SQLType:    engine.NormaliseTypeName("integer"),
				Nullable:   false,
				HasDefault: true,
			})
			primaryKeyColumns = append(primaryKeyColumns, name)
		} else {
			columns = append(columns, querier_dto.Column{
				Name:     name,
				SQLType:  engine.NormaliseTypeName("real"),
				Nullable: true,
			})
		}
	}

	return columns, primaryKeyColumns
}

// extractGenericVirtualColumns derives a best-effort text-typed column list for an
// unrecognised virtual table module.
//
// Takes tokens ([]token) which are the argument tokens of the USING clause.
// Takes engine (*SQLiteEngine) which supplies type normalisation.
//
// Returns []querier_dto.Column which is the inferred column schema.
func extractGenericVirtualColumns(tokens []token, engine *SQLiteEngine) []querier_dto.Column {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column

	for _, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		if len(segment) >= 2 && segment[1].kind == tokenOperator && segment[1].value == "=" {
			continue
		}

		if segment[0].kind != tokenIdentifier {
			continue
		}

		columns = append(columns, querier_dto.Column{
			Name:     segment[0].value,
			SQLType:  engine.NormaliseTypeName("text"),
			Nullable: true,
		})
	}

	return columns
}

// splitTokensOnComma splits tokens on top-level commas, leaving commas inside parentheses
// alone.
//
// Takes tokens ([]token) which are the tokens to split.
//
// Returns [][]token which is one slice per top-level comma-separated segment.
func splitTokensOnComma(tokens []token) [][]token {
	var segments [][]token
	var current []token
	depth := 0

	for _, currentToken := range tokens {
		if currentToken.kind == tokenLeftParen {
			depth++
		}
		if currentToken.kind == tokenRightParen {
			depth--
		}
		if currentToken.kind == tokenComma && depth == 0 {
			segments = append(segments, current)
			current = nil
			continue
		}
		current = append(current, currentToken)
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}

	return segments
}

// parseTableName parses an optionally schema-qualified table name and returns the table
// portion.
//
// Returns string which is the bare table name.
// Returns error when no identifier is found.
func (p *parser) parseTableName() (string, error) {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return "", err
	}

	if p.current().kind == tokenDot {
		p.advance()
		tableName, tableErr := p.parseIdentifierOrKeyword()
		if tableErr != nil {
			return "", tableErr
		}
		return tableName, nil
	}

	return name, nil
}

// parseCreateTrigger parses a CREATE TRIGGER statement up to its target table and ignores
// the action body.
//
// Returns *querier_dto.CatalogueMutation which records the trigger name and target table.
// Returns error when the trigger name cannot be parsed.
func (p *parser) parseCreateTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.mustKeyword("TRIGGER")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	triggerName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword("BEFORE")
	p.matchKeyword("AFTER")
	if p.matchKeyword("INSTEAD") {
		p.matchKeyword("OF")
	}

	p.matchKeyword("DELETE")
	p.matchKeyword("INSERT")
	p.matchKeyword("UPDATE")
	if p.matchKeyword("OF") {
		for !p.atEnd() {
			p.advance()
			if p.current().kind != tokenComma {
				break
			}
			p.advance()
		}
	}

	p.mustKeyword(keywordON)
	tableName, _ := p.parseTableName()

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTrigger,
		TriggerName: triggerName,
		TableName:   tableName,
	}, nil
}

// parseDropTrigger parses a DROP TRIGGER statement.
//
// Returns *querier_dto.CatalogueMutation which records the dropped trigger.
// Returns error when the trigger name cannot be parsed.
func (p *parser) parseDropTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("TRIGGER")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	triggerName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationDropTrigger,
		TriggerName: triggerName,
	}, nil
}
