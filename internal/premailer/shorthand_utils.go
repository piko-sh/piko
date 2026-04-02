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

package premailer

import (
	"slices"
	"strings"
)

// cssValueSplitState holds the parsing state for splitting CSS values.
type cssValueSplitState struct {
	// parts holds the split string segments from the CSS value.
	parts []string

	// current builds the CSS value part character by character.
	current strings.Builder

	// inQuotes indicates whether the parser is inside a quoted string.
	inQuotes bool

	// inParens tracks the depth of nested parentheses; 0 means outside.
	inParens int
}

// expandListStyleShorthand splits a CSS list-style shorthand value into its
// separate properties.
//
// The shorthand may include list-style-type, list-style-position, and
// list-style-image in any order. All three are optional.
//
// Takes value (string) which is the shorthand value to expand.
//
// Returns map[string]string which holds the individual properties, or nil if
// no valid properties are found.
func expandListStyleShorthand(value string) map[string]string {
	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	var listType, position, image string
	for _, part := range parts {
		if isListStyleType(part) {
			listType = part
		} else if isListStylePosition(part) {
			position = part
		} else if strings.HasPrefix(part, "url(") {
			image = part
		}
	}

	result := make(map[string]string)
	if listType != "" {
		result["list-style-type"] = listType
	}
	if position != "" {
		result["list-style-position"] = position
	}
	if image != "" {
		result["list-style-image"] = image
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// expandOverflowShorthand expands an overflow shorthand into separate
// properties. One value sets both axes; two values set x and y in that order.
//
// Takes value (string) which is the overflow shorthand, such as "hidden" or
// "scroll auto".
//
// Returns map[string]string which holds overflow-x and overflow-y properties,
// or nil when the input is empty or has more than two values.
func expandOverflowShorthand(value string) map[string]string {
	values := splitSpaceDelimited(value)
	if len(values) == 0 {
		return nil
	}

	if len(values) == 1 {
		return map[string]string{
			"overflow-x": values[0],
			"overflow-y": values[0],
		}
	}

	if len(values) == 2 {
		return map[string]string{
			"overflow-x": values[0],
			"overflow-y": values[1],
		}
	}

	return nil
}

// expandInsetShorthand expands an inset shorthand value into individual
// top, right, bottom, and left positioning properties.
//
// This is the logical property equivalent of setting all four positioning
// properties at once. Follows the same 1-4 value pattern as margin and
// padding.
//
// Takes value (string) which contains space-delimited inset values.
//
// Returns map[string]string which maps direction names to their values,
// or nil when the input is empty or has more than four values.
func expandInsetShorthand(value string) map[string]string {
	fv, ok := expandFourValues(value)
	if !ok {
		return nil
	}
	top, right, bottom, left := fv.a, fv.b, fv.c, fv.d

	return map[string]string{
		directionTop:    top,
		directionRight:  right,
		directionBottom: bottom,
		directionLeft:   left,
	}
}

// expandTextDecorationShorthand expands a text-decoration shorthand value into
// its longhand properties.
//
// The syntax is: <line> || <style> || <colour> || <thickness>. All values are
// optional and can appear in any order. For example: "underline solid blue 2px".
//
// Takes value (string) which is the shorthand text-decoration value to expand.
//
// Returns map[string]string which contains the expanded longhand properties,
// or nil when the value is empty or contains no recognised properties.
func expandTextDecorationShorthand(value string) map[string]string {
	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	var line, style, color, thickness string

	for _, part := range parts {
		if isTextDecorationLine(part) {
			line = part
		} else if isTextDecorationStyle(part) {
			style = part
		} else if isTextDecorationThickness(part) {
			thickness = part
		} else {
			color = part
		}
	}

	result := make(map[string]string)
	if line != "" {
		result["text-decoration-line"] = line
	}
	if style != "" {
		result["text-decoration-style"] = style
	}
	if color != "" {
		result["text-decoration-color"] = color
	}
	if thickness != "" {
		result["text-decoration-thickness"] = thickness
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// splitCommaOutsideParens splits a string on commas that are not
// inside parentheses or quotes.
//
// Takes s (string) which is the CSS value string to split.
//
// Returns []string which contains the comma-separated parts.
func splitCommaOutsideParens(s string) []string {
	var parts []string
	depth := 0
	inQuotes := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			inQuotes = !inQuotes
		case '(':
			if !inQuotes {
				depth++
			}
		case ')':
			if !inQuotes && depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 && !inQuotes {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// splitSpaceDelimited splits a CSS value by spaces while keeping quoted
// strings and function calls together.
//
// Takes value (string) which is the CSS property value to split.
//
// Returns []string which contains the split parts with quotes and brackets
// kept whole.
func splitSpaceDelimited(value string) []string {
	state := &cssValueSplitState{}
	value = strings.TrimSpace(value)

	for i := range len(value) {
		char := value[i]
		switch char {
		case '"', '\'':
			state.inQuotes = !state.inQuotes
			_ = state.current.WriteByte(char)
		case '(':
			state.inParens++
			_ = state.current.WriteByte(char)
		case ')':
			state.inParens--
			_ = state.current.WriteByte(char)
		case ' ', '\t', '\n':
			handleWhitespaceInSplit(char, state)
		default:
			_ = state.current.WriteByte(char)
		}
	}

	if state.current.Len() > 0 {
		state.parts = append(state.parts, state.current.String())
	}

	return state.parts
}

// handleWhitespaceInSplit processes a whitespace character during CSS value
// splitting. It keeps whitespace inside quotes or brackets, or uses it to
// separate values.
//
// Takes char (byte) which is the whitespace character to process.
// Takes state (*cssValueSplitState) which holds the current parsing state.
func handleWhitespaceInSplit(char byte, state *cssValueSplitState) {
	if state.inQuotes || state.inParens > 0 {
		_ = state.current.WriteByte(char)
		return
	}

	if state.current.Len() > 0 {
		state.parts = append(state.parts, state.current.String())
		state.current.Reset()
	}
}

// isListStyleType checks if a value is a valid CSS list-style-type keyword.
//
// Takes value (string) which is the CSS property value to check.
//
// Returns bool which is true if the value matches a known list-style-type
// keyword, or false otherwise.
func isListStyleType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	types := []string{
		"none",
		"disc", "circle", "square",
		"decimal", "decimal-leading-zero",
		"lower-alpha", "upper-alpha",
		"lower-latin", "upper-latin",
		"lower-greek",
		"lower-roman", "upper-roman",
		"hiragana", "katakana", "hiragana-iroha", "katakana-iroha",
		"cjk-decimal", "cjk-ideographic",
		"hebrew", "armenian", "georgian",
		"disclosure-open", "disclosure-closed",
	}

	return slices.Contains(types, value)
}

// isListStylePosition checks whether a value is a valid CSS list-style-position
// keyword.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is "inside" or "outside".
func isListStylePosition(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "inside" || value == "outside"
}

// isTextDecorationLine checks whether a value is a valid text-decoration-line
// keyword.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is a valid keyword such as none,
// underline, overline, line-through, or blink.
func isTextDecorationLine(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	lines := []string{"none", "underline", "overline", "line-through", "blink"}
	return slices.Contains(lines, value)
}

// isTextDecorationStyle checks whether a value is a valid text-decoration-style
// CSS keyword.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value matches a valid style keyword (solid,
// double, dotted, dashed, or wavy).
func isTextDecorationStyle(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	styles := []string{"solid", "double", "dotted", "dashed", "wavy"}
	return slices.Contains(styles, value)
}

// isTextDecorationThickness checks whether a value is a valid CSS
// text-decoration-thickness property value.
//
// Takes value (string) which is the CSS property value to check.
//
// Returns bool which is true if the value is auto, from-font, or a length
// unit such as px, em, rem, %, or 0.
func isTextDecorationThickness(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "auto" || value == "from-font" {
		return true
	}
	return strings.HasSuffix(value, "px") ||
		strings.HasSuffix(value, "em") ||
		strings.HasSuffix(value, "rem") ||
		strings.HasSuffix(value, "%") ||
		value == "0"
}
