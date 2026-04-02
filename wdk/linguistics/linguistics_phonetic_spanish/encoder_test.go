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

package linguistics_phonetic_spanish

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

	assert.Equal(t, "spanish", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "spanish", enc.GetLanguage())
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

	words := []string{"HOLA", "CIUDAD", "LLAMAR", "QUESO"}
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

	assert.Equal(t, enc.Encode("HOLA"), enc.Encode("hola"))
	assert.Equal(t, enc.Encode("Ciudad"), enc.Encode("CIUDAD"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("INTERNACIONAL")

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
		{name: "yeismo LL to J", input: "LLAMAR"},
		{name: "seseo Z to S", input: "ZAPATO"},
		{name: "B/V merger", input: "VACA"},
		{name: "CH digraph", input: "CHICO"},
		{name: "RR digraph", input: "PERRO"},
		{name: "GU before E", input: "GUERRA"},
		{name: "QU digraph", input: "QUESO"},
		{name: "H silent", input: "HOLA"},
		{name: "G before E", input: "GENTE"},
		{name: "C before I", input: "CIEN"},
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

func TestIsSpanishVowel(t *testing.T) {
	t.Parallel()

	vowels := []byte{'A', 'E', 'I', 'O', 'U'}
	for _, v := range vowels {
		assert.True(t, isSpanishVowel(v), "expected %c to be a vowel", v)
	}

	consonants := []byte{'B', 'C', 'D', 'F', 'G'}
	for _, c := range consonants {
		assert.False(t, isSpanishVowel(c), "expected %c not to be a vowel", c)
	}
}

func TestIsSpanishConsonant(t *testing.T) {
	t.Parallel()

	assert.True(t, isSpanishConsonant('B'))
	assert.True(t, isSpanishConsonant('C'))
	assert.False(t, isSpanishConsonant('A'))
	assert.False(t, isSpanishConsonant('0'))
}

func TestIsSoftVowel(t *testing.T) {
	t.Parallel()

	assert.True(t, isSoftVowel('E'))
	assert.True(t, isSoftVowel('I'))
	assert.False(t, isSoftVowel('A'))
	assert.False(t, isSoftVowel('O'))
	assert.False(t, isSoftVowel('U'))
}

func TestIsWordEnd(t *testing.T) {
	t.Parallel()

	assert.False(t, isWordEnd("TEST", 0))
	assert.True(t, isWordEnd("TEST", 3))
	assert.True(t, isWordEnd("TEST", 4))
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	assert.True(t, hasPrefix("CHICO", 0, "CH"))
	assert.True(t, hasPrefix("ACHICO", 1, "CH"))
	assert.False(t, hasPrefix("CHICO", 0, "PH"))
	assert.False(t, hasPrefix("CH", 1, "CH"))
}
