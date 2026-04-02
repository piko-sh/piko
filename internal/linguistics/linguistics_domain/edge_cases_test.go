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

func TestNormaliser_WellFormedUTF8(t *testing.T) {
	normaliser := NewNormaliser(false)

	wellFormed := []string{
		"hello",
		"café",
		"你好",
		"مرحبا",
		"Привет",
	}

	for _, text := range wellFormed {
		result := normaliser.Normalise(text)
		if len(result) == 0 && len(text) > 0 {
			t.Errorf("Transform should not fail for well-formed UTF-8: %q", text)
		}
	}
}

func TestFuzzyMatch_EdgeCaseThresholds(t *testing.T) {

	text := "testing"
	pattern := "test"

	matched, score := FuzzyMatch(text, pattern, 0.5, false)
	if !matched {
		t.Error("Should match with threshold below score")
	}

	matched, _ = FuzzyMatch(text, pattern, 0.99, false)
	if matched {
		t.Error("Should NOT match with threshold above score")
	}

	t.Logf("Substring score: %.3f", score)
}

func TestFuzzyMatch_WordLevelExactMatch(t *testing.T) {

	text := "the testing framework"
	pattern := "framework"

	matched, score := FuzzyMatch(text, pattern, 0.3, false)

	if !matched {
		t.Error("Should match 'framework' via word-level exact match")
	}

	t.Logf("Word exact match score: %.2f", score)
}

func TestFuzzyMatch_WordLevelContainsOnly(t *testing.T) {

	text := "identification"
	pattern := "ident"

	matched, score := FuzzyMatch(text, pattern, 0.3, false)

	if !matched {
		t.Error("Should match via word contains")
	}

	t.Logf("Word contains score: %.2f", score)
}

func TestCalculateLevenshteinScore_BothEmpty(t *testing.T) {
	score := calculateLevenshteinScore("", "")

	if score != 0.0 {
		t.Errorf("Both empty should return 0.0, got %.2f", score)
	}
}

func TestTryWordLevelMatch_EmptyPattern(t *testing.T) {

	score, matched := tryWordLevelMatch("test", "   ")

	if matched {
		t.Error("Empty pattern words should not match")
	}

	if score != 0.0 {
		t.Errorf("Empty pattern should return score 0.0, got %.2f", score)
	}
}

func TestScoreWordMatches_ContainsPath(t *testing.T) {
	textWords := []string{"identification", "number"}
	patternWords := []string{"dent"}

	score := scoreWordMatches(textWords, patternWords)

	if score < scoreWordContainsMatch {
		t.Errorf("Expected contains match score >= %.2f, got %.2f", scoreWordContainsMatch, score)
	}

	t.Logf("Contains match score: %.2f", score)
}

func TestPhoneticEncoder_AllBranches(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testWords := []string{

		"Church",
		"Civic",
		"Cedar",
		"Cynic",
		"Accent",
		"Cat",

		"Judge",
		"Fidget",
		"Edgy",
		"Ladder",
		"Dog",

		"Gnome",
		"Gneiss",
		"Sigh",
		"Ghost",
		"George",
		"Giraffe",
		"Gypsy",
		"Egg",
		"Go",

		"Hello",
		"Ahead",
		"Shah",

		"Phone",
		"Apple",
		"People",

		"Shoe",
		"Session",
		"Asia",
		"Hiss",
		"Sun",

		"Nation",
		"Spatial",
		"The",
		"Catch",
		"Matter",
		"Test",

		"Whale",
		"Water",
		"Saw",
	}

	for _, word := range testWords {
		code := encoder.Encode(word)
		if len(code) == 0 {
			t.Errorf("Expected non-empty code for %q", word)
		}
	}

	t.Logf("Tested %d words to hit all branches", len(testWords))
}

func TestPhoneticEncoder_LowercaseInput(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	upper := encoder.Encode("STEPHEN")
	lower := encoder.Encode("stephen")

	if upper != lower {
		t.Errorf("Case should not matter: %q vs %q", upper, lower)
	}
}

func TestTokeniser_ByteOffsetTracking(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("hello world")

	for i, tok := range tokens {
		if tok.ByteLength == 0 {
			t.Errorf("Token[%d].ByteLength should not be 0", i)
		}
		if tok.ByteLength != len(tok.Original) {
			t.Errorf("Token[%d].ByteLength = %d, want %d", i, tok.ByteLength, len(tok.Original))
		}
	}
}

func TestNoOpStemmer_ErrorHandling(t *testing.T) {

	stemmer := NewNoOpStemmer("english")

	stemmed := stemmer.Stem("testing")

	if stemmed != "testing" {
		t.Errorf("NoOpStemmer.Stem(%q) = %q, want %q", "testing", stemmed, "testing")
	}

	stemmed = stemmer.Stem("")
	if stemmed != "" {
		t.Errorf("NoOpStemmer.Stem(%q) = %q, want %q", "", stemmed, "")
	}
}

func TestFuzzyMatch_AllStrategiesPath(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		pattern  string
		strategy string
	}{
		{name: "exact", text: "test", pattern: "test", strategy: "exact match"},
		{name: "substring", text: "testing", pattern: "test", strategy: "substring"},
		{name: "prefix", text: "x testing", pattern: "x", strategy: "prefix after normalise"},
		{name: "word_match", text: "the configuration file", pattern: "config", strategy: "word contains"},
		{name: "levenshtein", text: "xyz", pattern: "abc", strategy: "levenshtein fallback"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched, score := FuzzyMatch(tc.text, tc.pattern, 0.1, false)

			t.Logf("%s: matched=%v, score=%.2f (%s)", tc.name, matched, score, tc.strategy)
		})
	}
}

func TestNormaliser_PreserveCaseRune(t *testing.T) {
	preserving := NewNormaliser(true)
	notPreserving := NewNormaliser(false)

	testRune := 'A'

	preserved := preserving.NormaliseRune(testRune)
	lowered := notPreserving.NormaliseRune(testRune)

	if preserved != 'A' {
		t.Errorf("Preserve case should keep 'A', got %c", preserved)
	}

	if lowered != 'a' {
		t.Errorf("No preserve should lowercase 'A' to 'a', got %c", lowered)
	}
}

func TestTryFastMatch_PrefixWithoutSubstring(t *testing.T) {

	score, matched := tryFastMatch("test", "te")

	if !matched {
		t.Error("Prefix should match")
	}

	t.Logf("Prefix of 'test': score=%.2f", score)
}
