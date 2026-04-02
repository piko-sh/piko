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
	"fmt"
	"math"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

// backgroundBox returns the layout-coordinate rectangle for the given
// background-clip or background-origin value. Defaults to border-box.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box to
// measure.
// Takes value (string) which is the CSS background-clip or
// background-origin keyword.
//
// Returns x (float64) which is the left edge position.
// Returns y (float64) which is the top edge position.
// Returns w (float64) which is the rectangle width.
// Returns h (float64) which is the rectangle height.
func backgroundBox(box *layouter_domain.LayoutBox, value string) (x, y, w, h float64) {
	switch value {
	case "padding-box":
		x = box.ContentX - box.Padding.Left
		y = box.ContentY - box.Padding.Top
		w = box.ContentWidth + box.Padding.Horizontal()
		h = box.ContentHeight + box.Padding.Vertical()
	case "content-box":
		x = box.ContentX
		y = box.ContentY
		w = box.ContentWidth
		h = box.ContentHeight
	default:
		x = box.BorderBoxX()
		y = box.BorderBoxY()
		w = box.BorderBoxWidth()
		h = box.BorderBoxHeight()
	}
	return x, y, w, h
}

// paintBackground renders the background colour and all background
// layers (images, gradients) for a layout box.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the box to paint.
func (painter *PdfPainter) paintBackground(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox) {
	colour := box.Style.BackgroundColour
	if colour.Alpha > 0 {
		stream.SaveState()
		painter.setFillColour(stream, colour)

		clipX, clipY, clipW, clipH := backgroundBox(box, box.Style.BgClip)
		pdfX := clipX
		pdfY := painter.pageHeight + painter.pageYOffset - clipY - clipH
		w := clipW
		h := clipH

		if painter.hasAnyBorderRadius(box) {
			emitRoundedRectPath(stream, pdfX, pdfY, w, h,
				box.Style.BorderTopLeftRadius,
				box.Style.BorderTopRightRadius,
				box.Style.BorderBottomRightRadius,
				box.Style.BorderBottomLeftRadius)
			stream.Fill()
		} else {
			stream.Rectangle(pdfX, pdfY, w, h)
			stream.Fill()
		}
		stream.RestoreState()
	}

	for i := len(box.Style.BgImages) - 1; i >= 0; i-- {
		bg := box.Style.BgImages[i]
		switch bg.Type {
		case layouter_domain.BackgroundImageURL:
			painter.paintBackgroundImage(ctx, stream, box, bg)
		case layouter_domain.BackgroundImageLinearGradient, layouter_domain.BackgroundImageRepeatingLinearGradient:
			painter.paintLinearGradient(stream, box, bg)
		case layouter_domain.BackgroundImageRadialGradient, layouter_domain.BackgroundImageRepeatingRadialGradient:
			painter.paintRadialGradient(stream, box, bg)
		}
	}
}

// paintBackgroundImage draws a URL-based CSS background-image. Supports
// background-size (cover, contain, explicit), background-position,
// and background-repeat (no-repeat, repeat, repeat-x, repeat-y).
//
// Takes ctx (context.Context) which controls cancellation.
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the target box.
func (painter *PdfPainter) paintBackgroundImage(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox, bg layouter_domain.BackgroundImage) {
	if painter.imageData == nil {
		return
	}

	data, format, err := painter.imageData.GetImageData(ctx, bg.URL)
	if err != nil || len(data) == 0 {
		return
	}

	pixelW, pixelH := ExtractImageDimensions(data, format)
	if pixelW == 0 || pixelH == 0 {
		return
	}

	resourceName := painter.imageEmbedder.RegisterImage(bg.URL, data, format, pixelW, pixelH)

	areaX, areaY, areaW, areaH := backgroundBox(box, box.Style.BgOrigin)

	intrinsicW := float64(pixelW)
	intrinsicH := float64(pixelH)
	imgW, imgH := painter.resolveBackgroundSize(box.Style.BgSize, areaW, areaH, intrinsicW, intrinsicH)

	posXFrac, posYFrac := parseObjectPosition(box.Style.BgPosition)
	imgX := areaX + (areaW-imgW)*posXFrac
	imgY := areaY + (areaH-imgH)*posYFrac

	repeat := strings.TrimSpace(box.Style.BgRepeat)
	if repeat == "" {
		repeat = "repeat"
	}

	stream.SaveState()
	pdfAreaY := painter.pageHeight + painter.pageYOffset - areaY - areaH
	if painter.hasAnyBorderRadius(box) {
		emitRoundedRectPath(stream, areaX, pdfAreaY, areaW, areaH,
			box.Style.BorderTopLeftRadius,
			box.Style.BorderTopRightRadius,
			box.Style.BorderBottomRightRadius,
			box.Style.BorderBottomLeftRadius)
		stream.ClipNonZero()
	} else {
		stream.Rectangle(areaX, pdfAreaY, areaW, areaH)
		stream.ClipNonZero()
	}

	painter.paintBackgroundTiles(stream, backgroundTileParams{
		imgX: imgX, imgY: imgY, imgW: imgW, imgH: imgH,
		areaX: areaX, areaY: areaY, areaW: areaW, areaH: areaH,
		repeat: repeat, resourceName: resourceName,
	})

	stream.RestoreState()
}

// backgroundTileParams groups the parameters for tiled background painting.
type backgroundTileParams struct {
	// repeat holds the CSS background-repeat value.
	repeat string

	// resourceName holds the PDF XObject resource key for the image.
	resourceName string

	// imgX holds the image origin X in layout coordinates.
	imgX float64

	// imgY holds the image origin Y in layout coordinates.
	imgY float64

	// imgW holds the rendered image width in points.
	imgW float64

	// imgH holds the rendered image height in points.
	imgH float64

	// areaX holds the tiling area origin X in layout coordinates.
	areaX float64

	// areaY holds the tiling area origin Y in layout coordinates.
	areaY float64

	// areaW holds the tiling area width in points.
	areaW float64

	// areaH holds the tiling area height in points.
	areaH float64
}

// paintBackgroundTiles emits the tiled background image draws.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes p (backgroundTileParams) which holds the tiling parameters.
func (painter *PdfPainter) paintBackgroundTiles(
	stream *ContentStream,
	p backgroundTileParams,
) {
	repeatX := p.repeat == "repeat" || p.repeat == "repeat-x"
	repeatY := p.repeat == "repeat" || p.repeat == "repeat-y"

	startX := resolveStartPosition(p.imgX, p.areaX, p.imgW, repeatX)
	startY := resolveStartPosition(p.imgY, p.areaY, p.imgH, repeatY)

	ty := startY
	for {
		painter.paintTileRow(stream, p, startX, ty, repeatX)
		if !repeatY {
			break
		}
		ty += p.imgH
		if ty >= p.areaY+p.areaH {
			break
		}
	}
}

// resolveStartPosition computes the starting coordinate for tiled
// repetition along one axis.
//
// Takes img (float64) which is the initial image position.
// Takes area (float64) which is the tiling area origin.
// Takes size (float64) which is the tile size along this axis.
// Takes repeat (bool) which indicates whether tiling is enabled.
//
// Returns float64 which is the adjusted starting coordinate.
func resolveStartPosition(img, area, size float64, repeat bool) float64 {
	start := img
	if repeat {
		for start > area {
			start -= size
		}
	}
	return start
}

// paintTileRow emits one horizontal row of tiled background images.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes p (backgroundTileParams) which holds the tiling parameters.
// Takes startX (float64) which is the first tile X position.
// Takes ty (float64) which is the row Y position in layout coordinates.
// Takes repeatX (bool) which indicates whether horizontal repeating
// is enabled.
func (painter *PdfPainter) paintTileRow(
	stream *ContentStream,
	p backgroundTileParams,
	startX, ty float64,
	repeatX bool,
) {
	tx := startX
	for {
		pdfTy := painter.pageHeight + painter.pageYOffset - ty - p.imgH

		stream.SaveState()
		stream.ConcatMatrix(p.imgW, 0, 0, p.imgH, tx, pdfTy)
		stream.PaintXObject(p.resourceName)
		stream.RestoreState()

		if !repeatX {
			break
		}
		tx += p.imgW
		if tx >= p.areaX+p.areaW {
			break
		}
	}
}

// resolveBackgroundSize computes the rendered size of a background image
// given the CSS background-size value, area dimensions, and intrinsic
// image dimensions.
//
// Takes bgSize (string) which is the CSS background-size value.
// Takes areaW, areaH (float64) which define the positioning area
// dimensions.
// Takes intrinsicW, intrinsicH (float64) which are the image's
// natural dimensions.
//
// Returns imgWidth (float64) which is the computed
// rendering width.
// Returns imgHeight (float64) which is the computed
// rendering height.
func (painter *PdfPainter) resolveBackgroundSize(
	bgSize string, areaW, areaH, intrinsicW, intrinsicH float64,
) (imgWidth float64, imgHeight float64) {
	bgSize = strings.TrimSpace(strings.ToLower(bgSize))
	switch bgSize {
	case "cover":
		scale := math.Max(areaW/intrinsicW, areaH/intrinsicH)
		return intrinsicW * scale, intrinsicH * scale
	case "contain":
		scale := math.Min(areaW/intrinsicW, areaH/intrinsicH)
		return intrinsicW * scale, intrinsicH * scale
	case "", "auto":
		return intrinsicW, intrinsicH
	default:

		parts := strings.Fields(bgSize)
		w := painter.parseBgDimension(parts[0], areaW, intrinsicW)
		h := intrinsicH
		if len(parts) >= 2 {
			h = painter.parseBgDimension(parts[1], areaH, intrinsicH)
		} else {
			if intrinsicW > 0 {
				h = intrinsicH * (w / intrinsicW)
			}
		}
		return w, h
	}
}

// parseBgDimension parses a single CSS background-size dimension value
// and resolves it to points.
//
// Takes s (string) which is the dimension string (e.g. "50%", "auto").
// Takes areaDim (float64) which is the reference area dimension.
// Takes intrinsicDim (float64) which is the image's natural dimension.
//
// Returns float64 which is the resolved size in points.
func (*PdfPainter) parseBgDimension(s string, areaDim, intrinsicDim float64) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "auto" {
		return intrinsicDim
	}
	if after, ok := strings.CutSuffix(s, percentSuffix); ok {
		v := parseNumber(after)
		return areaDim * v / percentageDivisorFloat
	}
	return parseLength(s)
}

// paintLinearGradient draws a CSS linear-gradient background using a
// native PDF Type 2 (axial) shading dictionary. When any colour stop
// has alpha < 1.0, a luminosity soft mask is created from the alpha
// channel so the gradient composites correctly over the background.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the target box.
// Takes bg (layouter_domain.BackgroundImage) which holds the gradient
// definition.
func (painter *PdfPainter) paintLinearGradient(stream *ContentStream, box *layouter_domain.LayoutBox, bg layouter_domain.BackgroundImage) {
	stops := NormaliseGradientStops(bg.Stops)
	if len(stops) < 2 {
		return
	}
	if bg.Type == layouter_domain.BackgroundImageRepeatingLinearGradient {
		stops = ExpandRepeatingStops(stops)
	}

	areaX, areaY, areaW, areaH := backgroundBox(box, box.Style.BgOrigin)

	pdfAreaX := areaX
	pdfAreaY := painter.pageHeight + painter.pageYOffset - areaY - areaH

	x0, y0, x1, y1 := ComputeLinearGradientAxis(bg.Angle, pdfAreaX, pdfAreaY, areaW, areaH)
	name := painter.shadingManager.RegisterLinearGradient(x0, y0, x1, y1, stops)

	stream.SaveState()

	if StopsHaveAlpha(stops) {
		alphaStops := AlphaStops(stops)

		localX0, localY0, localX1, localY1 := ComputeLinearGradientAxis(bg.Angle, 0, 0, areaW, areaH)
		maskName := painter.shadingManager.RegisterLinearGradientGray(localX0, localY0, localX1, localY1, alphaStops)
		painter.applyShadingSoftMask(stream, maskName, pdfAreaX, pdfAreaY, areaW, areaH)
	}

	if painter.hasAnyBorderRadius(box) {
		emitRoundedRectPath(stream, pdfAreaX, pdfAreaY, areaW, areaH,
			box.Style.BorderTopLeftRadius,
			box.Style.BorderTopRightRadius,
			box.Style.BorderBottomRightRadius,
			box.Style.BorderBottomLeftRadius)
		stream.ClipNonZero()
	} else {
		stream.Rectangle(pdfAreaX, pdfAreaY, areaW, areaH)
		stream.ClipNonZero()
	}
	stream.PaintShading(name)
	stream.RestoreState()
}

// paintRadialGradient draws a CSS radial-gradient background using
// a native PDF Type 3 (radial) shading dictionary.
//
// For circles the radius extends to the farthest corner. For ellipses
// the gradient has the same aspect ratio as the box and passes through
// the farthest corner; a CTM scale transform maps a circular shading
// to an ellipse since PDF Type 3 shading only supports circles.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which is the target box.
// Takes bg (layouter_domain.BackgroundImage) which holds the gradient
// definition.
func (painter *PdfPainter) paintRadialGradient(stream *ContentStream, box *layouter_domain.LayoutBox, bg layouter_domain.BackgroundImage) {
	stops := NormaliseGradientStops(bg.Stops)
	if len(stops) < 2 {
		return
	}
	if bg.Type == layouter_domain.BackgroundImageRepeatingRadialGradient {
		stops = ExpandRepeatingStops(stops)
	}

	areaX, areaY, areaW, areaH := backgroundBox(box, box.Style.BgOrigin)

	pdfAreaX := areaX
	pdfAreaY := painter.pageHeight + painter.pageYOffset - areaY - areaH

	cx := pdfAreaX + areaW/2
	cy := pdfAreaY + areaH/2

	isCircle := bg.Shape == layouter_domain.RadialShapeCircle

	stream.SaveState()

	painter.applyRadialAlphaMask(stream, stops, isCircle, pdfAreaX, pdfAreaY, areaW, areaH)

	if painter.hasAnyBorderRadius(box) {
		emitRoundedRectPath(stream, pdfAreaX, pdfAreaY, areaW, areaH,
			box.Style.BorderTopLeftRadius,
			box.Style.BorderTopRightRadius,
			box.Style.BorderBottomRightRadius,
			box.Style.BorderBottomLeftRadius)
		stream.ClipNonZero()
	} else {
		stream.Rectangle(pdfAreaX, pdfAreaY, areaW, areaH)
		stream.ClipNonZero()
	}

	if isCircle {
		r := math.Sqrt(areaW*areaW+areaH*areaH) / 2
		name := painter.shadingManager.RegisterRadialGradient(cx, cy, r, stops)
		stream.PaintShading(name)
	} else {
		rx := areaW / math.Sqrt2
		ry := areaH / math.Sqrt2
		name := painter.shadingManager.RegisterRadialGradient(cx, cy, ry, stops)
		sx := rx / ry
		stream.ConcatMatrix(sx, 0, 0, 1, cx*(1-sx), 0)
		stream.PaintShading(name)
	}

	stream.RestoreState()
}

// applyRadialAlphaMask applies a radial alpha soft mask when the gradient
// stops contain semi-transparent colours.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes stops ([]ResolvedStop) which are the gradient colour stops.
// Takes isCircle (bool) which selects circular versus elliptical shape.
// Takes pdfAreaX, pdfAreaY (float64) which specify the area origin in
// PDF coordinates.
// Takes areaW, areaH (float64) which are the area dimensions in points.
func (painter *PdfPainter) applyRadialAlphaMask(
	stream *ContentStream,
	stops []ResolvedStop,
	isCircle bool,
	pdfAreaX, pdfAreaY, areaW, areaH float64,
) {
	if !StopsHaveAlpha(stops) {
		return
	}
	alphaStops := AlphaStops(stops)
	localCx := areaW / 2
	localCy := areaH / 2
	if isCircle {
		r := math.Sqrt(areaW*areaW+areaH*areaH) / 2
		maskName := painter.shadingManager.RegisterRadialGradientGray(localCx, localCy, r, alphaStops)
		painter.applyShadingSoftMask(stream, maskName, pdfAreaX, pdfAreaY, areaW, areaH)
	} else {
		ry := areaH / math.Sqrt2
		maskName := painter.shadingManager.RegisterRadialGradientGray(localCx, localCy, ry, alphaStops)
		painter.applyShadingSoftMask(stream, maskName, pdfAreaX, pdfAreaY, areaW, areaH)
	}
}

// applyShadingSoftMask builds a luminosity soft mask from an
// already-registered grayscale shading and sets it as the active
// ExtGState.
//
// The shading must use local coordinates (0,0 to w,h). The mask is
// positioned at (x, y) in PDF page coordinates via CTM translation.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes shadingName (string) which is the registered shading key.
// Takes x, y (float64) which specify the mask position in PDF
// coordinates.
// Takes w, h (float64) which are the mask dimensions in points.
func (painter *PdfPainter) applyShadingSoftMask(stream *ContentStream, shadingName string, x, y, w, h float64) {
	var maskStream ContentStream
	maskStream.PaintShading(shadingName)
	content := []byte(maskStream.String())

	formObject := painter.writer.AllocateObject()
	bbox := fmt.Sprintf("0 0 %s %s", formatFloat(w), formatFloat(h))

	painter.maskFormObjects = append(painter.maskFormObjects, maskFormObject{
		objectNumber: formObject,
		bbox:         bbox,
		content:      content,
		shadingName:  shadingName,
	})

	gsName := painter.extGStateManager.RegisterSoftMask(formObject)

	stream.ConcatMatrix(1, 0, 0, 1, x, y)
	stream.SetExtGState(gsName)
	stream.ConcatMatrix(1, 0, 0, 1, -x, -y)
}

// applyMaskImage parses a CSS mask-image value and applies it as a
// PDF soft mask (SMask) via ExtGState. Currently supports
// linear-gradient masks; other values are silently skipped.
//
// Takes stream (*ContentStream) which receives PDF operators.
// Takes box (*layouter_domain.LayoutBox) which supplies the mask-image
// CSS value and box dimensions.
//
// Returns true if the mask was applied (caller must RestoreState).
func (painter *PdfPainter) applyMaskImage(stream *ContentStream, box *layouter_domain.LayoutBox) bool {
	value := strings.TrimSpace(box.Style.MaskImage)

	ctx := layouter_domain.DefaultResolutionContext()
	bg := layouter_domain.ParseMaskBackgroundImage(value, ctx)
	if bg.Type != layouter_domain.BackgroundImageLinearGradient &&
		bg.Type != layouter_domain.BackgroundImageRadialGradient {
		return false
	}

	stops := NormaliseGradientStops(bg.Stops)
	if len(stops) < 2 {
		return false
	}

	grayStops := painter.convertToGrayscaleStops(stops)

	bbw := box.BorderBoxWidth()
	bbh := box.BorderBoxHeight()

	pdfX := box.BorderBoxX()
	pdfY := painter.pageHeight + painter.pageYOffset - box.BorderBoxY() - bbh

	maskShadingName := painter.buildMaskShading(bg, bbw, bbh, grayStops)
	content := painter.buildMaskContent(maskShadingName)

	formObject := painter.writer.AllocateObject()
	bbox := fmt.Sprintf("0 0 %s %s", formatFloat(bbw), formatFloat(bbh))

	painter.maskFormObjects = append(painter.maskFormObjects, maskFormObject{
		objectNumber: formObject,
		bbox:         bbox,
		content:      content,
		shadingName:  maskShadingName,
	})

	gsName := painter.extGStateManager.RegisterSoftMask(formObject)

	stream.SaveState()
	stream.ConcatMatrix(1, 0, 0, 1, pdfX, pdfY)
	stream.SetExtGState(gsName)
	stream.ConcatMatrix(1, 0, 0, 1, -pdfX, -pdfY)

	return true
}

// convertToGrayscaleStops converts gradient stops to greyscale luminance values.
//
// Takes stops ([]ResolvedStop) which are the RGB gradient stops.
//
// Returns []ResolvedStop which holds the greyscale-converted stops.
func (*PdfPainter) convertToGrayscaleStops(stops []ResolvedStop) []ResolvedStop {
	grayStops := make([]ResolvedStop, len(stops))
	for i, stop := range stops {
		luminance := luminanceRed*stop.Red + luminanceGreen*stop.Green + luminanceBlue*stop.Blue
		grayStops[i] = ResolvedStop{
			Red:      luminance,
			Green:    luminance,
			Blue:     luminance,
			Position: stop.Position,
		}
	}
	return grayStops
}

// buildMaskShading registers and returns the shading name for a mask gradient.
//
// Takes bg (layouter_domain.BackgroundImage) which holds the gradient
// definition.
// Takes bbw, bbh (float64) which are the border box dimensions.
// Takes grayStops ([]ResolvedStop) which are the greyscale stops.
//
// Returns string which is the registered shading resource name.
func (painter *PdfPainter) buildMaskShading(bg layouter_domain.BackgroundImage, bbw, bbh float64, grayStops []ResolvedStop) string {
	if bg.Type == layouter_domain.BackgroundImageLinearGradient {
		x0, y0, x1, y1 := ComputeLinearGradientAxis(bg.Angle, 0, 0, bbw, bbh)
		return painter.shadingManager.RegisterLinearGradientGray(x0, y0, x1, y1, grayStops)
	}
	cx := bbw / 2
	cy := bbh / 2
	radius := math.Sqrt(cx*cx + cy*cy)
	return painter.shadingManager.RegisterRadialGradientGray(cx, cy, radius, grayStops)
}

// buildMaskContent produces the content stream bytes for a mask form XObject.
//
// Takes shadingName (string) which is the shading resource key.
//
// Returns []byte which holds the content stream for the mask.
func (*PdfPainter) buildMaskContent(shadingName string) []byte {
	var maskStream ContentStream
	maskStream.PaintShading(shadingName)
	return []byte(maskStream.String())
}
