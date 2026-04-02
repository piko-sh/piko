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
	"fmt"
	"math"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

// emitOverflowClip sets a clipping path to the box's padding box,
// respecting border-radius if present.
//
// Takes stream (*ContentStream) which receives the clipping operators.
// Takes box (*layouter_domain.LayoutBox) which provides the padding box geometry.
func (painter *PdfPainter) emitOverflowClip(stream *ContentStream, box *layouter_domain.LayoutBox) {
	paddingX := box.ContentX - box.Padding.Left
	paddingY := box.ContentY - box.Padding.Top
	paddingW := box.ContentWidth + box.Padding.Left + box.Padding.Right
	paddingH := box.ContentHeight + box.Padding.Top + box.Padding.Bottom

	clipX := paddingX
	clipY := painter.pageHeight + painter.pageYOffset - paddingY - paddingH

	if painter.hasAnyBorderRadius(box) {
		tlr := math.Max(0, box.Style.BorderTopLeftRadius-box.Border.Left)
		trr := math.Max(0, box.Style.BorderTopRightRadius-box.Border.Right)
		brr := math.Max(0, box.Style.BorderBottomRightRadius-box.Border.Right)
		blr := math.Max(0, box.Style.BorderBottomLeftRadius-box.Border.Left)
		emitRoundedRectPath(stream, clipX, clipY, paddingW, paddingH, tlr, trr, brr, blr)
	} else {
		stream.Rectangle(clipX, clipY, paddingW, paddingH)
	}
	stream.ClipNonZero()
}

// resolveTransformOrigin parses the CSS transform-origin value and
// returns the origin point in PDF coordinates.
//
// Defaults to the centre of the border box ("50% 50%"). Supports percentage
// values and the keywords left/centre/right/top/bottom.
//
// Takes box (*layouter_domain.LayoutBox) which provides the border box
// geometry and transform-origin style value.
//
// Returns originX (float64) which is the horizontal origin in PDF coordinates.
// Returns originY (float64) which is the vertical origin in PDF coordinates.
func (painter *PdfPainter) resolveTransformOrigin(box *layouter_domain.LayoutBox) (originX float64, originY float64) {
	borderX := box.BorderBoxX()
	borderY := box.BorderBoxY()
	borderW := box.BorderBoxWidth()
	borderH := box.BorderBoxHeight()

	oxFrac := originDefaultFraction
	oyFrac := originDefaultFraction

	origin := strings.TrimSpace(box.Style.TransformOrigin)
	if origin != "" {
		parts := strings.Fields(origin)
		if len(parts) >= 1 {
			oxFrac = parseOriginComponent(parts[0])
		}
		if len(parts) >= 2 {
			oyFrac = parseOriginComponent(parts[1])
		}
	}

	pdfX := borderX + borderW*oxFrac
	pdfY := painter.pageHeight + painter.pageYOffset - borderY - borderH*oyFrac

	return pdfX, pdfY
}

// parseOriginComponent parses a single transform-origin axis value.
//
// Takes s (string) which is the axis value to parse, accepting percentages
// ("50%") and keywords (left/centre/right/top/bottom).
//
// Returns float64 which is the fractional position from 0.0 to 1.0.
func parseOriginComponent(s string) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "left", "top":
		return 0
	case "right", "bottom":
		return 1
	case "center", "centre":
		return originDefaultFraction
	}
	if after, ok := strings.CutSuffix(s, percentSuffix); ok {
		v := parseNumber(after)
		return v / percentageDivisorFloat
	}
	return originDefaultFraction
}

// resolveObjectFitSize computes the rendered width and height of a
// replaced element given the CSS object-fit value, the content box
// dimensions, and the image's intrinsic dimensions.
//
// Takes fit (layouter_domain.ObjectFitType) which specifies the CSS
// object-fit mode.
// Takes contentW (float64) which is the content box width.
// Takes contentH (float64) which is the content box height.
// Takes intrinsicW (float64) which is the image's intrinsic width.
// Takes intrinsicH (float64) which is the image's intrinsic height.
//
// Returns renderW (float64) which is the computed render width.
// Returns renderH (float64) which is the computed render height.
func (*PdfPainter) resolveObjectFitSize(
	fit layouter_domain.ObjectFitType,
	contentW, contentH, intrinsicW, intrinsicH float64,
) (renderW float64, renderH float64) {
	switch fit {
	case layouter_domain.ObjectFitContain:
		scale := math.Min(contentW/intrinsicW, contentH/intrinsicH)
		return intrinsicW * scale, intrinsicH * scale

	case layouter_domain.ObjectFitCover:
		scale := math.Max(contentW/intrinsicW, contentH/intrinsicH)
		return intrinsicW * scale, intrinsicH * scale

	case layouter_domain.ObjectFitNone:
		return intrinsicW, intrinsicH

	case layouter_domain.ObjectFitScaleDown:
		containScale := math.Min(contentW/intrinsicW, contentH/intrinsicH)
		if containScale < 1 {
			return intrinsicW * containScale, intrinsicH * containScale
		}
		return intrinsicW, intrinsicH

	default:
		return contentW, contentH
	}
}

// parseObjectPosition parses a CSS object-position value and returns
// horizontal and vertical fractions (0-1).
//
// Takes value (string) which is the CSS object-position value to parse,
// defaulting to "50% 50%" when empty.
//
// Returns xFrac (float64) which is the horizontal fraction from 0.0 to 1.0.
// Returns yFrac (float64) which is the vertical fraction from 0.0 to 1.0.
func parseObjectPosition(value string) (xFrac float64, yFrac float64) {
	value = strings.TrimSpace(value)
	if value == "" {
		return originDefaultFraction, originDefaultFraction
	}
	parts := strings.Fields(value)
	xFrac = originDefaultFraction
	yFrac = originDefaultFraction
	if len(parts) >= 1 {
		xFrac = parseOriginComponent(parts[0])
	}
	if len(parts) >= 2 {
		yFrac = parseOriginComponent(parts[1])
	}
	return xFrac, yFrac
}

// pdfEscapeString escapes parentheses and backslashes in a string for
// use inside a PDF literal string delimited by parentheses.
//
// Takes s (string) which is the raw string to escape.
//
// Returns string which is the escaped string safe for PDF literal use.
func pdfEscapeString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '(', ')':
			b.WriteByte('\\')
			_, _ = b.WriteRune(r)
		case '\\':
			b.WriteString("\\\\")
		default:
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

// buildInfoDictionary builds the PDF info dictionary string, including
// metadata fields when set.
//
// Returns string which is the serialised PDF info dictionary.
func (painter *PdfPainter) buildInfoDictionary() string {
	var b strings.Builder
	b.WriteString("<< /Producer (Piko)")
	if painter.metadata != nil {
		if painter.metadata.Title != "" {
			fmt.Fprintf(&b, " /Title (%s)", pdfEscapeString(painter.metadata.Title))
		}
		if painter.metadata.Author != "" {
			fmt.Fprintf(&b, " /Author (%s)", pdfEscapeString(painter.metadata.Author))
		}
		if painter.metadata.Subject != "" {
			fmt.Fprintf(&b, " /Subject (%s)", pdfEscapeString(painter.metadata.Subject))
		}
		if painter.metadata.Keywords != "" {
			fmt.Fprintf(&b, " /Keywords (%s)", pdfEscapeString(painter.metadata.Keywords))
		}
		if painter.metadata.Creator != "" {
			fmt.Fprintf(&b, " /Creator (%s)", pdfEscapeString(painter.metadata.Creator))
		}
	}
	b.WriteString(pdfDictCloseSuffix)
	return b.String()
}

// collectLinkAnnotation checks whether the box originates from an <a>
// element with an href attribute. If so, it records a link annotation.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box to inspect.
func (painter *PdfPainter) collectLinkAnnotation(box *layouter_domain.LayoutBox) {
	if box.SourceNode == nil || box.SourceNode.TagName != "a" {
		return
	}
	href := ""
	for i := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[i].Name == "href" {
			href = box.SourceNode.Attributes[i].Value
			break
		}
	}
	if href == "" {
		return
	}

	pdfX := box.BorderBoxX()
	pdfBottom := painter.pageHeight + painter.pageYOffset - box.BorderBoxY() - box.BorderBoxHeight()
	pdfTop := painter.pageHeight + painter.pageYOffset - box.BorderBoxY()

	annot := pdfAnnotation{
		pageIndex: box.PageIndex,
		x1:        pdfX,
		y1:        pdfBottom,
		x2:        pdfX + box.BorderBoxWidth(),
		y2:        pdfTop,
	}

	if strings.HasPrefix(href, "#") {
		annot.dest = href[1:]
	} else {
		annot.uri = href
	}
	painter.annotations = append(painter.annotations, annot)
}

// collectNamedDestination checks whether the box has an id attribute.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box to inspect.
func (painter *PdfPainter) collectNamedDestination(box *layouter_domain.LayoutBox) {
	if box.SourceNode == nil {
		return
	}
	id := ""
	for i := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[i].Name == "id" {
			id = box.SourceNode.Attributes[i].Value
			break
		}
	}
	if id == "" {
		return
	}
	pdfY := painter.pageHeight + painter.pageYOffset - box.BorderBoxY()
	painter.namedDests = append(painter.namedDests, namedDestination{
		name:      id,
		pageIndex: box.PageIndex,
		y:         pdfY,
	})
}

// collectOutlineEntry checks whether the box originates from a heading element.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box to inspect.
func (painter *PdfPainter) collectOutlineEntry(box *layouter_domain.LayoutBox) {
	if box.SourceNode == nil {
		return
	}
	level := headingLevel(box.SourceNode.TagName)
	if level == 0 {
		return
	}

	title := extractTextContent(box)
	if title == "" {
		return
	}

	pdfY := painter.pageHeight + painter.pageYOffset - box.BorderBoxY()

	painter.outlineBuilder.AddEntry(OutlineEntry{
		Title:     title,
		Level:     level,
		PageIndex: box.PageIndex,
		YPosition: pdfY,
	})
}

// extractTextContent recursively collects all text content from a box.
//
// Takes box (*layouter_domain.LayoutBox) which is the root box to traverse.
//
// Returns string which is the concatenated text content.
func extractTextContent(box *layouter_domain.LayoutBox) string {
	if box.Text != "" {
		return box.Text
	}
	var b strings.Builder
	for _, child := range box.Children {
		b.WriteString(extractTextContent(child))
	}
	return b.String()
}

// paintOuterBoxShadows paints non-inset box shadows behind the box.
//
// Takes stream (*ContentStream) which receives the shadow drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides the box geometry
// and shadow style values.
func (painter *PdfPainter) paintOuterBoxShadows(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if len(box.Style.BoxShadow) == 0 {
		return
	}

	for _, shadow := range box.Style.BoxShadow {
		if shadow.Inset {
			continue
		}
		painter.paintSingleBoxShadow(stream, box, shadow)
	}
}

// paintInsetBoxShadows paints inset box shadows over the background.
//
// Takes stream (*ContentStream) which receives the shadow drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides the box geometry
// and shadow style values.
func (painter *PdfPainter) paintInsetBoxShadows(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if len(box.Style.BoxShadow) == 0 {
		return
	}

	for _, shadow := range box.Style.BoxShadow {
		if !shadow.Inset {
			continue
		}
		painter.paintSingleInsetBoxShadow(stream, box, shadow)
	}
}

// paintSingleBoxShadow paints a single outer box shadow.
//
// Takes stream (*ContentStream) which receives the shadow drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides the box geometry.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// offset, spread, blur, and colour.
func (painter *PdfPainter) paintSingleBoxShadow(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	shadow layouter_domain.BoxShadowValue,
) {
	borderX := box.BorderBoxX()
	borderY := box.BorderBoxY()
	borderW := box.BorderBoxWidth()
	borderH := box.BorderBoxHeight()
	pdfBorderY := painter.pageHeight + painter.pageYOffset - borderY - borderH

	shadowX := borderX + shadow.OffsetX - shadow.SpreadRadius
	shadowY := borderY + shadow.OffsetY - shadow.SpreadRadius
	shadowW := borderW + 2*shadow.SpreadRadius
	shadowH := borderH + 2*shadow.SpreadRadius

	hasRadius := painter.hasAnyBorderRadius(box)

	sp := shadowParams{
		shadowX: shadowX, shadowY: shadowY, shadowW: shadowW, shadowH: shadowH,
		borderX: borderX, pdfBorderY: pdfBorderY, borderW: borderW, borderH: borderH,
		hasRadius: hasRadius,
	}

	if shadow.BlurRadius <= 0 {
		painter.paintSharpOuterShadow(stream, box, shadow, sp)
		return
	}

	painter.paintBlurredOuterShadow(stream, box, shadow, sp)
}

// shadowParams groups the geometry for outer box shadow painting.
type shadowParams struct {
	// shadowX holds the shadow rectangle left edge in CSS coordinates.
	shadowX float64

	// shadowY holds the shadow rectangle top edge in CSS coordinates.
	shadowY float64

	// shadowW holds the shadow rectangle width.
	shadowW float64

	// shadowH holds the shadow rectangle height.
	shadowH float64

	// borderX holds the border box left edge in CSS coordinates.
	borderX float64

	// pdfBorderY holds the border box bottom edge in PDF coordinates.
	pdfBorderY float64

	// borderW holds the border box width.
	borderW float64

	// borderH holds the border box height.
	borderH float64

	// hasRadius indicates whether the box has any border-radius.
	hasRadius bool
}

// paintSharpOuterShadow paints a sharp (no blur) outer box shadow.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides border-radius values.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// colour and spread.
// Takes sp (shadowParams) which holds the precomputed shadow geometry.
func (painter *PdfPainter) paintSharpOuterShadow(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	shadow layouter_domain.BoxShadowValue,
	sp shadowParams,
) {
	stream.SaveState()
	painter.setFillColour(stream, shadow.Colour)
	if shadow.Colour.Alpha < 1.0 {
		name := painter.extGStateManager.RegisterOpacity(shadow.Colour.Alpha)
		stream.SetExtGState(name)
	}
	pdfY := painter.pageHeight + painter.pageYOffset - sp.shadowY - sp.shadowH
	if sp.hasRadius {
		tlr := math.Max(0, box.Style.BorderTopLeftRadius+shadow.SpreadRadius)
		trr := math.Max(0, box.Style.BorderTopRightRadius+shadow.SpreadRadius)
		brr := math.Max(0, box.Style.BorderBottomRightRadius+shadow.SpreadRadius)
		blr := math.Max(0, box.Style.BorderBottomLeftRadius+shadow.SpreadRadius)
		emitRoundedRectPath(stream, sp.shadowX, pdfY, sp.shadowW, sp.shadowH, tlr, trr, brr, blr)
	} else {
		stream.Rectangle(sp.shadowX, pdfY, sp.shadowW, sp.shadowH)
	}
	painter.emitBorderBoxCutout(stream, box, sp.borderX, sp.pdfBorderY, sp.borderW, sp.borderH, sp.hasRadius)
	stream.FillEvenOdd()
	stream.RestoreState()
}

// paintBlurredOuterShadow paints a blurred outer box shadow.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides border-radius values.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// colour, blur, and spread.
// Takes sp (shadowParams) which holds the precomputed shadow geometry.
func (painter *PdfPainter) paintBlurredOuterShadow(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	shadow layouter_domain.BoxShadowValue,
	sp shadowParams,
) {
	baseAlpha := shadow.Colour.Alpha / float64(shadowBlurSteps)
	blur := shadow.BlurRadius

	for i := shadowBlurSteps - 1; i >= 0; i-- {
		expand := blur * float64(i+1) / float64(shadowBlurSteps)
		layerX := sp.shadowX - expand
		layerY := sp.shadowY - expand
		layerW := sp.shadowW + 2*expand
		layerH := sp.shadowH + 2*expand
		pdfY := painter.pageHeight + painter.pageYOffset - layerY - layerH

		stream.SaveState()
		painter.setFillColour(stream, shadow.Colour)
		name := painter.extGStateManager.RegisterOpacity(baseAlpha)
		stream.SetExtGState(name)

		if sp.hasRadius {
			tlr := math.Max(0, box.Style.BorderTopLeftRadius+shadow.SpreadRadius+expand)
			trr := math.Max(0, box.Style.BorderTopRightRadius+shadow.SpreadRadius+expand)
			brr := math.Max(0, box.Style.BorderBottomRightRadius+shadow.SpreadRadius+expand)
			blr := math.Max(0, box.Style.BorderBottomLeftRadius+shadow.SpreadRadius+expand)
			emitRoundedRectPath(stream, layerX, pdfY, layerW, layerH, tlr, trr, brr, blr)
		} else {
			stream.Rectangle(layerX, pdfY, layerW, layerH)
		}
		painter.emitBorderBoxCutout(stream, box, sp.borderX, sp.pdfBorderY, sp.borderW, sp.borderH, sp.hasRadius)
		stream.FillEvenOdd()
		stream.RestoreState()
	}
}

// emitBorderBoxCutout adds a cutout path for the border box to the
// current path, used with even-odd fill to clip shadows.
//
// Takes stream (*ContentStream) which receives the cutout path operators.
// Takes box (*layouter_domain.LayoutBox) which provides border-radius values.
// Takes borderX (float64) which is the border box left edge in PDF coordinates.
// Takes pdfBorderY (float64) which is the border box bottom edge in PDF coordinates.
// Takes borderW (float64) which is the border box width.
// Takes borderH (float64) which is the border box height.
// Takes hasRadius (bool) which indicates whether rounded corners apply.
func (*PdfPainter) emitBorderBoxCutout(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	borderX, pdfBorderY, borderW, borderH float64,
	hasRadius bool,
) {
	if hasRadius {
		emitRoundedRectPath(stream, borderX, pdfBorderY, borderW, borderH,
			box.Style.BorderTopLeftRadius,
			box.Style.BorderTopRightRadius,
			box.Style.BorderBottomRightRadius,
			box.Style.BorderBottomLeftRadius)
	} else {
		stream.Rectangle(borderX, pdfBorderY, borderW, borderH)
	}
}

// paintSingleInsetBoxShadow paints a single inset box shadow clipped
// to the padding box.
//
// Takes stream (*ContentStream) which receives the shadow drawing operators.
// Takes box (*layouter_domain.LayoutBox) which provides the padding box
// geometry and border-radius values.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// offset, spread, blur, and colour.
func (painter *PdfPainter) paintSingleInsetBoxShadow(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	shadow layouter_domain.BoxShadowValue,
) {
	paddingX := box.ContentX - box.Padding.Left
	paddingY := box.ContentY - box.Padding.Top
	paddingW := box.ContentWidth + box.Padding.Left + box.Padding.Right
	paddingH := box.ContentHeight + box.Padding.Top + box.Padding.Bottom

	stream.SaveState()
	clipPdfY := painter.pageHeight + painter.pageYOffset - paddingY - paddingH
	hasRadius := painter.hasAnyBorderRadius(box)
	if hasRadius {
		tlr := math.Max(0, box.Style.BorderTopLeftRadius-box.Border.Left)
		trr := math.Max(0, box.Style.BorderTopRightRadius-box.Border.Right)
		brr := math.Max(0, box.Style.BorderBottomRightRadius-box.Border.Right)
		blr := math.Max(0, box.Style.BorderBottomLeftRadius-box.Border.Left)
		emitRoundedRectPath(stream, paddingX, clipPdfY, paddingW, paddingH, tlr, trr, brr, blr)
	} else {
		stream.Rectangle(paddingX, clipPdfY, paddingW, paddingH)
	}
	stream.ClipNonZero()

	shadowX := paddingX + shadow.OffsetX + shadow.SpreadRadius
	shadowY := paddingY + shadow.OffsetY + shadow.SpreadRadius
	shadowW := paddingW - 2*shadow.SpreadRadius
	shadowH := paddingH - 2*shadow.SpreadRadius

	if shadowW <= 0 || shadowH <= 0 {
		painter.setFillColour(stream, shadow.Colour)
		if shadow.Colour.Alpha < 1.0 {
			name := painter.extGStateManager.RegisterOpacity(shadow.Colour.Alpha)
			stream.SetExtGState(name)
		}
		stream.Rectangle(paddingX, clipPdfY, paddingW, paddingH)
		stream.Fill()
		stream.RestoreState()
		return
	}

	isp := insetShadowParams{
		paddingX: paddingX, paddingY: paddingY, paddingW: paddingW, paddingH: paddingH,
		shadowX: shadowX, shadowY: shadowY, shadowW: shadowW, shadowH: shadowH,
	}

	if shadow.BlurRadius <= 0 {
		painter.paintSharpInsetShadow(stream, shadow, isp)
		stream.RestoreState()
		return
	}

	painter.paintBlurredInsetShadow(stream, shadow, isp)
	stream.RestoreState()
}

// insetShadowParams groups the geometry for inset box shadow painting.
type insetShadowParams struct {
	// paddingX holds the padding box left edge in CSS coordinates.
	paddingX float64

	// paddingY holds the padding box top edge in CSS coordinates.
	paddingY float64

	// paddingW holds the padding box width.
	paddingW float64

	// paddingH holds the padding box height.
	paddingH float64

	// shadowX holds the inset shadow rectangle left edge in CSS coordinates.
	shadowX float64

	// shadowY holds the inset shadow rectangle top edge in CSS coordinates.
	shadowY float64

	// shadowW holds the inset shadow rectangle width.
	shadowW float64

	// shadowH holds the inset shadow rectangle height.
	shadowH float64
}

// paintSharpInsetShadow paints a sharp inset shadow.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// colour and alpha.
// Takes p (insetShadowParams) which holds the precomputed inset geometry.
func (painter *PdfPainter) paintSharpInsetShadow(
	stream *ContentStream,
	shadow layouter_domain.BoxShadowValue,
	p insetShadowParams,
) {
	painter.setFillColour(stream, shadow.Colour)
	if shadow.Colour.Alpha < 1.0 {
		name := painter.extGStateManager.RegisterOpacity(shadow.Colour.Alpha)
		stream.SetExtGState(name)
	}
	outerPdfY := painter.pageHeight + painter.pageYOffset - p.paddingY - p.paddingH
	stream.Rectangle(p.paddingX, outerPdfY, p.paddingW, p.paddingH)
	innerPdfY := painter.pageHeight + painter.pageYOffset - p.shadowY - p.shadowH

	stream.MoveTo(p.shadowX, innerPdfY)
	stream.LineTo(p.shadowX, innerPdfY+p.shadowH)
	stream.LineTo(p.shadowX+p.shadowW, innerPdfY+p.shadowH)
	stream.LineTo(p.shadowX+p.shadowW, innerPdfY)
	stream.ClosePath()
	stream.Fill()
}

// paintBlurredInsetShadow paints a blurred inset shadow.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes shadow (layouter_domain.BoxShadowValue) which specifies the shadow
// colour, blur radius, and alpha.
// Takes p (insetShadowParams) which holds the precomputed inset geometry.
func (painter *PdfPainter) paintBlurredInsetShadow(
	stream *ContentStream,
	shadow layouter_domain.BoxShadowValue,
	p insetShadowParams,
) {
	baseAlpha := shadow.Colour.Alpha / float64(shadowBlurSteps)
	blur := shadow.BlurRadius

	for i := range shadowBlurSteps {
		shrink := blur * float64(i+1) / float64(shadowBlurSteps)
		layerX := p.shadowX + shrink
		layerY := p.shadowY + shrink
		layerW := p.shadowW - 2*shrink
		layerH := p.shadowH - 2*shrink
		if layerW <= 0 || layerH <= 0 {
			continue
		}

		stream.SaveState()
		painter.setFillColour(stream, shadow.Colour)
		name := painter.extGStateManager.RegisterOpacity(baseAlpha)
		stream.SetExtGState(name)

		outerPdfY := painter.pageHeight + painter.pageYOffset - p.paddingY - p.paddingH
		stream.Rectangle(p.paddingX, outerPdfY, p.paddingW, p.paddingH)
		innerPdfY := painter.pageHeight + painter.pageYOffset - layerY - layerH
		stream.MoveTo(layerX, innerPdfY)
		stream.LineTo(layerX, innerPdfY+layerH)
		stream.LineTo(layerX+layerW, innerPdfY+layerH)
		stream.LineTo(layerX+layerW, innerPdfY)
		stream.ClosePath()
		stream.Fill()
		stream.RestoreState()
	}
}
