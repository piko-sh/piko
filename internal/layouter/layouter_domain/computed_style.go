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

import "fmt"

// CounterEntry represents a single counter-reset or
// counter-increment operation.
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

	// DimensionUnitPercentage represents a percentage of the containing
	// block.
	DimensionUnitPercentage

	// DimensionUnitMinContent represents the CSS "min-content" keyword.
	DimensionUnitMinContent

	// DimensionUnitMaxContent represents the CSS "max-content" keyword.
	DimensionUnitMaxContent

	// DimensionUnitFitContent represents the CSS
	// fit-content(<length-percentage>) function. Value holds
	// the argument resolved to points.
	DimensionUnitFitContent

	// DimensionUnitFitContentStretch represents the bare CSS
	// "fit-content" keyword (no argument). At resolution time
	// the available width is used as the clamp argument.
	DimensionUnitFitContentStretch
)

// Dimension represents a CSS length value that may be auto, an absolute
// length in points, or a percentage.
type Dimension struct {
	// Value holds the numeric value. Meaningless when Unit is
	// DimensionUnitAuto.
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

	// GridTrackFitContent represents the fit-content(<length>)
	// function for grid tracks. Value holds the argument in points.
	GridTrackFitContent

	// GridTrackFitContentPct represents the fit-content(<percentage>)
	// function for grid tracks. Value holds the raw percentage.
	GridTrackFitContentPct
)

// GridTrack represents a single track definition in a grid template.
type GridTrack struct {
	// Value holds the numeric value. Meaningless when Unit is
	// GridTrackAuto.
	Value float64

	// Unit identifies how Value should be interpreted.
	Unit GridTrackUnit
}

// GridAutoRepeat stores a deferred auto-fill or auto-fit repeat
// pattern that is expanded at layout time when the container
// width is known.
type GridAutoRepeat struct {
	// Pattern is the track list to repeat.
	Pattern []GridTrack

	// InsertIndex is the position in the fixed template tracks
	// where the expanded pattern should be spliced in.
	InsertIndex int

	// AfterCount is the number of fixed tracks that follow the
	// auto-repeat region in the original track list.
	AfterCount int

	// Type is GridAutoRepeatFill or GridAutoRepeatFit.
	Type GridAutoRepeatType
}

// GridLine represents a grid placement value for an item's start or
// end position.
type GridLine struct {
	// Line is the 1-based grid line number. Zero means auto-placement.
	Line int

	// Span is the number of tracks to span. Zero means no span keyword
	// was used.
	Span int

	// IsAuto indicates whether this is auto-placement.
	IsAuto bool
}

// BoxShadowValue represents a single box-shadow layer. Box-shadow is a visual
// property that does not affect layout; it is stored here for consumption by
// the paint phase.
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

// TextShadowValue represents a single text-shadow layer. Text-shadow is a
// visual property that does not affect layout; it is stored here for
// consumption by the paint phase.
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

	// Position is the stop position as a fraction (0-1).
	// A value of -1 indicates auto-placement.
	Position float64
}

// BackgroundImage represents a parsed CSS background-image value.
type BackgroundImage struct {
	// URL is the image URL for BackgroundImageURL type.
	URL string

	// Stops holds the colour stops for gradient types.
	Stops []GradientStop

	// Angle is the gradient angle in degrees for linear
	// gradients.
	Angle float64

	// Type identifies the kind of background image.
	Type BackgroundImageType

	// Shape is the radial gradient shape (circle or ellipse).
	// Only meaningful when Type is BackgroundImageRadialGradient.
	Shape RadialGradientShape
}

// ComputedStyle holds all resolved CSS properties for a single
// element.
//
// All length values are in points. Percentages are stored as
// Dimension values and resolved during layout when the
// containing block dimensions are known.
//
// Fields are ordered to minimise the GC pointer-scan window.
// All pointer-bearing fields (slices, maps, strings, pointers)
// are grouped first, then non-pointer fields by descending
// alignment.
type ComputedStyle struct {
	// GridAutoRepeatColumns holds the deferred auto-fill or auto-fit
	// repeat pattern for grid columns.
	GridAutoRepeatColumns *GridAutoRepeat

	// GridAutoRepeatRows holds the deferred auto-fill or auto-fit
	// repeat pattern for grid rows.
	GridAutoRepeatRows *GridAutoRepeat

	// CustomProperties holds the element's CSS custom property values
	// keyed by property name.
	CustomProperties map[string]string

	// Strings (16 bytes on 64-bit, ptrdata 8).
	// FontFamily holds the resolved CSS font-family name.
	FontFamily string

	// Content holds the resolved CSS content property value.
	Content string

	// BgSize holds the resolved CSS background-size value.
	BgSize string

	// BgPosition holds the resolved CSS background-position value.
	BgPosition string

	// BgRepeat holds the resolved CSS background-repeat value.
	BgRepeat string

	// BgAttachment holds the resolved CSS background-attachment value.
	BgAttachment string

	// BgOrigin holds the resolved CSS background-origin value.
	BgOrigin string

	// BgClip holds the resolved CSS background-clip value.
	BgClip string

	// ObjectPosition holds the resolved CSS object-position value.
	ObjectPosition string

	// BorderImageSource holds the resolved CSS border-image-source URL.
	BorderImageSource string

	// ClipPath holds the resolved CSS clip-path value.
	ClipPath string

	// MaskImage holds the resolved CSS mask-image value.
	MaskImage string

	// TransformOrigin holds the resolved CSS transform-origin value.
	TransformOrigin string

	// GridArea holds the resolved CSS grid-area shorthand name.
	GridArea string

	// Language holds the element's language tag for hyphenation and
	// text processing.
	Language string

	// TransformValue holds the resolved CSS transform function list
	// as a string.
	TransformValue string

	// Slices (24 bytes on 64-bit, ptrdata 8).
	// Filter holds the resolved CSS filter function list.
	Filter []FilterValue

	// BackdropFilter holds the resolved CSS backdrop-filter function list.
	BackdropFilter []FilterValue

	// BoxShadow holds the resolved CSS box-shadow layer list.
	BoxShadow []BoxShadowValue

	// TextShadow holds the resolved CSS text-shadow layer list.
	TextShadow []TextShadowValue

	// CounterReset holds the resolved CSS counter-reset operations.
	CounterReset []CounterEntry

	// CounterIncrement holds the resolved CSS counter-increment operations.
	CounterIncrement []CounterEntry

	// GridTemplateColumns holds the resolved CSS grid-template-columns
	// track list.
	GridTemplateColumns []GridTrack

	// GridTemplateRows holds the resolved CSS grid-template-rows
	// track list.
	GridTemplateRows []GridTrack

	// GridAutoColumns holds the resolved CSS grid-auto-columns
	// track sizes.
	GridAutoColumns []GridTrack

	// GridAutoRows holds the resolved CSS grid-auto-rows track sizes.
	GridAutoRows []GridTrack

	// GridTemplateAreas holds the resolved CSS grid-template-areas
	// as a row-major string grid.
	GridTemplateAreas [][]string

	// BgImages holds the resolved CSS background-image layer list.
	BgImages []BackgroundImage

	// TabStops holds the custom tab stop positions for text layout.
	TabStops []TabStop

	// Colour values (72 bytes each, contain float64 + int, no pointers).
	// BorderTopColour holds the resolved CSS border-top-color.
	BorderTopColour Colour

	// BorderRightColour holds the resolved CSS border-right-color.
	BorderRightColour Colour

	// BackgroundColour holds the resolved CSS background-color.
	BackgroundColour Colour

	// Colour holds the resolved CSS color (foreground text colour).
	Colour Colour

	// BorderBottomColour holds the resolved CSS border-bottom-color.
	BorderBottomColour Colour

	// BorderLeftColour holds the resolved CSS border-left-color.
	BorderLeftColour Colour

	// OutlineColour holds the resolved CSS outline-color.
	OutlineColour Colour

	// ColumnRuleColour holds the resolved CSS column-rule-color.
	ColumnRuleColour Colour

	// TextDecorationColour holds the resolved CSS text-decoration-color.
	TextDecorationColour Colour

	// TextStrokeColour holds the resolved CSS -webkit-text-stroke-color.
	TextStrokeColour Colour

	// GridLine values (24 bytes each, contain int + bool, no pointers).
	// GridColumnStart holds the resolved CSS grid-column-start placement.
	GridColumnStart GridLine

	// GridColumnEnd holds the resolved CSS grid-column-end placement.
	GridColumnEnd GridLine

	// GridRowStart holds the resolved CSS grid-row-start placement.
	GridRowStart GridLine

	// GridRowEnd holds the resolved CSS grid-row-end placement.
	GridRowEnd GridLine

	// Dimension values (16 bytes each, contain float64 + int, no pointers).
	// Height holds the resolved CSS height.
	Height Dimension

	// MaxHeight holds the resolved CSS max-height.
	MaxHeight Dimension

	// Width holds the resolved CSS width.
	Width Dimension

	// Right holds the resolved CSS right offset for positioned elements.
	Right Dimension

	// MinWidth holds the resolved CSS min-width.
	MinWidth Dimension

	// MinHeight holds the resolved CSS min-height.
	MinHeight Dimension

	// MaxWidth holds the resolved CSS max-width.
	MaxWidth Dimension

	// Top holds the resolved CSS top offset for positioned elements.
	Top Dimension

	// MarginTop holds the resolved CSS margin-top.
	MarginTop Dimension

	// MarginRight holds the resolved CSS margin-right.
	MarginRight Dimension

	// MarginBottom holds the resolved CSS margin-bottom.
	MarginBottom Dimension

	// MarginLeft holds the resolved CSS margin-left.
	MarginLeft Dimension

	// Bottom holds the resolved CSS bottom offset for positioned elements.
	Bottom Dimension

	// Left holds the resolved CSS left offset for positioned elements.
	Left Dimension

	// FlexBasis holds the resolved CSS flex-basis.
	FlexBasis Dimension

	// ColumnWidth holds the resolved CSS column-width.
	ColumnWidth Dimension

	// float64 values (8 bytes each, align 8).
	// Opacity holds the resolved CSS opacity value (0.0 to 1.0).
	Opacity float64

	// PaddingTop holds the resolved CSS padding-top in points.
	PaddingTop float64

	// PaddingRight holds the resolved CSS padding-right in points.
	PaddingRight float64

	// PaddingBottom holds the resolved CSS padding-bottom in points.
	PaddingBottom float64

	// PaddingLeft holds the resolved CSS padding-left in points.
	PaddingLeft float64

	// BorderTopWidth holds the resolved CSS border-top-width in points.
	BorderTopWidth float64

	// BorderRightWidth holds the resolved CSS border-right-width in points.
	BorderRightWidth float64

	// BorderBottomWidth holds the resolved CSS border-bottom-width in points.
	BorderBottomWidth float64

	// BorderLeftWidth holds the resolved CSS border-left-width in points.
	BorderLeftWidth float64

	// BorderTopLeftRadius holds the resolved CSS border-top-left-radius in points.
	BorderTopLeftRadius float64

	// BorderTopRightRadius holds the resolved CSS border-top-right-radius in points.
	BorderTopRightRadius float64

	// BorderBottomRightRadius holds the resolved CSS border-bottom-right-radius in points.
	BorderBottomRightRadius float64

	// BorderBottomLeftRadius holds the resolved CSS border-bottom-left-radius in points.
	BorderBottomLeftRadius float64

	// FontSize holds the resolved CSS font-size in points.
	FontSize float64

	// LineHeight holds the resolved CSS line-height in points.
	LineHeight float64

	// LetterSpacing holds the resolved CSS letter-spacing in points.
	LetterSpacing float64

	// WordSpacing holds the resolved CSS word-spacing in points.
	WordSpacing float64

	// TextIndent holds the resolved CSS text-indent in points.
	TextIndent float64

	// FlexGrow holds the resolved CSS flex-grow factor.
	FlexGrow float64

	// FlexShrink holds the resolved CSS flex-shrink factor.
	FlexShrink float64

	// RowGap holds the resolved CSS row-gap in points.
	RowGap float64

	// ColumnGap holds the resolved CSS column-gap in points.
	ColumnGap float64

	// BorderSpacing holds the resolved CSS border-spacing in points.
	BorderSpacing float64

	// OutlineWidth holds the resolved CSS outline-width in points.
	OutlineWidth float64

	// OutlineOffset holds the resolved CSS outline-offset in points.
	OutlineOffset float64

	// TabSize holds the resolved CSS tab-size in space widths.
	TabSize float64

	// BorderImageSlice holds the resolved CSS border-image-slice value.
	BorderImageSlice float64

	// BorderImageWidth holds the resolved CSS border-image-width value.
	BorderImageWidth float64

	// BorderImageOutset holds the resolved CSS border-image-outset value.
	BorderImageOutset float64

	// AspectRatio holds the resolved CSS aspect-ratio as width divided by height.
	AspectRatio float64

	// ColumnRuleWidth holds the resolved CSS column-rule-width in points.
	ColumnRuleWidth float64

	// TextStrokeWidth holds the resolved CSS -webkit-text-stroke-width in points.
	TextStrokeWidth float64

	// int values (8 bytes on 64-bit, 4 on 32-bit).
	// ZIndex holds the resolved CSS z-index stacking order.
	ZIndex int

	// FontWeight holds the resolved CSS font-weight (100-900).
	FontWeight int

	// Widows holds the resolved CSS widows count for pagination.
	Widows int

	// Order holds the resolved CSS order for flex and grid item ordering.
	Order int

	// Orphans holds the resolved CSS orphans count for pagination.
	Orphans int

	// ColumnCount holds the resolved CSS column-count.
	ColumnCount int

	// int-based enum types (same size as int).
	// Visibility holds the resolved CSS visibility.
	Visibility VisibilityType

	// ObjectFit holds the resolved CSS object-fit mode.
	ObjectFit ObjectFitType

	// Display holds the resolved CSS display type.
	Display DisplayType

	// Position holds the resolved CSS position scheme.
	Position PositionType

	// BoxSizing holds the resolved CSS box-sizing model.
	BoxSizing BoxSizingType

	// Float holds the resolved CSS float direction.
	Float FloatType

	// Clear holds the resolved CSS clear direction.
	Clear ClearType

	// OverflowX holds the resolved CSS overflow-x behaviour.
	OverflowX OverflowType

	// OverflowY holds the resolved CSS overflow-y behaviour.
	OverflowY OverflowType

	// FontStyle holds the resolved CSS font-style.
	FontStyle FontStyle

	// TextAlign holds the resolved CSS text-align direction.
	TextAlign TextAlignType

	// TextDecoration holds the resolved CSS text-decoration-line flags.
	TextDecoration TextDecorationFlag

	// TextDecorationStyle holds the resolved CSS text-decoration-style.
	TextDecorationStyle TextDecorationStyleType

	// TextRenderingMode holds the resolved CSS text-rendering hint.
	TextRenderingMode TextRenderingMode

	// TextTransform holds the resolved CSS text-transform mode.
	TextTransform TextTransformType

	// WhiteSpace holds the resolved CSS white-space handling mode.
	WhiteSpace WhiteSpaceType

	// WordBreak holds the resolved CSS word-break mode.
	WordBreak WordBreakType

	// OverflowWrap holds the resolved CSS overflow-wrap mode.
	OverflowWrap OverflowWrapType

	// BorderTopStyle holds the resolved CSS border-top-style.
	BorderTopStyle BorderStyleType

	// BorderRightStyle holds the resolved CSS border-right-style.
	BorderRightStyle BorderStyleType

	// BorderBottomStyle holds the resolved CSS border-bottom-style.
	BorderBottomStyle BorderStyleType

	// BorderLeftStyle holds the resolved CSS border-left-style.
	BorderLeftStyle BorderStyleType

	// FlexDirection holds the resolved CSS flex-direction.
	FlexDirection FlexDirectionType

	// FlexWrap holds the resolved CSS flex-wrap mode.
	FlexWrap FlexWrapType

	// JustifyContent holds the resolved CSS justify-content alignment.
	JustifyContent JustifyContentType

	// AlignItems holds the resolved CSS align-items alignment.
	AlignItems AlignItemsType

	// AlignSelf holds the resolved CSS align-self override.
	AlignSelf AlignSelfType

	// AlignContent holds the resolved CSS align-content alignment.
	AlignContent AlignContentType

	// JustifyItems holds the resolved CSS justify-items alignment.
	JustifyItems JustifyItemsType

	// JustifySelf holds the resolved CSS justify-self override.
	JustifySelf JustifySelfType

	// TableLayout holds the resolved CSS table-layout algorithm.
	TableLayout TableLayoutType

	// BorderCollapse holds the resolved CSS border-collapse model.
	BorderCollapse BorderCollapseType

	// CaptionSide holds the resolved CSS caption-side placement.
	CaptionSide CaptionSideType

	// VerticalAlign holds the resolved CSS vertical-align mode.
	VerticalAlign VerticalAlignType

	// ListStyleType holds the resolved CSS list-style-type marker.
	ListStyleType ListStyleType

	// ListStylePosition holds the resolved CSS list-style-position.
	ListStylePosition ListStylePositionType

	// WritingMode holds the resolved CSS writing-mode direction.
	WritingMode WritingModeType

	// Direction holds the resolved CSS direction for bidi text.
	Direction DirectionType

	// UnicodeBidi holds the resolved CSS unicode-bidi mode.
	UnicodeBidi UnicodeBidiType

	// Hyphens holds the resolved CSS hyphens mode.
	Hyphens HyphensType

	// OutlineStyle holds the resolved CSS outline-style.
	OutlineStyle BorderStyleType

	// BorderImageRepeat holds the resolved CSS border-image-repeat mode.
	BorderImageRepeat BorderImageRepeatType

	// PageBreakBefore holds the resolved CSS page-break-before mode.
	PageBreakBefore PageBreakType

	// PageBreakAfter holds the resolved CSS page-break-after mode.
	PageBreakAfter PageBreakType

	// PageBreakInside holds the resolved CSS page-break-inside mode.
	PageBreakInside PageBreakType

	// GridAutoFlow holds the resolved CSS grid-auto-flow placement algorithm.
	GridAutoFlow GridAutoFlowType

	// TextOverflow holds the resolved CSS text-overflow mode.
	TextOverflow TextOverflowType

	// ColumnFill holds the resolved CSS column-fill mode.
	ColumnFill ColumnFillType

	// ColumnRuleStyle holds the resolved CSS column-rule-style.
	ColumnRuleStyle BorderStyleType

	// ColumnSpan holds the resolved CSS column-span mode.
	ColumnSpan ColumnSpanType

	// MixBlendMode holds the resolved CSS mix-blend-mode.
	MixBlendMode BlendModeType

	// bool values (1 byte each, grouped at end to minimise padding).
	// HasTransform indicates whether the element has a CSS transform applied.
	HasTransform bool

	// TextDecorationColourSet indicates whether the text-decoration-color
	// was explicitly set rather than inherited from the colour property.
	TextDecorationColourSet bool

	// LineHeightAuto indicates whether the line-height is set to the
	// CSS "normal" (auto) value.
	LineHeightAuto bool

	// ZIndexAuto indicates whether the z-index is set to the CSS "auto" value.
	ZIndexAuto bool

	// AspectRatioAuto indicates whether the aspect-ratio is set to the
	// CSS "auto" value.
	AspectRatioAuto bool
}

// DimensionAuto returns a Dimension representing the CSS "auto"
// value.
//
// Returns the auto Dimension.
func DimensionAuto() Dimension {
	return Dimension{Unit: DimensionUnitAuto}
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

// IsAuto reports whether this dimension represents "auto".
//
// Returns true when the unit is DimensionUnitAuto.
func (d Dimension) IsAuto() bool {
	return d.Unit == DimensionUnitAuto
}

// IsMinContent reports whether this dimension represents
// "min-content".
//
// Returns true when the unit is DimensionUnitMinContent.
func (d Dimension) IsMinContent() bool {
	return d.Unit == DimensionUnitMinContent
}

// IsMaxContent reports whether this dimension represents
// "max-content".
//
// Returns true when the unit is DimensionUnitMaxContent.
func (d Dimension) IsMaxContent() bool {
	return d.Unit == DimensionUnitMaxContent
}

// IsFitContent reports whether this dimension represents a
// fit-content sizing keyword or function.
//
// Returns true for fit-content or fit-content(<arg>).
func (d Dimension) IsFitContent() bool {
	return d.Unit == DimensionUnitFitContent || d.Unit == DimensionUnitFitContentStretch
}

// IsIntrinsic reports whether this dimension represents an
// intrinsic sizing keyword (min-content, max-content, or
// fit-content).
//
// Returns true for any intrinsic sizing keyword.
func (d Dimension) IsIntrinsic() bool {
	return d.IsMinContent() || d.IsMaxContent() || d.IsFitContent()
}

// DimensionMinContent returns a Dimension representing the
// CSS "min-content" keyword.
//
// Returns the min-content Dimension.
func DimensionMinContent() Dimension {
	return Dimension{Unit: DimensionUnitMinContent}
}

// DimensionMaxContent returns a Dimension representing the
// CSS "max-content" keyword.
//
// Returns the max-content Dimension.
func DimensionMaxContent() Dimension {
	return Dimension{Unit: DimensionUnitMaxContent}
}

// DimensionFitContent returns a Dimension representing the
// CSS fit-content(<argument>) function with a resolved point
// value as the argument.
//
// Takes argument (float64) which specifies the clamp limit in points.
//
// Returns the fit-content Dimension.
func DimensionFitContent(argument float64) Dimension {
	return Dimension{Value: argument, Unit: DimensionUnitFitContent}
}

// DimensionFitContentStretch returns a Dimension representing
// the bare CSS "fit-content" keyword. The available width is
// used as the clamp argument at resolution time.
//
// Returns the fit-content-stretch Dimension.
func DimensionFitContentStretch() Dimension {
	return Dimension{Unit: DimensionUnitFitContentStretch}
}

// Resolve returns the absolute value in points.
//
// When the unit is auto, the fallback value is returned.
// When the unit is a percentage, the value is resolved
// against containingBlockSize.
//
// Takes containingBlockSize (float64) which is the size of the
// containing block used to resolve percentages.
//
// Takes fallback (float64) which is the value returned when
// the dimension is auto.
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

// DefaultGridLine returns a GridLine with auto-placement.
//
// Returns the auto-placed GridLine.
func DefaultGridLine() GridLine {
	return GridLine{IsAuto: true}
}

// InheritedComputedStyle returns a new ComputedStyle that carries
// only the CSS-inherited properties from the receiver. Non-inherited
// properties (display, position, float, overflow, dimensions, margins,
// padding, borders, flex/grid, z-index, opacity, background) are
// reset to their CSS initial values.
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

// DefaultComputedStyle returns a ComputedStyle with all properties
// set to their CSS initial values.
//
// Returns the default ComputedStyle.
func DefaultComputedStyle() ComputedStyle {
	style := ComputedStyle{
		Display:      DisplayInline,
		Position:     PositionStatic,
		Float:        FloatNone,
		Clear:        ClearNone,
		Visibility:   VisibilityVisible,
		OverflowX:    OverflowVisible,
		OverflowY:    OverflowVisible,
		ZIndexAuto:   true,
		Width:        DimensionAuto(),
		Height:       DimensionAuto(),
		MinWidth:     DimensionPt(0),
		MinHeight:    DimensionPt(0),
		MaxWidth:     DimensionAuto(),
		MaxHeight:    DimensionAuto(),
		MarginTop:    DimensionPt(0),
		MarginRight:  DimensionPt(0),
		MarginBottom: DimensionPt(0),
		MarginLeft:   DimensionPt(0),
		Top:          DimensionAuto(),
		Right:        DimensionAuto(),
		Bottom:       DimensionAuto(),
		Left:         DimensionAuto(),
	}
	applyDefaultTypography(&style)
	applyDefaultFlexAndLayout(&style)
	return style
}

// applyDefaultTypography sets the initial CSS values for font,
// text, and colour properties on a ComputedStyle.
//
// Takes style (*ComputedStyle) which specifies the style to initialise.
func applyDefaultTypography(style *ComputedStyle) {
	style.FontFamily = "serif"
	style.FontSize = defaultFontSizePt
	style.FontWeight = defaultFontWeight
	style.FontStyle = FontStyleNormal
	style.LineHeight = defaultLineHeightMultiplier * defaultFontSizePt
	style.LineHeightAuto = true
	style.TextAlign = TextAlignStart
	style.WhiteSpace = WhiteSpaceNormal
	style.WordBreak = WordBreakNormal
	style.Colour = ColourBlack
	style.BackgroundColour = ColourTransparent
}

// applyDefaultFlexAndLayout sets the initial CSS values for
// flexbox, grid, table, list, pagination, and visual properties
// on a ComputedStyle.
//
// Takes style (*ComputedStyle) which specifies the style to initialise.
func applyDefaultFlexAndLayout(style *ComputedStyle) {
	style.FlexDirection = FlexDirectionRow
	style.FlexWrap = FlexWrapNowrap
	style.JustifyContent = JustifyFlexStart
	style.AlignItems = AlignItemsStretch
	style.AlignSelf = AlignSelfAuto
	style.AlignContent = AlignContentStretch
	style.FlexGrow = 0
	style.FlexShrink = 1
	style.FlexBasis = DimensionAuto()
	style.TableLayout = TableLayoutAuto
	style.BorderCollapse = BorderCollapseSeparate
	style.CaptionSide = CaptionSideTop
	style.VerticalAlign = VerticalAlignBaseline
	style.ListStyleType = ListStyleTypeDisc
	style.ListStylePosition = ListStylePositionOutside
	style.Direction = DirectionLTR
	style.Hyphens = HyphensManual
	style.TabSize = defaultTabSize
	style.PageBreakBefore = PageBreakAuto
	style.PageBreakAfter = PageBreakAuto
	style.PageBreakInside = PageBreakAuto
	style.Orphans = 2
	style.Widows = 2
	style.Opacity = 1.0
	style.GridColumnStart = DefaultGridLine()
	style.GridColumnEnd = DefaultGridLine()
	style.GridRowStart = DefaultGridLine()
	style.GridRowEnd = DefaultGridLine()
	style.TransformOrigin = "50% 50%"
}
