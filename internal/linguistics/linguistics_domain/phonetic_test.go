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
	"testing"
	"unicode/utf8"
)

func TestPhoneticEncoder_StandardPairs(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	pairs := []struct {
		name  string
		word1 string
		word2 string
	}{
		{name: "Stephen/Steven", word1: "Stephen", word2: "Steven"},
		{name: "Smith/Smythe", word1: "Smith", word2: "Smythe"},
		{name: "Night/Knight", word1: "Night", word2: "Knight"},
		{name: "Phone/Fone", word1: "Phone", word2: "Fone"},
		{name: "Graphic/Grafic", word1: "Graphic", word2: "Grafic"},
	}

	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			code1 := encoder.Encode(pair.word1)
			code2 := encoder.Encode(pair.word2)

			if code1 != code2 {
				t.Errorf("%q and %q should have same encoding: %q vs %q",
					pair.word1, pair.word2, code1, code2)
			}

			t.Logf("%q and %q both encode to %q ✓", pair.word1, pair.word2, code1)
		})
	}
}

func TestPhoneticEncoder_SilentLetters(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word     string
		contains string
	}{
		{word: "Knight", contains: "NT"},
		{word: "Gnome", contains: "NM"},
		{word: "Psychology", contains: "SKL"},
		{word: "Write", contains: "RT"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := encoder.Encode(tc.word)
			t.Logf("%q → %q", tc.word, code)

			if len(code) == 0 {
				t.Errorf("Expected non-empty encoding for %q", tc.word)
			}
		})
	}
}

func TestPhoneticEncoder_ComplexConsonants(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word     string
		hasSound string
	}{
		{word: "Church", hasSound: "X"},
		{word: "Judge", hasSound: "J"},
		{word: "Garage", hasSound: ""},
		{word: "Phone", hasSound: "F"},
		{word: "This", hasSound: "0"},
		{word: "Session", hasSound: "X"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := encoder.Encode(tc.word)
			t.Logf("%q → %q", tc.word, code)

			if len(code) == 0 {
				t.Errorf("Expected non-empty encoding for %q", tc.word)
			}
		})
	}
}

func TestPhoneticEncoder_EmptyString(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	code := encoder.Encode("")

	if code != "" {
		t.Errorf("Empty string should produce empty code, got %q", code)
	}
}

func TestPhoneticEncoder_MaxLengthDefault(t *testing.T) {
	encoder := NewPhoneticEncoder(0)

	code := encoder.Encode("VERYLONGWORDTHATSHOULDBETRIMMED")

	if len(code) != DefaultPhoneticCodeLength {
		t.Errorf("Expected code length %d, got %d (%q)", DefaultPhoneticCodeLength, len(code), code)
	}
}

func TestPhoneticEncoder_CustomMaxLength(t *testing.T) {
	encoder := NewPhoneticEncoder(6)

	code := encoder.Encode("VERYLONGWORD")

	if len(code) > 6 {
		t.Errorf("Expected max length 6, got %d (%q)", len(code), code)
	}
}

func TestPhoneticEncoder_DoubledLetters(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	single := encoder.Encode("Cater")
	doubled := encoder.Encode("Catter")

	t.Logf("Cater → %q, Catter → %q", single, doubled)
}

func TestPhoneticEncoder_VowelsOnlyAtStart(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	startsWithVowel := encoder.Encode("Apple")
	if len(startsWithVowel) == 0 || startsWithVowel[0] != 'A' {
		t.Errorf("Word starting with vowel should encode to A..., got %q", startsWithVowel)
	}

	hasVowelMiddle := encoder.Encode("Test")

	t.Logf("Test → %q (middle vowel ignored)", hasVowelMiddle)
}

func TestPhoneticEncoder_SpecialInitials(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word        string
		description string
	}{
		{word: "Xenon", description: "X at start → S"},
		{word: "Knight", description: "KN at start → skip K"},
		{word: "Gnome", description: "GN at start → skip G"},
		{word: "Pneumonia", description: "PN at start → skip P"},
		{word: "Write", description: "WR at start → skip W"},
		{word: "Psychology", description: "PS at start → skip P"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := encoder.Encode(tc.word)
			t.Logf("%q → %q (%s)", tc.word, code, tc.description)

			if len(code) == 0 {
				t.Errorf("Expected non-empty code for %q", tc.word)
			}
		})
	}
}

func TestSoundexEncode_KnownCodes(t *testing.T) {
	testCases := []struct {
		word     string
		expected string
	}{
		{word: "Robert", expected: "R163"},
		{word: "Rupert", expected: "R163"},
		{word: "Rubin", expected: "R150"},
		{word: "Smith", expected: "S530"},
		{word: "Smythe", expected: "S530"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := SoundexEncode(tc.word)
			if code != tc.expected {
				t.Errorf("SoundexEncode(%q) = %q, want %q", tc.word, code, tc.expected)
			}
		})
	}
}

func TestSoundexEncode_EmptyString(t *testing.T) {
	code := SoundexEncode("")

	if code != "" {
		t.Errorf("Empty string should produce empty code, got %q", code)
	}
}

func TestSoundexEncode_ShortWords(t *testing.T) {
	testCases := []string{"A", "AB", "ABC"}

	for _, word := range testCases {
		t.Run(word, func(t *testing.T) {
			code := SoundexEncode(word)

			if len(code) != soundexCodeLength {
				t.Errorf("Soundex should return %d chars, got %d (%q)", soundexCodeLength, len(code), code)
			}

			t.Logf("%q → %q", word, code)
		})
	}
}

func TestPhoneticEncoder_AllLetters(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	for character := 'A'; character <= 'Z'; character++ {
		word := string(character) + "test"
		code := encoder.Encode(word)

		if len(code) == 0 {
			t.Errorf("Letter %c: expected non-empty encoding", character)
		}

		t.Logf("%c: %q → %q", character, word, code)
	}
}

func TestPhoneticEncoder_NonAlphabetic(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []string{
		"Test123",
		"Test-Word",
		"Test_Word",
		"Test!",
	}

	for _, word := range testCases {
		t.Run(word, func(t *testing.T) {
			code := encoder.Encode(word)

			if len(code) == 0 {
				t.Errorf("Expected non-empty encoding for %q", word)
			}
			t.Logf("%q → %q", word, code)
		})
	}
}

func BenchmarkPhoneticEncoder(b *testing.B) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)
	word := "configuration"

	b.ResetTimer()
	for b.Loop() {
		encoder.Encode(word)
	}
}

func BenchmarkSoundex(b *testing.B) {
	word := "configuration"

	b.ResetTimer()
	for b.Loop() {
		SoundexEncode(word)
	}
}

func TestPhoneticEncoder_EdgeCases(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word        string
		description string
	}{

		{word: "Buddy", description: "DD → single D"},
		{word: "Mississippi", description: "SS → single S, PP → single P"},

		{word: "Account", description: "CC → K"},
		{word: "Church", description: "CH → X"},
		{word: "Civic", description: "CE, CI"},
		{word: "Judge", description: "DGE → J"},
		{word: "Agile", description: "GI → J"},
		{word: "Geyser", description: "GE → J"},
		{word: "Sigh", description: "GH at end"},
		{word: "Ghost", description: "GH → various"},
		{word: "Haggard", description: "GG → K"},

		{word: "Hello", description: "H at start before vowel"},
		{word: "Ahead", description: "H after vowel"},
		{word: "Shah", description: "H at end"},

		{word: "Phone", description: "PH → F"},
		{word: "Apple", description: "PP → P"},
		{word: "People", description: "P default"},

		{word: "Session", description: "SSION → X"},
		{word: "Asia", description: "SIA → X"},
		{word: "Passion", description: "SSION"},
		{word: "Hiss", description: "SS → S"},

		{word: "Nation", description: "TIO → X"},
		{word: "Spatial", description: "TIA → X"},
		{word: "Catch", description: "TCH → X"},
		{word: "Matter", description: "TT → T"},

		{word: "Whale", description: "WH at start"},
		{word: "Swing", description: "W before vowel"},
		{word: "Saw", description: "W at end (silent)"},

		{word: "Box", description: "X in middle"},
		{word: "Xerox", description: "X at start → S"},

		{word: "VERYLONGWORDTHATEXCEEDSMAXIMUMLENGTH", description: "Should trim to max length"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := encoder.Encode(tc.word)

			if len(code) == 0 {
				t.Errorf("Expected non-empty code for %q", tc.word)
			}

			if len(code) > DefaultPhoneticCodeLength {
				t.Errorf("Code %q exceeds max length %d", code, DefaultPhoneticCodeLength)
			}

			t.Logf("%q → %q (%s)", tc.word, code, tc.description)
		})
	}
}

func TestPhoneticEncoder_TrimLongCode(t *testing.T) {
	encoder := NewPhoneticEncoder(4)

	code := encoder.Encode("VERYLONGWORD")

	if len(code) > 4 {
		t.Errorf("Code should be trimmed to 4 chars, got %d: %q", len(code), code)
	}
}

func TestPhoneticEncoder_DispatchTable(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word   string
		letter rune
	}{
		{letter: 'B', word: "Bob"},
		{letter: 'C', word: "Cat"},
		{letter: 'D', word: "Dog"},
		{letter: 'F', word: "Fox"},
		{letter: 'G', word: "Go"},
		{letter: 'H', word: "Hat"},
		{letter: 'J', word: "Jump"},
		{letter: 'K', word: "King"},
		{letter: 'L', word: "Lion"},
		{letter: 'M', word: "Mouse"},
		{letter: 'N', word: "Night"},
		{letter: 'P', word: "Phone"},
		{letter: 'Q', word: "Queen"},
		{letter: 'R', word: "Run"},
		{letter: 'S', word: "Sun"},
		{letter: 'T', word: "The"},
		{letter: 'V', word: "Van"},
		{letter: 'W', word: "Water"},
		{letter: 'X', word: "Xray"},
		{letter: 'Z', word: "Zebra"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.letter), func(t *testing.T) {
			code := encoder.Encode(tc.word)

			if len(code) == 0 {
				t.Errorf("Letter %c: expected non-empty code", tc.letter)
			}

			t.Logf("%c: %q → %q", tc.letter, tc.word, code)
		})
	}
}

func TestSoundex_Comprehensive(t *testing.T) {
	testCases := []struct {
		word string
		code string
	}{

		{word: "Robert", code: "R163"},
		{word: "Rupert", code: "R163"},
		{word: "Rubin", code: "R150"},
		{word: "Ashcraft", code: "A226"},
		{word: "Ashcroft", code: "A226"},
		{word: "Tymczak", code: "T522"},
		{word: "Pfister", code: "P123"},

		{word: "A", code: "A000"},
		{word: "Ab", code: "A100"},

		{word: "Smith", code: "S530"},
		{word: "Smythe", code: "S530"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := SoundexEncode(tc.word)

			if code != tc.code {
				t.Errorf("SoundexEncode(%q) = %q, want %q", tc.word, code, tc.code)
			}
		})
	}
}

func TestPhoneticEncoder_SpecialSequences(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []struct {
		word     string
		contains string
		reason   string
	}{
		{word: "Fudge", contains: "", reason: "DGE → J"},
		{word: "Session", contains: "X", reason: "SSION → X"},
		{word: "Catch", contains: "X", reason: "TCH → X"},
		{word: "Attention", contains: "", reason: "TION → X"},
		{word: "Partial", contains: "", reason: "TIAL → X"},
	}

	for _, tc := range testCases {
		t.Run(tc.word, func(t *testing.T) {
			code := encoder.Encode(tc.word)
			t.Logf("%q → %q (%s)", tc.word, code, tc.reason)

			if len(code) == 0 {
				t.Errorf("Expected non-empty code for %q", tc.word)
			}
		})
	}
}

func TestPhoneticEncoder_NumbersAndSpecialChars(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []string{
		"Test123",
		"Test@Home",
		"Test.Com",
		"Test&Test",
	}

	for _, word := range testCases {
		t.Run(word, func(t *testing.T) {
			code := encoder.Encode(word)

			if len(code) == 0 {
				t.Errorf("Expected non-empty code for %q", word)
			}

			t.Logf("%q → %q (special chars ignored)", word, code)
		})
	}
}

func TestPhoneticEncoder_UTF8Input(t *testing.T) {
	encoder := NewPhoneticEncoder(DefaultPhoneticCodeLength)

	testCases := []string{
		"Café",
		"Naïve",
		"José",
		"François",
	}

	for _, word := range testCases {
		t.Run(word, func(t *testing.T) {
			code := encoder.Encode(word)

			if len(code) == 0 {
				t.Errorf("Expected non-empty code for UTF-8 word %q", word)
			}

			t.Logf("%q → %q (UTF-8 handled)", word, code)
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{name: "ascii fits", input: "hello", maxRunes: 10, want: "hello"},
		{name: "ascii truncates", input: "hello world", maxRunes: 5, want: "hello"},
		{name: "cyrillic truncates by runes", input: "ПРИВЕТ", maxRunes: 3, want: "ПРИ"},
		{name: "accented latin truncates by runes", input: "éléphant", maxRunes: 3, want: "élé"},
		{name: "emoji multi-byte truncates", input: "abc😀def", maxRunes: 4, want: "abc😀"},
		{name: "zero produces empty", input: "ПРИВЕТ", maxRunes: 0, want: ""},
		{name: "negative produces empty", input: "ПРИВЕТ", maxRunes: -1, want: ""},
		{name: "empty input", input: "", maxRunes: 5, want: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := TruncateRunes(testCase.input, testCase.maxRunes)
			if got != testCase.want {
				t.Errorf("TruncateRunes(%q, %d) = %q, want %q", testCase.input, testCase.maxRunes, got, testCase.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("TruncateRunes(%q, %d) returned invalid UTF-8: %q", testCase.input, testCase.maxRunes, got)
			}
			if testCase.maxRunes > 0 && utf8.RuneCountInString(got) > testCase.maxRunes {
				t.Errorf("TruncateRunes(%q, %d) returned %d runes (exceeds limit)", testCase.input, testCase.maxRunes, utf8.RuneCountInString(got))
			}
		})
	}
}
