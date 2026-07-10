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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// expressionScanContinue restarts the loop because the helper already advanced the
	// cursor (e.g. consumed `(`, parameter, or a join keyword that was actually a function
	// call).
	expressionScanContinue expressionScanAction = iota

	// expressionScanBreak stops the scan because a terminator was hit or the depth went
	// negative.
	expressionScanBreak

	// expressionScanAdvance steps over an inert token (an expression-interior identifier,
	// literal, operator, etc.).
	expressionScanAdvance
)

// expressionScanAction selects how skipTokensUntilTerminatorSet handles the current token
// after classifyExpressionToken has resolved its role. The classifier may already have
// advanced the parser, so the action distinguishes "stop", "continue without advancing",
// and "advance over a plain token".
type expressionScanAction int

var (
	// expressionTerminatorKeywords lists keywords that terminate a free-form SQL expression
	// such as WHERE, ON, and HAVING.
	//
	// JOIN-introducing keywords are included so that a JOIN's ON expression stops at the
	// next JOIN rather than swallowing the subsequent JOIN clause (which would prevent the
	// next joined table from being added to the scope chain and leave its projected columns
	// typed as `any`).
	expressionTerminatorKeywords = map[string]struct{}{
		keywordGROUP: {}, keywordHAVING: {}, keywordORDER: {}, keywordLIMIT: {},
		keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {}, "WINDOW": {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, keywordSET: {}, keywordON: {},
		keywordFROM: {}, keywordWHERE: {}, "INTO": {},
		keywordJOIN: {}, "INNER": {}, "LEFT": {}, "RIGHT": {},
		"FULL": {}, "CROSS": {}, "NATURAL": {},
	}

	// onConflictTargetPredicateTerminators lists the keywords that end the optional WHERE
	// index-predicate of an ON CONFLICT target at depth zero. The predicate stops before the
	// DO action; RETURNING guards against a malformed clause that omits the action entirely.
	onConflictTargetPredicateTerminators = map[string]struct{}{
		"DO":             {},
		keywordRETURNING: {},
	}

	// joinKeywordTerminators is the subset of expressionTerminatorKeywords whose tokens are
	// also valid SQL function names such as LEFT() and RIGHT().
	//
	// When such a token is immediately followed by an opening parenthesis it is a function
	// call rather than a JOIN starter, so the terminator check must skip it to avoid
	// prematurely ending the surrounding expression and dropping any parameters that follow.
	joinKeywordTerminators = map[string]struct{}{
		"INNER": {}, "LEFT": {}, "RIGHT": {}, "FULL": {}, "CROSS": {}, "NATURAL": {},
	}
)

// parseWhereClause consumes the body of a WHERE clause, tracking parameters until the
// next major clause keyword.
func (p *parser) parseWhereClause() {
	p.parseExpressionUntilTerminator()
}

// parseExpressionUntilTerminator scans a free-form expression up to the next top-level
// terminator keyword, registering any parameters seen along the way.
func (p *parser) parseExpressionUntilTerminator() {
	p.skipTokensUntilTerminatorSet(expressionTerminatorKeywords)
}

// skipTokensUntilTerminatorSet walks the token stream through balanced parentheses until
// a top-level terminator keyword or unbalanced ')' appears, registering any parameters
// seen along the way.
//
// Takes terminators (map[string]struct{}) which is the set of keywords that end the scan.
func (p *parser) skipTokensUntilTerminatorSet(terminators map[string]struct{}) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		newDepth, action := p.classifyExpressionToken(tok, depth, terminators)
		depth = newDepth
		switch action {
		case expressionScanContinue:
			continue
		case expressionScanBreak:
			return
		case expressionScanAdvance:
			p.advance()
		}
	}
}

// classifyExpressionToken inspects the cursor token and decides whether it adjusts the
// paren depth, terminates the scan, or advances past inert content.
//
// Parameter tokens and `(` are consumed in place because handleParameterInExpression and
// the depth bump must happen here; the caller's switch reflects that by returning
// expressionScanContinue without invoking advance again.
//
// Takes tok (token) which is the current token.
// Takes depth (int) which is the current paren nesting depth.
// Takes terminators (map[string]struct{}) which is the keyword set that signals the end
// of the scan when met at depth zero.
//
// Returns int which is the updated paren depth.
// Returns expressionScanAction which tells the caller how to proceed.
func (p *parser) classifyExpressionToken(tok token, depth int, terminators map[string]struct{}) (int, expressionScanAction) {
	if tok.kind == tokenLeftParen {
		if p.isSubqueryStart() {
			innerAnalysis, analyseError := p.analyseSubqueryBody()
			if analyseError != nil {
				log.Debug("postgres: predicate subquery analysis failed; parameters dropped",
					logger_domain.String("error", analyseError.Error()),
				)
			} else {
				p.predicateSubqueries = append(p.predicateSubqueries, innerAnalysis)
			}
			return depth, expressionScanContinue
		}
		p.advance()
		return depth + 1, expressionScanContinue
	}
	if tok.kind == tokenRightParen {
		if depth == 0 {
			return depth, expressionScanBreak
		}
		p.advance()
		return depth - 1, expressionScanContinue
	}
	if depth == 0 && p.isKeywordTerminator(tok, terminators) {
		if p.isJoinKeywordFunctionCall(tok) {
			p.advance()
			return depth, expressionScanContinue
		}
		return depth, expressionScanBreak
	}
	if isParameterToken(tok.kind) {
		p.handleParameterInExpression()
		return depth, expressionScanContinue
	}
	return depth, expressionScanAdvance
}

// isJoinKeywordFunctionCall reports whether a JOIN-like terminator keyword is actually
// being used as a function name such as LEFT(...) or RIGHT(...).
//
// The check is the immediate `(` follower, since SQL only allows those identifiers to
// introduce a JOIN or a function call.
//
// Takes tok (token) which is the candidate identifier.
//
// Returns bool which is true when the keyword is followed by `(`.
func (p *parser) isJoinKeywordFunctionCall(tok token) bool {
	_, isJoinKeyword := joinKeywordTerminators[strings.ToUpper(tok.value)]
	if !isJoinKeyword {
		return false
	}
	return p.peek().kind == tokenLeftParen
}

// isKeywordTerminator reports whether tok is an identifier whose upper-cased value is a
// member of the supplied terminator keyword set.
//
// Takes tok (token) which is the token to test.
// Takes terminators (map[string]struct{}) which is the terminator keyword set.
//
// Returns bool which is true when tok is a terminator keyword.
func (*parser) isKeywordTerminator(tok token, terminators map[string]struct{}) bool {
	if tok.kind != tokenIdentifier {
		return false
	}
	_, ok := terminators[strings.ToUpper(tok.value)]
	return ok
}

// parseGroupByClause parses the comma-separated list of GROUP BY columns at the current
// position.
//
// Returns []querier_dto.ColumnReference which is the list of grouped column references.
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

// parseGroupByColumn parses a single GROUP BY column reference, which may be a bare
// identifier or an alias-qualified column.
//
// Returns querier_dto.ColumnReference which is the parsed column.
// Returns bool which is true when the current token started a column.
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
	// orderByTerminators lists the keywords that end an ORDER BY scan at the top level.
	orderByTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {},
		keywordRETURNING: {}, "WINDOW": {},
	}
)

// parseOrderByList scans the ORDER BY tail through balanced parens until a terminator
// keyword appears.
func (p *parser) parseOrderByList() {
	p.skipTokensUntilTerminatorSet(orderByTerminators)
}

var (
	// limitOffsetValueTerminators lists the keywords that end a LIMIT/OFFSET/FETCH value
	// expression at the top level.
	//
	// Scanning a compound value (a function call or parenthesised expression) stops at the
	// keyword introducing the next clause so the caller can resume.
	limitOffsetValueTerminators = map[string]struct{}{
		keywordLIMIT: {}, keywordOFFSET: {}, keywordFETCH: {}, keywordFOR: {},
		keywordUNION: {}, keywordINTERSECT: {}, keywordEXCEPT: {}, keywordRETURNING: {},
		keywordROW: {}, keywordROWS: {}, keywordWITH: {}, "ONLY": {}, "TIES": {},
		"NEXT": {}, "FIRST": {},
	}
)

// consumeParameterOrAdvance consumes a LIMIT/OFFSET value.
//
// When the value is a bare placeholder it is registered with the given context
// (preserving the integer typing and the limit/offset name that context drives).
// Otherwise the value is a compound expression such as COALESCE($1, 100) or a
// parenthesised subexpression, which is scanned in full so any nested placeholder is
// still registered rather than dropped.
//
// Takes context (querier_dto.ParameterContext) which classifies a bare-placeholder
// value's syntactic role.
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

// retagLimitOffsetParameters re-tags every parameter registered while scanning a LIMIT or
// OFFSET value with the limit/offset context.
//
// The placeholder in LIMIT COALESCE($n, 100) is one such parameter; re-tagging overrides
// any incidental function-argument context the inner scan assigned. A placeholder in a
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

// parseLimitOffset parses any combination of LIMIT, OFFSET, and FETCH clauses that appear
// at the end of a SELECT.
func (p *parser) parseLimitOffset() {
	p.parseLeadingOffset()

	if p.matchKeyword(keywordLIMIT) {
		p.parseLimitClause()
		return
	}

	p.parseFetchClause()
	p.parseTrailingOffset()
}

// parseLeadingOffset consumes an OFFSET clause that appears before any LIMIT or FETCH.
func (p *parser) parseLeadingOffset() {
	if !p.isKeyword(keywordOFFSET) {
		return
	}
	p.advance()
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseLimitClause consumes the body of a LIMIT clause along with any following OFFSET or
// comma-separated offset value.
func (p *parser) parseLimitClause() {
	if !p.matchKeyword(keywordALL) {
		p.consumeParameterOrAdvance(querier_dto.ParameterContextLimit)
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

// parseFetchClause consumes a FETCH FIRST/NEXT ... ROWS ONLY/WITH TIES clause and tags
// any parameter as a limit parameter.
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
		referenceCountBefore := len(p.parameterRefs)
		p.skipTokensUntilTerminatorSet(limitOffsetValueTerminators)
		p.retagLimitOffsetParameters(referenceCountBefore, querier_dto.ParameterContextLimit)
	}

	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
	p.matchKeyword("ONLY")
	p.matchKeyword(keywordWITH)
	p.matchKeyword("TIES")
}

// parseTrailingOffset consumes an OFFSET clause that follows a FETCH clause.
func (p *parser) parseTrailingOffset() {
	if p.isKeyword(keywordLIMIT) || !p.matchKeyword(keywordOFFSET) {
		return
	}
	p.consumeParameterOrAdvance(querier_dto.ParameterContextOffset)
	p.matchKeyword(keywordROW)
	p.matchKeyword(keywordROWS)
}

// parseCompoundQuery consumes a compound query operator (UNION, INTERSECT, EXCEPT) and
// any ALL modifier.
//
// Returns querier_dto.CompoundOperator which is the matched operator, or 0 when no
// compound keyword follows.
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

// skipForUpdateClause consumes a FOR UPDATE / SHARE / KEY SHARE locking clause along with
// any OF list and NOWAIT / SKIP LOCKED modifiers.
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

// parseValuesClause walks one or more parenthesised VALUES rows, associating each
// parameter with its target column.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which is the target column list.
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

// columnRefForIndex builds a column reference for the column at index in columnNames
// within tableName.
//
// Takes tableName (string) which is the column's table alias.
// Takes columnNames ([]string) which is the column list.
// Takes index (int) which is the desired column index.
//
// Returns *querier_dto.ColumnReference which is the column reference, or nil when index
// is out of range.
func (*parser) columnRefForIndex(tableName string, columnNames []string, index int) *querier_dto.ColumnReference {
	if index >= len(columnNames) {
		return nil
	}
	return &querier_dto.ColumnReference{
		TableAlias: tableName,
		ColumnName: columnNames[index],
	}
}

// parseValuesRow walks the elements of a single VALUES row, advancing the column index
// for each comma.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which is the target column list.
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

// parseValuesRowElement consumes a single VALUES row element, registering a parameter
// token or skipping a parenthesised expression.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which is the target column list.
// Takes columnIndex (int) which is the current row column index.
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
// explicit cast type when one follows the parameter such as $1::int.
//
// Non-parameter tokens are advanced over so the cursor lands just past the matching
// closing paren. This is used by parseValuesRowElement to keep INSERT parameters inside
// function calls like COALESCE($1::int, default) properly typed; without it those
// parameters would be dropped and the emitter would default them to *any in the generated
// params struct.
//
// Takes columnRef (*querier_dto.ColumnReference) which is the INSERT column the
// parenthesised expression is the value for. Every parameter inside the expression is
// bound to it.
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
			p.handleInsertValueExpressionToken(tok, columnRef)
		}
	}
}

// handleInsertValueExpressionToken processes a non-paren token inside
// scanInsertValueExpression.
//
// Parameter tokens are bound to columnRef and tagged with any inline `::type` cast that
// follows; everything else is stepped over. Extracted so the surrounding scan keeps its
// nesting shallow.
//
// Takes tok (token) which is the current token under inspection.
// Takes columnRef (*querier_dto.ColumnReference) which is the INSERT column the
// surrounding parenthesised expression is the value for; every parameter is bound to it.
func (p *parser) handleInsertValueExpressionToken(tok token, columnRef *querier_dto.ColumnReference) {
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

// parseValuesFirstRow parses the first row of a standalone VALUES query, deriving a
// synthetic column name for each element so the projection has named output columns.
//
// Returns []querier_dto.RawOutputColumn which holds one entry per element of the first
// row.
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

// skipValuesTrailingRows consumes any rows after the first in a standalone VALUES query
// while still registering parameter tokens.
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
	// insertSourceTerminators lists the keywords that end the source expression of an INSERT
	// statement.
	insertSourceTerminators = map[string]struct{}{
		keywordON: {}, keywordRETURNING: {},
	}
)

// parseInsertSource skips tokens of an INSERT's source query until an ON CONFLICT or
// RETURNING clause appears.
func (p *parser) parseInsertSource() {
	p.skipTokensUntilTerminatorSet(insertSourceTerminators)
}

// parseOnConflict consumes an ON CONFLICT clause including its target, DO NOTHING or DO
// UPDATE action, and any nested SET / WHERE clause.
//
// Takes tableName (string) which is the target table name.
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
// parenthesised ON CONFLICT target, which Postgres permits for partial-unique-index
// inference.
func (p *parser) skipConflictTargetPredicate() {
	if !p.matchKeyword(keywordWHERE) {
		return
	}
	p.skipTokensUntilTerminatorSet(onConflictTargetPredicateTerminators)
}

// parseSetClause walks a SET clause of an UPDATE, dispatching between single-column and
// multi-column assignment forms.
//
// Takes tableName (string) which is the target table name.
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

// parseSingleColumnSetClause parses a `column = value` assignment, tagging any parameter
// on the right-hand side with its target column.
//
// Takes tableName (string) which is the target table name.
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

// parseMultiColumnSetClause parses an `(col1, col2) = (val1, val2)` assignment, tagging
// each parameter with its target column.
//
// Takes tableName (string) which is the target table name.
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

// parseMultiColumnSetValues parses the value list inside a `(col1, col2) = (val1, val2)`
// SET assignment, registering each parameter with its target column.
//
// Takes tableName (string) which is the target table name.
// Takes columnNames ([]string) which is the assigned column list.
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
	// setExpressionTerminators lists the keywords that end a SET clause's right-hand-side
	// expression at the top level.
	setExpressionTerminators = map[string]struct{}{
		keywordWHERE: {}, keywordFROM: {}, keywordRETURNING: {},
		keywordORDER: {}, keywordLIMIT: {},
	}
)

// skipSetExpression walks the right-hand side of a SET assignment through balanced parens
// until a comma or terminator keyword appears, registering any parameters seen along the
// way.
func (p *parser) skipSetExpression() {
	p.skipSetExpressionWithColumn(nil)
}

// skipSetExpressionWithColumn walks the RHS of a single SET assignment, stopping at a
// comma or a SET-clause terminator such as WHERE, FROM, or RETURNING at depth 0.
//
// Parameters encountered inside the RHS are registered with the supplied columnRef as
// assignment context so they pick up the column's Go type; inline casts (`$N::type`) are
// honoured too. Passing nil for columnRef preserves the no-context behaviour for sites
// that bind parameters without a column-aware path.
//
// A parameter that is the operand of a comparison inside the expression (for example the
// `$1` in `SET col = CASE WHEN other >= $1 THEN ... END`) takes its type from the
// compared column, not the assignment target: the scanner remembers the most recent
// `<column> <comparison-operator>` pair and binds the following parameter to that column
// with comparison context. Any other parameter (one that contributes to the assigned
// value, such as `COALESCE($1, col)`) keeps the assignment-target column reference so it
// still picks up that column's Go type. Without this the comparison operand was wrongly
// typed from the SET target, producing a Params field whose Go type did not match the
// value the caller binds.
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

		newDepth, stop, handled := p.handleSetExpressionParen(tok, depth)
		if stop {
			return
		}
		if handled {
			depth = newDepth
			continue
		}
		if depth == 0 && p.isSetExpressionTerminator(tok) {
			return
		}

		if isParameterToken(tok.kind) {
			p.bindSetExpressionParameter(tok, columnRef, comparisonColumn)
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

// handleSetExpressionParen processes a parenthesis token within a SET expression scan.
//
// It reports the next depth value, whether to stop the outer loop (unmatched right paren
// at depth zero), and whether the token was a parenthesis at all. The comparison state is
// left untouched so a `(` between a comparison operator and its parameter does not break
// the pairing.
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

// bindSetExpressionParameter records the parameter at the cursor with the column
// reference and context that match its position in the SET expression.
//
// When comparisonColumn is non-nil the parameter is an operand of a comparison (for
// example `WHEN attempt >= $1`), so it is recorded against that column with comparison
// context. Otherwise it contributes to the assigned value and is recorded against the
// SET-target columnRef with assignment context (or unknown context when no target is
// known). An inline `::type` cast immediately after the parameter is recorded as the
// explicit cast type in both cases.
//
// Takes parameterToken (token) which is the parameter being recorded.
// Takes columnRef (*querier_dto.ColumnReference) which is the SET target column, or nil
// for legacy contexts.
// Takes comparisonColumn (*querier_dto.ColumnReference) which is the column the parameter
// is compared against, or nil when it is not a comparison operand.
func (p *parser) bindSetExpressionParameter(
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

// isSetExpressionTerminator reports whether tok ends a SET clause's right-hand
// expression.
//
// Takes tok (token) which is the token to test.
//
// Returns bool which is true for commas and SET terminator keywords.
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
