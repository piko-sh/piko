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
)

func TestFuzzyMatch_ExactMatch(t *testing.T) {
	matched, score := FuzzyMatch("hello", "hello", 0.5, false)

	if !matched {
		t.Error("Exact match should return true")
	}
	if score != 1.0 {
		t.Errorf("Exact match score should be 1.0, got %.2f", score)
	}
}

func TestFuzzyMatch_CaseSensitive(t *testing.T) {

	matched, _ := FuzzyMatch("Hello", "hello", 0.5, false)
	if !matched {
		t.Error("Case-insensitive should match")
	}

	matched, _ = FuzzyMatch("Hello", "ell", 0.5, true)
	if !matched {
		t.Error("Case-sensitive substring 'ell' should match in 'Hello'")
	}

	matched, score := FuzzyMatch("Hello", "hello", 0.9, true)
	if matched {
		t.Errorf("Case-sensitive exact should NOT match (got score %.2f)", score)
	}
}

func TestFuzzyMatch_SubstringMatch(t *testing.T) {
	matched, score := FuzzyMatch("configuration guide", "config", 0.3, false)

	if !matched {
		t.Error("Substring 'config' should match in 'configuration guide'")
	}

	if score < scoreExactSubstring {
		t.Errorf("Substring match score should be >= %.2f, got %.2f", scoreExactSubstring, score)
	}

	t.Logf("Substring match score: %.2f", score)
}

func TestFuzzyMatch_PrefixMatch(t *testing.T) {
	matched, score := FuzzyMatch("configuration", "config", 0.5, false)

	if !matched {
		t.Error("Prefix 'config' should match 'configuration'")
	}

	if score < scoreExactSubstring {
		t.Errorf("Match score should be >= %.2f, got %.2f", scoreExactSubstring, score)
	}

	t.Logf("Prefix match score: %.2f (substring strategy matched first)", score)
}

func TestFuzzyMatch_WordLevelMatch(t *testing.T) {
	testCases := []struct {
		text        string
		pattern     string
		shouldMatch bool
		minScore    float64
	}{

		{text: "user identification number", pattern: "id", shouldMatch: true, minScore: scoreExactSubstring},
		{text: "configuration file", pattern: "config", shouldMatch: true, minScore: scoreExactSubstring},
		{text: "test data", pattern: "test", shouldMatch: true, minScore: scoreExactSubstring},
	}

	for _, tc := range testCases {
		t.Run(tc.pattern, func(t *testing.T) {
			matched, score := FuzzyMatch(tc.text, tc.pattern, 0.3, false)

			if matched != tc.shouldMatch {
				t.Errorf("FuzzyMatch(%q, %q) matched=%v, want %v", tc.text, tc.pattern, matched, tc.shouldMatch)
			}

			if matched && score < tc.minScore {
				t.Errorf("Score %.2f below minimum %.2f", score, tc.minScore)
			}

			t.Logf("%q in %q: matched=%v, score=%.2f", tc.pattern, tc.text, matched, score)
		})
	}
}

func TestFuzzyMatch_LevenshteinFallback(t *testing.T) {

	matched, score := FuzzyMatch("configuration", "konfig", 0.3, false)

	if !matched {
		t.Error("Fuzzy match via Levenshtein should succeed")
	}

	t.Logf("Levenshtein fuzzy score: %.2f", score)
}

func TestFuzzyMatch_Threshold(t *testing.T) {
	text := "configuration"
	pattern := "config"

	matched, _ := FuzzyMatch(text, pattern, 0.3, false)
	if !matched {
		t.Error("Should match with low threshold 0.3")
	}

	matched, _ = FuzzyMatch(text, pattern, 0.99, false)
	if matched {
		t.Error("Should NOT match with high threshold 0.99")
	}
}

func TestFuzzyMatch_EmptyInputs(t *testing.T) {

	matched, score := FuzzyMatch("text", "", 0.5, false)
	if !matched || score != 1.0 {
		t.Error("Empty pattern should match with score 1.0")
	}

	matched, score = FuzzyMatch("", "pattern", 0.5, false)
	if matched || score != 0.0 {
		t.Error("Empty text should NOT match non-empty pattern")
	}

	matched, score = FuzzyMatch("", "", 0.5, false)
	if !matched || score != 1.0 {
		t.Error("Both empty should match with score 1.0")
	}
}

func TestFuzzyMatch_UTF8(t *testing.T) {
	testCases := []struct {
		text    string
		pattern string
	}{
		{text: "café résumé", pattern: "café"},
		{text: "你好世界", pattern: "你好"},
		{text: "Привет мир", pattern: "Привет"},
	}

	for _, tc := range testCases {
		t.Run(tc.pattern, func(t *testing.T) {
			matched, score := FuzzyMatch(tc.text, tc.pattern, 0.5, false)

			if !matched {
				t.Errorf("Should match UTF-8 pattern %q in %q", tc.pattern, tc.text)
			}

			t.Logf("UTF-8 match: %q in %q, score=%.2f", tc.pattern, tc.text, score)
		})
	}
}

func TestFuzzyMatch_HyphenatedWords(t *testing.T) {

	matched, score := FuzzyMatch("api-key config", "key", 0.5, false)

	if !matched {
		t.Error("Should find 'key' in 'api-key' via word matching")
	}

	t.Logf("Found 'key' in 'api-key' with score %.2f", score)
}

func TestNormaliseForMatching(t *testing.T) {

	text, pattern := normaliseForMatching("Hello", "WORLD", false)
	if text != "hello" || pattern != "world" {
		t.Errorf("Case-insensitive: got (%q, %q), want ('hello', 'world')", text, pattern)
	}

	text, pattern = normaliseForMatching("Hello", "WORLD", true)
	if text != "Hello" || pattern != "WORLD" {
		t.Errorf("Case-sensitive: got (%q, %q), want ('Hello', 'WORLD')", text, pattern)
	}
}

func TestTryFastMatch(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		pattern  string
		expected float64
		matched  bool
	}{
		{name: "exact", text: "test", pattern: "test", expected: 1.0, matched: true},
		{name: "substring", text: "testing", pattern: "test", expected: scoreExactSubstring, matched: true},
		{name: "prefix", text: "configuration", pattern: "config", expected: scoreExactSubstring, matched: true},
		{name: "no match", text: "hello", pattern: "world", expected: 0.0, matched: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, matched := tryFastMatch(tc.text, tc.pattern)

			if matched != tc.matched {
				t.Errorf("matched = %v, want %v", matched, tc.matched)
			}

			if matched && score < tc.expected-0.01 {
				t.Errorf("score = %.2f, want >= %.2f", score, tc.expected)
			}
		})
	}
}

func TestScoreWordMatches(t *testing.T) {
	testCases := []struct {
		name         string
		textWords    []string
		patternWords []string
		minScore     float64
	}{
		{
			name:         "exact word match",
			textWords:    []string{"hello", "world"},
			patternWords: []string{"world"},
			minScore:     scoreWordExactMatch,
		},
		{
			name:         "prefix word match",
			textWords:    []string{"configuration", "file"},
			patternWords: []string{"config"},
			minScore:     scoreWordPrefixMatch,
		},
		{
			name:         "contains word match",
			textWords:    []string{"identification"},
			patternWords: []string{"id"},
			minScore:     scoreWordContainsMatch,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scoreWordMatches(tc.textWords, tc.patternWords)

			if score < tc.minScore {
				t.Errorf("Score %.2f below minimum %.2f", score, tc.minScore)
			}

			t.Logf("%s: score=%.2f", tc.name, score)
		})
	}
}

func TestCalculateLevenshteinScore(t *testing.T) {
	testCases := []struct {
		text     string
		pattern  string
		minScore float64
	}{
		{text: "test", pattern: "test", minScore: 1.0},
		{text: "test", pattern: "tests", minScore: 0.8},
		{text: "configuration", pattern: "configurtion", minScore: 0.9},
	}

	for _, tc := range testCases {
		score := calculateLevenshteinScore(tc.text, tc.pattern)

		if score < tc.minScore {
			t.Errorf("Score for %q vs %q = %.2f, want >= %.2f", tc.text, tc.pattern, score, tc.minScore)
		}

		t.Logf("%q vs %q: score=%.2f", tc.text, tc.pattern, score)
	}
}

func TestSplitIntoWords(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{input: "hello world", expected: []string{"hello", "world"}},
		{input: "api-key", expected: []string{"api-key"}},
		{input: "user_id", expected: []string{"user_id"}},
		{input: "test!!", expected: []string{"test"}},
		{input: "  spaced  ", expected: []string{"spaced"}},
		{input: "", expected: []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := splitIntoWords(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d words, got %d: %v", len(tc.expected), len(result), result)
			}

			for i, exp := range tc.expected {
				if i >= len(result) {
					break
				}
				if result[i] != exp {
					t.Errorf("Word[%d]: expected %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func BenchmarkFuzzyMatch(b *testing.B) {
	text := "The quick brown fox jumps over the lazy dog configuration"
	pattern := "config"

	b.ResetTimer()
	for b.Loop() {
		FuzzyMatch(text, pattern, 0.3, false)
	}
}

func BenchmarkFuzzyMatch_ExactPath(b *testing.B) {
	text := "configuration"
	pattern := "configuration"

	b.ResetTimer()
	for b.Loop() {
		FuzzyMatch(text, pattern, 0.3, false)
	}
}

func BenchmarkFuzzyMatch_LevenshteinPath(b *testing.B) {
	text := "configuration"
	pattern := "xyzabc"

	b.ResetTimer()
	for b.Loop() {
		FuzzyMatch(text, pattern, 0.1, false)
	}
}
