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

// Tokeniser splits text into individual tokens with position tracking.
// It implements TokeniserPort and uses NormaliserPort for dependency injection.
type Tokeniser struct {
	// normaliser converts raw token text into a standard form.
	normaliser NormaliserPort

	// stopWords is a set of words to filter out during tokenisation.
	stopWords map[string]bool

	// minTokenLength is the minimum length in bytes; shorter tokens are filtered out.
	minTokenLength int

	// maxTokenLength is the maximum token length in bytes; longer tokens are removed.
	maxTokenLength int
}

// NewTokeniser creates a new tokeniser with the given configuration.
// This is the production constructor that uses a real Normaliser.
//
// Takes config (AnalyserConfig) which specifies the tokeniser settings.
//
// Returns *Tokeniser which is ready for use with a real normaliser.
func NewTokeniser(config AnalyserConfig) *Tokeniser {
	return newTokeniserWithDeps(
		NewNormaliser(config.PreserveCase),
		config,
	)
}

// Tokenise splits the input text into tokens with position and offset tracking.
//
// Takes text (string) which is the input text to split into tokens.
//
// Returns []Token which contains the tokens with their normalised forms and
// positions.
func (t *Tokeniser) Tokenise(text string) []Token {
	var tokens []Token
	var currentToken strings.Builder
	position := 0
	tokenStart := 0

	for i, r := range text {
		if isWordChar(r) {
			if currentToken.Len() == 0 {
				tokenStart = i
			}
			_, _ = currentToken.WriteRune(r)
			continue
		}
		tokens, position = t.flushToken(&currentToken, tokenStart, position, tokens)
	}

	tokens, _ = t.flushToken(&currentToken, tokenStart, position, tokens)
	return tokens
}

// TokeniseToStrings returns just the normalised token strings.
// Useful when position information is not needed.
//
// Takes text (string) which is the input to tokenise.
//
// Returns []string which contains the normalised tokens.
func (t *Tokeniser) TokeniseToStrings(text string) []string {
	tokens := t.Tokenise(text)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, token.Normalised)
	}
	return result
}

// flushToken creates a token from the current buffer if non-empty, appends it
// to the tokens slice, and resets the buffer.
//
// Takes currentToken (*strings.Builder) which holds the accumulated text.
// Takes tokenStart (int) which marks where the token began.
// Takes position (int) which is the current token index.
// Takes tokens ([]Token) which is the slice to append to.
//
// Returns []Token which is the updated tokens slice.
// Returns int which is the updated position.
func (t *Tokeniser) flushToken(
	currentToken *strings.Builder,
	tokenStart int,
	position int,
	tokens []Token,
) ([]Token, int) {
	if currentToken.Len() == 0 {
		return tokens, position
	}

	token := t.createToken(currentToken.String(), position, tokenStart)
	currentToken.Reset()

	if token != nil {
		return append(tokens, *token), position + 1
	}
	return tokens, position
}

// createToken creates a Token from the raw string, applying normalisation
// and filtering. Returns nil if the token should be filtered out.
//
// Takes raw (string) which is the original token text.
// Takes position (int) which is the token's position in the stream.
// Takes byteOffset (int) which is the byte offset in the source.
//
// Returns *Token which is the created token, or nil if filtered.
func (t *Tokeniser) createToken(raw string, position int, byteOffset int) *Token {
	if len(raw) < t.minTokenLength || len(raw) > t.maxTokenLength {
		return nil
	}

	normalised := t.normaliser.Normalise(raw)

	if t.stopWords != nil && t.stopWords[normalised] {
		return nil
	}

	return &Token{
		Original:   raw,
		Normalised: normalised,
		Stemmed:    "",
		Phonetic:   "",
		Position:   position,
		ByteOffset: byteOffset,
		ByteLength: len(raw),
	}
}

// newTokeniserWithDeps creates a new tokeniser with injected
// dependencies. This constructor enables dependency injection for
// testing.
//
// Use this in tests to inject a mock normaliser:
// tokeniser := newTokeniserWithDeps(mockNormaliser, config)
//
// Takes normaliser (NormaliserPort) which handles text
// normalisation.
// Takes config (AnalyserConfig) which specifies token processing
// settings.
//
// Returns *Tokeniser which is ready for tokenising text.
func newTokeniserWithDeps(normaliser NormaliserPort, config AnalyserConfig) *Tokeniser {
	return &Tokeniser{
		normaliser:     normaliser,
		stopWords:      config.StopWords,
		minTokenLength: config.MinTokenLength,
		maxTokenLength: config.MaxTokenLength,
	}
}

// extractWords splits text into lowercase words. Use this for simple text
// processing when full Token metadata is not needed.
//
// Takes text (string) which is the input text to split into words.
//
// Returns []string which contains the extracted lowercase words.
func extractWords(text string) []string {
	words := make([]string, 0)
	var currentWord strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			_, _ = currentWord.WriteRune(unicode.ToLower(r))
		} else if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
			currentWord.Reset()
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}
