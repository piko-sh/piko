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

package db_engine_clickhouse

import (
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// kwWhere is the WHERE clause keyword used as a stop token.
	kwWhere = "WHERE"

	// kwGroup is the GROUP clause keyword used as a stop token.
	kwGroup = "GROUP"

	// kwHaving is the HAVING clause keyword used as a stop token.
	kwHaving = "HAVING"

	// kwOrder is the ORDER clause keyword used as a stop token.
	kwOrder = "ORDER"

	// kwLimit is the LIMIT clause keyword used as a stop token.
	kwLimit = "LIMIT"

	// kwOffset is the OFFSET clause keyword used as a stop token.
	kwOffset = "OFFSET"

	// kwSettings is the SETTINGS clause keyword used as a stop token.
	kwSettings = "SETTINGS"

	// kwFormat is the FORMAT clause keyword used as a stop token.
	kwFormat = "FORMAT"

	// kwUnion is the UNION compound-operator keyword used as a stop token.
	kwUnion = "UNION"

	// kwIntersect is the INTERSECT compound-operator keyword used as a stop token.
	kwIntersect = "INTERSECT"

	// kwExcept is the EXCEPT compound-operator keyword used as a stop token.
	kwExcept = "EXCEPT"

	// kwWith is the WITH keyword introducing a CTE list or modifier.
	kwWith = "WITH"

	// kwWindow is the WINDOW clause keyword used as a stop token.
	kwWindow = "WINDOW"

	// kwAs is the AS alias keyword.
	kwAs = "AS"

	// kwBy is the BY keyword following GROUP, ORDER, or LIMIT.
	kwBy = "BY"

	// kwFrom is the FROM clause keyword used as a stop token.
	kwFrom = "FROM"

	// kwOn is the ON join-predicate keyword.
	kwOn = "ON"

	// kwFinal is the FINAL table-modifier keyword.
	kwFinal = "FINAL"

	// kwPrewhere is the PREWHERE clause keyword used as a stop token.
	kwPrewhere = "PREWHERE"

	// kwJoin is the JOIN keyword used as a stop token.
	kwJoin = "JOIN"

	// kwInner is the INNER join-kind keyword.
	kwInner = "INNER"

	// kwLeft is the LEFT join-kind keyword.
	kwLeft = "LEFT"

	// kwRight is the RIGHT join-kind keyword.
	kwRight = "RIGHT"

	// kwFull is the FULL join-kind keyword.
	kwFull = "FULL"

	// kwCross is the CROSS join-kind keyword.
	kwCross = "CROSS"

	// kwAsof is the ASOF join-strictness keyword.
	kwAsof = "ASOF"

	// kwSemi is the SEMI join-strictness keyword.
	kwSemi = "SEMI"

	// kwAnti is the ANTI join-strictness keyword.
	kwAnti = "ANTI"

	// kwArray is the ARRAY keyword introducing an ARRAY JOIN.
	kwArray = "ARRAY"

	// kwMaterialized is the MATERIALIZED CTE qualifier keyword.
	kwMaterialized = "MATERIALIZED"

	// kwNot is the NOT keyword, also used in NOT MATERIALIZED.
	kwNot = "NOT"

	// kwTo is the TO keyword used in LIMIT WITH FILL TO.
	kwTo = "TO"

	// kwStep is the STEP keyword used in LIMIT WITH FILL STEP.
	kwStep = "STEP"

	// kwFill is the FILL keyword used in WITH FILL.
	kwFill = "FILL"

	// kwNulls is the NULLS keyword used in ORDER BY NULLS FIRST/LAST.
	kwNulls = "NULLS"

	// kwAsc is the ASC sort-direction keyword.
	kwAsc = "ASC"

	// kwDesc is the DESC sort-direction keyword.
	kwDesc = "DESC"

	// kwFirst is the FIRST keyword used in NULLS FIRST.
	kwFirst = "FIRST"

	// kwLast is the LAST keyword used in NULLS LAST.
	kwLast = "LAST"

	// kwAny is the ANY join-strictness keyword.
	kwAny = "ANY"

	// kwAll is the ALL keyword used in UNION ALL and GROUP BY ALL.
	kwAll = "ALL"

	// kwGlobal is the GLOBAL join-distribution keyword.
	kwGlobal = "GLOBAL"

	// kwQualify is the QUALIFY clause keyword used as a stop token.
	kwQualify = "QUALIFY"

	// kwGrouping is the GROUPING keyword introducing GROUPING SETS.
	kwGrouping = "GROUPING"

	// kwSets is the SETS keyword used in GROUPING SETS.
	kwSets = "SETS"

	// kwTies is the TIES keyword used in LIMIT WITH TIES.
	kwTies = "TIES"

	// engineKeyGroupByAll captures `GROUP BY ALL` as a single boolean on the analysis's
	// EngineSpecific map. ClickHouse expands GROUP BY ALL to the implicit non-aggregate
	// column list at execution time; downstream consumers need only know the modifier was
	// present.
	engineKeyGroupByAll = "GROUP_BY_ALL"

	// engineKeyGroupingSets captures the parenthesised tuple list of a `GROUP BY GROUPING
	// SETS ((a, b), (c))` clause as flat text so downstream consumers can re-emit it
	// verbatim or parse it further.
	engineKeyGroupingSets = "GROUPING_SETS"

	// engineKeyQualify captures the QUALIFY clause expression text so downstream consumers
	// know it was present and can decide whether to attempt rewriting the predicate.
	engineKeyQualify = "QUALIFY"

	// engineKeyCTEMaterialized is the EngineSpecific key used to record the trailing
	// MATERIALIZED / NOT MATERIALIZED CTE qualifier.
	engineKeyCTEMaterialized = "CTE_MATERIALIZED"

	// engineKeyLimitFillFrom captures the textual `FROM <expr>` body of a `LIMIT ... WITH
	// FILL FROM ...` clause.
	engineKeyLimitFillFrom = "LIMIT_FILL_FROM"

	// engineKeyLimitFillTo captures the textual `TO <expr>` body of a `LIMIT ... WITH FILL
	// TO ...` clause.
	engineKeyLimitFillTo = "LIMIT_FILL_TO"

	// engineKeyLimitFillStep captures the textual `STEP <expr>` body of a `LIMIT ... WITH
	// FILL STEP ...` clause.
	engineKeyLimitFillStep = "LIMIT_FILL_STEP"
)

var (
	// stopAfterPrewhere terminates a PREWHERE expression scan at the clauses that may
	// legally follow PREWHERE.
	stopAfterPrewhere = []string{kwWhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept, kwQualify}

	// stopAfterHaving terminates a HAVING expression scan at the clauses that may legally
	// follow HAVING.
	stopAfterHaving = []string{kwOrder, kwLimit, kwSettings, kwFormat, kwWindow, kwUnion, kwIntersect, kwExcept, kwQualify}

	// stopAfterWindow terminates a WINDOW expression scan at the clauses that may legally
	// follow WINDOW.
	stopAfterWindow = []string{kwOrder, kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept, kwQualify}

	// stopAfterOrderBy terminates an ORDER BY expression scan at the clauses that may
	// legally follow ORDER BY.
	stopAfterOrderBy = []string{kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept}

	// stopAfterOrderByColumn terminates the per-column expression scan inside an ORDER BY
	// list.
	//
	// It includes the column-modifier keywords ASC, DESC, NULLS, WITH, FILL, FROM, TO, and
	// STEP so each modifier attaches to the column whose expression it follows.
	stopAfterOrderByColumn = []string{
		kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept,
		kwAsc, kwDesc, kwNulls, kwWith, kwFill, kwFrom, kwTo, kwStep,
	}

	// stopArrayJoinSource terminates the textual capture of an ARRAY JOIN expression source.
	// The cursor stops on the optional AS alias keyword, the surrounding clause keywords
	// that signal the end of the FROM tail, and the join-chain keywords that may follow the
	// entry without an intervening comma.
	stopArrayJoinSource = []string{
		kwAs,
		kwWhere, kwPrewhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat,
		kwUnion, kwIntersect, kwExcept, kwJoin, kwInner, kwLeft, kwRight, kwFull, kwCross,
		kwArray, kwAsof, kwSemi, kwAnti, kwOn,
	}

	// stopAfterSettings terminates a SETTINGS expression scan at the clauses that may
	// legally follow SETTINGS.
	stopAfterSettings = []string{kwFormat, kwUnion, kwIntersect, kwExcept}

	// stopFromClause is the stop-keyword set used while reading the FROM clause's expression
	// bodies (SAMPLE and ON predicates).
	//
	// It contains every clause that can legally follow FROM together with the join-related
	// keywords that mark the start of an embedded join chain. ANY, ALL, and GLOBAL are
	// included because they are valid ClickHouse strictness and distribution prefixes that
	// introduce the next join entry.
	stopFromClause = []string{
		kwWhere, kwPrewhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat,
		kwUnion, kwIntersect, kwExcept, kwJoin, kwInner, kwLeft, kwRight, kwFull, kwCross,
		kwArray, kwAsof, kwSemi, kwAnti, kwAny, kwAll, kwGlobal,
	}

	// stopJoinOnClause is the stop-keyword set used while reading the ON predicate of a
	// JOIN.
	//
	// It is identical to stopFromClause except that ASOF, SEMI, and ANTI come before ARRAY
	// because the join chain ordering is different here.
	stopJoinOnClause = []string{
		kwWhere, kwPrewhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat,
		kwUnion, kwIntersect, kwExcept, kwJoin, kwInner, kwLeft, kwRight, kwFull, kwCross,
		kwAsof, kwSemi, kwAnti, kwArray, kwAny, kwAll, kwGlobal,
	}

	// stopGroupByClause is the stop-keyword set for the GROUP BY expression body.
	//
	// It includes WINDOW because GROUP BY can precede WINDOW in the SELECT grammar.
	stopGroupByClause = []string{kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept, kwWindow}

	// projectionStopKeywords is the stop-keyword set that terminates a single projection
	// item: the trailing alias keyword and every clause that can follow the SELECT list.
	projectionStopKeywords = []string{kwAs, kwFrom, kwWhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept}

	// infixOperatorKeywords lists the SQL keywords that act as infix operators following an
	// expression.
	//
	// cursorLooksLikeOperator uses it to disambiguate a direct column reference (`col`) from
	// the left side of a binary form (`col AND ...`, `col IN (...)`). It is declared at
	// package level so the lookup is a single map fetch rather than a switch on each call.
	infixOperatorKeywords = map[string]bool{
		"AND":     true,
		"OR":      true,
		"IN":      true,
		"BETWEEN": true,
		"LIKE":    true,
		"ILIKE":   true,
		"IS":      true,
		"NOT":     true,
	}
)

// analyseSelect parses a SELECT statement into a RawQueryAnalysis.
//
// It handles every ClickHouse SELECT shape the engine accepts: WITH CTE clauses (regular
// and recursive), the SELECT projection list with optional DISTINCT and DISTINCT ON, FROM
// tables, subqueries, and table functions with optional FINAL and SAMPLE, INNER, LEFT,
// RIGHT, FULL, CROSS, ASOF, SEMI, and ANTI joins, ARRAY JOIN and LEFT ARRAY JOIN,
// PREWHERE and WHERE, GROUP BY with optional WITH ROLLUP, WITH CUBE, or WITH TOTALS,
// HAVING, WINDOW, ORDER BY with optional ASC, DESC, NULLS FIRST, NULLS LAST, and WITH
// FILL, LIMIT in both `LIMIT n` and `LIMIT m, n` forms with optional BY and OFFSET,
// SETTINGS assignments, UNION, INTERSECT, and EXCEPT compound branches, and a FORMAT
// specifier.
//
// The parser is bounded by maxParseDepth so adversarial inputs that nest subqueries
// unboundedly do not blow the stack.
//
// Returns *querier_dto.RawQueryAnalysis which is the parsed analysis.
// Returns error when the statement is malformed or a parameter type tag is bad.
func (p *parser) analyseSelect() (*querier_dto.RawQueryAnalysis, error) {
	if p.analysisDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.analysisDepth++
	defer func() { p.analysisDepth-- }()

	analysis := &querier_dto.RawQueryAnalysis{ReadOnly: true}

	if err := p.analyseSelectHeader(analysis); err != nil {
		return nil, err
	}
	if err := p.analyseSelectFilters(analysis); err != nil {
		return nil, err
	}
	if err := p.analyseSelectGroupings(analysis); err != nil {
		return nil, err
	}
	if err := p.analyseSelectTail(analysis); err != nil {
		return nil, err
	}

	if p.analysisDepth == 1 && p.firstParameterTypeError != nil {
		return analysis, p.firstParameterTypeError
	}
	return analysis, nil
}

// analyseSelectHeader parses the WITH, SELECT, DISTINCT, projection, and FROM portions of
// a SELECT statement.
//
// It is split out so the cognitive complexity of analyseSelect stays inside the linter
// budget; each helper handles a distinct phase of the SELECT grammar.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the header is malformed.
func (p *parser) analyseSelectHeader(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(kwWith) {
		if err := p.parseCTEList(analysis); err != nil {
			return err
		}
	}
	if !p.matchKeyword("SELECT") {
		return fmt.Errorf("expected SELECT at position %d", p.current().position)
	}

	if p.matchKeyword("DISTINCT") {
		if p.matchKeyword(kwOn) {
			if p.current().kind == tokenLeftParen {
				_ = p.skipParenthesised()
			}
		}
	}
	if err := p.parseProjectionList(analysis); err != nil {
		return err
	}
	if p.matchKeyword(kwFrom) {
		if err := p.parseFromClause(analysis); err != nil {
			return err
		}
	}
	return nil
}

// analyseSelectFilters parses the PREWHERE and WHERE portion of a SELECT.
//
// PREWHERE is treated as a WHERE for count-rewrite purposes; it sets HasWhereClause so
// the downstream rewriter takes the WHERE path even on PREWHERE-only queries.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a filter expression is malformed.
func (p *parser) analyseSelectFilters(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(kwPrewhere) {
		analysis.HasWhereClause = true
		if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopAfterPrewhere...); err != nil {
			return err
		}
	}
	if p.matchKeyword(kwWhere) {
		analysis.HasWhereClause = true
		if err := p.parseWhereClause(analysis); err != nil {
			return err
		}
	}
	return nil
}

// analyseSelectGroupings parses the GROUP BY, HAVING, and WINDOW clauses of a SELECT,
// including the optional WITH ROLLUP, WITH CUBE, and WITH TOTALS modifiers on GROUP BY.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a grouping clause is malformed.
func (p *parser) analyseSelectGroupings(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(kwGroup) {
		if err := p.parseGroupByBody(analysis); err != nil {
			return err
		}
	}
	if p.matchKeyword(kwHaving) {
		if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopAfterHaving...); err != nil {
			return err
		}
	}
	if p.matchKeyword(kwWindow) {
		if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopAfterWindow...); err != nil {
			return err
		}
	}
	return nil
}

// parseGroupByBody handles the body of a GROUP BY clause: the BY keyword check, the
// column list, and the optional WITH ROLLUP, WITH CUBE, or WITH TOTALS trailing modifier.
//
// ClickHouse extends the SQL grammar with two extra forms. `GROUP BY ALL` groups by every
// non-aggregate column in the projection list, and the presence of the modifier is
// recorded on the analysis's EngineSpecific map under engineKeyGroupByAll. `GROUP BY
// GROUPING SETS ((a, b), (c))` groups by every supplied tuple in turn, equivalent to
// UNION ALL across one GROUP BY per set, and the parenthesised tuple list body is
// captured under engineKeyGroupingSets.
//
// Both forms are detected by peeking the first keyword after BY; on match the regular
// column-list parser is bypassed because the grammar following ALL or GROUPING SETS does
// not contain a normal column reference list.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the GROUP BY body is malformed.
func (p *parser) parseGroupByBody(analysis *querier_dto.RawQueryAnalysis) error {
	if !p.matchKeyword(kwBy) {
		return fmt.Errorf("expected BY after GROUP at position %d", p.current().position)
	}
	switch {
	case p.matchKeyword(kwAll):
		if analysis.EngineSpecific == nil {
			analysis.EngineSpecific = map[string]string{}
		}
		analysis.EngineSpecific[engineKeyGroupByAll] = "true"
	case p.isKeyword(kwGrouping) && strings.EqualFold(p.peek().value, kwSets):
		if err := p.parseGroupingSetsBody(analysis); err != nil {
			return err
		}
	default:
		if err := p.parseGroupByColumns(analysis); err != nil {
			return err
		}
	}
	if p.matchKeyword(kwWith) {
		if !p.isAnyKeyword("ROLLUP", "CUBE", "TOTALS") {
			return fmt.Errorf("expected ROLLUP / CUBE / TOTALS after WITH at position %d", p.current().position)
		}
		p.advance()
	}
	return nil
}

// parseGroupingSetsBody consumes the body of a `GROUP BY GROUPING SETS (...)` clause.
//
// The leading GROUPING and SETS tokens have not been consumed; the helper advances past
// both, captures the parenthesised body as flat text, and records it on the analysis's
// EngineSpecific map under engineKeyGroupingSets so downstream consumers can re-emit or
// further parse the grouping list.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the GROUPING SETS body is malformed.
func (p *parser) parseGroupingSetsBody(analysis *querier_dto.RawQueryAnalysis) error {
	p.advance()
	p.advance()
	if p.current().kind != tokenLeftParen {
		return fmt.Errorf("expected '(' after GROUPING SETS at position %d", p.current().position)
	}
	body, err := p.captureParenthesisedBodyAsText()
	if err != nil {
		return err
	}
	if analysis.EngineSpecific == nil {
		analysis.EngineSpecific = map[string]string{}
	}
	analysis.EngineSpecific[engineKeyGroupingSets] = body
	return nil
}

// analyseSelectTail parses the optional QUALIFY clause, the ORDER BY, LIMIT, SETTINGS,
// and FORMAT clauses, plus the UNION, INTERSECT, and EXCEPT compound branches of a
// SELECT.
//
// QUALIFY filters rows after window-function evaluation; it mirrors the HAVING clause
// shape and is parsed with the same expression helper. The predicate text is captured
// under engineKeyQualify on the analysis's EngineSpecific map so downstream consumers
// know the modifier was present without needing to re-scan the SQL.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a tail clause is malformed.
func (p *parser) analyseSelectTail(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(kwQualify) {
		if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopAfterOrderBy...); err != nil {
			return err
		}
		if analysis.EngineSpecific == nil {
			analysis.EngineSpecific = map[string]string{}
		}
		analysis.EngineSpecific[engineKeyQualify] = "true"
	}
	if p.matchKeyword(kwOrder) {
		if !p.matchKeyword(kwBy) {
			return fmt.Errorf("expected BY after ORDER at position %d", p.current().position)
		}
		p.parseOrderByList(analysis)
	}
	if p.matchKeyword(kwLimit) {
		p.parseLimitClause(analysis)
	}
	if p.matchKeyword(kwSettings) {
		if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopAfterSettings...); err != nil {
			return err
		}
	}
	if p.matchKeyword(kwFormat) {
		p.advance()
	}
	return p.analyseCompoundBranches(analysis)
}

// analyseCompoundBranches handles the UNION, INTERSECT, and EXCEPT branches that may
// follow a SELECT.
//
// Each branch is parsed recursively through the same parser instance, so the
// analysisDepth counter established by the outer analyseSelect call continues to bound
// the recursion. The inner analyseSelect re-enters with the incremented depth so a SELECT
// tree of N stacked UNION branches still terminates within maxParseDepth recursive
// frames.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a compound branch is malformed.
func (p *parser) analyseCompoundBranches(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		op, matched := p.matchCompoundOperator()
		if !matched {
			return nil
		}
		branch, branchErr := p.analyseSelect()
		if branchErr != nil {
			return branchErr
		}
		analysis.CompoundBranches = append(analysis.CompoundBranches, querier_dto.RawCompoundBranch{
			Operator: op,
			Query:    branch,
		})
	}
}

// parseCTEList reads the body of a `WITH cte1 AS (query), cte2 AS (query)` clause and
// stores the parsed definitions on the analysis.
//
// ClickHouse also accepts `WITH expr AS alias` (scalar CTE) and the recursive variant
// `WITH RECURSIVE cte AS (...)`; both are handled by tolerantly consuming the body.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the WITH clause is malformed or input ends unexpectedly.
func (p *parser) parseCTEList(analysis *querier_dto.RawQueryAnalysis) error {
	isRecursive := p.matchKeyword("RECURSIVE")
	for {
		if p.atEnd() {
			return errUnexpectedEndOfWithInput
		}
		more, err := p.parseSingleCTE(analysis, isRecursive)
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
	}
}

// captureCTEMaterializedQualifier consumes an optional trailing MATERIALIZED or NOT
// MATERIALIZED qualifier on a CTE definition.
//
// The value is recorded under the engineKeyCTEMaterialized key on the supplied
// definition's EngineSpecific map. The NOT branch is peeked rather than matched directly
// because NOT is also a generic predicate keyword; the qualifier is only consumed when
// the next token is MATERIALIZED.
//
// Takes definition (*querier_dto.RawCTEDefinition) which is the CTE entry whose
// EngineSpecific map should be populated.
func (p *parser) captureCTEMaterializedQualifier(definition *querier_dto.RawCTEDefinition) {
	if p.matchKeyword(kwMaterialized) {
		definition.EngineSpecific = map[string]string{engineKeyCTEMaterialized: "true"}
		return
	}
	if p.isKeyword(kwNot) && strings.EqualFold(p.peek().value, kwMaterialized) {
		p.advance()
		p.advance()
		definition.EngineSpecific = map[string]string{engineKeyCTEMaterialized: "false"}
	}
}

// parseSingleCTE consumes one CTE entry from the WITH list. Returns more=true when the
// caller should keep iterating because a trailing comma promised another entry;
// more=false otherwise.
//
// Recognises the optional trailing `MATERIALIZED` and `NOT MATERIALIZED` qualifiers
// ClickHouse permits after the closing paren of the CTE body. When present the qualifier
// is captured under the engineKeyCTEMaterialized key on the definition's EngineSpecific
// map so downstream consumers can decide between in-place inlining and materialised
// execution.
//
// Takes analysis (*querier_dto.RawQueryAnalysis), the analysis to populate with the
// parsed CTE.
// Takes isRecursive (bool), the WITH RECURSIVE flag propagated from parseCTEList.
//
// Returns more (bool), true when another CTE follows.
// Returns error when the entry is malformed.
func (p *parser) parseSingleCTE(analysis *querier_dto.RawQueryAnalysis, isRecursive bool) (more bool, err error) {
	cteName, parseErr := p.parseIdentifierOrKeyword()
	if parseErr != nil {
		return false, parseErr
	}
	if !p.matchKeyword(kwAs) {
		if consumeErr := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, "SELECT", kwWith); consumeErr != nil {
			return false, consumeErr
		}
		if p.current().kind == tokenComma {
			p.advance()
			return true, nil
		}
		return false, nil
	}
	if p.current().kind != tokenLeftParen {
		return false, fmt.Errorf("expected '(' after AS in CTE at position %d", p.current().position)
	}
	body, bodyErr := p.collectParenthesised()
	if bodyErr != nil {
		return false, bodyErr
	}
	nested := newParser(body)
	nested.analysisDepth = p.analysisDepth
	nested.maxParseDepth = p.maxParseDepth
	nestedAnalysis, nestedErr := nested.analyseSelect()
	if nestedErr != nil {
		return false, fmt.Errorf("CTE %q: %w", cteName, nestedErr)
	}
	definition := querier_dto.RawCTEDefinition{
		Name:             cteName,
		OutputColumns:    nestedAnalysis.OutputColumns,
		FromTables:       nestedAnalysis.FromTables,
		JoinClauses:      nestedAnalysis.JoinClauses,
		CompoundBranches: nestedAnalysis.CompoundBranches,

		ParameterReferences: nestedAnalysis.ParameterReferences,
		IsRecursive:         isRecursive,
	}
	p.captureCTEMaterializedQualifier(&definition)
	p.mergeNestedParameterReferences(analysis, nested, nestedAnalysis)

	analysis.CTEDefinitions = append(analysis.CTEDefinitions, definition)
	if p.current().kind == tokenComma {
		p.advance()
		return true, nil
	}
	return false, nil
}

// parseProjectionList reads the SELECT projection list.
//
// Each expression becomes one RawOutputColumn; `*` and `t.*` produce star-expansion
// markers.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a projection item is malformed.
func (p *parser) parseProjectionList(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		column, err := p.parseProjectionColumn(analysis)
		if err != nil {
			return err
		}
		analysis.OutputColumns = append(analysis.OutputColumns, column)
		if p.current().kind != tokenComma {
			return nil
		}
		p.advance()
	}
}

// parseProjectionColumn reads one item of the SELECT list.
//
// The analysis is threaded through so a `{name:Type}` placeholder inside a projection
// expression (for example `id + {bonus:UInt32} AS total`) is registered against the
// analysis and contributes a typed argument to the generated method.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns querier_dto.RawOutputColumn which is the parsed projection column.
// Returns error when the projection item is malformed.
func (p *parser) parseProjectionColumn(analysis *querier_dto.RawQueryAnalysis) (querier_dto.RawOutputColumn, error) {
	if star, ok := p.tryParseStarProjection(); ok {
		return star, nil
	}

	saved := p.position
	tableAlias, columnName, isDirectColumn := p.tryParseDirectColumnReference()
	if isDirectColumn && !p.cursorLooksLikeOperator() {
		return p.finaliseDirectColumnProjection(tableAlias, columnName)
	}
	if isDirectColumn {
		p.position = saved
	}

	expression := p.tryProjectionExpressionTree()
	p.consumeExpressionUntilCommaOrKeyword(
		analysis, querier_dto.ParameterContextComparison,
		projectionStopKeywords...,
	)
	alias, _, err := p.parseOptionalAlias()
	if err != nil {
		return querier_dto.RawOutputColumn{}, err
	}
	return querier_dto.RawOutputColumn{Name: alias, Expression: expression}, nil
}

// tryParseStarProjection consumes a `*` or `table.*` star expansion when the cursor is
// positioned on one. Returns ok=false otherwise so the caller can fall through to the
// regular column / expression paths.
//
// Returns column (querier_dto.RawOutputColumn), the star projection with TableAlias
// populated for qualified stars.
// Returns ok (bool), true when a star projection was consumed.
func (p *parser) tryParseStarProjection() (column querier_dto.RawOutputColumn, ok bool) {
	if p.current().kind == tokenStar {
		p.advance()
		return querier_dto.RawOutputColumn{IsStar: true}, true
	}
	if p.current().kind == tokenIdentifier && p.peek().kind == tokenDot {
		identifier := p.current().value
		const qualifiedStarLookahead = 2
		if p.position+qualifiedStarLookahead < len(p.tokens) && p.tokens[p.position+qualifiedStarLookahead].kind == tokenStar {
			p.advance()
			p.advance()
			p.advance()
			return querier_dto.RawOutputColumn{IsStar: true, TableAlias: identifier}, true
		}
	}
	return querier_dto.RawOutputColumn{}, false
}

// finaliseDirectColumnProjection wraps the alias-handling that follows a successful
// direct column reference. The optional `[AS] alias` becomes the projection's display
// name when present; otherwise the column name itself is used.
//
// Takes tableAlias (string), the table qualifier captured from the reference (empty for
// bare-column references).
// Takes columnName (string), the column identifier.
//
// Returns the populated RawOutputColumn and any alias parse error.
func (p *parser) finaliseDirectColumnProjection(tableAlias string, columnName string) (querier_dto.RawOutputColumn, error) {
	alias, _, err := p.parseOptionalAlias()
	if err != nil {
		return querier_dto.RawOutputColumn{}, err
	}
	name := columnName
	if alias != "" {
		name = alias
	}
	return querier_dto.RawOutputColumn{
		Name:       name,
		TableAlias: tableAlias,
		ColumnName: columnName,
	}, nil
}

// consumeExpressionUntilCommaOrKeyword consumes tokens, honouring paren depth, until
// reaching a top-level comma or one of the named clause keywords.
//
// The projection-list reader uses it so each item's trailing alias is left for
// parseOptionalAlias to consume. Any `{name:Type}` placeholder encountered is registered
// against the analysis so projection-list parameters are retained.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
// Takes stopKeywords (...string) which are the clause keywords that halt the scan.
func (p *parser) consumeExpressionUntilCommaOrKeyword(analysis *querier_dto.RawQueryAnalysis, context querier_dto.ParameterContext, stopKeywords ...string) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tokenIsTopLevelStop(tok, stopKeywords) {
			return
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// tokenIsTopLevelStop reports whether tok terminates a depth-0 expression scan.
//
// A top-level comma and any identifier matching one of the supplied stop keywords both
// halt the scan.
//
// Takes tok (token) which is the token under inspection.
// Takes stopKeywords ([]string) which are the clause keywords that halt the scan.
//
// Returns bool which is true when the token terminates the scan.
func tokenIsTopLevelStop(tok token, stopKeywords []string) bool {
	if tok.kind == tokenComma {
		return true
	}
	if tok.kind != tokenIdentifier {
		return false
	}
	for _, stop := range stopKeywords {
		if strings.EqualFold(tok.value, stop) {
			return true
		}
	}
	return false
}

// advanceParenDepth updates the running paren depth for the supplied token.
//
// Takes tok (token) which is the token under inspection.
// Takes depth (int) which is the current paren depth before this token.
//
// Returns newDepth (int) which is the paren depth after this token.
// Returns halt (bool) which is true on an unbalanced close paren at depth 0.
func advanceParenDepth(tok token, depth int) (newDepth int, halt bool) {
	switch tok.kind {
	case tokenLeftParen:
		return depth + 1, false
	case tokenRightParen:
		if depth == 0 {
			return depth, true
		}
		return depth - 1, false
	default:
		return depth, false
	}
}

// tryParseDirectColumnReference checks whether the cursor is on a bare column reference
// (`col`, `t.col`) NOT followed by a paren (which would mean a function call). On success
// advances the cursor past the reference and returns table alias + column name; on miss
// returns false with the cursor unchanged.
//
// Returns tableAlias (string), the qualifier portion of a qualified reference, or "" for
// a bare-column form.
// Returns columnName (string), the column identifier.
// Returns matched (bool), true when a column reference was consumed.
func (p *parser) tryParseDirectColumnReference() (tableAlias string, columnName string, matched bool) {
	saved := p.position
	if p.current().kind != tokenIdentifier {
		return "", "", false
	}
	first := p.current().value
	p.advance()
	if p.current().kind == tokenLeftParen {
		p.position = saved
		return "", "", false
	}
	if p.current().kind != tokenDot {
		return "", first, true
	}
	p.advance()
	if p.current().kind != tokenIdentifier {
		p.position = saved
		return "", "", false
	}
	second := p.current().value
	p.advance()
	if p.current().kind == tokenLeftParen {
		p.position = saved
		return "", "", false
	}
	return first, second, true
}

// parseOptionalAlias reads an optional `[AS] alias` after a projection-list expression.
//
// The keyword rejection list covers every clause-starting word so downstream clause
// handlers such as WINDOW, OFFSET, and WITH see the keyword still on the stream rather
// than consumed as a table alias. Every keyword is referenced through the kw* constants
// declared at the top of this file so the alias rejection set shares a single source of
// truth with the stop-keyword slices used by the consume helpers.
//
// Returns string which is the alias, or "" when none is present.
// Returns bool which is true when the AS keyword was present.
// Returns error when the alias is malformed.
func (p *parser) parseOptionalAlias() (string, bool, error) {
	hadAs := p.matchKeyword(kwAs)
	if p.current().kind != tokenIdentifier {
		return "", hadAs, nil
	}

	if p.isAnyKeyword(
		kwFrom, kwWhere, kwGroup, kwHaving, kwOrder, kwLimit, kwOffset,
		kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept, kwJoin,
		kwInner, kwLeft, kwRight, kwFull, kwCross, kwOn, "USING",
		kwPrewhere, kwArray, kwFinal, "SAMPLE", kwAsof, kwSemi, kwAnti,
		kwWindow, kwWith, "RECURSIVE", kwBy, kwTies, kwFill,
		kwQualify, kwAny, kwAll, kwGlobal,
	) {
		return "", hadAs, nil
	}
	alias := p.current().value
	p.advance()
	return alias, hadAs, nil
}

// cursorLooksLikeOperator reports whether the current token would extend a preceding
// expression as an infix operator such as `+`, `=`, `IN`, or `AND`.
//
// parseProjectionColumn uses it to disambiguate `SELECT col, ...` (direct column) from
// `SELECT col + 1, ...` (computed expression). Without this check the direct-column
// branch consumes `col` and leaves the binary operator dangling, corrupting subsequent
// projection parsing. The keyword set is owned by infixOperatorKeywords at package level
// so the membership test is a single map fetch.
//
// Returns bool which is true when the current token continues an expression.
func (p *parser) cursorLooksLikeOperator() bool {
	tok := p.current()
	if tok.kind == tokenOperator {
		return true
	}
	if tok.kind == tokenCast {
		return true
	}
	if tok.kind == tokenLeftBracket {
		return true
	}

	if tok.kind == tokenDot {
		return true
	}
	if tok.kind == tokenIdentifier {
		return infixOperatorKeywords[strings.ToUpper(tok.value)]
	}
	return false
}

// parseArrayJoinList reads one or more comma-separated ARRAY JOIN entries of shape
// `<source> [AS <alias>]`.
//
// The source may be a bare column reference, a literal array (`[1, 2, 3]`), or an
// arbitrary array-producing expression (`arrayMap(f, src)`). The bare column form treats
// the source name as both the source and the alias because ClickHouse shadows the
// original Array column under the same name within the SELECT scope. Each entry is
// appended to analysis.ArrayJoinClauses so the domain analyser can register the element
// column under its alias.
//
// Bare column references populate SourceColumn. Expression sources, meaning anything not
// a simple identifier followed by AS, a comma, or a clause keyword, populate
// SourceExpression with the captured textual body. Parameter placeholders inside
// expression bodies are tracked so `{name:Type}` references inside the source still
// register against the analysis.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes isLeft (bool) which is the LEFT ARRAY JOIN flag for each entry.
//
// Returns error when an ARRAY JOIN entry is malformed.
func (p *parser) parseArrayJoinList(analysis *querier_dto.RawQueryAnalysis, isLeft bool) error {
	for {
		if p.atEnd() {
			return nil
		}
		clause, err := p.parseSingleArrayJoinEntry(analysis, isLeft)
		if err != nil {
			return err
		}
		analysis.ArrayJoinClauses = append(analysis.ArrayJoinClauses, clause)
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		return nil
	}
}

// parseSingleArrayJoinEntry parses one ARRAY JOIN source-and-alias entry. The source is
// captured either as a bare-column reference (populating SourceColumn) or as a textual
// expression body (populating SourceExpression) depending on which shape the cursor
// reveals.
//
// Takes analysis (*querier_dto.RawQueryAnalysis), the analysis to register any captured
// parameter references against.
// Takes isLeft (bool), the LEFT ARRAY JOIN flag propagated from parseFromArrayJoinChain.
//
// Returns the populated RawArrayJoinClause and any parse error.
func (p *parser) parseSingleArrayJoinEntry(
	analysis *querier_dto.RawQueryAnalysis, isLeft bool,
) (querier_dto.RawArrayJoinClause, error) {
	if column, alias, isBare := p.tryParseBareArrayJoinSource(); isBare {
		return querier_dto.RawArrayJoinClause{
			Alias:        alias,
			SourceColumn: column,
			IsLeft:       isLeft,
		}, nil
	}
	expression := p.captureExpressionTrackingParams(
		analysis, querier_dto.ParameterContextComparison, stopArrayJoinSource...,
	)
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return querier_dto.RawArrayJoinClause{}, fmt.Errorf("expected ARRAY JOIN source at position %d", p.current().position)
	}
	alias := expression
	if p.matchKeyword(kwAs) {
		explicit, aliasErr := p.parseIdentifierOrKeyword()
		if aliasErr != nil {
			return querier_dto.RawArrayJoinClause{}, aliasErr
		}
		alias = explicit
	}
	return querier_dto.RawArrayJoinClause{
		Alias:            alias,
		SourceExpression: expression,
		IsLeft:           isLeft,
	}, nil
}

// tryParseBareArrayJoinSource peeks at the cursor to decide whether the ARRAY JOIN source
// is a simple bare column reference with an optional AS alias.
//
// On a miss the cursor is left untouched so the caller can fall through to the
// expression-source path.
//
// Returns column (string) which is the bare column name.
// Returns alias (string) which is the column name when no AS alias appears, or the
// explicit alias otherwise.
// Returns matched (bool) which is true when a bare-column source was consumed.
func (p *parser) tryParseBareArrayJoinSource() (column string, alias string, matched bool) {
	if p.current().kind != tokenIdentifier {
		return "", "", false
	}
	if isTopClauseKeyword(p.current().value) {
		return "", "", false
	}
	saved := p.position
	candidate := p.current().value
	p.advance()

	if p.current().kind == tokenComma {
		return candidate, candidate, true
	}
	if p.matchKeyword(kwAs) {
		explicit, err := p.parseIdentifierOrKeyword()
		if err != nil {
			p.position = saved
			return "", "", false
		}
		return candidate, explicit, true
	}
	if p.current().kind == tokenIdentifier && isTopClauseKeyword(p.current().value) {
		return candidate, candidate, true
	}
	if p.atEnd() {
		return candidate, candidate, true
	}
	p.position = saved
	return "", "", false
}

// parseFromClause reads the FROM clause body, supporting tables, derived tables
// (subqueries), table-valued functions, and JOINs.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the FROM clause is malformed.
func (p *parser) parseFromClause(analysis *querier_dto.RawQueryAnalysis) error {
	if err := p.parseFromSource(analysis, querier_dto.JoinInner); err != nil {
		return err
	}
	if err := p.parseFromTrailingModifiers(analysis); err != nil {
		return err
	}
	if err := p.parseFromArrayJoinChain(analysis); err != nil {
		return err
	}
	return p.parseFromJoinChain(analysis)
}

// parseFromTrailingModifiers consumes the FINAL and SAMPLE clauses that may follow the
// first table reference.
//
// SAMPLE accepts an expression body which the analyser captures so ClickHouse
// `{name:Type}` placeholders inside the sample expression are reflected in the parameter
// list.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a SAMPLE expression is malformed.
func (p *parser) parseFromTrailingModifiers(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		if p.matchKeyword(kwFinal) {
			continue
		}
		if p.matchKeyword("SAMPLE") {
			if err := p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopFromClause...); err != nil {
				return err
			}
			continue
		}
		return nil
	}
}

// parseFromArrayJoinChain handles the ARRAY JOIN and LEFT ARRAY JOIN chain that may
// appear after the trailing modifiers.
//
// Each match runs through parseArrayJoinList to register the joined arrays. The cursor is
// saved before consuming LEFT and restored when the expected ARRAY does not follow.
// Without the save and restore the LEFT keyword can be eaten by matchKeyword and then the
// missing ARRAY check would return, leaving the parser positioned after a stray LEFT that
// the surrounding clause never sees.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when an ARRAY JOIN entry is malformed.
func (p *parser) parseFromArrayJoinChain(analysis *querier_dto.RawQueryAnalysis) error {
	for p.isAnyKeyword(kwArray, kwLeft) {
		savedPosition := p.position
		isLeft := p.matchKeyword(kwLeft)
		if !p.matchKeyword(kwArray) {
			p.position = savedPosition
			return nil
		}
		if !p.matchKeyword(kwJoin) {
			return fmt.Errorf("expected JOIN after ARRAY at position %d", p.current().position)
		}
		if err := p.parseArrayJoinList(analysis, isLeft); err != nil {
			return err
		}
	}
	return nil
}

// parseFromJoinChain processes the JOIN chain that may follow the initial table
// reference.
//
// Each JOIN reads a fresh source and its ON or USING qualifier; ON predicates run through
// the parameter tracker because ClickHouse parameters can appear in join conditions.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a JOIN clause is malformed.
func (p *parser) parseFromJoinChain(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		joinKind, matched := p.matchJoinKeyword()
		if !matched {
			return nil
		}
		if err := p.parseFromSource(analysis, joinKind); err != nil {
			return err
		}
		_ = p.matchKeyword(kwFinal)
		if err := p.parseJoinQualifier(analysis); err != nil {
			return err
		}
	}
}

// parseJoinQualifier reads the ON or USING clause that follows a JOIN.
//
// It is pulled out so parseFromJoinChain stays small enough to satisfy the
// max-control-nesting linter.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the join qualifier is malformed.
func (p *parser) parseJoinQualifier(analysis *querier_dto.RawQueryAnalysis) error {
	if p.matchKeyword(kwOn) {
		return p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopJoinOnClause...)
	}
	if p.matchKeyword("USING") && p.current().kind == tokenLeftParen {
		_ = p.skipParenthesised()
	}
	return nil
}

// parseFromSource reads one table reference in a FROM or JOIN clause, handling plain
// tables, derived-table subqueries, and table-valued function calls.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes joinKind (querier_dto.JoinKind) which is the join kind for this source.
//
// Returns error when the table reference is malformed.
func (p *parser) parseFromSource(analysis *querier_dto.RawQueryAnalysis, joinKind querier_dto.JoinKind) error {
	if p.current().kind == tokenLeftParen {
		return p.parseFromDerivedSubquery(analysis, joinKind)
	}
	first, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return err
	}
	if p.current().kind == tokenDot {
		return p.parseFromQualifiedTable(analysis, joinKind, first)
	}
	if p.current().kind == tokenLeftParen {
		return p.parseFromTableValuedFunction(analysis, joinKind, first)
	}
	alias, _, aliasErr := p.parseOptionalAlias()
	if aliasErr != nil {
		return aliasErr
	}
	appendTableReference(analysis, querier_dto.TableReference{
		Name:  first,
		Alias: alias,
	}, joinKind)
	return nil
}

// parseFromDerivedSubquery handles `(SELECT ...) [AS] alias` derived table references.
//
// The subquery is fully re-parsed via a nested parser so the analysis records the inner
// output columns and join chain alongside the alias.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes joinKind (querier_dto.JoinKind) which is the join kind for this source.
//
// Returns error when the derived subquery is malformed.
func (p *parser) parseFromDerivedSubquery(analysis *querier_dto.RawQueryAnalysis, joinKind querier_dto.JoinKind) error {
	body, err := p.collectParenthesised()
	if err != nil {
		return err
	}
	nested := newParser(body)
	nested.analysisDepth = p.analysisDepth
	nested.maxParseDepth = p.maxParseDepth
	nestedAnalysis, nestedErr := nested.analyseSelect()
	if nestedErr != nil {
		return nestedErr
	}
	alias, _, aliasErr := p.parseOptionalAlias()
	if aliasErr != nil {
		return aliasErr
	}
	p.mergeNestedParameterReferences(analysis, nested, nestedAnalysis)
	analysis.RawDerivedTables = append(analysis.RawDerivedTables, querier_dto.RawDerivedTableReference{
		InnerQuery: nestedAnalysis,
		Alias:      alias,
		JoinKind:   joinKind,
	})
	return nil
}

// parseFromQualifiedTable handles `schema.table [AS] alias` after the leading identifier
// and dot have already been consumed.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes joinKind (querier_dto.JoinKind) which is the join kind for this source.
// Takes schemaName (string) which is the leading schema qualifier.
//
// Returns error when the qualified table reference is malformed.
func (p *parser) parseFromQualifiedTable(analysis *querier_dto.RawQueryAnalysis, joinKind querier_dto.JoinKind, schemaName string) error {
	p.advance()
	second, secondErr := p.parseIdentifierOrKeyword()
	if secondErr != nil {
		return secondErr
	}

	if p.current().kind == tokenLeftParen {
		_ = p.skipParenthesised()
		alias, _, aliasErr := p.parseOptionalAlias()
		if aliasErr != nil {
			return aliasErr
		}
		analysis.RawTableValuedFunctions = append(analysis.RawTableValuedFunctions, querier_dto.RawTableValuedFunctionReference{
			FunctionName: schemaName + "." + second,
			Alias:        alias,
			JoinKind:     joinKind,
		})
		return nil
	}
	alias, _, aliasErr := p.parseOptionalAlias()
	if aliasErr != nil {
		return aliasErr
	}
	appendTableReference(analysis, querier_dto.TableReference{
		Schema: schemaName,
		Name:   second,
		Alias:  alias,
	}, joinKind)
	return nil
}

// parseFromTableValuedFunction handles a table-valued function call like `numbers(10)
// [AS] alias`. The argument list is consumed opaquely; downstream resolution maps the
// function name to its declared column shape.
//
// This consumes the argument list opaquely (skipParenthesised) on purpose, so a
// placeholder inside a ClickHouse table function (for example `numbers({n:UInt64})`) is
// never registered as a parameter at all. ClickHouse therefore does NOT take part in the
// P1 function-argument type back-propagation that the placeholder dialects use: there is
// no captured placeholder to type, and ClickHouse's built-in table functions (numbers,
// generateSeries, remote, cluster) do not expose a catalogue-declared, per-argument typed
// signature to recover a type from. Even when a `{name:Type}` placeholder did appear here
// it would already carry its own CastType (see registerClickHouseParameter and
// parseFunctionCallExpression), so the shared resolver would type it from that tag rather
// than from an argument ordinal. Bringing this path into scope would mean parsing the
// argument list non-opaquely for no typing gain, so it stays opaque.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes joinKind (querier_dto.JoinKind) which is the join kind for this source.
// Takes functionName (string) which is the table function name.
//
// Returns error when the table-valued function reference is malformed.
func (p *parser) parseFromTableValuedFunction(analysis *querier_dto.RawQueryAnalysis, joinKind querier_dto.JoinKind, functionName string) error {
	_ = p.skipParenthesised()
	alias, _, aliasErr := p.parseOptionalAlias()
	if aliasErr != nil {
		return aliasErr
	}
	analysis.RawTableValuedFunctions = append(analysis.RawTableValuedFunctions, querier_dto.RawTableValuedFunctionReference{
		FunctionName: functionName,
		Alias:        alias,
		JoinKind:     joinKind,
	})
	return nil
}

// appendTableReference adds the table to the FROM list, and also records a JoinClause
// when the join kind is non-Inner so downstream consumers can track per-join nullability.
//
// The first FROM entry is always treated as inner-joined; subsequent joined entries need
// both the FromTables append and the JoinClauses append.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
// Takes ref (querier_dto.TableReference) which is the table reference to append.
// Takes joinKind (querier_dto.JoinKind) which is the join kind for this table.
func appendTableReference(analysis *querier_dto.RawQueryAnalysis, ref querier_dto.TableReference, joinKind querier_dto.JoinKind) {
	analysis.FromTables = append(analysis.FromTables, ref)
	if len(analysis.FromTables) > 1 && joinKind != querier_dto.JoinInner {
		analysis.JoinClauses = append(analysis.JoinClauses, querier_dto.JoinClause{
			Table: ref,
			Kind:  joinKind,
		})
	}
}

// parseWhereClause consumes the WHERE expression. Tracks any `{name:Type}` parameter
// references encountered and adds them to the analysis with ParameterContextComparison.
//
// Unlike the placeholder dialects (sqlite/postgres/mysql/duckdb), ClickHouse does NOT
// need a PredicateSubqueries carrier for a parameter that sits inside a WHERE-position
// subquery: a ClickHouse parameter is `{name:Type}` and carries its own explicit type, so
// it is never typed by resolving a column reference against the scope chain. The shared
// analyser's column-scope resolution (which raises the false Q001 when a subquery-local
// alias is unknown to the outer scope) is therefore never exercised for ClickHouse
// parameters, so the predicate-subquery capture done for the other engines is unnecessary
// here.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when the WHERE expression is malformed.
func (p *parser) parseWhereClause(analysis *querier_dto.RawQueryAnalysis) error {
	return p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, stopGroupByClause...)
}

// parseGroupByColumns reads the GROUP BY column list.
//
// ClickHouse allows expressions and column ordinals (`GROUP BY 1, 2`); the helper
// captures identifiers and skips more complex expressions.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
//
// Returns error when a GROUP BY column is malformed.
func (p *parser) parseGroupByColumns(analysis *querier_dto.RawQueryAnalysis) error {
	for {
		if p.atEnd() {
			return nil
		}

		table, column, isDirect := p.tryParseDirectColumnReference()
		if isDirect {
			analysis.GroupByColumns = append(analysis.GroupByColumns, querier_dto.ColumnReference{
				TableAlias: table,
				ColumnName: column,
			})
		} else {
			p.consumeOneExpression(analysis, querier_dto.ParameterContextComparison)
		}
		if p.current().kind != tokenComma {
			break
		}
		p.advance()
	}
	return nil
}

// parseLimitClause reads a LIMIT clause body.
//
// ClickHouse supports `LIMIT n`, `LIMIT n OFFSET m`, `LIMIT m, n`, `LIMIT n WITH TIES`,
// and `LIMIT n BY cols`. Every sub-expression is consumed via the parameter-tracking
// helper so `LIMIT {n:UInt32}` and `LIMIT {n:UInt32} BY user_id` register the placeholder
// against the analysis. The first expression's stop list includes the top-level comma so
// the `LIMIT m, n` form parses correctly: in that form the first expression is the offset
// and the second is the row count.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to populate.
func (p *parser) parseLimitClause(analysis *querier_dto.RawQueryAnalysis) {
	limitFirstStops := []string{kwOffset, kwBy, kwWith, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept}
	limitOffsetStops := []string{kwBy, kwWith, kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept}
	limitByStops := []string{kwSettings, kwFormat, kwUnion, kwIntersect, kwExcept, kwWith}
	p.consumeExpressionTrackingParamsCommaAware(analysis, querier_dto.ParameterContextComparison, limitFirstStops...)
	for {
		switch {
		case p.matchKeyword(kwOffset):
			_ = p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, limitOffsetStops...)
		case p.matchKeyword(kwWith):
			p.parseLimitWithModifier(analysis)
		case p.matchKeyword(kwBy):
			_ = p.consumeExpressionTrackingParams(analysis, querier_dto.ParameterContextComparison, limitByStops...)
			return
		case p.current().kind == tokenComma:
			p.advance()
			p.consumeExpressionTrackingParamsCommaAware(analysis, querier_dto.ParameterContextComparison, limitFirstStops...)
		default:
			return
		}
	}
}

// consumeExpressionTrackingParamsCommaAware consumes an expression body like
// consumeExpressionTrackingParams but also stops on a top-level comma.
//
// Clauses such as `LIMIT m, n` and `ORDER BY x, y, ...` use it where commas separate
// sibling expressions rather than continuing a single expression. The function never
// returns an error; the signature is kept void so the loop body stays free of an
// uninspected return value.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
// Takes stopKeywords (...string) which are the clause keywords that halt the scan.
func (p *parser) consumeExpressionTrackingParamsCommaAware(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
	stopKeywords ...string,
) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tokenIsTopLevelStop(tok, stopKeywords) {
			return
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// matchCompoundOperator recognises UNION ALL, UNION DISTINCT, INTERSECT, and EXCEPT and
// consumes the operator tokens.
//
// Returns querier_dto.CompoundOperator which is the matched operator enum value.
// Returns bool which is true when an operator was consumed.
func (p *parser) matchCompoundOperator() (querier_dto.CompoundOperator, bool) {
	switch {
	case p.matchKeyword("UNION"):
		if p.matchKeyword("ALL") {
			return querier_dto.CompoundUnionAll, true
		}
		p.matchKeyword("DISTINCT")
		return querier_dto.CompoundUnion, true
	case p.matchKeyword("INTERSECT"):
		return querier_dto.CompoundIntersect, true
	case p.matchKeyword("EXCEPT"):
		return querier_dto.CompoundExcept, true
	}
	return 0, false
}

// consumeOneExpression consumes a single expression, terminating at a comma or top-level
// clause keyword.
//
// The expression may be any combination of identifiers, numbers, strings, parameters,
// parenthesised groups, and infix operators. GROUP BY items use it. Any `{name:Type}`
// placeholder is registered against the analysis so a parameter inside a non-column GROUP
// BY expression (for example `GROUP BY toStartOfInterval(ts, {step:UInt32})`) is
// retained.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
func (p *parser) consumeOneExpression(analysis *querier_dto.RawQueryAnalysis, context querier_dto.ParameterContext) {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 {
			if tok.kind == tokenComma {
				return
			}
			if tok.kind == tokenIdentifier && isTopClauseKeyword(tok.value) {
				return
			}
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return
		}
		depth = newDepth
		p.advance()
	}
}

// identifierMatchesAny reports whether name matches one of the supplied stop keywords
// case-insensitively.
//
// It is extracted to avoid inlining a nested loop in the consume helpers.
//
// Takes name (string) which is the identifier to test.
// Takes stopKeywords ([]string) which are the keywords to match against.
//
// Returns bool which is true when name matches a stop keyword.
func identifierMatchesAny(name string, stopKeywords []string) bool {
	return slices.ContainsFunc(stopKeywords, func(stop string) bool {
		return strings.EqualFold(name, stop)
	})
}

// consumeExpressionTrackingParams consumes an expression body and records every
// `{name:Type}` parameter token it encounters as a parameter reference on the analysis.
//
// WHERE clauses use it where parameter context matters for downstream codegen.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
// Takes stopKeywords (...string) which are the clause keywords that halt the scan.
//
// Returns error which is always nil; the signature keeps the error result so callers can
// chain it inline.
func (p *parser) consumeExpressionTrackingParams(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
	stopKeywords ...string,
) error {
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier && identifierMatchesAny(tok.value, stopKeywords) {
			return nil
		}
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			return nil
		}
		depth = newDepth
		p.advance()
	}
	return nil
}

// collectClickHouseParametersUntilEnd walks the rest of the statement and registers any
// `{name:Type}` placeholders with the analysis.
//
// INSERT VALUES paths use it where the rest of the statement is parsed opaquely, because
// the driver handles the values list at runtime, but parameter references must still be
// surfaced to the codegen layer so the generated method gets typed arguments.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register
// placeholders against.
// Takes context (querier_dto.ParameterContext) which is the context to record for any
// placeholder found.
func (p *parser) collectClickHouseParametersUntilEnd(
	analysis *querier_dto.RawQueryAnalysis,
	context querier_dto.ParameterContext,
) {
	for !p.atEnd() {
		tok := p.current()
		if tok.kind == tokenClickHouseParam {
			p.registerClickHouseParameter(analysis, tok, context)
		}
		p.advance()
	}
}

// registerClickHouseParameter records a `{name:Type}` placeholder on the analysis. The
// token's value is `name:Type`; we split on `:` to extract the name and the type tag for
// downstream resolution.
//
// Type-parse errors are tracked on the parser so the surrounding analyser (analyseSelect
// / analyseInsert) can surface a warning via the engine's diagnostic channel. The
// parameter is still registered with a nil CastType so codegen falls back to the
// unknown-type path instead of dropping the binding entirely.
//
// A tag that parses cleanly but names no known type (for example {x:Strign}) resolves to
// the Unknown category rather than an error. That case is also recorded on the parser so
// the binding is no longer silently untyped; the parameter keeps its Unknown cast so
// codegen retains the binding.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to register the
// placeholder against.
// Takes tok (token) which is the `{name:Type}` placeholder token.
// Takes context (querier_dto.ParameterContext) which is the context to record.
func (p *parser) registerClickHouseParameter(
	analysis *querier_dto.RawQueryAnalysis,
	tok token,
	context querier_dto.ParameterContext,
) {
	name, typeName := splitClickHouseParamBody(tok.value)
	number, exists := p.namedParameterMap[name]
	if !exists {
		p.parameterCount++
		number = p.parameterCount
		p.namedParameterMap[name] = number
	}
	var castType *querier_dto.SQLType
	if typeName != "" {
		t, err := parseClickHouseType(typeName)

		if err == nil && t.Nullable {
			t.SQLType.Nullable = true
		}
		switch {
		case err != nil:
			if p.firstParameterTypeError == nil {
				p.firstParameterTypeError = fmt.Errorf("clickhouse: parameter %q has malformed type tag %q at position %d: %w", name, typeName, tok.position, err)
			}
		case t.SQLType.Category == querier_dto.TypeCategoryUnknown:

			castType = &t.SQLType
			if p.firstParameterTypeError == nil {
				p.firstParameterTypeError = fmt.Errorf("clickhouse: parameter %q has unrecognised type tag %q at position %d", name, typeName, tok.position)
			}
		default:
			castType = &t.SQLType
		}
	}
	analysis.ParameterReferences = append(analysis.ParameterReferences, querier_dto.RawParameterReference{
		Name:     name,
		Number:   number,
		Context:  context,
		CastType: castType,
	})
}

// mergeNestedParameterReferences folds a nested parser's parameter references into the
// parent analysis so they appear in the final parameter list.
//
// The nested parser is the one for a CTE body or a FROM-derived subquery. Each named
// placeholder is re-keyed through the parent parser's namedParameterMap so binding
// numbers stay consistent across the whole statement and the same {name:Type} used in
// several scopes collapses to one parameter. This mirrors the INSERT ... SELECT merge in
// analyseInsertBody; without it CTE and derived-subquery parameters are dropped because
// each nested parser numbers from one and never flattens into the outer list.
//
// The first malformed parameter-type tag seen inside a nested body is also lifted onto
// the parent parser so the surrounding analyser surfaces a diagnostic, matching the
// behaviour for top-level SELECT and INSERT placeholders.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the parent analysis to extend.
// Takes nested (*parser) which is the sub-parser that produced nestedAnalysis.
// Takes nestedAnalysis (*querier_dto.RawQueryAnalysis) which holds the nested references.
func (p *parser) mergeNestedParameterReferences(
	analysis *querier_dto.RawQueryAnalysis,
	nested *parser,
	nestedAnalysis *querier_dto.RawQueryAnalysis,
) {
	if nestedAnalysis != nil {
		for index := range nestedAnalysis.ParameterReferences {
			ref := nestedAnalysis.ParameterReferences[index]
			if ref.Name != "" {
				number, exists := p.namedParameterMap[ref.Name]
				if !exists {
					p.parameterCount++
					number = p.parameterCount
					p.namedParameterMap[ref.Name] = number
				}
				ref.Number = number
			}
			analysis.ParameterReferences = append(analysis.ParameterReferences, ref)
		}
	}
	if p.firstParameterTypeError == nil && nested != nil && nested.firstParameterTypeError != nil {
		p.firstParameterTypeError = nested.firstParameterTypeError
	}
}

// splitClickHouseParamBody splits the placeholder body `name:Type` into its two halves.
//
// Type may be empty when malformed; the catalogue resolver surfaces a diagnostic in that
// case.
//
// Takes body (string) which is the placeholder body of shape `name:Type`.
//
// Returns name (string) which is the parameter identifier.
// Returns typeName (string) which is the type tag, possibly empty.
func splitClickHouseParamBody(body string) (name string, typeName string) {
	nameSegment, typeSegment, found := strings.Cut(body, ":")
	if !found {
		return body, ""
	}
	return strings.TrimSpace(nameSegment), strings.TrimSpace(typeSegment)
}

// isTopClauseKeyword reports whether the supplied identifier is one of the top-level
// clause keywords that terminates an expression.
//
// Takes name (string) which is the identifier to test.
//
// Returns bool which is true when name is a top-level clause keyword.
func isTopClauseKeyword(name string) bool {
	switch strings.ToUpper(name) {
	case kwFrom, kwWhere, kwGroup, kwHaving, kwOrder, kwLimit, kwSettings,
		kwFormat, kwUnion, kwIntersect, kwExcept, kwJoin, kwInner, kwLeft,
		kwRight, kwFull, kwCross, kwOn, "USING", kwPrewhere, kwWindow,
		kwArray, kwFinal, "SAMPLE", kwAsof, kwSemi, kwAnti, kwOffset, kwBy,
		kwWith, kwQualify, kwAny, kwAll, kwGlobal:
		return true
	}
	return false
}
