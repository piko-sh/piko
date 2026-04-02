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

package driven_svgwriter

import (
	"strconv"
	"strings"
)

const (
	// defaultStrokeMitreLimit holds the SVG default stroke mitre limit.
	defaultStrokeMitreLimit = 4

	// defaultFontSize holds the SVG default font size in pixels.
	defaultFontSize = 16

	// styleValueNormal holds the CSS "normal" keyword used for font properties.
	styleValueNormal = "normal"
)

// Style holds resolved visual properties for an SVG node.
type Style struct {
	// Fill holds the resolved fill colour, or nil for "none".
	Fill *Colour

	// Stroke holds the resolved stroke colour, or nil for "none".
	Stroke *Colour

	// StrokeLineJoin holds the stroke line join style (mitre, round, or bevel).
	StrokeLineJoin string

	// FontFamily holds the CSS font-family value.
	FontFamily string

	// TextDecoration holds the CSS text-decoration value.
	TextDecoration string

	// StrokeRef holds the URL reference identifier for a stroke paint server.
	StrokeRef string

	// DominantBaseline holds the CSS dominant-baseline value for text alignment.
	DominantBaseline string

	// TextAnchor holds the CSS text-anchor value (start, middle, or end).
	TextAnchor string

	// FillRule holds the fill rule (nonzero or evenodd).
	FillRule string

	// StrokeLineCap holds the stroke line cap style (butt, round, or square).
	StrokeLineCap string

	// FillRef holds the URL reference identifier for a fill paint server.
	FillRef string

	// Visibility holds the CSS visibility value (visible, hidden, or collapse).
	Visibility string

	// FontStyle holds the CSS font-style value (normal, italic, or oblique).
	FontStyle string

	// FontWeight holds the CSS font-weight value.
	FontWeight string

	// Display holds the CSS display value.
	Display string

	// StrokeDashArray holds the stroke dash pattern as a slice of lengths.
	StrokeDashArray []float64

	// Colour holds the CSS color property for currentColor inheritance.
	Colour Colour

	// StrokeDashOffset holds the offset into the stroke dash pattern.
	StrokeDashOffset float64

	// FontSize holds the font size in pixels.
	FontSize float64

	// Opacity holds the element opacity in [0,1].
	Opacity float64

	// StrokeMitreLimit holds the stroke mitre limit ratio.
	StrokeMitreLimit float64

	// StrokeWidth holds the stroke width in user units.
	StrokeWidth float64

	// StrokeOpacity holds the stroke opacity in [0,1].
	StrokeOpacity float64

	// FillOpacity holds the fill opacity in [0,1].
	FillOpacity float64

	// LetterSpacing holds the additional spacing between characters in pixels.
	LetterSpacing float64

	// WordSpacing holds the additional spacing between words in pixels.
	WordSpacing float64
}

// DefaultStyle returns a Style with SVG default values.
// Per SVG spec the default fill is black.
//
// Returns Style which holds the SVG specification default values for all
// style properties.
func DefaultStyle() Style {
	black := Colour{R: 0, G: 0, B: 0, A: 1}
	return Style{
		Fill:             &black,
		FillOpacity:      1,
		FillRule:         "nonzero",
		StrokeOpacity:    1,
		StrokeWidth:      1,
		StrokeLineCap:    "butt",
		StrokeLineJoin:   "miter",
		StrokeMitreLimit: defaultStrokeMitreLimit,
		Opacity:          1,
		Display:          "inline",
		Visibility:       "visible",
		Colour:           black,
		FontFamily:       "sans-serif",
		FontSize:         defaultFontSize,
		FontWeight:       styleValueNormal,
		FontStyle:        styleValueNormal,
		TextAnchor:       "start",
		DominantBaseline: "auto",
	}
}

// ResolveStyle resolves style from a node's attributes and inline style,
// inheriting from the parent style where appropriate.
//
// Takes node (*Node) which specifies the SVG element whose attributes provide
// style overrides. Takes parent (*Style) which specifies the inherited style
// from the parent element.
//
// Returns Style which holds the fully resolved style with inheritance applied.
func ResolveStyle(node *Node, parent *Style) Style {
	s := Style{
		Fill:             parent.Fill,
		FillRef:          parent.FillRef,
		FillOpacity:      parent.FillOpacity,
		FillRule:         parent.FillRule,
		Stroke:           parent.Stroke,
		StrokeRef:        parent.StrokeRef,
		StrokeOpacity:    parent.StrokeOpacity,
		StrokeWidth:      parent.StrokeWidth,
		StrokeLineCap:    parent.StrokeLineCap,
		StrokeLineJoin:   parent.StrokeLineJoin,
		StrokeMitreLimit: parent.StrokeMitreLimit,
		Visibility:       parent.Visibility,
		Colour:           parent.Colour,
		FontFamily:       parent.FontFamily,
		FontSize:         parent.FontSize,
		FontWeight:       parent.FontWeight,
		FontStyle:        parent.FontStyle,
		TextAnchor:       parent.TextAnchor,
		DominantBaseline: parent.DominantBaseline,
		TextDecoration:   parent.TextDecoration,
		LetterSpacing:    parent.LetterSpacing,
		WordSpacing:      parent.WordSpacing,
	}

	s.Opacity = 1
	s.Display = "inline"
	s.StrokeDashArray = nil
	s.StrokeDashOffset = 0

	if node == nil || node.Attrs == nil {
		return s
	}

	applyProperties(&s, node.Attrs)

	if styleAttr, ok := node.Attrs["style"]; ok {
		props := parseInlineStyle(styleAttr)
		applyProperties(&s, props)
	}

	return s
}

// applyProperties applies the given CSS property map to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes props
// (map[string]string) which specifies the CSS property key-value pairs.
func applyProperties(s *Style, props map[string]string) {
	for key, val := range props {
		val = strings.TrimSpace(val)
		applyFillProperties(s, key, val)
		applyStrokeProperties(s, key, val)
		applyGeneralProperties(s, key, val)
		applyFontProperties(s, key, val)
		applyTextProperties(s, key, val)
	}
}

// applyFillProperties applies fill-related CSS properties to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes key (string)
// which specifies the CSS property name. Takes val (string) which specifies
// the CSS property value.
func applyFillProperties(s *Style, key, val string) {
	switch key {
	case "fill":
		applyFillColour(s, val)
	case "fill-opacity":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.FillOpacity = clamp01(v)
		}
	case "fill-rule":
		if val == "nonzero" || val == "evenodd" {
			s.FillRule = val
		}
	}
}

// applyFillColour parses and applies a fill colour value to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes val (string)
// which specifies the CSS fill value (colour, "none", or url() reference).
func applyFillColour(s *Style, val string) {
	if val == "none" {
		s.Fill = nil
		s.FillRef = ""
	} else if ref := ParseURLRef(val); ref != "" {
		s.FillRef = ref
	} else if c, ok := ParseColour(val); ok {
		s.Fill = resolveCurrentColour(c, s.Colour)
		s.FillRef = ""
	}
}

// applyStrokeProperties applies stroke-related CSS properties to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes key (string)
// which specifies the CSS property name. Takes val (string) which specifies
// the CSS property value.
func applyStrokeProperties(s *Style, key, val string) {
	switch key {
	case "stroke":
		applyStrokeColour(s, val)
	case "stroke-opacity":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.StrokeOpacity = clamp01(v)
		}
	case "stroke-width":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.StrokeWidth = v
		}
	case "stroke-linecap":
		applyStrokeLineCap(s, val)
	case "stroke-linejoin":
		applyStrokeLineJoin(s, val)
	case "stroke-miterlimit":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.StrokeMitreLimit = v
		}
	case "stroke-dasharray":
		applyStrokeDashArray(s, val)
	case "stroke-dashoffset":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.StrokeDashOffset = v
		}
	}
}

// applyStrokeColour parses and applies a stroke colour value to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes val (string)
// which specifies the CSS stroke value (colour, "none", or url() reference).
func applyStrokeColour(s *Style, val string) {
	if val == "none" {
		s.Stroke = nil
		s.StrokeRef = ""
	} else if ref := ParseURLRef(val); ref != "" {
		s.StrokeRef = ref
	} else if c, ok := ParseColour(val); ok {
		s.Stroke = resolveCurrentColour(c, s.Colour)
		s.StrokeRef = ""
	}
}

// resolveCurrentColour returns a pointer to the inherited colour when the
// parsed colour is currentColor, otherwise a pointer to the parsed colour
// itself.
//
// Takes parsed (Colour) which specifies the colour value to check. Takes
// inherited (Colour) which specifies the inherited currentColor value.
//
// Returns *Colour which holds a pointer to either the inherited colour or a
// copy of the parsed colour.
func resolveCurrentColour(parsed Colour, inherited Colour) *Colour {
	if parsed.IsCurrentColour() {
		return &inherited
	}
	return new(parsed)
}

// applyStrokeLineCap sets the stroke line cap if the value is valid.
//
// Takes s (*Style) which specifies the style to mutate. Takes val (string)
// which specifies the line cap value (butt, round, or square).
func applyStrokeLineCap(s *Style, val string) {
	if val == "butt" || val == "round" || val == "square" {
		s.StrokeLineCap = val
	}
}

// applyStrokeLineJoin sets the stroke line join if the value is valid.
//
// Takes s (*Style) which specifies the style to mutate. Takes val (string)
// which specifies the line join value (mitre, round, or bevel).
func applyStrokeLineJoin(s *Style, val string) {
	if val == "miter" || val == "round" || val == "bevel" {
		s.StrokeLineJoin = val
	}
}

// applyStrokeDashArray parses and applies a stroke dash array to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes val (string)
// which specifies the dash array value ("none" or comma/space-separated lengths).
func applyStrokeDashArray(s *Style, val string) {
	if val == "none" {
		s.StrokeDashArray = nil
	} else {
		s.StrokeDashArray = parseDashArray(val)
	}
}

// applyGeneralProperties applies general CSS properties such as opacity,
// display, visibility, and color to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes key (string)
// which specifies the CSS property name. Takes val (string) which specifies
// the CSS property value.
func applyGeneralProperties(s *Style, key, val string) {
	switch key {
	case "opacity":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.Opacity = clamp01(v)
		}
	case "display":
		s.Display = val
	case "visibility":
		if val == "visible" || val == "hidden" || val == "collapse" {
			s.Visibility = val
		}
	case "color":
		if c, ok := ParseColour(val); ok && !c.IsCurrentColour() {
			s.Colour = c
		}
	}
}

// applyFontProperties applies font-related CSS properties to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes key (string)
// which specifies the CSS property name. Takes val (string) which specifies
// the CSS property value.
func applyFontProperties(s *Style, key, val string) {
	switch key {
	case "font-family":
		s.FontFamily = val
	case "font-size":
		if v, err := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64); err == nil {
			s.FontSize = v
		}
	case "font-weight":
		s.FontWeight = val
	case "font-style":
		s.FontStyle = val
	}
}

// applyTextProperties applies text-related CSS properties to the style.
//
// Takes s (*Style) which specifies the style to mutate. Takes key (string)
// which specifies the CSS property name. Takes val (string) which specifies
// the CSS property value.
func applyTextProperties(s *Style, key, val string) {
	switch key {
	case "text-anchor":
		if val == "start" || val == "middle" || val == "end" {
			s.TextAnchor = val
		}
	case "dominant-baseline":
		if val == "auto" || val == "middle" || val == "hanging" || val == "central" ||
			val == "alphabetic" || val == "text-before-edge" || val == "text-after-edge" {
			s.DominantBaseline = val
		}
	case "text-decoration":
		s.TextDecoration = val
	case "letter-spacing":
		if val == styleValueNormal {
			s.LetterSpacing = 0
		} else if v, err := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64); err == nil {
			s.LetterSpacing = v
		}
	case "word-spacing":
		if val == styleValueNormal {
			s.WordSpacing = 0
		} else if v, err := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64); err == nil {
			s.WordSpacing = v
		}
	}
}

// parseInlineStyle parses a CSS inline style attribute into a property map.
//
// Takes s (string) which specifies the semicolon-separated CSS declarations.
//
// Returns map[string]string which holds the parsed property name-value pairs.
func parseInlineStyle(s string) map[string]string {
	result := make(map[string]string)
	for decl := range strings.SplitSeq(s, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		key, val, found := strings.Cut(decl, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key != "" {
			result[key] = val
		}
	}
	return result
}

// ParseURLRef extracts the id from a url(#id) reference.
//
// Takes s (string) which specifies the CSS value that may contain a url()
// reference.
//
// Returns string which holds the extracted identifier, or "" if the value is
// not a url() reference.
func ParseURLRef(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "url(") {
		return ""
	}
	s = strings.TrimPrefix(s, "url(")
	s = strings.TrimSuffix(s, ")")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "'\"")
	return strings.TrimPrefix(s, "#")
}

// parseDashArray parses a stroke-dasharray value into a slice of float64 lengths.
//
// Takes s (string) which specifies the comma or space-separated dash lengths.
//
// Returns []float64 which holds the parsed dash lengths, or nil if any value
// is invalid.
func parseDashArray(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	var result []float64
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil
		}
		result = append(result, v)
	}
	return result
}
