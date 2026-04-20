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

package linguistics_stemmer_hebrew

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripNikkud(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "no nikkud", input: "שלום", expected: "שלום"},
		{name: "basic nikkud", input: "שָׁלוֹם", expected: "שלום"},
		{name: "deity word", input: "אֱלֹהִים", expected: "אלהים"},
		{name: "genesis word", input: "בְּרֵאשִׁית", expected: "בראשית"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, stripNikkud(testCase.input))
		})
	}
}

func TestIsNikkud(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    rune
		expected bool
	}{
		{name: "cantillation mark", input: '\u0591', expected: true},
		{name: "vowel point lower end", input: '\u05BD', expected: true},
		{name: "maqaf punctuation", input: '\u05BE', expected: false},
		{name: "vowel point upper start", input: '\u05BF', expected: true},
		{name: "vowel point upper end", input: '\u05C7', expected: true},
		{name: "hebrew letter alef", input: '\u05D0', expected: false},
		{name: "ascii letter", input: 'a', expected: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, isNikkud(testCase.input))
		})
	}
}

func TestTryPrefixStrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantWord string
		wantOK   bool
	}{
		{name: "he prefix", input: "הבית", wantWord: "בית", wantOK: true},
		{name: "compound prefix", input: "והבית", wantWord: "בית", wantOK: true},
		{name: "three letter word rejected", input: "בית", wantWord: "בית", wantOK: false},
		{name: "no prefix match", input: "תפוח", wantWord: "תפוח", wantOK: false},
		{name: "vowel bounded stem preserved", input: "בואו", wantWord: "בואו", wantOK: false},
		{name: "vowel bounded stem preserved aleph", input: "בארו", wantWord: "בארו", wantOK: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, ok := tryPrefixStrip(testCase.input)
			assert.Equal(t, testCase.wantOK, ok)
			assert.Equal(t, testCase.wantWord, result)
		})
	}
}

func TestUnsafeSingleCharStrip(t *testing.T) {
	t.Parallel()

	assert.True(t, unsafeSingleCharStrip("ב", "ואו"), "ב + ואו both vowel-bounded")
	assert.True(t, unsafeSingleCharStrip("ב", "ארו"), "ב + ארו leading vowel with content prefix")
	assert.True(t, unsafeSingleCharStrip("ל", "ארו"), "ל + ארו content prefix leading vowel")
	assert.True(t, unsafeSingleCharStrip("מ", "ארו"), "מ + ארו content prefix leading vowel")
	assert.True(t, unsafeSingleCharStrip("ה", "ארו"), "ארו is still vowel-bounded on both ends")
	assert.False(t, unsafeSingleCharStrip("ה", "ארנ"), "ה on consonant-ending stem allows strip")
	assert.False(t, unsafeSingleCharStrip("ו", "ארנ"), "ו is not a content prefix for leading-vowel rule")
	assert.False(t, unsafeSingleCharStrip("ב", "שלום"), "remainder longer than three runes")
	assert.False(t, unsafeSingleCharStrip("וה", "בית"), "multi-char prefix bypasses the guard")
}

func TestIsContentPrefix(t *testing.T) {
	t.Parallel()

	assert.True(t, isContentPrefix("ב"))
	assert.True(t, isContentPrefix("כ"))
	assert.True(t, isContentPrefix("ל"))
	assert.True(t, isContentPrefix("מ"))
	assert.False(t, isContentPrefix("ה"))
	assert.False(t, isContentPrefix("ו"))
	assert.False(t, isContentPrefix("ש"))
	assert.False(t, isContentPrefix("וה"))
}

func TestIsHebrewVowelLetter(t *testing.T) {
	t.Parallel()

	assert.True(t, isHebrewVowelLetter('א'))
	assert.True(t, isHebrewVowelLetter('ה'))
	assert.True(t, isHebrewVowelLetter('ו'))
	assert.True(t, isHebrewVowelLetter('י'))
	assert.False(t, isHebrewVowelLetter('ב'))
	assert.False(t, isHebrewVowelLetter('ל'))
	assert.False(t, isHebrewVowelLetter('a'))
}

func TestTrySuffixStrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantWord string
		wantOK   bool
	}{
		{name: "masculine plural", input: "ילדימ", wantWord: "ילד", wantOK: true},
		{name: "feminine plural", input: "חברות", wantWord: "חבר", wantOK: true},
		{name: "feminine adjective", input: "ישראלית", wantWord: "ישראל", wantOK: true},
		{name: "short word rejected", input: "בית", wantWord: "בית", wantOK: false},
		{name: "no suffix match", input: "אבא", wantWord: "אבא", wantOK: false},
		{name: "longer suffix rejected then shorter accepted", input: "אבתיהמ", wantWord: "אבת", wantOK: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, ok := trySuffixStrip(testCase.input)
			assert.Equal(t, testCase.wantOK, ok)
			assert.Equal(t, testCase.wantWord, result)
		})
	}
}

func TestStem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty input", input: "", expected: ""},
		{name: "short word unchanged", input: "גם", expected: "גמ"},
		{name: "three letter root unchanged", input: "בנק", expected: "בנק"},
		{name: "he prefix", input: "הבנק", expected: "בנק"},
		{name: "vav prefix", input: "ובנק", expected: "בנק"},
		{name: "bet prefix", input: "בבנק", expected: "בנק"},
		{name: "lamed prefix", input: "לבנק", expected: "בנק"},
		{name: "mem prefix", input: "מבנק", expected: "בנק"},
		{name: "shin prefix", input: "שבנק", expected: "בנק"},
		{name: "compound vav-he prefix", input: "והבנק", expected: "בנק"},
		{name: "compound kaf-shin-he prefix", input: "כשהגיע", expected: "גיע"},
		{name: "masculine plural suffix", input: "בנקים", expected: "בנק"},
		{name: "feminine plural suffix", input: "חברות", expected: "חבר"},
		{name: "feminine adjective suffix", input: "ישראלית", expected: "ישראל"},
		{name: "possessive suffix", input: "בנקיהם", expected: "בנק"},
		{name: "combined prefix and suffix", input: "שהילדים", expected: "ילד"},
		{name: "combined vav-he and ot", input: "והחברות", expected: "חבר"},
		{name: "combined lamed and im", input: "לבנקים", expected: "בנק"},
		{name: "nikkud stripped then prefix", input: "הַבַּנְק", expected: "בנק"},
		{name: "no prefix or suffix", input: "תפוח", expected: "תפוח"},
		{name: "final form folded to regular", input: "דם", expected: "דמ"},
		{name: "regular and final forms produce same stem", input: "חכמ", expected: "חכמ"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, stem(testCase.input))
		})
	}
}

func TestStem_Consistency(t *testing.T) {
	t.Parallel()

	word := "שהילדים"
	first := stem(word)
	second := stem(word)
	assert.Equal(t, first, second)
}

func TestStem_FinalAndRegularFormsProduceSameStem(t *testing.T) {
	t.Parallel()

	assert.Equal(t, stem("חכם"), stem("חכמ"), "final mem and regular mem must stem identically")
	assert.Equal(t, stem("ספרך"), stem("ספרכ"), "final kaf and regular kaf must stem identically")
}

func TestFoldFinalForm(t *testing.T) {
	t.Parallel()

	assert.Equal(t, '\u05DB', foldFinalForm('\u05DA'), "final kaf folds to kaf")
	assert.Equal(t, '\u05DE', foldFinalForm('\u05DD'), "final mem folds to mem")
	assert.Equal(t, '\u05E0', foldFinalForm('\u05DF'), "final nun folds to nun")
	assert.Equal(t, '\u05E4', foldFinalForm('\u05E3'), "final pe folds to pe")
	assert.Equal(t, '\u05E6', foldFinalForm('\u05E5'), "final tsadi folds to tsadi")
	assert.Equal(t, '\u05D0', foldFinalForm('\u05D0'), "non-final rune unchanged")
	assert.Equal(t, 'a', foldFinalForm('a'), "non-Hebrew rune unchanged")
}

func TestNormaliseFinalForms(t *testing.T) {
	t.Parallel()

	assert.Empty(t, normaliseFinalForms(""))
	assert.Equal(t, "חכמ", normaliseFinalForms("חכם"))
	assert.Equal(t, "בית", normaliseFinalForms("בית"))
}
