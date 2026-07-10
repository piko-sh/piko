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

// parseWhereClause skips tokens of a WHERE expression up to the next clause boundary,
// registering parameters along the way.
func (p *parser) parseWhereClause() {
	p.parseExpressionUntilTerminator()
}

var (
	// expressionTerminatorKeywords lists keywords that terminate a free-form expression.
	//
	// The set covers WHERE, ON and HAVING. JOIN-introducing keywords are included so a
	// JOIN's ON expression stops at the next JOIN rather than swallowing the subsequent JOIN
	// clause, which would otherwise prevent the next joined table from being added to the
	// scope chain and leave its projected columns typed as `any`.
	expressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordQUALIFY: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		keywordPIVOT: {}, keywordUNPIVOT: {},
		keywordJOIN: {}, "INNER": {}, "LEFT": {}, "RIGHT": {},
		"FULL": {}, "CROSS": {}, "NATURAL": {},
	}

	// joinKeywordTerminators is the subset of expressionTerminatorKeywords whose tokens are
	// also valid SQL function names.
	//
	// Examples include LEFT() and RIGHT(). When such a token is immediately followed by an
	// opening parenthesis it is a function call rather than a JOIN starter, so the
	// terminator check skips it to avoid prematurely ending the surrounding expression and
	// silently dropping any parameters that follow.
	joinKeywordTerminators = map[string]struct{}{
		"INNER": {}, "LEFT": {}, "RIGHT": {}, "FULL": {}, "CROSS": {}, "NATURAL": {},
	}

	// whereExpressionTerminatorKeywords is the terminator set used for WHERE and HAVING
	// expressions.
	//
	// It deliberately omits the JOIN-introducing keywords (JOIN, INNER, LEFT, RIGHT, FULL,
	// CROSS, NATURAL) because those are all legal unquoted column names in DuckDB, and a
	// WHERE/HAVING predicate never legitimately starts a JOIN at top level (JOINs precede
	// WHERE in the FROM clause). Including them would treat a column named
	// `left`/`right`/`inner` and the like as a clause boundary and silently drop every
	// parameter that follows it. The ON-clause scan keeps the JOIN keywords via
	// expressionTerminatorKeywords so a join condition still stops at the next JOIN.
	whereExpressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordQUALIFY: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		keywordPIVOT: {}, keywordUNPIVOT: {},
	}
)

// parseExpressionUntilTerminator skips a WHERE or HAVING expression up to the next clause
// boundary, registering parameters along the way.
func (p *parser) parseExpressionUntilTerminator() {
	p.skipTokensUntilTerminatorSet(whereExpressionTerminatorKeywords)
}

// parseJoinConditionExpression skips a JOIN ON predicate.
//
// Unlike a WHERE/HAVING expression, a join condition is genuinely followed by the next
// JOIN, so the JOIN-introducing keywords remain terminators here (via
// expressionTerminatorKeywords). The handleExpressionTerminator function-call exemption
// still distinguishes LEFT()/RIGHT() calls from JOIN starters.
func (p *parser) parseJoinConditionExpression() {
	p.skipTokensUntilTerminatorSet(expressionTerminatorKeywords)
}

// skipTokensUntilTerminatorSet advances tokens until a top-level keyword from terminators
// or an unmatched closing parenthesis.
//
// Takes terminators (map[string]struct{}) which is the upper-case keyword set that ends
// the walk.
func (p *parser) skipTokensUntilTerminatorSet(terminators map[string]struct{}) {
	p.skipTokensUntilTerminator(terminators, false)
}

// skipTokensUntilTerminator advances tokens to a stopping point.
//
// It stops on a top-level keyword from terminators, an unmatched closing parenthesis, or
// (when stopAtTopLevelComma is set) a comma at depth zero. The comma stop lets the "LIMIT
// value, offset" comma form split its two operands even when the LIMIT value is a
// compound expression whose own commas sit inside parentheses.
//
// Takes terminators (map[string]struct{}) which is the upper-case keyword set that ends
// the walk.
// Takes stopAtTopLevelComma (bool) which stops the walk on a depth-zero comma when true.
func (p *parser) skipTokensUntilTerminator(terminators map[string]struct{}, stopAtTopLevelComma bool) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()

		nextDepth, stop, handled := p.handleExpressionParen(tok, depth)
		if stop {
			break
		}
		if handled {
			depth = nextDepth
			continue
		}
		if stopsAtTopLevelComma(tok, depth, stopAtTopLevelComma) {
			break
		}
		if depth == 0 && p.isKeywordTerminator(tok, terminators) {
			if p.handleExpressionTerminator(tok) {
				break
			}
			continue
		}
		if isParameterToken(tok.kind) {
			p.handleParameterInExpression()
			continue
		}

		p.advance()
	}
}

// stopsAtTopLevelComma reports whether the walk should stop on the given token: a
// depth-zero comma when the comma-stop mode is enabled. It isolates the comma-form split
// condition so the main scan loop stays flat.
//
// Takes tok (token) which is the token under consideration.
// Takes depth (int) which is the current parenthesis nesting depth.
// Takes enabled (bool) which is true when the comma stop is active.
//
// Returns bool which is true when the walk should break on this token.
func stopsAtTopLevelComma(tok token, depth int, enabled bool) bool {
	return enabled && depth == 0 && tok.kind == tokenComma
}

// handleExpressionParen processes a parenthesis token within an expression scan. Returns
// the next depth value, whether to break the outer loop (unmatched right paren at depth
// zero), and whether the token was a parenthesis at all (and thus already consumed).
//
// Takes tok (token) which is the token under consideration.
// Takes depth (int) which is the current parenthesis nesting depth.
//
// Returns nextDepth (int) which is the updated nesting depth.
// Returns stop (bool) which is true when the outer loop should break.
// Returns handled (bool) which is true when the token was consumed here.
func (p *parser) handleExpressionParen(tok token, depth int) (nextDepth int, stop bool, handled bool) {
	if tok.kind == tokenLeftParen {
		if p.isSubqueryStart() {
			if innerAnalysis, ok := p.analyseSubqueryBody(); ok {
				p.predicateSubqueries = append(p.predicateSubqueries, innerAnalysis)
			}
			return depth, false, true
		}
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

// handleExpressionTerminator distinguishes a real JOIN keyword from a function call
// sharing the same name (LEFT(), RIGHT(), etc.). Consumes the identifier when it is
// followed by an opening parenthesis, returning false so the caller continues; otherwise
// returns true so the caller breaks out of the expression scan.
//
// Takes tok (token) which is the candidate terminator token.
//
// Returns bool which is true when the caller should break.
func (p *parser) handleExpressionTerminator(tok token) bool {
	if _, isJoinKeyword := joinKeywordTerminators[strings.ToUpper(tok.value)]; isJoinKeyword && p.peek().kind == tokenLeftParen {
		p.advance()
		return false
	}
	return true
}

// isKeywordTerminator reports whether tok is an identifier whose upper-case value is in
// the terminator set.
//
// Takes tok (token) which is the token under consideration.
// Takes terminators (map[string]struct{}) which is the upper-case keyword set to test.
//
// Returns bool which is true when the token is a terminator keyword.
func (*parser) isKeywordTerminator(tok token, terminators map[string]struct{}) bool {
	if tok.kind != tokenIdentifier {
		return false
	}
	_, ok := terminators[strings.ToUpper(tok.value)]
	return ok
}

// parseGroupByClause reads a comma-separated GROUP BY column list.
//
// Returns []querier_dto.ColumnReference which is the parsed column reference list.
func (p *parser) parseGroupByClause() []querier_dto.ColumnReference {
	var columns []querier_dto.ColumnReference

	for {
		column, ok := p.parseGroupByColumn()
		if ok {
			columns = append(columns, column)
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}

	return columns
}

// parseGroupByColumn parses a single GROUP BY entry, accepting bare or schema-qualified
// identifiers.
//
// Returns querier_dto.ColumnReference which is the parsed reference.
// Returns bool which is true when a column reference was produced.
func (p *parser) parseGroupByColumn() (querier_dto.ColumnReference, bool) {
	if p.current().kind != tokenIdentifier {
		p.advance()
		return querier_dto.ColumnReference{}, false
	}
	first := p.advance().value
	if p.current().kind != tokenDot {
		return querier_dto.ColumnReference{ColumnName: first}, true
	}
	p.advance()
	if p.current().kind != tokenIdentifier {
		return querier_dto.ColumnReference{ColumnName: first}, true
	}
	second := p.advance().value
	return querier_dto.ColumnReference{TableAlias: first, ColumnName: second}, true
}

var (
	// orderByTerminators lists keywords that end an ORDER BY list at depth zero.
	orderByTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, "WINDOW": {},
	}
)

// parseOrderByList skips ORDER BY expressions up to the next clause boundary.
func (p *parser) parseOrderByList() {
	p.skipTokensUntilTerminatorSet(orderByTerminators)
}

var (
	// limitOffsetValueTerminators lists the keywords that end a LIMIT/OFFSET/FETCH value
	// expression at the top level, so scanning a compound value (a function call or
	// parenthesised expression) stops at the keyword introducing the next clause.
	limitOffsetValueTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {}, keywordRETURNING: {},
		keywordROW: {}, keywordROWS: {}, keywordWITH: {}, "ONLY": {}, "TIES": {},
		"NEXT": {}, "FIRST": {},
	}
)

// consumeParameterOrAdvance consumes a LIMIT/OFFSET value. A bare placeholder is
// registered with the given context (preserving integer typing and the limit/offset name
// that context drives); a compound value such as COALESCE($1, 100) or a parenthesised
// subexpression is scanned in full so any nested placeholder is still registered rather
// than dropped.
//
// Takes context (querier_dto.ParameterContext) which tags a bare-placeholder value.
func (p *parser) consumeParameterOrAdvance(context querier_dto.ParameterContext) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, context, nil, nil)
		return
	}
	referenceCountBefore := len(p.parameterRefs)
	p.skipTokensUntilTerminatorSet(limitOffsetValueTerminators)
	p.retagLimitOffsetParameters(referenceCountBefore, context)
}

// consumeLimitValueOrAdvance consumes the LIMIT value in the "LIMIT value, offset" comma
// form. It behaves like consumeParameterOrAdvance for a bare placeholder, but a compound
// LIMIT value is scanned only up to the top-level comma so the trailing offset operand is
// not swallowed into the LIMIT expression.
func (p *parser) consumeLimitValueOrAdvance() {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextLimit, nil, nil)
		return
	}
	referenceCountBefore := len(p.parameterRefs)
	p.skipTokensUntilTerminator(limitOffsetValueTerminators, true)
	p.retagLimitOffsetParameters(referenceCountBefore, querier_dto.ParameterContextLimit)
}

// retagLimitOffsetParameters re-tags every parameter registered while scanning a LIMIT or
// OFFSET value with the limit/offset context.
//
// The placeholder in LIMIT COALESCE($n, 100) is one example. Re-tagging overrides any
// incidental function-argument context the inner scan assigned. A placeholder in a
// LIMIT/OFFSET value is an integer row count, so it types as an integer rather than the
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

// parseLimitOffset dispatches over the LIMIT, OFFSET, and FETCH clause shapes accepted by
// DuckDB.
func (p *parser) parseLimitOffset() {
	p.parseLeadingOffset()

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitClause()
		return
	}

	p.parseFetchClause()
	p.parseTrailingOffset()
}

// parseLeadingOffset consumes an OFFSET clause that appears before any LIMIT or FETCH
// clause.
func (p *parser) parseLeadingOffset() {
	if !p.isKeyword(keywordOFFSET) {
		return
	}
	p.advance()
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseLimitClause consumes a LIMIT expression and any trailing OFFSET or "limit, offset"
// comma form.
func (p *parser) parseLimitClause() {
	if !p.matchKeyword(keywordALL) {
		p.consumeLimitValueOrAdvance()
	}

	if p.matchKeyword(keywordOFFSET) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
		p.matchKeyword(keywordROW)
		p.matchKeyword(keywordROWS)
	} else if p.current().kind == tokenComma {
		p.advance()
		p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	}
}

// parseFetchClause consumes a FETCH FIRST/NEXT N ROWS [ONLY|WITH TIES] clause when one is
// present.
func (p *parser) parseFetchClause() {
	if !p.matchKeyword(keywordFETCH) {
		return
	}
	p.matchKeyword(keywordFIRST)
	p.matchKeyword("NEXT")

	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		p.advance()
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextLimit, nil, nil)
	} else if p.current().kind == tokenNumber {
		p.advance()
	} else {
		p.skipTokensUntilTerminatorSet(limitOffsetValueTerminators)
	}

	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
	p.matchKeyword("ONLY")
	p.matchKeyword(keywordWITH)
	p.matchKeyword("TIES")
}

// parseTrailingOffset consumes an OFFSET clause that follows a FETCH clause, unless a
// LIMIT is about to start instead.
func (p *parser) parseTrailingOffset() {
	if p.isKeyword(keywordLIMIT) || !p.matchKeyword(keywordOFFSET) {
		return
	}
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseCompoundQuery consumes an optional UNION, INTERSECT, or EXCEPT operator, honouring
// the ALL modifier.
//
// Returns querier_dto.CompoundOperator which is the operator that was consumed, or zero
// when none was present.
func (p *parser) parseCompoundQuery() querier_dto.CompoundOperator {
	if p.matchKeyword(keywordUNION) {
		if p.matchKeyword(keywordALL) {
			return querier_dto.CompoundUnionAll
		}
		return querier_dto.CompoundUnion
	}
	if p.matchKeyword(keywordINTERSECT) {
		p.matchKeyword(keywordALL)
		return querier_dto.CompoundIntersect
	}
	if p.matchKeyword(keywordEXCEPT) {
		p.matchKeyword(keywordALL)
		return querier_dto.CompoundExcept
	}
	return 0
}

// skipForUpdateClause consumes a FOR UPDATE/SHARE locking clause and records its presence
// on the parser.
func (p *parser) skipForUpdateClause() {
	if !p.matchKeyword(keywordFOR) {
		return
	}
	p.hasForUpdate = true
	p.matchKeyword("UPDATE")
	p.matchKeyword("NO")
	p.matchKeyword(keywordKEY)
	p.matchKeyword("SHARE")
	if p.matchKeyword("OF") {
		for {
			if p.current().kind == tokenIdentifier {
				p.advance()
			}
			if p.current().kind != tokenComma {
				break
			}
			p.advance()
		}
	}
	p.matchKeyword("NOWAIT")
	p.matchKeyword("SKIP")
	p.matchKeyword("LOCKED")
}

// parseValuesClause walks a VALUES (...), (...), ... source attached to an INSERT,
// binding parameters to their target columns.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the explicit column list, or nil when none was
// provided.
func (p *parser) parseValuesClause(tableName string, columnNames []string) {
	for p.current().kind == tokenLeftParen {
		p.advance()
		p.parseValuesRow(tableName, columnNames)

		if p.current().kind == tokenRightParen {
			p.advance()
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// columnRefForIndex builds the column reference associated with the index-th positional
// value in a VALUES row.
//
// Takes tableName (string) which is the target table for the reference.
// Takes columnNames ([]string) which is the declared column list.
// Takes index (int) which is the position within the row.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when index
// is past the declared list.
func (*parser) columnRefForIndex(tableName string, columnNames []string, index int) *querier_dto.ColumnReference {
	if index >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[index],
	}
}

// parseValuesRow walks one parenthesised VALUES row's elements.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the declared column list.
func (p *parser) parseValuesRow(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		p.parseValuesRowElement(tableName, columnNames, columnIndex)

		if p.current().kind == tokenComma {
			p.advance()
			columnIndex++
		}
	}
}

// parseValuesRowElement handles one element of a VALUES row, binding a parameter to its
// target column when present.
//
// Takes tableName (string) which is the INSERT target table.
// Takes columnNames ([]string) which is the declared column list.
// Takes columnIndex (int) which is the element's position in the row.
func (p *parser) parseValuesRowElement(tableName string, columnNames []string, columnIndex int) {
	if isParameterToken(p.current().kind) {
		parameterToken := p.current()
		columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
		p.advance()
		var castType *querier_dto.SQLType
		if p.current().kind == tokenCast {
			castType = p.consumeInlineCast()
		}
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, castType)
		return
	}
	if p.current().kind == tokenLeftParen {
		columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
		p.scanInsertValueExpression(columnRef)
		return
	}
	if p.current().kind != tokenComma {
		p.advance()
	}
}

// scanInsertValueExpression walks the parenthesised expression starting at the current
// token, registering every parameter it encounters with the supplied columnRef and an
// explicit cast type when one follows the parameter (e.g. $1::INT).
//
// The helper mirrors the equivalent one in the PostgreSQL engine.
//
// Takes columnRef (*querier_dto.ColumnReference) which is the INSERT column this VALUES
// position binds to.
func (p *parser) scanInsertValueExpression(columnRef *querier_dto.ColumnReference) {
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
			p.handleInsertValueDefaultToken(tok, columnRef)
		}
	}
}

// handleInsertValueDefaultToken processes a non-parenthesis token encountered while
// scanning an INSERT VALUES expression. Parameter tokens register the supplied columnRef
// and pick up any inline cast that follows ($N::type); other tokens are skipped.
//
// Takes tok (token) which is the token under consideration.
// Takes columnRef (*querier_dto.ColumnReference) which is the column the value binds to.
func (p *parser) handleInsertValueDefaultToken(tok token, columnRef *querier_dto.ColumnReference) {
	if !isParameterToken(tok.kind) {
		p.advance()
		return
	}
	parameterToken := tok
	p.advance()
	var castType *querier_dto.SQLType
	if p.current().kind == tokenCast {
		castType = p.consumeInlineCast()
	}
	p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, castType)
}

// parseValuesFirstRow parses the first row of a top-level VALUES statement, deriving one
// synthetic output column per element.
//
// Returns []querier_dto.RawOutputColumn which holds the columns named column1, column2,
// and so on, in element order.
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

// skipValuesTrailingRows walks the rows after the first in a top-level VALUES statement,
// registering parameters without emitting columns.
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

var (
	// insertSourceTerminators lists the keywords that end an INSERT source expression at
	// depth zero.
	insertSourceTerminators = map[string]struct{}{
		keywordON: {}, keywordRETURNING: {},
	}

	// onConflictTargetPredicateTerminators lists the keywords that end the optional WHERE
	// index-predicate of an ON CONFLICT target at depth zero. The predicate stops before the
	// DO action; RETURNING guards against a malformed clause that omits the action entirely.
	onConflictTargetPredicateTerminators = map[string]struct{}{
		"DO": {}, keywordRETURNING: {},
	}
)

// parseInsertSource skips the SELECT or query expression supplying an INSERT, stopping at
// ON CONFLICT or RETURNING.
func (p *parser) parseInsertSource() {
	p.skipTokensUntilTerminatorSet(insertSourceTerminators)
}

// parseOnConflict consumes an ON CONFLICT clause and analyses any DO UPDATE body's SET
// and WHERE expressions.
//
// Takes tableName (string) which is the target table for assignment column references.
func (p *parser) parseOnConflict(tableName string) {
	p.matchKeyword("CONFLICT")

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		p.skipConflictTargetPredicate()
	}

	if p.matchKeyword(keywordON) {
		p.matchKeyword(keywordCONSTRAINT)
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
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
				p.parseWhereClause()
			}
		}
	}
}

// skipConflictTargetPredicate consumes the optional WHERE index_predicate that follows a
// parenthesised ON CONFLICT target, used for partial-unique-index inference. The
// predicate ends at the DO action, leaving the conflict action and any trailing RETURNING
// clause for the caller to parse.
func (p *parser) skipConflictTargetPredicate() {
	if !p.matchKeyword(keywordWHERE) {
		return
	}
	p.skipTokensUntilTerminatorSet(onConflictTargetPredicateTerminators)
}

// parseSetClause walks an UPDATE SET list, dispatching to single- or multi-column
// assignment parsers.
//
// Takes tableName (string) which is the target table for assignment column references.
func (p *parser) parseSetClause(tableName string) {
	for {
		if p.current().kind == tokenLeftParen {
			p.parseMultiColumnSetClause(tableName)
		} else {
			p.parseSingleColumnSetClause(tableName)
		}

		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
}

// parseSingleColumnSetClause parses one "column = expression" assignment, binding
// parameter targets to the column when possible.
//
// Takes tableName (string) which is the target table.
func (p *parser) parseSingleColumnSetClause(tableName string) {
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
		var castType *querier_dto.SQLType
		if p.current().kind == tokenCast {
			castType = p.consumeInlineCast()
		}
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, castType)
	}

	p.skipSetExpressionWithColumn(columnRef)
}

// parseMultiColumnSetClause parses a "(c1, c2, ...) = (v1, v2, ...)" SET assignment.
//
// Takes tableName (string) which is the target table.
func (p *parser) parseMultiColumnSetClause(tableName string) {
	columnNames, _ := p.parseColumnList()

	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}

	if p.current().kind != tokenLeftParen {
		p.skipSetExpression()
		return
	}

	p.advance()
	p.parseMultiColumnSetValues(tableName, columnNames)
	if p.current().kind == tokenRightParen {
		p.advance()
	}
}

// parseMultiColumnSetValues walks the value list inside a multi-column SET assignment,
// binding parameter targets by position.
//
// Takes tableName (string) which is the target table.
// Takes columnNames ([]string) which is the LHS column list.
func (p *parser) parseMultiColumnSetValues(tableName string, columnNames []string) {
	columnIndex := 0
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if isParameterToken(p.current().kind) {
			parameterToken := p.current()
			columnRef := p.columnRefForIndex(tableName, columnNames, columnIndex)
			p.advance()
			p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextAssignment, columnRef, nil)
		} else {
			p.skipSetExpression()
		}
		if p.current().kind == tokenComma {
			p.advance()
			columnIndex++
		}
	}
}

var (
	// setExpressionTerminators lists keywords that end a SET expression at depth zero.
	setExpressionTerminators = map[string]struct{}{
		keywordWHERE: {}, keywordFROM: {}, keywordRETURNING: {},
		keywordORDER: {}, keywordLIMIT: {},
	}
)

// skipSetExpression advances past a single SET assignment expression up to the next comma
// or terminator keyword.
func (p *parser) skipSetExpression() {
	p.skipSetExpressionWithColumn(nil)
}

// skipSetExpressionWithColumn walks the RHS of a single SET assignment, stopping at a
// comma or a SET-clause terminator (WHERE, FROM, RETURNING, etc.) at depth 0.
//
// A parameter that is the operand of a comparison inside the expression (for example the
// `$1` in `SET col = CASE WHEN other >= $1 THEN ... END`) takes its type from the
// compared column, not the assignment target: the scanner remembers the most recent
// `<column> <comparison-operator>` pair and binds the following parameter to that column.
// Any other parameter (one that contributes to the assigned value, such as `COALESCE($1,
// col)`) keeps the assignment-target column reference so it still picks up that column's
// Go type; inline casts ($N::type) are honoured too. Without this the comparison operand
// was wrongly typed from the SET target, producing a Params field whose Go type did not
// match the value the caller binds. Pass nil for columnRef to keep the legacy no-context
// behaviour.
//
// Takes columnRef (*querier_dto.ColumnReference) which the parameters should be bound to
// as assignment values. nil leaves them with ParameterContextUnknown.
func (p *parser) skipSetExpressionWithColumn(columnRef *querier_dto.ColumnReference) {
	tableAlias := ""
	if columnRef != nil {
		tableAlias = columnRef.TableAlias
	}
	depth := 0
	var comparisonColumn *querier_dto.ColumnReference
	lastIdentifier := ""
	for !p.atEnd() {
		tok := p.current()

		nextDepth, stop, handled := p.handleExpressionParen(tok, depth)
		if stop {
			break
		}
		if handled {
			depth = nextDepth
			continue
		}

		if depth == 0 && p.isSetExpressionTerminator(tok) {
			break
		}
		if isParameterToken(tok.kind) {
			p.registerSetExpressionParameter(tok, columnRef, comparisonColumn)
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
// in "WHEN attempt >= $1" is typed from attempt. Any other token clears the pending
// state.
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

// registerSetExpressionParameter consumes a parameter token within a SET assignment RHS
// and records it with the column reference and context that match its position.
//
// When comparisonColumn is non-nil the parameter is an operand of a comparison (for
// example `WHEN attempt >= $1`), so it is recorded against that column with comparison
// context. Otherwise it contributes to the assigned value and is recorded against the
// SET-target columnRef as assignment context (or unknown context when no target is
// known). In either case an inline cast ($N::type) that follows is honoured.
//
// Takes parameterToken (token) which is the placeholder token to record.
// Takes columnRef (*querier_dto.ColumnReference) which links the parameter to its
// SET-target column when known.
// Takes comparisonColumn (*querier_dto.ColumnReference) which is the column the parameter
// is compared against, or nil when not a comparison operand.
func (p *parser) registerSetExpressionParameter(
	parameterToken token,
	columnRef *querier_dto.ColumnReference,
	comparisonColumn *querier_dto.ColumnReference,
) {
	p.advance()
	var castType *querier_dto.SQLType
	if p.current().kind == tokenCast {
		castType = p.consumeInlineCast()
	}
	if comparisonColumn != nil {
		p.registerParameterFromToken(parameterToken, querier_dto.ParameterContextComparison, comparisonColumn, castType)
		return
	}
	context := querier_dto.ParameterContextUnknown
	if columnRef != nil {
		context = querier_dto.ParameterContextAssignment
	}
	p.registerParameterFromToken(parameterToken, context, columnRef, castType)
}

// isSetExpressionTerminator reports whether tok ends one SET assignment expression.
//
// Takes tok (token) which is the candidate token.
//
// Returns bool which is true at a comma or one of the terminator keywords.
func (*parser) isSetExpressionTerminator(tok token) bool {
	if tok.kind == tokenComma {
		return true
	}
	if tok.kind == tokenIdentifier {
		_, ok := setExpressionTerminators[strings.ToUpper(tok.value)]
		return ok
	}
	return false
}
