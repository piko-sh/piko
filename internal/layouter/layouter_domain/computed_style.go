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

import (
	"fmt"
)

// CounterEntry represents a single counter-reset or counter-increment operation.
type CounterEntry struct {
	// Name holds the counter identifier.
	Name string

	// Value holds the numeric value for the counter operation.
	Value int
}

const (
	// defaultFontSizePt is the initial font size in points.
	defaultFontSizePt = 12.0

	// defaultFontWeight is the initial font weight value.
	defaultFontWeight = 400

	// defaultTabSize is the default number of spaces per tab character.
	defaultTabSize = 8
)

// DimensionUnit identifies how a Dimension value should be interpreted.
type DimensionUnit int

const (
	// DimensionUnitAuto represents the CSS "auto" value.
	DimensionUnitAuto DimensionUnit = iota

	// DimensionUnitPoints represents an absolute length in points.
	DimensionUnitPoints

	// DimensionUnitPercentage represents a percentage of the containing block.
	DimensionUnitPercentage

	// DimensionUnitMinContent represents the CSS "min-content" keyword.
	DimensionUnitMinContent

	// DimensionUnitMaxContent represents the CSS "max-content" keyword.
	DimensionUnitMaxContent

	// DimensionUnitFitContent represents the CSS fit-content(<length-percentage>) function.
	// Value holds the argument resolved to points.
	DimensionUnitFitContent

	// DimensionUnitFitContentStretch represents the bare CSS "fit-content" keyword (no
	// argument). At resolution time the available width is used as the clamp argument.
	DimensionUnitFitContentStretch
)

// Dimension represents a CSS length value that may be auto, an absolute length in points,
// or a percentage.
type Dimension struct {
	// Value holds the numeric value. Meaningless when Unit is DimensionUnitAuto.
	Value float64

	// Unit identifies how Value should be interpreted.
	Unit DimensionUnit
}

// GridTrackUnit identifies how a grid track size should be interpreted.
type GridTrackUnit int

const (
	// GridTrackAuto represents an auto-sized track.
	GridTrackAuto GridTrackUnit = iota

	// GridTrackPoints represents a fixed track size in points.
	GridTrackPoints

	// GridTrackPercentage represents a percentage of the container.
	GridTrackPercentage

	// GridTrackFr represents a flexible fraction of remaining space.
	GridTrackFr

	// GridTrackMinContent represents the min-content sizing keyword.
	GridTrackMinContent

	// GridTrackMaxContent represents the max-content sizing keyword.
	GridTrackMaxContent

	// GridTrackFitContent represents the fit-content(<length>) function for grid tracks.
	// Value holds the argument in points.
	GridTrackFitContent

	// GridTrackFitContentPct represents the fit-content(<percentage>) function for grid
	// tracks. Value holds the raw percentage.
	GridTrackFitContentPct
)

// GridTrack represents a single track definition in a grid template.
type GridTrack struct {
	// Value holds the numeric value. Meaningless when Unit is GridTrackAuto.
	Value float64

	// Unit identifies how Value should be interpreted.
	Unit GridTrackUnit
}

// GridAutoRepeat stores a deferred auto-fill or auto-fit repeat pattern that is expanded
// at layout time when the container width is known.
type GridAutoRepeat struct {
	// Pattern is the track list to repeat.
	Pattern []GridTrack

	// InsertIndex is the position in the fixed template tracks where the expanded pattern
	// should be spliced in.
	InsertIndex int

	// AfterCount is the number of fixed tracks that follow the auto-repeat region in the
	// original track list.
	AfterCount int

	// Type is GridAutoRepeatFill or GridAutoRepeatFit.
	Type GridAutoRepeatType
}

// GridLine represents a grid placement value for an item's start or end position.
type GridLine struct {
	// Line is the 1-based grid line number. Zero means auto-placement.
	Line int

	// Span is the number of tracks to span. Zero means no span keyword was used.
	Span int

	// IsAuto indicates whether this is auto-placement.
	IsAuto bool
}

// BoxShadowValue represents a single box-shadow layer. Box-shadow is a visual property
// that does not affect layout; it is stored here for consumption by the paint phase.
type BoxShadowValue struct {
	// OffsetX is the horizontal shadow offset in points.
	OffsetX float64

	// OffsetY is the vertical shadow offset in points.
	OffsetY float64

	// BlurRadius is the shadow blur radius in points.
	BlurRadius float64

	// SpreadRadius is the shadow spread radius in points.
	SpreadRadius float64

	// Colour is the shadow colour.
	Colour Colour

	// Inset indicates whether the shadow is inset.
	Inset bool
}

// TextShadowValue represents a single text-shadow layer. Text-shadow is a visual property
// that does not affect layout; it is stored here for consumption by the paint phase.
type TextShadowValue struct {
	// OffsetX is the horizontal shadow offset in points.
	OffsetX float64

	// OffsetY is the vertical shadow offset in points.
	OffsetY float64

	// BlurRadius is the shadow blur radius in points.
	BlurRadius float64

	// Colour is the shadow colour.
	Colour Colour
}

// GradientStop represents a single colour stop in a CSS gradient.
type GradientStop struct {
	// Colour is the stop colour.
	Colour Colour

	// Position is the stop position as a fraction (0-1). A value of -1 indicates
	// auto-placement.
	Position float64
}

// BackgroundImage represents a parsed CSS background-image value.
type BackgroundImage struct {
	// URL is the image URL for BackgroundImageURL type.
	URL string

	// Stops holds the colour stops for gradient types.
	Stops []GradientStop

	// Angle is the gradient angle in degrees for linear gradients.
	Angle float64

	// Type identifies the kind of background image.
	Type BackgroundImageType

	// Shape is the radial gradient shape (circle or ellipse). Only meaningful when Type is
	// BackgroundImageRadialGradient.
	Shape RadialGradientShape
}

// ComputedStyle holds all resolved CSS properties for a single element.
//
// All length values are in points. Percentages are stored as Dimension values and
// resolved during layout when the containing block dimensions are known.
//
// The properties are grouped into embedded value groups whose fields are promoted, so
// each property is read and written directly on the ComputedStyle. The groups are ordered
// to minimise the GC pointer-scan window, placing the pointer-bearing reference group
// first and the pointer-free groups after it in descending alignment order.
type ComputedStyle struct {
	// styleReferenceValues holds the pointer-bearing properties (pointers, maps, strings and
	// slices).
	styleReferenceValues

	// styleColourValues holds the colour properties.
	styleColourValues

	// styleGeometryValues holds the grid placement lines and CSS dimensions.
	styleGeometryValues

	// styleNumericValues holds the properties resolved to float64 and int values.
	styleNumericValues

	// styleKeywordValues holds the properties resolved to CSS keyword enums.
	styleKeywordValues

	// styleFlagValues holds the boolean flags.
	styleFlagValues
}

var (
	// initialComputedStyle holds every property at its CSS initial value.
	//
	// It is built once so DefaultComputedStyle returns a plain copy instead of assembling
	// each value group per call. Every reference-typed property in it is nil or an immutable
	// string, so copies never share mutable state.
	initialComputedStyle = newInitialComputedStyle()
)

// IsAuto reports whether this dimension represents "auto".
//
// Returns true when the unit is DimensionUnitAuto.
func (d Dimension) IsAuto() bool {
	return d.Unit == DimensionUnitAuto
}

// IsMinContent reports whether this dimension represents "min-content".
//
// Returns true when the unit is DimensionUnitMinContent.
func (d Dimension) IsMinContent() bool {
	return d.Unit == DimensionUnitMinContent
}

// IsMaxContent reports whether this dimension represents "max-content".
//
// Returns true when the unit is DimensionUnitMaxContent.
func (d Dimension) IsMaxContent() bool {
	return d.Unit == DimensionUnitMaxContent
}

// IsFitContent reports whether this dimension represents a fit-content sizing keyword or
// function.
//
// Returns true for fit-content or fit-content(<arg>).
func (d Dimension) IsFitContent() bool {
	return d.Unit == DimensionUnitFitContent || d.Unit == DimensionUnitFitContentStretch
}

// IsIntrinsic reports whether this dimension represents an intrinsic sizing keyword
// (min-content, max-content, or fit-content).
//
// Returns true for any intrinsic sizing keyword.
func (d Dimension) IsIntrinsic() bool {
	return d.IsMinContent() || d.IsMaxContent() || d.IsFitContent()
}

// Resolve returns the absolute value in points.
//
// When the unit is auto, the fallback value is returned. When the unit is a percentage,
// the value is resolved against containingBlockSize.
//
// Takes containingBlockSize (float64) which is the size of the containing block used to
// resolve percentages.
//
// Takes fallback (float64) which is the value returned when the dimension is auto.
//
// Returns the resolved value in points.
func (d Dimension) Resolve(containingBlockSize, fallback float64) float64 {
	switch d.Unit {
	case DimensionUnitPoints:
		return d.Value
	case DimensionUnitPercentage:
		return d.Value / percentageDivisor * containingBlockSize
	default:
		return fallback
	}
}

// String returns a human-readable representation of the dimension.
//
// Returns the formatted string.
func (d Dimension) String() string {
	switch d.Unit {
	case DimensionUnitAuto:
		return "auto"
	case DimensionUnitPoints:
		return fmt.Sprintf("%.2fpt", d.Value)
	case DimensionUnitPercentage:
		return fmt.Sprintf("%.2f%%", d.Value)
	case DimensionUnitMinContent:
		return "min-content"
	case DimensionUnitMaxContent:
		return "max-content"
	case DimensionUnitFitContent:
		return fmt.Sprintf("fit-content(%.2fpt)", d.Value)
	case DimensionUnitFitContentStretch:
		return "fit-content"
	default:
		return "unknown"
	}
}

// InheritedComputedStyle returns a new ComputedStyle that carries only the CSS-inherited
// properties from the receiver. Non-inherited properties (display, position, float,
// overflow, dimensions, margins, padding, borders, flex/grid, z-index, opacity,
// background) are reset to their CSS initial values.
//
// Returns the inherited-only ComputedStyle.
func (s *ComputedStyle) InheritedComputedStyle() ComputedStyle {
	result := DefaultComputedStyle()

	result.CustomProperties = s.CustomProperties
	result.FontFamily = s.FontFamily
	result.FontSize = s.FontSize
	result.FontWeight = s.FontWeight
	result.FontStyle = s.FontStyle
	result.Colour = s.Colour
	result.LineHeight = s.LineHeight
	result.LineHeightAuto = s.LineHeightAuto
	result.LetterSpacing = s.LetterSpacing
	result.WordSpacing = s.WordSpacing
	result.TextAlign = s.TextAlign
	result.TextIndent = s.TextIndent
	result.TextDecoration = s.TextDecoration
	result.TextTransform = s.TextTransform
	result.WhiteSpace = s.WhiteSpace
	result.WordBreak = s.WordBreak
	result.OverflowWrap = s.OverflowWrap
	result.Visibility = s.Visibility
	result.WritingMode = s.WritingMode
	result.ListStyleType = s.ListStyleType
	result.ListStylePosition = s.ListStylePosition
	result.BorderCollapse = s.BorderCollapse
	result.BorderSpacing = s.BorderSpacing
	result.CaptionSide = s.CaptionSide
	result.Direction = s.Direction
	result.Hyphens = s.Hyphens
	result.TabSize = s.TabSize
	result.TabStops = s.TabStops
	result.TextShadow = s.TextShadow
	result.Orphans = s.Orphans
	result.Widows = s.Widows

	return result
}

// DimensionAuto returns a Dimension representing the CSS "auto" value.
//
// Returns the auto Dimension.
func DimensionAuto() Dimension {
	return Dimension{}
}

// DimensionPt returns a Dimension with an absolute value in points.
//
// Takes value (float64) which is the length in points.
//
// Returns the point-valued Dimension.
func DimensionPt(value float64) Dimension {
	return Dimension{Value: value, Unit: DimensionUnitPoints}
}

// DimensionPct returns a Dimension with a percentage value.
//
// Takes value (float64) which is the percentage.
//
// Returns the percentage-valued Dimension.
func DimensionPct(value float64) Dimension {
	return Dimension{Value: value, Unit: DimensionUnitPercentage}
}

// DimensionMinContent returns a Dimension representing the CSS "min-content" keyword.
//
// Returns the min-content Dimension.
func DimensionMinContent() Dimension {
	return Dimension{Unit: DimensionUnitMinContent, Value: 0}
}

// DimensionMaxContent returns a Dimension representing the CSS "max-content" keyword.
//
// Returns the max-content Dimension.
func DimensionMaxContent() Dimension {
	return Dimension{Unit: DimensionUnitMaxContent, Value: 0}
}

// DimensionFitContent returns a Dimension representing the CSS fit-content(<argument>)
// function with a resolved point value as the argument.
//
// Takes argument (float64) which specifies the clamp limit in points.
//
// Returns the fit-content Dimension.
func DimensionFitContent(argument float64) Dimension {
	return Dimension{Value: argument, Unit: DimensionUnitFitContent}
}

// DimensionFitContentStretch returns a Dimension representing the bare CSS "fit-content"
// keyword. The available width is used as the clamp argument at resolution time.
//
// Returns the fit-content-stretch Dimension.
func DimensionFitContentStretch() Dimension {
	return Dimension{Unit: DimensionUnitFitContentStretch, Value: 0}
}

// DefaultGridLine returns a GridLine with auto-placement.
//
// Returns the auto-placed GridLine.
func DefaultGridLine() GridLine {
	return GridLine{IsAuto: true, Line: 0, Span: 0}
}

// DefaultComputedStyle returns a ComputedStyle with all properties set to their CSS
// initial values.
//
// Returns the default ComputedStyle.
func DefaultComputedStyle() ComputedStyle {
	return initialComputedStyle
}

// newInitialComputedStyle assembles a ComputedStyle from the initial values of each
// embedded value group.
//
// Returns ComputedStyle which holds every property at its CSS initial value.
func newInitialComputedStyle() ComputedStyle {
	return ComputedStyle{
		styleReferenceValues: newStyleReferenceValues(),
		styleColourValues:    newStyleColourValues(),
		styleGeometryValues:  newStyleGeometryValues(),
		styleNumericValues:   newStyleNumericValues(),
		styleKeywordValues:   newStyleKeywordValues(),
		styleFlagValues:      newStyleFlagValues(),
	}
}
