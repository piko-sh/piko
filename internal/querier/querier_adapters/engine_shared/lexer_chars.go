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

package engine_shared

import (
	"unicode"
	"unicode/utf8"
)

// IsIdentStart reports whether the rune may begin an identifier. ASCII letters and the
// underscore take a fast path; runes beyond the ASCII range are accepted when
// unicode.IsLetter reports them as letters, so Unicode-named identifiers begin correctly.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for letters, underscore, or non-ASCII letters.
func IsIdentStart(character rune) bool {
	if character < utf8.RuneSelf {
		return (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			character == '_'
	}
	return unicode.IsLetter(character)
}

// IsIdentPart reports whether the rune may appear within an identifier after the first
// character, namely an identifier-start rune or a digit.
//
// ASCII runes take a fast path; runes beyond the ASCII range are accepted when
// unicode.IsLetter or unicode.IsDigit reports them, so Unicode letters and digits inside
// an identifier body are preserved.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for identifier-start runes or decimal digits.
func IsIdentPart(character rune) bool {
	if character < utf8.RuneSelf {
		return IsIdentStart(character) || (character >= '0' && character <= '9')
	}
	return unicode.IsLetter(character) || unicode.IsDigit(character)
}

// IsDigit reports whether the byte is an ASCII decimal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for bytes in the range '0' to '9'.
func IsDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// IsHexDigit reports whether the byte is an ASCII hexadecimal digit in either case.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for '0'-'9', 'a'-'f', or 'A'-'F'.
func IsHexDigit(character byte) bool {
	return IsDigit(character) ||
		(character >= 'a' && character <= 'f') ||
		(character >= 'A' && character <= 'F')
}

// IsOctalDigit reports whether the byte is an ASCII octal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for bytes in the range '0' to '7'.
func IsOctalDigit(character byte) bool {
	return character >= '0' && character <= '7'
}

// IsBinaryDigit reports whether the byte is an ASCII binary digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for '0' or '1'.
func IsBinaryDigit(character byte) bool {
	return character == '0' || character == '1'
}
