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
	"regexp"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// AttributeMapper is a function type that converts a CSS property and its
// value into an HTML attribute name and value. It returns the attribute name,
// attribute value, and a boolean that indicates whether the mapping succeeded.
type AttributeMapper func(cssProperty, cssValue string) (attributeName string, attributeValue string, ok bool)

var (
	// propertyToAttributeMap is a dispatch table that maps a CSS property to its
	// corresponding AttributeMapper function. This is highly extensible and
	// testable.
	propertyToAttributeMap = map[string]AttributeMapper{
		"width":            mapWidthHeight,
		"height":           mapWidthHeight,
		"background-color": mapBgColor,
		"background":       mapBackgroundShorthand,
		"background-image": mapBackgroundImage,
		"text-align":       mapTextAlign,
		"vertical-align":   mapVerticalAlign,
		"white-space":      mapWhiteSpace,
		"border":           mapBorder,
		"border-spacing":   mapCellspacing,
		"border-collapse":  mapBorderCollapse,
		"padding":          mapPadding,
		"cellpadding":      mapCellpadding,
		"cellspacing":      mapCellspacing,
	}

	// targetElementsForAttributeMap specifies which HTML attributes should be
	// applied to which tags. This is needed for compatibility (e.g., "bgcolor" only
	// applies to table elements).
	targetElementsForAttributeMap = map[string]map[string]bool{
		"width":       {"table": true, "td": true, "th": true, "img": true},
		"height":      {"table": true, "td": true, "th": true, "img": true},
		"bgcolor":     {"body": true, "table": true, "tr": true, "th": true, "td": true},
		"background":  {"table": true, "td": true, "th": true, "body": true},
		"align":       {"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "td": true, "th": true, "table": true, "img": true},
		"valign":      {"td": true, "th": true, "tr": true},
		"nowrap":      {"td": true, "th": true},
		"border":      {"table": true, "img": true},
		"cellspacing": {"table": true},
		"cellpadding": {"table": true},
	}

	// pxAndImportantRemover strips px units and !important from CSS values.
	pxAndImportantRemover = regexp.MustCompile(`px\s*(!important)?$`)

	importantRemover = regexp.MustCompile(`\s*!important\s*$`)
)

// ApplyAttributesFromStyle converts CSS properties to HTML attributes on a
// node.
//
// It uses a lookup table to map each CSS property to the correct HTML attribute
// for the element type.
//
// Takes node (*ast_domain.TemplateNode) which is the element to receive
// attributes.
// Takes styleMap (map[string]property) which contains the CSS properties to
// convert.
func ApplyAttributesFromStyle(node *ast_domain.TemplateNode, styleMap map[string]property) {
	nodeTag := strings.ToLower(node.TagName)

	for propName, propVal := range styleMap {
		applyPropertyAttribute(node, nodeTag, propName, propVal.value)
	}
}

// applyPropertyAttribute tries to apply a single CSS property as an HTML
// attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the target node to modify.
// Takes nodeTag (string) which is the HTML tag name of the node.
// Takes propName (string) which is the CSS property name to map.
// Takes propValue (string) which is the CSS property value to apply.
func applyPropertyAttribute(node *ast_domain.TemplateNode, nodeTag, propName, propValue string) {
	mapper, ok := propertyToAttributeMap[propName]
	if !ok {
		return
	}

	attributeName, attributeValue, success := mapper(propName, propValue)
	if !success {
		return
	}

	applicableTags, found := targetElementsForAttributeMap[attributeName]
	if !found {
		return
	}

	if !applicableTags[nodeTag] {
		return
	}

	if node.HasAttribute(attributeName) {
		return
	}

	node.SetAttribute(attributeName, attributeValue)
}

// mapWidthHeight converts CSS width or height values to HTML attribute format.
//
// Takes prop (string) which is the property name (width or height).
// Takes value (string) which is the CSS value to convert.
//
// Returns attributeName (string) which is the HTML attribute name.
// Returns attributeValue (string) which is the converted value without units for
// pixel values, or with the percent sign kept for percentage values.
// Returns ok (bool) which is true when the value was converted.
func mapWidthHeight(prop, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.TrimSpace(value)
	cleanValue := importantRemover.ReplaceAllString(value, "")
	cleanValue = strings.TrimSpace(cleanValue)

	if strings.HasSuffix(cleanValue, "px") {
		attributeValue = pxAndImportantRemover.ReplaceAllString(value, "")
		attributeValue = strings.TrimSpace(attributeValue)
		return prop, attributeValue, true
	}

	if strings.HasSuffix(cleanValue, literalPercent) {
		attributeValue = importantRemover.ReplaceAllString(value, "")
		attributeValue = strings.TrimSpace(attributeValue)
		return prop, attributeValue, true
	}

	if cleanValue == stringZero {
		return prop, stringZero, true
	}

	return "", "", false
}

// mapBgColor converts a background-color CSS property to a bgcolor HTML
// attribute.
//
// Takes value (string) which is the CSS colour value.
//
// Returns attributeName (string) which is "bgcolor".
// Returns attributeValue (string) which is the trimmed colour value.
// Returns ok (bool) which is always true.
func mapBgColor(_, value string) (attributeName string, attributeValue string, ok bool) {
	return "bgcolor", strings.TrimSpace(value), true
}

// mapBackgroundShorthand attempts to find a colour in a 'background' shorthand
// property. This is a simple implementation that looks for the first value that
// could be a colour.
//
// Takes value (string) which is the background shorthand CSS value to
// parse.
//
// Returns attributeName (string) which is "bgcolor" when a colour is found.
// Returns attributeValue (string) which is the extracted colour value.
// Returns ok (bool) which is true when a colour was found.
func mapBackgroundShorthand(_, value string) (attributeName string, attributeValue string, ok bool) {
	for part := range strings.FieldsSeq(value) {
		if strings.HasPrefix(part, "#") || strings.HasPrefix(part, "rgb") || strings.HasPrefix(part, "hsl") {
			return "bgcolor", part, true
		}
		if _, isColorName := colourNameToHex[strings.ToLower(part)]; isColorName {
			return "bgcolor", part, true
		}
	}
	return "", "", false
}

// mapTextAlign converts a text-align CSS property to an align HTML
// attribute.
//
// Takes value (string) which is the alignment value.
//
// Returns attributeName (string) which is "align".
// Returns attributeValue (string) which is the trimmed alignment value.
// Returns ok (bool) which is always true.
func mapTextAlign(_, value string) (attributeName string, attributeValue string, ok bool) {
	return "align", strings.TrimSpace(value), true
}

// mapVerticalAlign converts a vertical-align CSS property to a valign
// HTML attribute.
//
// Takes value (string) which is the vertical alignment value.
//
// Returns attributeName (string) which is "valign".
// Returns attributeValue (string) which is the trimmed alignment value.
// Returns ok (bool) which is always true.
func mapVerticalAlign(_, value string) (attributeName string, attributeValue string, ok bool) {
	return "valign", strings.TrimSpace(value), true
}

// mapBorder converts a border CSS property to an HTML border attribute
// when the border value is zero or none.
//
// Takes value (string) which is the CSS border value.
//
// Returns attributeName (string) which is "border" when the value is zero or
// none.
// Returns attributeValue (string) which is "0" when the value is zero or
// none.
// Returns ok (bool) which is true only when the border is zero or none.
func mapBorder(_, value string) (attributeName string, attributeValue string, ok bool) {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	if lowerValue == stringZero || lowerValue == "none" {
		return literalBorder, stringZero, true
	}
	return "", "", false
}

// mapCellspacing converts a border-spacing CSS property to a cellspacing
// HTML attribute when the value is zero.
//
// Takes value (string) which is the CSS spacing value.
//
// Returns attributeName (string) which is "cellspacing" when matched.
// Returns attributeValue (string) which is "0" when the value is zero.
// Returns ok (bool) which is true only when the value is zero or 0px.
func mapCellspacing(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.TrimSpace(value)
	if value == valueZero {
		return "cellspacing", valueZero, true
	}
	if strings.EqualFold(value, valueZeroPx) {
		return "cellspacing", valueZero, true
	}
	return "", "", false
}

// mapCellpadding converts a cellpadding CSS property to an HTML
// cellpadding attribute when the value is zero.
//
// Takes value (string) which is the CSS padding value.
//
// Returns attributeName (string) which is "cellpadding" when matched.
// Returns attributeValue (string) which is "0" when the value is zero.
// Returns ok (bool) which is true only when the value is zero or 0px.
func mapCellpadding(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.TrimSpace(value)
	if value == valueZero {
		return "cellpadding", valueZero, true
	}
	if strings.EqualFold(value, valueZeroPx) {
		return "cellpadding", valueZero, true
	}
	return "", "", false
}

// mapPadding converts a padding CSS property to a cellpadding HTML
// attribute when the value is zero.
//
// Takes value (string) which is the CSS padding value.
//
// Returns attributeName (string) which is "cellpadding" when matched.
// Returns attributeValue (string) which is "0" when the value is zero.
// Returns ok (bool) which is true only when the value is zero or 0px.
func mapPadding(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.TrimSpace(value)
	if value == valueZero || strings.EqualFold(value, valueZeroPx) {
		return "cellpadding", valueZero, true
	}
	return "", "", false
}

// mapBackgroundImage converts a background-image CSS property to an HTML
// background attribute by extracting the URL from a url() function.
//
// Takes value (string) which is the CSS background-image value.
//
// Returns attributeName (string) which is "background" when the URL is valid.
// Returns attributeValue (string) which is the extracted URL.
// Returns ok (bool) which is true when a valid url() value is found.
func mapBackgroundImage(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "url(") {
		return "", "", false
	}

	url := strings.TrimSuffix(strings.TrimPrefix(value, "url("), ")")
	url = strings.Trim(url, `"'`)
	url = strings.TrimSpace(url)

	if url == "" {
		return "", "", false
	}

	return "background", url, true
}

// mapBorderCollapse converts a border-collapse CSS property to a
// cellspacing HTML attribute when the value is "collapse".
//
// Takes value (string) which is the CSS border-collapse value.
//
// Returns attributeName (string) which is "cellspacing" when matched.
// Returns attributeValue (string) which is "0" when the value is "collapse".
// Returns ok (bool) which is true only when the value is "collapse".
func mapBorderCollapse(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "collapse" {
		return "cellspacing", valueZero, true
	}
	return "", "", false
}

// mapWhiteSpace converts a white-space CSS property to a nowrap HTML
// attribute when the value is "nowrap".
//
// Takes value (string) which is the CSS white-space value.
//
// Returns attributeName (string) which is "nowrap" when matched.
// Returns attributeValue (string) which is "nowrap" when the value is
// "nowrap".
// Returns ok (bool) which is true only when the value is "nowrap".
func mapWhiteSpace(_, value string) (attributeName string, attributeValue string, ok bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "nowrap" {
		return "nowrap", "nowrap", true
	}
	return "", "", false
}
