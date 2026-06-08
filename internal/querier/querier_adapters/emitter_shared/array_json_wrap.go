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

package emitter_shared

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// sqlWhitespace is the set of bytes treated as SQL whitespace when trimming projection
	// items and keywords.
	sqlWhitespace = " \t\n\r"

	// keywordON is the DISTINCT ON keyword the wrap declines to rewrite.
	keywordON = "ON"

	// goSliceTypePrefix marks a Go slice type. An array column whose resolved type starts
	// with it uses the framework default ([]T) and is eligible for the to_json wrap; a type
	// override (for example github.com/lib/pq.StringArray) does not and is left untouched.
	goSliceTypePrefix = "[]"
)

// arrayColumnUsesDefaultSlice reports a default-slice array column.
//
// This is the only shape the to_json wrap should rewrite. A per-column go_type override
// or a config type override (for example mapping text[] to lib/pq.StringArray, which
// scans arrays itself) resolves to a non-slice type and is skipped so the user's chosen
// type is respected.
//
// Takes column (*querier_dto.OutputColumn) which is the output column to classify.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves the column's Go type.
//
// Returns bool which is true when the column is a default-slice array column.
func arrayColumnUsesDefaultSlice(column *querier_dto.OutputColumn, mappings *querier_dto.TypeMappingTable) bool {
	if column.SQLType.Category != querier_dto.TypeCategoryArray || column.GoTypeOverride != nil {
		return false
	}
	resolved := ResolveGoType(column.SQLType, column.Nullable, mappings)
	return strings.HasPrefix(resolved.Name, goSliceTypePrefix)
}

// validateArrayColumnsWrappable fails generation for an unreachable array column.
//
// It applies when an engine that decodes array columns through to_json (postgres family,
// duckdb) has an array OUTPUT column the wrap cannot reach. Without it the column would
// compile as a typed slice but crash at runtime on the driver's raw array text. Engines
// that scan arrays natively (pgx, ClickHouse) or have no array type return an empty
// ArrayJSONWrapFunc and are skipped.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which are the analysed queries to check.
// Takes strategy (MethodStrategy) which supplies the engine's array-to-JSON function.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type so
// a config override (for example text[] -> pq.StringArray) is not treated as a default
// slice.
//
// Returns error which names the offending query and column when an array column cannot be
// wrapped, or nil when every array column is wrappable.
func validateArrayColumnsWrappable(queries []*querier_dto.AnalysedQuery, strategy MethodStrategy, mappings *querier_dto.TypeMappingTable) error {
	if strategy == nil {
		return nil
	}
	wrapFunc := strategy.ArrayJSONWrapFunc()
	if wrapFunc == "" {
		return nil
	}

	for _, query := range queries {
		_, wrapped := WrapArrayColumnsAsJSON(query.SQL, query.OutputColumns, wrapFunc, strategy.QuoteIdentifier, mappings)
		for index := range query.OutputColumns {
			column := &query.OutputColumns[index]
			if !arrayColumnUsesDefaultSlice(column, mappings) || wrapped[index] {
				continue
			}
			return fmt.Errorf(
				"query %q: array column %q cannot be auto-decoded in this query shape "+
					"(a compound UNION/INTERSECT/EXCEPT query, a piko.embed group, or a SELECT or "+
					"RETURNING projection item piko cannot rewrite, such as an unaliased expression or a "+
					"star mixed with other items); rewrite it as an explicit aliased projection so piko "+
					"can wrap it in %s() for typed scanning",
				query.Name, column.Name, wrapFunc)
		}
	}
	return nil
}

// WrappedArrayColumns returns the set of output-column indices that the array-JSON wrap
// rewrites for this query and strategy, or nil when none. It lets the scan codegen and
// the import collector agree with BuildSQLConstant on exactly which columns are
// JSON-decoded, since all three derive the set from the same query SQL.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the SQL and output columns.
// Takes strategy (MethodStrategy) which supplies the engine's array-to-JSON function.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type.
//
// Returns map[int]bool which marks the wrapped output-column indices (nil when none).
func WrappedArrayColumns(query *querier_dto.AnalysedQuery, strategy MethodStrategy, mappings *querier_dto.TypeMappingTable) map[int]bool {
	if strategy == nil {
		return nil
	}
	wrapFunc := strategy.ArrayJSONWrapFunc()
	if wrapFunc == "" {
		return nil
	}
	_, wrapped := WrapArrayColumnsAsJSON(query.SQL, query.OutputColumns, wrapFunc, strategy.QuoteIdentifier, mappings)
	return wrapped
}

// WrapArrayColumnsAsJSON rewrites a query's output projection so each array output column
// is wrapped in jsonFunc (for example to_json), letting database/sql decode the column as
// JSON into a typed slice rather than failing on the driver's raw array text.
//
// The output projection is the RETURNING list of a data-modifying statement (INSERT,
// UPDATE or DELETE ... RETURNING) when one is present at the top level, and otherwise the
// top-level SELECT projection. RETURNING takes precedence so an INSERT ... SELECT ...
// RETURNING wraps the returned columns rather than the source-row SELECT it reads from.
//
// The rewrite is all-or-nothing and conservative: it returns the original SQL and a nil
// set when the projection cannot be rewritten with confidence (a compound
// UNION/INTERSECT/EXCEPT query, an embed group, an item count that does not match the
// output columns, or a projection item it cannot safely split). A plain SELECT * or
// RETURNING * is expanded into an explicit projection from the analysed columns' source
// names so its array columns can be wrapped; a leading DISTINCT or DISTINCT ON (...) is
// preserved ahead of a SELECT rewrite. The returned set holds the output-column indices
// that were wrapped so the scan can decode exactly those columns.
//
// Takes sql (string) which is the query SQL whose projection may be rewritten.
// Takes columns ([]querier_dto.OutputColumn) which are the analysed output columns in
// projection order.
// Takes jsonFunc (string) which is the engine's array-to-JSON function, or empty to
// disable.
// Takes quote (func(string) string) which quotes an identifier; used only when expanding
// a star projection.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type so
// only columns that default to a plain slice are wrapped.
//
// Returns string which is the rewritten SQL (or the original when no rewrite is applied).
// Returns map[int]bool which marks the wrapped output-column indices (nil when none).
func WrapArrayColumnsAsJSON(sql string, columns []querier_dto.OutputColumn, jsonFunc string, quote func(string) string, mappings *querier_dto.TypeMappingTable) (string, map[int]bool) {
	if jsonFunc == "" || !hasWrappableArrayColumn(columns, mappings) {
		return sql, nil
	}

	if projectionStart, projectionEnd, ok := topLevelReturningBounds(sql); ok {
		return wrapReturningProjection(sql, projectionStart, projectionEnd, columns, jsonFunc, quote, mappings)
	}

	return wrapSelectProjection(sql, columns, jsonFunc, quote, mappings)
}

// wrapSelectProjection rewrites the array output columns of a top-level SELECT
// projection, preserving a leading DISTINCT or DISTINCT ON (...) clause ahead of the
// rewrite. It returns the original SQL and a nil set when the projection cannot be
// located or rewritten with confidence.
//
// Takes sql (string) which is the query SQL.
// Takes columns ([]querier_dto.OutputColumn) which are the analysed output columns.
// Takes jsonFunc (string) which is the engine's array-to-JSON function.
// Takes quote (func(string) string) which quotes an identifier for a star expansion.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type.
//
// Returns string which is the rewritten SQL (or the original on bail).
// Returns map[int]bool which marks the wrapped indices (nil on bail).
func wrapSelectProjection(sql string, columns []querier_dto.OutputColumn, jsonFunc string, quote func(string) string, mappings *querier_dto.TypeMappingTable) (string, map[int]bool) {
	projectionStart, projectionEnd, ok := topLevelProjectionBounds(sql)
	if !ok {
		return sql, nil
	}

	distinctPrefix, body, ok := splitLeadingDistinct(sql[projectionStart:projectionEnd])
	if !ok {
		return sql, nil
	}

	items := splitTopLevelCommas(body)

	if projection, wrapped, isStar := expandStarProjection(items, columns, jsonFunc, quote, mappings); isStar {
		if wrapped == nil {
			return sql, nil
		}
		prefix := strings.TrimRight(sql[:projectionStart]+distinctPrefix, sqlWhitespace) + " "
		return prefix + projection + " " + sql[projectionEnd:], wrapped
	}

	if len(items) != len(columns) {
		return sql, nil
	}

	head := sql[:projectionStart] + distinctPrefix
	result, wrapped := rebuildWrappedProjection(head, sql[projectionEnd:], items, columns, jsonFunc, quote, mappings)
	if wrapped == nil {
		return sql, nil
	}
	return result, wrapped
}

// wrapReturningProjection rewrites the array output columns of a top-level RETURNING
// list, whose byte span the caller has already located. It mirrors wrapSelectProjection
// but has no DISTINCT to preserve and ends at the statement terminator rather than a FROM
// clause.
//
// Takes sql (string) which is the query SQL.
// Takes projectionStart (int) which is the offset just after the RETURNING keyword.
// Takes projectionEnd (int) which is the offset of the statement terminator or end.
// Takes columns ([]querier_dto.OutputColumn) which are the analysed output columns.
// Takes jsonFunc (string) which is the engine's array-to-JSON function.
// Takes quote (func(string) string) which quotes an identifier for a star expansion.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type.
//
// Returns string which is the rewritten SQL (or the original on bail).
// Returns map[int]bool which marks the wrapped indices (nil on bail).
func wrapReturningProjection(
	sql string,
	projectionStart int,
	projectionEnd int,
	columns []querier_dto.OutputColumn,
	jsonFunc string,
	quote func(string) string,
	mappings *querier_dto.TypeMappingTable,
) (string, map[int]bool) {
	items := splitTopLevelCommas(sql[projectionStart:projectionEnd])

	if projection, wrapped, isStar := expandStarProjection(items, columns, jsonFunc, quote, mappings); isStar {
		if wrapped == nil {
			return sql, nil
		}
		prefix := strings.TrimRight(sql[:projectionStart], sqlWhitespace) + " "
		return prefix + projection + sql[projectionEnd:], wrapped
	}

	if len(items) != len(columns) {
		return sql, nil
	}

	result, wrapped := rebuildWrappedProjection(sql[:projectionStart], sql[projectionEnd:], items, columns, jsonFunc, quote, mappings)
	if wrapped == nil {
		return sql, nil
	}
	return result, wrapped
}

// topLevelReturningBounds locates the byte span of a top-level RETURNING projection, the
// output projection of a data-modifying INSERT/UPDATE/DELETE statement.
//
// It matches the last depth-0 RETURNING keyword so a CTE-internal RETURNING (at paren
// depth greater than zero) is ignored and a column named "returning" earlier in the
// statement cannot be mistaken for the clause. The projection runs from just after the
// keyword to the statement terminator, since RETURNING is always the final clause.
//
// Takes sql (string) which is the query SQL.
//
// Returns projectionStart (int) which is the offset just after the RETURNING keyword.
// Returns projectionEnd (int) which is the offset of the terminator or end of input.
// Returns ok (bool) which is true when a top-level RETURNING clause was found.
func topLevelReturningBounds(sql string) (projectionStart int, projectionEnd int, ok bool) {
	returningIndex := findLastTopLevelKeywordIndex(sql, "RETURNING")
	if returningIndex == -1 {
		return 0, 0, false
	}
	projectionStart = returningIndex + len("RETURNING")
	projectionEnd = topLevelStatementTerminator(sql, projectionStart)
	return projectionStart, projectionEnd, true
}

// topLevelStatementTerminator returns the index of the first depth-0 semicolon at or
// after start, or len(sql) when there is none.
//
// String literals and comments are skipped with the shared advancePastSQLNoise primitive
// so a semicolon inside a quoted value is not mistaken for the terminator. RETURNING is
// always the final clause, so the first such semicolon ends the projection and everything
// before it is the RETURNING list.
//
// Takes sql (string) which is the query SQL.
// Takes start (int) which is the offset to begin scanning from.
//
// Returns int which is the terminator index, or len(sql) when absent.
func topLevelStatementTerminator(sql string, start int) int {
	position := start
	for position < len(sql) {
		if nextPosition, skipped := advancePastSQLNoise(sql, position); skipped {
			position = nextPosition
			continue
		}
		if sql[position] == ';' {
			return position
		}
		position++
	}
	return len(sql)
}

// expandStarProjection rebuilds a star projection into an explicit list.
//
// This lets its array columns be wrapped in jsonFunc. The analysed output columns carry
// the real source column name, source qualifier, and projection order, so the rebuilt
// list matches the order the scan expects. It declines (isStar false) when the single
// projection item is not a star, and bails (isStar true, wrapped nil) when a column lacks
// the source information needed to reconstruct its reference, so the caller keeps the
// original SQL and the loud validator reports it.
//
// Takes items ([]string) which are the top-level projection items.
// Takes columns ([]querier_dto.OutputColumn) which are the analysed output columns in
// order.
// Takes jsonFunc (string) which is the engine's array-to-JSON function.
// Takes quote (func(string) string) which quotes an identifier for the engine.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves each column's Go type.
//
// Returns projection (string) which is the rebuilt explicit projection (empty on bail).
// Returns wrapped (map[int]bool) which marks the wrapped output-column indices (nil on
// bail).
// Returns isStar (bool) which is true when the projection was a star (expanded or
// bailed).
func expandStarProjection(
	items []string,
	columns []querier_dto.OutputColumn,
	jsonFunc string,
	quote func(string) string,
	mappings *querier_dto.TypeMappingTable,
) (projection string, wrapped map[int]bool, isStar bool) {
	if len(items) != 1 || !isStarProjectionItem(strings.TrimSpace(items[0])) {
		return "", nil, false
	}
	if quote == nil {
		return "", nil, true
	}

	wrapped = make(map[int]bool)
	var builder strings.Builder
	for index := range columns {
		column := &columns[index]
		if column.SourceColumn == "" {
			return "", nil, true
		}
		if index > 0 {
			builder.WriteString(", ")
		}

		reference := quote(column.SourceColumn)
		if column.SourceQualifier != "" {
			reference = quote(column.SourceQualifier) + "." + reference
		}
		if arrayColumnUsesDefaultSlice(column, mappings) {
			builder.WriteString(jsonFunc + "(" + reference + ") AS " + quote(column.Name))
			wrapped[index] = true
			continue
		}
		builder.WriteString(reference)
	}
	return builder.String(), wrapped, true
}

// hasWrappableArrayColumn reports whether the output columns include an array column and
// no embed group. An embed group projects multiple columns per item, breaking the
// one-to-one item-to-column mapping the wrap relies on, so its presence disables the
// wrap.
//
// Takes columns ([]querier_dto.OutputColumn) which are the analysed output columns.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves column Go types.
//
// Returns bool which is true when a default-slice array column is present and no embed
// group is.
func hasWrappableArrayColumn(columns []querier_dto.OutputColumn, mappings *querier_dto.TypeMappingTable) bool {
	hasArray := false
	for index := range columns {
		if columns[index].IsEmbedded {
			return false
		}
		if arrayColumnUsesDefaultSlice(&columns[index], mappings) {
			hasArray = true
		}
	}
	return hasArray
}

// topLevelProjectionBounds locates the byte span of the top-level SELECT projection,
// declining (ok false) on a compound query (UNION / INTERSECT / EXCEPT), whose branches
// must keep matching column types, and when no depth-0 SELECT is found.
//
// The projection ends at the earliest top-level clause keyword that can follow it (FROM,
// WHERE, GROUP, HAVING, WINDOW, ORDER, LIMIT, OFFSET, FETCH or FOR) or, for a FROM-less
// SELECT such as a bare function call that returns an array, at the statement terminator.
// A SELECT with a FROM therefore still ends exactly at that FROM.
//
// Takes sql (string) which is the query SQL.
//
// Returns projectionStart (int) which is the index after SELECT.
// Returns projectionEnd (int) which is the index where the projection ends.
// Returns ok (bool) which is true when a safe projection span was found.
func topLevelProjectionBounds(sql string) (projectionStart int, projectionEnd int, ok bool) {
	for _, keyword := range []string{"UNION", "INTERSECT", "EXCEPT"} {
		if findTopLevelKeywordIndex(sql, 0, keyword) != -1 {
			return 0, 0, false
		}
	}
	selectIndex := findTopLevelKeywordIndex(sql, 0, "SELECT")
	if selectIndex == -1 {
		return 0, 0, false
	}
	projectionStart = selectIndex + len("SELECT")
	return projectionStart, topLevelProjectionEnd(sql, projectionStart), true
}

// topLevelProjectionEnd returns the end offset of a SELECT projection.
//
// The end is the earliest top-level clause keyword that can follow a projection, or the
// statement terminator when the SELECT carries no trailing clause. A clause keyword
// nested inside a projection item, such as the ORDER BY of an array_agg(...), sits at
// parenthesis depth greater than zero and is not matched.
//
// Takes sql (string) which is the query SQL.
// Takes start (int) which is the projection start, just after the SELECT keyword.
//
// Returns int which is the projection end offset.
func topLevelProjectionEnd(sql string, start int) int {
	end := topLevelStatementTerminator(sql, start)
	for _, keyword := range []string{"FROM", "WHERE", "GROUP", "HAVING", "WINDOW", "ORDER", "LIMIT", "OFFSET", "FETCH", "FOR"} {
		if index := findTopLevelKeywordIndex(sql, start, keyword); index != -1 && index < end {
			end = index
		}
	}
	return end
}

// rebuildWrappedProjection reassembles the query with each default-slice array column's
// projection item wrapped in jsonFunc. It returns an empty string and a nil set if any
// array item cannot be rewritten safely, keeping the rewrite all-or-nothing.
//
// Takes head (string) which is the text up to and including the projection (SELECT plus
// any DISTINCT prefix).
// Takes tail (string) which is the text from FROM onward.
// Takes items ([]string) which are the projection items in order.
// Takes columns ([]querier_dto.OutputColumn) which parallel the items.
// Takes jsonFunc (string) which is the array-to-JSON function.
// Takes quote (func(string) string) which quotes a synthesised output alias.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves column Go types.
//
// Returns string which is the rewritten SQL (empty on bail).
// Returns map[int]bool which marks the wrapped indices (nil on bail).
func rebuildWrappedProjection(
	head string,
	tail string,
	items []string,
	columns []querier_dto.OutputColumn,
	jsonFunc string,
	quote func(string) string,
	mappings *querier_dto.TypeMappingTable,
) (string, map[int]bool) {
	wrapped := make(map[int]bool)
	var builder strings.Builder
	builder.WriteString(head)
	for index, item := range items {
		if index > 0 {
			builder.WriteByte(',')
		}
		if !arrayColumnUsesDefaultSlice(&columns[index], mappings) {
			builder.WriteString(item)
			continue
		}
		rewritten, ok := wrapProjectionItem(item, jsonFunc, quote, columns[index].Name)
		if !ok {
			return "", nil
		}
		builder.WriteString(rewritten)
		wrapped[index] = true
	}
	builder.WriteString(tail)
	return builder.String(), wrapped
}

// splitLeadingDistinct separates a leading DISTINCT clause from the list.
//
// This lets the wrap preserve it ahead of the projection, including a DISTINCT ON key
// list. A plain DISTINCT is safe because DISTINCT to_json(col) equals DISTINCT col
// (to_json is a deterministic function of the column); a DISTINCT ON (keys) keeps its
// parenthesised key list verbatim, since the keys ride ahead of the projection and
// to_json does not change them. It declines (ok false) only when DISTINCT ON is present
// but its parenthesised key list is malformed.
//
// Takes projection (string) which is the text between SELECT and FROM.
//
// Returns prefix (string) which is the leading DISTINCT text to re-emit verbatim (empty
// when there is no DISTINCT).
// Returns body (string) which is the column-list portion to split and wrap.
// Returns ok (bool) which is false only for a malformed DISTINCT ON, signalling a bail.
func splitLeadingDistinct(projection string) (prefix string, body string, ok bool) {
	const distinct = "DISTINCT"
	leading := projection[:len(projection)-len(strings.TrimLeft(projection, sqlWhitespace))]
	rest := projection[len(leading):]
	if len(rest) < len(distinct) || !strings.EqualFold(rest[:len(distinct)], distinct) {
		return "", projection, true
	}

	afterKeyword := rest[len(distinct):]
	if afterKeyword == "" || !isSQLSpace(afterKeyword[0]) {
		return "", projection, true
	}

	afterTrimmed := strings.TrimLeft(afterKeyword, sqlWhitespace)
	keywordWhitespace := afterKeyword[:len(afterKeyword)-len(afterTrimmed)]

	if !startsWithKeyword(afterTrimmed, keywordON) {
		return leading + distinct + keywordWhitespace, afterTrimmed, true
	}

	onPrefix, onBody, onOK := splitDistinctOnList(afterTrimmed)
	if !onOK {
		return "", "", false
	}
	return leading + distinct + keywordWhitespace + onPrefix, onBody, true
}

// startsWithKeyword reports whether text begins with keyword (case-insensitively) as a
// whole word: the keyword must be followed by whitespace, an opening parenthesis, or end
// of text, so an identifier such as "online" is not mistaken for the keyword "ON".
//
// Takes text (string) which is the text to inspect.
// Takes keyword (string) which is the keyword to match at the start.
//
// Returns bool which is true when text starts with the whole keyword.
func startsWithKeyword(text string, keyword string) bool {
	if len(text) < len(keyword) || !strings.EqualFold(text[:len(keyword)], keyword) {
		return false
	}
	return len(text) == len(keyword) || isSQLSpace(text[len(keyword)]) || text[len(keyword)] == '('
}

// splitDistinctOnList separates a DISTINCT ON key clause.
//
// The text begins with the ON keyword (verified by the caller). The parenthesised key
// list is captured verbatim, including any nested parentheses, so the distinct key is
// kept exactly. It declines (ok false) when the ON keyword is not followed by a balanced
// parenthesised list.
//
// Takes text (string) which begins at the ON keyword.
//
// Returns prefix (string) which is "ON (keys) " (with original spacing) to re-emit
// verbatim.
// Returns body (string) which is the projection text after the key list.
// Returns ok (bool) which is true when a balanced key list was found.
func splitDistinctOnList(text string) (prefix string, body string, ok bool) {
	afterON := text[len(keywordON):]
	parenLeading := afterON[:len(afterON)-len(strings.TrimLeft(afterON, sqlWhitespace))]
	list := strings.TrimLeft(afterON, sqlWhitespace)
	if list == "" || list[0] != '(' {
		return "", "", false
	}

	closeIndex := matchEnclosingParen(list)
	if closeIndex == -1 {
		return "", "", false
	}

	afterList := list[closeIndex+1:]
	noiseLength := leadingNoiseLength(afterList)
	listTrailing := afterList[:noiseLength]
	body = afterList[noiseLength:]
	prefix = keywordON + parenLeading + list[:closeIndex+1] + listTrailing
	return prefix, body, true
}

// leadingNoiseLength returns the leading whitespace-and-comment run length.
//
// It measures the run of SQL whitespace and line or block comments in text, and lets
// splitDistinctOnList carry a comment that sits between a DISTINCT ON key list and the
// projection into the prefix, rather than treating it as the first projection item (which
// would fail the column-reference match and bail).
//
// Takes text (string) which is the text to measure from its start.
//
// Returns int which is the length of the leading whitespace-and-comment run.
func leadingNoiseLength(text string) int {
	position := 0
	for position < len(text) {
		switch {
		case isSQLSpace(text[position]):
			position++
		case text[position] == '-' && position+1 < len(text) && text[position+1] == '-':
			position = skipSQLLineComment(text, position)
		case text[position] == '/' && position+1 < len(text) && text[position+1] == '*':
			position = skipSQLBlockComment(text, position)
		default:
			return position
		}
	}
	return position
}

// matchEnclosingParen returns the index of the parenthesis that closes the one at
// text[0], counting nesting depth and skipping string literals and comments. It returns
// -1 when no balanced closer is found.
//
// Takes text (string) whose first byte is an opening parenthesis.
//
// Returns int which is the index of the matching closing parenthesis, or -1.
func matchEnclosingParen(text string) int {
	depth := 0
	position := 0
	for position < len(text) {
		if next, skipped := advancePastSQLNoise(text, position); skipped {
			position = next
			continue
		}
		switch text[position] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return position
			}
		}
		position++
	}
	return -1
}

// isSQLSpace reports whether b is a SQL whitespace byte.
//
// Takes b (byte) which is the byte to classify.
//
// Returns bool which is true for space, tab, newline, or carriage return.
func isSQLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// splitTopLevelCommas splits a projection into items on commas that sit at parenthesis
// depth zero, skipping string literals and comments so a comma inside a subquery,
// function call, or literal does not split an item.
//
// Takes projection (string) which is the text between SELECT and FROM.
//
// Returns []string which are the projection items in order, each retaining its original
// surrounding whitespace.
func splitTopLevelCommas(projection string) []string {
	var items []string
	depth := 0
	start := 0
	position := 0
	for position < len(projection) {
		if next, skipped := advancePastSQLNoise(projection, position); skipped {
			position = next
			continue
		}
		switch projection[position] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				items = append(items, projection[start:position])
				start = position + 1
			}
		}
		position++
	}
	items = append(items, projection[start:])
	return items
}

// wrapProjectionItem applies jsonFunc to one array-column item.
//
// An item with an explicit AS keeps that alias verbatim (it is the caller's own SQL
// text); a bare or dotted column reference is aliased to the quoted output column name.
// Any other shape (an unaliased expression, an implicit space alias) is declined so the
// caller can fall back.
//
// Takes item (string) which is the raw projection item, possibly with leading whitespace.
// Takes jsonFunc (string) which is the engine's array-to-JSON function.
// Takes quote (func(string) string) which quotes the synthesised output alias.
// Takes outputName (string) which is the alias to use when the item has no explicit AS.
//
// Returns string which is the rewritten item.
// Returns bool which is true when the item could be rewritten safely.
func wrapProjectionItem(item string, jsonFunc string, quote func(string) string, outputName string) (string, bool) {
	trimmed := strings.TrimSpace(item)
	if trimmed == "" {
		return "", false
	}
	leading := item[:len(item)-len(strings.TrimLeft(item, sqlWhitespace))]
	trailing := item[len(strings.TrimRight(item, sqlWhitespace)):]

	var core string
	switch {
	case isWrappableColumnReference(trimmed):
		core = jsonFunc + "(" + trimmed + ") AS " + quote(outputName)
	default:
		asIndex := findTopLevelKeywordIndex(trimmed, 0, "AS")
		if asIndex == -1 {
			return "", false
		}
		expression := strings.TrimSpace(trimmed[:asIndex])
		alias := strings.TrimSpace(trimmed[asIndex+len("AS"):])
		if expression == "" || alias == "" {
			return "", false
		}
		core = jsonFunc + "(" + expression + ") AS " + alias
	}

	return leading + core + trailing, true
}

// isWrappableColumnReference reports a bare or dotted column reference.
//
// Each segment is an unquoted or a double-quoted identifier, the only unaliased
// projection shape the array-JSON wrap rewrites without an explicit AS. It replaces a
// regular expression with the shared lexer identifier primitives so a quoted identifier
// that contains an escaped quote or a dot is classified correctly rather than mis-split.
//
// Takes item (string) which is the trimmed projection item.
//
// Returns bool which is true when item is a whole, well-formed column reference.
func isWrappableColumnReference(item string) bool {
	end, star, ok := scanQualifiedReference(item)
	return ok && !star && end == len(item)
}

// isStarProjectionItem reports whether item is a whole-row star ("*") or a qualified star
// ("alias.*"), the two projection shapes expandStarProjection rebuilds into an explicit
// column list. The qualifier is scanned with the shared lexer identifier primitives.
//
// Takes item (string) which is the trimmed projection item.
//
// Returns bool which is true when item is a star or a qualified star.
func isStarProjectionItem(item string) bool {
	if item == "*" {
		return true
	}
	end, star, ok := scanQualifiedReference(item)
	return ok && star && end == len(item)
}

// scanQualifiedReference scans a dotted reference at the start of text where each segment
// is an unquoted identifier (using the shared IsIdentStart/IsIdentPart classifiers) or a
// double-quoted identifier (using ScanDoubledDelimiter, which honours "" escaping), and
// the dotted tail may be a single "*". It is the lexer-backed replacement for the
// projection-item regexes.
//
// Takes text (string) which is the candidate reference.
//
// Returns end (int) which is the offset consumed.
// Returns star (bool) which is true when the reference ended in ".*".
// Returns ok (bool) which is true when a well-formed reference was scanned to its end.
func scanQualifiedReference(text string) (end int, star bool, ok bool) {
	position := 0
	for {
		segmentEnd, segmentOK := scanIdentifierSegment(text, position)
		if !segmentOK {
			return 0, false, false
		}
		position = segmentEnd
		if position == len(text) {
			return position, false, true
		}
		if text[position] != '.' {
			return 0, false, false
		}
		position++
		if position < len(text) && text[position] == '*' {
			if position+1 != len(text) {
				return 0, false, false
			}
			return position + 1, true, true
		}
	}
}

// scanIdentifierSegment scans one identifier at position: a double-quoted identifier
// (with doubled-quote escaping) via ScanDoubledDelimiter, or an unquoted identifier via
// the shared rune classifiers.
//
// Takes text (string) which is being scanned.
// Takes position (int) which is the offset of the segment start.
//
// Returns int which is the offset after the identifier.
// Returns bool which is true when an identifier was scanned.
func scanIdentifierSegment(text string, position int) (int, bool) {
	if position >= len(text) {
		return 0, false
	}
	if text[position] == '"' {
		_, end, closed := engine_shared.ScanDoubledDelimiter(text, position, '"')
		if !closed {
			return 0, false
		}
		return end, true
	}
	character, width := utf8.DecodeRuneInString(text[position:])
	if !engine_shared.IsIdentStart(character) {
		return 0, false
	}
	position += width
	for position < len(text) {
		character, width = utf8.DecodeRuneInString(text[position:])
		if !engine_shared.IsIdentPart(character) {
			break
		}
		position += width
	}
	return position, true
}
