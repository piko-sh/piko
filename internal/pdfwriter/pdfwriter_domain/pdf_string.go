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
	"fmt"
	"strings"
)

const (
	// asciiPrintableMin is the lowest printable ASCII byte (space).
	asciiPrintableMin = 0x20

	// asciiPrintableMax is the highest printable ASCII byte (tilde).
	asciiPrintableMax = 0x7e

	// utf16SupplementaryBase is the first codepoint requiring a surrogate pair.
	utf16SupplementaryBase = 0x10000

	// utf16HighSurrogateBase is the start of the high-surrogate range.
	utf16HighSurrogateBase = 0xd800

	// utf16LowSurrogateBase is the start of the low-surrogate range.
	utf16LowSurrogateBase = 0xdc00

	// utf16SurrogateShift is the bit shift separating the surrogate halves.
	utf16SurrogateShift = 10

	// utf16LowSurrogateMask masks the low ten bits for the low surrogate.
	utf16LowSurrogateMask = 0x3ff

	// utf16ByteOrderMark is the big-endian byte-order mark that signals a PDF text string is
	// UTF-16, written as the opening of a hexadecimal string.
	utf16ByteOrderMark = "<FEFF"
)

// encodePdfTextString encodes a string as a complete PDF text-string token, including its
// delimiters. Strings containing only printable ASCII are emitted as an escaped literal
// string "(...)"; any string with a character outside printable ASCII is emitted as a
// UTF-16BE hexadecimal string "<FEFF...>" with a byte-order mark.
//
// Takes text (string) which is the raw string to encode.
//
// Returns string which is the delimited PDF text-string token.
func encodePdfTextString(text string) string {
	if isASCIIPrintable(text) {
		return "(" + pdfEscapeString(text) + ")"
	}
	return encodeUTF16BEHexString(text)
}

// isASCIIPrintable reports whether every byte of text is printable ASCII. Any multi-byte
// UTF-8 sequence contains bytes outside the printable ASCII range, so byte iteration is
// sufficient to detect non-ASCII content.
//
// Takes text (string) which is the string to inspect.
//
// Returns bool which is true when the string is safe to emit as a literal.
func isASCIIPrintable(text string) bool {
	for byteIndex := range len(text) {
		character := text[byteIndex]
		if character < asciiPrintableMin || character > asciiPrintableMax {
			return false
		}
	}
	return true
}

// encodeUTF16BEHexString encodes text as a UTF-16BE hexadecimal PDF string with a leading
// byte-order mark, surrogate-encoding supplementary-plane runes.
//
// Takes text (string) which is the raw string to encode.
//
// Returns string which is the "<FEFF...>" hexadecimal string token.
func encodeUTF16BEHexString(text string) string {
	var builder strings.Builder
	builder.WriteString(utf16ByteOrderMark)
	for _, character := range text {
		builder.WriteString(utf16BEUnitsHex(character))
	}
	builder.WriteByte('>')
	return builder.String()
}

// utf16BEUnitsHex returns the uppercase hexadecimal UTF-16BE code units for a rune,
// without delimiters.
//
// Takes character (rune) which is the codepoint to encode.
//
// Returns string which is the hexadecimal code unit(s).
func utf16BEUnitsHex(character rune) string {
	if character > bmpMaxCodepoint {
		offset := character - utf16SupplementaryBase
		high := utf16HighSurrogateBase + (offset >> utf16SurrogateShift)
		low := utf16LowSurrogateBase + (offset & utf16LowSurrogateMask)
		return fmt.Sprintf("%04X%04X", high, low)
	}
	return fmt.Sprintf("%04X", character)
}
