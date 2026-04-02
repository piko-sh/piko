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

// Builds a PDF content stream from drawing operations. Each method appends
// raw PDF operators to an internal buffer. The final string is used as the
// stream body of a page's content object.

import (
	"fmt"
	"strings"
)

// thousandthsOfEm is the number of font design units per em,
// used to convert point-based advance differences to TJ adjustment values.
const thousandthsOfEm = 1000

// ContentStream accumulates PDF drawing operators for a single page.
type ContentStream struct {
	// builder holds the accumulated PDF operator text.
	builder strings.Builder
}

// SaveState pushes the current graphics state onto the stack (q operator).
func (stream *ContentStream) SaveState() {
	stream.builder.WriteString("q\n")
}

// RestoreState pops the graphics state from the stack (Q operator).
func (stream *ContentStream) RestoreState() {
	stream.builder.WriteString("Q\n")
}

// SetFillColourRGB sets the fill colour using RGB values in [0, 1].
//
// Takes red, green, blue (float64) which are the colour channel
// intensities, each in the range [0, 1].
func (stream *ContentStream) SetFillColourRGB(red, green, blue float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s rg\n",
		formatFloat(red), formatFloat(green), formatFloat(blue))
}

// SetStrokeColourRGB sets the stroke colour using RGB values in [0, 1].
//
// Takes red, green, blue (float64) which are the colour channel
// intensities, each in the range [0, 1].
func (stream *ContentStream) SetStrokeColourRGB(red, green, blue float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s RG\n",
		formatFloat(red), formatFloat(green), formatFloat(blue))
}

// SetFillColourGrey sets the fill colour using a grayscale value in [0, 1].
//
// Takes grey (float64) which is the grayscale intensity in [0, 1].
func (stream *ContentStream) SetFillColourGrey(grey float64) {
	fmt.Fprintf(&stream.builder, "%s g\n", formatFloat(grey))
}

// SetStrokeColourGrey sets the stroke colour using a grayscale value in [0, 1].
//
// Takes grey (float64) which is the grayscale intensity in [0, 1].
func (stream *ContentStream) SetStrokeColourGrey(grey float64) {
	fmt.Fprintf(&stream.builder, "%s G\n", formatFloat(grey))
}

// SetFillColourCMYK sets the fill colour using CMYK values in [0, 1].
//
// Takes cyan, magenta, yellow, key (float64) which are the CMYK
// channel values, each in the range [0, 1].
func (stream *ContentStream) SetFillColourCMYK(cyan, magenta, yellow, key float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s k\n",
		formatFloat(cyan), formatFloat(magenta), formatFloat(yellow), formatFloat(key))
}

// SetStrokeColourCMYK sets the stroke colour using CMYK values in [0, 1].
//
// Takes cyan, magenta, yellow, key (float64) which are the CMYK
// channel values, each in the range [0, 1].
func (stream *ContentStream) SetStrokeColourCMYK(cyan, magenta, yellow, key float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s K\n",
		formatFloat(cyan), formatFloat(magenta), formatFloat(yellow), formatFloat(key))
}

// SetLineWidth sets the stroke line width in points.
//
// Takes width (float64) which is the line width in points.
func (stream *ContentStream) SetLineWidth(width float64) {
	fmt.Fprintf(&stream.builder, "%s w\n", formatFloat(width))
}

// Rectangle appends a rectangle path. PDF coordinates: x, y is the
// lower-left corner.
//
// Takes x, y (float64) which specify the lower-left corner position.
// Takes width, height (float64) which specify the rectangle dimensions.
func (stream *ContentStream) Rectangle(x, y, width, height float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s re\n",
		formatFloat(x), formatFloat(y), formatFloat(width), formatFloat(height))
}

// Fill fills the current path using the non-zero winding rule.
func (stream *ContentStream) Fill() {
	stream.builder.WriteString("f\n")
}

// FillEvenOdd fills the current path using the even-odd rule.
func (stream *ContentStream) FillEvenOdd() {
	stream.builder.WriteString("f*\n")
}

// Stroke strokes the current path.
func (stream *ContentStream) Stroke() {
	stream.builder.WriteString("S\n")
}

// FillAndStroke fills and strokes the current path.
func (stream *ContentStream) FillAndStroke() {
	stream.builder.WriteString("B\n")
}

// MoveTo starts a new subpath at the given point.
//
// Takes x, y (float64) which specify the starting coordinates.
func (stream *ContentStream) MoveTo(x, y float64) {
	fmt.Fprintf(&stream.builder, "%s %s m\n", formatFloat(x), formatFloat(y))
}

// LineTo appends a straight line segment from the current point.
//
// Takes x, y (float64) which specify the endpoint coordinates.
func (stream *ContentStream) LineTo(x, y float64) {
	fmt.Fprintf(&stream.builder, "%s %s l\n", formatFloat(x), formatFloat(y))
}

// CurveTo appends a cubic Bezier curve from the current point using two
// control points (x1, y1) and (x2, y2) and the endpoint (x3, y3).
//
// Takes x1, y1 (float64) which specify the first control point.
// Takes x2, y2 (float64) which specify the second control point.
// Takes x3, y3 (float64) which specify the curve endpoint.
func (stream *ContentStream) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s %s %s c\n",
		formatFloat(x1), formatFloat(y1),
		formatFloat(x2), formatFloat(y2),
		formatFloat(x3), formatFloat(y3))
}

// ClosePath closes the current subpath by appending a straight line
// segment from the current point to the starting point of the subpath.
func (stream *ContentStream) ClosePath() {
	stream.builder.WriteString("h\n")
}

// Circle appends a closed circular path centred at (cx, cy) with
// radius r, using four cubic Bezier curves (the standard kappa
// approximation with k = 0.5522847498).
//
// Takes cx, cy (float64) which specify the centre coordinates.
// Takes r (float64) which is the circle radius in points.
func (stream *ContentStream) Circle(cx, cy, r float64) {
	const kappa = 0.5522847498
	kr := kappa * r
	stream.MoveTo(cx+r, cy)
	stream.CurveTo(cx+r, cy+kr, cx+kr, cy+r, cx, cy+r)
	stream.CurveTo(cx-kr, cy+r, cx-r, cy+kr, cx-r, cy)
	stream.CurveTo(cx-r, cy-kr, cx-kr, cy-r, cx, cy-r)
	stream.CurveTo(cx+kr, cy-r, cx+r, cy-kr, cx+r, cy)
	stream.ClosePath()
}

// ClipNonZero intersects the current clipping path with the current
// path using the non-zero winding rule, then ends the path without
// painting (W n operators).
func (stream *ContentStream) ClipNonZero() {
	stream.builder.WriteString("W n\n")
}

// ClipEvenOdd intersects the current clipping path with the current
// path using the even-odd rule, then ends the path without painting
// (W* n operators).
func (stream *ContentStream) ClipEvenOdd() {
	stream.builder.WriteString("W* n\n")
}

// FillEvenOddAndStroke fills the current path using the even-odd rule
// and then strokes it (B* operator).
func (stream *ContentStream) FillEvenOddAndStroke() {
	stream.builder.WriteString("B*\n")
}

// EndPath ends the current path without painting it (n operator).
// Used after clipping operations to consume the path.
func (stream *ContentStream) EndPath() {
	stream.builder.WriteString("n\n")
}

// SetMiterLimit sets the mitre limit for stroke joins (M operator).
// When the ratio of mitre length to line width exceeds this limit,
// mitreed joins are converted to bevel joins.
//
// Takes limit (float64) which is the mitre limit ratio.
func (stream *ContentStream) SetMiterLimit(limit float64) {
	fmt.Fprintf(&stream.builder, "%s M\n", formatFloat(limit))
}

// SetDashPattern sets the line dash pattern for subsequent stroke
// operations. The array defines alternating dash and gap lengths in
// points; phase is the starting offset into the pattern.
//
// Takes array ([]float64) which holds alternating dash/gap lengths.
// Takes phase (float64) which is the offset into the pattern.
func (stream *ContentStream) SetDashPattern(array []float64, phase float64) {
	stream.builder.WriteByte('[')
	for i, value := range array {
		if i > 0 {
			stream.builder.WriteByte(' ')
		}
		stream.builder.WriteString(formatFloat(value))
	}
	fmt.Fprintf(&stream.builder, "] %s d\n", formatFloat(phase))
}

// SetLineCap sets the line cap style: 0 = butt, 1 = round,
// 2 = projecting square.
//
// Takes style (int) which is the cap style index.
func (stream *ContentStream) SetLineCap(style int) {
	fmt.Fprintf(&stream.builder, "%d J\n", style)
}

// SetLineJoin sets the line join style: 0 = mitre, 1 = round,
// 2 = bevel.
//
// Takes join (int) which is the join style index.
func (stream *ContentStream) SetLineJoin(join int) {
	fmt.Fprintf(&stream.builder, "%d j\n", join)
}

// SetExtGState applies a named graphics state parameter dictionary
// from the page resources.
//
// Takes name (string) which is the resource dictionary key.
func (stream *ContentStream) SetExtGState(name string) {
	fmt.Fprintf(&stream.builder, "/%s gs\n", name)
}

// ConcatMatrix concatenates a 2D affine transformation matrix with
// the current transformation matrix. The six values [a b c d e f]
// define the matrix.
//
// Takes a, b, c, d, e, f (float64) which are the affine matrix
// components.
func (stream *ContentStream) ConcatMatrix(a, b, c, d, e, f float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s %s %s cm\n",
		formatFloat(a), formatFloat(b),
		formatFloat(c), formatFloat(d),
		formatFloat(e), formatFloat(f))
}

// PaintXObject paints the named XObject (image or form) from the page
// resources.
//
// Takes name (string) which is the XObject resource key.
func (stream *ContentStream) PaintXObject(name string) {
	fmt.Fprintf(&stream.builder, "/%s Do\n", name)
}

// PaintShading paints a shading pattern identified by name.
//
// Takes name (string) which is the shading resource key.
func (stream *ContentStream) PaintShading(name string) {
	fmt.Fprintf(&stream.builder, "/%s sh\n", name)
}

// BeginText begins a text object.
func (stream *ContentStream) BeginText() {
	stream.builder.WriteString("BT\n")
}

// EndText ends a text object.
func (stream *ContentStream) EndText() {
	stream.builder.WriteString("ET\n")
}

// SetCharSpacing sets extra spacing between characters in points (Tc operator).
// A value of 0 resets to the default.
//
// Takes spacing (float64) which is the extra character spacing in points.
func (stream *ContentStream) SetCharSpacing(spacing float64) {
	fmt.Fprintf(&stream.builder, "%s Tc\n", formatFloat(spacing))
}

// SetWordSpacing sets extra spacing added after each ASCII space
// character (code 0x20) in points (Tw operator).
//
// A value of 0 resets to the default. For CIDFont Type2 with identity
// encoding, Tw may not apply because the space glyph has a CID other
// than 32; use glyph advance adjustments instead.
//
// Takes spacing (float64) which is the extra word spacing in points.
func (stream *ContentStream) SetWordSpacing(spacing float64) {
	fmt.Fprintf(&stream.builder, "%s Tw\n", formatFloat(spacing))
}

// SetTextRenderingMode sets how text characters are painted (Tr operator).
// Mode 0 = fill (default), 1 = stroke, 2 = fill then stroke,
// 3 = invisible, 4 = fill then clip, 5 = stroke then clip,
// 6 = fill stroke then clip, 7 = clip only.
//
// Takes mode (int) which is the rendering mode index.
func (stream *ContentStream) SetTextRenderingMode(mode int) {
	fmt.Fprintf(&stream.builder, "%d Tr\n", mode)
}

// SetFont selects a font by its resource name and size in points.
//
// Takes name (string) which is the font resource key.
// Takes size (float64) which is the font size in points.
func (stream *ContentStream) SetFont(name string, size float64) {
	fmt.Fprintf(&stream.builder, "/%s %s Tf\n", name, formatFloat(size))
}

// MoveText moves the text position by the given offsets.
//
// Takes x, y (float64) which are the horizontal and vertical offsets.
func (stream *ContentStream) MoveText(x, y float64) {
	fmt.Fprintf(&stream.builder, "%s %s Td\n", formatFloat(x), formatFloat(y))
}

// SetTextMatrix sets the text matrix and text line matrix (Tm operator).
// The six values [a b c d e f] define an affine transformation applied
// to text drawn within the current text object.
//
// Takes a, b, c, d, e, f (float64) which are the affine matrix
// components.
func (stream *ContentStream) SetTextMatrix(a, b, c, d, e, f float64) {
	fmt.Fprintf(&stream.builder, "%s %s %s %s %s %s Tm\n",
		formatFloat(a), formatFloat(b),
		formatFloat(c), formatFloat(d),
		formatFloat(e), formatFloat(f))
}

// ShowText paints the given text string using the Tj operator (Type1 fonts).
//
// Takes text (string) which is the text to render.
func (stream *ContentStream) ShowText(text string) {
	fmt.Fprintf(&stream.builder, "%s Tj\n", escapeString(text))
}

// ShowGlyphs emits a TJ operator that renders the given glyphs with
// per-glyph positioning adjustments. Each glyph is placed using its
// shaped advance (from harfbuzz) rather than the font's default advance,
// so kerning and GPOS adjustments are honoured in the PDF output.
//
// Takes glyphIDs ([]uint16) which are the glyph identifiers.
// Takes shapedAdvances ([]float64) which are the harfbuzz-shaped advances
// in points.
// Takes defaultAdvances ([]float64) which are the font's default advances
// in points (from the hmtx table, scaled to fontSize).
// Takes fontSize (float64) which is the font size in points.
func (stream *ContentStream) ShowGlyphs(glyphIDs []uint16, shapedAdvances []float64, defaultAdvances []float64, fontSize float64) {
	if len(glyphIDs) == 0 {
		return
	}

	stream.builder.WriteByte('[')
	for i, glyphID := range glyphIDs {
		fmt.Fprintf(&stream.builder, "<%04X>", glyphID)
		if i < len(glyphIDs)-1 && i < len(shapedAdvances) && i < len(defaultAdvances) {
			diffPt := defaultAdvances[i] - shapedAdvances[i]
			if diffPt != 0 && fontSize > 0 {
				adjustment := diffPt * thousandthsOfEm / fontSize
				fmt.Fprintf(&stream.builder, " %s", formatFloat(adjustment))
			}
		}
	}
	stream.builder.WriteString("] TJ\n")
}

// BeginMarkedContent starts a marked content sequence with the given
// structure tag and marked content ID (BDC operator). Used for tagged PDF.
//
// Takes tag (string) which is the structure element type.
// Takes mcid (int) which is the marked content identifier.
func (stream *ContentStream) BeginMarkedContent(tag string, mcid int) {
	fmt.Fprintf(&stream.builder, "/%s <</MCID %d>> BDC\n", tag, mcid)
}

// EndMarkedContent ends the current marked content sequence (EMC operator).
func (stream *ContentStream) EndMarkedContent() {
	stream.builder.WriteString("EMC\n")
}

// String returns the accumulated PDF operators as a string.
//
// Returns string which holds the complete content stream text.
func (stream *ContentStream) String() string {
	return stream.builder.String()
}

// formatFloat formats a float64 as a compact string for PDF output.
// Whole numbers are rendered without a decimal point; all others use
// two decimal places.
//
// Takes value (float64) which is the number to format.
//
// Returns string which holds the formatted representation.
func formatFloat(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%.2f", value)
}

// escapeString wraps text in PDF literal string delimiters and
// escapes parentheses and backslashes.
//
// Takes text (string) which is the raw text to escape.
//
// Returns string which holds the parenthesised PDF string literal.
func escapeString(text string) string {
	var builder strings.Builder
	builder.WriteByte('(')
	for _, character := range text {
		switch character {
		case '(', ')':
			builder.WriteByte('\\')
			_, _ = builder.WriteRune(character)
		case '\\':
			builder.WriteString("\\\\")
		default:
			_, _ = builder.WriteRune(character)
		}
	}
	builder.WriteByte(')')
	return builder.String()
}
