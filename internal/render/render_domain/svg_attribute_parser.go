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

package render_domain

import (
	"sync"

	"piko.sh/piko/internal/ast/ast_domain"
)

// svgAttribute holds an SVG attribute name and value pair.
// It is a smaller version of ast_domain.HTMLAttribute, saving 48 bytes per
// attribute by not storing location or range data.
type svgAttribute struct {
	// Name is the attribute name.
	Name string

	// Value is the attribute's value.
	Value string
}

// svgAttrSliceInitialCap is the starting capacity for pooled attribute slices.
// Most SVG elements have 3 to 8 attributes, so 8 is a sensible default.
const svgAttrSliceInitialCap = 8

// svgAttrSlicePool reuses svgAttribute slices to reduce allocation pressure
// during SVG attribute parsing.
var svgAttrSlicePool = sync.Pool{
	New: func() any {
		return new(make([]svgAttribute, 0, svgAttrSliceInitialCap))
	},
}

// ParseSVGAttributes parses SVG tag attributes and converts them to HTML
// attribute format.
//
// This is the main function for parsing SVG tag attributes. It combines
// parsing and conversion in a single call and handles pool management
// internally.
//
// This replaces html.NewTokenizer-based parsing with better performance:
//   - 6x faster than html.NewTokenizer
//   - 96% fewer allocations (26 to 1 per call)
//   - Keeps attribute case correct (needed for SVG, unlike HTML tokeniser)
//
// Takes tagContent (string) which contains the raw SVG tag content to parse.
//
// Returns []ast_domain.HTMLAttribute which contains the parsed attributes,
// or nil for empty or whitespace-only input.
func ParseSVGAttributes(tagContent string) []ast_domain.HTMLAttribute {
	if len(tagContent) == 0 {
		return nil
	}

	hasContent := false
	for i := range len(tagContent) {
		if !isWhitespaceASCII(tagContent[i]) {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return nil
	}

	attrs := parseSVGTagAttributes(tagContent)
	defer putSVGAttrSlice(attrs)

	return convertSVGToHTMLAttributes(attrs)
}

// getSVGAttrSlice retrieves a pooled attribute slice.
// The caller must return the slice via putSVGAttrSlice when done.
//
// Returns *[]svgAttribute which is a slice ready for use.
func getSVGAttrSlice() *[]svgAttribute {
	s, ok := svgAttrSlicePool.Get().(*[]svgAttribute)
	if !ok {
		return new(make([]svgAttribute, 0, svgAttrSliceInitialCap))
	}
	return s
}

// putSVGAttrSlice returns a slice to the pool for reuse.
//
// Takes s (*[]svgAttribute) which is the slice to return.
func putSVGAttrSlice(s *[]svgAttribute) {
	if s == nil {
		return
	}
	*s = (*s)[:0]
	svgAttrSlicePool.Put(s)
}

// parseSVGTagAttributes extracts attributes from an SVG opening tag.
//
// This parser does not allocate memory. It works directly on the source string.
// The returned slice comes from a pool. You must return it with putSVGAttrSlice
// when finished.
//
// The parser is built for SVG content where:
//   - Attribute names are case-sensitive (viewBox, strokeWidth, etc.)
//   - Content is ASCII only (valid SVG/XML)
//   - Speed matters (called for each SVG on cache miss)
//
// Takes tagContent (string) which is the SVG opening tag content without the
// tag name.
//
// Returns *[]svgAttribute which holds the parsed attributes from the tag.
func parseSVGTagAttributes(tagContent string) *[]svgAttribute {
	attrs := getSVGAttrSlice()
	position := 0

	for position < len(tagContent) {
		name, value, newPos, found := parseSingleAttribute(tagContent, position)
		if !found {
			break
		}
		position = newPos
		if name != "" {
			*attrs = append(*attrs, svgAttribute{Name: name, Value: value})
		}
	}

	return attrs
}

// parseSingleAttribute parses one attribute from the tag content.
//
// Takes s (string) which contains the raw tag content to parse.
// Takes position (int) which specifies the starting position in the string.
//
// Returns name (string) which is the attribute name, or empty if none is found.
// Returns value (string) which is the attribute value, or empty for boolean
// attributes.
// Returns newPos (int) which is the position after this attribute is parsed.
// Returns found (bool) which is true when parsing should continue.
func parseSingleAttribute(s string, position int) (name, value string, newPos int, found bool) {
	position = skipWhitespaceASCII(s, position)
	if position >= len(s) {
		return "", "", position, false
	}

	name, position = parseAttrName(s, position)
	if name == "" {
		return "", "", position + 1, true
	}

	position = skipWhitespaceASCII(s, position)

	if position >= len(s) || s[position] != '=' {
		return name, "", position, true
	}
	position++

	position = skipWhitespaceASCII(s, position)

	if position >= len(s) {
		return name, "", position, false
	}

	value, position = parseAttrValue(s, position)
	return name, value, position, true
}

// isWhitespaceASCII reports whether a byte is ASCII whitespace.
// Faster than unicode.IsSpace for ASCII-only content like SVG tags.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is a space, tab, newline, or carriage return.
func isWhitespaceASCII(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// skipWhitespaceASCII moves past whitespace characters in a string.
//
// Takes s (string) which is the input string to scan.
// Takes position (int) which is the starting position in the string.
//
// Returns int which is the position of the first non-whitespace character,
// or the string length if only whitespace remains.
func skipWhitespaceASCII(s string, position int) int {
	for position < len(s) && isWhitespaceASCII(s[position]) {
		position++
	}
	return position
}

// parseAttrName extracts an attribute name starting at the given position.
// Returns an empty string and the original position if no valid name is found.
//
// Takes s (string) which is the input string to parse.
// Takes position (int) which is the starting position in the string.
//
// Returns string which is the attribute name, or empty if none is found.
// Returns int which is the position after the name, or the original position.
func parseAttrName(s string, position int) (string, int) {
	nameStart := position
	for position < len(s) {
		c := s[position]
		if c == '=' || isWhitespaceASCII(c) || c == '>' || c == '/' {
			break
		}
		position++
	}
	if position == nameStart {
		return "", nameStart
	}
	return s[nameStart:position], position
}

// parseAttrValue extracts an attribute value starting at the given position.
// It handles both quoted ("..." or '...') and unquoted values.
//
// Takes s (string) which is the input string to parse.
// Takes position (int) which is the starting position in the string.
//
// Returns string which is the extracted attribute value.
// Returns int which is the position after the value.
func parseAttrValue(s string, position int) (string, int) {
	if position >= len(s) {
		return "", position
	}

	quote := s[position]
	if quote == '"' || quote == '\'' {
		return parseQuotedValue(s, position, quote)
	}
	return parseUnquotedValue(s, position)
}

// parseQuotedValue extracts the text between matching quote marks.
//
// Takes s (string) which holds the text to parse.
// Takes position (int) which is the position of the opening quote.
// Takes quote (byte) which is the quote character to match.
//
// Returns string which is the value without the surrounding quotes.
// Returns int which is the position after the closing quote.
func parseQuotedValue(s string, position int, quote byte) (string, int) {
	position++
	valueStart := position
	for position < len(s) && s[position] != quote {
		position++
	}
	value := s[valueStart:position]
	if position < len(s) {
		position++
	}
	return value, position
}

// parseUnquotedValue extracts an unquoted attribute value from a string.
//
// Takes s (string) which contains the attribute string to parse.
// Takes position (int) which is the starting position in the string.
//
// Returns string which is the extracted value.
// Returns int which is the position after the value ends.
func parseUnquotedValue(s string, position int) (string, int) {
	valueStart := position
	for position < len(s) && !isWhitespaceASCII(s[position]) && s[position] != '>' {
		position++
	}
	return s[valueStart:position], position
}

// convertSVGToHTMLAttributes converts svgAttribute values to HTMLAttribute
// values. It only allocates the final slice; the svgAttribute parsing itself
// uses no extra memory.
//
// Takes attrs (*[]svgAttribute) which contains the SVG attributes to convert.
//
// Returns []ast_domain.HTMLAttribute which contains the converted attributes,
// or nil if attrs is nil or empty.
func convertSVGToHTMLAttributes(attrs *[]svgAttribute) []ast_domain.HTMLAttribute {
	if attrs == nil || len(*attrs) == 0 {
		return nil
	}

	result := make([]ast_domain.HTMLAttribute, len(*attrs))
	for i, attr := range *attrs {
		result[i] = ast_domain.HTMLAttribute{
			Name:           attr.Name,
			Value:          attr.Value,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		}
	}
	return result
}
