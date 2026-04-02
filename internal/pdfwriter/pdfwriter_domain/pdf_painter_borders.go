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

// paintBorders draws all CSS border sides for the given layout box.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds
// the layout box whose borders are drawn.
func (painter *PdfPainter) paintBorders(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if painter.hasAnyBorderRadius(box) {
		painter.paintBordersRounded(stream, box)
		return
	}
	painter.paintBordersStraight(stream, box)
}

// straightBorderSide holds the data for a single straight border side.
type straightBorderSide struct {
	// colour holds the border colour for this side.
	colour layouter_domain.Colour

	// style holds the border style type (solid, dashed, dotted, etc.).
	style layouter_domain.BorderStyleType

	// width holds the border width in points.
	width float64

	// startX holds the starting X coordinate of the border line.
	startX float64

	// startY holds the starting Y coordinate of the border line.
	startY float64

	// endX holds the ending X coordinate of the border line.
	endX float64

	// endY holds the ending Y coordinate of the border line.
	endY float64

	// outerOffsetX holds the X direction multiplier for inset from the outer edge.
	outerOffsetX float64

	// outerOffsetY holds the Y direction multiplier for inset from the outer edge.
	outerOffsetY float64

	// adjacentStart holds the width of the adjacent border at the start of this side.
	adjacentStart float64

	// adjacentEnd holds the width of the adjacent border at the end of this side.
	adjacentEnd float64

	// isTopOrLeft indicates whether this is a top or left border side.
	isTopOrLeft bool
}

// paintBordersStraight draws straight-line borders (no border-radius).
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds
// the layout box whose borders are drawn.
func (painter *PdfPainter) paintBordersStraight(stream *ContentStream, box *layouter_domain.LayoutBox) {
	borderBoxX := box.BorderBoxX()
	borderBoxY := box.BorderBoxY()
	borderBoxWidth := box.BorderBoxWidth()
	borderBoxHeight := box.BorderBoxHeight()

	pdfLeft := borderBoxX
	pdfBottom := painter.pageHeight + painter.pageYOffset - borderBoxY - borderBoxHeight
	pdfRight := borderBoxX + borderBoxWidth
	pdfTop := painter.pageHeight + painter.pageYOffset - borderBoxY

	sides := [borderSideCount]straightBorderSide{
		{
			width: box.Border.Top, colour: box.Style.BorderTopColour, style: box.Style.BorderTopStyle,
			startX: pdfLeft, startY: pdfTop, endX: pdfRight, endY: pdfTop,
			outerOffsetX: 0, outerOffsetY: -1, isTopOrLeft: true,
			adjacentStart: box.Border.Left, adjacentEnd: box.Border.Right,
		},
		{
			width: box.Border.Bottom, colour: box.Style.BorderBottomColour, style: box.Style.BorderBottomStyle,
			startX: pdfLeft, startY: pdfBottom, endX: pdfRight, endY: pdfBottom,
			outerOffsetX: 0, outerOffsetY: 1, isTopOrLeft: false,
			adjacentStart: box.Border.Left, adjacentEnd: box.Border.Right,
		},
		{
			width: box.Border.Left, colour: box.Style.BorderLeftColour, style: box.Style.BorderLeftStyle,
			startX: pdfLeft, startY: pdfBottom, endX: pdfLeft, endY: pdfTop,
			outerOffsetX: 1, outerOffsetY: 0, isTopOrLeft: true,
			adjacentStart: box.Border.Bottom, adjacentEnd: box.Border.Top,
		},
		{
			width: box.Border.Right, colour: box.Style.BorderRightColour, style: box.Style.BorderRightStyle,
			startX: pdfRight, startY: pdfBottom, endX: pdfRight, endY: pdfTop,
			outerOffsetX: -1, outerOffsetY: 0, isTopOrLeft: false,
			adjacentStart: box.Border.Bottom, adjacentEnd: box.Border.Top,
		},
	}

	for i := range sides {
		painter.paintStraightBorderSide(stream, &sides[i]) //nolint:gosec // loop bounded by array size
	}
}

// paintStraightBorderSide renders a single straight border side.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes side (*straightBorderSide) which holds the geometry and style for the border side.
func (painter *PdfPainter) paintStraightBorderSide(stream *ContentStream, side *straightBorderSide) {
	if side.width <= 0 || side.style == layouter_domain.BorderStyleNone {
		return
	}

	if side.style == layouter_domain.BorderStyleDouble && side.width >= borderDoubleMinWidth {
		painter.paintDoubleBorderSide(stream, side)
		return
	}

	if side.style == layouter_domain.BorderStyleGroove || side.style == layouter_domain.BorderStyleRidge {
		painter.paint3DBorderSide(stream, side)
		return
	}

	if side.style == layouter_domain.BorderStyleInset || side.style == layouter_domain.BorderStyleOutset {
		painter.paintInsetOutsetBorderSide(stream, side)
		return
	}

	stream.SaveState()
	painter.setStrokeColour(stream, side.colour)
	stream.SetLineWidth(side.width)
	painter.applyBorderDashPattern(stream, side.style, side.width)
	midOffset := side.width / 2
	stream.MoveTo(side.startX+side.outerOffsetX*midOffset, side.startY+side.outerOffsetY*midOffset)
	stream.LineTo(side.endX+side.outerOffsetX*midOffset, side.endY+side.outerOffsetY*midOffset)
	stream.Stroke()
	stream.RestoreState()
}

// paintInsetOutsetBorderSide renders an inset or outset border side.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes side (*straightBorderSide) which holds the geometry and style for the border side.
func (painter *PdfPainter) paintInsetOutsetBorderSide(stream *ContentStream, side *straightBorderSide) {
	useDark := (side.style == layouter_domain.BorderStyleInset) == side.isTopOrLeft
	colour := side.colour
	if useDark {
		colour = darkenColour(side.colour, darkenFactor)
	} else {
		colour = lightenColour(side.colour, lightenFactor)
	}
	stream.SaveState()
	painter.setStrokeColour(stream, colour)
	stream.SetLineWidth(side.width)
	midOffset := side.width / 2
	stream.MoveTo(side.startX+side.outerOffsetX*midOffset, side.startY+side.outerOffsetY*midOffset)
	stream.LineTo(side.endX+side.outerOffsetX*midOffset, side.endY+side.outerOffsetY*midOffset)
	stream.Stroke()
	stream.RestoreState()
}

// paintDoubleBorderSide draws a CSS double border for a single side:
// two lines of 1/3 total width separated by a 1/3 gap.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes side (*straightBorderSide) which holds the geometry and style for the border side.
func (painter *PdfPainter) paintDoubleBorderSide(stream *ContentStream, side *straightBorderSide) {
	lineW := side.width / borderDoubleDivisor

	outerOffset := lineW / 2
	stream.SaveState()
	painter.setStrokeColour(stream, side.colour)
	stream.SetLineWidth(lineW)
	stream.MoveTo(side.startX+side.outerOffsetX*outerOffset, side.startY+side.outerOffsetY*outerOffset)
	stream.LineTo(side.endX+side.outerOffsetX*outerOffset, side.endY+side.outerOffsetY*outerOffset)
	stream.Stroke()
	stream.RestoreState()

	innerOffset := side.width - lineW/2
	innerSx := side.startX + side.outerOffsetX*innerOffset
	innerSy := side.startY + side.outerOffsetY*innerOffset
	innerEx := side.endX + side.outerOffsetX*innerOffset
	innerEy := side.endY + side.outerOffsetY*innerOffset

	if side.outerOffsetY != 0 {
		innerSx += 2 * side.adjacentStart / borderDoubleDivisor
		innerEx -= 2 * side.adjacentEnd / borderDoubleDivisor
	} else {
		innerSy += 2 * side.adjacentStart / borderDoubleDivisor
		innerEy -= 2 * side.adjacentEnd / borderDoubleDivisor
	}

	stream.SaveState()
	painter.setStrokeColour(stream, side.colour)
	stream.SetLineWidth(lineW)
	stream.MoveTo(innerSx, innerSy)
	stream.LineTo(innerEx, innerEy)
	stream.Stroke()
	stream.RestoreState()
}

// paint3DBorderSide draws a groove or ridge border for a single side.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes side (*straightBorderSide) which holds the geometry and style for the border side.
func (painter *PdfPainter) paint3DBorderSide(stream *ContentStream, side *straightBorderSide) {
	dark := darkenColour(side.colour, darkenFactor)
	light := lightenColour(side.colour, lightenFactor)

	var outerColour, innerColour layouter_domain.Colour
	grooveDarkOuter := side.isTopOrLeft
	if side.style == layouter_domain.BorderStyleRidge {
		grooveDarkOuter = !grooveDarkOuter
	}
	if grooveDarkOuter {
		outerColour = dark
		innerColour = light
	} else {
		outerColour = light
		innerColour = dark
	}

	halfW := side.width / 2

	outerOffset := halfW / 2
	stream.SaveState()
	painter.setStrokeColour(stream, outerColour)
	stream.SetLineWidth(halfW)
	stream.MoveTo(side.startX+side.outerOffsetX*outerOffset, side.startY+side.outerOffsetY*outerOffset)
	stream.LineTo(side.endX+side.outerOffsetX*outerOffset, side.endY+side.outerOffsetY*outerOffset)
	stream.Stroke()
	stream.RestoreState()

	innerOffset := side.width - halfW/2
	stream.SaveState()
	painter.setStrokeColour(stream, innerColour)
	stream.SetLineWidth(halfW)
	stream.MoveTo(side.startX+side.outerOffsetX*innerOffset, side.startY+side.outerOffsetY*innerOffset)
	stream.LineTo(side.endX+side.outerOffsetX*innerOffset, side.endY+side.outerOffsetY*innerOffset)
	stream.Stroke()
	stream.RestoreState()
}

// applyBorderDashPattern sets the dash pattern for a
// border style.
//
// Takes stream (*ContentStream) which holds the PDF
// content stream to write to.
// Takes style (layouter_domain.BorderStyleType) which
// specifies the border style (dashed, dotted, etc.).
// Takes width (float64) which specifies the border width
// used to compute dash lengths.
func (*PdfPainter) applyBorderDashPattern(stream *ContentStream, style layouter_domain.BorderStyleType, width float64) {
	switch style {
	case layouter_domain.BorderStyleDashed:
		stream.SetDashPattern([]float64{width * borderDoubleDivisor, width * borderDoubleDivisor}, 0)
	case layouter_domain.BorderStyleDotted:
		stream.SetLineCap(1)
		stream.SetDashPattern([]float64{0, width * 2}, 0)
	default:
	}
}

// borderCornerRadii holds the four corner radii for rounded borders.
type borderCornerRadii struct {
	// topLeft holds the top-left corner radius.
	topLeft float64

	// topRight holds the top-right corner radius.
	topRight float64

	// bottomRight holds the bottom-right corner radius.
	bottomRight float64

	// bottomLeft holds the bottom-left corner radius.
	bottomLeft float64
}

// roundedBorderSide holds the data for a single rounded border side.
type roundedBorderSide struct {
	// colour holds the border colour for this side.
	colour layouter_domain.Colour

	// style holds the border style type (solid, dashed, dotted, etc.).
	style layouter_domain.BorderStyleType

	// width holds the border width in points.
	width float64

	// clipX holds the X coordinates of the quadrant clipping polygon.
	clipX [borderSideCount]float64

	// clipY holds the Y coordinates of the quadrant clipping polygon.
	clipY [borderSideCount]float64
}

// isUniformBorder reports whether all four border sides have the same
// width, colour, and style.
//
// Takes box (*layouter_domain.LayoutBox) which holds the layout box to inspect.
//
// Returns bool indicating whether all four sides are uniform.
func isUniformBorder(box *layouter_domain.LayoutBox) bool {
	return box.Border.Top == box.Border.Right &&
		box.Border.Right == box.Border.Bottom &&
		box.Border.Bottom == box.Border.Left &&
		box.Style.BorderTopColour == box.Style.BorderRightColour &&
		box.Style.BorderRightColour == box.Style.BorderBottomColour &&
		box.Style.BorderBottomColour == box.Style.BorderLeftColour &&
		box.Style.BorderTopStyle == box.Style.BorderRightStyle &&
		box.Style.BorderRightStyle == box.Style.BorderBottomStyle &&
		box.Style.BorderBottomStyle == box.Style.BorderLeftStyle
}

// paintBordersRounded draws borders with rounded corners. Uses a
// quadrant-clipping approach for non-uniform border widths or colours.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds
// the layout box whose borders are drawn.
func (painter *PdfPainter) paintBordersRounded(stream *ContentStream, box *layouter_domain.LayoutBox) {
	borderBoxX := box.BorderBoxX()
	borderBoxY := box.BorderBoxY()
	borderBoxWidth := box.BorderBoxWidth()
	borderBoxHeight := box.BorderBoxHeight()

	pdfX := borderBoxX
	pdfY := painter.pageHeight + painter.pageYOffset - borderBoxY - borderBoxHeight

	radii := borderCornerRadii{
		topLeft:     box.Style.BorderTopLeftRadius,
		topRight:    box.Style.BorderTopRightRadius,
		bottomRight: box.Style.BorderBottomRightRadius,
		bottomLeft:  box.Style.BorderBottomLeftRadius,
	}

	if isUniformBorder(box) && box.Border.Top > 0 && box.Style.BorderTopStyle != layouter_domain.BorderStyleNone {
		painter.paintUniformRoundedBorder(stream, box, pdfX, pdfY, borderBoxWidth, borderBoxHeight, radii)
		return
	}

	cx := pdfX + borderBoxWidth/2
	cy := pdfY + borderBoxHeight/2

	sides := [borderSideCount]roundedBorderSide{
		{
			width: box.Border.Top, colour: box.Style.BorderTopColour, style: box.Style.BorderTopStyle,
			clipX: [borderSideCount]float64{pdfX, pdfX + borderBoxWidth, cx + borderBoxWidth/2, cx - borderBoxWidth/2},
			clipY: [borderSideCount]float64{pdfY + borderBoxHeight, pdfY + borderBoxHeight, cy, cy},
		},
		{
			width: box.Border.Right, colour: box.Style.BorderRightColour, style: box.Style.BorderRightStyle,
			clipX: [borderSideCount]float64{pdfX + borderBoxWidth, pdfX + borderBoxWidth, cx, cx},
			clipY: [borderSideCount]float64{pdfY + borderBoxHeight, pdfY, cy, cy},
		},
		{
			width: box.Border.Bottom, colour: box.Style.BorderBottomColour, style: box.Style.BorderBottomStyle,
			clipX: [borderSideCount]float64{pdfX + borderBoxWidth, pdfX, cx - borderBoxWidth/2, cx + borderBoxWidth/2},
			clipY: [borderSideCount]float64{pdfY, pdfY, cy, cy},
		},
		{
			width: box.Border.Left, colour: box.Style.BorderLeftColour, style: box.Style.BorderLeftStyle,
			clipX: [borderSideCount]float64{pdfX, pdfX, cx, cx},
			clipY: [borderSideCount]float64{pdfY, pdfY + borderBoxHeight, cy, cy},
		},
	}

	for i := range sides {
		side := &sides[i] //nolint:gosec // loop bounded by array size
		if side.width <= 0 || side.style == layouter_domain.BorderStyleNone {
			continue
		}
		painter.paintRoundedBorderSide(stream, side, pdfX, pdfY, borderBoxWidth, borderBoxHeight, radii)
	}
}

// paintUniformRoundedBorder draws a uniform rounded border (all sides same).
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds
// the layout box whose borders are drawn.
// Takes pdfX (float64) which specifies the left edge in PDF coordinates.
// Takes pdfY (float64) which specifies the bottom edge in PDF coordinates.
// Takes borderBoxWidth (float64) which specifies the width of the border box.
// Takes borderBoxHeight (float64) which specifies the height of the border box.
// Takes radii (borderCornerRadii) which holds the four corner radii.
func (painter *PdfPainter) paintUniformRoundedBorder(
	stream *ContentStream,
	box *layouter_domain.LayoutBox,
	pdfX, pdfY, borderBoxWidth, borderBoxHeight float64,
	radii borderCornerRadii,
) {
	bw := box.Border.Top
	colour := box.Style.BorderTopColour

	if box.Style.BorderTopStyle == layouter_domain.BorderStyleDouble && bw >= borderDoubleMinWidth {
		lineW := bw / borderDoubleDivisor

		outerInset := lineW / 2
		stream.SaveState()
		painter.setStrokeColour(stream, colour)
		stream.SetLineWidth(lineW)
		emitRoundedRectPath(stream,
			pdfX+outerInset, pdfY+outerInset,
			borderBoxWidth-2*outerInset, borderBoxHeight-2*outerInset,
			math.Max(0, radii.topLeft-outerInset),
			math.Max(0, radii.topRight-outerInset),
			math.Max(0, radii.bottomRight-outerInset),
			math.Max(0, radii.bottomLeft-outerInset))
		stream.Stroke()
		stream.RestoreState()

		innerInset := bw - lineW/2
		stream.SaveState()
		painter.setStrokeColour(stream, colour)
		stream.SetLineWidth(lineW)
		emitRoundedRectPath(stream,
			pdfX+innerInset, pdfY+innerInset,
			borderBoxWidth-2*innerInset, borderBoxHeight-2*innerInset,
			math.Max(0, radii.topLeft-innerInset),
			math.Max(0, radii.topRight-innerInset),
			math.Max(0, radii.bottomRight-innerInset),
			math.Max(0, radii.bottomLeft-innerInset))
		stream.Stroke()
		stream.RestoreState()
		return
	}

	stream.SaveState()
	painter.setStrokeColour(stream, colour)
	stream.SetLineWidth(bw)
	painter.applyBorderDashPattern(stream, box.Style.BorderTopStyle, bw)

	halfBW := bw / 2
	emitRoundedRectPath(stream,
		pdfX+halfBW, pdfY+halfBW,
		borderBoxWidth-bw, borderBoxHeight-bw,
		math.Max(0, radii.topLeft-halfBW),
		math.Max(0, radii.topRight-halfBW),
		math.Max(0, radii.bottomRight-halfBW),
		math.Max(0, radii.bottomLeft-halfBW))
	stream.Stroke()
	stream.RestoreState()
}

// paintRoundedBorderSide draws a single side of a non-uniform rounded border.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes side (*roundedBorderSide) which holds the geometry and style for the border side.
// Takes pdfX (float64) which specifies the left edge in PDF coordinates.
// Takes pdfY (float64) which specifies the bottom edge in PDF coordinates.
// Takes borderBoxWidth (float64) which specifies the width of the border box.
// Takes borderBoxHeight (float64) which specifies the height of the border box.
// Takes radii (borderCornerRadii) which holds the four corner radii.
func (painter *PdfPainter) paintRoundedBorderSide(
	stream *ContentStream,
	side *roundedBorderSide,
	pdfX, pdfY, borderBoxWidth, borderBoxHeight float64,
	radii borderCornerRadii,
) {
	stream.SaveState()

	stream.MoveTo(side.clipX[0], side.clipY[0])
	stream.LineTo(side.clipX[1], side.clipY[1])
	stream.LineTo(side.clipX[2], side.clipY[2])
	stream.LineTo(side.clipX[3], side.clipY[3])
	stream.ClosePath()
	stream.ClipNonZero()

	if side.style == layouter_domain.BorderStyleDouble && side.width >= borderDoubleMinWidth {
		lineW := side.width / borderDoubleDivisor

		outerInset := lineW / 2
		painter.setStrokeColour(stream, side.colour)
		stream.SetLineWidth(lineW)
		emitRoundedRectPath(stream,
			pdfX+outerInset, pdfY+outerInset,
			borderBoxWidth-2*outerInset, borderBoxHeight-2*outerInset,
			math.Max(0, radii.topLeft-outerInset),
			math.Max(0, radii.topRight-outerInset),
			math.Max(0, radii.bottomRight-outerInset),
			math.Max(0, radii.bottomLeft-outerInset))
		stream.Stroke()

		innerInset := side.width - lineW/2
		stream.SetLineWidth(lineW)
		emitRoundedRectPath(stream,
			pdfX+innerInset, pdfY+innerInset,
			borderBoxWidth-2*innerInset, borderBoxHeight-2*innerInset,
			math.Max(0, radii.topLeft-innerInset),
			math.Max(0, radii.topRight-innerInset),
			math.Max(0, radii.bottomRight-innerInset),
			math.Max(0, radii.bottomLeft-innerInset))
		stream.Stroke()
	} else {
		painter.setStrokeColour(stream, side.colour)
		stream.SetLineWidth(side.width)
		painter.applyBorderDashPattern(stream, side.style, side.width)

		halfBW := side.width / 2
		emitRoundedRectPath(stream,
			pdfX+halfBW, pdfY+halfBW,
			borderBoxWidth-side.width, borderBoxHeight-side.width,
			math.Max(0, radii.topLeft-halfBW),
			math.Max(0, radii.topRight-halfBW),
			math.Max(0, radii.bottomRight-halfBW),
			math.Max(0, radii.bottomLeft-halfBW))
		stream.Stroke()
	}

	stream.RestoreState()
}

// resolveBorderImageEdges returns the four border-image edge widths,
// using border-image-width when set or falling back to border widths.
//
// Takes box (*layouter_domain.LayoutBox) which holds the layout box to inspect.
//
// Returns the top, right, bottom, and left edge widths.
func resolveBorderImageEdges(box *layouter_domain.LayoutBox) (top, right, bottom, left float64) {
	biw := box.Style.BorderImageWidth
	if biw > 0 {
		return biw, biw, biw, biw
	}
	return box.Style.BorderTopWidth, box.Style.BorderRightWidth,
		box.Style.BorderBottomWidth, box.Style.BorderLeftWidth
}

// paintBorderImage draws a CSS border-image using 9-slice
// rendering.
//
// Takes stream (*ContentStream) which holds the PDF
// content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds the
// layout box whose border image is drawn.
func (painter *PdfPainter) paintBorderImage(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.Style.BorderImageSource == "" || painter.imageData == nil {
		return
	}

	source := box.Style.BorderImageSource
	data, format, err := painter.imageData.GetImageData(ctx, source)
	if err != nil || len(data) == 0 {
		return
	}

	pixelW, pixelH := ExtractImageDimensions(data, format)
	if pixelW == 0 || pixelH == 0 {
		return
	}

	resourceName := painter.imageEmbedder.RegisterImage(source, data, format, pixelW, pixelH)
	imgW := float64(pixelW)
	imgH := float64(pixelH)

	slice := box.Style.BorderImageSlice
	if slice <= 0 {
		slice = percentageDivisor
	}
	sliceFrac := slice / percentageDivisorFloat
	sliceTop := imgH * sliceFrac
	sliceBottom := sliceTop
	sliceLeft := imgW * sliceFrac
	sliceRight := sliceLeft

	outset := box.Style.BorderImageOutset

	bbx := box.BorderBoxX() - outset
	bby := box.BorderBoxY() - outset
	bbw := box.BorderBoxWidth() + 2*outset
	bbh := box.BorderBoxHeight() + 2*outset

	edgeTop, edgeRight, edgeBottom, edgeLeft := resolveBorderImageEdges(box)

	pdfBbx := bbx
	pdfBby := painter.pageHeight + painter.pageYOffset - bby - bbh

	painter.paintBorderImageSlices(stream, borderImageSliceInput{
		resourceName: resourceName,
		imgW:         imgW, imgH: imgH,
		sliceTop: sliceTop, sliceBottom: sliceBottom, sliceLeft: sliceLeft, sliceRight: sliceRight,
		pdfBbx: pdfBbx, pdfBby: pdfBby, bbw: bbw, bbh: bbh,
		edgeTop: edgeTop, edgeRight: edgeRight, edgeBottom: edgeBottom, edgeLeft: edgeLeft,
	})
}

// borderImageSliceRegion defines a source and destination rectangle for 9-slice rendering.
type borderImageSliceRegion struct {
	// dstX holds the destination X coordinate in PDF space.
	dstX float64

	// dstY holds the destination Y coordinate in PDF space.
	dstY float64

	// dstW holds the destination width in PDF space.
	dstW float64

	// dstH holds the destination height in PDF space.
	dstH float64

	// srcX holds the source X coordinate in image pixels.
	srcX float64

	// srcY holds the source Y coordinate in image pixels.
	srcY float64

	// srcW holds the source width in image pixels.
	srcW float64

	// srcH holds the source height in image pixels.
	srcH float64
}

// borderImageSliceInput groups parameters for 9-slice border image rendering.
type borderImageSliceInput struct {
	// resourceName holds the PDF image resource name.
	resourceName string

	// imgW holds the source image width in pixels.
	imgW float64

	// imgH holds the source image height in pixels.
	imgH float64

	// sliceTop holds the top slice offset in image pixels.
	sliceTop float64

	// sliceBottom holds the bottom slice offset in image pixels.
	sliceBottom float64

	// sliceLeft holds the left slice offset in image pixels.
	sliceLeft float64

	// sliceRight holds the right slice offset in image pixels.
	sliceRight float64

	// pdfBbx holds the border box X position in PDF coordinates.
	pdfBbx float64

	// pdfBby holds the border box Y position in PDF coordinates.
	pdfBby float64

	// bbw holds the border box width including outset.
	bbw float64

	// bbh holds the border box height including outset.
	bbh float64

	// edgeTop holds the top edge width for destination slicing.
	edgeTop float64

	// edgeRight holds the right edge width for destination slicing.
	edgeRight float64

	// edgeBottom holds the bottom edge width for destination slicing.
	edgeBottom float64

	// edgeLeft holds the left edge width for destination slicing.
	edgeLeft float64
}

// paintBorderImageSlices renders all 9 slices of a border image.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes p (borderImageSliceInput) which holds the slice
// geometry and image resource details.
func (*PdfPainter) paintBorderImageSlices(
	stream *ContentStream,
	p borderImageSliceInput,
) {
	midDstX := p.pdfBbx + p.edgeLeft
	midDstY := p.pdfBby + p.edgeBottom
	midDstW := p.bbw - p.edgeLeft - p.edgeRight
	midDstH := p.bbh - p.edgeTop - p.edgeBottom
	midSrcX := p.sliceLeft
	midSrcY := p.sliceTop
	midSrcW := p.imgW - p.sliceLeft - p.sliceRight
	midSrcH := p.imgH - p.sliceTop - p.sliceBottom

	regions := []borderImageSliceRegion{
		{p.pdfBbx, p.pdfBby + p.bbh - p.edgeTop, p.edgeLeft, p.edgeTop, 0, 0, p.sliceLeft, p.sliceTop},
		{p.pdfBbx + p.bbw - p.edgeRight, p.pdfBby + p.bbh - p.edgeTop, p.edgeRight, p.edgeTop, p.imgW - p.sliceRight, 0, p.sliceRight, p.sliceTop},
		{p.pdfBbx, p.pdfBby, p.edgeLeft, p.edgeBottom, 0, p.imgH - p.sliceBottom, p.sliceLeft, p.sliceBottom},
		{p.pdfBbx + p.bbw - p.edgeRight, p.pdfBby, p.edgeRight, p.edgeBottom, p.imgW - p.sliceRight, p.imgH - p.sliceBottom, p.sliceRight, p.sliceBottom},
		{midDstX, p.pdfBby + p.bbh - p.edgeTop, midDstW, p.edgeTop, midSrcX, 0, midSrcW, p.sliceTop},
		{midDstX, p.pdfBby, midDstW, p.edgeBottom, midSrcX, p.imgH - p.sliceBottom, midSrcW, p.sliceBottom},
		{p.pdfBbx, midDstY, p.edgeLeft, midDstH, 0, midSrcY, p.sliceLeft, midSrcH},
		{p.pdfBbx + p.bbw - p.edgeRight, midDstY, p.edgeRight, midDstH, p.imgW - p.sliceRight, midSrcY, p.sliceRight, midSrcH},
		{midDstX, midDstY, midDstW, midDstH, midSrcX, midSrcY, midSrcW, midSrcH},
	}

	for _, r := range regions {
		if r.dstW <= 0 || r.dstH <= 0 || r.srcW <= 0 || r.srcH <= 0 {
			continue
		}

		scaleX := r.dstW / r.srcW * p.imgW
		scaleY := r.dstH / r.srcH * p.imgH
		offsetX := r.dstX - (r.srcX/p.imgW)*scaleX
		offsetY := r.dstY - ((p.imgH-r.srcY-r.srcH)/p.imgH)*scaleY

		stream.SaveState()
		stream.Rectangle(r.dstX, r.dstY, r.dstW, r.dstH)
		stream.ClipNonZero()
		stream.ConcatMatrix(scaleX, 0, 0, scaleY, offsetX, offsetY)
		stream.PaintXObject(p.resourceName)
		stream.RestoreState()
	}
}

// paintOutline draws the CSS outline around the border box.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds
// the layout box whose outline is drawn.
func (painter *PdfPainter) paintOutline(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.Style.OutlineWidth <= 0 || box.Style.OutlineStyle == layouter_domain.BorderStyleNone {
		return
	}

	ow := box.Style.OutlineWidth
	offset := box.Style.OutlineOffset

	borderX := box.BorderBoxX()
	borderY := box.BorderBoxY()
	borderW := box.BorderBoxWidth()
	borderH := box.BorderBoxHeight()

	expand := offset + ow/2
	rectX := borderX - expand
	rectY := painter.pageHeight + painter.pageYOffset - borderY - borderH - expand
	rectW := borderW + 2*expand
	rectH := borderH + 2*expand

	stream.SaveState()
	painter.setStrokeColour(stream, box.Style.OutlineColour)
	stream.SetLineWidth(ow)
	painter.applyBorderDashPattern(stream, box.Style.OutlineStyle, ow)
	stream.Rectangle(rectX, rectY, rectW, rectH)
	stream.Stroke()
	stream.RestoreState()
}

// paintColumnRules draws vertical rules between multi-column children.
//
// Takes stream (*ContentStream) which holds the PDF content stream to write to.
// Takes box (*layouter_domain.LayoutBox) which holds the multi-column layout box.
func (painter *PdfPainter) paintColumnRules(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.Style.ColumnRuleWidth <= 0 || box.Style.ColumnRuleStyle == layouter_domain.BorderStyleNone {
		return
	}
	if len(box.Children) < 2 {
		return
	}

	rw := box.Style.ColumnRuleWidth
	stream.SaveState()
	painter.setStrokeColour(stream, box.Style.ColumnRuleColour)
	stream.SetLineWidth(rw)
	painter.applyBorderDashPattern(stream, box.Style.ColumnRuleStyle, rw)

	for i := 0; i < len(box.Children)-1; i++ {
		left := box.Children[i]
		right := box.Children[i+1]

		leftRightEdge := left.BorderBoxX() + left.BorderBoxWidth()
		rightLeftEdge := right.BorderBoxX()
		ruleX := (leftRightEdge + rightLeftEdge) / 2

		ruleTopY := box.ContentY
		ruleBottomY := box.ContentY + box.ContentHeight

		pdfTop := painter.pageHeight + painter.pageYOffset - ruleTopY
		pdfBottom := painter.pageHeight + painter.pageYOffset - ruleBottomY

		stream.MoveTo(ruleX, pdfBottom)
		stream.LineTo(ruleX, pdfTop)
		stream.Stroke()
	}

	stream.RestoreState()
}
