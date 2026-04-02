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

// fourValues holds four resolved CSS shorthand values.
type fourValues struct {
	// a is the first (top) value in the CSS shorthand expansion.
	a string

	// b is the second (right) value in the CSS shorthand expansion.
	b string

	// c is the third (bottom) value in the CSS shorthand expansion.
	c string

	// d is the fourth (left) value in the CSS shorthand expansion.
	d string
}

// expandFourValues applies the CSS 1-4 value shorthand expansion pattern,
// returning the four resolved values in order (first, second, third, fourth).
//
// Takes value (string) which contains the space-delimited shorthand values.
//
// Returns fourValues which contains the resolved values.
// Returns bool which is false when the input is empty or has more than four
// values.
func expandFourValues(value string) (fourValues, bool) {
	values := splitSpaceDelimited(value)
	switch len(values) {
	case oneValueCount:
		return fourValues{a: values[indexFirst], b: values[indexFirst], c: values[indexFirst], d: values[indexFirst]}, true
	case twoValueCount:
		return fourValues{a: values[indexFirst], b: values[indexSecond], c: values[indexFirst], d: values[indexSecond]}, true
	case threeValueCount:
		return fourValues{a: values[indexFirst], b: values[indexSecond], c: values[indexThird], d: values[indexSecond]}, true
	case fourValueCount:
		return fourValues{a: values[indexFirst], b: values[indexSecond], c: values[indexThird], d: values[indexFourth]}, true
	default:
		return fourValues{}, false
	}
}

// expandFourValueShorthand expands CSS shorthand properties that follow the
// standard 1-4 value pattern into individual directional properties.
//
// Handles margin, padding, border-width, border-style, and border-colour which
// all follow the same pattern:
//   - 1 value: all sides
//   - 2 values: top/bottom, right/left
//   - 3 values: top, right/left, bottom
//   - 4 values: top, right, bottom, left
//
// Takes prefix (string) which is the property name prefix (e.g. "margin").
// Takes value (string) which contains the space-delimited shorthand values.
// Takes suffix (...string) which provides an optional property suffix
// (e.g. "width" for border-width).
//
// Returns map[string]string which maps each directional property name to its
// value, or nil when the input is empty or has more than four values.
func expandFourValueShorthand(prefix, value string, suffix ...string) map[string]string {
	fv, ok := expandFourValues(value)
	if !ok {
		return nil
	}
	top, right, bottom, left := fv.a, fv.b, fv.c, fv.d

	var propSuffix string
	if len(suffix) > 0 {
		propSuffix = literalDash + suffix[0]
	}

	return map[string]string{
		prefix + literalDash + directionTop + propSuffix:    top,
		prefix + literalDash + directionRight + propSuffix:  right,
		prefix + literalDash + directionBottom + propSuffix: bottom,
		prefix + literalDash + directionLeft + propSuffix:   left,
	}
}

// expandBorderShorthand expands a border shorthand value into separate border
// properties for all four sides. This helps email clients like Outlook display
// borders correctly.
//
// Takes value (string) which is the border shorthand (e.g. "1px solid black").
//
// Returns map[string]string which contains the expanded border properties for
// all four sides, or nil if the value is empty or has no valid properties.
func expandBorderShorthand(value string) map[string]string {
	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	var width, style, color string
	for _, part := range parts {
		if isBorderWidth(part) {
			width = part
		} else if isBorderStyle(part) {
			style = part
		} else {
			color = part
		}
	}

	result := make(map[string]string)

	sides := []string{directionTop, directionRight, directionBottom, directionLeft}
	if width != "" {
		for _, sideName := range sides {
			result[prefixBorder+sideName+"-width"] = width
		}
	}
	if style != "" {
		for _, sideName := range sides {
			result[prefixBorder+sideName+"-style"] = style
		}
	}
	if color != "" {
		for _, sideName := range sides {
			result[prefixBorder+sideName+"-color"] = color
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// expandDirectionalBorderShorthand expands a directional border shorthand like
// "border-top: 1px solid black" into separate width, style, and colour
// properties.
//
// Takes direction (string) which specifies the border side (e.g. "top").
// Takes value (string) which contains the shorthand value to expand.
//
// Returns map[string]string which maps each property name to its value, or nil
// when the value is empty or contains no valid parts.
func expandDirectionalBorderShorthand(direction, value string) map[string]string {
	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	var width, style, color string
	for _, part := range parts {
		if isBorderWidth(part) {
			width = part
		} else if isBorderStyle(part) {
			style = part
		} else {
			color = part
		}
	}

	result := make(map[string]string)
	if width != "" {
		result[prefixBorder+direction+"-width"] = width
	}
	if style != "" {
		result[prefixBorder+direction+"-style"] = style
	}
	if color != "" {
		result[prefixBorder+direction+"-color"] = color
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// expandWidthStyleColour parses a CSS shorthand value containing up to three
// tokens (width, style, colour) and returns each as a separate longhand
// property using the supplied prefix.
//
// Takes prefix (string) which is the CSS property prefix (e.g. "outline" or
// "column-rule").
// Takes value (string) which is the shorthand value to expand.
//
// Returns map[string]string which maps each longhand property to its value,
// or nil when the value is empty or has no valid parts.
func expandWidthStyleColour(prefix, value string) map[string]string {
	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	var width, style, color string
	for _, part := range parts {
		if isOutlineWidth(part) {
			width = part
		} else if isOutlineStyle(part) {
			style = part
		} else {
			color = part
		}
	}

	result := make(map[string]string)
	if width != "" {
		result[prefix+"-width"] = width
	}
	if style != "" {
		result[prefix+"-style"] = style
	}
	if color != "" {
		result[prefix+"-color"] = color
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// expandOutlineShorthand expands a CSS outline shorthand value into its
// separate properties.
//
// Outline uses the same format as border (width, style, colour) but applies
// to all sides at once.
//
// Takes value (string) which is the shorthand value like "1px solid black".
//
// Returns map[string]string which contains the expanded properties, or nil
// when the value is empty or has no valid outline parts.
func expandOutlineShorthand(value string) map[string]string {
	return expandWidthStyleColour("outline", value)
}

// expandColumnRuleShorthand splits a column-rule shorthand value into its
// separate width, style, and colour longhand properties. The syntax matches
// the outline shorthand: <width> || <style> || <colour> in any order.
//
// Takes value (string) which is the column-rule shorthand value to expand.
//
// Returns map[string]string which contains the expanded column-rule
// properties, or nil when the value cannot be parsed.
func expandColumnRuleShorthand(value string) map[string]string {
	return expandWidthStyleColour("column-rule", value)
}

// expandBorderRadiusShorthand expands "border-radius" into 4 corner properties.
//
// Takes value (string) which is the shorthand border-radius CSS value.
//
// Returns map[string]string which maps each corner property to its value, or
// nil when the input is empty or has more than four values.
func expandBorderRadiusShorthand(value string) map[string]string {
	fv, ok := expandFourValues(value)
	if !ok {
		return nil
	}
	topLeft, topRight, bottomRight, bottomLeft := fv.a, fv.b, fv.c, fv.d

	return map[string]string{
		"border-top-left-radius":     topLeft,
		"border-top-right-radius":    topRight,
		"border-bottom-right-radius": bottomRight,
		"border-bottom-left-radius":  bottomLeft,
	}
}

// isBorderWidth reports whether a value is a valid CSS border width.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is a valid border width keyword
// (thin, medium, thick) or a length unit (px, em, rem, pt, %, or 0).
func isBorderWidth(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "thin" || value == "medium" || value == "thick" {
		return true
	}
	return strings.HasSuffix(value, "px") ||
		strings.HasSuffix(value, "em") ||
		strings.HasSuffix(value, "rem") ||
		strings.HasSuffix(value, "pt") ||
		strings.HasSuffix(value, "%") ||
		value == "0"
}

// isBorderStyle reports whether a value is a valid CSS border style.
//
// Takes value (string) which is the border style to check.
//
// Returns bool which is true if the value matches a known border style.
func isBorderStyle(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	styles := []string{"none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"}
	return slices.Contains(styles, value)
}

// isOutlineWidth checks whether a value is a valid outline width.
// Outline widths use the same values as border widths.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is a valid outline width.
func isOutlineWidth(value string) bool {
	return isBorderWidth(value)
}

// isOutlineStyle checks whether a value is a valid outline style.
// Outline styles use the same values as border styles.
//
// Takes value (string) which is the style value to check.
//
// Returns bool which is true if the value is a valid outline style.
func isOutlineStyle(value string) bool {
	return isBorderStyle(value)
}
