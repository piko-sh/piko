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

package linguistics_phonetic_russian

import (
	"testing"
	"unicode/utf8"

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

	assert.Equal(t, "russian", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "russian", enc.GetLanguage())
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

	words := []string{"Москва", "Петров", "Иванов", "Борис"}
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

	assert.Equal(t, enc.Encode("МОСКВА"), enc.Encode("москва"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("Александр")

	assert.LessOrEqual(t, len(result), 3)
}

func TestEncode_CyrillicWords(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
	}{
		{name: "basic word", input: "Москва"},
		{name: "name", input: "Петров"},
		{name: "common name", input: "Иванов"},
		{name: "with YO", input: "Ёлка"},
		{name: "with YU", input: "Юрий"},
		{name: "with YA", input: "Яков"},
		{name: "with soft sign", input: "Конь"},
		{name: "with hard sign", input: "Объект"},
		{name: "with ZH", input: "Жизнь"},
		{name: "with SH", input: "Шапка"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := enc.Encode(tt.input)
			assert.NotEmpty(t, result, "encoding of %q should not be empty", tt.input)
		})
	}
}

func TestIsRussianVowel(t *testing.T) {
	t.Parallel()

	vowels := []rune{CyrA, CyrE, CyrYO, CyrI, CyrO, CyrU, CyrYI, CyrEE, CyrYU, CyrYA}
	for _, v := range vowels {
		assert.True(t, isRussianVowel(v), "expected %c to be a vowel", v)
	}

	consonants := []rune{CyrB, CyrV, CyrG, CyrD, CyrK}
	for _, c := range consonants {
		assert.False(t, isRussianVowel(c), "expected %c not to be a vowel", c)
	}
}

func TestIsVoicedConsonant(t *testing.T) {
	t.Parallel()

	voiced := []rune{CyrB, CyrV, CyrG, CyrD, CyrZH, CyrZ}
	for _, v := range voiced {
		assert.True(t, isVoicedConsonant(v), "expected %c to be voiced", v)
	}

	unvoiced := []rune{CyrK, CyrP, CyrS, CyrT, CyrF}
	for _, u := range unvoiced {
		assert.False(t, isVoicedConsonant(u), "expected %c not to be voiced", u)
	}
}

func TestDevoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    rune
		expected rune
	}{
		{name: "B to P", input: CyrB, expected: CyrP},
		{name: "V to F", input: CyrV, expected: CyrF},
		{name: "G to K", input: CyrG, expected: CyrK},
		{name: "D to T", input: CyrD, expected: CyrT},
		{name: "ZH to SH", input: CyrZH, expected: CyrSH},
		{name: "Z to S", input: CyrZ, expected: CyrS},
		{name: "unvoiced unchanged", input: CyrK, expected: CyrK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := devoice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWordEnd(t *testing.T) {
	t.Parallel()

	runes := []rune("ТЕСТ")
	assert.False(t, isWordEnd(runes, 0))
	assert.False(t, isWordEnd(runes, 1))
	assert.True(t, isWordEnd(runes, 3))
	assert.True(t, isWordEnd(runes, 4))
}

func TestEncode_CyrillicTruncationInRunes(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("ПРИВЕТСТВИЕ")

	assert.True(t, utf8.ValidString(result), "result should be valid UTF-8 even when truncated")
	assert.LessOrEqual(t, utf8.RuneCountInString(result), 3, "rune count must respect maxLength")
}
