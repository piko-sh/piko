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

func TestTokeniser_BasicTokenisation(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("Hello world test")

	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(tokens))
	}

	if tokens[0].Normalised != "hello" {
		t.Errorf("Expected 'hello', got %q", tokens[0].Normalised)
	}
	if tokens[0].Position != 0 {
		t.Errorf("Expected position 0, got %d", tokens[0].Position)
	}
}

func TestTokeniser_UTF8Handling(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "emoji",
			input:    "hello 🎉 world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "chinese characters",
			input:    "hello 你好 world",
			expected: []string{"hello", "你好", "world"},
		},
		{
			name:     "arabic",
			input:    "hello مرحبا world",
			expected: []string{"hello", "مرحبا", "world"},
		},
		{
			name:     "russian",
			input:    "hello привет world",
			expected: []string{"hello", "привет", "world"},
		},
		{
			name:     "mixed scripts",
			input:    "café naïve résumé",
			expected: []string{"cafe", "naive", "resume"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tokeniser.Tokenise(tc.input)

			if len(tokens) != len(tc.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tc.expected), len(tokens))
			}

			for i, expected := range tc.expected {
				if i >= len(tokens) {
					break
				}
				if tokens[i].Normalised != expected {
					t.Errorf("Token[%d]: expected %q, got %q", i, expected, tokens[i].Normalised)
				}
			}
		})
	}
}

func TestTokeniser_HandlesHyphens(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("api-key user-defined")

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Normalised != "api-key" {
		t.Errorf("Expected 'api-key' as single token, got %q", tokens[0].Normalised)
	}
	if tokens[1].Normalised != "user-defined" {
		t.Errorf("Expected 'user-defined' as single token, got %q", tokens[1].Normalised)
	}
}

func TestTokeniser_HandlesUnderscores(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("user_id field_name")

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Normalised != "user_id" {
		t.Errorf("Expected 'user_id' as single token, got %q", tokens[0].Normalised)
	}
}

func TestTokeniser_StopWordFiltering(t *testing.T) {
	config := DefaultConfig()

	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("the quick brown fox")

	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens after stop word filtering, got %d", len(tokens))
	}

	for _, tok := range tokens {
		if tok.Normalised == "the" {
			t.Error("Stop word 'the' should have been filtered")
		}
	}
}

func TestTokeniser_MinMaxLength(t *testing.T) {
	config := DefaultConfig()
	config.MinTokenLength = 3
	config.MaxTokenLength = 8
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("a bb ccc dddddddd eeeeeeeee")

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
}

func TestTokeniser_PositionTracking(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("first second third")

	for i, tok := range tokens {
		if tok.Position != i {
			t.Errorf("Token[%d].Position = %d, want %d", i, tok.Position, i)
		}
	}
}

func TestTokeniser_EmptyInput(t *testing.T) {
	config := DefaultConfig()
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("")

	if len(tokens) != 0 {
		t.Errorf("Expected 0 tokens for empty string, got %d", len(tokens))
	}
}

func TestTokeniser_OnlyPunctuation(t *testing.T) {
	config := DefaultConfig()
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("!!! ... ???")

	if len(tokens) != 0 {
		t.Errorf("Expected 0 tokens for punctuation-only string, got %d", len(tokens))
	}
}

func TestTokeniser_TokeniseToStrings(t *testing.T) {
	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.TokeniseToStrings("hello world test")

	expected := []string{"hello", "world", "test"}
	if len(tokens) != len(expected) {
		t.Errorf("Expected %d strings, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if i >= len(tokens) {
			break
		}
		if tokens[i] != exp {
			t.Errorf("String[%d]: expected %q, got %q", i, exp, tokens[i])
		}
	}
}

func Test_extractWords(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{input: "Hello World", expected: []string{"hello", "world"}},
		{input: "API-Key", expected: []string{"api", "key"}},
		{input: "user_id", expected: []string{"user", "id"}},
		{input: "123abc", expected: []string{"123abc"}},
		{input: "", expected: []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := extractWords(tc.input)

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

func TestTokeniser_WithmockNormaliser(t *testing.T) {

	mockNormaliser := &mockNormaliser{
		NormaliseFunc: func(text string) string {
			return strings.ToUpper(text)
		},
	}

	config := DefaultConfig()
	config.StopWords = nil
	tokeniser := newTokeniserWithDeps(mockNormaliser, config)

	tokens := tokeniser.Tokenise("hello")

	if tokens[0].Normalised != "HELLO" {
		t.Errorf("Expected 'HELLO' from mock normaliser, got %q", tokens[0].Normalised)
	}
}

func TestTokeniser_PreserveCase(t *testing.T) {
	config := DefaultConfig()
	config.PreserveCase = true
	config.StopWords = nil
	tokeniser := NewTokeniser(config)

	tokens := tokeniser.Tokenise("Hello World")

	if tokens[0].Normalised != "Hello" {
		t.Errorf("Expected 'Hello' with preserved case, got %q", tokens[0].Normalised)
	}
}

func BenchmarkTokeniser(b *testing.B) {
	config := DefaultConfig()
	tokeniser := NewTokeniser(config)

	text := "The quick brown fox jumps over the lazy dog. This is a test of tokenisation performance with stop words and normalisation."

	b.ResetTimer()
	for b.Loop() {
		tokeniser.Tokenise(text)
	}
}
