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
	"unicode"
)

// mockTokeniser implements TokeniserPort for testing.
type mockTokeniser struct {
	// TokeniseFunc overrides the default tokenisation when set; nil uses the default.
	TokeniseFunc func(text string) []Token

	// TokeniseToStringFunc is called by TokeniseToStrings when set.
	TokeniseToStringFunc func(text string) []string
}

// Tokenise implements TokeniserPort.
//
// Takes text (string) which is the input to split into tokens.
//
// Returns []Token which contains the tokenised words from the input.
func (m *mockTokeniser) Tokenise(text string) []Token {
	if m.TokeniseFunc != nil {
		return m.TokeniseFunc(text)
	}
	words := splitIntoWords(text)
	tokens := make([]Token, len(words))
	for i, word := range words {
		tokens[i] = Token{
			Original:   word,
			Normalised: word,
			Stemmed:    "",
			Phonetic:   "",
			Position:   i,
			ByteOffset: 0,
			ByteLength: len(word),
		}
	}
	return tokens
}

// TokeniseToStrings implements TokeniserPort.
//
// Takes text (string) which is the input to tokenise.
//
// Returns []string which contains the normalised form of each token.
func (m *mockTokeniser) TokeniseToStrings(text string) []string {
	if m.TokeniseToStringFunc != nil {
		return m.TokeniseToStringFunc(text)
	}
	tokens := m.Tokenise(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Normalised
	}
	return result
}

// MockStemmer provides a test double for StemmerPort.
type MockStemmer struct {
	// StemFunc overrides the default Stem behaviour when set.
	StemFunc func(word string) string

	// LanguageFunc overrides GetLanguage when set; nil uses FixedLanguage.
	LanguageFunc func() string

	// StemMappings maps words to their stems for testing.
	StemMappings map[string]string

	// FixedLanguage is the language returned by GetLanguage; empty uses default.
	FixedLanguage string
}

// Stem implements StemmerPort.
//
// Takes word (string) which is the word to stem.
//
// Returns string which is the stemmed word, or the original word if no
// stem mapping exists.
func (m *MockStemmer) Stem(word string) string {
	if m.StemFunc != nil {
		return m.StemFunc(word)
	}
	if m.StemMappings != nil {
		if stem, exists := m.StemMappings[word]; exists {
			return stem
		}
	}
	return word
}

// GetLanguage implements StemmerPort.
//
// Returns string which is the language, or English if none is set.
func (m *MockStemmer) GetLanguage() string {
	if m.LanguageFunc != nil {
		return m.LanguageFunc()
	}
	if m.FixedLanguage != "" {
		return m.FixedLanguage
	}
	return LanguageEnglish
}

// MockPhoneticEncoder is a mock implementation of PhoneticEncoderPort for
// testing.
type MockPhoneticEncoder struct {
	// EncodeFunc is called by Encode; if nil, falls back to PhoneticMap.
	EncodeFunc func(word string) string

	// LanguageFunc overrides GetLanguage when set; nil uses FixedLanguage.
	LanguageFunc func() string

	// PhoneticMap maps words to their phonetic codes for mock encoding.
	PhoneticMap map[string]string

	// FixedLanguage is the language returned by GetLanguage; empty uses default.
	FixedLanguage string
}

// Encode implements PhoneticEncoderPort.
//
// Takes word (string) which is the word to encode.
//
// Returns string which is the phonetic code for the word.
func (m *MockPhoneticEncoder) Encode(word string) string {
	if m.EncodeFunc != nil {
		return m.EncodeFunc(word)
	}
	if m.PhoneticMap != nil {
		if code, exists := m.PhoneticMap[word]; exists {
			return code
		}
	}
	code := strings.ToUpper(word)
	if len(code) > DefaultPhoneticCodeLength {
		code = code[:DefaultPhoneticCodeLength]
	}
	return code
}

// GetLanguage implements PhoneticEncoderPort.
//
// Returns string which is the set language, or English if none is set.
func (m *MockPhoneticEncoder) GetLanguage() string {
	if m.LanguageFunc != nil {
		return m.LanguageFunc()
	}
	if m.FixedLanguage != "" {
		return m.FixedLanguage
	}
	return LanguageEnglish
}

// mockNormaliser is a test double that implements NormaliserPort.
type mockNormaliser struct {
	// NormaliseFunc is called by Normalise when set; nil uses the default behaviour.
	NormaliseFunc func(text string) string

	// NormaliseRuneFunc overrides the default NormaliseRune behaviour when set.
	NormaliseRuneFunc func(r rune) rune
}

// Normalise implements NormaliserPort.
//
// Takes text (string) which is the input text to normalise.
//
// Returns string which is the normalised text.
func (m *mockNormaliser) Normalise(text string) string {
	if m.NormaliseFunc != nil {
		return m.NormaliseFunc(text)
	}
	return strings.ToLower(text)
}

// NormaliseRune implements NormaliserPort.
//
// Takes r (rune) which is the character to normalise.
//
// Returns rune which is the normalised character, or a lowercase version if
// no custom function is set.
func (m *mockNormaliser) NormaliseRune(r rune) rune {
	if m.NormaliseRuneFunc != nil {
		return m.NormaliseRuneFunc(r)
	}
	return unicode.ToLower(r)
}
