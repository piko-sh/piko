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

// styleReferenceValues groups the ComputedStyle properties whose values hold heap
// references, including pointers, maps, strings (16 bytes, ptrdata 8) and slices (24
// bytes, ptrdata 8). ComputedStyle embeds this group first so that every pointer-bearing
// word sits at the front and the GC pointer-scan window stays as small as possible.
type styleReferenceValues struct {
	// GridAutoRepeatColumns holds the deferred auto-fill or auto-fit repeat pattern for grid
	// columns.
	GridAutoRepeatColumns *GridAutoRepeat

	// GridAutoRepeatRows holds the deferred auto-fill or auto-fit repeat pattern for grid
	// rows.
	GridAutoRepeatRows *GridAutoRepeat

	// CustomProperties holds the element's CSS custom property values keyed by property
	// name.
	CustomProperties map[string]string

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

	// Language holds the element's language tag for hyphenation and text processing.
	Language string

	// TransformValue holds the resolved CSS transform function list as a string.
	TransformValue string

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

	// GridTemplateColumns holds the resolved CSS grid-template-columns track list.
	GridTemplateColumns []GridTrack

	// GridTemplateRows holds the resolved CSS grid-template-rows track list.
	GridTemplateRows []GridTrack

	// GridAutoColumns holds the resolved CSS grid-auto-columns track sizes.
	GridAutoColumns []GridTrack

	// GridAutoRows holds the resolved CSS grid-auto-rows track sizes.
	GridAutoRows []GridTrack

	// GridTemplateAreas holds the resolved CSS grid-template-areas as a row-major string
	// grid.
	GridTemplateAreas [][]string

	// BgImages holds the resolved CSS background-image layer list.
	BgImages []BackgroundImage

	// TabStops holds the custom tab stop positions for text layout.
	TabStops []TabStop
}

// styleColourValues groups the ComputedStyle colour properties. Each Colour is 72 bytes
// of float64 and int components with no pointers.
type styleColourValues struct {
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
}

// styleGeometryValues groups the ComputedStyle grid placement lines (24 bytes each) and
// CSS dimensions (16 bytes each) that size and offset the box. None hold pointers.
type styleGeometryValues struct {
	// GridColumnStart holds the resolved CSS grid-column-start placement.
	GridColumnStart GridLine

	// GridColumnEnd holds the resolved CSS grid-column-end placement.
	GridColumnEnd GridLine

	// GridRowStart holds the resolved CSS grid-row-start placement.
	GridRowStart GridLine

	// GridRowEnd holds the resolved CSS grid-row-end placement.
	GridRowEnd GridLine

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
}

// styleNumericValues groups numeric ComputedStyle properties, holding lengths in points,
// factors and ratios as float64, and counts and orders as int.
type styleNumericValues struct {
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
}

// styleKeywordValues groups the ComputedStyle properties resolved to CSS keywords. Each
// keyword is an int-based enum type with no pointers.
type styleKeywordValues struct {
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
}

// styleFlagValues groups the ComputedStyle boolean flags. ComputedStyle embeds this group
// last so the one-byte fields share a single trailing padding word.
type styleFlagValues struct {
	// HasTransform indicates whether the element has a CSS transform applied.
	HasTransform bool

	// TextDecorationColourSet indicates whether the text-decoration-color was explicitly set
	// rather than inherited from the colour property.
	TextDecorationColourSet bool

	// LineHeightAuto indicates whether the line-height is set to the CSS "normal" (auto)
	// value.
	LineHeightAuto bool

	// ZIndexAuto indicates whether the z-index is set to the CSS "auto" value.
	ZIndexAuto bool

	// AspectRatioAuto indicates whether the aspect-ratio is set to the CSS "auto" value.
	AspectRatioAuto bool
}

// newStyleReferenceValues returns the reference-valued properties set to their CSS
// initial values.
//
// Returns styleReferenceValues which holds the initial font family and transform origin,
// with every other reference left empty.
func newStyleReferenceValues() styleReferenceValues {
	values := styleReferenceValues{}
	values.FontFamily = "serif"
	values.TransformOrigin = "50% 50%"
	return values
}

// newStyleColourValues returns the colour properties set to their CSS initial values.
//
// Returns styleColourValues which holds a black foreground and a transparent background,
// with every other colour left as the zero Colour.
func newStyleColourValues() styleColourValues {
	values := styleColourValues{}
	values.BackgroundColour = ColourTransparent
	values.Colour = ColourBlack
	return values
}

// newStyleGeometryValues returns the placement and dimension properties set to their CSS
// initial values.
//
// Returns styleGeometryValues which holds auto grid placement, auto sizes and offsets,
// zero minimum sizes and zero margins.
func newStyleGeometryValues() styleGeometryValues {
	return styleGeometryValues{
		GridColumnStart: DefaultGridLine(),
		GridColumnEnd:   DefaultGridLine(),
		GridRowStart:    DefaultGridLine(),
		GridRowEnd:      DefaultGridLine(),
		Height:          DimensionAuto(),
		MaxHeight:       DimensionAuto(),
		Width:           DimensionAuto(),
		Right:           DimensionAuto(),
		MinWidth:        DimensionPt(0),
		MinHeight:       DimensionPt(0),
		MaxWidth:        DimensionAuto(),
		Top:             DimensionAuto(),
		MarginTop:       DimensionPt(0),
		MarginRight:     DimensionPt(0),
		MarginBottom:    DimensionPt(0),
		MarginLeft:      DimensionPt(0),
		Bottom:          DimensionAuto(),
		Left:            DimensionAuto(),
		FlexBasis:       DimensionAuto(),
		ColumnWidth:     DimensionAuto(),
	}
}

// newStyleNumericValues returns the numeric properties set to their CSS initial values.
//
// Returns styleNumericValues which holds full opacity, the default font size, weight,
// line height and tab size, a flex-shrink of one and two widows and orphans, with every
// other number zero.
func newStyleNumericValues() styleNumericValues {
	values := styleNumericValues{}
	values.Opacity = 1.0
	values.FontSize = defaultFontSizePt
	values.LineHeight = defaultLineHeightMultiplier * defaultFontSizePt
	values.FlexShrink = 1
	values.TabSize = defaultTabSize
	values.FontWeight = defaultFontWeight
	values.Widows = 2
	values.Orphans = 2
	return values
}

// newStyleKeywordValues returns the keyword properties set to their CSS initial values.
//
// Returns styleKeywordValues which holds the initial keyword for every property, starting
// from a visible, statically positioned inline box.
func newStyleKeywordValues() styleKeywordValues {
	return styleKeywordValues{
		Visibility:          VisibilityVisible,
		ObjectFit:           ObjectFitFill,
		Display:             DisplayInline,
		Position:            PositionStatic,
		BoxSizing:           BoxSizingContentBox,
		Float:               FloatNone,
		Clear:               ClearNone,
		OverflowX:           OverflowVisible,
		OverflowY:           OverflowVisible,
		FontStyle:           FontStyleNormal,
		TextAlign:           TextAlignStart,
		TextDecoration:      TextDecorationNone,
		TextDecorationStyle: TextDecorationStyleSolid,
		TextRenderingMode:   TextRenderFill,
		TextTransform:       TextTransformNone,
		WhiteSpace:          WhiteSpaceNormal,
		WordBreak:           WordBreakNormal,
		OverflowWrap:        OverflowWrapNormal,
		BorderTopStyle:      BorderStyleNone,
		BorderRightStyle:    BorderStyleNone,
		BorderBottomStyle:   BorderStyleNone,
		BorderLeftStyle:     BorderStyleNone,
		FlexDirection:       FlexDirectionRow,
		FlexWrap:            FlexWrapNowrap,
		JustifyContent:      JustifyFlexStart,
		AlignItems:          AlignItemsStretch,
		AlignSelf:           AlignSelfAuto,
		AlignContent:        AlignContentStretch,
		JustifyItems:        JustifyItemsStretch,
		JustifySelf:         JustifySelfAuto,
		TableLayout:         TableLayoutAuto,
		BorderCollapse:      BorderCollapseSeparate,
		CaptionSide:         CaptionSideTop,
		VerticalAlign:       VerticalAlignBaseline,
		ListStyleType:       ListStyleTypeDisc,
		ListStylePosition:   ListStylePositionOutside,
		WritingMode:         WritingModeHorizontalTB,
		Direction:           DirectionLTR,
		UnicodeBidi:         UnicodeBidiNormal,
		Hyphens:             HyphensManual,
		OutlineStyle:        BorderStyleNone,
		BorderImageRepeat:   BorderImageRepeatStretch,
		PageBreakBefore:     PageBreakAuto,
		PageBreakAfter:      PageBreakAuto,
		PageBreakInside:     PageBreakAuto,
		GridAutoFlow:        GridAutoFlowRow,
		TextOverflow:        TextOverflowClip,
		ColumnFill:          ColumnFillBalance,
		ColumnRuleStyle:     BorderStyleNone,
		ColumnSpan:          ColumnSpanNone,
		MixBlendMode:        BlendModeNormal,
	}
}

// newStyleFlagValues returns the boolean flags set to their CSS initial values.
//
// Returns styleFlagValues which marks line-height and z-index as auto, with every other
// flag cleared.
func newStyleFlagValues() styleFlagValues {
	return styleFlagValues{
		HasTransform:            false,
		TextDecorationColourSet: false,
		LineHeightAuto:          true,
		ZIndexAuto:              true,
		AspectRatioAuto:         false,
	}
}
