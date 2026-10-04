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

import (
	"cmp"
	"slices"
)

const (
	// TokenClassOther marks a token with no role in the scans.
	TokenClassOther TokenClass = iota

	// TokenClassLeftParen marks a "(" token.
	TokenClassLeftParen

	// TokenClassRightParen marks a ")" token.
	TokenClassRightParen

	// TokenClassBoundary marks a clause or boolean-connector keyword that a backward scan
	// must not cross at its own nesting level.
	TokenClassBoundary

	// TokenClassPattern marks a LIKE-family pattern operator keyword.
	TokenClassPattern
)

const (
	// notFound is the sentinel index for a scan that located nothing.
	notFound = -1
)

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

// TokenClass is the role a token plays in the parenthesis-aware backward scans that infer
// a parameter's context.
type TokenClass uint8

// ParenthesisScanIndex answers, for any token position of one statement, which "(" opens
// the group enclosing it and which LIKE-family operator it is a pattern operand of.
//
// Both answers are computed for every position in a single linear pass, so a statement
// with n parameters costs O(n log n) in total rather than the O(n^2) of one backward walk
// per parameter (an IN list of 100,000 parameters otherwise takes close to a minute). The
// answers match a backward walk exactly, including on unbalanced input.
type ParenthesisScanIndex struct {
	// enclosingParens holds, per position, the index of the enclosing "(" or notFound.
	enclosingParens []int

	// likeOperators holds, per position, the index of the enclosing LIKE-family operator or
	// notFound.
	likeOperators []int
}

// scanCandidate is a boundary or pattern token kept on the monotonic stack used to find
// the nearest such token at or above a nesting level.
type scanCandidate struct {
	// index is the token's position.
	index int

	// level is the unclamped parenthesis depth in front of the token.
	level int

	// isPattern reports whether the token is a pattern operator rather than a boundary.
	isPattern bool
}

// parenthesisScanBuilder carries the running state of the single pass that fills a
// ParenthesisScanIndex.
type parenthesisScanBuilder struct {
	// latestOpenParen maps an offset level to the index of the latest "(" opened from that
	// level, or notFound.
	latestOpenParen []int

	// latestBoundary maps an offset level to the index of the latest boundary token at that
	// level, or notFound.
	latestBoundary []int

	// candidates is the monotonic stack of boundary and pattern tokens, ordered so that
	// levels strictly increase from the bottom to the top.
	candidates []scanCandidate

	// levelOffset shifts the unclamped depth, which can go negative on unbalanced input,
	// into a valid slice index.
	levelOffset int
}

// NewParenthesisScanIndex classifies every token once and precomputes the enclosing "("
// and enclosing LIKE-family operator for every position from 0 to tokenCount inclusive.
//
// Takes tokenCount (int) which is the number of tokens in the statement.
// Takes classify (func(int) TokenClass) which reports the role of the token at an index;
// the engine supplies its own keyword sets so the index carries no dialect knowledge.
//
// Returns *ParenthesisScanIndex which answers lookups in constant time.
func NewParenthesisScanIndex(tokenCount int, classify func(index int) TokenClass) *ParenthesisScanIndex {
	tokenCount = max(tokenCount, 0)
	index := &ParenthesisScanIndex{
		enclosingParens: make([]int, tokenCount+1),
		likeOperators:   make([]int, tokenCount+1),
	}
	builder := newParenthesisScanBuilder(tokenCount)
	level := 0
	for position := 0; position <= tokenCount; position++ {
		index.enclosingParens[position] = builder.enclosingParen(level)
		index.likeOperators[position] = builder.likeOperator(level)
		if position == tokenCount {
			break
		}
		level = builder.record(position, level, classify(position))
	}
	return index
}

// EnclosingParen returns the "(" that opens the group enclosing position.
//
// A clause-boundary keyword at the position's own nesting level, found before the "(",
// means the position is not enclosed by that group, matching a backward walk that stops
// at the boundary.
//
// Takes position (int) which is the token index whose enclosing group is sought.
//
// Returns int which is the index of the enclosing "(", or -1 when there is none, a
// boundary intervenes, or position is out of range.
func (s *ParenthesisScanIndex) EnclosingParen(position int) int {
	if position < 0 || position >= len(s.enclosingParens) {
		return notFound
	}
	return s.enclosingParens[position]
}

// EnclosingLikeOperator returns the LIKE-family operator the token at position is a
// pattern operand of.
//
// The search walks outwards through enclosing groups and stops at the nearest boundary
// keyword not nested deeper than position, matching a backward walk that tracks
// parenthesis depth.
//
// Takes position (int) which is the parameter's token index.
//
// Returns int which is the operator's token index when found.
// Returns bool which is true when a pattern operator was located.
func (s *ParenthesisScanIndex) EnclosingLikeOperator(position int) (int, bool) {
	if position < 0 || position >= len(s.likeOperators) {
		return 0, false
	}
	operator := s.likeOperators[position]
	if operator == notFound {
		return 0, false
	}
	return operator, true
}

// enclosingParen answers the enclosing-group query for the current position.
//
// The nearest earlier position whose depth is below level is necessarily the "(" opened
// from level-1, and the group encloses the position unless a boundary at level appears
// after that "(".
//
// Takes level (int) which is the unclamped depth in front of the current position.
//
// Returns int which is the enclosing "(" index or notFound.
func (b *parenthesisScanBuilder) enclosingParen(level int) int {
	parentSlot := level - 1 + b.levelOffset
	if parentSlot < 0 {
		return notFound
	}
	openParen := b.latestOpenParen[parentSlot]
	if openParen == notFound || b.latestBoundary[level+b.levelOffset] > openParen {
		return notFound
	}
	return openParen
}

// likeOperator answers the enclosing-LIKE query for the current position using the
// nearest earlier boundary or pattern token whose depth is at most level to decide the
// result.
//
// Levels strictly increase towards the top of the candidate stack, so the eligible
// candidates form a prefix of the stack and the nearest of them is found by binary
// search.
//
// Takes level (int) which is the unclamped depth in front of the current position.
//
// Returns int which is the pattern operator's index, or notFound when the nearest
// eligible token is a boundary or there is none.
func (b *parenthesisScanBuilder) likeOperator(level int) int {
	eligible, _ := slices.BinarySearchFunc(b.candidates, level+1, compareCandidateLevel)
	if eligible == 0 {
		return notFound
	}
	nearest := b.candidates[eligible-1]
	if !nearest.isPattern {
		return notFound
	}
	return nearest.index
}

// record folds the token at position into the running state and returns the depth in
// front of the next token.
//
// Takes position (int) which is the token's index.
// Takes level (int) which is the unclamped depth in front of the token.
// Takes class (TokenClass) which is the token's role.
//
// Returns int which is the depth after the token.
func (b *parenthesisScanBuilder) record(position int, level int, class TokenClass) int {
	switch class {
	case TokenClassLeftParen:
		b.latestOpenParen[level+b.levelOffset] = position
		return level + 1
	case TokenClassRightParen:
		return level - 1
	case TokenClassBoundary:
		b.latestBoundary[level+b.levelOffset] = position
		b.pushCandidate(scanCandidate{index: position, level: level, isPattern: false})
	case TokenClassPattern:
		b.pushCandidate(scanCandidate{index: position, level: level, isPattern: true})
	case TokenClassOther:
	}
	return level
}

// pushCandidate adds a boundary or pattern token to the monotonic stack, first dropping
// every candidate at the same or a deeper level because the new, nearer token shadows
// them for every later query.
//
// Takes candidate (scanCandidate) which is the token to add.
func (b *parenthesisScanBuilder) pushCandidate(candidate scanCandidate) {
	for len(b.candidates) > 0 && b.candidates[len(b.candidates)-1].level >= candidate.level {
		b.candidates = b.candidates[:len(b.candidates)-1]
	}
	b.candidates = append(b.candidates, candidate)
}

// newParenthesisScanBuilder sizes the level tables for a statement of tokenCount tokens,
// whose depth can range from -tokenCount to tokenCount.
//
// Takes tokenCount (int) which is the number of tokens in the statement.
//
// Returns *parenthesisScanBuilder which is ready for the pass.
func newParenthesisScanBuilder(tokenCount int) *parenthesisScanBuilder {
	levelCount := 2*tokenCount + 1
	builder := &parenthesisScanBuilder{
		latestOpenParen: make([]int, levelCount),
		latestBoundary:  make([]int, levelCount),
		candidates:      nil,
		levelOffset:     tokenCount,
	}
	for level := range levelCount {
		builder.latestOpenParen[level] = notFound
		builder.latestBoundary[level] = notFound
	}
	return builder
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

// compareCandidateLevel orders a candidate's level against a target level for the binary
// search over the monotonic stack.
//
// Takes candidate (scanCandidate) which is the stack entry.
// Takes target (int) which is the level being searched for.
//
// Returns int which is negative, zero, or positive as the candidate's level is below,
// equal to, or above target.
func compareCandidateLevel(candidate scanCandidate, target int) int {
	return cmp.Compare(candidate.level, target)
}
