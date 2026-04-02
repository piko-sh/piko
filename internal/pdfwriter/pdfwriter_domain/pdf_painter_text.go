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
	"context"
	"math"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

// paintImage renders a replaced element (img tag) into the PDF stream,
// attempting SVG vector rendering first before falling back to raster.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the replaced element box.
func (painter *PdfPainter) paintImage(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.Type != layouter_domain.BoxReplaced || box.SourceNode == nil {
		return
	}

	source := extractSrcAttribute(box)
	if source == "" {
		return
	}

	if painter.trySVGVector(ctx, stream, box, source) {
		return
	}

	if painter.imageData == nil {
		return
	}

	data, format, err := painter.imageData.GetImageData(ctx, source)
	if err != nil || len(data) == 0 {
		return
	}

	pixelWidth, pixelHeight := ExtractImageDimensions(data, format)
	if pixelWidth == 0 || pixelHeight == 0 {
		return
	}

	resourceName := painter.imageEmbedder.RegisterImage(source, data, format, pixelWidth, pixelHeight)
	painter.emitRasterImage(stream, box, resourceName, float64(pixelWidth), float64(pixelHeight))
}

// trySVGVector attempts native SVG vector rendering, returning true on
// success.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the target box.
//
// Returns bool which indicates whether SVG rendering succeeded.
func (painter *PdfPainter) trySVGVector(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox, source string) bool {
	if painter.svgWriter == nil || painter.svgData == nil {
		return false
	}
	svgXML, ok := painter.svgData.GetSVGData(ctx, source)
	if !ok {
		return false
	}
	return painter.paintSVGVector(ctx, stream, box, svgXML) == nil
}

// emitRasterImage renders a raster image with object-fit and object-position
// handling.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which provides the content
// area dimensions.
// Takes resourceName (string) which is the image XObject key.
// Takes intrinsicW, intrinsicH (float64) which are the image's natural
// pixel dimensions.
func (painter *PdfPainter) emitRasterImage(
	stream *ContentStream, box *layouter_domain.LayoutBox,
	resourceName string, intrinsicW, intrinsicH float64,
) {
	contentW := box.ContentWidth
	contentH := box.ContentHeight

	renderW, renderH := painter.resolveObjectFitSize(
		box.Style.ObjectFit, contentW, contentH, intrinsicW, intrinsicH)

	posXFrac, posYFrac := parseObjectPosition(box.Style.ObjectPosition)
	renderX := box.ContentX + (contentW-renderW)*posXFrac
	renderY := box.ContentY + (contentH-renderH)*posYFrac

	pdfX := renderX
	pdfY := painter.pageHeight + painter.pageYOffset - renderY - renderH

	needsClip := box.Style.ObjectFit == layouter_domain.ObjectFitCover ||
		box.Style.ObjectFit == layouter_domain.ObjectFitNone

	stream.SaveState()
	if needsClip {
		clipX := box.ContentX
		clipY := painter.pageHeight + painter.pageYOffset - box.ContentY - contentH
		stream.Rectangle(clipX, clipY, contentW, contentH)
		stream.ClipNonZero()
	}
	stream.ConcatMatrix(renderW, 0, 0, renderH, pdfX, pdfY)
	stream.PaintXObject(resourceName)
	stream.RestoreState()
}

// extractSrcAttribute returns the "src" attribute value from a
// replaced element's source node.
//
// Takes box (*layouter_domain.LayoutBox) which is the replaced element.
//
// Returns string which is the src attribute value, or empty if absent.
func extractSrcAttribute(box *layouter_domain.LayoutBox) string {
	if box.SourceNode == nil {
		return ""
	}
	for i := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[i].Name == "src" {
			return box.SourceNode.Attributes[i].Value
		}
	}
	return ""
}

// paintSVGVector renders an SVG as native PDF vector commands using
// the configured SVGWriterPort.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which provides the content
// area.
//
// Returns error if SVG rendering fails.
func (painter *PdfPainter) paintSVGVector(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox, svgXML string) error {
	contentX := box.ContentX
	contentY := box.ContentY
	contentW := box.ContentWidth
	contentH := box.ContentHeight

	pdfX := contentX
	pdfY := painter.pageHeight + painter.pageYOffset - contentY - contentH

	renderCtx := SVGRenderContext{
		Stream:           stream,
		ShadingManager:   painter.shadingManager,
		ExtGStateManager: painter.extGStateManager,
		FontEmbedder:     painter.fontEmbedder,
		ImageEmbedder:    painter.imageEmbedder,
		PageHeight:       painter.pageHeight,
		RegisterFont: func(family string, weight int, style int, _ float64) string {
			resolved := painter.resolveFontData(family, weight, style)
			if !resolved.found {
				return ""
			}
			key := fontInstanceKey(resolved.key)
			return painter.fontEmbedder.RegisterFont(resolved.data, key)
		},
	}

	return painter.svgWriter.RenderSVG(ctx, svgXML, renderCtx, pdfX, pdfY, contentW, contentH)
}

// paintTextShadows renders text shadow layers behind the text.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
func (painter *PdfPainter) paintTextShadows(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if (box.Type != layouter_domain.BoxTextRun && box.Type != layouter_domain.BoxListMarker) || box.Text == "" {
		return
	}
	if len(box.Style.TextShadow) == 0 {
		return
	}

	requested := pdfFontKey{family: box.Style.FontFamily, weight: box.Style.FontWeight, style: int(box.Style.FontStyle)}
	resolved := painter.resolveFontData(box.Style.FontFamily, box.Style.FontWeight, int(box.Style.FontStyle))

	for _, shadow := range box.Style.TextShadow {
		if shadow.BlurRadius <= 0 {
			painter.paintTextShadowPass(stream, box, textShadowPassParams{
				resolved: resolved, requested: requested,
				offsetX: shadow.OffsetX, offsetY: shadow.OffsetY,
				colour: shadow.Colour, alpha: shadow.Colour.Alpha,
			})
		} else {
			painter.paintBlurredTextShadow(stream, box, resolved, requested, shadow)
		}
	}
}

// paintBlurredTextShadow renders multiple passes to approximate blur.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
// Takes resolved (resolvedFont) which holds the matched font data.
// Takes requested (pdfFontKey) which holds the originally requested
// font key.
// Takes shadow (layouter_domain.TextShadowValue) which holds the
// shadow parameters.
func (painter *PdfPainter) paintBlurredTextShadow(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	resolved resolvedFont,
	requested pdfFontKey,
	shadow layouter_domain.TextShadowValue,
) {
	baseAlpha := shadow.Colour.Alpha / float64(textShadowBlurSteps)
	for i := range textShadowBlurSteps {
		spread := shadow.BlurRadius * float64(i+1) / float64(textShadowBlurSteps)
		for _, dx := range []float64{-spread, spread} {
			for _, dy := range []float64{-spread, spread} {
				painter.paintTextShadowPass(stream, box, textShadowPassParams{
					resolved: resolved, requested: requested,
					offsetX: shadow.OffsetX + dx, offsetY: shadow.OffsetY + dy,
					colour: shadow.Colour, alpha: baseAlpha / borderSideCount,
				})
			}
		}
	}
}

// textShadowPassParams groups the parameters for a single text shadow pass.
type textShadowPassParams struct {
	// requested holds the originally requested font key.
	requested pdfFontKey

	// resolved holds the matched font data and key.
	resolved resolvedFont

	// colour holds the shadow colour.
	colour layouter_domain.Colour

	// offsetX holds the horizontal shadow offset in points.
	offsetX float64

	// offsetY holds the vertical shadow offset in points.
	offsetY float64

	// alpha holds the shadow opacity in [0, 1].
	alpha float64
}

// paintTextShadowPass renders text once at the given offset with the
// specified colour and opacity, producing one shadow layer.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
// Takes p (textShadowPassParams) which holds the shadow rendering
// parameters.
func (painter *PdfPainter) paintTextShadowPass(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	p textShadowPassParams,
) {
	syntheticBold := needsSyntheticBold(p.requested, p.resolved.key)
	syntheticItalic := needsSyntheticItalic(p.requested, p.resolved.key)

	stream.SaveState()
	if p.alpha < 1.0 {
		name := painter.extGStateManager.RegisterOpacity(p.alpha)
		stream.SetExtGState(name)
	}

	pdfX := box.ContentX + p.offsetX
	pdfY := painter.pageHeight + painter.pageYOffset - (box.ContentY + p.offsetY) - resolveBaselineOffset(box)

	sp := textShadowRenderParams{
		syntheticBold:   syntheticBold,
		syntheticItalic: syntheticItalic,
		colour:          p.colour,
		pdfX:            pdfX,
		pdfY:            pdfY,
	}

	if box.Glyphs != nil && p.resolved.found {
		painter.paintShadowWithEmbeddedFont(stream, box, p.resolved, sp)
	} else {
		painter.paintShadowWithFallbackFont(stream, box, sp)
	}

	stream.RestoreState()
}

// textShadowRenderParams groups the rendering parameters for a text shadow pass.
type textShadowRenderParams struct {
	// colour holds the shadow fill and stroke colour.
	colour layouter_domain.Colour

	// pdfX holds the text X position in PDF coordinates.
	pdfX float64

	// pdfY holds the text Y position in PDF coordinates.
	pdfY float64

	// syntheticBold indicates whether synthetic bold stroking is needed.
	syntheticBold bool

	// syntheticItalic indicates whether synthetic italic skewing is needed.
	syntheticItalic bool
}

// paintShadowWithEmbeddedFont renders a shadow pass using an embedded font.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
// Takes resolved (resolvedFont) which holds the matched font data.
// Takes sp (textShadowRenderParams) which holds the rendering
// parameters.
func (painter *PdfPainter) paintShadowWithEmbeddedFont(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	resolved resolvedFont,
	sp textShadowRenderParams,
) {
	resourceName := painter.fontEmbedder.RegisterFont(resolved.data, fontInstanceKey(resolved.key))
	unitsPerEm := float64(FontUnitsPerEm(resolved.data))

	runes := []rune(box.Text)
	glyphIDs := make([]uint16, len(box.Glyphs))
	shapedAdvances := make([]float64, len(box.Glyphs))
	defaultAdvances := make([]float64, len(box.Glyphs))
	for index, glyph := range box.Glyphs {
		glyphIDs[index] = glyph.GlyphID
		shapedAdvances[index] = glyph.XAdvance
		defaultAdvances[index] = float64(painter.glyphAdvanceWidth(resolved.key, resolved.data, glyph.GlyphID)) * box.Style.FontSize / unitsPerEm
		clusterRuneIndex := glyph.ClusterIndex
		if clusterRuneIndex < 0 || clusterRuneIndex >= len(runes) {
			clusterRuneIndex = index
		}
		runeCount := max(glyph.RuneCount, 1)
		clusterEnd := min(clusterRuneIndex+runeCount, len(runes))
		clusterText := string(runes[clusterRuneIndex:clusterEnd])
		painter.fontEmbedder.RecordGlyph(resourceName, glyph.GlyphID, clusterText)
	}
	if painter.isVariableFont(resolved.key) && painter.glyphWidthFunc != nil {
		for _, glyph := range box.Glyphs {
			painter.fontEmbedder.RecordGlyphWidth(resourceName, glyph.GlyphID,
				painter.glyphWidthFunc(resolved.key.family, resolved.key.weight, resolved.key.style, glyph.GlyphID))
		}
	}

	stream.BeginText()
	stream.SetCharSpacing(box.Style.LetterSpacing)
	if sp.syntheticBold {
		stream.SetTextRenderingMode(int(layouter_domain.TextRenderFillStroke))
		stream.SetLineWidth(box.Style.FontSize * syntheticBoldStrokeRatio)
		painter.setStrokeColour(stream, sp.colour)
	}
	painter.setFillColour(stream, sp.colour)
	stream.SetFont(resourceName, box.Style.FontSize)
	if sp.syntheticItalic {
		stream.SetTextMatrix(1, 0, syntheticItalicSkew, 1, sp.pdfX, sp.pdfY)
	} else {
		stream.MoveText(sp.pdfX, sp.pdfY)
	}
	stream.ShowGlyphs(glyphIDs, shapedAdvances, defaultAdvances, box.Style.FontSize)
	stream.EndText()
}

// paintShadowWithFallbackFont renders a shadow pass using the Helvetica
// fallback font.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
// Takes sp (textShadowRenderParams) which holds the rendering
// parameters.
func (painter *PdfPainter) paintShadowWithFallbackFont(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	sp textShadowRenderParams,
) {
	stream.BeginText()
	stream.SetCharSpacing(box.Style.LetterSpacing)
	if sp.syntheticBold {
		stream.SetTextRenderingMode(int(layouter_domain.TextRenderFillStroke))
		stream.SetLineWidth(box.Style.FontSize * syntheticBoldStrokeRatio)
		painter.setStrokeColour(stream, sp.colour)
	}
	painter.setFillColour(stream, sp.colour)
	stream.SetFont("F1", box.Style.FontSize)
	if sp.syntheticItalic {
		stream.SetTextMatrix(1, 0, syntheticItalicSkew, 1, sp.pdfX, sp.pdfY)
	} else {
		stream.MoveText(sp.pdfX, sp.pdfY)
	}
	stream.ShowText(box.Text)
	stream.EndText()
}

// paintText renders a text run or list marker into the PDF stream.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
func (painter *PdfPainter) paintText(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if (box.Type != layouter_domain.BoxTextRun && box.Type != layouter_domain.BoxListMarker) || box.Text == "" {
		return
	}

	requested := pdfFontKey{family: box.Style.FontFamily, weight: box.Style.FontWeight, style: int(box.Style.FontStyle)}
	resolved := painter.resolveFontData(box.Style.FontFamily, box.Style.FontWeight, int(box.Style.FontStyle))

	if box.Glyphs != nil && resolved.found {
		painter.paintTextWithEmbeddedFont(stream, box, resolved, requested)
		return
	}

	syntheticBold := needsSyntheticBold(requested, resolved.key)
	syntheticItalic := needsSyntheticItalic(requested, resolved.key)

	stream.BeginText()
	painter.setFillColour(stream, box.Style.Colour)
	stream.SetFont("F1", box.Style.FontSize)
	stream.SetCharSpacing(box.Style.LetterSpacing)
	if syntheticBold && box.Style.TextRenderingMode == layouter_domain.TextRenderFill {
		stream.SetTextRenderingMode(int(layouter_domain.TextRenderFillStroke))
		stream.SetLineWidth(box.Style.FontSize * syntheticBoldStrokeRatio)
		painter.setStrokeColour(stream, box.Style.Colour)
	} else {
		stream.SetTextRenderingMode(int(box.Style.TextRenderingMode))
	}

	pdfX := box.ContentX
	pdfY := painter.pageHeight + painter.pageYOffset - box.ContentY - resolveBaselineOffset(box)

	if syntheticItalic {
		stream.SetTextMatrix(1, 0, syntheticItalicSkew, 1, pdfX, pdfY)
	} else {
		stream.MoveText(pdfX, pdfY)
	}
	stream.ShowText(box.Text)
	stream.EndText()
}

// paintTextWithEmbeddedFont renders a text run using an embedded
// TrueType or OpenType font with harfbuzz-shaped glyph positioning.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
// Takes resolved (resolvedFont) which holds the matched font data.
// Takes requested (pdfFontKey) which holds the originally requested
// font key.
func (painter *PdfPainter) paintTextWithEmbeddedFont(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	resolved resolvedFont,
	requested pdfFontKey,
) {
	syntheticBold := needsSyntheticBold(requested, resolved.key)
	syntheticItalic := needsSyntheticItalic(requested, resolved.key)

	resourceName := painter.fontEmbedder.RegisterFont(resolved.data, fontInstanceKey(resolved.key))
	unitsPerEm := float64(FontUnitsPerEm(resolved.data))

	runes := []rune(box.Text)
	glyphIDs := make([]uint16, len(box.Glyphs))
	shapedAdvances := make([]float64, len(box.Glyphs))
	defaultAdvances := make([]float64, len(box.Glyphs))
	wordSpacing := box.Style.WordSpacing
	for index, glyph := range box.Glyphs {
		glyphIDs[index] = glyph.GlyphID
		advance := glyph.XAdvance

		clusterRuneIndex := glyph.ClusterIndex
		if clusterRuneIndex < 0 || clusterRuneIndex >= len(runes) {
			clusterRuneIndex = index
		}

		if wordSpacing != 0 && clusterRuneIndex < len(runes) && runes[clusterRuneIndex] == ' ' {
			advance += wordSpacing
		}
		shapedAdvances[index] = advance
		defaultAdvances[index] = float64(painter.glyphAdvanceWidth(resolved.key, resolved.data, glyph.GlyphID)) * box.Style.FontSize / unitsPerEm

		runeCount := max(glyph.RuneCount, 1)
		clusterEnd := min(clusterRuneIndex+runeCount, len(runes))
		clusterText := string(runes[clusterRuneIndex:clusterEnd])
		painter.fontEmbedder.RecordGlyph(resourceName, glyph.GlyphID, clusterText)
	}
	if painter.isVariableFont(resolved.key) && painter.glyphWidthFunc != nil {
		for _, glyph := range box.Glyphs {
			painter.fontEmbedder.RecordGlyphWidth(resourceName, glyph.GlyphID,
				painter.glyphWidthFunc(resolved.key.family, resolved.key.weight, resolved.key.style, glyph.GlyphID))
		}
	}

	stream.BeginText()
	stream.SetCharSpacing(box.Style.LetterSpacing)
	painter.emitTextRenderingMode(stream, box, syntheticBold)
	painter.setFillColour(stream, box.Style.Colour)
	stream.SetFont(resourceName, box.Style.FontSize)

	pdfX := box.ContentX
	pdfY := painter.pageHeight + painter.pageYOffset - box.ContentY - resolveBaselineOffset(box)

	if syntheticItalic {
		stream.SetTextMatrix(1, 0, syntheticItalicSkew, 1, pdfX, pdfY)
	} else {
		stream.MoveText(pdfX, pdfY)
	}
	stream.ShowGlyphs(glyphIDs, shapedAdvances, defaultAdvances, box.Style.FontSize)
	stream.EndText()
}

// emitTextRenderingMode sets text rendering mode and stroke properties.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which supplies the text
// rendering style.
// Takes syntheticBold (bool) which indicates whether synthetic bold
// is needed.
func (painter *PdfPainter) emitTextRenderingMode(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	syntheticBold bool,
) {
	if syntheticBold && box.Style.TextRenderingMode == layouter_domain.TextRenderFill {
		stream.SetTextRenderingMode(int(layouter_domain.TextRenderFillStroke))
		stream.SetLineWidth(box.Style.FontSize * syntheticBoldStrokeRatio)
		painter.setStrokeColour(stream, box.Style.Colour)
		return
	}
	stream.SetTextRenderingMode(int(box.Style.TextRenderingMode))
	if box.Style.TextRenderingMode != layouter_domain.TextRenderFill {
		if box.Style.TextStrokeWidth > 0 {
			stream.SetLineWidth(box.Style.TextStrokeWidth)
			if box.Style.TextStrokeColour != (layouter_domain.Colour{}) {
				painter.setStrokeColour(stream, box.Style.TextStrokeColour)
			} else {
				painter.setStrokeColour(stream, box.Style.Colour)
			}
		}
	}
}

// paintTextDecorations draws underline, overline, and line-through.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the text run box.
func (painter *PdfPainter) paintTextDecorations(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if (box.Type != layouter_domain.BoxTextRun && box.Type != layouter_domain.BoxListMarker) || box.Text == "" {
		return
	}

	decoration := box.Style.TextDecoration
	if decoration == 0 {
		return
	}

	fontSize := box.Style.FontSize
	lineWidth := math.Max(fontSize/14.0, 1.0)
	decoStyle := box.Style.TextDecorationStyle

	stream.SaveState()
	if box.Style.TextDecorationColourSet {
		painter.setStrokeColour(stream, box.Style.TextDecorationColour)
	} else {
		painter.setStrokeColour(stream, box.Style.Colour)
	}
	stream.SetLineWidth(lineWidth)

	startX := box.ContentX
	endX := box.ContentX + box.ContentWidth

	if decoration&layouter_domain.TextDecorationUnderline != 0 {
		layoutY := box.ContentY + resolveBaselineOffset(box) + fontSize*textUnderlineOffset
		pdfY := painter.pageHeight + painter.pageYOffset - layoutY
		painter.drawDecorationLine(stream, startX, endX, pdfY, lineWidth, decoStyle)
	}

	if decoration&layouter_domain.TextDecorationLineThrough != 0 {
		layoutY := box.ContentY + fontSize*textLineThroughFraction
		pdfY := painter.pageHeight + painter.pageYOffset - layoutY
		painter.drawDecorationLine(stream, startX, endX, pdfY, lineWidth, decoStyle)
	}

	if decoration&layouter_domain.TextDecorationOverline != 0 {
		layoutY := box.ContentY
		pdfY := painter.pageHeight + painter.pageYOffset - layoutY
		painter.drawDecorationLine(stream, startX, endX, pdfY, lineWidth, decoStyle)
	}

	stream.RestoreState()
}

// drawDecorationLine draws a single decoration line between startX and
// endX at the given Y coordinate, using the specified decoration style.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes startX, endX (float64) which define the horizontal extent.
// Takes pdfY (float64) which is the vertical position in PDF
// coordinates.
// Takes lineWidth (float64) which is the stroke width in points.
// Takes style (layouter_domain.TextDecorationStyleType) which selects
// the line style.
func (*PdfPainter) drawDecorationLine(
	stream *ContentStream,
	startX, endX, pdfY, lineWidth float64,
	style layouter_domain.TextDecorationStyleType,
) {
	switch style {
	case layouter_domain.TextDecorationStyleDashed:
		stream.SaveState()
		stream.SetDashPattern([]float64{lineWidth * borderDoubleDivisor, lineWidth * borderDoubleDivisor}, 0)
		stream.MoveTo(startX, pdfY)
		stream.LineTo(endX, pdfY)
		stream.Stroke()
		stream.RestoreState()

	case layouter_domain.TextDecorationStyleDotted:
		stream.SaveState()
		stream.SetLineCap(1)
		stream.SetDashPattern([]float64{0, lineWidth * 2}, 0)
		stream.MoveTo(startX, pdfY)
		stream.LineTo(endX, pdfY)
		stream.Stroke()
		stream.RestoreState()

	case layouter_domain.TextDecorationStyleDouble:
		halfWidth := lineWidth / 2
		gap := lineWidth * decorationDoubleGapRatio
		stream.SaveState()
		stream.SetLineWidth(halfWidth)
		stream.MoveTo(startX, pdfY-gap/2)
		stream.LineTo(endX, pdfY-gap/2)
		stream.Stroke()
		stream.MoveTo(startX, pdfY+gap/2)
		stream.LineTo(endX, pdfY+gap/2)
		stream.Stroke()
		stream.RestoreState()

	case layouter_domain.TextDecorationStyleWavy:
		emitWavyLine(stream, startX, endX, pdfY, lineWidth*2, lineWidth*textShadowBlurSteps)

	default:
		stream.MoveTo(startX, pdfY)
		stream.LineTo(endX, pdfY)
		stream.Stroke()
	}
}

// emitWavyLine draws a sinusoidal wave between startX and endX at baseY
// using cubic Bezier curves.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes startX, endX (float64) which define the horizontal extent.
// Takes baseY (float64) which is the centre line Y coordinate.
// Takes amplitude (float64) which is the wave peak height in points.
// Takes wavelength (float64) which is the full wave length in points.
func emitWavyLine(stream *ContentStream, startX, endX, baseY, amplitude, wavelength float64) {
	halfWave := wavelength / 2
	x := startX
	up := true

	stream.MoveTo(x, baseY)
	for x < endX {
		nextX := math.Min(x+halfWave, endX)
		fraction := (nextX - x) / halfWave
		var peakY float64
		if up {
			peakY = baseY + amplitude*fraction
		} else {
			peakY = baseY - amplitude*fraction
		}
		cpX := x + (nextX-x)/2
		stream.CurveTo(cpX, peakY, cpX, peakY, nextX, baseY)
		x = nextX
		up = !up
	}
	stream.Stroke()
}
