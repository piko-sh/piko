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
	"strings"
)

// shorthandExpander is a function type that expands shorthand properties into
// their full key-value pairs.
type shorthandExpander func(string) map[string]string

// shorthandExpanderMap maps CSS property names to their expansion functions.
var shorthandExpanderMap = map[string]shorthandExpander{
	"margin":          func(v string) map[string]string { return expandFourValueShorthand("margin", v) },
	"padding":         func(v string) map[string]string { return expandFourValueShorthand("padding", v) },
	"border-width":    func(v string) map[string]string { return expandFourValueShorthand(literalBorder, v, "width") },
	"border-style":    func(v string) map[string]string { return expandFourValueShorthand(literalBorder, v, "style") },
	"border-color":    func(v string) map[string]string { return expandFourValueShorthand(literalBorder, v, "color") },
	literalBorder:     expandBorderShorthand,
	"border-top":      func(v string) map[string]string { return expandDirectionalBorderShorthand(directionTop, v) },
	"border-right":    func(v string) map[string]string { return expandDirectionalBorderShorthand(directionRight, v) },
	"border-bottom":   func(v string) map[string]string { return expandDirectionalBorderShorthand(directionBottom, v) },
	"border-left":     func(v string) map[string]string { return expandDirectionalBorderShorthand(directionLeft, v) },
	"outline":         expandOutlineShorthand,
	"column-rule":     expandColumnRuleShorthand,
	"list-style":      expandListStyleShorthand,
	"overflow":        expandOverflowShorthand,
	"inset":           expandInsetShorthand,
	"text-decoration": expandTextDecorationShorthand,
	"background":      expandBackgroundShorthand,
	"font":            expandFontShorthand,
	"border-radius":   expandBorderRadiusShorthand,
	"flex":            expandFlexShorthand,
	"flex-flow":       expandFlexFlowShorthand,
}

// shorthandLonghands maps shorthand property names to their longhand
// equivalents, used for distributing CSS-wide keywords.
var shorthandLonghands = map[string][]string{
	"margin":       {"margin-top", "margin-right", "margin-bottom", "margin-left"},
	"padding":      {"padding-top", "padding-right", "padding-bottom", "padding-left"},
	"border-width": {"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"},
	"border-style": {"border-top-style", "border-right-style", "border-bottom-style", "border-left-style"},
	"border-color": {"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"},
	literalBorder: {
		"border-top-width", "border-top-style", "border-top-color",
		"border-right-width", "border-right-style", "border-right-color",
		"border-bottom-width", "border-bottom-style", "border-bottom-color",
		"border-left-width", "border-left-style", "border-left-color",
	},
	"border-top":      {"border-top-width", "border-top-style", "border-top-color"},
	"border-right":    {"border-right-width", "border-right-style", "border-right-color"},
	"border-bottom":   {"border-bottom-width", "border-bottom-style", "border-bottom-color"},
	"border-left":     {"border-left-width", "border-left-style", "border-left-color"},
	"outline":         {"outline-width", "outline-style", "outline-color"},
	"column-rule":     {"column-rule-width", "column-rule-style", "column-rule-color"},
	"overflow":        {"overflow-x", "overflow-y"},
	"inset":           {"top", "right", "bottom", "left"},
	"text-decoration": {"text-decoration-line", "text-decoration-style", "text-decoration-color", "text-decoration-thickness"},
	"background":      {"background-color", "background-image", "background-repeat", "background-position", "background-size"},
	"font":            {"font-style", "font-weight", "font-size", "line-height", "font-family"},
	"border-radius":   {"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius"},
	"flex":            {"flex-grow", "flex-shrink", "flex-basis"},
	"flex-flow":       {"flex-direction", "flex-wrap"},
}

// expandShorthand expands a CSS shorthand property into its longhand form.
//
// CSS-wide keywords (inherit, initial, unset, revert) are not expanded here
// because the email path keeps shorthands as-is. Use expandCSSWideKeyword
// to distribute keywords to longhands when needed (e.g. for layout).
//
// Takes propName (string) which is the CSS property name to expand.
// Takes value (string) which is the property value to expand.
//
// Returns map[string]string which maps longhand property names to values,
// or nil if the property is not a shorthand, cannot be expanded, or the value
// is a CSS-wide keyword.
func expandShorthand(propName, value string) map[string]string {
	if isCSSWideKeyword(value) {
		return nil
	}

	if expander, ok := shorthandExpanderMap[propName]; ok {
		return expander(value)
	}

	return nil
}

// expandCSSWideKeyword distributes a CSS-wide keyword (inherit, initial, unset,
// revert) across all longhands of a shorthand property.
//
// Takes propName (string) which is the shorthand property name.
// Takes value (string) which is the CSS-wide keyword.
//
// Returns map[string]string with all longhands set to the keyword, or nil if
// the property is not a known shorthand.
func expandCSSWideKeyword(propName, value string) map[string]string {
	longhands, ok := shorthandLonghands[propName]
	if !ok {
		return nil
	}
	result := make(map[string]string, len(longhands))
	for _, lh := range longhands {
		result[lh] = value
	}
	return result
}

// isCSSWideKeyword checks if a value is a CSS-wide keyword that should not be
// expanded.
//
// Takes value (string) which is the CSS property value to check.
//
// Returns bool which is true if the value is inherit, initial, unset, or
// revert.
func isCSSWideKeyword(value string) bool {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	return lowerValue == "inherit" || lowerValue == "initial" || lowerValue == "unset" || lowerValue == "revert"
}
