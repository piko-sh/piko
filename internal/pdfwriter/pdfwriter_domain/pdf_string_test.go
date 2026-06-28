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

package pdfwriter_domain

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodePdfTextStringASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "Hello World", want: "(Hello World)"},
		{name: "parentheses", input: "a (b) c", want: "(a \\(b\\) c)"},
		{name: "backslash", input: "a\\b", want: "(a\\\\b)"},
		{name: "empty", input: "", want: "()"},
		{name: "digits and symbols", input: "v1.2 #3 @x!", want: "(v1.2 #3 @x!)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := encodePdfTextString(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestEncodePdfTextStringUTF16(t *testing.T) {
	tests := []struct {
		name  string
		input string

		want string
	}{
		{name: "middle dot only", input: "·", want: "<FEFF00B7>"},
		{name: "right single quote", input: "I’m", want: "<FEFF00492019006D>"},
		{name: "emoji astral", input: "😀", want: "<FEFFD83DDE00>"},
		{name: "title with em dash", input: "Michael Haddon — CV"},
		{name: "mixed bmp and astral", input: "x𝐀y·"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := encodePdfTextString(test.input)
			require.True(t, strings.HasPrefix(got, "<FEFF") && strings.HasSuffix(got, ">"), "encodePdfTextString(%q) = %q, want UTF-16BE hex token", test.input, got)
			if test.want != "" {
				assert.Equal(t, test.want, got)
			}
			assert.Equal(t, test.input, decodeHexTextString(t, got), "round-trip of %q", test.input)
		})
	}
}

func TestEncodePdfTextStringEmDashRoundTrip(t *testing.T) {
	const title = "Michael Haddon — CV"
	encoded := encodePdfTextString(title)
	require.True(t, strings.HasPrefix(encoded, "<FEFF"), "expected UTF-16BE hex string, got %q", encoded)
	assert.Contains(t, encoded, "2014", "expected em dash codepoint 2014")
	assert.Equal(t, title, decodeHexTextString(t, encoded), "title round-trip")
}

func TestIsASCIIPrintable(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "Hello", want: true},
		{input: " ~", want: true},
		{input: "tab\there", want: false},
		{input: "newline\n", want: false},
		{input: "café", want: false},
		{input: "—", want: false},
		{input: "", want: true},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, isASCIIPrintable(test.input), "isASCIIPrintable(%q)", test.input)
	}
}

func TestUTF16BEUnitsHex(t *testing.T) {
	tests := []struct {
		input rune
		want  string
	}{
		{input: 'A', want: "0041"},
		{input: '—', want: "2014"},
		{input: '·', want: "00B7"},
		{input: '😀', want: "D83DDE00"},
		{input: '𝐀', want: "D835DC00"},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, utf16BEUnitsHex(test.input), "utf16BEUnitsHex(%q)", test.input)
	}
}

func decodeHexTextString(t *testing.T, token string) string {
	t.Helper()
	trimmed := strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">")
	raw, err := hex.DecodeString(trimmed)
	require.NoError(t, err, "hex decode of %q failed", token)
	require.GreaterOrEqual(t, len(raw), fieldSize16, "hex string %q too short to hold a byte-order mark", token)
	units := make([]uint16, 0, (len(raw)-fieldSize16)/fieldSize16)
	for byteIndex := fieldSize16; byteIndex+1 < len(raw); byteIndex += fieldSize16 {
		units = append(units, binary.BigEndian.Uint16(raw[byteIndex:]))
	}
	return string(utf16.Decode(units))
}
