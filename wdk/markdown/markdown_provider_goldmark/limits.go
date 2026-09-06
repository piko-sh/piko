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

package markdown_provider_goldmark

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/yuin/goldmark"
)

const (
	// defaultMaxInputSize is the largest markdown document, in bytes, that Parse accepts
	// unless WithMaxInputSize sets another limit.
	defaultMaxInputSize = 16 << 20

	// defaultMaxNestingDepth is the most blockquote and list markers that may open on a
	// single line unless WithMaxNestingDepth sets another limit. Goldmark's parsing time
	// grows faster than linearly with nesting, so deeper input is rejected up front.
	defaultMaxNestingDepth = 1000

	// maxOrderedMarkerDigits is the longest run of digits CommonMark accepts in an ordered
	// list marker.
	maxOrderedMarkerDigits = 9
)

var (
	// ErrInputTooLarge is returned by Parse when the markdown document is larger than the
	// configured size limit.
	ErrInputTooLarge = errors.New("markdown input exceeds the size limit")

	// ErrNestingTooDeep is returned by Parse when a line opens more nested blockquotes and
	// list items than the configured nesting limit.
	ErrNestingTooDeep = errors.New("markdown nesting exceeds the depth limit")
)

// Option configures a Parser created by NewParser.
type Option func(*parserConfig)

// parserConfig holds the settings gathered from the options passed to NewParser.
type parserConfig struct {
	// extensions holds the goldmark extensions added after the built-in ones.
	extensions []goldmark.Extender

	// maxInputSize is the largest document, in bytes, that Parse accepts.
	maxInputSize int

	// maxNestingDepth is the most blockquote and list markers that may open on one line.
	maxNestingDepth int
}

// WithExtensions adds goldmark extensions, such as syntax highlighting, after the
// built-in ones.
//
// Takes extensions (...goldmark.Extender) which are the extensions to add.
//
// Returns Option which applies the extensions.
func WithExtensions(extensions ...goldmark.Extender) Option {
	return func(config *parserConfig) {
		config.extensions = append(config.extensions, extensions...)
	}
}

// WithMaxInputSize sets the largest markdown document, in bytes, that Parse accepts;
// larger documents fail with ErrInputTooLarge. The default is 16 MiB.
//
// Takes size (int) which is the limit in bytes; values below one keep the default.
//
// Returns Option which applies the limit.
func WithMaxInputSize(size int) Option {
	return func(config *parserConfig) {
		if size > 0 {
			config.maxInputSize = size
		}
	}
}

// WithMaxNestingDepth sets the most blockquote and list markers that may open on a single
// line; deeper lines fail with ErrNestingTooDeep before goldmark parses them. The default
// is 1000.
//
// Takes depth (int) which is the limit; values below one keep the default.
//
// Returns Option which applies the limit.
func WithMaxNestingDepth(depth int) Option {
	return func(config *parserConfig) {
		if depth > 0 {
			config.maxNestingDepth = depth
		}
	}
}

// newParserConfig applies options over the default settings.
//
// Takes options ([]Option) which adjust the defaults.
//
// Returns parserConfig which holds the resulting settings.
func newParserConfig(options []Option) parserConfig {
	config := parserConfig{
		extensions:      nil,
		maxInputSize:    defaultMaxInputSize,
		maxNestingDepth: defaultMaxNestingDepth,
	}
	for _, option := range options {
		option(&config)
	}
	return config
}

// checkInputLimits rejects documents that are too large or nested too deeply to parse
// safely. The nesting check is a single linear pass over the input.
//
// Takes content ([]byte) which is the markdown document.
// Takes maxInputSize (int) which is the largest accepted size in bytes.
// Takes maxNestingDepth (int) which is the most markers that may open on one line.
//
// Returns error which wraps ErrInputTooLarge or ErrNestingTooDeep, or nil when the
// document is within both limits.
func checkInputLimits(content []byte, maxInputSize, maxNestingDepth int) error {
	if len(content) > maxInputSize {
		return fmt.Errorf("%w: %d bytes is more than the %d byte limit", ErrInputTooLarge, len(content), maxInputSize)
	}
	if lineNumber := findOverNestedLine(content, maxNestingDepth); lineNumber > 0 {
		return fmt.Errorf("%w: line %d opens more than %d nested blockquotes or list items", ErrNestingTooDeep, lineNumber, maxNestingDepth)
	}
	return nil
}

// findOverNestedLine finds the first line that opens more than limit blockquote or list
// markers.
//
// Takes content ([]byte) which is the markdown document.
// Takes limit (int) which is the most markers allowed on one line.
//
// Returns int which is the one-based number of the first offending line, or zero when
// every line is within the limit.
func findOverNestedLine(content []byte, limit int) int {
	lineNumber := 1
	for remaining := content; len(remaining) > 0; lineNumber++ {
		line, rest, _ := bytes.Cut(remaining, []byte{'\n'})
		if countOpeningMarkers(line, limit) > limit {
			return lineNumber
		}
		remaining = rest
	}
	return 0
}

// countOpeningMarkers counts the blockquote and list markers that open at the start of a
// line, stopping as soon as the count passes limit.
//
// Takes line ([]byte) which is one line of the document without its line ending.
// Takes limit (int) which is the count after which scanning stops.
//
// Returns int which is the number of markers counted, at most limit plus one.
func countOpeningMarkers(line []byte, limit int) int {
	count := 0
	position := 0
	for count <= limit {
		position = skipMarkdownSpaces(line, position)
		width := containerMarkerWidth(line[position:])
		if width == 0 {
			return count
		}
		count++
		position += width
	}
	return count
}

// skipMarkdownSpaces advances past spaces and tabs.
//
// Takes line ([]byte) which is the line being scanned.
// Takes position (int) which is where to start.
//
// Returns int which is the position of the first byte that is not a space or tab.
func skipMarkdownSpaces(line []byte, position int) int {
	for position < len(line) && isMarkdownSpace(line[position]) {
		position++
	}
	return position
}

// containerMarkerWidth measures a blockquote or list marker at the start of rest.
//
// Takes rest ([]byte) which is the remainder of the line.
//
// Returns int which is the byte length of the marker, or zero when rest does not start
// with a '>', a bullet followed by a space or tab, or an ordered marker of up to nine
// digits and '.' or ')' followed by a space or tab.
func containerMarkerWidth(rest []byte) int {
	if len(rest) == 0 {
		return 0
	}
	switch rest[0] {
	case '>':
		return 1
	case '-', '*', '+':
		if len(rest) > 1 && isMarkdownSpace(rest[1]) {
			return 1
		}
		return 0
	}
	digits := 0
	for digits < len(rest) && digits < maxOrderedMarkerDigits && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits+1 >= len(rest) {
		return 0
	}
	if (rest[digits] == '.' || rest[digits] == ')') && isMarkdownSpace(rest[digits+1]) {
		return digits + 1
	}
	return 0
}

// isMarkdownSpace reports whether b is a space or tab.
//
// Takes b (byte) which is the byte to test.
//
// Returns bool which is true for a space or tab.
func isMarkdownSpace(b byte) bool {
	return b == ' ' || b == '\t'
}
