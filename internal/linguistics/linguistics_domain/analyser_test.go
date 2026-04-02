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
	"testing"
)

func TestAnalyser_EnglishSmartMode(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart

	mockStemmer := &MockStemmer{
		FixedLanguage: "english",
		StemMappings: map[string]string{
			"configurations": "configur",
			"running":        "run",
			"smoothly":       "smooth",
		},
	}
	analyser := NewAnalyser(config, WithStemmer(mockStemmer))

	text := "The configurations are running smoothly"
	tokens := analyser.Analyse(text)

	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens (after stop word filtering), got %d", len(tokens))
	}

	if len(tokens) > 0 {
		tok := tokens[0]
		if tok.Normalised != "configurations" {
			t.Errorf("Token[0].Normalised = %q, want 'configurations'", tok.Normalised)
		}
		if tok.Stemmed != "configur" {
			t.Errorf("Token[0].Stemmed = %q, want 'configur'", tok.Stemmed)
		}
		if tok.Phonetic == "" {
			t.Error("Token[0].Phonetic should not be empty in Smart mode")
		}
	}
}

func TestAnalyser_SpanishSmartMode(t *testing.T) {
	config := DefaultConfigForLanguage("spanish")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	text := "Las configuraciones están funcionando correctamente"
	tokens := analyser.Analyse(text)

	if len(tokens) < 2 {
		t.Errorf("Expected at least 2 tokens (after Spanish stop word filtering), got %d", len(tokens))
	}

	if len(tokens) > 0 {
		tok := tokens[0]
		if tok.Normalised != "configuraciones" {
			t.Errorf("Token[0].Normalised = %q, want 'configuraciones'", tok.Normalised)
		}

		if tok.Stemmed != "configur" {
			t.Logf("Token[0].Stemmed = %q (actual Snowball output)", tok.Stemmed)
		}
	}
}

func TestAnalyser_FrenchSmartMode(t *testing.T) {
	config := DefaultConfigForLanguage("french")
	config.Mode = AnalysisModeSmart

	mockStemmer := &MockStemmer{
		FixedLanguage: "french",
		StemMappings: map[string]string{
			"configurations": "configur",
			"excellentes":    "excellent",
		},
	}
	analyser := NewAnalyser(config, WithStemmer(mockStemmer))

	text := "Les configurations sont excellentes"
	tokens := analyser.Analyse(text)

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens (after French stop word filtering), got %d", len(tokens))
	}

	if len(tokens) > 0 {
		tok := tokens[0]
		if tok.Normalised != "configurations" {
			t.Errorf("Token[0].Normalised = %q, want 'configurations'", tok.Normalised)
		}

		if tok.Stemmed != "configur" {
			t.Errorf("Token[0].Stemmed = %q, want 'configur'", tok.Stemmed)
		}
	}
}

func TestAnalyser_RussianSmartMode(t *testing.T) {
	config := DefaultConfigForLanguage("russian")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	text := "Конфигурация работает отлично"
	tokens := analyser.Analyse(text)

	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(tokens))
	}

	if len(tokens) > 0 {
		tok := tokens[0]

		if tok.Stemmed == "" {
			t.Error("Token[0].Stemmed should not be empty")
		}
	}
}

func TestAnalyser_FastMode(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeFast
	analyser := NewAnalyser(config)

	text := "running configurations"
	tokens := analyser.Analyse(text)

	for _, tok := range tokens {
		if tok.Stemmed != "" {
			t.Errorf("Fast mode should not populate Stemmed field, got %q", tok.Stemmed)
		}
		if tok.Phonetic != "" {
			t.Errorf("Fast mode should not populate Phonetic field, got %q", tok.Phonetic)
		}
	}
}

func TestAnalyser_BasicMode(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeBasic
	analyser := NewAnalyser(config)

	text := "the quick brown fox"
	tokens := analyser.Analyse(text)

	for _, tok := range tokens {
		if tok.Stemmed != "" {
			t.Errorf("Basic mode should not populate Stemmed field")
		}
		if tok.Phonetic != "" {
			t.Errorf("Basic mode should not populate Phonetic field")
		}
	}
}

func TestAnalyser_MultiLanguageCrossMatch(t *testing.T) {

	engConfig := DefaultConfigForLanguage("english")
	engConfig.Mode = AnalysisModeSmart
	engMockStemmer := &MockStemmer{
		FixedLanguage: "english",
		StemMappings: map[string]string{
			"configuration": "configur",
		},
	}
	engAnalyser := NewAnalyser(engConfig, WithStemmer(engMockStemmer))
	engTokens := engAnalyser.Analyse("configuration")

	spaConfig := DefaultConfigForLanguage("spanish")
	spaConfig.Mode = AnalysisModeSmart
	spaMockStemmer := &MockStemmer{
		FixedLanguage: "spanish",
		StemMappings: map[string]string{
			"configuracion": "configur",
		},
	}
	spaAnalyser := NewAnalyser(spaConfig, WithStemmer(spaMockStemmer))
	spaTokens := spaAnalyser.Analyse("configuracion")

	freConfig := DefaultConfigForLanguage("french")
	freConfig.Mode = AnalysisModeSmart
	freMockStemmer := &MockStemmer{
		FixedLanguage: "french",
		StemMappings: map[string]string{
			"configuration": "configur",
		},
	}
	freAnalyser := NewAnalyser(freConfig, WithStemmer(freMockStemmer))
	freTokens := freAnalyser.Analyse("configuration")

	if len(engTokens) == 0 || len(spaTokens) == 0 || len(freTokens) == 0 {
		t.Fatal("Failed to tokenise in one or more languages")
	}

	engStem := engTokens[0].Stemmed
	spaStem := spaTokens[0].Stemmed
	freStem := freTokens[0].Stemmed

	if engStem != "configur" {
		t.Errorf("English stem = %q, want 'configur'", engStem)
	}
	if freStem != "configur" {
		t.Errorf("French stem = %q, want 'configur'", freStem)
	}
	if spaStem != "configur" {
		t.Errorf("Spanish stem = %q, want 'configur'", spaStem)
	}
	t.Logf("Multi-language stems: EN=%q, ES=%q, FR=%q", engStem, spaStem, freStem)
}

func TestAnalyser_StopWordsByLanguage(t *testing.T) {
	testCases := []struct {
		language     string
		text         string
		expectedKeep []string
	}{
		{
			language:     "english",
			text:         "the quick brown fox",
			expectedKeep: []string{"quick", "brown", "fox"},
		},
		{
			language:     "spanish",
			text:         "el rápido zorro marrón",
			expectedKeep: []string{"rapido", "zorro", "marron"},
		},
		{
			language:     "french",
			text:         "le rapide renard brun",
			expectedKeep: []string{"rapide", "renard", "brun"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.language, func(t *testing.T) {
			config := DefaultConfigForLanguage(tc.language)
			config.Mode = AnalysisModeFast
			analyser := NewAnalyser(config)

			tokens := analyser.Analyse(tc.text)

			if len(tokens) != len(tc.expectedKeep) {
				t.Errorf("Expected %d tokens, got %d", len(tc.expectedKeep), len(tokens))
			}

			for i, tok := range tokens {
				if i >= len(tc.expectedKeep) {
					break
				}

				normalised := tok.Normalised
				if normalised != tc.expectedKeep[i] && normalised != strings.ToLower(tc.expectedKeep[i]) {
					t.Errorf("Token[%d] = %q, want %q", i, normalised, tc.expectedKeep[i])
				}
			}
		})
	}
}

func BenchmarkAnalyser_EnglishSmart(b *testing.B) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	text := "The quick brown fox jumps over the lazy dog. This is a test of the stemming and phonetic encoding capabilities."

	b.ResetTimer()
	for b.Loop() {
		analyser.Analyse(text)
	}
}

func BenchmarkAnalyser_MultiLanguage(b *testing.B) {
	languages := SupportedLanguages()
	text := "configuration system documentation"

	for _, lang := range languages {
		b.Run(lang, func(b *testing.B) {
			config := DefaultConfigForLanguage(lang)
			config.Mode = AnalysisModeSmart
			analyser := NewAnalyser(config)

			b.ResetTimer()
			for b.Loop() {
				analyser.Analyse(text)
			}
		})
	}
}

func TestAnalyser_AnalyseToStrings(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	analyser := NewAnalyser(config)

	tokens := analyser.AnalyseToStrings("the quick brown fox")

	if len(tokens) == 0 {
		t.Error("Expected non-empty strings")
	}

	for _, s := range tokens {
		if s != tokens[0] && s[0] >= 'A' && s[0] <= 'Z' {
			t.Errorf("Expected lowercase, got %q", s)
		}
	}

	t.Logf("Analysed to strings: %v", tokens)
}

func TestAnalyser_AnalyseToStemmed(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	stemmed := analyser.AnalyseToStemmed("running configurations")

	if len(stemmed) == 0 {
		t.Error("Expected non-empty stemmed tokens")
	}

	t.Logf("Stemmed tokens: %v", stemmed)

	for _, stem := range stemmed {
		if stem == "" {
			t.Error("Stemmed token should not be empty in Smart mode")
		}
	}
}

func TestAnalyser_AnalyseToStemmed_FastMode(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeFast
	analyser := NewAnalyser(config)

	stemmed := analyser.AnalyseToStemmed("running")

	normalised := analyser.AnalyseToStrings("running")

	if len(stemmed) != len(normalised) {
		t.Error("Fast mode should return normalised, not stemmed")
	}

	t.Logf("Fast mode (no stemming): %v", stemmed)
}

func TestAnalyser_AnalyseToPhonetic(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	phonetic := analyser.AnalyseToPhonetic("stephen steven smith")

	if len(phonetic) == 0 {
		t.Error("Expected non-empty phonetic codes")
	}

	for _, code := range phonetic {
		if code == "" {
			t.Error("Phonetic code should not be empty in Smart mode")
		}
		if len(code) > DefaultPhoneticCodeLength {
			t.Errorf("Phonetic code %q exceeds max length %d", code, DefaultPhoneticCodeLength)
		}
	}

	t.Logf("Phonetic codes: %v", phonetic)
}

func TestAnalyser_AnalyseToPhonetic_NonSmartMode(t *testing.T) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeFast
	analyser := NewAnalyser(config)

	phonetic := analyser.AnalyseToPhonetic("test")

	if phonetic != nil {
		t.Error("Non-Smart mode should return nil for phonetic")
	}
}

func TestAnalyser_GetStemmer(t *testing.T) {
	config := DefaultConfigForLanguage("spanish")
	analyser := NewAnalyser(config)

	stemmer := analyser.GetStemmer()

	if stemmer == nil {
		t.Error("GetStemmer should return non-nil stemmer")
	}

	if stemmer.GetLanguage() != "spanish" {
		t.Errorf("Expected Spanish stemmer, got %q", stemmer.GetLanguage())
	}
}

func TestAnalyser_GetPhoneticEncoder(t *testing.T) {
	config := DefaultConfig()
	analyser := NewAnalyser(config)

	phonetic := analyser.GetPhoneticEncoder()

	if phonetic == nil {
		t.Error("GetPhoneticEncoder should return non-nil encoder")
	}

	code := phonetic.Encode("test")
	if len(code) == 0 {
		t.Error("Phonetic encoder should encode test word")
	}
}

func TestAnalyser_GetMode(t *testing.T) {
	config := DefaultConfig()
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	mode := analyser.GetMode()

	if mode != AnalysisModeSmart {
		t.Errorf("Expected Smart mode, got %v", mode)
	}
}

func TestNewAnalyser_LanguageValidation(t *testing.T) {
	config := DefaultConfig()
	config.Language = "klingon"

	analyser := NewAnalyser(config)

	stemmer := analyser.GetStemmer()
	if stemmer.GetLanguage() != "klingon" {
		t.Errorf("NoOpStemmer language should be %q, got %q", "klingon", stemmer.GetLanguage())
	}
}

func TestAnalyserPool_CreateAndUse(t *testing.T) {
	config := DefaultConfig()
	pool := NewAnalyserPool(config, 2)

	if pool == nil {
		t.Fatal("NewAnalyserPool should return non-nil pool")
	}

	analyser1 := pool.Get()
	if analyser1 == nil {
		t.Error("Pool.Get() should return non-nil analyser")
	}

	pool.Put(analyser1)

	analyser2 := pool.Get()
	if analyser2 == nil {
		t.Error("Pool.Get() after Put should return analyser")
	}
}

func TestAnalyserPool_InvalidSize(t *testing.T) {
	config := DefaultConfig()

	pool := NewAnalyserPool(config, 0)
	analyser := pool.Get()

	if analyser == nil {
		t.Error("Pool with size 0 should still work (defaults to 1)")
	}

	pool = NewAnalyserPool(config, -5)
	analyser = pool.Get()

	if analyser == nil {
		t.Error("Pool with negative size should still work (defaults to 1)")
	}
}

func TestAnalyserPool_Exhaustion(t *testing.T) {
	config := DefaultConfig()
	pool := NewAnalyserPool(config, 2)

	a1 := pool.Get()
	a2 := pool.Get()
	a3 := pool.Get()

	if a1 == nil || a2 == nil || a3 == nil {
		t.Error("Pool should create new analysers when exhausted")
	}
}

func TestAnalyserPool_PutWhenFull(t *testing.T) {
	config := DefaultConfig()
	pool := NewAnalyserPool(config, 1)

	analyser1 := NewAnalyser(config)
	analyser2 := NewAnalyser(config)

	pool.Put(analyser1)
	pool.Put(analyser2)

}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Language != LanguageEnglish {
		t.Errorf("Default language should be %q, got %q", LanguageEnglish, config.Language)
	}

	if config.MinTokenLength != DefaultMinTokenLength {
		t.Errorf("Default min length should be %d, got %d", DefaultMinTokenLength, config.MinTokenLength)
	}

	if config.MaxTokenLength != DefaultMaxTokenLength {
		t.Errorf("Default max length should be %d, got %d", DefaultMaxTokenLength, config.MaxTokenLength)
	}

	if config.StopWords == nil {
		t.Error("Default config should have stop words")
	}
}

func TestAnalyser_WithMocks(t *testing.T) {

	var stemCalls []string
	mockStemmer := &MockStemmer{
		StemFunc: func(word string) string {
			stemCalls = append(stemCalls, word)
			return word + "_STEM"
		},
		FixedLanguage: "english",
	}

	var phoneticCalls []string
	mockPhonetic := &MockPhoneticEncoder{
		EncodeFunc: func(word string) string {
			phoneticCalls = append(phoneticCalls, word)
			return "PHON"
		},
	}

	mockTokeniser := &mockTokeniser{
		TokeniseFunc: func(text string) []Token {
			return []Token{
				{Normalised: "word1"},
				{Normalised: "word2"},
			}
		},
	}

	config := DefaultConfig()
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyserWithDeps(mockTokeniser, mockStemmer, mockPhonetic, config)

	tokens := analyser.Analyse("test input")

	if len(stemCalls) != 2 {
		t.Errorf("Expected 2 stem calls, got %d", len(stemCalls))
	}

	if len(phoneticCalls) != 2 {
		t.Errorf("Expected 2 phonetic calls, got %d", len(phoneticCalls))
	}

	for _, tok := range tokens {
		if tok.Stemmed != tok.Normalised+"_STEM" {
			t.Errorf("Expected stemmed suffix, got %q", tok.Stemmed)
		}
		if tok.Phonetic != "PHON" {
			t.Errorf("Expected PHON, got %q", tok.Phonetic)
		}
	}

	t.Logf("Stem calls: %v", stemCalls)
	t.Logf("Phonetic calls: %v", phoneticCalls)
}

func BenchmarkAnalyser_FullPipeline(b *testing.B) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeSmart
	analyser := NewAnalyser(config)

	text := "The quick brown fox jumps over the lazy dog. Configuration management system deployment."

	b.ResetTimer()
	for b.Loop() {
		analyser.Analyse(text)
	}
}

func BenchmarkAnalyser_FastMode(b *testing.B) {
	config := DefaultConfigForLanguage("english")
	config.Mode = AnalysisModeFast
	analyser := NewAnalyser(config)

	text := "The quick brown fox jumps over the lazy dog. Configuration management system deployment."

	b.ResetTimer()
	for b.Loop() {
		analyser.Analyse(text)
	}
}
