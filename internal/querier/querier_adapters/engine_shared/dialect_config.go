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
	"fmt"
)

// DialectConfig declares the lexical rules that vary between SQL dialects.
//
// Each engine constructs one value describing its dialect and passes the embedded rule
// sets to the shared scanners, so the scanning code itself never branches on a dialect
// name: every difference is a data value here.
type DialectConfig struct {
	// Comments describes how the dialect recognises line and block comments.
	Comments CommentRules

	// Numbers describes which base-prefixed integer literals the dialect accepts.
	Numbers NumberRules
}

// CommentRules captures the dialect-specific variations in SQL comment syntax. The shared
// comment scanner reproduces every engine's behaviour by reading these flags rather than
// by holding per-dialect code paths.
type CommentRules struct {
	// DoubleDashRequiresWhitespace gates whether a "--" sequence begins a line comment.
	//
	// When set, a "--" begins a line comment only when the next byte is whitespace or a
	// control character (any byte at or below a space, including a newline, a carriage
	// return, or end of input). MySQL requires this so that "a--b" is not misread as a
	// comment; the other dialects treat "--" as a comment unconditionally.
	DoubleDashRequiresWhitespace bool

	// HashLineComment makes a "#" begin a line comment that runs to end of line. MySQL
	// accepts this form; the other dialects do not.
	HashLineComment bool

	// NestedBlockComments makes "/* ... */" comments nest, so an inner "/*" must be balanced
	// by a matching "*/" before the outer comment closes.
	//
	// Postgres, DuckDB, and ClickHouse nest; MySQL and SQLite close on the first "*/".
	NestedBlockComments bool
}

// SkipWhitespaceAndComments advances past a run of whitespace and SQL comments starting
// at position, honouring the dialect's comment rules. It stops at the first byte that is
// neither whitespace nor part of a comment.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which is the byte offset to begin scanning from.
//
// Returns int which is the byte offset of the first non-whitespace, non-comment byte (or
// the length of input when the remainder is whitespace and comments).
// Returns error when a block comment is opened but never closed before end of input.
func (rules CommentRules) SkipWhitespaceAndComments(input string, position int) (int, error) {
	for position < len(input) {
		character := input[position]

		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			position++
			continue
		}

		if rules.startsDoubleDashComment(input, position) {
			position = consumeToLineEnd(input, position)
			continue
		}

		if rules.HashLineComment && character == '#' {
			position = consumeToLineEnd(input, position)
			continue
		}

		if character == '/' && position+1 < len(input) && input[position+1] == '*' {
			next, err := skipBlockComment(input, position, rules.NestedBlockComments)
			if err != nil {
				return next, err
			}
			position = next
			continue
		}

		break
	}

	return position, nil
}

// startsDoubleDashComment reports whether a "--" line comment begins at position under
// the dialect's rules.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which is the candidate comment start.
//
// Returns bool which is true when a line comment begins at position.
func (rules CommentRules) startsDoubleDashComment(input string, position int) bool {
	if position+1 >= len(input) || input[position] != '-' || input[position+1] != '-' {
		return false
	}
	if !rules.DoubleDashRequiresWhitespace {
		return true
	}

	if position+2 >= len(input) {
		return true
	}
	return input[position+2] <= ' '
}

// consumeToLineEnd advances position past the remainder of the current line.
//
// It stops at the next newline or carriage return so the caller's whitespace handling
// consumes it. Stopping on a carriage return as well as a newline lets the scanner
// terminate a line comment under classic Mac (lone "\r") line endings as well as Unix and
// Windows endings.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which is the byte offset to begin from.
//
// Returns int which is the offset of the next newline or carriage return, or the length
// of input.
func consumeToLineEnd(input string, position int) int {
	for position < len(input) && input[position] != '\n' && input[position] != '\r' {
		position++
	}
	return position
}

// skipBlockComment advances past a "/* ... */" block comment beginning at position,
// nesting inner comments when nested is true.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which indexes the opening "/*".
// Takes nested (bool) which enables balanced nesting of inner block comments.
//
// Returns int which is the offset after the closing "*/", or where scanning stopped on an
// unterminated comment.
// Returns error when the comment is not closed before end of input.
func skipBlockComment(input string, position int, nested bool) (int, error) {
	startPosition := position
	position += 2
	depth := 1
	for position < len(input) && depth > 0 {
		if nested && position+1 < len(input) && input[position] == '/' && input[position+1] == '*' {
			depth++
			position += 2
			continue
		}
		if position+1 < len(input) && input[position] == '*' && input[position+1] == '/' {
			depth--
			position += 2
			continue
		}
		position++
	}

	if depth > 0 {
		return position, fmt.Errorf("unterminated block comment at position %d", startPosition)
	}

	return position, nil
}

// NumberRules captures which base-prefixed integer literals a dialect accepts. The shared
// number scanner reads these flags so one scanner serves dialects that differ only in
// their recognised bases.
type NumberRules struct {
	// HexPrefix accepts "0x" / "0X" hexadecimal integer literals.
	HexPrefix bool

	// OctalPrefix accepts "0o" / "0O" octal integer literals.
	OctalPrefix bool

	// BinaryPrefix accepts "0b" / "0B" binary integer literals.
	BinaryPrefix bool

	// RequireDigitsAfterPrefix rejects a base prefix that is followed by no digits of its
	// base (for example "0x" with nothing after it) as a lexical error, rather than
	// accepting the bare prefix as a literal. ClickHouse is strict; the other dialects
	// accept the bare prefix.
	RequireDigitsAfterPrefix bool
}

// PrefixValidator returns the digit predicate for a base-prefix byte (the byte following
// the leading "0"), or nil when the dialect does not accept that prefix.
//
// Takes prefix (byte) which is the byte that follows the leading "0".
//
// Returns func(byte) bool which validates digits for the matched base, or nil when the
// prefix is not an enabled base.
func (rules NumberRules) PrefixValidator(prefix byte) func(byte) bool {
	switch prefix {
	case 'x', 'X':
		if rules.HexPrefix {
			return IsHexDigit
		}
	case 'o', 'O':
		if rules.OctalPrefix {
			return IsOctalDigit
		}
	case 'b', 'B':
		if rules.BinaryPrefix {
			return IsBinaryDigit
		}
	}
	return nil
}

// TryReadPrefixedNumber consumes a base-prefixed integer literal ("0x", "0o", or "0b")
// that begins at position when the dialect enables that prefix.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which indexes the candidate leading "0".
//
// Returns int which is the offset after the literal, or position unchanged when no
// enabled prefix matched.
// Returns bool which is true when a prefixed literal was consumed.
// Returns error when RequireDigitsAfterPrefix is set and the prefix lacks digits.
func (rules NumberRules) TryReadPrefixedNumber(input string, position int) (int, bool, error) {
	if position+1 >= len(input) || input[position] != '0' {
		return position, false, nil
	}

	prefix := input[position+1]
	validator := rules.PrefixValidator(prefix)
	if validator == nil {
		return position, false, nil
	}

	startPosition := position
	position += 2
	digitsStart := position
	for position < len(input) && validator(input[position]) {
		position++
	}

	if rules.RequireDigitsAfterPrefix && position == digitsStart {
		return startPosition, false, fmt.Errorf("base-prefixed number at position %d has no digits after the \"0%c\" prefix", startPosition, prefix)
	}

	return position, true, nil
}
