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

package linguistics_phonetic_hebrew

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

func TestNew(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	require.NotNil(t, encoder)
	assert.Equal(t, DefaultMaxLength, encoder.maxLength)
}

func TestNewWithMaxLength(t *testing.T) {
	t.Parallel()

	encoder, err := NewWithMaxLength(10)
	require.NoError(t, err)
	require.NotNil(t, encoder)
	assert.Equal(t, 10, encoder.maxLength)
}

func TestNewWithMaxLength_ZeroUsesDefault(t *testing.T) {
	t.Parallel()

	encoder, err := NewWithMaxLength(0)
	require.NoError(t, err)
	assert.Equal(t, DefaultMaxLength, encoder.maxLength)

	negative, err := NewWithMaxLength(-5)
	require.NoError(t, err)
	assert.Equal(t, DefaultMaxLength, negative.maxLength)
}

func TestFactory(t *testing.T) {
	t.Parallel()

	encoder, err := Factory()
	require.NoError(t, err)
	require.NotNil(t, encoder)
	var _ linguistics_domain.PhoneticEncoderPort = encoder
}

func TestEncoder_GetLanguage(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, Language, encoder.GetLanguage())
	assert.Equal(t, "hebrew", encoder.GetLanguage())
}

func TestEncoder_Encode_EmptyString(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Empty(t, encoder.Encode(""))
}

func TestEncoder_Encode_Consistency(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	word := "שלום"
	first := encoder.Encode(word)
	second := encoder.Encode(word)
	assert.Equal(t, first, second)
}

func TestEncoder_Encode_MaxLengthTruncation(t *testing.T) {
	t.Parallel()

	encoder, err := NewWithMaxLength(3)
	require.NoError(t, err)
	result := encoder.Encode("אברהמנושץ")
	assert.LessOrEqual(t, len(result), 3)
}

func TestEncoder_Encode_HebrewWords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "bayit", input: "בית", expected: "BJT"},
		{name: "shalom", input: "שלום", expected: "SLVM"},
		{name: "aba silent alef", input: "אבא", expected: "V"},
		{name: "elohim silent he", input: "אלוהים", expected: "LVJM"},
		{name: "chet starts word", input: "חבר", expected: "XVR"},
		{name: "final forms", input: "ספרים", expected: "SFRJM"},
		{name: "qof merges with kaf", input: "קר", expected: "KR"},
		{name: "dalet mid word", input: "יד", expected: "JD"},
		{name: "dalet starts word", input: "דם", expected: "DM"},
		{name: "nun mid word", input: "בנק", expected: "BNK"},
		{name: "gimel plain", input: "גדול", expected: "GDVL"},
		{name: "zayin plain", input: "זכר", expected: "ZXR"},
		{name: "tsadi plain", input: "צבא", expected: "TSV"},
		{name: "he at word start", input: "הלך", expected: "HLX"},
		{name: "he word final silent", input: "אלה", expected: "L"},
		{name: "shin mid word", input: "חשב", expected: "XSV"},
	}
	encoder, err := New()
	require.NoError(t, err)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, encoder.Encode(testCase.input))
		})
	}
}

func TestEncoder_Encode_SilentLetters(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Empty(t, encoder.Encode("א"))
	assert.Empty(t, encoder.Encode("ע"))
}

func TestEncoder_Encode_BegadkephatInitial(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "BT", encoder.Encode("בת"))
	assert.Equal(t, "KL", encoder.Encode("כל"))
	assert.Equal(t, "PR", encoder.Encode("פר"))
}

func TestEncoder_Encode_BegadkephatMedial(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "LV", encoder.Encode("לב"))
	assert.Equal(t, "LX", encoder.Encode("לכ"))
	assert.Equal(t, "LF", encoder.Encode("לפ"))
}

func TestEncoder_Encode_Dagesh(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "LB", encoder.Encode("לבּ"))
	assert.Equal(t, "LK", encoder.Encode("לכּ"))
	assert.Equal(t, "LP", encoder.Encode("לפּ"))
}

func TestEncoder_Encode_ShinDot(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "SHLVM", encoder.Encode("שׁלום"))
}

func TestEncoder_Encode_SinDot(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "SR", encoder.Encode("שׂר"))
}

func TestEncoder_Encode_GereshLoanwords(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "DJVV", encoder.Encode("ג׳וב"))
	assert.Equal(t, "ZHKT", encoder.Encode("ז׳קט"))
	assert.Equal(t, "CHR", encoder.Encode("צ׳ר"))
	assert.Equal(t, "DHM", encoder.Encode("ד׳ם"), "dalet-geresh loanwords map to DH")
}

func TestEncoder_Encode_FinalForms(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "MLX", encoder.Encode("מלך"))
	assert.Equal(t, "KM", encoder.Encode("קם"))
	assert.Equal(t, "BN", encoder.Encode("בן"))
	assert.Equal(t, "KF", encoder.Encode("כף"))
	assert.Equal(t, "RTS", encoder.Encode("רץ"))
}

func TestEncoder_Encode_NikkudStripped(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	withNikkud := encoder.Encode("בָּיִת")
	withoutNikkud := encoder.Encode("בית")
	assert.Equal(t, "BJT", withoutNikkud)
	assert.NotEmpty(t, withNikkud)
}

func TestEncoder_Encode_DoubleConsonantCollapse(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "M", encoder.Encode("ממ"))
}

func TestEncoder_Encode_NonHebrewLetters(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "BA", encoder.Encode("בa"))
}

func TestEncoder_Encode_SingleRuneInputs(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "H", encoder.Encode("ה"))
	assert.Empty(t, encoder.Encode("א"))
	assert.Empty(t, encoder.Encode("ע"))
}

func TestEncoder_Encode_YiddishLigatures(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Equal(t, "V", encoder.Encode("\u05F0"), "double vav ligature maps to V")
	assert.Equal(t, "VJ", encoder.Encode("\u05F1"), "vav-yod ligature maps to VJ")
	assert.Equal(t, "J", encoder.Encode("\u05F2"), "double yod ligature maps to J")
}

func TestEncoder_Encode_StrayCombiningMark(t *testing.T) {
	t.Parallel()

	encoder, err := New()
	require.NoError(t, err)
	assert.Empty(t, encoder.Encode("\u05C1"), "lone shin dot should be skipped")
	assert.Empty(t, encoder.Encode("\u0591"), "lone cantillation mark should be skipped")
}

func TestEncoder_Encode_MaxLengthOne(t *testing.T) {
	t.Parallel()

	encoder, err := NewWithMaxLength(1)
	require.NoError(t, err)
	assert.Len(t, encoder.Encode("ג׳וב"), 1, "truncation should respect maxLength=1")
}

func TestEncoder_Encode_TruncationOnOversizedCode(t *testing.T) {
	t.Parallel()

	encoder, err := NewWithMaxLength(3)
	require.NoError(t, err)
	result := encoder.Encode("ג׳ז׳צ׳")
	assert.Len(t, result, 3, "triple-geresh words must truncate at byte limit")
}

func TestEncoder_Encode_TruncationAlwaysProducesValidUTF8(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"ш",
		"φ",
		"š",
		"ק\u05C1שаб",
		"בα",
	}
	for _, input := range inputs {
		for maxLength := 1; maxLength <= 4; maxLength++ {
			encoder, err := NewWithMaxLength(maxLength)
			require.NoError(t, err)
			result := encoder.Encode(input)
			assert.True(t, utf8.ValidString(result),
				"encoder produced invalid UTF-8 for input %q at maxLength %d: %q",
				input, maxLength, result)
			assert.LessOrEqual(t, len(result), maxLength,
				"truncation exceeded byte limit for input %q", input)
		}
	}
}

func TestTruncateToRuneBoundary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		maxBytes int
		expected string
	}{
		{name: "ascii within bounds", input: "ABC", maxBytes: 3, expected: "ABC"},
		{name: "ascii truncated", input: "ABCDE", maxBytes: 3, expected: "ABC"},
		{name: "multibyte truncated mid rune", input: "ש", maxBytes: 1, expected: ""},
		{name: "keeps whole multibyte rune", input: "ABש", maxBytes: 3, expected: "AB"},
		{name: "keeps multiple whole multibyte runes", input: "AB", maxBytes: 2, expected: "AB"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := truncateToRuneBoundary(testCase.input, testCase.maxBytes)
			assert.Equal(t, testCase.expected, result)
			assert.True(t, utf8.ValidString(result), "result must be valid UTF-8")
		})
	}
}

func TestPreserveSemanticMarks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "no marks", input: "שלום", expected: "שלום"},
		{name: "strip vowel points", input: "שָׁלוֹם", expected: "שׁלום"},
		{name: "keep dagesh", input: "בּ", expected: "בּ"},
		{name: "strip cantillation keep sin dot", input: "שׂוֹ", expected: "שׂו"},
		{name: "keep geresh", input: "ג׳", expected: "ג׳"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, preserveSemanticMarks(testCase.input))
		})
	}
}

func TestIsHebrewMark(t *testing.T) {
	t.Parallel()

	assert.True(t, isHebrewMark('\u0591'))
	assert.True(t, isHebrewMark(dagesh))
	assert.True(t, isHebrewMark(shinDot))
	assert.True(t, isHebrewMark(sinDot))
	assert.True(t, isHebrewMark(geresh))
	assert.True(t, isHebrewMark('\u05F4'))
	assert.False(t, isHebrewMark(hebAlef), "alef is a letter, not a mark")
	assert.False(t, isHebrewMark(hebTav), "tav is a letter, not a mark")
	assert.False(t, isHebrewMark(hebShin), "shin is a letter, not a mark")
	assert.False(t, isHebrewMark('a'))
	assert.False(t, isHebrewMark('\u0590'))
	assert.False(t, isHebrewMark('\u05F5'))
}

func TestIsSemanticMark(t *testing.T) {
	t.Parallel()

	assert.True(t, isSemanticMark(dagesh))
	assert.True(t, isSemanticMark(shinDot))
	assert.True(t, isSemanticMark(sinDot))
	assert.True(t, isSemanticMark(geresh))
	assert.False(t, isSemanticMark('\u0591'))
	assert.False(t, isSemanticMark(hebAlef))
}

func TestHasDagesh(t *testing.T) {
	t.Parallel()

	runes := []rune("בּת")
	assert.True(t, hasDagesh(runes, 0))
	assert.False(t, hasDagesh(runes, 1))
	assert.False(t, hasDagesh([]rune("בת"), 0))
}

func TestHasGeresh(t *testing.T) {
	t.Parallel()

	runes := []rune("ג׳ב")
	assert.True(t, hasGeresh(runes, 0))
	assert.False(t, hasGeresh(runes, 1))
}

func TestNextRune(t *testing.T) {
	t.Parallel()

	runes := []rune("שלום")
	assert.Equal(t, 'ל', nextRune(runes, 0))
	assert.Equal(t, 'ו', nextRune(runes, 1))
	assert.Equal(t, rune(-1), nextRune(runes, 3))
}

func TestEncoder_RegisteredViaInit(t *testing.T) {
	t.Parallel()

	port := linguistics_domain.CreatePhoneticEncoder(Language)
	require.NotNil(t, port)
	assert.Equal(t, "hebrew", port.GetLanguage())
}
