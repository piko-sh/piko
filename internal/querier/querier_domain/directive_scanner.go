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

package querier_domain

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// directiveLineScanner provides character-level scanning
// over a single directive comment line.
type directiveLineScanner struct {
	// line holds the full text of the line being scanned.
	line string

	// lineNumber holds the one-based line number within the source file.
	lineNumber int

	// position holds the current zero-based byte offset within the line.
	position int
}

// newDirectiveLineScanner creates a scanner positioned at the start of the given line.
//
// Takes line (string) which holds the text to scan.
//
// Takes lineNumber (int) which specifies the one-based line number for span tracking.
//
// Returns *directiveLineScanner which holds the
// initialised scanner.
func newDirectiveLineScanner(line string, lineNumber int) *directiveLineScanner {
	return &directiveLineScanner{
		line:       line,
		lineNumber: lineNumber,
	}
}

// atEnd reports whether the scanner has consumed all characters in the line.
//
// Returns bool which indicates whether the position is
// at or past the end of the line.
func (scanner *directiveLineScanner) atEnd() bool {
	return scanner.position >= len(scanner.line)
}

// current returns the byte at the current position, or zero if at end.
//
// Returns byte which holds the current character or 0
// if the scanner is at end.
func (scanner *directiveLineScanner) current() byte {
	if scanner.atEnd() {
		return 0
	}
	return scanner.line[scanner.position]
}

// advance consumes the current byte and advances the position by one.
//
// Returns byte which holds the byte that was consumed.
func (scanner *directiveLineScanner) advance() byte {
	current := scanner.line[scanner.position]
	scanner.position++
	return current
}

// column returns the one-based column number of the current position.
//
// Returns int which holds the current column number.
func (scanner *directiveLineScanner) column() int {
	return scanner.position + 1
}

// skipWhitespace advances the scanner past any spaces and tabs at the current position.
func (scanner *directiveLineScanner) skipWhitespace() {
	for !scanner.atEnd() && (scanner.current() == ' ' || scanner.current() == '\t') {
		scanner.position++
	}
}

// spanFrom constructs a TextSpan from the given start column to the current position.
//
// Takes startColumn (int) which specifies the one-based start column of the span.
//
// Returns querier_dto.TextSpan which holds the
// constructed span on the current line.
func (scanner *directiveLineScanner) spanFrom(startColumn int) querier_dto.TextSpan {
	return querier_dto.TextSpan{
		Line:      scanner.lineNumber,
		Column:    startColumn,
		EndLine:   scanner.lineNumber,
		EndColumn: scanner.column(),
	}
}

// matchString attempts to match an exact string at the
// current position and advances past it on success.
//
// Takes expected (string) which specifies the string to
// match.
//
// Returns bool which indicates whether the match
// succeeded.
func (scanner *directiveLineScanner) matchString(expected string) bool {
	if scanner.position+len(expected) > len(scanner.line) {
		return false
	}
	if scanner.line[scanner.position:scanner.position+len(expected)] == expected {
		scanner.position += len(expected)
		return true
	}
	return false
}

// matchKeyword attempts a case-insensitive keyword match
// at the current position with word boundary checking.
//
// Takes keyword (string) which specifies the keyword to
// match.
//
// Returns bool which indicates whether the match
// succeeded.
func (scanner *directiveLineScanner) matchKeyword(keyword string) bool {
	if scanner.position+len(keyword) > len(scanner.line) {
		return false
	}
	candidate := scanner.line[scanner.position : scanner.position+len(keyword)]
	if !strings.EqualFold(candidate, keyword) {
		return false
	}
	afterEnd := scanner.position + len(keyword)
	if afterEnd < len(scanner.line) && isWordCharacter(scanner.line[afterEnd]) {
		return false
	}
	scanner.position = afterEnd
	return true
}

// matchByte attempts to match a single byte at the
// current position and advances past it on success.
//
// Takes expected (byte) which specifies the byte to
// match.
//
// Returns bool which indicates whether the match
// succeeded.
func (scanner *directiveLineScanner) matchByte(expected byte) bool {
	if scanner.atEnd() || scanner.current() != expected {
		return false
	}
	scanner.position++
	return true
}

// lookingAt checks whether the given prefix appears at
// the current position without advancing.
//
// Takes prefix (string) which specifies the string to
// check for.
//
// Returns bool which indicates whether the prefix is
// present.
func (scanner *directiveLineScanner) lookingAt(prefix string) bool {
	if scanner.position+len(prefix) > len(scanner.line) {
		return false
	}
	return scanner.line[scanner.position:scanner.position+len(prefix)] == prefix
}

// readWord reads a contiguous sequence of word characters from the current position.
//
// Returns string which holds the consumed word text.
//
// Returns querier_dto.TextSpan which holds the span
// covering the consumed word.
func (scanner *directiveLineScanner) readWord() (string, querier_dto.TextSpan) {
	startColumn := scanner.column()
	start := scanner.position
	for !scanner.atEnd() && isWordCharacter(scanner.current()) {
		scanner.position++
	}
	return scanner.line[start:scanner.position], scanner.spanFrom(startColumn)
}

// readDigits reads a contiguous sequence of ASCII digit
// characters from the current position.
//
// Returns string which holds the consumed digit text.
//
// Returns querier_dto.TextSpan which holds the span
// covering the consumed digits.
func (scanner *directiveLineScanner) readDigits() (string, querier_dto.TextSpan) {
	startColumn := scanner.column()
	start := scanner.position
	for !scanner.atEnd() && scanner.current() >= '0' && scanner.current() <= '9' {
		scanner.position++
	}
	return scanner.line[start:scanner.position], scanner.spanFrom(startColumn)
}

// readUntilByte reads characters until the specified
// delimiter byte is encountered or end of line is
// reached.
//
// Takes delimiter (byte) which specifies the stop
// character.
//
// Returns string which holds the consumed text up to but
// not including the delimiter.
//
// Returns querier_dto.TextSpan which holds the span
// covering the consumed text.
func (scanner *directiveLineScanner) readUntilByte(delimiter byte) (string, querier_dto.TextSpan) {
	startColumn := scanner.column()
	start := scanner.position
	for !scanner.atEnd() && scanner.current() != delimiter {
		scanner.position++
	}
	return scanner.line[start:scanner.position], scanner.spanFrom(startColumn)
}

// readUntilWhitespace reads characters until a space or
// tab is encountered or end of line is reached.
//
// Returns string which holds the consumed text.
//
// Returns querier_dto.TextSpan which holds the span
// covering the consumed text.
func (scanner *directiveLineScanner) readUntilWhitespace() (string, querier_dto.TextSpan) {
	startColumn := scanner.column()
	start := scanner.position
	for !scanner.atEnd() && scanner.current() != ' ' && scanner.current() != '\t' {
		scanner.position++
	}
	return scanner.line[start:scanner.position], scanner.spanFrom(startColumn)
}

// readRemainder reads all remaining characters from the
// current position to the end of the line.
//
// Returns string which holds the remaining text.
//
// Returns querier_dto.TextSpan which holds the span
// covering the remainder.
func (scanner *directiveLineScanner) readRemainder() (string, querier_dto.TextSpan) {
	startColumn := scanner.column()
	start := scanner.position
	scanner.position = len(scanner.line)
	return scanner.line[start:], scanner.spanFrom(startColumn)
}

// isWordCharacter reports whether the given byte is a letter, digit, or underscore.
//
// Takes character (byte) which specifies the byte to test.
//
// Returns bool which indicates whether the byte is a
// word character.
func isWordCharacter(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		(character >= '0' && character <= '9') ||
		character == '_'
}
