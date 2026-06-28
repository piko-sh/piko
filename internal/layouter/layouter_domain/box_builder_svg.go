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

package layouter_domain

import (
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// viewBoxFieldCount is the number of whitespace- or comma-separated fields in a
	// well-formed SVG viewBox attribute: min-x, min-y, width, height.
	viewBoxFieldCount = 4
)

// parseInlineSVGDimensions derives the intrinsic width and height (in points) of an
// inline <svg> element from its width/height attributes, falling back to the viewBox.
//
// Takes node (*ast_domain.TemplateNode) which is the inline <svg> element.
//
// Returns width (float64) which is the intrinsic width in points.
// Returns height (float64) which is the intrinsic height in points.
// Returns ok (bool) which is false when no usable dimensions are present.
func parseInlineSVGDimensions(node *ast_domain.TemplateNode) (width float64, height float64, ok bool) {
	var widthAttr, heightAttr, viewBoxAttr string
	for i := range node.Attributes {
		switch node.Attributes[i].Name {
		case "width":
			widthAttr = node.Attributes[i].Value
		case "height":
			heightAttr = node.Attributes[i].Value
		case "viewBox", "viewbox":
			viewBoxAttr = node.Attributes[i].Value
		}
	}

	width = parseSVGLength(widthAttr)
	height = parseSVGLength(heightAttr)

	if width > 0 && height > 0 {
		return width, height, true
	}

	viewBoxWidth, viewBoxHeight := parseViewBoxDimensions(viewBoxAttr)
	if width <= 0 {
		width = viewBoxWidth
	}
	if height <= 0 {
		height = viewBoxHeight
	}

	if width > 0 && height > 0 {
		return width, height, true
	}
	return 0, 0, false
}

// parseSVGLength parses an SVG length attribute (e.g. "16", "16px", "16pt") into points.
//
// Unitless and px values are treated as points (1:1 at the layouter's CSS-pixel == point
// scale).
//
// Takes value (string) which is the raw SVG length attribute.
//
// Returns float64 which is the length in points, or 0 on failure or for percentage
// values.
func parseSVGLength(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasSuffix(trimmed, "%") {
		return 0
	}
	trimmed = strings.TrimSuffix(trimmed, "px")
	trimmed = strings.TrimSuffix(trimmed, "pt")
	trimmed = strings.TrimSpace(trimmed)
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

// parseViewBoxDimensions returns the width and height components of an SVG viewBox
// ("min-x min-y width height").
//
// Takes viewBox (string) which is the raw SVG viewBox attribute.
//
// Returns width (float64) which is the viewBox width, or 0 when the viewBox is absent or
// malformed.
// Returns height (float64) which is the viewBox height, or 0 when the viewBox is absent
// or malformed.
func parseViewBoxDimensions(viewBox string) (width float64, height float64) {
	fields := strings.FieldsFunc(viewBox, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n'
	})
	if len(fields) != viewBoxFieldCount {
		return 0, 0
	}
	width, errW := strconv.ParseFloat(fields[2], 64)
	height, errH := strconv.ParseFloat(fields[3], 64)
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}
