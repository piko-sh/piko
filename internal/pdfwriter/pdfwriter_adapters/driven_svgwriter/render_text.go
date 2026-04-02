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
	"strings"
)

const (
	// fallbackFontSize holds the default font size in points when none is specified.
	fallbackFontSize = 16

	// fontWeightBold holds the numeric font weight value for bold text.
	fontWeightBold = 700

	// fontWeightLighter holds the numeric font weight value for lighter text.
	fontWeightLighter = 300

	// fontWeightNormal holds the numeric font weight value for normal text.
	fontWeightNormal = 400

	// fontWeightDigitMultiplier holds the base-10 multiplier
	// used when parsing numeric font weight strings.
	fontWeightDigitMultiplier = 10

	// fontWeightMin holds the minimum valid numeric font weight.
	fontWeightMin = 100

	// fontWeightMax holds the maximum valid numeric font weight.
	fontWeightMax = 900

	// baselineMiddleFactor holds the proportion of font size
	// used as the vertical offset for middle baseline
	// alignment.
	baselineMiddleFactor = 0.35

	// baselineHangFactor holds the proportion of font size
	// used as the vertical offset for hanging baseline
	// alignment.
	baselineHangFactor = 0.8

	// decorationThicknessFactor holds the proportion of font
	// size used to compute text decoration line thickness.
	decorationThicknessFactor = 0.05

	// decorationMinThickness holds the minimum text decoration line thickness in points.
	decorationMinThickness = 0.5

	// underlineOffsetFactor holds the proportion of font size
	// used as the vertical offset below the baseline for
	// underlines.
	underlineOffsetFactor = 0.15

	// overlineOffsetFactor holds the proportion of font size
	// used as the vertical offset above the baseline for
	// overlines.
	overlineOffsetFactor = 0.8

	// lineThroughOffsetFactor holds the proportion of font
	// size used as the vertical offset above the baseline
	// for line-through.
	lineThroughOffsetFactor = 0.3

	// gradientHalfDefault holds the default midpoint value for gradient coordinates.
	gradientHalfDefault = 0.5

	// percentDivisor holds the divisor used when converting percentage values to fractions.
	percentDivisor = 100
)

// renderText renders an SVG <text> element with optional <tspan> children.
//
// Takes rc (*renderContext) which provides the PDF stream and resource managers.
// Takes node (*Node) which holds the parsed text element.
// Takes style (*Style) which holds the resolved CSS style properties.
func renderText(rc *renderContext, node *Node, style *Style) {
	if rc.registerFont == nil {
		return
	}

	x := attrFloat(node, "x", 0)
	y := attrFloat(node, "y", 0)

	hasTspan := false
	for _, child := range node.Children {
		if child.Tag == "tspan" {
			hasTspan = true
			break
		}
	}

	rc.stream.SaveState()

	if hasTspan {
		renderTextWithTspan(rc, node, style, x, y)
	} else {
		text := node.Text
		if text == "" {
			var sb strings.Builder
			collectText(node, &sb)
			text = sb.String()
		}
		text = strings.TrimSpace(text)
		if text == "" {
			rc.stream.RestoreState()
			return
		}
		renderTextRun(rc, text, style, x, y)
	}

	rc.stream.RestoreState()
}

// renderTextRun renders a single text run at the specified
// position with the given style.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes text (string) which holds the text content to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
// Takes x (float64) which specifies the horizontal
// position.
// Takes y (float64) which specifies the vertical position.
func renderTextRun(rc *renderContext, text string, style *Style, x, y float64) {
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = fallbackFontSize
	}

	weight := parseFontWeight(style.FontWeight)
	fontStyle := parseFontStyle(style.FontStyle)

	fontName := rc.registerFont(style.FontFamily, weight, fontStyle, fontSize)
	if fontName == "" {
		return
	}

	textWidth := measureTextWidth(rc, style, weight, fontStyle, fontSize, text)

	x = applyTextAnchor(style, x, textWidth)

	y = applyBaselineAdjustment(style, y, fontSize)

	if style.Fill != nil {
		c := *style.Fill
		rc.stream.SetFillColourRGB(c.R, c.G, c.B)
	}

	rc.stream.ConcatMatrix(1, 0, 0, -1, 0, 2*y)

	rc.stream.BeginText()
	rc.stream.SetFont(fontName, fontSize)

	if style.LetterSpacing != 0 {
		rc.stream.SetCharSpacing(style.LetterSpacing)
	}
	if style.WordSpacing != 0 {
		rc.stream.SetWordSpacing(style.WordSpacing)
	}

	rc.stream.MoveText(x, y)
	rc.stream.ShowText(text)
	rc.stream.EndText()

	if style.TextDecoration != "" && style.TextDecoration != "none" && textWidth > 0 {
		renderTextDecoration(rc, style, x, y, textWidth, fontSize)
	}
}

// measureTextWidth measures the rendered width of a text
// string including letter spacing adjustments.
//
// Takes rc (*renderContext) which provides the text
// measurement callback.
// Takes style (*Style) which holds the resolved style
// properties including letter spacing.
// Takes weight (int) which specifies the numeric font
// weight.
// Takes fontStyle (int) which specifies the font style
// code.
// Takes fontSize (float64) which specifies the font size
// in points.
// Takes text (string) which holds the text to measure.
//
// Returns float64 which holds the computed text width.
func measureTextWidth(rc *renderContext, style *Style, weight, fontStyle int, fontSize float64, text string) float64 {
	if rc.measureText == nil {
		return 0
	}
	textWidth := rc.measureText(style.FontFamily, weight, fontStyle, fontSize, text)

	if style.LetterSpacing != 0 {
		runeCount := 0
		for range text {
			runeCount++
		}
		if runeCount > 1 {
			textWidth += style.LetterSpacing * float64(runeCount-1)
		}
	}
	return textWidth
}

// applyTextAnchor adjusts the x position based on the
// text-anchor property for middle or end alignment.
//
// Takes style (*Style) which holds the resolved style
// properties including text anchor.
// Takes x (float64) which specifies the initial horizontal
// position.
// Takes textWidth (float64) which specifies the rendered
// text width.
//
// Returns float64 which holds the adjusted x position.
func applyTextAnchor(style *Style, x, textWidth float64) float64 {
	if style.TextAnchor != "start" && textWidth > 0 {
		switch style.TextAnchor {
		case "middle":
			x -= textWidth / 2
		case "end":
			x -= textWidth
		}
	}
	return x
}

// applyBaselineAdjustment adjusts the y position based on
// the dominant-baseline property.
//
// Takes style (*Style) which holds the resolved style
// properties including dominant baseline.
// Takes y (float64) which specifies the initial vertical
// position.
// Takes fontSize (float64) which specifies the font size
// in points.
//
// Returns float64 which holds the adjusted y position.
func applyBaselineAdjustment(style *Style, y, fontSize float64) float64 {
	switch style.DominantBaseline {
	case "middle", "central":
		y += fontSize * baselineMiddleFactor
	case "hanging", "text-before-edge":
		y += fontSize * baselineHangFactor
	}
	return y
}

// renderTextWithTspan renders a text element that contains
// tspan children, advancing the cursor between runs.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the parent text element.
// Takes parentStyle (*Style) which holds the inherited
// style properties.
// Takes baseX (float64) which specifies the initial
// horizontal cursor position.
// Takes baseY (float64) which specifies the initial
// vertical cursor position.
func renderTextWithTspan(rc *renderContext, node *Node, parentStyle *Style, baseX, baseY float64) {
	curX := baseX
	curY := baseY

	for _, child := range node.Children {
		if child.Tag != "tspan" {
			curX = renderDirectTextContent(rc, child, node, parentStyle, curX, curY)
			continue
		}

		curX, curY = renderTspanChild(rc, child, parentStyle, curX, curY)
	}
}

// renderTspanChild renders a single tspan child element
// and returns the updated cursor position.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes child (*Node) which holds the tspan element to
// render.
// Takes parentStyle (*Style) which holds the inherited
// style properties.
// Takes curX (float64) which specifies the current
// horizontal cursor position.
// Takes curY (float64) which specifies the current
// vertical cursor position.
//
// Returns newX (float64) which holds the updated
// horizontal cursor position.
// Returns newY (float64) which holds the updated vertical
// cursor position.
func renderTspanChild(rc *renderContext, child *Node, parentStyle *Style, curX, curY float64) (newX float64, newY float64) {
	childStyle := ResolveStyle(child, parentStyle)

	if _, ok := child.Attrs["x"]; ok {
		curX = attrFloat(child, "x", curX)
	}
	if _, ok := child.Attrs["y"]; ok {
		curY = attrFloat(child, "y", curY)
	}
	curX += attrFloat(child, "dx", 0)
	curY += attrFloat(child, "dy", 0)

	text := resolveTspanText(child)
	if text == "" {
		return curX, curY
	}

	renderTextRun(rc, text, &childStyle, curX, curY)

	if rc.measureText != nil {
		curX += measureTspanWidth(rc, &childStyle, text)
	}

	return curX, curY
}

// resolveTspanText extracts and trims the text content
// from a tspan node, collecting from children if needed.
//
// Takes child (*Node) which holds the tspan element to
// extract text from.
//
// Returns string which holds the trimmed text content.
func resolveTspanText(child *Node) string {
	text := child.Text
	if text == "" {
		var sb strings.Builder
		collectText(child, &sb)
		text = sb.String()
	}
	return strings.TrimSpace(text)
}

// measureTspanWidth measures the rendered width of a tspan
// text run using the resolved style.
//
// Takes rc (*renderContext) which provides the text
// measurement callback.
// Takes style (*Style) which holds the resolved style
// properties.
// Takes text (string) which holds the text to measure.
//
// Returns float64 which holds the computed text width.
func measureTspanWidth(rc *renderContext, style *Style, text string) float64 {
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = fallbackFontSize
	}
	weight := parseFontWeight(style.FontWeight)
	fontStyle := parseFontStyle(style.FontStyle)
	return rc.measureText(style.FontFamily, weight, fontStyle, fontSize, text)
}

// renderDirectTextContent renders plain text content
// between tspan elements and returns the updated x cursor.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes child (*Node) which holds the text content node.
// Takes parent (*Node) which holds the parent text
// element.
// Takes parentStyle (*Style) which holds the inherited
// style properties.
// Takes curX (float64) which specifies the current
// horizontal cursor position.
// Takes curY (float64) which specifies the current
// vertical cursor position.
//
// Returns float64 which holds the updated horizontal
// cursor position.
func renderDirectTextContent(rc *renderContext, child, parent *Node, parentStyle *Style, curX, curY float64) float64 {
	text := strings.TrimSpace(child.Text)
	if text == "" && child.Tag == "" && parent.Text != "" {
		return curX
	}
	if text != "" {
		renderTextRun(rc, text, parentStyle, curX, curY)
		if rc.measureText != nil {
			fontSize := parentStyle.FontSize
			if fontSize <= 0 {
				fontSize = fallbackFontSize
			}
			weight := parseFontWeight(parentStyle.FontWeight)
			fontStyle := parseFontStyle(parentStyle.FontStyle)
			curX += rc.measureText(parentStyle.FontFamily, weight, fontStyle, fontSize, text)
		}
	}
	return curX
}

// collectText recursively gathers all text content from a
// node and its children into the string builder.
//
// Takes node (*Node) which holds the node to collect text
// from.
// Takes sb (*strings.Builder) which holds the builder to
// append text into.
func collectText(node *Node, sb *strings.Builder) {
	if node.Text != "" {
		sb.WriteString(node.Text)
	}
	for _, child := range node.Children {
		collectText(child, sb)
	}
}

// parseFontWeight converts an SVG font-weight string to
// its numeric value.
//
// Takes s (string) which holds the font-weight attribute
// value.
//
// Returns int which holds the numeric font weight.
func parseFontWeight(s string) int {
	switch s {
	case "bold", "bolder":
		return fontWeightBold
	case "lighter":
		return fontWeightLighter
	case styleValueNormal, "":
		return fontWeightNormal
	default:

		v := 0
		for _, ch := range s {
			if ch >= '0' && ch <= '9' {
				v = v*fontWeightDigitMultiplier + int(ch-'0')
			}
		}
		if v >= fontWeightMin && v <= fontWeightMax {
			return v
		}
		return fontWeightNormal
	}
}

// parseFontStyle converts an SVG font-style string to a
// numeric code, returning 1 for italic or oblique.
//
// Takes s (string) which holds the font-style attribute
// value.
//
// Returns int which holds the numeric font style code.
func parseFontStyle(s string) int {
	switch s {
	case "italic", "oblique":
		return 1
	default:
		return 0
	}
}

// renderTextDecoration draws underline, overline, and/or
// line-through lines for the given text run.
//
// The decoration value may contain multiple
// space-separated values (e.g. "underline line-through").
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// including text decoration and colour.
// Takes x (float64) which specifies the horizontal start
// position of the text run.
// Takes y (float64) which specifies the baseline vertical
// position of the text run.
// Takes textWidth (float64) which specifies the rendered
// width of the text run.
// Takes fontSize (float64) which specifies the font size
// used for offset calculations.
func renderTextDecoration(rc *renderContext, style *Style, x, y, textWidth, fontSize float64) {
	decoration := strings.ToLower(style.TextDecoration)

	if style.Stroke != nil && style.StrokeWidth > 0 {
		c := *style.Stroke
		rc.stream.SetStrokeColourRGB(c.R, c.G, c.B)
	} else if style.Fill != nil {
		c := *style.Fill
		rc.stream.SetStrokeColourRGB(c.R, c.G, c.B)
	}

	lineThickness := fontSize * decorationThicknessFactor
	if lineThickness < decorationMinThickness {
		lineThickness = decorationMinThickness
	}
	rc.stream.SetLineWidth(lineThickness)

	if strings.Contains(decoration, "underline") {
		decoY := y + fontSize*underlineOffsetFactor
		rc.stream.MoveTo(x, decoY)
		rc.stream.LineTo(x+textWidth, decoY)
		rc.stream.Stroke()
	}

	if strings.Contains(decoration, "overline") {
		decoY := y - fontSize*overlineOffsetFactor
		rc.stream.MoveTo(x, decoY)
		rc.stream.LineTo(x+textWidth, decoY)
		rc.stream.Stroke()
	}

	if strings.Contains(decoration, "line-through") {
		decoY := y - fontSize*lineThroughOffsetFactor
		rc.stream.MoveTo(x, decoY)
		rc.stream.LineTo(x+textWidth, decoY)
		rc.stream.Stroke()
	}
}
