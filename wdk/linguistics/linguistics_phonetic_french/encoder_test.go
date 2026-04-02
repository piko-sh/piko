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

package linguistics_phonetic_french

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	enc, err := New()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, DefaultMaxLength, enc.maxLength)
}

func TestNewWithMaxLength(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(4)

	require.NoError(t, err)
	assert.Equal(t, 4, enc.maxLength)
}

func TestNewWithMaxLength_ZeroUsesDefault(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(0)

	require.NoError(t, err)
	assert.Equal(t, DefaultMaxLength, enc.maxLength)
}

func TestGetLanguage(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	assert.Equal(t, "french", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "french", enc.GetLanguage())
}

func TestEncode_EmptyString(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	assert.Equal(t, "", enc.Encode(""))
}

func TestEncode_Consistency(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	words := []string{"BONJOUR", "MAISON", "CHAPEAU", "PHONETIQUE"}
	for _, word := range words {
		first := enc.Encode(word)
		second := enc.Encode(word)
		assert.Equal(t, first, second, "encoding of %q should be consistent", word)
	}
}

func TestEncode_CaseInsensitive(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	assert.Equal(t, enc.Encode("BONJOUR"), enc.Encode("bonjour"))
	assert.Equal(t, enc.Encode("Maison"), enc.Encode("MAISON"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("INTERNATIONAL")

	assert.LessOrEqual(t, len(result), 3)
}

func TestEncode_Words(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
	}{
		{name: "CH digraph", input: "CHAPEAU"},
		{name: "PH digraph", input: "PHONETIQUE"},
		{name: "GN digraph", input: "AGNEAU"},
		{name: "QU digraph", input: "QUAND"},
		{name: "AU vowel", input: "AUSSI"},
		{name: "OI vowel", input: "BOIRE"},
		{name: "OU vowel", input: "TOUJOURS"},
		{name: "EAU trigraph", input: "BEAU"},
		{name: "TION pattern", input: "NATION"},
		{name: "single letter", input: "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := enc.Encode(tt.input)
			assert.NotEmpty(t, result, "encoding of %q should not be empty", tt.input)
			assert.LessOrEqual(t, len(result), DefaultMaxLength)
		})
	}
}

func TestIsFrenchVowel(t *testing.T) {
	t.Parallel()

	vowels := []byte{'A', 'E', 'I', 'O', 'U', 'Y'}
	for _, v := range vowels {
		assert.True(t, isFrenchVowel(v), "expected %c to be a vowel", v)
	}

	consonants := []byte{'B', 'C', 'D', 'F', 'G'}
	for _, c := range consonants {
		assert.False(t, isFrenchVowel(c), "expected %c not to be a vowel", c)
	}
}

func TestIsFrenchConsonant(t *testing.T) {
	t.Parallel()

	assert.True(t, isFrenchConsonant('B'))
	assert.True(t, isFrenchConsonant('C'))
	assert.False(t, isFrenchConsonant('A'))
	assert.False(t, isFrenchConsonant('0'))
}

func TestIsNasalFollower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		word     string
		position int
		expected bool
	}{
		{name: "at word end", word: "AN", position: 1, expected: true},
		{name: "before consonant", word: "ANT", position: 1, expected: true},
		{name: "before vowel", word: "ANE", position: 1, expected: false},
		{name: "before another N", word: "ANN", position: 1, expected: false},
		{name: "before another M", word: "AMM", position: 1, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isNasalFollower(tt.word, tt.position)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	assert.True(t, hasPrefix("CHAPEAU", 0, "CH"))
	assert.True(t, hasPrefix("ACHETER", 1, "CH"))
	assert.False(t, hasPrefix("CHAPEAU", 0, "PH"))
	assert.False(t, hasPrefix("CH", 1, "CH"))
}

func TestHasSuffix(t *testing.T) {
	t.Parallel()

	assert.True(t, hasSuffix("PETIT", "IT"))
	assert.True(t, hasSuffix("GRAND", "AND"))
	assert.False(t, hasSuffix("AB", "ABC"))
}

func TestRemoveSilentEndings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "ENT suffix long word", input: "MANGENT", expected: "MANG"},
		{name: "S suffix", input: "AMIS", expected: "AMI"},
		{name: "T suffix", input: "PETIT", expected: "PETI"},
		{name: "D suffix", input: "GRAND", expected: "GRAN"},
		{name: "X suffix", input: "VOIX", expected: "VOI"},
		{name: "Z suffix", input: "ASSEZ", expected: "ASSE"},
		{name: "short word no change", input: "ET", expected: "ET"},
		{name: "no silent ending", input: "BEAU", expected: "BEAU"},
		{name: "short ENT has silent T removed", input: "VENT", expected: "VEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := removeSilentEndings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
