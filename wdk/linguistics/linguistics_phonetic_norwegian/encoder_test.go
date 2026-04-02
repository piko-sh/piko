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

package linguistics_phonetic_norwegian

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

	assert.Equal(t, "norwegian", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "norwegian", enc.GetLanguage())
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

	words := []string{"OSLO", "BERGEN", "FJORD", "KJELL"}
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

	assert.Equal(t, enc.Encode("OSLO"), enc.Encode("oslo"))
	assert.Equal(t, enc.Encode("Bergen"), enc.Encode("BERGEN"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("TRONDHEIM")

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
		{name: "KJ sound", input: "KJELL"},
		{name: "SJ sound", input: "SJEL"},
		{name: "SKJ sound", input: "SKJORTE"},
		{name: "RS retroflex", input: "NORSK"},
		{name: "AU vowel", input: "HAUG"},
		{name: "EI vowel", input: "STEIN"},
		{name: "OY vowel", input: "OYA"},
		{name: "DJ sound", input: "DJUP"},
		{name: "GJ sound", input: "GJERDE"},
		{name: "NG digraph", input: "LANG"},
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
