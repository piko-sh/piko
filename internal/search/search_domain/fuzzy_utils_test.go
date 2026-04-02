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

package search_domain

import (
	"testing"
)

func TestFindSimilarTermsJaroWinkler(t *testing.T) {
	vocabulary := []string{
		"configuration",
		"configure",
		"config",
		"container",
		"constant",
		"construct",
		"content",
	}

	testCases := []struct {
		name          string
		queryTerm     string
		expectTerms   []string
		minSimilarity float64
		maxResults    int
	}{
		{
			name:          "missing character",
			queryTerm:     "configurtion",
			minSimilarity: 0.85,
			maxResults:    3,
			expectTerms:   []string{"configuration"},
		},
		{
			name:          "transposed characters",
			queryTerm:     "ocnfiguration",
			minSimilarity: 0.85,
			maxResults:    3,
			expectTerms:   []string{"configuration"},
		},
		{
			name:          "prefix match",
			queryTerm:     "conf",
			minSimilarity: 0.70,
			maxResults:    5,
			expectTerms:   []string{"configuration", "configure", "config"},
		},
		{
			name:          "completely different",
			queryTerm:     "document",
			minSimilarity: 0.85,
			maxResults:    3,
			expectTerms:   []string{},
		},
		{
			name:          "extra character",
			queryTerm:     "configurationn",
			minSimilarity: 0.85,
			maxResults:    3,
			expectTerms:   []string{"configuration"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := findSimilarTermsJaroWinkler(tc.queryTerm, vocabulary, tc.minSimilarity, tc.maxResults)

			if len(tc.expectTerms) == 0 {
				if len(results) > 0 {
					t.Errorf("Expected no matches, but got %d results", len(results))
				}
				return
			}

			if len(results) == 0 {
				t.Errorf("Expected matches for %q, but got none", tc.queryTerm)
				return
			}

			resultMap := make(map[string]float64)
			for _, result := range results {
				resultMap[result.Term] = result.Similarity
			}

			for _, expectedTerm := range tc.expectTerms {
				if similarity, found := resultMap[expectedTerm]; !found {
					t.Errorf("Expected to find %q in results, but it was not found", expectedTerm)
				} else {
					t.Logf("Found %q with similarity %.3f", expectedTerm, similarity)
				}
			}
		})
	}
}

func TestFindTermsWithinEditDistance(t *testing.T) {
	vocabulary := []string{
		"configuration",
		"configure",
		"config",
		"container",
		"constant",
	}

	testCases := []struct {
		name        string
		queryTerm   string
		expectTerms []string
		maxDistance int
	}{
		{
			name:        "distance 1 - missing char",
			queryTerm:   "configurtion",
			maxDistance: 1,
			expectTerms: []string{"configuration"},
		},
		{
			name:        "distance 2 - two edits",
			queryTerm:   "confgurtion",
			maxDistance: 2,
			expectTerms: []string{"configuration"},
		},
		{
			name:        "strict distance 1",
			queryTerm:   "config",
			maxDistance: 1,
			expectTerms: []string{},
		},
		{
			name:        "exact match excluded",
			queryTerm:   "configuration",
			maxDistance: 2,
			expectTerms: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := findTermsWithinEditDistance(tc.queryTerm, vocabulary, tc.maxDistance)

			if len(tc.expectTerms) == 0 {
				if len(results) > 0 {
					t.Errorf("Expected no matches, but got %d: %v", len(results), results)
				}
				return
			}

			if len(results) == 0 {
				t.Errorf("Expected matches, but got none")
				return
			}

			resultMap := make(map[string]int)
			for _, result := range results {
				resultMap[result.Term] = result.Distance
			}

			for _, expectedTerm := range tc.expectTerms {
				if distance, found := resultMap[expectedTerm]; !found {
					t.Errorf("Expected to find %q in results", expectedTerm)
				} else {
					t.Logf("Found %q with distance %d", expectedTerm, distance)
				}
			}
		})
	}
}

func TestFindBestFuzzyMatch(t *testing.T) {
	vocabulary := []string{
		"configuration",
		"configure",
		"configured",
		"configuring",
		"container",
	}

	testCases := []struct {
		name        string
		queryTerm   string
		expectTerm  string
		expectScore float64
		maxDistance int
	}{
		{
			name:        "close match",
			queryTerm:   "configurtion",
			maxDistance: 2,
			expectTerm:  "configuration",
			expectScore: 0.90,
		},
		{
			name:        "very close match",
			queryTerm:   "configration",
			maxDistance: 2,
			expectTerm:  "configuration",
			expectScore: 0.85,
		},
		{
			name:        "no match",
			queryTerm:   "document",
			maxDistance: 2,
			expectTerm:  "",
			expectScore: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			term, score := findBestFuzzyMatch(tc.queryTerm, vocabulary, tc.maxDistance)

			if tc.expectTerm == "" {
				if term != "" {
					t.Errorf("Expected no match, but got %q with score %.3f", term, score)
				}
				return
			}

			if term != tc.expectTerm {
				t.Errorf("Expected match %q, got %q", tc.expectTerm, term)
			}

			if score < tc.expectScore {
				t.Errorf("Expected score >= %.3f, got %.3f", tc.expectScore, score)
			}

			t.Logf("Matched %q → %q with score %.3f", tc.queryTerm, term, score)
		})
	}
}

func TestJaroWinkler_NameTypos(t *testing.T) {
	testCases := []struct {
		a        string
		b        string
		minScore float64
	}{
		{a: "SHACKLEFORD", b: "SHACKELFORD", minScore: 0.98},
		{a: "DUNNINGHAM", b: "CUNNIGHAM", minScore: 0.89},
		{a: "NICHLESON", b: "NICHULSON", minScore: 0.95},
		{a: "JONES", b: "JOHNSON", minScore: 0.83},
		{a: "MASSEY", b: "MASSIE", minScore: 0.93},
	}

	for _, tc := range testCases {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			vocabulary := []string{tc.b}
			results := findSimilarTermsJaroWinkler(tc.a, vocabulary, tc.minScore-0.01, 1)

			if len(results) == 0 {
				t.Errorf("Expected to match %q with %q (min %.3f)", tc.a, tc.b, tc.minScore)
			} else if results[0].Similarity < tc.minScore {
				t.Errorf("Expected similarity >= %.3f, got %.3f", tc.minScore, results[0].Similarity)
			} else {
				t.Logf("%q matches %q with similarity %.3f", tc.a, tc.b, results[0].Similarity)
			}
		})
	}
}

func TestUkkonen_Performance(t *testing.T) {

	vocabulary := []string{
		"configuration",
		"documentation",
		"implementation",
		"authentication",
		"authorization",
	}

	queryTerm := "configurtion"

	results := findTermsWithinEditDistance(queryTerm, vocabulary, 1)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Term != "configuration" {
		t.Errorf("Expected 'configuration', got %q", results[0].Term)
	}

	if len(results) > 0 && results[0].Distance != 1 {
		t.Errorf("Expected distance 1, got %d", results[0].Distance)
	}
}

func TestRankTermsBySimilarity(t *testing.T) {
	vocabulary := []string{
		"configuration",
		"configure",
		"config",
		"confirm",
		"conflict",
		"container",
		"construct",
	}

	results := rankTermsBySimilarity("conf", vocabulary, 5)

	if len(results) < 3 {
		t.Errorf("Expected at least 3 results, got %d", len(results))
	}

	if len(results) > 0 && results[0].Similarity < 0.7 {
		t.Errorf("Expected high similarity for top result, got %.3f", results[0].Similarity)
	}

	for i := 1; i < len(results); i++ {
		if results[i].Similarity > results[i-1].Similarity {
			t.Errorf("Results not sorted: results[%d].Similarity (%.3f) > results[%d].Similarity (%.3f)",
				i, results[i].Similarity, i-1, results[i-1].Similarity)
		}
	}

	t.Logf("Top 5 matches for 'conf':")
	for i, result := range results {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %q (similarity: %.3f)", i+1, result.Term, result.Similarity)
	}
}

func TestFuzzyUtils_EmptyInput(t *testing.T) {
	vocabulary := []string{"test", "data"}

	t.Run("empty query JaroWinkler", func(t *testing.T) {
		results := findSimilarTermsJaroWinkler("", vocabulary, 0.8, 5)
		if len(results) != 0 {
			t.Errorf("Expected no results for empty query, got %d", len(results))
		}
	})

	t.Run("empty vocabulary JaroWinkler", func(t *testing.T) {
		results := findSimilarTermsJaroWinkler("test", []string{}, 0.8, 5)
		if len(results) != 0 {
			t.Errorf("Expected no results for empty vocabulary, got %d", len(results))
		}
	})

	t.Run("empty query Ukkonen", func(t *testing.T) {
		results := findTermsWithinEditDistance("", vocabulary, 2)
		if len(results) != 0 {
			t.Errorf("Expected no results for empty query, got %d", len(results))
		}
	})

	t.Run("empty vocabulary Ukkonen", func(t *testing.T) {
		results := findTermsWithinEditDistance("test", []string{}, 2)
		if len(results) != 0 {
			t.Errorf("Expected no results for empty vocabulary, got %d", len(results))
		}
	})
}

func TestFuzzyUtils_TypoScenarios(t *testing.T) {
	vocabulary := []string{
		"documentation",
		"deployment",
		"development",
		"debugging",
		"database",
	}

	typoScenarios := []struct {
		typo     string
		expected string
		reason   string
	}{
		{
			typo:     "documnetation",
			expected: "documentation",
			reason:   "character transposition",
		},
		{
			typo:     "deploymen",
			expected: "deployment",
			reason:   "missing character",
		},
		{
			typo:     "developmnet",
			expected: "development",
			reason:   "character transposition",
		},
		{
			typo:     "debuging",
			expected: "debugging",
			reason:   "missing double consonant",
		},
	}

	for _, scenario := range typoScenarios {
		t.Run(scenario.typo, func(t *testing.T) {

			results := findSimilarTermsJaroWinkler(scenario.typo, vocabulary, 0.85, 1)

			if len(results) == 0 {
				t.Errorf("Failed to match typo %q → %q (%s)",
					scenario.typo, scenario.expected, scenario.reason)
				return
			}

			if results[0].Term != scenario.expected {
				t.Errorf("Expected to match %q, got %q (similarity: %.3f)",
					scenario.expected, results[0].Term, results[0].Similarity)
			} else {
				t.Logf("Typo %q → %q (%.3f similarity) - %s",
					scenario.typo, scenario.expected, results[0].Similarity, scenario.reason)
			}
		})
	}
}

func TestCompareAlgorithms(t *testing.T) {
	vocabulary := []string{"configuration"}

	scenarios := []struct {
		query       string
		description string
	}{
		{query: "configurtion", description: "missing single char"},
		{query: "ocnfiguration", description: "transposed chars"},
		{query: "konfig", description: "very different"},
		{query: "config", description: "significantly shorter"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.description, func(t *testing.T) {

			ukkonenResults := findTermsWithinEditDistance(scenario.query, vocabulary, 3)
			ukkonenMatch := len(ukkonenResults) > 0

			jaroResults := findSimilarTermsJaroWinkler(scenario.query, vocabulary, 0.75, 1)
			jaroMatch := len(jaroResults) > 0

			t.Logf("Query: %q", scenario.query)
			if ukkonenMatch {
				t.Logf("  Ukkonen: distance=%d", ukkonenResults[0].Distance)
			} else {
				t.Logf("  Ukkonen: no match")
			}
			if jaroMatch {
				t.Logf("  Jaro-Winkler: similarity=%.3f", jaroResults[0].Similarity)
			} else {
				t.Logf("  Jaro-Winkler: no match")
			}
		})
	}
}

func BenchmarkJaroWinkler(b *testing.B) {
	vocabulary := make([]string, 1000)
	for i := range 1000 {
		vocabulary[i] = generateRandomTerm(10)
	}

	queryTerm := "configuration"

	b.ResetTimer()
	for b.Loop() {
		findSimilarTermsJaroWinkler(queryTerm, vocabulary, 0.85, 5)
	}
}

func BenchmarkUkkonen(b *testing.B) {
	vocabulary := make([]string, 1000)
	for i := range 1000 {
		vocabulary[i] = generateRandomTerm(10)
	}

	queryTerm := "configuration"

	b.ResetTimer()
	for b.Loop() {
		findTermsWithinEditDistance(queryTerm, vocabulary, 2)
	}
}

func BenchmarkFuzzyVocabularyScan(b *testing.B) {

	vocabulary := make([]string, 10000)
	for i := range 10000 {
		vocabulary[i] = generateRandomTerm(8)
	}

	queryTerm := "configurtion"

	b.Run("JaroWinkler_10k", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			findSimilarTermsJaroWinkler(queryTerm, vocabulary, 0.85, 3)
		}
	})

	b.Run("Ukkonen_10k", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			findTermsWithinEditDistance(queryTerm, vocabulary, 2)
		}
	})
}

func generateRandomTerm(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range length {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}
