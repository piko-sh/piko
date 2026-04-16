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

package layouter_domain

// ObjectFitType represents the CSS object-fit property.
type ObjectFitType int

const (
	// ObjectFitFill represents CSS object-fit: fill.
	ObjectFitFill ObjectFitType = iota

	// ObjectFitContain represents CSS object-fit: contain.
	ObjectFitContain

	// ObjectFitCover represents CSS object-fit: cover.
	ObjectFitCover

	// ObjectFitNone represents CSS object-fit: none.
	ObjectFitNone

	// ObjectFitScaleDown represents CSS object-fit: scale-down.
	ObjectFitScaleDown
)

// BackgroundImageType identifies the kind of CSS
// background-image value.
type BackgroundImageType int

const (
	// BackgroundImageNone represents no background image.
	BackgroundImageNone BackgroundImageType = iota

	// BackgroundImageURL represents a url() background.
	BackgroundImageURL

	// BackgroundImageLinearGradient represents a
	// linear-gradient() background.
	BackgroundImageLinearGradient

	// BackgroundImageRadialGradient represents a
	// radial-gradient() background.
	BackgroundImageRadialGradient

	// BackgroundImageRepeatingLinearGradient represents a
	// repeating-linear-gradient() background.
	BackgroundImageRepeatingLinearGradient

	// BackgroundImageRepeatingRadialGradient represents a
	// repeating-radial-gradient() background.
	BackgroundImageRepeatingRadialGradient
)

// RadialGradientShape identifies the shape of a CSS
// radial-gradient: ellipse (default) or circle.
type RadialGradientShape int

const (
	// RadialShapeEllipse represents the default elliptical
	// radial gradient shape.
	RadialShapeEllipse RadialGradientShape = iota

	// RadialShapeCircle represents a circular radial
	// gradient shape.
	RadialShapeCircle
)

// BorderImageRepeatType represents the CSS
// border-image-repeat property.
type BorderImageRepeatType int

const (
	// BorderImageRepeatStretch represents CSS
	// border-image-repeat: stretch.
	BorderImageRepeatStretch BorderImageRepeatType = iota

	// BorderImageRepeatRepeat represents CSS
	// border-image-repeat: repeat.
	BorderImageRepeatRepeat

	// BorderImageRepeatRound represents CSS
	// border-image-repeat: round.
	BorderImageRepeatRound

	// BorderImageRepeatSpace represents CSS
	// border-image-repeat: space.
	BorderImageRepeatSpace
)

// BlendModeType represents the CSS mix-blend-mode property.
type BlendModeType int

const (
	// BlendModeNormal represents the default compositing.
	BlendModeNormal BlendModeType = iota

	// BlendModeMultiply represents mix-blend-mode: multiply.
	BlendModeMultiply

	// BlendModeScreen represents mix-blend-mode: screen.
	BlendModeScreen

	// BlendModeOverlay represents mix-blend-mode: overlay.
	BlendModeOverlay

	// BlendModeDarken represents mix-blend-mode: darken.
	BlendModeDarken

	// BlendModeLighten represents mix-blend-mode: lighten.
	BlendModeLighten

	// BlendModeColorDodge represents mix-blend-mode: color-dodge.
	BlendModeColorDodge

	// BlendModeColorBurn represents mix-blend-mode: color-burn.
	BlendModeColorBurn

	// BlendModeHardLight represents mix-blend-mode: hard-light.
	BlendModeHardLight

	// BlendModeSoftLight represents mix-blend-mode: soft-light.
	BlendModeSoftLight

	// BlendModeDifference represents mix-blend-mode: difference.
	BlendModeDifference

	// BlendModeExclusion represents mix-blend-mode: exclusion.
	BlendModeExclusion

	// BlendModeHue represents mix-blend-mode: hue.
	BlendModeHue

	// BlendModeSaturation represents mix-blend-mode: saturation.
	BlendModeSaturation

	// BlendModeColor represents mix-blend-mode: color.
	BlendModeColor

	// BlendModeLuminosity represents mix-blend-mode: luminosity.
	BlendModeLuminosity
)

// FilterFunction identifies a CSS filter function.
type FilterFunction int

const (
	// FilterNone represents no filter.
	FilterNone FilterFunction = iota

	// FilterBlur represents filter: blur().
	FilterBlur

	// FilterBrightness represents filter: brightness().
	FilterBrightness

	// FilterContrast represents filter: contrast().
	FilterContrast

	// FilterGrayscale represents filter: grayscale().
	FilterGrayscale

	// FilterSepia represents filter: sepia().
	FilterSepia

	// FilterSaturate represents filter: saturate().
	FilterSaturate

	// FilterHueRotate represents filter: hue-rotate().
	FilterHueRotate

	// FilterInvert represents filter: invert().
	FilterInvert

	// FilterOpacity represents filter: opacity().
	FilterOpacity

	// FilterDropShadow represents filter: drop-shadow().
	FilterDropShadow
)

// FilterValue represents a single CSS filter function with its amount.
type FilterValue struct {
	// Function holds the filter function type.
	Function FilterFunction

	// Amount holds the numeric argument for the filter function.
	Amount float64
}

// TextRenderingMode controls how text characters are painted in the PDF.
type TextRenderingMode int

const (
	// TextRenderFill paints text with the fill colour (default).
	TextRenderFill TextRenderingMode = iota

	// TextRenderStroke paints text with the stroke colour only.
	TextRenderStroke

	// TextRenderFillStroke paints text with both fill and stroke,
	// producing an outlined effect.
	TextRenderFillStroke

	// TextRenderInvisible makes text invisible (useful for searchable
	// overlays on scanned documents).
	TextRenderInvisible
)

// TabAlign represents the alignment of text at a tab stop position.
type TabAlign int

const (
	// TabAlignLeft aligns text to the left of the tab stop (default).
	TabAlignLeft TabAlign = iota

	// TabAlignRight aligns text to the right of the tab stop.
	TabAlignRight

	// TabAlignCenter centres text around the tab stop.
	TabAlignCenter
)

// TabStop defines a tab stop position with alignment and an optional
// leader character (e.g. dots in a table of contents).
type TabStop struct {
	// Position is the horizontal position in points from the left
	// edge of the containing block.
	Position float64

	// Align controls how text is aligned at the stop position.
	Align TabAlign

	// Leader is the character repeated to fill the gap before the
	// tab stop. Zero means no leader.
	Leader rune
}

// objectFitTypeNames maps ObjectFitType values to their CSS keyword strings.
var objectFitTypeNames = [...]string{
	ObjectFitFill:      "Fill",
	ObjectFitContain:   "Contain",
	ObjectFitCover:     "Cover",
	ObjectFitNone:      "None",
	ObjectFitScaleDown: "ScaleDown",
}

// backgroundImageTypeNames maps BackgroundImageType values to their CSS keyword strings.
var backgroundImageTypeNames = [...]string{
	BackgroundImageNone:                    cssKeywordNone,
	BackgroundImageURL:                     "url",
	BackgroundImageLinearGradient:          "linear-gradient",
	BackgroundImageRadialGradient:          "radial-gradient",
	BackgroundImageRepeatingLinearGradient: "repeating-linear-gradient",
	BackgroundImageRepeatingRadialGradient: "repeating-radial-gradient",
}

// radialGradientShapeNames maps RadialGradientShape values to their CSS keyword strings.
var radialGradientShapeNames = [...]string{
	RadialShapeEllipse: "ellipse",
	RadialShapeCircle:  "circle",
}

// borderImageRepeatTypeNames maps BorderImageRepeatType values to their CSS keyword strings.
var borderImageRepeatTypeNames = [...]string{
	BorderImageRepeatStretch: "stretch",
	BorderImageRepeatRepeat:  "repeat",
	BorderImageRepeatRound:   "round",
	BorderImageRepeatSpace:   "space",
}

// blendModeTypeNames maps BlendModeType values to their CSS keyword strings.
var blendModeTypeNames = [...]string{
	BlendModeNormal:     "Normal",
	BlendModeMultiply:   "Multiply",
	BlendModeScreen:     "Screen",
	BlendModeOverlay:    "Overlay",
	BlendModeDarken:     "Darken",
	BlendModeLighten:    "Lighten",
	BlendModeColorDodge: "ColorDodge",
	BlendModeColorBurn:  "ColorBurn",
	BlendModeHardLight:  "HardLight",
	BlendModeSoftLight:  "SoftLight",
	BlendModeDifference: "Difference",
	BlendModeExclusion:  "Exclusion",
	BlendModeHue:        "Hue",
	BlendModeSaturation: "Saturation",
	BlendModeColor:      "Color",
	BlendModeLuminosity: "Luminosity",
}

// String returns the Go constant name suffix for this object-fit type.
//
// Returns string which is the constant name suffix.
func (o ObjectFitType) String() string {
	if int(o) < len(objectFitTypeNames) {
		return objectFitTypeNames[o]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this background image type.
//
// Returns string which is the CSS keyword.
func (b BackgroundImageType) String() string {
	if int(b) < len(backgroundImageTypeNames) {
		return backgroundImageTypeNames[b]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this radial gradient shape.
//
// Returns string which is the CSS keyword.
func (r RadialGradientShape) String() string {
	if int(r) < len(radialGradientShapeNames) {
		return radialGradientShapeNames[r]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this border-image-repeat type.
//
// Returns string which is the CSS keyword.
func (b BorderImageRepeatType) String() string {
	if int(b) < len(borderImageRepeatTypeNames) {
		return borderImageRepeatTypeNames[b]
	}
	return cssKeywordUnknown
}

// String returns the PDF blend mode name for this blend mode type.
//
// Returns string which is the PDF blend mode name.
func (b BlendModeType) String() string {
	if int(b) < len(blendModeTypeNames) {
		return blendModeTypeNames[b]
	}
	return cssKeywordUnknown
}

// ParseBlendMode converts a CSS mix-blend-mode keyword to a
// BlendModeType.
//
// Takes value (string) which is the CSS keyword to parse.
//
// Returns BlendModeType which is the parsed blend mode, or
// BlendModeNormal for unrecognised values.
func ParseBlendMode(value string) BlendModeType {
	switch value {
	case "multiply":
		return BlendModeMultiply
	case "screen":
		return BlendModeScreen
	case "overlay":
		return BlendModeOverlay
	case "darken":
		return BlendModeDarken
	case "lighten":
		return BlendModeLighten
	case "color-dodge":
		return BlendModeColorDodge
	case "color-burn":
		return BlendModeColorBurn
	case "hard-light":
		return BlendModeHardLight
	case "soft-light":
		return BlendModeSoftLight
	case "difference":
		return BlendModeDifference
	case "exclusion":
		return BlendModeExclusion
	case "hue":
		return BlendModeHue
	case "saturation":
		return BlendModeSaturation
	case "color":
		return BlendModeColor
	case "luminosity":
		return BlendModeLuminosity
	default:
		return BlendModeNormal
	}
}
