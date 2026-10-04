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

package engine_shared_test

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

func TestParenthesisScanIndexEnclosingParen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		tokens   string
		position int
		want     int
	}{
		{name: "parameter directly inside an IN list", tokens: "x IN ( $ )", position: 3, want: 2},
		{name: "later IN-list element", tokens: "x IN ( $ , $ , $ )", position: 7, want: 2},
		{name: "nested group is skipped", tokens: "f ( ( a ) , $ )", position: 6, want: 1},
		{name: "innermost group wins", tokens: "f ( g ( $ ) )", position: 4, want: 3},
		{name: "top level has no enclosing group", tokens: "a = $", position: 2, want: -1},
		{name: "boundary at the same level stops the search", tokens: "( a AND $ )", position: 3, want: -1},
		{name: "boundary inside a nested group does not stop the search", tokens: "( ( a AND b ) , $ )", position: 7, want: 0},
		{name: "position past the last token", tokens: "( a", position: 2, want: 0},
		{name: "negative position", tokens: "( a )", position: -1, want: -1},
		{name: "position beyond the statement", tokens: "( a )", position: 9, want: -1},
		{name: "stray closing paren consumes the earlier group", tokens: "( a ) ) $", position: 4, want: -1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			classes := classifyWords(testCase.tokens)
			index := engine_shared.NewParenthesisScanIndex(len(classes), classifierFor(classes))

			assert.Equal(t, testCase.want, index.EnclosingParen(testCase.position))
		})
	}
}

func TestParenthesisScanIndexEnclosingLikeOperator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		tokens    string
		position  int
		wantIndex int
		wantFound bool
	}{
		{name: "pattern directly after LIKE", tokens: "x LIKE $", position: 2, wantIndex: 1, wantFound: true},
		{name: "pattern inside a function call", tokens: "x LIKE f ( a , $ , b )", position: 6, wantIndex: 1, wantFound: true},
		{name: "boundary between operator and parameter", tokens: "x LIKE y AND z = $", position: 6, wantFound: false},
		{name: "no operator at all", tokens: "x IN ( $ , $ )", position: 5, wantFound: false},
		{name: "operator nested deeper is ignored", tokens: "( x LIKE y ) = $", position: 6, wantFound: false},
		{name: "sibling group one level deeper is visible after leaving the group", tokens: "x LIKE f ( a ) || ( $ )", position: 8, wantIndex: 1, wantFound: true},
		{name: "position beyond the statement", tokens: "x LIKE $", position: 7, wantFound: false},
		{name: "negative position", tokens: "x LIKE $", position: -2, wantFound: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			classes := classifyWords(testCase.tokens)
			index := engine_shared.NewParenthesisScanIndex(len(classes), classifierFor(classes))

			gotIndex, gotFound := index.EnclosingLikeOperator(testCase.position)
			assert.Equal(t, testCase.wantFound, gotFound)
			if testCase.wantFound {
				assert.Equal(t, testCase.wantIndex, gotIndex)
			}
		})
	}
}

func TestParenthesisScanIndexMatchesBackwardWalk(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewPCG(1, 2))
	classPool := []engine_shared.TokenClass{
		engine_shared.TokenClassLeftParen,
		engine_shared.TokenClassRightParen,
		engine_shared.TokenClassBoundary,
		engine_shared.TokenClassPattern,
		engine_shared.TokenClassOther,
		engine_shared.TokenClassOther,
	}

	for range 2000 {
		classes := make([]engine_shared.TokenClass, random.IntN(40))
		for position := range classes {
			classes[position] = classPool[random.IntN(len(classPool))]
		}
		index := engine_shared.NewParenthesisScanIndex(len(classes), classifierFor(classes))

		for position := 0; position <= len(classes); position++ {
			require.Equal(t, backwardEnclosingParen(classes, position), index.EnclosingParen(position),
				"enclosing paren for %v at %d", classes, position)

			wantIndex, wantFound := backwardEnclosingLikeOperator(classes, position)
			gotIndex, gotFound := index.EnclosingLikeOperator(position)
			require.Equal(t, wantFound, gotFound, "like operator for %v at %d", classes, position)
			require.Equal(t, wantIndex, gotIndex, "like operator for %v at %d", classes, position)
		}
	}
}

func TestParenthesisScanIndexHandlesLongInLists(t *testing.T) {
	t.Parallel()

	const parameterCount = 100_000

	var builder strings.Builder
	builder.WriteString("x IN (")
	for range parameterCount {
		builder.WriteString(" $ ,")
	}
	builder.WriteString(" $ )")
	classes := classifyWords(builder.String())

	index := engine_shared.NewParenthesisScanIndex(len(classes), classifierFor(classes))

	lastParameter := len(classes) - 2
	assert.Equal(t, 2, index.EnclosingParen(lastParameter))
	_, found := index.EnclosingLikeOperator(lastParameter)
	assert.False(t, found)
}

func TestNewParenthesisScanIndexWithNoTokens(t *testing.T) {
	t.Parallel()

	index := engine_shared.NewParenthesisScanIndex(0, classifierFor(nil))

	assert.Equal(t, -1, index.EnclosingParen(0))
	_, found := index.EnclosingLikeOperator(0)
	assert.False(t, found)
}

func TestIsClauseBoundaryKeyword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		keyword string
		want    bool
	}{
		{keyword: "AND", want: true},
		{keyword: "WHERE", want: true},
		{keyword: "ESCAPE", want: true},
		{keyword: "IN", want: false},
		{keyword: "and", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.keyword, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, engine_shared.IsClauseBoundaryKeyword(testCase.keyword))
		})
	}
}

func BenchmarkParenthesisScanIndexInList(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("x IN (")
	for range 10_000 {
		builder.WriteString(" $ ,")
	}
	builder.WriteString(" $ )")
	classes := classifyWords(builder.String())
	classify := classifierFor(classes)

	for b.Loop() {
		index := engine_shared.NewParenthesisScanIndex(len(classes), classify)
		for position := range classes {
			_ = index.EnclosingParen(position)
			_, _ = index.EnclosingLikeOperator(position)
		}
	}
}

func classifyWords(sketch string) []engine_shared.TokenClass {
	words := strings.Fields(sketch)
	classes := make([]engine_shared.TokenClass, len(words))
	for position, word := range words {
		switch {
		case word == "(":
			classes[position] = engine_shared.TokenClassLeftParen
		case word == ")":
			classes[position] = engine_shared.TokenClassRightParen
		case word == "LIKE":
			classes[position] = engine_shared.TokenClassPattern
		case engine_shared.IsClauseBoundaryKeyword(word):
			classes[position] = engine_shared.TokenClassBoundary
		default:
			classes[position] = engine_shared.TokenClassOther
		}
	}
	return classes
}

func classifierFor(classes []engine_shared.TokenClass) func(int) engine_shared.TokenClass {
	return func(position int) engine_shared.TokenClass {
		return classes[position]
	}
}

func backwardEnclosingParen(classes []engine_shared.TokenClass, start int) int {
	if start < 0 || start > len(classes) {
		return -1
	}
	depth := 0
	for position := start - 1; position >= 0; position-- {
		switch {
		case classes[position] == engine_shared.TokenClassRightParen:
			depth++
		case classes[position] == engine_shared.TokenClassLeftParen:
			if depth == 0 {
				return position
			}
			depth--
		case depth == 0 && classes[position] == engine_shared.TokenClassBoundary:
			return -1
		}
	}
	return -1
}

func backwardEnclosingLikeOperator(classes []engine_shared.TokenClass, start int) (int, bool) {
	if start < 0 || start > len(classes) {
		return 0, false
	}
	depth := 0
	for position := start - 1; position >= 0; position-- {
		switch {
		case classes[position] == engine_shared.TokenClassRightParen:
			depth++
		case classes[position] == engine_shared.TokenClassLeftParen:
			depth--
		case depth <= 0 && classes[position] == engine_shared.TokenClassBoundary:
			return 0, false
		case depth <= 0 && classes[position] == engine_shared.TokenClassPattern:
			return position, true
		}
	}
	return 0, false
}
