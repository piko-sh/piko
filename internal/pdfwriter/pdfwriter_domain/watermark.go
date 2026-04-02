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
)

const (
	// defaultWatermarkFontSize is the font size in points when none is specified.
	defaultWatermarkFontSize = 60

	// defaultWatermarkColour is the default RGB component value for light grey.
	defaultWatermarkColour = 0.85

	// defaultWatermarkAngle is the default rotation angle in degrees.
	defaultWatermarkAngle = 45

	// defaultWatermarkOpacity is the default opacity value in [0, 1].
	defaultWatermarkOpacity = 0.3

	// degreesToRadians is the conversion factor from degrees to radians.
	degreesToRadians = math.Pi / 180.0

	// averageCharWidthRatio is the approximate ratio of
	// character width to font size for Helvetica.
	averageCharWidthRatio = 0.52
)

// WatermarkConfig holds parameters for a diagonal text watermark rendered
// behind content on every page.
type WatermarkConfig struct {
	// Text is the watermark string (e.g. "DRAFT", "CONFIDENTIAL").
	Text string

	// FontSize in points. Defaults to 60 if zero.
	FontSize float64

	// ColourR, ColourG, ColourB are RGB fill colour components in [0, 1].
	// Default to 0.85 (light grey) if all three are zero.
	ColourR float64

	// ColourG holds the green component of the RGB fill colour in [0, 1].
	ColourG float64

	// ColourB holds the blue component of the RGB fill colour in [0, 1].
	ColourB float64

	// Angle in degrees. Defaults to 45 if zero.
	Angle float64

	// Opacity in [0, 1]. Defaults to 0.3 if zero.
	Opacity float64
}

// applyDefaults fills in zero-value fields with sensible defaults.
func (wm *WatermarkConfig) applyDefaults() {
	if wm.FontSize == 0 {
		wm.FontSize = defaultWatermarkFontSize
	}
	if wm.ColourR == 0 && wm.ColourG == 0 && wm.ColourB == 0 {
		wm.ColourR = defaultWatermarkColour
		wm.ColourG = defaultWatermarkColour
		wm.ColourB = defaultWatermarkColour
	}
	if wm.Angle == 0 {
		wm.Angle = defaultWatermarkAngle
	}
	if wm.Opacity == 0 {
		wm.Opacity = defaultWatermarkOpacity
	}
}

// buildWatermarkStream generates the content stream operators for a
// watermark rendered behind page content. The watermark uses Helvetica
// (Type1) so no font embedding is required.
//
// Takes fontResourceName (string) which is the Helvetica resource name
// (e.g. "FW").
// Takes gsName (string) which is the ExtGState resource name for opacity.
// Takes pageWidth, pageHeight (float64) which are the page dimensions.
//
// Returns the content stream operators as a string.
func buildWatermarkStream(
	wm *WatermarkConfig,
	fontResourceName string,
	gsName string,
	pageWidth, pageHeight float64,
) string {
	var stream ContentStream

	stream.SaveState()
	stream.SetExtGState(gsName)
	stream.BeginText()
	stream.SetFont(fontResourceName, wm.FontSize)
	stream.SetFillColourRGB(wm.ColourR, wm.ColourG, wm.ColourB)

	rad := wm.Angle * degreesToRadians
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	textWidth := float64(len(wm.Text)) * wm.FontSize * averageCharWidthRatio
	cx := pageWidth/2 - (textWidth/2)*cosA
	cy := pageHeight/2 - (textWidth/2)*sinA

	stream.ConcatMatrix(cosA, sinA, -sinA, cosA, cx, cy)
	stream.ShowText(wm.Text)
	stream.EndText()
	stream.RestoreState()

	return stream.String()
}
