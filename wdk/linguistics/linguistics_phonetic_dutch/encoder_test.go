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

package linguistics_phonetic_dutch

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

	assert.Equal(t, "dutch", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "dutch", enc.GetLanguage())
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

	words := []string{"AMSTERDAM", "SCHOOL", "SCHRIJVEN", "RUIT"}
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

	assert.Equal(t, enc.Encode("AMSTERDAM"), enc.Encode("amsterdam"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("AMSTERDAM")

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
		{name: "CH digraph", input: "SCHOOL"},
		{name: "IJ digraph", input: "SCHRIJVEN"},
		{name: "OE digraph", input: "MOEDER"},
		{name: "OU digraph", input: "HOUD"},
		{name: "SCH trigraph", input: "SCHOOL"},
		{name: "PH to F", input: "PHARMA"},
		{name: "EI digraph", input: "KLEIN"},
		{name: "AA double vowel", input: "AARDE"},
		{name: "NG digraph", input: "LANG"},
		{name: "V to F", input: "VADER"},
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

func TestEncode_TruncationStaysValidUTF8(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("SCHRIJVEN")

	assert.True(t, utf8.ValidString(result), "result should be valid UTF-8")
	assert.LessOrEqual(t, utf8.RuneCountInString(result), 3, "rune count must respect maxLength")
}
