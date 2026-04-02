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

package formatter_domain

import "context"

const (
	// defaultIndentSize is the default number of spaces used for each indent
	// level.
	defaultIndentSize = 2

	// defaultMaxLineLength is the default maximum line length before attribute
	// wrapping.
	defaultMaxLineLength = 100

	// defaultAttributeWrapIndent is the default additional indent for wrapped
	// attributes.
	defaultAttributeWrapIndent = 1
)

// FileFormat specifies the type of file being formatted.
type FileFormat int

const (
	// FormatAuto automatically detects the file format based on content.
	FormatAuto FileFormat = iota

	// FormatPK treats the input as a Piko Single File Component (.pk)
	// containing <template>, <script>, and <style> blocks.
	FormatPK

	// FormatHTML treats the input as plain HTML without SFC block structure.
	FormatHTML
)

// FormatterService is the main interface for formatting Piko template files. It
// provides methods to format source code according to consistent, opinionated
// rules.
type FormatterService interface {
	// Format formats raw source bytes of a .pk file and returns formatted bytes.
	// It formats both the template and script blocks according to Piko rules.
	//
	// Takes source ([]byte) which contains the raw .pk file content.
	//
	// Returns []byte which contains the formatted output.
	// Returns error when parsing or formatting fails.
	Format(ctx context.Context, source []byte) ([]byte, error)

	// FormatWithOptions formats source code using the given formatting options.
	//
	// Takes source ([]byte) which is the code to format.
	// Takes opts (*FormatOptions) which controls how the code is formatted.
	//
	// Returns []byte which is the formatted source code.
	// Returns error when formatting fails.
	FormatWithOptions(ctx context.Context, source []byte, opts *FormatOptions) ([]byte, error)

	// FormatRange formats only a specific range within the source. The range is
	// specified in line and character coordinates, which are zero-based.
	//
	// Takes source ([]byte) which contains the original file content.
	// Takes formatRange (Range) which specifies the region to format.
	// Takes opts (*FormatOptions) which configures the formatting behaviour.
	//
	// Returns []byte which contains the entire file with only the range modified.
	// Returns error when formatting fails.
	FormatRange(ctx context.Context, source []byte, formatRange Range, opts *FormatOptions) ([]byte, error)
}

// Range represents a text range with start and end positions.
// Positions use zero-based line and character offsets.
type Range struct {
	// StartLine is the zero-based line number where the range begins.
	StartLine uint32

	// StartCharacter is the zero-based character position within the start line.
	StartCharacter uint32

	// EndLine is the zero-based line number where the range ends.
	EndLine uint32

	// EndCharacter is the zero-based character position within the end line.
	EndCharacter uint32
}

// FormatOptions configures the formatting behaviour.
// For now, we start with zero configuration (opinionated formatting),
// but this struct allows for future extensibility.
type FormatOptions struct {
	// FileFormat specifies the type of file being formatted.
	// Default: FormatAuto (auto-detect based on content).
	FileFormat FileFormat

	// IndentSize is the number of spaces per indentation level.
	// Default is 2, following common HTML and template conventions.
	IndentSize int

	// MaxLineLength is the maximum line length before opening tag attributes wrap.
	// Set to 0 to disable; default is 100.
	MaxLineLength int

	// AttributeWrapIndent is the number of extra indent levels to add to wrapped
	// attributes, relative to the element's indentation. Default is 1.
	AttributeWrapIndent int

	// PreserveEmptyLines controls whether multiple blank lines in a row are kept.
	// Default: false (reduce to a single blank line).
	PreserveEmptyLines bool

	// SortAttributes determines whether to sort HTML attributes alphabetically
	// for deterministic output and idempotency; default is true.
	SortAttributes bool

	// RawHTMLMode treats p-* attributes as regular HTML attributes
	// rather than Piko directives, allowing formatting of rendered
	// HTML output that contains runtime p-key values (e.g., "r.0:0")
	// which are not valid expressions. Default: false.
	RawHTMLMode bool
}

// DefaultFormatOptions returns the default formatting options.
//
// Returns *FormatOptions which contains sensible defaults for text formatting.
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		FileFormat:          FormatAuto,
		IndentSize:          defaultIndentSize,
		PreserveEmptyLines:  false,
		SortAttributes:      true,
		MaxLineLength:       defaultMaxLineLength,
		AttributeWrapIndent: defaultAttributeWrapIndent,
	}
}
