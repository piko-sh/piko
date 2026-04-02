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

package lsp_domain

import (
	"math"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/sfcparser"
)

const floatTolerance = 0.01

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestParseHexColor(t *testing.T) {
	testCases := []struct {
		name    string
		hex     string
		expectR float64
		expectG float64
		expectB float64
		expectA float64
	}{
		{
			name:    "short hex RGB",
			hex:     "fff",
			expectR: 1.0,
			expectG: 1.0,
			expectB: 1.0,
			expectA: 1.0,
		},
		{
			name:    "short hex black",
			hex:     "000",
			expectR: 0.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "short hex red",
			hex:     "f00",
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "standard hex white",
			hex:     "ffffff",
			expectR: 1.0,
			expectG: 1.0,
			expectB: 1.0,
			expectA: 1.0,
		},
		{
			name:    "standard hex black",
			hex:     "000000",
			expectR: 0.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "standard hex red",
			hex:     "ff0000",
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "standard hex green",
			hex:     "00ff00",
			expectR: 0.0,
			expectG: 1.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "standard hex blue",
			hex:     "0000ff",
			expectR: 0.0,
			expectG: 0.0,
			expectB: 1.0,
			expectA: 1.0,
		},
		{
			name:    "hex with alpha fully opaque",
			hex:     "ff0000ff",
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 1.0,
		},
		{
			name:    "hex with alpha semi-transparent",
			hex:     "ff000080",
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 0.5,
		},
		{
			name:    "hex with alpha fully transparent",
			hex:     "ff000000",
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
			expectA: 0.0,
		},
		{
			name:    "uppercase hex",
			hex:     "AABBCC",
			expectR: 0.667,
			expectG: 0.733,
			expectB: 0.8,
			expectA: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, a := parseHexColor(tc.hex)

			if !floatEqual(r, tc.expectR) {
				t.Errorf("r = %f, want %f", r, tc.expectR)
			}
			if !floatEqual(g, tc.expectG) {
				t.Errorf("g = %f, want %f", g, tc.expectG)
			}
			if !floatEqual(b, tc.expectB) {
				t.Errorf("b = %f, want %f", b, tc.expectB)
			}
			if !floatEqual(a, tc.expectA) {
				t.Errorf("a = %f, want %f", a, tc.expectA)
			}
		})
	}
}

func TestHexCharToInt(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected int
	}{
		{name: "digit 0", char: '0', expected: 0},
		{name: "digit 5", char: '5', expected: 5},
		{name: "digit 9", char: '9', expected: 9},
		{name: "lowercase a", char: 'a', expected: 10},
		{name: "lowercase f", char: 'f', expected: 15},
		{name: "uppercase A", char: 'A', expected: 10},
		{name: "uppercase F", char: 'F', expected: 15},
		{name: "invalid char", char: 'g', expected: 0},
		{name: "space", char: ' ', expected: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hexCharToInt(tc.char)
			if result != tc.expected {
				t.Errorf("hexCharToInt(%c) = %d, want %d", tc.char, result, tc.expected)
			}
		})
	}
}

func TestIsHexDigit(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{name: "digit 0", char: '0', expected: true},
		{name: "digit 9", char: '9', expected: true},
		{name: "lowercase a", char: 'a', expected: true},
		{name: "lowercase f", char: 'f', expected: true},
		{name: "uppercase A", char: 'A', expected: true},
		{name: "uppercase F", char: 'F', expected: true},
		{name: "lowercase g", char: 'g', expected: false},
		{name: "uppercase G", char: 'G', expected: false},
		{name: "space", char: ' ', expected: false},
		{name: "hash", char: '#', expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isHexDigit(tc.char)
			if result != tc.expected {
				t.Errorf("isHexDigit(%c) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestIsValidHexLength(t *testing.T) {
	testCases := []struct {
		name     string
		length   int
		expected bool
	}{
		{name: "short hex (3)", length: 3, expected: true},
		{name: "standard hex (6)", length: 6, expected: true},
		{name: "hex with alpha (8)", length: 8, expected: true},
		{name: "too short (2)", length: 2, expected: false},
		{name: "invalid (4)", length: 4, expected: false},
		{name: "invalid (5)", length: 5, expected: false},
		{name: "invalid (7)", length: 7, expected: false},
		{name: "too long (9)", length: 9, expected: false},
		{name: "zero", length: 0, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidHexLength(tc.length)
			if result != tc.expected {
				t.Errorf("isValidHexLength(%d) = %v, want %v", tc.length, result, tc.expected)
			}
		})
	}
}

func TestParseColorArgs(t *testing.T) {
	testCases := []struct {
		name            string
		argumentsString string
		expected        []string
	}{
		{
			name:            "comma separated",
			argumentsString: "255, 128, 64",
			expected:        []string{"255", "128", "64"},
		},
		{
			name:            "space separated",
			argumentsString: "255 128 64",
			expected:        []string{"255", "128", "64"},
		},
		{
			name:            "comma separated with alpha",
			argumentsString: "255, 128, 64, 0.5",
			expected:        []string{"255", "128", "64", "0.5"},
		},
		{
			name:            "no spaces",
			argumentsString: "255,128,64",
			expected:        []string{"255", "128", "64"},
		},
		{
			name:            "extra whitespace",
			argumentsString: "  255 ,  128  , 64  ",
			expected:        []string{"255", "128", "64"},
		},
		{
			name:            "single value",
			argumentsString: "100",
			expected:        []string{"100"},
		},
		{
			name:            "empty string",
			argumentsString: "",
			expected:        []string{},
		},
		{
			name:            "percentages",
			argumentsString: "100%, 50%, 25%",
			expected:        []string{"100%", "50%", "25%"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseColorArgs(tc.argumentsString)

			if len(result) != len(tc.expected) {
				t.Errorf("parseColorArgs(%q) returned %d arguments, want %d", tc.argumentsString, len(result), len(tc.expected))
				return
			}

			for i := range tc.expected {
				if result[i] != tc.expected[i] {
					t.Errorf("parseColorArgs(%q)[%d] = %q, want %q", tc.argumentsString, i, result[i], tc.expected[i])
				}
			}
		})
	}
}

func TestParseColorComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		maxValue int
		expected float64
	}{
		{
			name:     "integer value",
			input:    "255",
			maxValue: 255,
			expected: 1.0,
		},
		{
			name:     "half value",
			input:    "127",
			maxValue: 255,
			expected: 0.498,
		},
		{
			name:     "zero",
			input:    "0",
			maxValue: 255,
			expected: 0.0,
		},
		{
			name:     "percentage 100",
			input:    "100%",
			maxValue: 255,
			expected: 1.0,
		},
		{
			name:     "percentage 50",
			input:    "50%",
			maxValue: 255,
			expected: 0.5,
		},
		{
			name:     "percentage 0",
			input:    "0%",
			maxValue: 255,
			expected: 0.0,
		},
		{
			name:     "with whitespace",
			input:    "  128  ",
			maxValue: 255,
			expected: 0.502,
		},
		{
			name:     "maxValue zero returns raw value",
			input:    "42",
			maxValue: 0,
			expected: 42.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseColorComponent(tc.input, tc.maxValue)
			if !floatEqual(result, tc.expected) {
				t.Errorf("parseColorComponent(%q, %d) = %f, want %f", tc.input, tc.maxValue, result, tc.expected)
			}
		})
	}
}

func TestParseAlphaComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "decimal 1",
			input:    "1",
			expected: 1.0,
		},
		{
			name:     "decimal 0.5",
			input:    "0.5",
			expected: 0.5,
		},
		{
			name:     "decimal 0",
			input:    "0",
			expected: 0.0,
		},
		{
			name:     "percentage 100",
			input:    "100%",
			expected: 1.0,
		},
		{
			name:     "percentage 50",
			input:    "50%",
			expected: 0.5,
		},
		{
			name:     "percentage 0",
			input:    "0%",
			expected: 0.0,
		},
		{
			name:     "with whitespace",
			input:    "  0.75  ",
			expected: 0.75,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseAlphaComponent(tc.input)
			if !floatEqual(result, tc.expected) {
				t.Errorf("parseAlphaComponent(%q) = %f, want %f", tc.input, result, tc.expected)
			}
		})
	}
}

func TestParseHueComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "zero degrees",
			input:    "0",
			expected: 0.0,
		},
		{
			name:     "180 degrees",
			input:    "180",
			expected: 0.5,
		},
		{
			name:     "360 degrees wraps to 0",
			input:    "360",
			expected: 0.0,
		},
		{
			name:     "with deg suffix",
			input:    "90deg",
			expected: 0.25,
		},
		{
			name:     "negative wraps",
			input:    "-90",
			expected: 0.75,
		},
		{
			name:     "over 360 wraps",
			input:    "450",
			expected: 0.25,
		},
		{
			name:     "with whitespace",
			input:    "  120  ",
			expected: 0.333,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseHueComponent(tc.input)
			if !floatEqual(result, tc.expected) {
				t.Errorf("parseHueComponent(%q) = %f, want %f", tc.input, result, tc.expected)
			}
		})
	}
}

func TestParsePercentComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "100 percent",
			input:    "100%",
			expected: 1.0,
		},
		{
			name:     "50 percent",
			input:    "50%",
			expected: 0.5,
		},
		{
			name:     "0 percent",
			input:    "0%",
			expected: 0.0,
		},
		{
			name:     "without percent sign",
			input:    "0.5",
			expected: 0.5,
		},
		{
			name:     "with whitespace",
			input:    "  75%  ",
			expected: 0.75,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parsePercentComponent(tc.input)
			if !floatEqual(result, tc.expected) {
				t.Errorf("parsePercentComponent(%q) = %f, want %f", tc.input, result, tc.expected)
			}
		})
	}
}

func TestHslToRGB(t *testing.T) {
	testCases := []struct {
		name    string
		h, s, l float64
		expectR float64
		expectG float64
		expectB float64
	}{
		{
			name:    "red",
			h:       0.0,
			s:       1.0,
			l:       0.5,
			expectR: 1.0,
			expectG: 0.0,
			expectB: 0.0,
		},
		{
			name:    "green",
			h:       0.333,
			s:       1.0,
			l:       0.5,
			expectR: 0.0,
			expectG: 1.0,
			expectB: 0.0,
		},
		{
			name:    "blue",
			h:       0.667,
			s:       1.0,
			l:       0.5,
			expectR: 0.0,
			expectG: 0.0,
			expectB: 1.0,
		},
		{
			name:    "white",
			h:       0.0,
			s:       0.0,
			l:       1.0,
			expectR: 1.0,
			expectG: 1.0,
			expectB: 1.0,
		},
		{
			name:    "black",
			h:       0.0,
			s:       0.0,
			l:       0.0,
			expectR: 0.0,
			expectG: 0.0,
			expectB: 0.0,
		},
		{
			name:    "gray",
			h:       0.0,
			s:       0.0,
			l:       0.5,
			expectR: 0.5,
			expectG: 0.5,
			expectB: 0.5,
		},
		{
			name:    "dark red with low lightness triggers chroma branch",
			h:       0.0,
			s:       1.0,
			l:       0.25,
			expectR: 0.5,
			expectG: 0.0,
			expectB: 0.0,
		},
		{
			name:    "light blue with high lightness",
			h:       0.667,
			s:       1.0,
			l:       0.75,
			expectR: 0.5,
			expectG: 0.5,
			expectB: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b := hslToRGB(tc.h, tc.s, tc.l)

			if !floatEqual(r, tc.expectR) {
				t.Errorf("r = %f, want %f", r, tc.expectR)
			}
			if !floatEqual(g, tc.expectG) {
				t.Errorf("g = %f, want %f", g, tc.expectG)
			}
			if !floatEqual(b, tc.expectB) {
				t.Errorf("b = %f, want %f", b, tc.expectB)
			}
		})
	}
}

func TestHueToRGB(t *testing.T) {
	testCases := []struct {
		name     string
		p        float64
		q        float64
		t        float64
		expected float64
	}{
		{
			name:     "t less than zero wraps around",
			p:        0.0,
			q:        1.0,
			t:        -0.5,
			expected: 1.0,
		},
		{
			name:     "t greater than one wraps down",
			p:        0.0,
			q:        1.0,
			t:        1.5,
			expected: 1.0,
		},
		{
			name:     "t in first segment below one sixth",
			p:        0.0,
			q:        1.0,
			t:        0.1,
			expected: 0.6,
		},
		{
			name:     "t in second segment below one half",
			p:        0.0,
			q:        0.8,
			t:        0.3,
			expected: 0.8,
		},
		{
			name:     "t in third segment below two thirds",
			p:        0.2,
			q:        0.8,
			t:        0.6,
			expected: 0.44,
		},
		{
			name:     "t in final segment returns p",
			p:        0.3,
			q:        0.9,
			t:        0.8,
			expected: 0.3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hueToRGB(tc.p, tc.q, tc.t)
			if !floatEqual(result, tc.expected) {
				t.Errorf("hueToRGB(%f, %f, %f) = %f, want %f", tc.p, tc.q, tc.t, result, tc.expected)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{name: "integer", input: "42", expected: 42.0},
		{name: "decimal", input: "3.14", expected: 3.14},
		{name: "negative", input: "-5.5", expected: -5.5},
		{name: "positive sign", input: "+10", expected: 10.0},
		{name: "zero", input: "0", expected: 0.0},
		{name: "decimal zero", input: "0.0", expected: 0.0},
		{name: "with whitespace", input: "  123  ", expected: 123.0},
		{name: "empty string", input: "", expected: 0.0},
		{name: "leading decimal", input: ".5", expected: 0.5},
		{name: "trailing decimal", input: "5.", expected: 5.0},
		{name: "number with trailing non-digit chars", input: "42px", expected: 42.0},
		{name: "decimal with trailing non-digit chars", input: "3.14em", expected: 3.14},
		{name: "multiple dots only first is parsed", input: "1.2.3", expected: 1.2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFloat(tc.input)
			if !floatEqual(result, tc.expected) {
				t.Errorf("parseFloat(%q) = %f, want %f", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTrimWhitespace(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no whitespace", input: "hello", expected: "hello"},
		{name: "leading space", input: "  hello", expected: "hello"},
		{name: "trailing space", input: "hello  ", expected: "hello"},
		{name: "both sides", input: "  hello  ", expected: "hello"},
		{name: "tabs", input: "\thello\t", expected: "hello"},
		{name: "newlines", input: "\nhello\n", expected: "hello"},
		{name: "mixed whitespace", input: " \t\n hello \n\t ", expected: "hello"},
		{name: "empty string", input: "", expected: ""},
		{name: "only whitespace", input: "   ", expected: ""},
		{name: "internal whitespace preserved", input: "  hello world  ", expected: "hello world"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := trimWhitespace(tc.input)
			if result != tc.expected {
				t.Errorf("trimWhitespace(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsWhitespace(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{name: "space", char: ' ', expected: true},
		{name: "tab", char: '\t', expected: true},
		{name: "newline", char: '\n', expected: true},
		{name: "carriage return", char: '\r', expected: true},
		{name: "letter", char: 'a', expected: false},
		{name: "digit", char: '5', expected: false},
		{name: "zero byte", char: 0, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isWhitespace(tc.char)
			if result != tc.expected {
				t.Errorf("isWhitespace(%d) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestContainsChar(t *testing.T) {
	testCases := []struct {
		name      string
		s         string
		character byte
		expected  bool
	}{
		{name: "contains char", s: "hello", character: 'l', expected: true},
		{name: "does not contain", s: "hello", character: 'x', expected: false},
		{name: "empty string", s: "", character: 'a', expected: false},
		{name: "char at start", s: "hello", character: 'h', expected: true},
		{name: "char at end", s: "hello", character: 'o', expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsChar(tc.s, tc.character)
			if result != tc.expected {
				t.Errorf("containsChar(%q, %c) = %v, want %v", tc.s, tc.character, result, tc.expected)
			}
		})
	}
}

func TestSplitByChar(t *testing.T) {
	testCases := []struct {
		name      string
		s         string
		expected  []string
		delimiter byte
	}{
		{
			name:      "comma separated",
			s:         "a,b,c",
			delimiter: ',',
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "no delimiter",
			s:         "abc",
			delimiter: ',',
			expected:  []string{"abc"},
		},
		{
			name:      "empty string",
			s:         "",
			delimiter: ',',
			expected:  []string{},
		},
		{
			name:      "empty parts",
			s:         "a,,b",
			delimiter: ',',
			expected:  []string{"a", "", "b"},
		},
		{
			name:      "trailing delimiter",
			s:         "a,b,",
			delimiter: ',',
			expected:  []string{"a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitByChar(tc.s, tc.delimiter)

			if len(result) != len(tc.expected) {
				t.Errorf("splitByChar(%q, %c) returned %d parts, want %d", tc.s, tc.delimiter, len(result), len(tc.expected))
				return
			}

			for i := range tc.expected {
				if result[i] != tc.expected[i] {
					t.Errorf("splitByChar(%q, %c)[%d] = %q, want %q", tc.s, tc.delimiter, i, result[i], tc.expected[i])
				}
			}
		})
	}
}

func TestSplitByWhitespace(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		expected []string
	}{
		{
			name:     "space separated",
			s:        "a b c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "multiple spaces",
			s:        "a   b   c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "tabs and spaces",
			s:        "a\tb\tc",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "leading and trailing whitespace",
			s:        "  a b c  ",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "no whitespace",
			s:        "abc",
			expected: []string{"abc"},
		},
		{
			name:     "empty string",
			s:        "",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			s:        "   ",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitByWhitespace(tc.s)

			if len(result) != len(tc.expected) {
				t.Errorf("splitByWhitespace(%q) returned %d parts, want %d", tc.s, len(result), len(tc.expected))
				return
			}

			for i := range tc.expected {
				if result[i] != tc.expected[i] {
					t.Errorf("splitByWhitespace(%q)[%d] = %q, want %q", tc.s, i, result[i], tc.expected[i])
				}
			}
		})
	}
}

func TestColorToHex(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		r        int
		g        int
		b        int
	}{
		{name: "white", r: 255, g: 255, b: 255, expected: "#ffffff"},
		{name: "black", r: 0, g: 0, b: 0, expected: "#000000"},
		{name: "red", r: 255, g: 0, b: 0, expected: "#ff0000"},
		{name: "green", r: 0, g: 255, b: 0, expected: "#00ff00"},
		{name: "blue", r: 0, g: 0, b: 255, expected: "#0000ff"},
		{name: "gray", r: 128, g: 128, b: 128, expected: "#808080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := colorToHex(tc.r, tc.g, tc.b)
			if result != tc.expected {
				t.Errorf("colorToHex(%d, %d, %d) = %q, want %q", tc.r, tc.g, tc.b, result, tc.expected)
			}
		})
	}
}

func TestColorToHexAlpha(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		r        int
		g        int
		b        int
		a        int
	}{
		{name: "white opaque", r: 255, g: 255, b: 255, a: 255, expected: "#ffffffff"},
		{name: "black transparent", r: 0, g: 0, b: 0, a: 0, expected: "#00000000"},
		{name: "red half alpha", r: 255, g: 0, b: 0, a: 128, expected: "#ff000080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := colorToHexAlpha(tc.r, tc.g, tc.b, tc.a)
			if result != tc.expected {
				t.Errorf("colorToHexAlpha(%d, %d, %d, %d) = %q, want %q", tc.r, tc.g, tc.b, tc.a, result, tc.expected)
			}
		})
	}
}

func TestColorToRGB(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		r        int
		g        int
		b        int
	}{
		{name: "white", r: 255, g: 255, b: 255, expected: "rgb(255, 255, 255)"},
		{name: "black", r: 0, g: 0, b: 0, expected: "rgb(0, 0, 0)"},
		{name: "red", r: 255, g: 0, b: 0, expected: "rgb(255, 0, 0)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := colorToRGB(tc.r, tc.g, tc.b)
			if result != tc.expected {
				t.Errorf("colorToRGB(%d, %d, %d) = %q, want %q", tc.r, tc.g, tc.b, result, tc.expected)
			}
		})
	}
}

func TestColorToRGBA(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		r        int
		g        int
		b        int
		a        float64
	}{
		{name: "white opaque", r: 255, g: 255, b: 255, a: 1.0, expected: "rgba(255, 255, 255, 1)"},
		{name: "black transparent", r: 0, g: 0, b: 0, a: 0.0, expected: "rgba(0, 0, 0, 0)"},
		{name: "red half alpha", r: 255, g: 0, b: 0, a: 0.5, expected: "rgba(255, 0, 0, 0.50)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := colorToRGBA(tc.r, tc.g, tc.b, tc.a)
			if result != tc.expected {
				t.Errorf("colorToRGBA(%d, %d, %d, %f) = %q, want %q", tc.r, tc.g, tc.b, tc.a, result, tc.expected)
			}
		})
	}
}

func TestIntToHex(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		n        int
	}{
		{name: "zero", n: 0, expected: "00"},
		{name: "15", n: 15, expected: "0f"},
		{name: "16", n: 16, expected: "10"},
		{name: "255", n: 255, expected: "ff"},
		{name: "128", n: 128, expected: "80"},
		{name: "negative clamped", n: -5, expected: "00"},
		{name: "over 255 clamped", n: 300, expected: "ff"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := intToHex(tc.n)
			if result != tc.expected {
				t.Errorf("intToHex(%d) = %q, want %q", tc.n, result, tc.expected)
			}
		})
	}
}

func TestIntToString(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		n        int
	}{
		{name: "zero", n: 0, expected: "0"},
		{name: "positive", n: 42, expected: "42"},
		{name: "negative", n: -5, expected: "-5"},
		{name: "large", n: 12345, expected: "12345"},
		{name: "255", n: 255, expected: "255"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := intToString(tc.n)
			if result != tc.expected {
				t.Errorf("intToString(%d) = %q, want %q", tc.n, result, tc.expected)
			}
		})
	}
}

func TestFloatToString(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		f        float64
	}{
		{name: "one", f: 1.0, expected: "1"},
		{name: "almost one", f: 0.99, expected: "1"},
		{name: "zero", f: 0.0, expected: "0"},
		{name: "almost zero", f: 0.005, expected: "0"},
		{name: "half", f: 0.5, expected: "0.50"},
		{name: "quarter", f: 0.25, expected: "0.25"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := floatToString(tc.f)
			if result != tc.expected {
				t.Errorf("floatToString(%f) = %q, want %q", tc.f, result, tc.expected)
			}
		})
	}
}

func TestConvertCharPosToLineCol(t *testing.T) {
	testCases := []struct {
		name       string
		content    string
		charPos    int
		expectLine int
		expectCol  int
	}{
		{
			name:       "start of content",
			content:    "hello\nworld",
			charPos:    0,
			expectLine: 0,
			expectCol:  0,
		},
		{
			name:       "middle of first line",
			content:    "hello\nworld",
			charPos:    3,
			expectLine: 0,
			expectCol:  3,
		},
		{
			name:       "start of second line",
			content:    "hello\nworld",
			charPos:    6,
			expectLine: 1,
			expectCol:  0,
		},
		{
			name:       "middle of second line",
			content:    "hello\nworld",
			charPos:    8,
			expectLine: 1,
			expectCol:  2,
		},
		{
			name:       "multiple lines",
			content:    "a\nb\nc\nd",
			charPos:    6,
			expectLine: 3,
			expectCol:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line, column := convertCharPosToLineColumn(tc.content, tc.charPos)
			if line != tc.expectLine {
				t.Errorf("line = %d, want %d", line, tc.expectLine)
			}
			if column != tc.expectCol {
				t.Errorf("column = %d, want %d", column, tc.expectCol)
			}
		})
	}
}

func TestFindClosingParen(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		start    int
		expected int
	}{
		{
			name:     "simple case",
			content:  "rgb(255, 0, 0)",
			start:    4,
			expected: 14,
		},
		{
			name:     "nested parens",
			content:  "rgb(calc(100 + 50), 0, 0)",
			start:    4,
			expected: 25,
		},
		{
			name:     "no closing paren",
			content:  "rgb(255, 0, 0",
			start:    4,
			expected: -1,
		},
		{
			name:     "empty parens",
			content:  "rgb()",
			start:    4,
			expected: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findClosingParen(tc.content, tc.start)
			if result != tc.expected {
				t.Errorf("findClosingParen(%q, %d) = %d, want %d", tc.content, tc.start, result, tc.expected)
			}
		})
	}
}

func TestMatchHexColor(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expectHex   string
		position    int
		expectStart int
		expectEnd   int
		expectNil   bool
	}{
		{
			name:      "not a hash",
			content:   "rgb(255)",
			position:  0,
			expectNil: true,
		},
		{
			name:        "valid short hex",
			content:     "#fff",
			position:    0,
			expectNil:   false,
			expectHex:   "fff",
			expectStart: 0,
			expectEnd:   4,
		},
		{
			name:        "valid standard hex",
			content:     "#ff0000",
			position:    0,
			expectNil:   false,
			expectHex:   "ff0000",
			expectStart: 0,
			expectEnd:   7,
		},
		{
			name:        "valid hex with alpha",
			content:     "#ff000080",
			position:    0,
			expectNil:   false,
			expectHex:   "ff000080",
			expectStart: 0,
			expectEnd:   9,
		},
		{
			name:      "invalid length",
			content:   "#ff00",
			position:  0,
			expectNil: true,
		},
		{
			name:        "hex in middle of string",
			content:     "color: #ff0000;",
			position:    7,
			expectNil:   false,
			expectHex:   "ff0000",
			expectStart: 7,
			expectEnd:   14,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchHexColor(tc.content, tc.position)

			if tc.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.hexValue != tc.expectHex {
				t.Errorf("hexValue = %q, want %q", result.hexValue, tc.expectHex)
			}
			if result.start != tc.expectStart {
				t.Errorf("start = %d, want %d", result.start, tc.expectStart)
			}
			if result.end != tc.expectEnd {
				t.Errorf("end = %d, want %d", result.end, tc.expectEnd)
			}
		})
	}
}

func TestParseRGBColor(t *testing.T) {
	testCases := []struct {
		match       *colorFuncMatch
		name        string
		expectColor protocol.Color
		expectOK    bool
	}{
		{
			name: "valid rgb",
			match: &colorFuncMatch{
				functionName:    "rgb",
				argumentsString: "255, 0, 0",
			},
			expectOK:    true,
			expectColor: protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
		},
		{
			name: "valid rgba",
			match: &colorFuncMatch{
				functionName:    "rgba",
				argumentsString: "255, 0, 0, 0.5",
			},
			expectOK:    true,
			expectColor: protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 0.5},
		},
		{
			name: "rgb wrong argument count",
			match: &colorFuncMatch{
				functionName:    "rgb",
				argumentsString: "255, 0",
			},
			expectOK: false,
		},
		{
			name: "rgba wrong argument count",
			match: &colorFuncMatch{
				functionName:    "rgba",
				argumentsString: "255, 0, 0",
			},
			expectOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			color, ok := parseRGBColor(tc.match)

			if ok != tc.expectOK {
				t.Errorf("ok = %v, want %v", ok, tc.expectOK)
			}

			if tc.expectOK {
				if !floatEqual(color.Red, tc.expectColor.Red) {
					t.Errorf("Red = %f, want %f", color.Red, tc.expectColor.Red)
				}
				if !floatEqual(color.Green, tc.expectColor.Green) {
					t.Errorf("Green = %f, want %f", color.Green, tc.expectColor.Green)
				}
				if !floatEqual(color.Blue, tc.expectColor.Blue) {
					t.Errorf("Blue = %f, want %f", color.Blue, tc.expectColor.Blue)
				}
				if !floatEqual(color.Alpha, tc.expectColor.Alpha) {
					t.Errorf("Alpha = %f, want %f", color.Alpha, tc.expectColor.Alpha)
				}
			}
		})
	}
}

func TestGetDocumentColors_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: &document{AnnotationResult: nil},
		},
		{
			name: "nil EntryPointStyleBlocks",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					EntryPointStyleBlocks: nil,
				},
			},
		},
		{
			name: "wrong type for EntryPointStyleBlocks",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					EntryPointStyleBlocks: "not a slice",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colors, err := tc.document.GetDocumentColors()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(colors) != 0 {
				t.Errorf("expected empty colours, got %d", len(colors))
			}
		})
	}
}

func TestGetDocumentColors_WithStyleBlocks(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			EntryPointStyleBlocks: []sfcparser.Style{
				{
					Content: "body { color: #ff0000; }",
					ContentLocation: sfcparser.Location{
						Line:   3,
						Column: 0,
					},
				},
			},
		},
	}

	colors, err := document.GetDocumentColors()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(colors) != 1 {
		t.Fatalf("expected 1 colour, got %d", len(colors))
	}

	if !floatEqual(colors[0].Color.Red, 1.0) {
		t.Errorf("Red = %f, want 1.0", colors[0].Color.Red)
	}
	if !floatEqual(colors[0].Color.Green, 0.0) {
		t.Errorf("Green = %f, want 0.0", colors[0].Color.Green)
	}
	if !floatEqual(colors[0].Color.Blue, 0.0) {
		t.Errorf("Blue = %f, want 0.0", colors[0].Color.Blue)
	}
}

func TestGetDocumentColors_EmptyStyleBlockContent(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			EntryPointStyleBlocks: []sfcparser.Style{
				{
					Content: "",
					ContentLocation: sfcparser.Location{
						Line:   3,
						Column: 0,
					},
				},
			},
		},
	}

	colors, err := document.GetDocumentColors()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(colors) != 0 {
		t.Errorf("expected 0 colours for empty content, got %d", len(colors))
	}
}

func TestGetDocumentColors_MultipleStyleBlocks(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			EntryPointStyleBlocks: []sfcparser.Style{
				{
					Content: "body { color: #ff0000; }",
					ContentLocation: sfcparser.Location{
						Line:   3,
						Column: 0,
					},
				},
				{
					Content: "p { color: rgb(0, 255, 0); }",
					ContentLocation: sfcparser.Location{
						Line:   10,
						Column: 0,
					},
				},
			},
		},
	}

	colors, err := document.GetDocumentColors()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(colors) != 2 {
		t.Fatalf("expected 2 colours, got %d", len(colors))
	}
}

func TestGetDocumentColors_MixedColorTypes(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			EntryPointStyleBlocks: []sfcparser.Style{
				{
					Content: "a { color: #00ff00; background: rgb(255, 0, 0); border: hsl(240, 100%, 50%); }",
					ContentLocation: sfcparser.Location{
						Line:   5,
						Column: 0,
					},
				},
			},
		},
	}

	colors, err := document.GetDocumentColors()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(colors) != 3 {
		t.Fatalf("expected 3 colours (hex + rgb + hsl), got %d", len(colors))
	}
}

func TestFindHexColorsWithOffset(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name      string
		content   string
		baseLine  int
		baseCol   int
		wantCount int
	}{
		{
			name:      "single hex colour",
			content:   "color: #ff0000;",
			baseLine:  0,
			baseCol:   0,
			wantCount: 1,
		},
		{
			name:      "multiple hex colours",
			content:   "#ff0000 #00ff00 #0000ff",
			baseLine:  0,
			baseCol:   0,
			wantCount: 3,
		},
		{
			name:      "no hex colours",
			content:   "color: red;",
			baseLine:  0,
			baseCol:   0,
			wantCount: 0,
		},
		{
			name:      "hex colour on second line",
			content:   "first\n#aabbcc",
			baseLine:  5,
			baseCol:   0,
			wantCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colors := document.findHexColorsWithOffset(tc.content, tc.baseLine, tc.baseCol)
			if len(colors) != tc.wantCount {
				t.Errorf("got %d colours, want %d", len(colors), tc.wantCount)
			}
		})
	}
}

func TestFindHexColorsWithOffset_PositionOffsets(t *testing.T) {
	document := &document{}

	colors := document.findHexColorsWithOffset("#ff0000", 10, 5)

	if len(colors) != 1 {
		t.Fatalf("expected 1 colour, got %d", len(colors))
	}

	if colors[0].Range.Start.Line != 10 {
		t.Errorf("Start.Line = %d, want 10", colors[0].Range.Start.Line)
	}
	if colors[0].Range.Start.Character != 5 {
		t.Errorf("Start.Character = %d, want 5", colors[0].Range.Start.Character)
	}
}

func TestFindRGBColorsWithOffset(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name      string
		content   string
		baseLine  int
		baseCol   int
		wantCount int
	}{
		{
			name:      "single rgb colour",
			content:   "color: rgb(255, 0, 0);",
			baseLine:  0,
			baseCol:   0,
			wantCount: 1,
		},
		{
			name:      "rgba colour",
			content:   "color: rgba(255, 0, 0, 0.5);",
			baseLine:  0,
			baseCol:   0,
			wantCount: 1,
		},
		{
			name:      "no rgb colours",
			content:   "color: #ff0000;",
			baseLine:  0,
			baseCol:   0,
			wantCount: 0,
		},
		{
			name:      "multiple rgb colours",
			content:   "rgb(255, 0, 0) rgb(0, 255, 0)",
			baseLine:  0,
			baseCol:   0,
			wantCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colors := document.findRGBColorsWithOffset(tc.content, tc.baseLine, tc.baseCol)
			if len(colors) != tc.wantCount {
				t.Errorf("got %d colours, want %d", len(colors), tc.wantCount)
			}
		})
	}
}

func TestFindHSLColorsWithOffset(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name      string
		content   string
		baseLine  int
		baseCol   int
		wantCount int
	}{
		{
			name:      "single hsl colour",
			content:   "color: hsl(0, 100%, 50%);",
			baseLine:  0,
			baseCol:   0,
			wantCount: 1,
		},
		{
			name:      "hsla colour",
			content:   "color: hsla(0, 100%, 50%, 0.5);",
			baseLine:  0,
			baseCol:   0,
			wantCount: 1,
		},
		{
			name:      "no hsl colours",
			content:   "color: #ff0000;",
			baseLine:  0,
			baseCol:   0,
			wantCount: 0,
		},
		{
			name:      "multiple hsl colours",
			content:   "hsl(0, 100%, 50%) hsl(120, 100%, 50%)",
			baseLine:  0,
			baseCol:   0,
			wantCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colors := document.findHSLColorsWithOffset(tc.content, tc.baseLine, tc.baseCol)
			if len(colors) != tc.wantCount {
				t.Errorf("got %d colours, want %d", len(colors), tc.wantCount)
			}
		})
	}
}

func TestMatchColorFunc(t *testing.T) {
	testCases := []struct {
		matcher    colorFuncMatcher
		name       string
		content    string
		expectFunc string
		expectArgs string
		position   int
		expectNil  bool
	}{
		{
			name:      "no match",
			content:   "color: red",
			position:  0,
			matcher:   rgbMatcher,
			expectNil: true,
		},
		{
			name:       "rgb match",
			content:    "rgb(255, 0, 0)",
			position:   0,
			matcher:    rgbMatcher,
			expectNil:  false,
			expectFunc: "rgb",
			expectArgs: "255, 0, 0",
		},
		{
			name:       "rgba match",
			content:    "rgba(255, 0, 0, 0.5)",
			position:   0,
			matcher:    rgbMatcher,
			expectNil:  false,
			expectFunc: "rgba",
			expectArgs: "255, 0, 0, 0.5",
		},
		{
			name:       "hsl match",
			content:    "hsl(0, 100%, 50%)",
			position:   0,
			matcher:    hslMatcher,
			expectNil:  false,
			expectFunc: "hsl",
			expectArgs: "0, 100%, 50%",
		},
		{
			name:       "hsla match",
			content:    "hsla(0, 100%, 50%, 0.5)",
			position:   0,
			matcher:    hslMatcher,
			expectNil:  false,
			expectFunc: "hsla",
			expectArgs: "0, 100%, 50%, 0.5",
		},
		{
			name:      "position past content length",
			content:   "rgb",
			position:  0,
			matcher:   rgbMatcher,
			expectNil: true,
		},
		{
			name:      "no closing paren",
			content:   "rgb(255, 0, 0",
			position:  0,
			matcher:   rgbMatcher,
			expectNil: true,
		},
		{
			name:       "match at offset position",
			content:    "color: rgb(128, 128, 128)",
			position:   7,
			matcher:    rgbMatcher,
			expectNil:  false,
			expectFunc: "rgb",
			expectArgs: "128, 128, 128",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchColorFunc(tc.content, tc.position, tc.matcher)

			if tc.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.functionName != tc.expectFunc {
				t.Errorf("functionName = %q, want %q", result.functionName, tc.expectFunc)
			}
			if result.argumentsString != tc.expectArgs {
				t.Errorf("argumentsString = %q, want %q", result.argumentsString, tc.expectArgs)
			}
		})
	}
}

func TestBuildColorInfo(t *testing.T) {
	testCases := []struct {
		match         *colorFuncMatch
		name          string
		content       string
		color         protocol.Color
		baseLine      int
		baseCol       int
		wantStartLine uint32
		wantStartChar uint32
		wantEndLine   uint32
		wantEndChar   uint32
	}{
		{
			name:    "single line with no offset",
			content: "rgb(255, 0, 0)",
			match: &colorFuncMatch{
				start: 0,
				end:   14,
			},
			baseLine:      0,
			baseCol:       0,
			color:         protocol.Color{Red: 1.0},
			wantStartLine: 0,
			wantStartChar: 0,
			wantEndLine:   0,
			wantEndChar:   14,
		},
		{
			name:    "single line with base offsets",
			content: "rgb(255, 0, 0)",
			match: &colorFuncMatch{
				start: 0,
				end:   14,
			},
			baseLine:      5,
			baseCol:       10,
			color:         protocol.Color{Red: 1.0},
			wantStartLine: 5,
			wantStartChar: 10,
			wantEndLine:   5,
			wantEndChar:   24,
		},
		{
			name:    "baseCol only applied to first line",
			content: "first\nrgb(255, 0, 0)",
			match: &colorFuncMatch{
				start: 6,
				end:   20,
			},
			baseLine:      5,
			baseCol:       10,
			color:         protocol.Color{Red: 1.0},
			wantStartLine: 6,
			wantStartChar: 0,
			wantEndLine:   6,
			wantEndChar:   14,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildColorInfo(tc.content, tc.match, tc.baseLine, tc.baseCol, tc.color)

			if result.Range.Start.Line != tc.wantStartLine {
				t.Errorf("Start.Line = %d, want %d", result.Range.Start.Line, tc.wantStartLine)
			}
			if result.Range.Start.Character != tc.wantStartChar {
				t.Errorf("Start.Character = %d, want %d", result.Range.Start.Character, tc.wantStartChar)
			}
			if result.Range.End.Line != tc.wantEndLine {
				t.Errorf("End.Line = %d, want %d", result.Range.End.Line, tc.wantEndLine)
			}
			if result.Range.End.Character != tc.wantEndChar {
				t.Errorf("End.Character = %d, want %d", result.Range.End.Character, tc.wantEndChar)
			}
		})
	}
}

func TestBuildHexColorInfo(t *testing.T) {
	content := "color: #ff0000;"
	match := &hexColorMatch{
		hexValue: "ff0000",
		start:    7,
		end:      14,
	}

	result := buildHexColorInfo(content, match, 0, 0)

	if !floatEqual(result.Color.Red, 1.0) {
		t.Errorf("Red = %f, want 1.0", result.Color.Red)
	}
	if !floatEqual(result.Color.Green, 0.0) {
		t.Errorf("Green = %f, want 0.0", result.Color.Green)
	}
	if !floatEqual(result.Color.Blue, 0.0) {
		t.Errorf("Blue = %f, want 0.0", result.Color.Blue)
	}

	if result.Range.Start.Character != 7 {
		t.Errorf("Start.Character = %d, want 7", result.Range.Start.Character)
	}
	if result.Range.End.Character != 14 {
		t.Errorf("End.Character = %d, want 14", result.Range.End.Character)
	}
}

func TestGetColorPresentations(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name       string
		wantLabels []string
		color      protocol.Color
		wantCount  int
	}{
		{
			name: "opaque red",
			color: protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			wantCount:  2,
			wantLabels: []string{"#ff0000", "rgb(255, 0, 0)"},
		},
		{
			name: "semi-transparent red",
			color: protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.5,
			},
			wantCount:  2,
			wantLabels: []string{"#ff00007f", "rgba(255, 0, 0, 0.50)"},
		},
		{
			name: "opaque white",
			color: protocol.Color{
				Red:   1.0,
				Green: 1.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			wantCount:  2,
			wantLabels: []string{"#ffffff", "rgb(255, 255, 255)"},
		},
		{
			name: "opaque black",
			color: protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			wantCount:  2,
			wantLabels: []string{"#000000", "rgb(0, 0, 0)"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			presentations, err := document.GetColorPresentations(tc.color)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(presentations) != tc.wantCount {
				t.Fatalf("got %d presentations, want %d", len(presentations), tc.wantCount)
			}

			for i, wantLabel := range tc.wantLabels {
				if presentations[i].Label != wantLabel {
					t.Errorf("presentations[%d].Label = %q, want %q", i, presentations[i].Label, wantLabel)
				}
			}
		})
	}
}

func TestMatchRGBFunc(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		position  int
		expectNil bool
	}{
		{
			name:      "matches rgb",
			content:   "rgb(255, 0, 0)",
			position:  0,
			expectNil: false,
		},
		{
			name:      "matches rgba",
			content:   "rgba(255, 0, 0, 1)",
			position:  0,
			expectNil: false,
		},
		{
			name:      "no match for hsl",
			content:   "hsl(0, 100%, 50%)",
			position:  0,
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchRGBFunc(tc.content, tc.position)
			if tc.expectNil && result != nil {
				t.Errorf("expected nil, got %+v", result)
			}
			if !tc.expectNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestMatchHSLFunc(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		position  int
		expectNil bool
	}{
		{
			name:      "matches hsl",
			content:   "hsl(0, 100%, 50%)",
			position:  0,
			expectNil: false,
		},
		{
			name:      "matches hsla",
			content:   "hsla(0, 100%, 50%, 1)",
			position:  0,
			expectNil: false,
		},
		{
			name:      "no match for rgb",
			content:   "rgb(255, 0, 0)",
			position:  0,
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchHSLFunc(tc.content, tc.position)
			if tc.expectNil && result != nil {
				t.Errorf("expected nil, got %+v", result)
			}
			if !tc.expectNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestParseHSLColor(t *testing.T) {
	testCases := []struct {
		match       *colorFuncMatch
		name        string
		expectColor protocol.Color
		expectOK    bool
	}{
		{
			name: "valid hsl red",
			match: &colorFuncMatch{
				functionName:    "hsl",
				argumentsString: "0, 100%, 50%",
			},
			expectOK:    true,
			expectColor: protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
		},
		{
			name: "valid hsla",
			match: &colorFuncMatch{
				functionName:    "hsla",
				argumentsString: "0, 100%, 50%, 0.5",
			},
			expectOK:    true,
			expectColor: protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 0.5},
		},
		{
			name: "hsl wrong argument count",
			match: &colorFuncMatch{
				functionName:    "hsl",
				argumentsString: "0, 100%",
			},
			expectOK: false,
		},
		{
			name: "hsla wrong argument count returns false",
			match: &colorFuncMatch{
				functionName:    "hsla",
				argumentsString: "0, 100%, 50%",
			},
			expectOK: false,
		},
		{
			name: "invalid functionName returns false",
			match: &colorFuncMatch{
				functionName:    "rgb",
				argumentsString: "0, 100%, 50%",
			},
			expectOK: false,
		},
		{
			name: "hsl with low lightness triggers chroma path",
			match: &colorFuncMatch{
				functionName:    "hsl",
				argumentsString: "0, 100%, 25%",
			},
			expectOK:    true,
			expectColor: protocol.Color{Red: 0.5, Green: 0.0, Blue: 0.0, Alpha: 1.0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			color, ok := parseHSLColor(tc.match)

			if ok != tc.expectOK {
				t.Errorf("ok = %v, want %v", ok, tc.expectOK)
			}

			if tc.expectOK {
				if !floatEqual(color.Red, tc.expectColor.Red) {
					t.Errorf("Red = %f, want %f", color.Red, tc.expectColor.Red)
				}
				if !floatEqual(color.Green, tc.expectColor.Green) {
					t.Errorf("Green = %f, want %f", color.Green, tc.expectColor.Green)
				}
				if !floatEqual(color.Blue, tc.expectColor.Blue) {
					t.Errorf("Blue = %f, want %f", color.Blue, tc.expectColor.Blue)
				}
				if !floatEqual(color.Alpha, tc.expectColor.Alpha) {
					t.Errorf("Alpha = %f, want %f", color.Alpha, tc.expectColor.Alpha)
				}
			}
		})
	}
}
