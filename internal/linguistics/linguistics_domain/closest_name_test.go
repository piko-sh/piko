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

package linguistics_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneLevenshteinDistance(t *testing.T) {
	testCases := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{

		{name: "Identical strings", s1: "hello", s2: "hello", expected: 0},
		{name: "Single substitution", s1: "kitten", s2: "sitten", expected: 1},
		{name: "Single insertion", s1: "sit", s2: "sits", expected: 1},
		{name: "Single deletion", s1: "saturday", s2: "saurday", expected: 1},
		{name: "Multiple edits", s1: "saturday", s2: "sunday", expected: 3},
		{name: "Transposition (costs 2)", s1: "martha", s2: "marhta", expected: 2},

		{name: "One string is empty", s1: "hello", s2: "", expected: 5},
		{name: "Other string is empty", s1: "", s2: "world", expected: 5},
		{name: "Both strings are empty", s1: "", s2: "", expected: 0},
		{name: "One is a prefix of the other", s1: "test", s2: "testing", expected: 3},

		{name: "Different case", s1: "Hello", s2: "hello", expected: 1},
		{name: "All different case", s1: "WORLD", s2: "world", expected: 5},

		{name: "Identical unicode", s1: "你好", s2: "你好", expected: 0},
		{name: "Single unicode insertion", s1: "你好世界", s2: "你好世界a", expected: 1},
		{name: "Mixed ascii and unicode", s1: "hello你好", s2: "hellö你好", expected: 1},
		{name: "Complex unicode edits", s1: " résumé", s2: "resume", expected: 3},
		{name: "Emoji", s1: "😄", s2: "😃", expected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := runeLevenshteinDistance(tc.s1, tc.s2)
			assert.Equal(t, tc.expected, actual, "Distance between '%s' and '%s' should be %d", tc.s1, tc.s2, tc.expected)
		})
	}
}

func TestClosestName(t *testing.T) {
	t.Parallel()

	candidates := []string{"title", "subtitle", "task-list", "quote"}
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "one letter swapped", input: "titel", want: "title"},
		{name: "case is ignored", input: "TITLE", want: "title"},
		{name: "missing letter", input: "task-lst", want: "task-list"},
		{name: "too different", input: "banner", want: ""},
		{name: "empty input", input: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, ClosestName(testCase.input, candidates))
		})
	}
}

func TestClosestName_NoCandidates(t *testing.T) {
	t.Parallel()

	assert.Empty(t, ClosestName("title", nil))
}
