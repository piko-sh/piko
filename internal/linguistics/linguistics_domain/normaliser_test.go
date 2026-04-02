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
)

func TestNormaliser_DiacriticRemoval(t *testing.T) {
	normaliser := NewNormaliser(false)

	testCases := []struct {
		input    string
		expected string
	}{

		{input: "café", expected: "cafe"},
		{input: "naïve", expected: "naive"},
		{input: "résumé", expected: "resume"},
		{input: "François", expected: "francois"},

		{input: "niño", expected: "nino"},
		{input: "mañana", expected: "manana"},
		{input: "José", expected: "jose"},

		{input: "über", expected: "uber"},
		{input: "Müller", expected: "muller"},
		{input: "Fräulein", expected: "fraulein"},

		{input: "São Paulo", expected: "sao paulo"},
		{input: "ação", expected: "acao"},

		{input: "Åse", expected: "ase"},

		{input: "Zürich", expected: "zurich"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normaliser.Normalise(tc.input)
			if result != tc.expected {
				t.Errorf("Normalise(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestNormaliser_CaseFolding(t *testing.T) {
	normaliser := NewNormaliser(false)

	testCases := []struct {
		input    string
		expected string
	}{
		{input: "HELLO", expected: "hello"},
		{input: "MixedCase", expected: "mixedcase"},
		{input: "CAFÉ", expected: "cafe"},
		{input: "İstanbul", expected: "istanbul"},
	}

	for _, tc := range testCases {
		result := normaliser.Normalise(tc.input)
		if result != tc.expected {
			t.Errorf("Normalise(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormaliser_PreserveCase(t *testing.T) {
	normaliser := NewNormaliser(true)

	testCases := []struct {
		input    string
		expected string
	}{
		{input: "Hello", expected: "Hello"},
		{input: "WORLD", expected: "WORLD"},
		{input: "café", expected: "cafe"},
	}

	for _, tc := range testCases {
		result := normaliser.Normalise(tc.input)
		if result != tc.expected {
			t.Errorf("Normalise(%q) with PreserveCase = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormaliser_NormaliseRune(t *testing.T) {
	normaliser := NewNormaliser(false)

	testCases := []struct {
		input    rune
		expected rune
	}{
		{input: 'A', expected: 'a'},
		{input: 'Z', expected: 'z'},
		{input: 'a', expected: 'a'},
		{input: '5', expected: '5'},
	}

	for _, tc := range testCases {
		result := normaliser.NormaliseRune(tc.input)
		if result != tc.expected {
			t.Errorf("NormaliseRune(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormaliser_EmptyString(t *testing.T) {
	normaliser := NewNormaliser(false)

	result := normaliser.Normalise("")

	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func Test_isWordChar(t *testing.T) {
	testCases := []struct {
		char     rune
		expected bool
	}{
		{char: 'a', expected: true},
		{char: 'Z', expected: true},
		{char: '5', expected: true},
		{char: '_', expected: true},
		{char: '-', expected: true},
		{char: ' ', expected: false},
		{char: '.', expected: false},
		{char: '!', expected: false},
		{char: '@', expected: false},
		{char: '你', expected: true},
	}

	for _, tc := range testCases {
		result := isWordChar(tc.char)
		if result != tc.expected {
			t.Errorf("isWordChar(%q) = %v, want %v", tc.char, result, tc.expected)
		}
	}
}

func Test_isSeparator(t *testing.T) {
	testCases := []struct {
		char     rune
		expected bool
	}{
		{char: ' ', expected: true},
		{char: '\t', expected: true},
		{char: '\n', expected: true},
		{char: '.', expected: true},
		{char: '!', expected: true},
		{char: 'a', expected: false},
		{char: '5', expected: false},
	}

	for _, tc := range testCases {
		result := isSeparator(tc.char)
		if result != tc.expected {
			t.Errorf("isSeparator(%q) = %v, want %v", tc.char, result, tc.expected)
		}
	}
}

func TestNormaliser_UnicodeNormalisation(t *testing.T) {
	normaliser := NewNormaliser(false)

	nfc := "café"
	nfd := "cafe\u0301"

	resultNFC := normaliser.Normalise(nfc)
	resultNFD := normaliser.Normalise(nfd)

	if resultNFC != resultNFD {
		t.Errorf("NFC and NFD should normalise to same result: %q vs %q", resultNFC, resultNFD)
	}

	expected := "cafe"
	if resultNFC != expected {
		t.Errorf("Expected %q, got %q", expected, resultNFC)
	}
}

func BenchmarkNormaliser(b *testing.B) {
	normaliser := NewNormaliser(false)
	text := "Thé qüick bröwn föx jümps øver thé låzy dög with café and naïve résumé"

	b.ResetTimer()
	for b.Loop() {
		normaliser.Normalise(text)
	}
}

func BenchmarkNormaliser_ASCII(b *testing.B) {
	normaliser := NewNormaliser(false)
	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for b.Loop() {
		normaliser.Normalise(text)
	}
}
