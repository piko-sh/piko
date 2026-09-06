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

package pdfwriter_domain

import (
	"math"
	"strconv"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

// ClipShapeType identifies the kind of CSS clip-path basic shape.
type ClipShapeType int

const (
	// ClipShapeNone means no clip shape was parsed.
	ClipShapeNone ClipShapeType = iota

	// ClipShapeCircle represents clip-path: circle().
	ClipShapeCircle

	// ClipShapeEllipse represents clip-path: ellipse().
	ClipShapeEllipse

	// ClipShapeInset represents clip-path: inset().
	ClipShapeInset

	// ClipShapePolygon represents clip-path: polygon().
	ClipShapePolygon
)

const (
	// clipDefaultCentre is the default centre position as a fraction.
	clipDefaultCentre = 0.5

	// clipPercentDivisor converts a percentage to a fraction.
	clipPercentDivisor = 100

	// clipPxToPointFactor converts CSS pixels to PDF points.
	clipPxToPointFactor = 0.75

	// clipMinPolygonVertices is the minimum vertex count for a polygon.
	clipMinPolygonVertices = 3

	// clipMinInsetFieldsRadii is the field count when all four inset edges are specified.
	clipMinInsetFieldsRadii = 4
)

// ClipShape holds the parsed parameters of a CSS clip-path basic shape.
type ClipShape struct {
	// Points holds the polygon vertices as fractions of box dimensions.
	Points [][2]float64

	// Type identifies the kind of clip shape.
	Type ClipShapeType

	// CenterX holds the horizontal centre as a fraction of box width.
	CenterX float64

	// CenterY holds the vertical centre as a fraction of box height.
	CenterY float64

	// RadiusX holds the horizontal radius as a fraction of box width.
	RadiusX float64

	// RadiusY holds the vertical radius as a fraction of box height.
	RadiusY float64

	// InsetTop holds the top inset distance in points.
	InsetTop float64

	// InsetRight holds the right inset distance in points.
	InsetRight float64

	// InsetBottom holds the bottom inset distance in points.
	InsetBottom float64

	// InsetLeft holds the left inset distance in points.
	InsetLeft float64

	// InsetRadius holds the corner radius for inset shapes in points.
	InsetRadius float64
}

// ParseClipPath parses a CSS clip-path value into a ClipShape. Returns a shape with Type
// == ClipShapeNone for unsupported, empty or malformed values, including a truncated
// function such as "circle(" with no closing parenthesis.
//
// Takes value (string) which is the raw CSS clip-path value.
// Takes boxWidth (float64) which is the reference box width in points.
// Takes boxHeight (float64) which is the reference box height in points.
//
// Returns ClipShape with parsed parameters.
func ParseClipPath(value string, boxWidth, boxHeight float64) ClipShape {
	value = strings.TrimSpace(value)
	if value == "" || value == "none" {
		return ClipShape{}
	}

	if inner, ok := layouter_domain.CSSFunctionArguments(value, "circle"); ok {
		return parseClipCircle(inner, boxWidth, boxHeight)
	}
	if inner, ok := layouter_domain.CSSFunctionArguments(value, "ellipse"); ok {
		return parseClipEllipse(inner, boxWidth, boxHeight)
	}
	if inner, ok := layouter_domain.CSSFunctionArguments(value, "inset"); ok {
		return parseClipInset(inner, boxWidth, boxHeight)
	}
	if inner, ok := layouter_domain.CSSFunctionArguments(value, "polygon"); ok {
		return parseClipPolygon(inner, boxWidth, boxHeight)
	}

	return ClipShape{}
}

// parseClipCircle parses the arguments of a CSS circle() function.
//
// Takes inner (string) which is the text inside the parentheses.
// Takes boxWidth (float64) which is the reference box width in points.
// Takes boxHeight (float64) which is the reference box height in points.
//
// Returns ClipShape with circle parameters.
func parseClipCircle(inner string, boxWidth, boxHeight float64) ClipShape {
	shape := ClipShape{
		Type:        ClipShapeCircle,
		CenterX:     clipDefaultCentre,
		CenterY:     clipDefaultCentre,
		RadiusX:     clipDefaultCentre,
		Points:      nil,
		RadiusY:     0,
		InsetTop:    0,
		InsetRight:  0,
		InsetBottom: 0,
		InsetLeft:   0,
		InsetRadius: 0,
	}

	parts := strings.SplitN(inner, " at ", 2)
	if len(parts) == 2 {
		shape.CenterX, shape.CenterY = parsePosition(strings.TrimSpace(parts[1]))
	}

	radiusStr := strings.TrimSpace(parts[0])
	if radiusStr != "" && radiusStr != "closest-side" && radiusStr != "farthest-side" {
		shape.RadiusX = resolveClipLength(radiusStr, math.Min(boxWidth, boxHeight))
		if math.Min(boxWidth, boxHeight) > 0 {
			shape.RadiusX /= math.Min(boxWidth, boxHeight)
		}
	}

	return shape
}

// parseClipEllipse parses the arguments of a CSS ellipse() function.
//
// Takes inner (string) which is the text inside the parentheses.
// Takes boxWidth (float64) which is the reference box width in points.
// Takes boxHeight (float64) which is the reference box height in points.
//
// Returns ClipShape with ellipse parameters.
func parseClipEllipse(inner string, boxWidth, boxHeight float64) ClipShape {
	shape := ClipShape{
		Type:        ClipShapeEllipse,
		CenterX:     clipDefaultCentre,
		CenterY:     clipDefaultCentre,
		RadiusX:     clipDefaultCentre,
		RadiusY:     clipDefaultCentre,
		Points:      nil,
		InsetTop:    0,
		InsetRight:  0,
		InsetBottom: 0,
		InsetLeft:   0,
		InsetRadius: 0,
	}

	parts := strings.SplitN(inner, " at ", 2)
	if len(parts) == 2 {
		shape.CenterX, shape.CenterY = parsePosition(strings.TrimSpace(parts[1]))
	}

	radiiStr := strings.TrimSpace(parts[0])
	radii := strings.Fields(radiiStr)
	if len(radii) >= 2 {
		shape.RadiusX = resolveClipLength(radii[0], boxWidth)
		if boxWidth > 0 {
			shape.RadiusX /= boxWidth
		}
		shape.RadiusY = resolveClipLength(radii[1], boxHeight)
		if boxHeight > 0 {
			shape.RadiusY /= boxHeight
		}
	}

	return shape
}

// parseClipInset parses the arguments of a CSS inset() function.
//
// Takes inner (string) which is the text inside the parentheses.
// Takes boxWidth (float64) which is the reference box width in points.
// Takes boxHeight (float64) which is the reference box height in points.
//
// Returns ClipShape with inset edge distances and optional radius.
func parseClipInset(inner string, boxWidth, boxHeight float64) ClipShape {
	shape := ClipShape{}
	shape.Type = ClipShapeInset

	roundParts := strings.SplitN(inner, " round ", 2)
	if len(roundParts) == 2 {
		shape.InsetRadius = resolveClipLength(strings.TrimSpace(roundParts[1]), math.Min(boxWidth, boxHeight))
	}

	fields := strings.Fields(strings.TrimSpace(roundParts[0]))
	switch len(fields) {
	case 1:
		v := resolveClipLength(fields[0], boxHeight)
		shape.InsetTop = v
		shape.InsetRight = resolveClipLength(fields[0], boxWidth)
		shape.InsetBottom = v
		shape.InsetLeft = shape.InsetRight
	case 2:
		shape.InsetTop = resolveClipLength(fields[0], boxHeight)
		shape.InsetBottom = shape.InsetTop
		shape.InsetRight = resolveClipLength(fields[1], boxWidth)
		shape.InsetLeft = shape.InsetRight
	case clipMinPolygonVertices:
		shape.InsetTop = resolveClipLength(fields[0], boxHeight)
		shape.InsetRight = resolveClipLength(fields[1], boxWidth)
		shape.InsetBottom = resolveClipLength(fields[2], boxHeight)
		shape.InsetLeft = shape.InsetRight
	case clipMinInsetFieldsRadii:
		shape.InsetTop = resolveClipLength(fields[0], boxHeight)
		shape.InsetRight = resolveClipLength(fields[1], boxWidth)
		shape.InsetBottom = resolveClipLength(fields[2], boxHeight)
		shape.InsetLeft = resolveClipLength(fields[3], boxWidth)
	}

	return shape
}

// parseClipPolygon parses the arguments of a CSS polygon() function.
//
// Takes inner (string) which is the comma-separated vertex list.
// Takes boxWidth (float64) which is the reference box width in points.
// Takes boxHeight (float64) which is the reference box height in points.
//
// Returns ClipShape with polygon vertices as fractional coordinates.
func parseClipPolygon(inner string, boxWidth, boxHeight float64) ClipShape {
	shape := ClipShape{}
	shape.Type = ClipShapePolygon

	for vertex := range strings.SplitSeq(inner, ",") {
		coords := strings.Fields(strings.TrimSpace(vertex))
		if len(coords) < 2 {
			continue
		}
		x := clipFraction(resolveClipLength(coords[0], boxWidth), boxWidth)
		y := clipFraction(resolveClipLength(coords[1], boxHeight), boxHeight)
		shape.Points = append(shape.Points, [2]float64{x, y})
	}

	return shape
}

// parsePosition parses a CSS position value into fractional x and y coordinates (0 to 1).
//
// Takes pos (string) which is the space-separated position keywords or percentages.
//
// Returns x (float64) which is the horizontal fractional coordinate.
// Returns y (float64) which is the vertical fractional coordinate.
func parsePosition(pos string) (x, y float64) {
	parts := strings.Fields(pos)
	if len(parts) == 0 {
		return clipDefaultCentre, clipDefaultCentre
	}

	x = parsePercentOrKeyword(parts[0])
	y = clipDefaultCentre
	if len(parts) >= 2 {
		y = parsePercentOrKeyword(parts[1])
	}
	return x, y
}

// parsePercentOrKeyword converts a CSS position keyword or percentage string to a
// fractional value in [0, 1].
//
// Takes s (string) which is the keyword or percentage string.
//
// Returns float64 which is the fractional position.
func parsePercentOrKeyword(s string) float64 {
	switch s {
	case "left", "top":
		return 0
	case "right", "bottom":
		return 1
	case "center":
		return clipDefaultCentre
	}

	if after, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(after, 64)
		if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) {
			return v / clipPercentDivisor
		}
	}

	return clipDefaultCentre
}

// resolveClipLength resolves a CSS length or percentage string to an absolute value in
// points. Non-finite results are invalid and resolve to zero, so NaN or an infinity never
// reaches the PDF content stream.
//
// Takes s (string) which is the CSS length value.
// Takes reference (float64) which is the reference dimension for percentage resolution.
//
// Returns float64 which is the resolved length in points.
func resolveClipLength(s string, reference float64) float64 {
	resolved := resolveRawClipLength(s, reference)
	if math.IsNaN(resolved) || math.IsInf(resolved, 0) {
		return 0
	}
	return resolved
}

// resolveRawClipLength resolves a CSS length or percentage string to points without
// checking that the result is finite.
//
// Takes s (string) which is the CSS length value.
// Takes reference (float64) which is the reference dimension for percentage resolution.
//
// Returns float64 which is the resolved length in points.
func resolveRawClipLength(s string, reference float64) float64 {
	s = strings.TrimSpace(s)

	if after, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(after, 64)
		if err == nil {
			return v / clipPercentDivisor * reference
		}
		return 0
	}

	if after, ok := strings.CutSuffix(s, "px"); ok {
		v, err := strconv.ParseFloat(after, 64)
		if err == nil {
			return v * clipPxToPointFactor
		}
		return 0
	}

	if after, ok := strings.CutSuffix(s, "pt"); ok {
		v, err := strconv.ParseFloat(after, 64)
		if err == nil {
			return v
		}
		return 0
	}

	v, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return v * clipPxToPointFactor
	}

	return 0
}

// EmitClipPath writes PDF path operators for the given clip shape relative to a box at
// (pdfX, pdfY) with dimensions (w, h).
//
// Takes stream (*ContentStream) to write operators to.
// Takes shape (ClipShape) which is the parsed clip shape.
// Takes pdfX (float64) which is the reference box horizontal position in PDF coordinates.
// Takes pdfY (float64) which is the reference box vertical position in PDF coordinates.
// Takes w (float64) which is the rectangle width in points.
// Takes h (float64) which is the rectangle height in points.
func EmitClipPath(stream *ContentStream, shape ClipShape, pdfX, pdfY, w, h float64) {
	switch shape.Type { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case ClipShapeCircle:
		cx := pdfX + shape.CenterX*w
		cy := pdfY + shape.CenterY*h
		r := shape.RadiusX * math.Min(w, h)
		emitCirclePath(stream, cx, cy, r)
	case ClipShapeEllipse:
		cx := pdfX + shape.CenterX*w
		cy := pdfY + shape.CenterY*h
		rx := shape.RadiusX * w
		ry := shape.RadiusY * h
		emitEllipsePath(stream, cx, cy, rx, ry)
	case ClipShapeInset:
		ix := pdfX + shape.InsetLeft
		iy := pdfY + shape.InsetBottom
		iw := w - shape.InsetLeft - shape.InsetRight
		ih := h - shape.InsetTop - shape.InsetBottom
		if shape.InsetRadius > 0 {
			r := shape.InsetRadius
			emitRoundedRectPath(stream, ix, iy, iw, ih, r, r, r, r)
		} else {
			stream.Rectangle(ix, iy, iw, ih)
		}
	case ClipShapePolygon:
		if len(shape.Points) < clipMinPolygonVertices {
			return
		}
		for i, pt := range shape.Points {
			px := pdfX + pt[0]*w
			py := pdfY + (1-pt[1])*h
			if i == 0 {
				stream.MoveTo(px, py)
			} else {
				stream.LineTo(px, py)
			}
		}
		stream.ClosePath()
	}
}

// emitCirclePath approximates a circle using 4 cubic Bezier curves.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes cx (float64) which is the horizontal centre coordinate in points.
// Takes cy (float64) which is the vertical centre coordinate in points.
// Takes r (float64) which is the circle radius in points.
func emitCirclePath(stream *ContentStream, cx, cy, r float64) {
	k := r * kappa
	stream.MoveTo(cx+r, cy)
	stream.CurveTo(cx+r, cy+k, cx+k, cy+r, cx, cy+r)
	stream.CurveTo(cx-k, cy+r, cx-r, cy+k, cx-r, cy)
	stream.CurveTo(cx-r, cy-k, cx-k, cy-r, cx, cy-r)
	stream.CurveTo(cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	stream.ClosePath()
}

// emitEllipsePath approximates an ellipse using 4 cubic Bezier curves.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes cx (float64) which is the horizontal centre coordinate in points.
// Takes cy (float64) which is the vertical centre coordinate in points.
// Takes rx (float64) which is the horizontal radius in points.
// Takes ry (float64) which is the vertical radius in points.
func emitEllipsePath(stream *ContentStream, cx, cy, rx, ry float64) {
	kx := rx * kappa
	ky := ry * kappa
	stream.MoveTo(cx+rx, cy)
	stream.CurveTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	stream.CurveTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	stream.CurveTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	stream.CurveTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	stream.ClosePath()
}

// clipFraction converts an absolute coordinate to a fraction of the reference dimension,
// returning zero when the reference is not positive so a zero-sized box cannot produce
// NaN or infinite coordinates.
//
// Takes value (float64) which is the absolute coordinate in points.
// Takes reference (float64) which is the reference dimension in points.
//
// Returns float64 which is the fractional coordinate.
func clipFraction(value, reference float64) float64 {
	if !(reference > 0) {
		return 0
	}
	return value / reference
}
