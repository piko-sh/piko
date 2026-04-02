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

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

const (
	// percentSuffix holds the suffix string used to identify percentage values.
	percentSuffix = "%"
)

// resolveGradientFill resolves a url(#id) fill reference to a gradient
// and paints the current path with it.
//
// Takes rc (*renderContext) which provides the PDF stream and defs map.
// Takes ref (string) which is the gradient element id to look up.
// Takes style (*Style) which provides the fill rule for clipping.
//
// Returns bool which is true if a gradient was applied, false if the
// caller should fall back to solid fill.
func resolveGradientFill(rc *renderContext, ref string, style *Style) bool {
	node, ok := rc.defs[ref]
	if !ok {
		return false
	}

	switch node.Tag {
	case "linearGradient":
		return paintLinearGradient(rc, node, style)
	case "radialGradient":
		return paintRadialGradient(rc, node, style)
	}
	return false
}

// paintLinearGradient registers a linear gradient shading and paints
// it into the current clipped path.
//
// Takes rc (*renderContext) which provides the PDF stream and shading manager.
// Takes node (*Node) which is the linearGradient element with stops and coordinates.
// Takes style (*Style) which provides the fill rule for clipping.
//
// Returns bool which is true if the gradient was painted successfully.
func paintLinearGradient(rc *renderContext, node *Node, style *Style) bool {
	stops := parseGradientStops(node)
	if len(stops) < 2 {
		return false
	}

	x1 := gradientCoord(node, "x1", 0)
	y1 := gradientCoord(node, "y1", 0)
	x2 := gradientCoord(node, "x2", 1)
	y2 := gradientCoord(node, "y2", 0)

	if tf, ok := node.Attrs["gradientTransform"]; ok {
		m := ParseTransform(tf)

		nx1 := m.A*x1 + m.C*y1 + m.E
		ny1 := m.B*x1 + m.D*y1 + m.F
		nx2 := m.A*x2 + m.C*y2 + m.E
		ny2 := m.B*x2 + m.D*y2 + m.F
		x1, y1, x2, y2 = nx1, ny1, nx2, ny2
	}

	shadingName := rc.shadingManager.RegisterLinearGradient(x1, y1, x2, y2, stops)

	if style.FillRule == "evenodd" {
		rc.stream.ClipEvenOdd()
	} else {
		rc.stream.ClipNonZero()
	}
	rc.stream.PaintShading(shadingName)
	return true
}

// paintRadialGradient registers a radial gradient shading and paints
// it into the current clipped path.
//
// Takes rc (*renderContext) which provides the PDF stream and shading manager.
// Takes node (*Node) which is the radialGradient element with stops and coordinates.
// Takes style (*Style) which provides the fill rule for clipping.
//
// Returns bool which is true if the gradient was painted successfully.
func paintRadialGradient(rc *renderContext, node *Node, style *Style) bool {
	stops := parseGradientStops(node)
	if len(stops) < 2 {
		return false
	}

	cx := gradientCoord(node, "cx", gradientHalfDefault)
	cy := gradientCoord(node, "cy", gradientHalfDefault)
	r := gradientCoord(node, "r", gradientHalfDefault)

	if tf, ok := node.Attrs["gradientTransform"]; ok {
		m := ParseTransform(tf)
		cx = m.A*cx + m.C*cy + m.E
		cy = m.B*cx + m.D*cy + m.F

		r = r * (m.A + m.D) / 2
	}

	shadingName := rc.shadingManager.RegisterRadialGradient(cx, cy, r, stops)

	if style.FillRule == "evenodd" {
		rc.stream.ClipEvenOdd()
	} else {
		rc.stream.ClipNonZero()
	}
	rc.stream.PaintShading(shadingName)
	return true
}

// parseGradientStops extracts colour stops from a gradient element's
// <stop> children.
//
// Takes node (*Node) which is the gradient element containing stop children.
//
// Returns []pdfwriter_domain.ResolvedStop which holds the parsed and
// normalised gradient stops.
func parseGradientStops(node *Node) []pdfwriter_domain.ResolvedStop {
	var stops []pdfwriter_domain.ResolvedStop

	for _, child := range node.Children {
		if child.Tag != "stop" {
			continue
		}

		offset := parseStopOffset(child.Attrs["offset"])
		r, g, b, a := parseStopColour(child)

		if opStr, ok := child.Attrs["stop-opacity"]; ok {
			if v, err := strconv.ParseFloat(opStr, 64); err == nil {
				a = clamp01(v)
			}
		}

		if styleAttr, ok := child.Attrs["style"]; ok {
			r, g, b, a = applyInlineStopStyle(styleAttr, r, g, b, a)
		}

		stops = append(stops, pdfwriter_domain.ResolvedStop{
			Position: offset,
			Red:      r,
			Green:    g,
			Blue:     b,
			Alpha:    a,
		})
	}

	normaliseStopPositions(stops)
	return stops
}

// applyInlineStopStyle parses a stop element's inline style attribute
// and applies any stop-color or stop-opacity declarations.
//
// Takes styleAttr (string) which is the inline style attribute value.
// Takes r (float64) which is the current red component.
// Takes g (float64) which is the current green component.
// Takes b (float64) which is the current blue component.
// Takes a (float64) which is the current alpha component.
//
// Returns red (float64) which is the updated red component.
// Returns green (float64) which is the updated green component.
// Returns blue (float64) which is the updated blue component.
// Returns alpha (float64) which is the updated alpha component.
func applyInlineStopStyle(styleAttr string, r, g, b, a float64) (red float64, green float64, blue float64, alpha float64) {
	for decl := range strings.SplitSeq(styleAttr, ";") {
		decl = strings.TrimSpace(decl)
		key, val, found := strings.Cut(decl, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "stop-color":
			if c, ok := ParseColour(val); ok && !c.IsCurrentColour() {
				r, g, b = c.R, c.G, c.B
			}
		case "stop-opacity":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				a = clamp01(v)
			}
		}
	}
	return r, g, b, a
}

// normaliseStopPositions fills in missing (negative) stop positions by
// interpolating between neighbouring known positions.
//
// Takes stops ([]pdfwriter_domain.ResolvedStop) which is the slice of
// stops to normalise in place.
func normaliseStopPositions(stops []pdfwriter_domain.ResolvedStop) {
	if len(stops) > 0 && stops[0].Position < 0 {
		stops[0].Position = 0
	}
	if len(stops) > 1 && stops[len(stops)-1].Position < 0 {
		stops[len(stops)-1].Position = 1
	}

	for i := 1; i < len(stops)-1; i++ {
		if stops[i].Position < 0 {
			prev := stops[i-1].Position
			next := findNextKnownPosition(stops, i)
			stops[i].Position = prev + (next-prev)/float64(len(stops)-i)
		}
	}
}

// findNextKnownPosition returns the position of the next stop with a
// non-negative position after index i, defaulting to 1.0.
//
// Takes stops ([]pdfwriter_domain.ResolvedStop) which is the stop slice to search.
// Takes i (int) which is the index to start searching after.
//
// Returns float64 which is the next known position, or 1.0 if none exists.
func findNextKnownPosition(stops []pdfwriter_domain.ResolvedStop, i int) float64 {
	for j := i + 1; j < len(stops); j++ {
		if stops[j].Position >= 0 {
			return stops[j].Position
		}
	}
	return 1.0
}

// parseStopOffset parses a gradient stop offset value, handling both
// percentage and decimal formats.
//
// Takes s (string) which is the offset attribute value.
//
// Returns float64 which is the normalised offset in the range 0 to 1,
// or -1 if the value is empty or unparseable.
func parseStopOffset(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1
	}
	if trimmed, found := strings.CutSuffix(s, percentSuffix); found {
		v, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
		if err != nil {
			return -1
		}
		return clamp01(v / percentDivisor)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return clamp01(v)
}

// parseStopColour extracts the stop-color attribute from a stop element
// and returns its RGBA components.
//
// Takes node (*Node) which is the stop element to read the colour from.
//
// Returns r (float64) which is the red component.
// Returns g (float64) which is the green component.
// Returns b (float64) which is the blue component.
// Returns a (float64) which is the alpha component, defaulting to 1.0.
func parseStopColour(node *Node) (r, g, b, a float64) {
	a = 1.0
	colourStr, ok := node.Attrs["stop-color"]
	if !ok {
		return 0, 0, 0, a
	}
	c, parsed := ParseColour(colourStr)
	if !parsed || c.IsCurrentColour() {
		return 0, 0, 0, a
	}
	return c.R, c.G, c.B, c.A
}

// gradientCoord reads a named coordinate attribute from a gradient node,
// handling percentage values and falling back to a default.
//
// Takes node (*Node) which is the gradient element.
// Takes name (string) which is the attribute name to read.
// Takes fallback (float64) which is returned when the attribute is absent or unparseable.
//
// Returns float64 which is the parsed coordinate value.
func gradientCoord(node *Node, name string, fallback float64) float64 {
	s, ok := node.Attrs[name]
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if trimmed, found := strings.CutSuffix(s, percentSuffix); found {
		v, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
		if err != nil {
			return fallback
		}
		return v / percentDivisor
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}
