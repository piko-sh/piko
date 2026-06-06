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

package querier_domain

import (
	"errors"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// countRewriteTokenCapacity is the initial capacity used for the token slice in
	// tokeniseForCountRewrite. Chosen as a small power-of-two that covers the typical query
	// without forcing a reallocation on the first append.
	countRewriteTokenCapacity = 32

	// countRewriteMaxDepth caps the parenthesis depth the count rewriter is willing to
	// follow before declaring the input malformed. Hand-written SQL nests subqueries no more
	// than a handful of levels in practice; capping the depth defends against runaway
	// recursion / pathological inputs that would otherwise let stepCountRewriteToken's depth
	// counter wander without bound.
	countRewriteMaxDepth = 256
)

var (
	// ErrCountRewriteNoSelect is returned by RewriteSelectAsCount when the input SQL does
	// not have a recognisable top-level SELECT keyword. The runtime builder will surface
	// this as a codegen error so the .sql file can be fixed.
	ErrCountRewriteNoSelect = errors.New("count rewrite: input is not a SELECT statement")
)

// RewriteSelectAsCount is the shared SELECT-clause rewriter used by every engine.
//
// It is dialect-neutral because it operates on the tokens common to every supported
// engine: SELECT, FROM, DISTINCT, ORDER BY, LIMIT, OFFSET. Engine-specific keywords such
// as RETURNING and QUALIFY appear after WHERE, GROUP, and HAVING so they remain in place.
//
// The function uses a small hand-rolled tokeniser rather than the per-engine parser
// because the engines do not expose token-position information from their AST nodes and
// because the rewrite cares only about top-level keywords, not full SQL semantics.
//
// Takes originalSQL (string) which is the SELECT statement to rewrite.
// Takes analysis (*querier_dto.RawQueryAnalysis) which carries the structural hints used
// to decide whether wrapping is required.
//
// Returns countSQL (string) which is the rewritten SQL.
// Returns wrapped (bool) which is true when the wrap path was taken.
// Returns error when the input is not a SELECT.
func RewriteSelectAsCount(originalSQL string, analysis *querier_dto.RawQueryAnalysis) (countSQL string, wrapped bool, err error) {
	trimmed := strings.TrimSpace(originalSQL)
	if trimmed == "" {
		return "", false, ErrCountRewriteNoSelect
	}

	tokens, depthOk := tokeniseForCountRewrite(trimmed)
	if !depthOk {
		return "", false, ErrCountRewriteNoSelect
	}
	selectIndex := findTopLevelKeyword(tokens, "SELECT")
	if selectIndex < 0 {
		return "", false, ErrCountRewriteNoSelect
	}

	if shouldWrapForCount(tokens, selectIndex, analysis) {
		return wrapAsCountSubquery(trimmed), true, nil
	}

	rewritten, ok := replaceSelectProjection(trimmed, tokens, selectIndex)
	if !ok {
		return wrapAsCountSubquery(trimmed), true, nil
	}
	return stripTrailingOrderLimitOffset(rewritten), false, nil
}

// sqlToken is a single keyword, identifier, or punctuation token used by the rewriter.
//
// The byte ranges allow the caller to splice the original SQL without losing whitespace
// or comments.
type sqlToken struct {
	// value is the uppercased token text.
	value string

	// start is the byte offset of the token's first byte in the source SQL.
	start int

	// end is the byte offset one past the token's last byte in the source SQL.
	end int

	// depth is the parenthesis nesting depth at which the token appears.
	depth int
}

// tokeniseForCountRewrite walks the SQL and emits one token per keyword or identifier.
//
// It tracks parenthesis depth so the caller can ignore SELECT and FROM inside subqueries
// when looking for the outermost forms. String literals and SQL comments are skipped
// without emitting tokens. The boolean result is false when the running depth ever
// exceeds countRewriteMaxDepth, so the caller aborts with ErrCountRewriteNoSelect instead
// of producing tokens from a pathological input.
//
// Takes sql (string) which is the SQL text to tokenise.
//
// Returns []sqlToken which is the ordered list of keyword and identifier tokens.
// Returns bool which is true when scanning stayed within the depth bound.
func tokeniseForCountRewrite(sql string) ([]sqlToken, bool) {
	tokens := make([]sqlToken, 0, countRewriteTokenCapacity)
	depth := 0
	index := 0
	for index < len(sql) {
		nextIndex, newDepth, identToken, hasToken := stepCountRewriteToken(sql, index, depth)
		if newDepth > countRewriteMaxDepth {
			return nil, false
		}
		if hasToken {
			tokens = append(tokens, identToken)
		}
		index = nextIndex
		depth = newDepth
	}
	return tokens, true
}

// stepCountRewriteToken consumes one logical token or skips one whitespace, comment,
// string, or punctuation run starting at index in sql.
//
// Takes sql (string) which is the SQL text being scanned.
// Takes index (int) which is the byte offset to begin from.
// Takes depth (int) which is the current parenthesis nesting depth.
//
// Returns nextIndex (int) which is the byte offset to resume scanning from.
// Returns newDepth (int) which is the updated parenthesis nesting depth.
// Returns identToken (sqlToken) which is the emitted token when one was produced.
// Returns hasToken (bool) which is true when identToken is valid.
func stepCountRewriteToken(sql string, index, depth int) (nextIndex, newDepth int, identToken sqlToken, hasToken bool) {
	ch := sql[index]
	if isCountRewriteWhitespace(ch) {
		return index + 1, depth, sqlToken{}, false
	}
	if ch == '(' {
		return index + 1, depth + 1, sqlToken{}, false
	}
	if ch == ')' {
		updated := depth
		if updated > 0 {
			updated--
		}
		return index + 1, updated, sqlToken{}, false
	}
	if ch == '-' && index+1 < len(sql) && sql[index+1] == '-' {
		return skipCountRewriteLineComment(sql, index), depth, sqlToken{}, false
	}
	if ch == '/' && index+1 < len(sql) && sql[index+1] == '*' {
		return skipCountRewriteBlockComment(sql, index), depth, sqlToken{}, false
	}
	if ch == '\'' || ch == '"' || ch == '`' {
		return skipCountRewriteString(sql, index), depth, sqlToken{}, false
	}
	if isIdentifierStart(ch) {
		end := index
		for end < len(sql) && isIdentifierPart(sql[end]) {
			end++
		}
		return end, depth, sqlToken{
			value: strings.ToUpper(sql[index:end]),
			start: index,
			end:   end,
			depth: depth,
		}, true
	}

	return index + 1, depth, sqlToken{}, false
}

// isCountRewriteWhitespace reports whether ch is a whitespace byte recognised by the
// count-rewrite tokeniser.
//
// Newline is treated as whitespace because the tokeniser does not need to distinguish
// line boundaries from other gaps.
//
// Takes ch (byte) which is the candidate byte.
//
// Returns bool which is true when ch is whitespace.
func isCountRewriteWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

// skipCountRewriteLineComment advances past a "--" line comment, stopping at the newline
// or end of input.
//
// Takes sql (string) which is the SQL text being scanned.
// Takes index (int) which is the byte offset of the comment start.
//
// Returns int which is the byte offset of the newline or end of input.
func skipCountRewriteLineComment(sql string, index int) int {
	for index < len(sql) && sql[index] != '\n' {
		index++
	}
	return index
}

// skipCountRewriteBlockComment advances past a "/* ... */" block comment, including the
// closing "*/".
//
// When the closer is missing the scanner stops at end of input so the caller terminates
// cleanly.
//
// Takes sql (string) which is the SQL text being scanned.
// Takes index (int) which is the byte offset of the opening "/*".
//
// Returns int which is the byte offset after the closing "*/", or end of input when the
// comment is unterminated.
func skipCountRewriteBlockComment(sql string, index int) int {
	index += 2
	for index+1 < len(sql) && (sql[index] != '*' || sql[index+1] != '/') {
		index++
	}
	if index+1 < len(sql) {
		index += 2
	}
	return index
}

// skipCountRewriteString advances past a single-quoted, double-quoted, or backtick-quoted
// literal.
//
// Backslash escapes and SQL-standard doubled single-quote escapes (a doubled single quote
// inside a single-quoted string) are honoured.
//
// Takes sql (string) which is the SQL text being scanned.
// Takes index (int) which is the byte offset of the opening quote.
//
// Returns int which is the byte offset after the closing quote, or end of input when the
// literal is unterminated.
func skipCountRewriteString(sql string, index int) int {
	quote := sql[index]
	index++
	for index < len(sql) {
		if sql[index] == '\\' && index+1 < len(sql) {
			index += 2
			continue
		}
		if sql[index] == quote {
			if index+1 < len(sql) && sql[index+1] == quote {
				index += 2
				continue
			}
			break
		}
		index++
	}
	if index < len(sql) {
		index++
	}
	return index
}

// findTopLevelKeyword returns the index of the first top-level keyword token matching
// name.
//
// Only tokens at depth zero are considered, and the match is case-insensitive because
// token values are stored uppercased.
//
// Takes tokens ([]sqlToken) which is the token list to search.
// Takes name (string) which is the uppercased keyword to find.
//
// Returns int which is the index of the matching token, or -1 when absent.
func findTopLevelKeyword(tokens []sqlToken, name string) int {
	for index := range tokens {
		if tokens[index].depth == 0 && tokens[index].value == name {
			return index
		}
	}
	return -1
}

// findTopLevelFROM locates the FROM that pairs with the SELECT at selectIndex.
//
// The scan runs forward, and the first depth-zero FROM after the SELECT is the matching
// one because subquery FROM clauses always appear inside parentheses.
//
// Takes tokens ([]sqlToken) which is the token list to search.
// Takes selectIndex (int) which is the index of the paired SELECT token.
//
// Returns int which is the index of the matching FROM token, or -1 when absent.
func findTopLevelFROM(tokens []sqlToken, selectIndex int) int {
	for index := selectIndex + 1; index < len(tokens); index++ {
		if tokens[index].depth == 0 && tokens[index].value == "FROM" {
			return index
		}
	}
	return -1
}

// shouldWrapForCount reports whether the input has a feature that changes count
// semantics.
//
// Wrapping is required to preserve the row-count contract when the query uses GROUP BY,
// DISTINCT in the top-level projection, window functions, or a top-level set operation.
//
// Takes tokens ([]sqlToken) which is the tokenised input.
// Takes selectIndex (int) which is the index of the top-level SELECT token.
// Takes analysis (*querier_dto.RawQueryAnalysis) which carries the structural hints.
//
// Returns bool which is true when the count must be computed by wrapping the query.
func shouldWrapForCount(tokens []sqlToken, selectIndex int, analysis *querier_dto.RawQueryAnalysis) bool {
	if analysis != nil && len(analysis.GroupByColumns) > 0 {
		return true
	}

	if analysis != nil && len(analysis.CompoundBranches) > 0 {
		return true
	}

	if selectIndex+1 < len(tokens) && tokens[selectIndex+1].depth == 0 && tokens[selectIndex+1].value == "DISTINCT" {
		return true
	}

	fromIndex := findTopLevelFROM(tokens, selectIndex)
	end := fromIndex
	if end < 0 {
		end = len(tokens)
	}
	for index := selectIndex + 1; index < end; index++ {
		if tokens[index].depth == 0 && tokens[index].value == "OVER" {
			return true
		}
	}

	return false
}

// replaceSelectProjection swaps the projection list between the top-level SELECT and its
// paired FROM for a "COUNT(*)" expression.
//
// Takes sql (string) which is the source SQL to rewrite.
// Takes tokens ([]sqlToken) which is the tokenised input.
// Takes selectIndex (int) which is the index of the top-level SELECT token.
//
// Returns string which is the rewritten SQL when the replacement succeeded.
// Returns bool which is true when the projection was replaced.
func replaceSelectProjection(sql string, tokens []sqlToken, selectIndex int) (string, bool) {
	fromIndex := findTopLevelFROM(tokens, selectIndex)
	if fromIndex < 0 {
		return "", false
	}
	selectEnd := tokens[selectIndex].end
	fromStart := tokens[fromIndex].start
	return sql[:selectEnd] + " COUNT(*) " + sql[fromStart:], true
}

// stripTrailingOrderLimitOffset removes any top-level ORDER BY, LIMIT, or OFFSET tail
// from the rewritten count SQL.
//
// These clauses do not affect the row count, so dropping them keeps the output clear and
// avoids the engine performing redundant ordering work.
//
// Takes sql (string) which is the count SQL to trim.
//
// Returns string which is the SQL with the trailing ordering and pagination removed.
func stripTrailingOrderLimitOffset(sql string) string {
	tokens, depthOk := tokeniseForCountRewrite(sql)
	if !depthOk {
		return strings.TrimRight(sql, " \t\r\n;")
	}

	boundary := findTopLevelKeyword(tokens, "SELECT")
	for index := range tokens {
		if tokens[index].depth != 0 {
			continue
		}
		switch tokens[index].value {
		case "FROM", "UNION", "INTERSECT", "EXCEPT":
			if index > boundary {
				boundary = index
			}
		}
	}

	cut := len(sql)
	for index := boundary + 1; index < len(tokens); index++ {
		if tokens[index].depth != 0 {
			continue
		}
		if value := tokens[index].value; value == "ORDER" || value == "LIMIT" || value == "OFFSET" {
			cut = tokens[index].start
			break
		}
	}
	return strings.TrimRight(sql[:cut], " \t\r\n;")
}

// wrapAsCountSubquery is the safe fallback for queries that GROUP BY, SELECT DISTINCT, or
// use window functions.
//
// The inner query has its trailing ordering and pagination stripped because those
// operations have no effect on the count.
//
// Takes sql (string) which is the query to wrap.
//
// Returns string which is a "SELECT COUNT(*)" over the wrapped inner query.
func wrapAsCountSubquery(sql string) string {
	inner := stripTrailingOrderLimitOffset(sql)
	return "SELECT COUNT(*) FROM (" + inner + ") sub"
}

// isIdentifierStart reports whether c can begin a SQL identifier.
//
// The check is restricted to ASCII letters and underscore because SQL keywords are ASCII,
// unquoted UTF-8 identifiers are not valid SQL, and quoted UTF-8 identifiers take the
// string-scan branch instead. A rune-based unicode.IsLetter check would misclassify UTF-8
// lead bytes in the 0x80 to 0xFF range as letters and split multi-byte identifiers.
//
// Takes c (byte) which is the candidate byte.
//
// Returns bool which is true when c can start an identifier.
func isIdentifierStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// isIdentifierPart reports whether c can continue a SQL identifier.
//
// The check is restricted to ASCII for the same reason as isIdentifierStart.
//
// Takes c (byte) which is the candidate byte.
//
// Returns bool which is true when c can continue an identifier.
func isIdentifierPart(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}
