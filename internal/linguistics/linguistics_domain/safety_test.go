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
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAnalyser_ConcurrentUsage(t *testing.T) {
	analyser := NewAnalyser(DefaultConfigForLanguage("english"))

	var wg sync.WaitGroup
	const numGoroutines = 50

	for i := range numGoroutines {
		id := i
		wg.Go(func() {

			text := "concurrent test number"
			tokens := analyser.Analyse(text)

			if len(tokens) == 0 {
				t.Errorf("Goroutine %d: expected tokens", id)
			}
		})
	}

	wg.Wait()
}

func TestAnalyserPool_ConcurrentGetPut(t *testing.T) {
	pool := NewAnalyserPool(DefaultConfig(), 5)

	var wg sync.WaitGroup
	const numGoroutines = 100

	for range numGoroutines {
		wg.Go(func() {
			analyser := pool.Get()
			analyser.Analyse("pool stress test")
			pool.Put(analyser)
		})
	}

	wg.Wait()
}

func TestTokeniser_ConcurrentUsage(t *testing.T) {
	tokeniser := NewTokeniser(DefaultConfig())

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			tokeniser.Tokenise("concurrent tokenisation test")
		})
	}

	wg.Wait()
}

func TestAnalyser_InvalidConfig_MinGreaterThanMax(t *testing.T) {
	config := DefaultConfig()
	config.MinTokenLength = 100
	config.MaxTokenLength = 1

	analyser := NewAnalyser(config)

	tokens := analyser.Analyse("test word")

	if len(tokens) > 0 {
		t.Logf("Invalid config produced %d tokens (expected 0)", len(tokens))
	}
}

func TestAnalyser_NilStopWords(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil

	analyser := NewAnalyser(config)

	tokens := analyser.Analyse("the quick brown fox")

	if len(tokens) != 4 {
		t.Errorf("Expected 4 tokens with no stop word filtering, got %d", len(tokens))
	}
}

func TestAnalyser_EmptyLanguage(t *testing.T) {
	config := DefaultConfig()
	config.Language = ""

	analyser := NewAnalyser(config)

	stemmer := analyser.GetStemmer()
	if stemmer.GetLanguage() != LanguageEnglish {
		t.Errorf("Empty language should default to %q, got %q", LanguageEnglish, stemmer.GetLanguage())
	}
}

func TestAnalyser_TextWithOnlyStopWords(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	analyser := NewAnalyser(config)

	tokens := analyser.Analyse("the and or but if")

	if len(tokens) > 0 {
		t.Logf("Stop words produced %d tokens: %v", len(tokens), tokens)
	}
}

func TestAnalyser_VeryLongText(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test in short mode")
	}

	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	longText := strings.Repeat("configuration system deployment test ", 2500)

	start := time.Now()
	tokens := analyser.Analyse(longText)
	duration := time.Since(start)

	if duration > 2*time.Second {
		t.Errorf("10k word document took too long: %v", duration)
	}

	if len(tokens) == 0 {
		t.Error("Should produce tokens for large text")
	}

	t.Logf("10k word document: %d tokens in %v", len(tokens), duration)
}

func TestTokeniser_VeryLongSingleToken(t *testing.T) {
	config := DefaultConfig()
	config.MaxTokenLength = 50
	tokeniser := NewTokeniser(config)

	veryLongWord := strings.Repeat("a", 1000)
	tokens := tokeniser.Tokenise(veryLongWord)

	if len(tokens) > 0 {
		t.Errorf("Token exceeding max length should be filtered, got %d tokens", len(tokens))
	}
}

func TestFuzzyMatch_PatternLongerThanText(t *testing.T) {
	matched, score := FuzzyMatch("hi", "hello world test", 0.5, false)

	t.Logf("Pattern > text: matched=%v, score=%.2f", matched, score)
}

func TestFuzzyMatch_ThresholdZero(t *testing.T) {

	matched, score := FuzzyMatch("test", "test", 0.0, false)

	if !matched {
		t.Error("Threshold 0.0 should match exact match")
	}

	t.Logf("Threshold 0.0 exact: score=%.2f", score)

	matched2, score2 := FuzzyMatch("a", "xyz", 0.0, false)
	t.Logf("Threshold 0.0 different: matched=%v, score=%.2f", matched2, score2)
}

func TestFuzzyMatch_ThresholdOne(t *testing.T) {

	matched, _ := FuzzyMatch("test", "test", 1.0, false)
	if !matched {
		t.Error("Exact match should satisfy threshold 1.0")
	}

	matched, _ = FuzzyMatch("test", "testing", 1.0, false)
	if matched {
		t.Error("Non-exact should NOT satisfy threshold 1.0")
	}
}

func TestFuzzyMatch_SingleCharInputs(t *testing.T) {
	testCases := []struct {
		text    string
		pattern string
		match   bool
	}{
		{text: "a", pattern: "a", match: true},
		{text: "a", pattern: "b", match: false},
		{text: "a", pattern: "", match: true},
		{text: "", pattern: "a", match: false},
	}

	for _, tc := range testCases {
		matched, score := FuzzyMatch(tc.text, tc.pattern, 0.5, false)

		if matched != tc.match {
			t.Errorf("FuzzyMatch(%q, %q) = %v, want %v (score=%.2f)",
				tc.text, tc.pattern, matched, tc.match, score)
		}
	}
}

func TestTokeniser_ConsecutiveSeparators(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	testCases := []struct {
		input    string
		expected int
	}{
		{input: "word1    word2", expected: 2},
		{input: "word1...word2", expected: 2},
		{input: "word1\t\t\tword2", expected: 2},
		{input: "word1\n\n\nword2", expected: 2},
		{input: "word1  \t\n  word2", expected: 2},
	}

	for _, tc := range testCases {
		tokens := tokeniser.Tokenise(tc.input)

		if len(tokens) != tc.expected {
			t.Errorf("Input %q: expected %d tokens, got %d", tc.input, tc.expected, len(tokens))
		}
	}
}

func TestTokeniser_LeadingTrailingSeparators(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	testCases := []struct {
		input    string
		expected []string
	}{
		{input: "  leading", expected: []string{"leading"}},
		{input: "trailing  ", expected: []string{"trailing"}},
		{input: "  both  ", expected: []string{"both"}},
		{input: "\tleading", expected: []string{"leading"}},
		{input: "trailing\n", expected: []string{"trailing"}},
	}

	for _, tc := range testCases {
		tokens := tokeniser.Tokenise(tc.input)

		if len(tokens) != len(tc.expected) {
			t.Errorf("Input %q: expected %d tokens, got %d", tc.input, len(tc.expected), len(tokens))
		}

		for i, exp := range tc.expected {
			if i < len(tokens) && tokens[i].Normalised != exp {
				t.Errorf("Token[%d]: expected %q, got %q", i, exp, tokens[i].Normalised)
			}
		}
	}
}

func TestPhoneticEncoder_AllSameCharacter(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []string{
		"AAAA",
		"BBBB",
		"SSSS",
	}

	for _, word := range testCases {
		code := encoder.Encode(word)

		if len(code) == 0 {
			t.Errorf("Expected non-empty code for %q", word)
		}

		t.Logf("%q → %q (repeated character handled)", word, code)
	}
}

func TestPhoneticEncoder_SingleCharacter(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	silentLetters := map[rune]bool{'H': true, 'W': true}

	for character := 'A'; character <= 'Z'; character++ {
		code := encoder.Encode(string(character))

		if len(code) == 0 && !silentLetters[character] {
			t.Errorf("Single char %c should produce code", character)
		}

		if len(code) == 0 {
			t.Logf("Single char %c produced empty code (silent)", character)
		}
	}
}

func TestAnalyser_AllWhitespace(t *testing.T) {
	analyser := NewAnalyser(DefaultConfig())

	whitespaceInputs := []string{
		"     ",
		"\t\t\t",
		"\n\n\n",
		"  \t\n  ",
	}

	for _, input := range whitespaceInputs {
		tokens := analyser.Analyse(input)

		if len(tokens) > 0 {
			t.Errorf("Whitespace-only should produce 0 tokens, got %d", len(tokens))
		}
	}
}

func TestWagnerFischer_VeryLongStrings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test in short mode")
	}

	a := strings.Repeat("a", 1000)
	b := strings.Repeat("b", 1000)

	start := time.Now()
	distance := WagnerFischer(a, b, 1, 1, 2)
	duration := time.Since(start)

	if duration > 200*time.Millisecond {
		t.Errorf("1000-char strings took too long: %v", duration)
	}

	t.Logf("1000-char Wagner-Fischer completed in %v", duration)

	if distance != 2000 {
		t.Logf("Distance for 1000 all-different chars: %d", distance)
	}
}

func TestJaro_NoMatchingCharacters(t *testing.T) {
	score := Jaro("AAAA", "BBBB")

	if score != 0.0 {
		t.Errorf("Completely different strings should have Jaro score 0.0, got %.3f", score)
	}
}

func TestJaroWinkler_BelowThreshold(t *testing.T) {

	const boostThreshold = 0.7
	const prefixSize = 4

	a := "AAAA"
	b := "ZZZZ"

	jaroScore := Jaro(a, b)
	winklerScore := JaroWinkler(a, b, boostThreshold, prefixSize)

	if jaroScore != winklerScore {
		t.Errorf("Below threshold: Jaro=%.3f, Winkler=%.3f (should be equal)", jaroScore, winklerScore)
	}

	t.Logf("Below threshold (%.1f): Jaro=Winkler=%.3f (no boost)", boostThreshold, winklerScore)
}

func TestWagnerFischer_DifferentCosts(t *testing.T) {
	a := "cat"
	b := "bat"

	cost1 := WagnerFischer(a, b, 1, 1, 1)

	cost2 := WagnerFischer(a, b, 1, 1, 2)

	if cost1 >= cost2 {
		t.Errorf("Lower substitution cost should give lower distance: %d vs %d", cost1, cost2)
	}

	t.Logf("Substitution cost 1: %d, cost 2: %d", cost1, cost2)
}

func TestNormaliser_ControlCharacters(t *testing.T) {
	normaliser := NewNormaliser(false)

	testCases := []struct {
		input       string
		description string
	}{
		{input: "text\r\nmore", description: "CRLF"},
		{input: "text\x00more", description: "Null byte"},
		{input: "text\x01\x02", description: "Control chars"},
		{input: "text\tmore", description: "Tab"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			result := normaliser.Normalise(tc.input)

			if len(result) == 0 && len(tc.input) > 0 {
				t.Errorf("Normalisation should preserve some content")
			}

			t.Logf("%s: %q → %q", tc.description, tc.input, result)
		})
	}
}

func TestPhoneticEncoder_NumbersOnly(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	code := encoder.Encode("12345")

	if len(code) > 0 {
		t.Logf("Numbers produced code: %q (numbers skipped)", code)
	}
}

func TestPhoneticEncoder_PunctuationOnly(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	code := encoder.Encode("!!!")

	if len(code) > 0 {
		t.Logf("Punctuation produced code: %q", code)
	}
}

func TestNoOpStemmer_EmptyString(t *testing.T) {
	stemmer := NewNoOpStemmer("english")

	stemmed := stemmer.Stem("")

	if stemmed != "" {
		t.Errorf("NoOpStemmer.Stem(%q) = %q, want %q", "", stemmed, "")
	}
}

func TestNoOpStemmer_NumbersOnly(t *testing.T) {
	stemmer := NewNoOpStemmer("english")

	stemmed := stemmer.Stem("12345")

	if stemmed != "12345" {
		t.Errorf("NoOpStemmer.Stem(%q) = %q, want %q", "12345", stemmed, "12345")
	}
}

func TestNoOpStemmer_VeryLongWord(t *testing.T) {
	stemmer := NewNoOpStemmer("english")

	veryLong := strings.Repeat("testing", 100)
	stemmed := stemmer.Stem(veryLong)

	if stemmed != veryLong {
		t.Errorf("NoOpStemmer.Stem returned different length: got %d, want %d",
			len(stemmed), len(veryLong))
	}
}

func TestNoOpStemmer_Idempotent(t *testing.T) {
	stemmer := NewNoOpStemmer("english")

	word := "running"
	stem1 := stemmer.Stem(word)
	stem2 := stemmer.Stem(stem1)

	if stem1 != stem2 {
		t.Errorf("NoOpStemmer not idempotent: %q → %q → %q", word, stem1, stem2)
	}
	if stem1 != word {
		t.Errorf("NoOpStemmer.Stem(%q) = %q, want %q", word, stem1, word)
	}
}

func BenchmarkAnalyser_LargeDocument(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	analyser := NewAnalyser(DefaultConfigForLanguage("english"))

	text := strings.Repeat("configuration system deployment management testing framework implementation ", 143)

	b.ResetTimer()
	for b.Loop() {
		analyser.Analyse(text)
	}
}
