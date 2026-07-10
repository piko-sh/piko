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
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safeconv"
)

// parseFromClause parses a FROM clause into a table list and a join list.
//
// Returns []querier_dto.TableReference which are the top-level tables.
// Returns []querier_dto.JoinClause which are the explicit joins.
// Returns error when a join target or condition cannot be parsed.
func (p *parser) parseFromClause() ([]querier_dto.TableReference, []querier_dto.JoinClause, error) {
	var tables []querier_dto.TableReference
	var joins []querier_dto.JoinClause

	table, err := p.parseFromTableSource(querier_dto.JoinInner)
	if err != nil {
		return nil, nil, err
	}
	if table != nil {
		tables = append(tables, *table)
	}

	for {
		joinKind := p.parseJoinKeyword()
		if joinKind < 0 {
			commaTable, commaErr := p.parseCommaJoin()
			if commaErr != nil {
				return nil, nil, commaErr
			}
			if commaTable == nil {
				break
			}
			tables = append(tables, *commaTable)
			continue
		}

		join, joinErr := p.parseJoinTarget(joinKind)
		if joinErr != nil {
			return nil, nil, joinErr
		}
		if join != nil {
			joins = append(joins, *join)
		}

		p.parseJoinCondition()
	}

	return tables, joins, nil
}

// parseCommaJoin parses an additional comma-joined table after the initial FROM source.
//
// Returns *querier_dto.TableReference which is the new table, or nil when no comma is
// present.
// Returns error when the table source cannot be parsed.
func (p *parser) parseCommaJoin() (*querier_dto.TableReference, error) {
	if p.current().kind != tokenComma {
		return nil, nil
	}
	p.advance()
	return p.parseFromTableSource(querier_dto.JoinInner)
}

// parseFromTableSource parses a single FROM source which may be a table, derived table,
// or table-valued function.
//
// Takes joinKind (querier_dto.JoinKind) which describes how the source participates in
// the surrounding FROM clause.
//
// Returns *querier_dto.TableReference which is the table reference for bare tables, or
// nil for derived tables and table-valued functions.
// Returns error when a derived table fails to parse.
func (p *parser) parseFromTableSource(joinKind querier_dto.JoinKind) (*querier_dto.TableReference, error) {
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(joinKind)
		return nil, nil
	}
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(joinKind); err != nil {
			return nil, err
		}
		return nil, nil
	}
	tableName, alias := p.parseTableReference()
	return &querier_dto.TableReference{Name: tableName, Alias: alias}, nil
}

// parseJoinTarget parses the table or derived target of an explicit JOIN.
//
// Takes joinKind (int) which is the int-encoded join kind from parseJoinKeyword.
//
// Returns *querier_dto.JoinClause which is the parsed join, or nil for derived tables and
// table-valued functions.
// Returns error when a derived table fails to parse.
func (p *parser) parseJoinTarget(joinKind int) (*querier_dto.JoinClause, error) {
	joinKindValue := safeconv.IntToUint8(joinKind)
	if p.isTableValuedFunctionStart() {
		p.parseTableValuedFunction(querier_dto.JoinKind(joinKindValue))
		return nil, nil
	}
	if p.isSubqueryStart() {
		if err := p.parseDerivedTable(querier_dto.JoinKind(joinKindValue)); err != nil {
			return nil, err
		}
		return nil, nil
	}
	joinTable, joinAlias := p.parseTableReference()
	return &querier_dto.JoinClause{
		Kind: querier_dto.JoinKind(joinKindValue),
		Table: querier_dto.TableReference{
			Name:  joinTable,
			Alias: joinAlias,
		},
	}, nil
}

// parseJoinCondition consumes an ON expression or USING column list when present after a
// JOIN target.
func (p *parser) parseJoinCondition() {
	if p.matchKeyword(keywordON) {
		p.parseJoinConditionExpression()
	} else if p.matchKeyword("USING") {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
	}
}

// isSubqueryStart reports whether the cursor is positioned at the opening of a `(SELECT
// ...)` derived table.
//
// Returns bool which is true when the next two tokens are `(` followed by `SELECT`.
func (p *parser) isSubqueryStart() bool {
	return p.current().kind == tokenLeftParen && p.peek().kind == tokenIdentifier &&
		strings.EqualFold(p.peek().value, keywordSELECT)
}

// parseDerivedTable parses a `(SELECT ...) [AS] alias` derived table.
//
// Takes joinKind (querier_dto.JoinKind) which describes how the derived table
// participates in the FROM clause.
//
// Returns error when the inner SELECT cannot be analysed.
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

	p.rawDerivedTables = append(p.rawDerivedTables, querier_dto.RawDerivedTableReference{
		Alias:      alias,
		InnerQuery: innerAnalysis,
		JoinKind:   joinKind,
	})

	return nil
}

// parseTableReference parses a `[schema.]name [[AS] alias]` table reference.
//
// Returns tableName (string) which is the bare table name.
// Returns tableAlias (string) which is the alias when present, else "".
func (p *parser) parseTableReference() (tableName string, tableAlias string) {
	if p.current().kind != tokenIdentifier {
		return "", ""
	}

	name := p.advance().value

	if p.current().kind == tokenDot {
		p.advance()
		if p.current().kind == tokenIdentifier {
			name = p.advance().value
		}
	}

	alias := ""
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() &&
		!p.isAnyKeyword(keywordSET, keywordVALUES, "DEFAULT", keywordWHERE, "INNER", "LEFT", "RIGHT",
			"FULL", "CROSS", "NATURAL", keywordJOIN, keywordON, "USING") {
		alias = p.advance().value
	}

	return name, alias
}

// isTableValuedFunctionStart reports whether the cursor is at an `identifier(` shape that
// is not the SELECT keyword.
//
// Returns bool which is true for table-valued function call syntax.
func (p *parser) isTableValuedFunctionStart() bool {
	return p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen &&
		!strings.EqualFold(p.current().value, keywordSELECT)
}

// parseTableValuedFunction parses a `name(args...) [[AS] alias]` table-valued function
// call and records it on the parser.
//
// Takes joinKind (querier_dto.JoinKind) which describes how the function participates in
// the FROM clause.
func (p *parser) parseTableValuedFunction(joinKind querier_dto.JoinKind) {
	functionName := strings.ToLower(p.advance().value)
	p.advance()

	argumentOrdinal := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		refsCountBefore := len(p.parameterRefs)
		p.parseExpression()
		p.markFunctionArgumentParameters(refsCountBefore, functionName, argumentOrdinal)
		argumentOrdinal++
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	alias := functionName
	if p.matchKeyword(keywordAS) {
		if p.current().kind == tokenIdentifier {
			alias = p.advance().value
		}
	} else if p.current().kind == tokenIdentifier && !p.isJoinKeyword() && !p.isSelectTerminator() {
		alias = p.advance().value
	}

	p.rawTableValuedFunctions = append(p.rawTableValuedFunctions, querier_dto.RawTableValuedFunctionReference{
		FunctionName: functionName,
		Alias:        alias,
		JoinKind:     joinKind,
	})
}

// parseJoinKeyword consumes a JOIN keyword sequence and reports the resulting join kind.
//
// Returns int which is the int-encoded join kind, or -1 when no join keyword is present.
func (p *parser) parseJoinKeyword() int {
	p.matchKeyword("NATURAL")

	if p.matchKeyword("INNER") {
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinInner)
	}
	if p.matchKeyword("LEFT") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinLeft)
	}
	if p.matchKeyword("RIGHT") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinRight)
	}
	if p.matchKeyword("FULL") {
		p.matchKeyword("OUTER")
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinFull)
	}
	if p.matchKeyword("CROSS") {
		p.matchKeyword(keywordJOIN)
		return int(querier_dto.JoinCross)
	}
	if p.matchKeyword(keywordJOIN) {
		return int(querier_dto.JoinInner)
	}

	return -1
}

// isJoinKeyword reports whether the current token is a JOIN-related keyword.
//
// Returns bool which is true for JOIN, INNER, LEFT, RIGHT, FULL, CROSS, or NATURAL.
func (p *parser) isJoinKeyword() bool {
	return p.isAnyKeyword(keywordJOIN, "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "NATURAL")
}

// matchCompoundOperator consumes a compound query operator when the cursor sits on UNION,
// INTERSECT, or EXCEPT.
//
// Returns querier_dto.CompoundOperator which is the consumed operator, or zero when none
// is matched.
func (p *parser) matchCompoundOperator() querier_dto.CompoundOperator {
	if p.matchKeyword(keywordUNION) {
		if p.matchKeyword("ALL") {
			return querier_dto.CompoundUnionAll
		}
		return querier_dto.CompoundUnion
	}
	if p.matchKeyword(keywordINTERSECT) {
		return querier_dto.CompoundIntersect
	}
	if p.matchKeyword(keywordEXCEPT) {
		return querier_dto.CompoundExcept
	}
	return 0
}

// resolveComparisonContext detects when a parameter sits on the right-hand side of a
// comparison operator.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is ParameterContextComparison when a
// comparison context is found, else ParameterContextUnknown.
// Returns *querier_dto.ColumnReference which is the LHS column when identifiable, else
// nil.
func (p *parser) resolveComparisonContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	if paramPosition < 2 {
		return querier_dto.ParameterContextUnknown, nil
	}

	if context, columnRef := p.resolveIsComparisonContext(paramPosition); context != querier_dto.ParameterContextUnknown {
		return context, columnRef
	}

	prevToken := p.tokens[paramPosition-1]
	if prevToken.kind != tokenOperator || !isComparisonOperator(prevToken.value) {
		return querier_dto.ParameterContextUnknown, nil
	}
	columnRef := p.extractColumnReference(paramPosition - 2)
	if columnRef == nil {
		return querier_dto.ParameterContextUnknown, nil
	}
	return querier_dto.ParameterContextComparison, columnRef
}

// resolveIsComparisonContext detects SQLite null-safe equality where a placeholder
// follows IS or IS NOT.
//
// An example is f.version_parent_id IS ?1. SQLite treats "x IS y" as a null-safe
// comparison for arbitrary operands, so the placeholder is typed from the LHS column
// exactly as "col = ?" is. IS is a plain identifier rather than an operator token, so
// isComparisonOperator is deliberately not widened; the IS handling lives here. IS NULL
// and IS NOT NULL never reach this path because no placeholder follows the NULL literal,
// so they register no parameter.
//
// Takes paramPosition (int) which is the placeholder's token index.
//
// Returns querier_dto.ParameterContext which is ParameterContextComparison when an IS
// comparison is found, else ParameterContextUnknown.
// Returns *querier_dto.ColumnReference which is the LHS column when identifiable, else
// nil.
func (p *parser) resolveIsComparisonContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	isPosition := paramPosition - 1
	if isPosition >= 0 && p.tokens[isPosition].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[isPosition].value, keywordNOT) {
		isPosition--
	}

	if isPosition < 1 || p.tokens[isPosition].kind != tokenIdentifier ||
		!strings.EqualFold(p.tokens[isPosition].value, "IS") {
		return querier_dto.ParameterContextUnknown, nil
	}

	columnRef := p.extractColumnReference(isPosition - 1)
	if columnRef == nil {
		return querier_dto.ParameterContextUnknown, nil
	}
	return querier_dto.ParameterContextComparison, columnRef
}

// resolveLikeContext detects an enclosing pattern operator around a parameter.
//
// Walks back from a parameter position through balanced parens (stopping at boolean or
// clause boundaries) to find an enclosing LIKE, GLOB, REGEXP, or MATCH operator and
// returns its LHS column reference, falling through to a wider scan of the LHS expression
// when the immediate-left token is not itself a column. NOT LIKE is captured because
// resolveLikeOperatorColumn skips a leading NOT token before resolving the column.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which is ParameterContextLike when the parameter
// sits inside a pattern operator's right-hand side, else ParameterContextUnknown.
// Returns *querier_dto.ColumnReference which holds the LHS column when one can be
// identified, else nil.
func (p *parser) resolveLikeContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference) {
	likePosition, found := p.findEnclosingLikeOperator(paramPosition)
	if !found {
		return querier_dto.ParameterContextUnknown, nil
	}
	return querier_dto.ParameterContextLike, p.resolveLikeOperatorColumn(likePosition)
}

// findEnclosingLikeOperator walks back from paramPosition through balanced parens looking
// for a pattern operator at depth 0, returning its token index when found. Boolean and
// clause keywords end the walk.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns int which is the LIKE-style operator's token index when found.
// Returns bool which is true when an operator was located.
func (p *parser) findEnclosingLikeOperator(paramPosition int) (int, bool) {
	return engine_shared.FindEnclosingLikeOperator(paramPosition,
		func(index int) bool { return p.tokens[index].kind == tokenLeftParen },
		func(index int) bool { return p.tokens[index].kind == tokenRightParen },
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(p.tokens[index].value))
		},
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikePatternKeyword(strings.ToUpper(p.tokens[index].value))
		},
	)
}

// resolveLikeOperatorColumn picks the column reference associated with a LIKE operator's
// left-hand side, first inspecting the immediately preceding token (skipping a single
// NOT) and falling back to a scan of the wider LHS expression for shapes like `(a || b)
// LIKE ?`.
//
// Takes likePosition (int) which is the LIKE-style operator's token index.
//
// Returns *querier_dto.ColumnReference which is the inferred LHS column or nil when none
// can be identified.
func (p *parser) resolveLikeOperatorColumn(likePosition int) *querier_dto.ColumnReference {
	columnPosition := likePosition - 1
	if columnPosition >= 0 && p.tokens[columnPosition].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[columnPosition].value, "NOT") {
		columnPosition--
	}
	if columnRef := p.extractColumnReference(columnPosition); columnRef != nil && !isKeywordColumnReference(columnRef) {
		return columnRef
	}
	return p.findColumnInExpressionRange(p.findLikeExpressionStart(likePosition), likePosition-1)
}

// isKeywordColumnReference reports whether a column reference's bare column name is
// actually a SQL keyword (LIKE, AND, etc.) rather than a column. Edge case for malformed
// input like "LIKE ?" at position 0 where the backward walk would otherwise treat the
// keyword itself as the LHS column.
//
// Takes columnRef (*querier_dto.ColumnReference) which is the candidate column reference.
//
// Returns bool which is true when the bare column name is a keyword.
func isKeywordColumnReference(columnRef *querier_dto.ColumnReference) bool {
	if columnRef == nil || columnRef.TableAlias != "" {
		return false
	}
	upper := strings.ToUpper(columnRef.ColumnName)
	return isLikePatternKeyword(upper) || isLikeBoundaryKeyword(upper) || isReservedNonColumnKeyword(upper)
}

// findLikeExpressionStart locates the first token of the LHS expression preceding a
// LIKE-style operator by walking back to the previous predicate boundary or the start of
// the token stream. Used to bound the scan when the LHS is a complex expression rather
// than a bare column.
//
// Takes likePosition (int) which is the LIKE keyword's token index.
//
// Returns int which is the token index where the LHS starts (0 when no boundary precedes
// it).
func (p *parser) findLikeExpressionStart(likePosition int) int {
	parenDepth := 0
	for i := likePosition - 1; i >= 0; i-- {
		tok := p.tokens[i]
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenRightParen:
			parenDepth++
			continue
		case tokenLeftParen:
			if parenDepth > 0 {
				parenDepth--
				continue
			}
			return i + 1
		}
		if parenDepth > 0 {
			continue
		}
		if tok.kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(tok.value)) {
			return i + 1
		}
	}
	return 0
}

// findColumnInExpressionRange scans [start, end] forward for the first identifier that
// looks like a column, skipping function names, SQL types, aliases, and reserved keywords
// so complex LHS expressions like `(name || ' ' || role)` or `CAST(COALESCE(...) AS
// TEXT)` still resolve to a meaningful column.
//
// Takes start (int) which is the inclusive start of the range.
// Takes end (int) which is the inclusive end of the range.
//
// Returns *querier_dto.ColumnReference which is the first column found, or nil when none
// is plausible.
func (p *parser) findColumnInExpressionRange(start, end int) *querier_dto.ColumnReference {
	if end < start || end >= len(p.tokens) {
		return nil
	}
	for i := start; i <= end; i++ {
		tok := p.tokens[i]
		if tok.kind != tokenIdentifier {
			continue
		}
		if isLikelyColumnIdentifier(p.tokens, i) {
			return p.extractColumnReference(i)
		}
		if isQualifiedColumnTail(p.tokens, i) {
			return p.extractColumnReference(i)
		}
	}
	return nil
}

// isLikelyColumnIdentifier checks whether the identifier at position looks like a column
// rather than a function, type, alias, or reserved keyword by inspecting neighbouring
// tokens.
//
// Takes tokens ([]token) which is the parser's token slice.
// Takes position (int) which is the candidate identifier's index.
//
// Returns bool which is true when the identifier looks like a column.
func isLikelyColumnIdentifier(tokens []token, position int) bool {
	if position < 0 || position >= len(tokens) {
		return false
	}
	tok := tokens[position]
	if tok.kind != tokenIdentifier {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenLeftParen {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenDot {
		return false
	}
	if position > 0 && tokens[position-1].kind == tokenIdentifier &&
		strings.EqualFold(tokens[position-1].value, "AS") {
		return false
	}
	upper := strings.ToUpper(tok.value)
	if isLikeBoundaryKeyword(upper) || isLikePatternKeyword(upper) {
		return false
	}
	if isReservedNonColumnKeyword(upper) {
		return false
	}
	return true
}

// isQualifiedColumnTail reports whether the identifier at position is the column half of
// a qualified `<alias>.<column>` reference, so the leftmost-column scan does not skip it.
//
// Takes tokens ([]token) which is the parser's token slice.
// Takes position (int) which is the candidate identifier's index.
//
// Returns bool which is true for the column half of `<alias>.<column>`.
func isQualifiedColumnTail(tokens []token, position int) bool {
	if position < 2 {
		return false
	}
	if tokens[position-1].kind != tokenDot {
		return false
	}
	if tokens[position-2].kind != tokenIdentifier {
		return false
	}
	if position+1 < len(tokens) && tokens[position+1].kind == tokenLeftParen {
		return false
	}
	return true
}

// isReservedNonColumnKeyword lists identifier values that must not be treated as column
// references when scanning a LIKE LHS: casts, type names, literals, and modifier keywords
// that lex as identifiers.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true when the value is never a column.
func isReservedNonColumnKeyword(keyword string) bool {
	switch keyword {
	case "AS", "CAST", "COLLATE", "DISTINCT", "ALL", "NULL", "TRUE", "FALSE",
		"INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT", "REAL", "FLOAT",
		"DOUBLE", "NUMERIC", "DECIMAL", "TEXT", "VARCHAR", "CHAR", "BLOB",
		"BOOLEAN", "BOOL", "DATE", "TIME", "TIMESTAMP", "DATETIME",
		"ESCAPE", "BETWEEN", "IS", "IN", "EXISTS", "NOT", "ASC", "DESC":
		return true
	}
	return false
}

// isLikePatternKeyword reports whether a keyword introduces a SQLite string pattern
// match.
//
// Recognised operators are LIKE, GLOB, REGEXP, and MATCH. SQLite has no ILIKE or RLIKE;
// case-insensitive matching uses LIKE semantics configured at compile time. MATCH is the
// FTS5 virtual-table operator.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true for LIKE, GLOB, REGEXP, or MATCH.
func isLikePatternKeyword(keyword string) bool {
	switch keyword {
	case "LIKE", "GLOB", "REGEXP", "MATCH":
		return true
	}
	return false
}

// isLikeBoundaryKeyword reports whether a keyword ends the predicate that could contain a
// LIKE pattern, so the backward walk should stop.
//
// Takes keyword (string) which is the upper-case identifier value.
//
// Returns bool which is true at any clause or boolean boundary.
func isLikeBoundaryKeyword(keyword string) bool {
	return engine_shared.IsClauseBoundaryKeyword(keyword)
}

// detectParameterContext infers the surrounding clause context of a parameter token by
// inspecting its enclosing parenthesis.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns querier_dto.ParameterContext which describes the inferred context, or
// ParameterContextUnknown when nothing matches.
// Returns *querier_dto.ColumnReference which is the inferred LHS column, or nil when none
// applies.
// Returns *querier_dto.SQLType which is the CAST target type when the parameter sits
// inside a CAST expression, else nil.
func (p *parser) detectParameterContext(paramPosition int) (querier_dto.ParameterContext, *querier_dto.ColumnReference, *querier_dto.SQLType) {
	enclosingParen := p.findEnclosingParen(paramPosition)
	if enclosingParen < 0 {
		return querier_dto.ParameterContextUnknown, nil, nil
	}

	if enclosingParen >= 2 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, "IN") {
		columnRef := p.extractColumnReferenceBeforeIN(enclosingParen - 1)
		return querier_dto.ParameterContextInList, columnRef, nil
	}

	if enclosingParen >= 1 &&
		p.tokens[enclosingParen-1].kind == tokenIdentifier &&
		strings.EqualFold(p.tokens[enclosingParen-1].value, "CAST") {
		castType := p.extractCastType(paramPosition)
		if castType != nil {
			return querier_dto.ParameterContextCast, nil, castType
		}
	}

	if enclosingParen >= 1 && p.tokens[enclosingParen-1].kind == tokenIdentifier {
		functionName := strings.ToUpper(p.tokens[enclosingParen-1].value)
		if functionName != "IN" && functionName != "CAST" &&
			functionName != keywordSELECT && functionName != keywordWHERE {
			return querier_dto.ParameterContextFunctionArgument, nil, nil
		}
	}

	return querier_dto.ParameterContextUnknown, nil, nil
}

// enclosingFunctionArgument resolves the enclosing function-call metadata for a
// placeholder the flat scan tagged as a function argument.
//
// The metadata is the lower-cased function name (the identifier immediately before the
// enclosing parenthesis) and the placeholder's 0-based argument ordinal. The ordinal
// counts top-level commas between the opening parenthesis and the placeholder, so each
// comma-separated argument slot is counted once regardless of how many tokens it spans.
// The lower-cased name matches how parseFunctionCall and parseTableValuedFunction record
// the function name elsewhere, so the analyser resolves the same overload.
//
// Takes paramPosition (int) which is the placeholder's token index.
//
// Returns name (string) which is the lower-cased enclosing function name.
// Returns ordinal (int) which is the placeholder's 0-based argument slot.
// Returns ok (bool) which is true when an enclosing function call was identified.
func (p *parser) enclosingFunctionArgument(paramPosition int) (name string, ordinal int, ok bool) {
	enclosingParen := p.findEnclosingParen(paramPosition)
	if enclosingParen < 1 || p.tokens[enclosingParen-1].kind != tokenIdentifier {
		return "", 0, false
	}

	ordinal = p.countTopLevelArgumentsBefore(enclosingParen, paramPosition)
	return strings.ToLower(p.tokens[enclosingParen-1].value), ordinal, true
}

// countTopLevelArgumentsBefore counts the top-level comma separators between the token
// after openParen and paramPosition, which is the 0-based argument ordinal the
// placeholder occupies. Commas nested inside deeper parentheses (a sub-call's own
// argument list) are skipped so they do not inflate the ordinal.
//
// Takes openParen (int) which is the opening parenthesis token index of the call.
// Takes paramPosition (int) which is the placeholder's token index.
//
// Returns int which is the 0-based argument ordinal.
func (p *parser) countTopLevelArgumentsBefore(openParen, paramPosition int) int {
	ordinal := 0
	depth := 0
	for i := openParen + 1; i < paramPosition && i < len(p.tokens); i++ {
		switch p.tokens[i].kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenComma:
			if depth == 0 {
				ordinal++
			}
		}
	}
	return ordinal
}

// findEnclosingParen returns the token index of the parenthesis that encloses position.
//
// Takes position (int) which is the inner token index to search from.
//
// Returns int which is the enclosing `(` token index, or -1 when none encloses position.
func (p *parser) findEnclosingParen(position int) int {
	return engine_shared.FindEnclosingParen(position,
		func(index int) bool { return p.tokens[index].kind == tokenLeftParen },
		func(index int) bool { return p.tokens[index].kind == tokenRightParen },
		func(index int) bool {
			return p.tokens[index].kind == tokenIdentifier && isLikeBoundaryKeyword(strings.ToUpper(p.tokens[index].value))
		},
	)
}

// extractColumnReferenceBeforeIN returns the column referenced immediately before an IN
// keyword.
//
// Takes inPosition (int) which is the IN keyword's token index.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when none is
// available.
func (p *parser) extractColumnReferenceBeforeIN(inPosition int) *querier_dto.ColumnReference {
	if inPosition < 1 {
		return nil
	}
	return p.extractColumnReference(inPosition - 1)
}

// extractCastType resolves the target type of a CAST expression around a parameter
// position.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns *querier_dto.SQLType which is the target type, or nil when no AS keyword or
// type name is found.
func (p *parser) extractCastType(paramPosition int) *querier_dto.SQLType {
	asPosition := p.findASKeywordAfter(paramPosition)
	if asPosition < 0 {
		return nil
	}
	typeStart := asPosition + 1
	if typeStart >= len(p.tokens) || p.tokens[typeStart].kind != tokenIdentifier {
		return nil
	}
	typeName := p.collectCastTypeName(typeStart)
	return new(normaliseTypeName(typeName))
}

// findASKeywordAfter locates the next AS keyword after a position, stopping at the
// closing parenthesis of the surrounding CAST.
//
// Takes paramPosition (int) which is the parameter's token index.
//
// Returns int which is the AS keyword's token index, or -1 when none is found before the
// closing parenthesis.
func (p *parser) findASKeywordAfter(paramPosition int) int {
	for i := paramPosition + 1; i < len(p.tokens); i++ {
		if p.tokens[i].kind == tokenRightParen {
			return -1
		}
		if p.tokens[i].kind == tokenIdentifier && strings.EqualFold(p.tokens[i].value, keywordAS) {
			return i
		}
	}
	return -1
}

// collectCastTypeName joins consecutive identifier tokens into a space-separated CAST
// type name.
//
// Takes start (int) which is the token index of the first identifier.
//
// Returns string which is the space-joined type name.
func (p *parser) collectCastTypeName(start int) string {
	var builder strings.Builder
	builder.WriteString(p.tokens[start].value)
	for j := start + 1; j < len(p.tokens); j++ {
		if p.tokens[j].kind != tokenIdentifier || p.isCastTypeTerminator(j) {
			break
		}
		builder.WriteString(" ")
		builder.WriteString(p.tokens[j].value)
	}
	return builder.String()
}

// isCastTypeTerminator reports whether the token at position ends a CAST target type
// name.
//
// Takes position (int) which is the token index to inspect.
//
// Returns bool which is true at a closing parenthesis or a clause keyword.
func (p *parser) isCastTypeTerminator(position int) bool {
	if p.tokens[position].kind == tokenRightParen {
		return true
	}
	return p.isKeywordAt(
		position,
		keywordFROM, keywordWHERE, keywordGROUP, keywordHAVING,
		keywordORDER, keywordLIMIT,
	)
}

// isKeywordAt reports whether the identifier at position matches any of the supplied
// keywords ignoring case.
//
// Takes position (int) which is the token index to inspect.
// Takes keywords (...string) which are the candidate upper-case keywords.
//
// Returns bool which is true when the token matches any keyword.
func (p *parser) isKeywordAt(position int, keywords ...string) bool {
	if position >= len(p.tokens) || p.tokens[position].kind != tokenIdentifier {
		return false
	}
	return slices.Contains(keywords, strings.ToUpper(p.tokens[position].value))
}

// extractColumnReference reads a bare or `alias.column` column reference at the given
// token position.
//
// Takes position (int) which is the column identifier's token index.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when the
// token is not an identifier.
func (p *parser) extractColumnReference(position int) *querier_dto.ColumnReference {
	if position < 0 || position >= len(p.tokens) {
		return nil
	}

	tok := p.tokens[position]
	if tok.kind != tokenIdentifier {
		return nil
	}

	if position >= 2 && p.tokens[position-1].kind == tokenDot && p.tokens[position-2].kind == tokenIdentifier {
		return &querier_dto.ColumnReference{
			TableAlias: p.tokens[position-2].value,
			ColumnName: tok.value,
		}
	}

	return &querier_dto.ColumnReference{
		ColumnName: tok.value,
	}
}

// parseGroupByList parses a comma-separated GROUP BY column list.
//
// Returns []querier_dto.ColumnReference which is the parsed column list.
func (p *parser) parseGroupByList() []querier_dto.ColumnReference {
	var columns []querier_dto.ColumnReference

	for {
		columns = append(columns, p.parseGroupByItem()...)

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns
}

var (
	// groupByItemTerminators lists the keywords that end a GROUP BY item at depth zero, so a
	// non-column group key (a function call or arithmetic expression) is scanned in full and
	// the cursor lands on the next clause keyword rather than mid-expression.
	groupByItemTerminators = map[string]bool{
		keywordHAVING: true, keywordORDER: true, keywordLIMIT: true,
		keywordUNION: true, keywordINTERSECT: true, keywordEXCEPT: true,
		keywordRETURNING: true,
	}
)

// parseGroupByItem parses one GROUP BY item.
//
// A bare column or `alias.column` reference is captured as a column reference. Any other
// group key (such as DATE(created_at) or year + 1) is scanned as a full expression so the
// cursor stops on the next clause keyword and a following HAVING and its parameters are
// not abandoned.
//
// Returns []querier_dto.ColumnReference which holds the parsed column reference when the
// item is a plain column, or nil for an expression group key.
func (p *parser) parseGroupByItem() []querier_dto.ColumnReference {
	if column, ok := p.tryParseGroupByColumn(); ok {
		return column
	}

	p.scanGroupByExpression()
	return nil
}

// tryParseGroupByColumn consumes a plain `column` or `alias.column` group key when the
// next item boundary follows it directly, leaving the cursor unmoved otherwise so the
// caller can fall back to a full expression scan.
//
// Returns []querier_dto.ColumnReference which holds the single parsed column reference.
// Returns bool which is true when a plain column reference was consumed.
func (p *parser) tryParseGroupByColumn() ([]querier_dto.ColumnReference, bool) {
	if p.current().kind != tokenIdentifier {
		return nil, false
	}

	if isGroupByItemBoundary(p.peek()) {
		return []querier_dto.ColumnReference{{ColumnName: p.advance().value}}, true
	}

	if p.peek().kind != tokenDot {
		return nil, false
	}

	if p.tokenAt(p.position+2).kind != tokenIdentifier || !isGroupByItemBoundary(p.tokenAt(p.position+3)) {
		return nil, false
	}

	alias := p.advance().value
	p.advance()
	return []querier_dto.ColumnReference{{TableAlias: alias, ColumnName: p.advance().value}}, true
}

// scanGroupByExpression walks an expression group key at depth zero, registering any
// parameters, and stops at the first depth-zero comma or clause terminator so the next
// clause is parsed correctly.
func (p *parser) scanGroupByExpression() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()

		if tok.kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if tok.kind == tokenRightParen {
			if depth == 0 {
				break
			}
			depth--
			p.advance()
			continue
		}

		if depth == 0 && (tok.kind == tokenComma || isGroupByItemTerminator(tok)) {
			break
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// isGroupByItemBoundary reports whether tok closes a GROUP BY item, marking the preceding
// tokens as a complete plain column reference.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true for a comma, a depth-zero clause terminator, or end of
// input.
func isGroupByItemBoundary(tok token) bool {
	return tok.kind == tokenComma || tok.kind == tokenEOF || isGroupByItemTerminator(tok)
}

// isGroupByItemTerminator reports whether tok is a keyword that ends a GROUP BY item.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true for any keyword listed in groupByItemTerminators.
func isGroupByItemTerminator(tok token) bool {
	return tok.kind == tokenIdentifier && groupByItemTerminators[strings.ToUpper(tok.value)]
}

var (
	// orderByTerminators lists keywords that end the ORDER BY clause at depth zero during
	// the forward scan.
	orderByTerminators = map[string]bool{
		keywordLIMIT: true, keywordUNION: true, keywordINTERSECT: true,
		keywordEXCEPT: true, keywordRETURNING: true,
	}
)

// parseOrderByList walks through the ORDER BY clause, registering any parameters and
// stopping at the first depth-zero terminator.
func (p *parser) parseOrderByList() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()

		if tok.kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if tok.kind == tokenRightParen {
			if depth == 0 {
				break
			}
			depth--
			p.advance()
			continue
		}

		if depth == 0 && isOrderByTerminator(tok) {
			break
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// isOrderByTerminator reports whether tok ends the ORDER BY clause.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true for any keyword listed in orderByTerminators.
func isOrderByTerminator(tok token) bool {
	return tok.kind == tokenIdentifier && orderByTerminators[strings.ToUpper(tok.value)]
}

var (
	// limitOffsetValueTerminators lists the keywords that end a LIMIT or OFFSET value at
	// depth zero. SQLite allows an arbitrary integer expression in LIMIT or OFFSET, so a
	// compound value is scanned up to one of these keywords (or a top-level comma) that
	// begins the next clause.
	limitOffsetValueTerminators = map[string]bool{
		"OFFSET": true, keywordLIMIT: true, keywordUNION: true, keywordINTERSECT: true,
		keywordEXCEPT: true, keywordRETURNING: true,
	}
)

// parseLimitOffset consumes a LIMIT value with an optional OFFSET or comma-separated
// second value, registering any parameter bindings.
func (p *parser) parseLimitOffset() {
	p.consumeLimitOffsetValue(querier_dto.ParameterContextLimit)

	if p.matchKeyword("OFFSET") {
		p.consumeLimitOffsetValue(querier_dto.ParameterContextOffset)
	} else if p.current().kind == tokenComma {
		p.advance()
		p.consumeLimitOffsetValue(querier_dto.ParameterContextOffset)
	}
}

// consumeLimitOffsetValue consumes a single LIMIT or OFFSET value.
//
// A bare placeholder is registered with the given context (preserving the integer typing
// and limit or offset name that context drives). A compound value such as COALESCE(?1,
// 100) or a parenthesised subexpression is scanned in full so a nested placeholder is
// still registered rather than dropped; the scan stops at a top-level comma or a keyword
// that begins the next clause.
//
// Takes context (querier_dto.ParameterContext) which tags a bare-placeholder value.
func (p *parser) consumeLimitOffsetValue(context querier_dto.ParameterContext) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, context, nil, nil)
		return
	}

	referenceCountBefore := len(p.parameterRefs)
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if tok.kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if tok.kind == tokenRightParen {
			if depth == 0 {
				break
			}
			depth--
			p.advance()
			continue
		}
		if depth == 0 && (tok.kind == tokenComma || isLimitOffsetValueTerminator(tok)) {
			break
		}
		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}
		p.advance()
	}
	p.retagLimitOffsetParameters(referenceCountBefore, context)
}

// retagLimitOffsetParameters re-tags every parameter registered while scanning a compound
// LIMIT or OFFSET value with the limit or offset context.
//
// An example is the placeholder in LIMIT COALESCE(?n, 100). Re-tagging overrides any
// incidental function-argument context the inner scan assigned. A placeholder in a LIMIT
// or OFFSET value is an integer row count, so it types as an integer rather than the
// polymorphic COALESCE argument type.
//
// Takes referenceCountBefore (int), the parameterRefs length before the value scan.
// Takes context (querier_dto.ParameterContext), the limit or offset context to apply.
func (p *parser) retagLimitOffsetParameters(referenceCountBefore int, context querier_dto.ParameterContext) {
	for i := referenceCountBefore; i < len(p.parameterRefs); i++ {
		p.parameterRefs[i].Context = context
		p.parameterRefs[i].EnclosingFunctionName = ""
		p.parameterRefs[i].ArgumentOrdinal = 0
	}
}

// isLimitOffsetValueTerminator reports whether tok is a keyword that ends a LIMIT/OFFSET
// value at depth zero.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true for any keyword in limitOffsetValueTerminators.
func isLimitOffsetValueTerminator(tok token) bool {
	return tok.kind == tokenIdentifier && limitOffsetValueTerminators[strings.ToUpper(tok.value)]
}

// isInsertSourceTerminator reports whether tok ends the INSERT source scan at the given
// parenthesis depth.
//
// Takes tok (token) which is the current token.
// Takes depth (int) which is the current parenthesis depth.
//
// Returns bool which is true at depth zero for `)`, ON, or RETURNING.
func isInsertSourceTerminator(tok token, depth int) bool {
	if tok.kind == tokenRightParen && depth == 0 {
		return true
	}
	if depth == 0 && tok.kind == tokenIdentifier {
		upper := strings.ToUpper(tok.value)
		return upper == keywordON || upper == keywordRETURNING
	}
	return false
}

// parseInsertValues dispatches on the INSERT payload: a VALUES row list, DEFAULT VALUES,
// a SELECT/WITH query source, or an opaque source.
//
// Takes tableName (string) which scopes column lookups in the VALUES clause.
// Takes columnNames ([]string) which lists the target columns for parameter binding.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed SELECT body of an INSERT
// ... SELECT (nil for VALUES/DEFAULT VALUES/opaque sources). The body is analysed so its
// FROM/JOIN relations and parameters resolve against the body's own scope; analyseSelect
// stops at a trailing ON CONFLICT (keywordON is a WHERE terminator) or RETURNING.
// Returns error when the SELECT body fails to parse.
func (p *parser) parseInsertValues(tableName string, columnNames []string) (*querier_dto.RawQueryAnalysis, error) {
	switch {
	case p.matchKeyword(keywordVALUES):
		p.parseValuesClause(tableName, columnNames)
	case p.matchKeyword("DEFAULT"):
		p.matchKeyword(keywordVALUES)
	case p.isKeyword(keywordSELECT) || p.isKeyword(keywordWITH):

		p.insertTargetTable = tableName
		p.insertTargetColumns = columnNames
		return p.analyseSelect()
	default:
		p.parseInsertSource()
	}
	return nil, nil
}

// parseInsertSource walks the token stream of an INSERT source, registering any parameter
// bindings and tracking parenthesis depth.
func (p *parser) parseInsertSource() {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if isInsertSourceTerminator(tok, depth) {
			break
		}

		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		}

		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// parseValuesClause parses a `VALUES (...), (...)` row list and associates each parameter
// with its target column.
//
// Takes tableName (string) which is the target table name used as the column reference
// alias.
// Takes columnNames ([]string) which are the target column names in positional order.
func (p *parser) parseValuesClause(tableName string, columnNames []string) {
	for p.current().kind == tokenLeftParen {
		p.advance()
		p.parseOneValueRow(tableName, columnNames)

		if p.current().kind == tokenRightParen {
			p.advance()
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// parseOneValueRow parses a single `(...)` row inside a VALUES clause.
//
// Takes tableName (string) which is the target table name used as the column reference
// alias.
// Takes columnNames ([]string) which are the target column names in positional order.
func (p *parser) parseOneValueRow(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		columnIndex += p.parseOneValueElement(tableName, columnNames, columnIndex)
	}
}

// parseOneValueElement consumes one VALUES element and returns whether it crossed a comma
// into the next column.
//
// Takes tableName (string) which is the target table name used as the column reference
// alias.
// Takes columnNames ([]string) which are the target column names in positional order.
// Takes columnIndex (int) which is the current column ordinal.
//
// Returns int which is 1 when a trailing comma was consumed, else 0.
func (p *parser) parseOneValueElement(tableName string, columnNames []string, columnIndex int) int {
	if isParameterToken(p.current().kind) {
		columnReference := columnRefForIndex(tableName, columnNames, columnIndex)
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnReference, nil)
	} else if p.current().kind == tokenLeftParen {
		columnReference := columnRefForIndex(tableName, columnNames, columnIndex)
		p.scanInsertValueExpression(columnReference)
	} else if p.current().kind != tokenComma {
		p.advance()
	}

	if p.current().kind == tokenComma {
		p.advance()
		return 1
	}
	return 0
}

// scanInsertValueExpression walks the parenthesised expression starting at the current
// token, registering every parameter it encounters with the supplied columnReference.
// Non-parameter tokens are advanced over so the cursor lands just past the matching
// closing paren.
//
// Used by parseOneValueElement to keep INSERT parameters inside function calls like
// COALESCE(?, default) properly bound to the target column. Mirrors the equivalent
// helpers in the PostgreSQL, MySQL and DuckDB engines.
//
// Takes columnReference (*querier_dto.ColumnReference) which is the INSERT column the
// parenthesised expression is the value for.
func (p *parser) scanInsertValueExpression(columnReference *querier_dto.ColumnReference) {
	if p.current().kind != tokenLeftParen {
		return
	}
	p.advance()
	depth := 1
	for !p.atEnd() && depth > 0 {
		tok := p.current()
		switch tok.kind {
		case tokenLeftParen:
			depth++
			p.advance()
		case tokenRightParen:
			depth--
			p.advance()
		default:
			if isParameterToken(tok.kind) {
				parameterToken := tok
				p.advance()
				p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnReference, nil)
				continue
			}
			p.advance()
		}
	}
}

// columnRefForIndex resolves the target column reference for a positional VALUES element.
//
// Takes tableName (string) which is the target table name used as the column reference
// alias.
// Takes columnNames ([]string) which are the target column names in positional order.
// Takes columnIndex (int) which is the current column ordinal.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when the
// index is beyond the declared columns.
func columnRefForIndex(tableName string, columnNames []string, columnIndex int) *querier_dto.ColumnReference {
	if columnIndex >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[columnIndex],
	}
}

// parseSetClause parses an UPDATE SET clause, registering each parameter against its
// target column.
//
// Takes tableName (string) which is the target table name used as the column reference
// alias.
func (p *parser) parseSetClause(tableName string) {
	for {
		columnName := ""
		if p.current().kind == tokenIdentifier {
			columnName = p.advance().value
		}

		if p.current().kind == tokenOperator && p.current().value == "=" {
			p.advance()
		}

		var columnRef *querier_dto.ColumnReference
		if columnName != "" {
			columnRef = &querier_dto.ColumnReference{
				TableAlias: tableName,
				ColumnName: columnName,
			}
		}

		if isParameterToken(p.current().kind) {
			parameterToken := p.current()
			p.advance()
			p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
		}

		p.skipSetExpressionWithColumn(columnRef)

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

var (
	// setExpressionTerminators lists keywords that end a SET expression at depth zero during
	// the forward scan.
	setExpressionTerminators = map[string]bool{
		keywordWHERE: true, keywordFROM: true, keywordRETURNING: true,
		keywordORDER: true, keywordLIMIT: true,
	}
)

// skipSetExpressionWithColumn walks the RHS of a single SET assignment, stopping at a
// comma or a SET-clause terminator (WHERE, FROM, RETURNING, etc.) at depth 0.
//
// A parameter that is the operand of a comparison inside the expression (for example the
// `?` in `SET col = CASE WHEN other >= ? THEN ... END`) takes its type from the compared
// column, not the assignment target: the scanner remembers the most recent `<column>
// <comparison-operator>` pair and binds the following parameter to that column. Any other
// parameter (one that contributes to the assigned value, such as `COALESCE(?, col)`)
// keeps the assignment-target column reference so it still picks up that column's Go
// type. Without this the comparison operand was wrongly typed from the SET target,
// producing a Params field whose Go type did not match the value the caller binds.
//
// Takes setTargetRef (*querier_dto.ColumnReference) which is the column on the left of
// the SET assignment; nil keeps the legacy no-context behaviour.
func (p *parser) skipSetExpressionWithColumn(setTargetRef *querier_dto.ColumnReference) {
	tableAlias := ""
	if setTargetRef != nil {
		tableAlias = setTargetRef.TableAlias
	}
	depth := 0
	var comparisonColumn *querier_dto.ColumnReference
	lastIdentifier := ""
	for !p.atEnd() {
		tok := p.current()

		nextDepth, stop, handled := p.handleSetExpressionParen(tok, depth)
		if stop {
			break
		}
		if handled {
			depth = nextDepth
			continue
		}
		if depth == 0 && isSetExpressionTerminator(tok) {
			break
		}

		if isParameterToken(tok.kind) {
			p.registerSetExpressionParameter(tok, setTargetRef, comparisonColumn)
			comparisonColumn = nil
			lastIdentifier = ""
			continue
		}

		lastIdentifier, comparisonColumn = comparisonColumnForSetParameter(tok, tableAlias, lastIdentifier)
		p.advance()
	}
}

// comparisonColumnForSetParameter advances the comparison scan over a SET expression.
//
// When the previous identifier is immediately followed by a comparison operator, that
// column is reported so a following parameter is typed from it; for example the parameter
// in "WHEN attempt >= ?" is typed from attempt. Any other token clears the pending state.
//
// Takes tok (token) which is the token being observed.
// Takes tableAlias (string) which qualifies the reported column reference.
// Takes lastIdentifier (string) which is the identifier from the previous step.
//
// Returns nextIdentifier (string) carried into the next step.
// Returns comparisonColumn (*querier_dto.ColumnReference) which is the compared column,
// or nil when no identifier is followed by a comparison operator.
func comparisonColumnForSetParameter(
	tok token,
	tableAlias, lastIdentifier string,
) (nextIdentifier string, comparisonColumn *querier_dto.ColumnReference) {
	switch {
	case tok.kind == tokenIdentifier:
		return tok.value, nil
	case tok.kind == tokenOperator && isComparisonOperator(tok.value) && lastIdentifier != "":
		return "", &querier_dto.ColumnReference{TableAlias: tableAlias, ColumnName: lastIdentifier}
	default:
		return "", nil
	}
}

// handleSetExpressionParen processes a parenthesis token within a SET expression scan.
// Returns the next depth value, whether to stop the outer loop (unmatched right paren at
// depth zero), and whether the token was a parenthesis at all.
//
// Takes tok (token) which is the token under consideration.
// Takes depth (int) which is the current parenthesis nesting depth.
//
// Returns nextDepth (int) which is the updated nesting depth.
// Returns stop (bool) which is true when the outer loop should break.
// Returns handled (bool) which is true when the token was a left or right paren and has
// already been consumed.
func (p *parser) handleSetExpressionParen(tok token, depth int) (nextDepth int, stop bool, handled bool) {
	if tok.kind == tokenLeftParen {
		p.advance()
		return depth + 1, false, true
	}
	if tok.kind == tokenRightParen {
		if depth == 0 {
			return depth, true, true
		}
		p.advance()
		return depth - 1, false, true
	}
	return depth, false, false
}

// registerSetExpressionParameter consumes the parameter token and registers it with the
// column reference and context that match its position in the SET expression.
//
// When comparisonColumn is non-nil the parameter is an operand of a comparison (for
// example `WHEN attempt >= ?`), so it is recorded against that column with comparison
// context. Otherwise it is treated as contributing to the assigned value and recorded
// against the SET-target column with assignment context (or unknown context when no
// target is known).
//
// Takes parameterToken (token) which is the placeholder token to record.
// Takes setTargetRef (*querier_dto.ColumnReference) which is the SET-target column.
// Takes comparisonColumn (*querier_dto.ColumnReference) which is the column the parameter
// is compared against, or nil when the parameter is not a comparison operand.
func (p *parser) registerSetExpressionParameter(
	parameterToken token,
	setTargetRef *querier_dto.ColumnReference,
	comparisonColumn *querier_dto.ColumnReference,
) {
	p.advance()
	if comparisonColumn != nil {
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextComparison, comparisonColumn, nil)
		return
	}
	context := querier_dto.ParameterContextUnknown
	if setTargetRef != nil {
		context = querier_dto.ParameterContextAssignment
	}
	p.registerParameterFromToken(parameterToken, context, setTargetRef, nil)
}

// isSetExpressionTerminator reports whether tok ends a SET expression scan at depth zero.
//
// Takes tok (token) which is the token to inspect.
//
// Returns bool which is true for commas or keywords in setExpressionTerminators.
func isSetExpressionTerminator(tok token) bool {
	if tok.kind == tokenComma {
		return true
	}
	return tok.kind == tokenIdentifier && setExpressionTerminators[strings.ToUpper(tok.value)]
}

// skipOnConflict consumes an ON CONFLICT clause including any DO NOTHING or DO UPDATE SET
// action.
//
// Takes tableName (string) which is the target table name used when recursing into the DO
// UPDATE SET clause.
func (p *parser) skipOnConflict(tableName string) {
	p.matchKeyword("CONFLICT")

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		p.skipConflictTargetPredicate()
	}

	if p.matchKeyword("DO") {
		if p.matchKeyword("NOTHING") {
			return
		}
		if p.matchKeyword("UPDATE") {
			if p.matchKeyword(keywordSET) {
				p.parseSetClause(tableName)
			}
			if p.matchKeyword(keywordWHERE) {
				p.parseWhereExpression()
			}
		}
	}
}

var (
	// onConflictTargetPredicateTerminators lists the keywords that end the optional WHERE
	// index-predicate of an ON CONFLICT target at depth zero. The predicate stops before the
	// DO action; RETURNING guards against a malformed clause that omits the action entirely.
	onConflictTargetPredicateTerminators = map[string]bool{
		"DO": true, keywordRETURNING: true,
	}
)

// skipConflictTargetPredicate consumes the optional WHERE index-predicate that follows a
// parenthesised ON CONFLICT target, which SQLite permits for partial-index UPSERT.
func (p *parser) skipConflictTargetPredicate() {
	if !p.matchKeyword(keywordWHERE) {
		return
	}
	p.parseExpressionUntilTerminator(onConflictTargetPredicateTerminators)
}
