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
	"strconv"
	"strings"
)

// cssBackgroundImage is the CSS property name for background images.
const cssBackgroundImage = "background-image"

// backgroundKeywordMaps holds keyword classification maps for background
// properties.
type backgroundKeywordMaps struct {
	// repeat maps valid CSS background-repeat keywords to true.
	repeat map[string]bool

	// attachment maps valid background-attachment values for quick lookup.
	attachment map[string]bool

	// position maps valid CSS position keywords to true.
	position map[string]bool
}

// fontProperties holds the parts of a CSS font shorthand value.
type fontProperties struct {
	// style is the CSS font-style value, such as "italic" or "oblique".
	style string

	// variant is the font variant value (normal or small-caps); empty means not
	// set.
	variant string

	// weight is the font-weight value (normal, bold, bolder, lighter, or 100-900).
	weight string

	// size is the CSS font-size value.
	size string

	// lineHeight holds the CSS line-height value; empty means not set.
	lineHeight string

	// family is the font family name for the font-family CSS property.
	family string
}

// expandBackgroundShorthand splits a background shorthand property into its
// separate parts for email client support.
//
// Handles common email patterns including:
//   - background: colour;
//   - background: url(image.png);
//   - background: url(image.png) no-repeat centre / cover;
//   - background: url(image.png) centre fixed;
//   - background: linear-gradient(...), linear-gradient(...);
//
// Supports: image, position, size (with /), repeat, attachment, colour.
// Multiple comma-separated background layers are supported; each layer's
// background-image values are combined into a single comma-separated list.
//
// Takes value (string) which is the background shorthand value to expand.
//
// Returns map[string]string which contains the expanded background properties,
// or nil when the value cannot be parsed.
func expandBackgroundShorthand(value string) map[string]string {
	layers := splitCommaOutsideParens(value)

	if len(layers) <= 1 {
		return expandSingleBackgroundLayer(value)
	}

	var images []string
	var lastResult map[string]string
	for _, layer := range layers {
		layer = strings.TrimSpace(layer)
		if layer == "" {
			continue
		}
		layerResult := expandSingleBackgroundLayer(layer)
		if layerResult == nil {
			continue
		}
		if img, ok := layerResult[cssBackgroundImage]; ok {
			images = append(images, img)
		}
		lastResult = layerResult
	}

	if lastResult == nil {
		return nil
	}

	result := make(map[string]string)

	for key, val := range lastResult {
		if key != cssBackgroundImage {
			result[key] = val
		}
	}
	if len(images) > 0 {
		result[cssBackgroundImage] = strings.Join(images, ", ")
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// expandSingleBackgroundLayer expands a single background layer value.
//
// Takes value (string) which is the CSS background layer shorthand value.
//
// Returns map[string]string which maps individual background property names
// to their values, or nil when no properties are found.
func expandSingleBackgroundLayer(value string) map[string]string {
	result := make(map[string]string)

	value = extractBackgroundSize(value, result)

	parts := splitSpaceDelimited(value)
	if len(parts) == 0 {
		return nil
	}

	classifyBackgroundParts(parts, result)

	if len(result) == 0 {
		return nil
	}
	return result
}

// extractBackgroundSize handles the position/size separator (/) and extracts
// background-size from a CSS background shorthand value.
//
// Takes value (string) which is the background shorthand value to parse.
// Takes result (map[string]string) which stores the extracted background-size.
//
// Returns string which is the value with the size part removed.
func extractBackgroundSize(value string, result map[string]string) string {
	positionAndSize, afterSlash, found := strings.Cut(value, "/")
	if !found {
		return value
	}

	sizeAndRest := splitSpaceDelimited(afterSlash)
	if len(sizeAndRest) == 0 {
		return value
	}

	size, remainingValue := parseBackgroundSizeValue(sizeAndRest)
	result["background-size"] = size

	return positionAndSize + literalSpace + remainingValue
}

// parseBackgroundSizeValue extracts the background-size value from parsed
// parts.
//
// Takes sizeAndRest ([]string) which contains the size value and any leftover
// background shorthand parts.
//
// Returns size (string) which is the extracted background-size value. This is
// either a single value like "cover" or a two-value pair like "100% 50%".
// Returns remaining (string) which contains any leftover parts after the size.
func parseBackgroundSizeValue(sizeAndRest []string) (size string, remaining string) {
	if len(sizeAndRest) >= 2 && !isBackgroundKeyword(sizeAndRest[1]) {
		size = sizeAndRest[0] + literalSpace + sizeAndRest[1]
		remaining = strings.Join(sizeAndRest[2:], literalSpace)
	} else {
		size = sizeAndRest[0]
		remaining = strings.Join(sizeAndRest[1:], literalSpace)
	}
	return size, remaining
}

// classifyBackgroundParts groups background shorthand parts by property type
// and stores them in the result map.
//
// Takes parts ([]string) which contains the background shorthand parts to
// classify.
// Takes result (map[string]string) which receives the classified properties.
func classifyBackgroundParts(parts []string, result map[string]string) {
	keywords := buildBackgroundKeywordMaps()
	remainingParts := make([]string, 0, len(parts))
	positionParts := make([]string, 0, len(parts))

	for _, part := range parts {
		lower := strings.ToLower(part)

		if strings.HasPrefix(part, "url(") ||
			strings.HasPrefix(lower, "linear-gradient(") ||
			strings.HasPrefix(lower, "radial-gradient(") ||
			strings.HasPrefix(lower, "repeating-linear-gradient(") ||
			strings.HasPrefix(lower, "repeating-radial-gradient(") ||
			strings.HasPrefix(lower, "conic-gradient(") {
			result[cssBackgroundImage] = part
			continue
		}

		if keywords.repeat[lower] {
			_, exists := result["background-repeat"]
			if !exists {
				result["background-repeat"] = lower
			}
			continue
		}

		if keywords.attachment[lower] {
			result["background-attachment"] = lower
			continue
		}

		if keywords.position[lower] {
			positionParts = append(positionParts, lower)
			continue
		}

		remainingParts = append(remainingParts, part)
	}

	if len(positionParts) > 0 {
		result["background-position"] = strings.Join(positionParts, literalSpace)
	}

	if len(remainingParts) > 0 {
		result["background-color"] = strings.Join(remainingParts, literalSpace)
	}
}

// buildBackgroundKeywordMaps creates lookup maps for CSS background keywords.
//
// Returns backgroundKeywordMaps which holds maps for repeat, attachment, and
// position keywords used when parsing background properties.
func buildBackgroundKeywordMaps() backgroundKeywordMaps {
	return backgroundKeywordMaps{
		repeat: map[string]bool{
			"no-repeat": true, "repeat-x": true, "repeat-y": true,
			"repeat": true, "space": true, "round": true,
		},
		attachment: map[string]bool{
			"scroll": true, "fixed": true, "local": true,
		},
		position: map[string]bool{
			"top": true, "right": true, "bottom": true, "left": true, "center": true,
		},
	}
}

// isBackgroundKeyword reports whether a value is a background-related CSS
// keyword such as repeat, attachment, or URL to help tell it apart from
// background-size values.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value matches a background keyword.
func isBackgroundKeyword(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	keywords := []string{
		"no-repeat", "repeat-x", "repeat-y", "repeat", "space", "round",
		"scroll", "fixed", "local",
		"url(",
	}
	for _, keyword := range keywords {
		if lower == keyword || strings.HasPrefix(lower, keyword) {
			return true
		}
	}
	return false
}

// expandFontShorthand breaks a CSS font shorthand value into its separate
// properties.
//
// CSS font shorthand follows this pattern:
// [font-style] [font-variant] [font-weight] font-size[/line-height] font-family
//
// The size and family are required. The style, variant, and weight are optional
// but must appear in that order. This function is needed because many email
// clients do not handle the font shorthand well.
//
// Takes value (string) which is the CSS font shorthand value to expand.
//
// Returns map[string]string which holds the separate font properties, or nil
// when the value is empty, is a system font keyword, or cannot be parsed.
func expandFontShorthand(value string) map[string]string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if isSystemFont(value) {
		return nil
	}

	parts := splitSpaceDelimited(value)
	if len(parts) < minFontPartCount {
		return nil
	}

	props := parseFontProperties(parts)
	if props == nil {
		return nil
	}

	return buildFontPropertiesMap(props)
}

// isSystemFont reports whether the given value is a system font keyword.
// System font keywords should not be expanded during font shorthand processing.
//
// Takes value (string) which is the font value to check.
//
// Returns bool which is true if the value matches a system font keyword.
func isSystemFont(value string) bool {
	systemFonts := []string{"caption", "icon", "menu", "message-box", "small-caption", "status-bar"}
	lowerValue := strings.ToLower(value)
	return slices.Contains(systemFonts, lowerValue)
}

// parseFontProperties breaks down font shorthand parts into separate values.
//
// Takes parts ([]string) which contains the font shorthand values to parse.
//
// Returns *fontProperties which holds the parsed font style, variant, weight,
// size, line height, and family. Returns nil when required values are missing.
func parseFontProperties(parts []string) *fontProperties {
	props := &fontProperties{}
	i := 0

	if i < len(parts)-2 && isFontStyle(parts[i]) {
		props.style = parts[i]
		i++
	}

	if i < len(parts)-2 && isFontVariant(parts[i]) {
		props.variant = parts[i]
		i++
	}

	if i < len(parts)-2 && isFontWeight(parts[i]) {
		props.weight = parts[i]
		i++
	}

	if !parseFontSizeAndLineHeight(parts, i, props) {
		return nil
	}
	i++

	if i >= len(parts) {
		return nil
	}
	props.family = strings.Join(parts[i:], literalSpace)

	return props
}

// parseFontSizeAndLineHeight extracts the font size and optional line height
// from a font shorthand value.
//
// Takes parts ([]string) which contains the font shorthand tokens to parse.
// Takes index (int) which specifies the position to read from in parts.
// Takes props (*fontProperties) which receives the parsed size and line height.
//
// Returns bool which is true if parsing succeeded, false if index is out of
// bounds.
func parseFontSizeAndLineHeight(parts []string, index int, props *fontProperties) bool {
	if index >= len(parts) {
		return false
	}

	sizeAndLineHeight := parts[index]

	size, lineHeight, found := strings.Cut(sizeAndLineHeight, "/")
	if found {
		props.size = size
		props.lineHeight = lineHeight
	} else {
		props.size = sizeAndLineHeight
	}

	return true
}

// buildFontPropertiesMap builds a map of CSS font properties from parsed
// components.
//
// Takes props (*fontProperties) which contains the parsed font property values.
//
// Returns map[string]string which maps CSS property names to their values, or
// nil when no properties are set.
func buildFontPropertiesMap(props *fontProperties) map[string]string {
	result := make(map[string]string)

	if props.style != "" {
		result["font-style"] = props.style
	}
	if props.variant != "" {
		result["font-variant"] = props.variant
	}
	if props.weight != "" {
		result["font-weight"] = props.weight
	}
	if props.size != "" {
		result["font-size"] = props.size
	}
	if props.lineHeight != "" {
		result["line-height"] = props.lineHeight
	}
	if props.family != "" {
		result["font-family"] = props.family
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// isFontStyle checks whether a value is a valid CSS font-style keyword.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is "normal", "italic", or "oblique",
// false otherwise.
func isFontStyle(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "normal" || value == "italic" || value == "oblique"
}

// isFontVariant checks whether a value is a valid CSS font-variant keyword.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is "normal" or "small-caps".
func isFontVariant(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "normal" || value == "small-caps"
}

// isFontWeight reports whether a value is a valid CSS font-weight.
//
// Takes value (string) which is the font-weight value to check.
//
// Returns bool which is true if the value is a valid keyword (normal, bold,
// bolder, lighter) or a number from 100 to 900 in steps of 100.
func isFontWeight(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	weightKeywords := []string{"normal", "bold", "bolder", "lighter"}
	if slices.Contains(weightKeywords, value) {
		return true
	}

	if weight, err := strconv.Atoi(value); err == nil {
		return weight >= fontWeightMin && weight <= fontWeightMax && weight%fontWeightInterval == 0
	}

	return false
}
