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

package linguistics_phonetic_german

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

	enc, err := NewWithMaxLength(5)

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, 5, enc.maxLength)
}

func TestNewWithMaxLength_ZeroUsesDefault(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(0)

	require.NoError(t, err)
	assert.Equal(t, DefaultMaxLength, enc.maxLength)
}

func TestNewWithMaxLength_NegativeUsesDefault(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(-1)

	require.NoError(t, err)
	assert.Equal(t, DefaultMaxLength, enc.maxLength)
}

func TestGetLanguage(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	assert.Equal(t, "german", enc.GetLanguage())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	enc, err := Factory()

	require.NoError(t, err)
	require.NotNil(t, enc)
	assert.Equal(t, "german", enc.GetLanguage())
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

	words := []string{"Mueller", "Schmidt", "Berlin", "Phonetik", "Taxi"}
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

	assert.Equal(t, enc.Encode("MUELLER"), enc.Encode("mueller"))
	assert.Equal(t, enc.Encode("SCHMIDT"), enc.Encode("schmidt"))
	assert.Equal(t, enc.Encode("Berlin"), enc.Encode("BERLIN"))
}

func TestEncode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("Alexanderplatz")

	assert.LessOrEqual(t, len(result), 3)
}

func TestEncode_MeierMeyerMatch(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	assert.Equal(t, enc.Encode("Meier"), enc.Encode("Meyer"),
		"Meier and Meyer should produce the same Cologne code")
}

func TestEncode_Words(t *testing.T) {
	t.Parallel()

	enc, err := New()
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
	}{
		{name: "simple vowel", input: "Anna"},
		{name: "consonant cluster", input: "Schmidt"},
		{name: "PH digraph", input: "Phonetik"},
		{name: "X produces two digits", input: "Taxi"},
		{name: "D before S", input: "DSC"},
		{name: "T before C", input: "TCH"},
		{name: "C after S", input: "SCH"},
		{name: "C at start before A", input: "Carl"},
		{name: "single letter", input: "A"},
		{name: "single consonant", input: "B"},
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

func TestRemoveDuplicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{name: "empty", input: []byte{}, expected: []byte{}},
		{name: "single", input: []byte{'1'}, expected: []byte{'1'}},
		{name: "no duplicates", input: []byte{'1', '2', '3'}, expected: []byte{'1', '2', '3'}},
		{name: "consecutive duplicates", input: []byte{'1', '1', '2', '2', '3'}, expected: []byte{'1', '2', '3'}},
		{name: "all same", input: []byte{'4', '4', '4', '4'}, expected: []byte{'4'}},
		{name: "non-consecutive same", input: []byte{'1', '2', '1'}, expected: []byte{'1', '2', '1'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := removeDuplicates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveInternalZeros(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{name: "empty", input: []byte{}, expected: []byte{}},
		{name: "leading zero kept", input: []byte{'0', '1', '0', '2'}, expected: []byte{'0', '1', '2'}},
		{name: "no leading zero", input: []byte{'1', '0', '2', '0', '3'}, expected: []byte{'1', '2', '3'}},
		{name: "only zero", input: []byte{'0'}, expected: []byte{'0'}},
		{name: "no zeros", input: []byte{'1', '2', '3'}, expected: []byte{'1', '2', '3'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := removeInternalZeros(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFollowingChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		word     string
		position int
		chars    string
		expected bool
	}{
		{name: "match", word: "AB", position: 0, chars: "B", expected: true},
		{name: "no match", word: "AC", position: 0, chars: "B", expected: false},
		{name: "end of word", word: "A", position: 0, chars: "B", expected: false},
		{name: "multi chars match", word: "AH", position: 0, chars: "HK", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isFollowingChar(tt.word, tt.position, tt.chars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrecedingChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		word     string
		position int
		chars    string
		expected bool
	}{
		{name: "match", word: "SA", position: 1, chars: "S", expected: true},
		{name: "no match", word: "BA", position: 1, chars: "S", expected: false},
		{name: "start of word", word: "A", position: 0, chars: "S", expected: false},
		{name: "multi chars match", word: "ZC", position: 1, chars: "SZ", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isPrecedingChar(tt.word, tt.position, tt.chars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCHardContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		word     string
		position int
		expected bool
	}{
		{name: "C before A", word: "CA", position: 0, expected: true},
		{name: "C before H", word: "CH", position: 0, expected: true},
		{name: "C before K", word: "CK", position: 0, expected: true},
		{name: "C before O", word: "CO", position: 0, expected: true},
		{name: "C before Q", word: "CQ", position: 0, expected: true},
		{name: "C before U", word: "CU", position: 0, expected: true},
		{name: "C before X", word: "CX", position: 0, expected: true},
		{name: "C before L at start", word: "CL", position: 0, expected: true},
		{name: "C before L not at start", word: "ACL", position: 1, expected: false},
		{name: "C before R at start", word: "CR", position: 0, expected: true},
		{name: "C before R not at start", word: "ACR", position: 1, expected: false},
		{name: "C before E", word: "CE", position: 0, expected: false},
		{name: "C before I", word: "CI", position: 0, expected: false},
		{name: "C at end", word: "C", position: 0, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isCHardContext(tt.word, tt.position)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextDependentHandlers(t *testing.T) {
	t.Parallel()

	t.Run("handleP before H", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneFVW), handleP("PH", 0))
	})

	t.Run("handleP not before H", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneBP), handleP("PA", 0))
	})

	t.Run("handleD before CSZ", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneSZ), handleD("DS", 0))
		assert.Equal(t, byte(CologneSZ), handleD("DC", 0))
		assert.Equal(t, byte(CologneSZ), handleD("DZ", 0))
	})

	t.Run("handleD default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneDT), handleD("DA", 0))
	})

	t.Run("handleT before CSZ", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneSZ), handleT("TS", 0))
		assert.Equal(t, byte(CologneSZ), handleT("TC", 0))
		assert.Equal(t, byte(CologneSZ), handleT("TZ", 0))
	})

	t.Run("handleT default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, byte(CologneDT), handleT("TA", 0))
	})
}

func TestEncode_TruncationStaysValidUTF8(t *testing.T) {
	t.Parallel()

	enc, err := NewWithMaxLength(3)
	require.NoError(t, err)

	result := enc.Encode("MUELLERSCHMIDT")

	assert.True(t, utf8.ValidString(result), "result should be valid UTF-8")
	assert.LessOrEqual(t, utf8.RuneCountInString(result), 3, "rune count must respect maxLength")
}
