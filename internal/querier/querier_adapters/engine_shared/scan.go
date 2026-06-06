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

// Package engine_shared holds dialect-agnostic plumbing shared by the SQL engine
// adapters.
//
// It is a toolkit, not a framework. Each engine calls these helpers and supplies its own
// dialect decisions through predicates and configuration, so the shared code never
// branches on a dialect. It lives at the adapter tier (alongside emitter_shared) and
// imports only querier_dto, so any engine module can use it without an import cycle.
package engine_shared

var (
	// clauseBoundaryKeywords is the SQL-standard clause and boolean-connector vocabulary
	// shared by every dialect; an upper-cased keyword in this set marks a clause or
	// expression boundary that a backwards parameter-context scan must not cross.
	clauseBoundaryKeywords = map[string]struct{}{
		"AND":       {},
		"OR":        {},
		"WHERE":     {},
		"HAVING":    {},
		"ON":        {},
		"WHEN":      {},
		"THEN":      {},
		"ELSE":      {},
		"CASE":      {},
		"FROM":      {},
		"GROUP":     {},
		"ORDER":     {},
		"LIMIT":     {},
		"OFFSET":    {},
		"RETURNING": {},
		"UNION":     {},
		"INTERSECT": {},
		"EXCEPT":    {},
		"BY":        {},
		"SELECT":    {},
		"INSERT":    {},
		"UPDATE":    {},
		"DELETE":    {},
		"VALUES":    {},
		"SET":       {},
		"ESCAPE":    {},
	}
)

// FindEnclosingParen scans backwards from start-1 for the unmatched left parenthesis that
// encloses the token at start, tracking parenthesis nesting depth. The caller injects the
// dialect-specific classification so this loop carries no token-kind knowledge of its
// own.
//
// Takes start (int) which is the index whose enclosing parenthesis is sought.
// Takes isLeftParen (func(int) bool) which reports whether the token at an index is a
// "(".
// Takes isRightParen (func(int) bool) which reports whether the token at an index is a
// ")".
// Takes isBoundary (func(int) bool) which reports a depth-0 token past which the search
// must stop and report "no enclosing parenthesis"; engines that scan unbounded pass a
// predicate that always returns false.
//
// Returns int which is the index of the enclosing left parenthesis, or -1 when none is
// found (or a boundary token is reached first at depth 0).
func FindEnclosingParen(start int, isLeftParen, isRightParen, isBoundary func(index int) bool) int {
	depth := 0
	for index := start - 1; index >= 0; index-- {
		switch {
		case isRightParen(index):
			depth++
		case isLeftParen(index):
			if depth == 0 {
				return index
			}
			depth--
		case depth == 0 && isBoundary(index):
			return -1
		}
	}
	return -1
}

// FindEnclosingLikeOperator scans backwards from start-1 for a LIKE-family pattern
// operator the parameter at start is an operand of.
//
// It tracks parenthesis depth so operators nested in an unrelated paren are skipped. The
// caller injects every classification (including its own dialect LIKE-family set via
// isPattern), so this loop carries no dialect knowledge. The scan stops, reporting "not
// found", at a clause boundary.
//
// Takes start (int) which is the parameter index the operator is sought for.
// Takes isLeftParen / isRightParen (func(int) bool) which classify parenthesis tokens.
// Takes isBoundary (func(int) bool) which reports a depth-0 clause boundary ending the
// scan.
// Takes isPattern (func(int) bool) which reports a depth-0 LIKE-family pattern operator.
//
// Returns int which is the operator's token index when found.
// Returns bool which is true when a pattern operator was located.
func FindEnclosingLikeOperator(start int, isLeftParen, isRightParen, isBoundary, isPattern func(index int) bool) (int, bool) {
	depth := 0
	for index := start - 1; index >= 0; index-- {
		switch {
		case isRightParen(index):
			depth++
		case isLeftParen(index):
			depth--
		case depth <= 0 && isBoundary(index):
			return 0, false
		case depth <= 0 && isPattern(index):
			return index, true
		}
	}
	return 0, false
}

// IsClauseBoundaryKeyword reports whether an upper-cased SQL keyword marks a clause or
// expression boundary that a backwards parameter-context scan must not cross.
//
// Crossing one at parenthesis depth zero means the scan has left the parameter's
// immediate expression, so the parameter is not enclosed by an IN-list or LIKE paren. The
// keyword set is the SQL-standard clause and boolean-connector vocabulary shared by every
// dialect.
//
// Takes keyword (string) which is the candidate keyword, already upper-cased by the
// caller.
//
// Returns bool which is true when the keyword is a clause or expression boundary.
func IsClauseBoundaryKeyword(keyword string) bool {
	_, ok := clauseBoundaryKeywords[keyword]
	return ok
}
