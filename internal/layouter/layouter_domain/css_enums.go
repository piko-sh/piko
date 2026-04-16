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

const (
	// cssKeywordUnknown is the fallback CSS keyword for unrecognised values.
	cssKeywordUnknown = "unknown"

	// cssKeywordNone is the CSS keyword "none".
	cssKeywordNone = "none"

	// cssKeywordCentre is the CSS keyword "centre".
	cssKeywordCentre = "centre"

	// cssKeywordFlexStart is the CSS keyword "flex-start".
	cssKeywordFlexStart = "flex-start"

	// cssKeywordFlexEnd is the CSS keyword "flex-end".
	cssKeywordFlexEnd = "flex-end"

	// cssKeywordLeft is the CSS keyword "left".
	cssKeywordLeft = "left"

	// cssKeywordRight is the CSS keyword "right".
	cssKeywordRight = "right"

	// cssKeywordAuto is the CSS keyword "auto".
	cssKeywordAuto = "auto"

	// cssKeywordStart is the CSS keyword "start".
	cssKeywordStart = "start"

	// cssKeywordEnd is the CSS keyword "end".
	cssKeywordEnd = "end"
)

// DisplayType represents the CSS display property.
type DisplayType int

const (
	// DisplayBlock represents CSS display: block.
	DisplayBlock DisplayType = iota

	// DisplayInline represents CSS display: inline.
	DisplayInline

	// DisplayInlineBlock represents CSS display: inline-block.
	DisplayInlineBlock

	// DisplayFlex represents CSS display: flex.
	DisplayFlex

	// DisplayInlineFlex represents CSS display: inline-flex.
	DisplayInlineFlex

	// DisplayTable represents CSS display: table.
	DisplayTable

	// DisplayTableRow represents CSS display: table-row.
	DisplayTableRow

	// DisplayTableCell represents CSS display: table-cell.
	DisplayTableCell

	// DisplayTableRowGroup represents CSS display: table-row-group.
	DisplayTableRowGroup

	// DisplayTableHeaderGroup represents CSS display: table-header-group.
	DisplayTableHeaderGroup

	// DisplayTableFooterGroup represents CSS display: table-footer-group.
	DisplayTableFooterGroup

	// DisplayTableCaption represents CSS display: table-caption.
	DisplayTableCaption

	// DisplayListItem represents CSS display: list-item.
	DisplayListItem

	// DisplayGrid represents CSS display: grid.
	DisplayGrid

	// DisplayInlineGrid represents CSS display: inline-grid.
	DisplayInlineGrid

	// DisplayNone represents CSS display: none.
	DisplayNone

	// DisplayContents represents CSS display: contents.
	DisplayContents
)

// PositionType represents the CSS position property.
type PositionType int

const (
	// PositionStatic represents CSS position: static.
	PositionStatic PositionType = iota

	// PositionRelative represents CSS position: relative.
	PositionRelative

	// PositionAbsolute represents CSS position: absolute.
	PositionAbsolute

	// PositionFixed represents CSS position: fixed.
	PositionFixed
)

// FloatType represents the CSS float property.
type FloatType int

const (
	// FloatNone represents CSS float: none.
	FloatNone FloatType = iota

	// FloatLeft represents CSS float: left.
	FloatLeft

	// FloatRight represents CSS float: right.
	FloatRight
)

// ClearType represents the CSS clear property.
type ClearType int

const (
	// ClearNone represents CSS clear: none.
	ClearNone ClearType = iota

	// ClearLeft represents CSS clear: left.
	ClearLeft

	// ClearRight represents CSS clear: right.
	ClearRight

	// ClearBoth represents CSS clear: both.
	ClearBoth
)

// TextAlignType represents the CSS text-align property.
type TextAlignType int

const (
	// TextAlignLeft represents CSS text-align: left.
	TextAlignLeft TextAlignType = iota

	// TextAlignCentre represents CSS text-align: centre.
	TextAlignCentre

	// TextAlignRight represents CSS text-align: right.
	TextAlignRight

	// TextAlignJustify represents CSS text-align: justify.
	TextAlignJustify

	// TextAlignStart represents CSS text-align: start. Resolves
	// to left in LTR and right in RTL.
	TextAlignStart

	// TextAlignEnd represents CSS text-align: end. Resolves to
	// right in LTR and left in RTL.
	TextAlignEnd
)

// TextDecorationFlag represents CSS text-decoration values as a bitmask.
type TextDecorationFlag int

const (
	// TextDecorationNone represents no text decoration.
	TextDecorationNone TextDecorationFlag = 0

	// TextDecorationUnderline represents CSS text-decoration: underline.
	TextDecorationUnderline TextDecorationFlag = 1 << iota

	// TextDecorationOverline represents CSS text-decoration: overline.
	TextDecorationOverline

	// TextDecorationLineThrough represents CSS text-decoration: line-through.
	TextDecorationLineThrough
)

// TextDecorationStyleType represents the CSS text-decoration-style property.
type TextDecorationStyleType int

const (
	// TextDecorationStyleSolid represents CSS text-decoration-style: solid.
	TextDecorationStyleSolid TextDecorationStyleType = iota

	// TextDecorationStyleDashed represents CSS text-decoration-style: dashed.
	TextDecorationStyleDashed

	// TextDecorationStyleDotted represents CSS text-decoration-style: dotted.
	TextDecorationStyleDotted

	// TextDecorationStyleDouble represents CSS text-decoration-style: double.
	TextDecorationStyleDouble

	// TextDecorationStyleWavy represents CSS text-decoration-style: wavy.
	TextDecorationStyleWavy
)

// TextTransformType represents the CSS text-transform property.
type TextTransformType int

const (
	// TextTransformNone represents CSS text-transform: none.
	TextTransformNone TextTransformType = iota

	// TextTransformUppercase represents CSS text-transform: uppercase.
	TextTransformUppercase

	// TextTransformLowercase represents CSS text-transform: lowercase.
	TextTransformLowercase

	// TextTransformCapitalise represents CSS text-transform: capitalise.
	TextTransformCapitalise
)

// WhiteSpaceType represents the CSS white-space property.
type WhiteSpaceType int

const (
	// WhiteSpaceNormal represents CSS white-space: normal.
	WhiteSpaceNormal WhiteSpaceType = iota

	// WhiteSpacePre represents CSS white-space: pre.
	WhiteSpacePre

	// WhiteSpaceNowrap represents CSS white-space: nowrap.
	WhiteSpaceNowrap

	// WhiteSpacePreWrap represents CSS white-space: pre-wrap.
	WhiteSpacePreWrap

	// WhiteSpacePreLine represents CSS white-space: pre-line.
	WhiteSpacePreLine
)

// WordBreakType represents the CSS word-break property.
type WordBreakType int

const (
	// WordBreakNormal represents CSS word-break: normal.
	WordBreakNormal WordBreakType = iota

	// WordBreakBreakAll represents CSS word-break: break-all.
	WordBreakBreakAll

	// WordBreakKeepAll represents CSS word-break: keep-all.
	WordBreakKeepAll
)

// OverflowWrapType represents the CSS overflow-wrap property.
type OverflowWrapType int

const (
	// OverflowWrapNormal represents CSS overflow-wrap: normal.
	OverflowWrapNormal OverflowWrapType = iota

	// OverflowWrapBreakWord represents CSS overflow-wrap: break-word.
	OverflowWrapBreakWord

	// OverflowWrapAnywhere represents CSS overflow-wrap: anywhere.
	OverflowWrapAnywhere
)

// OverflowType represents the CSS overflow property.
type OverflowType int

const (
	// OverflowVisible represents CSS overflow: visible.
	OverflowVisible OverflowType = iota

	// OverflowHidden represents CSS overflow: hidden.
	OverflowHidden

	// OverflowScroll represents CSS overflow: scroll.
	OverflowScroll

	// OverflowAuto represents CSS overflow: auto.
	OverflowAuto
)

// VisibilityType represents the CSS visibility property.
type VisibilityType int

const (
	// VisibilityVisible represents CSS visibility: visible.
	VisibilityVisible VisibilityType = iota

	// VisibilityHidden represents CSS visibility: hidden.
	VisibilityHidden

	// VisibilityCollapse represents CSS visibility: collapse.
	VisibilityCollapse
)

// BorderStyleType represents the CSS border-style property.
type BorderStyleType int

const (
	// BorderStyleNone represents CSS border-style: none.
	BorderStyleNone BorderStyleType = iota

	// BorderStyleSolid represents CSS border-style: solid.
	BorderStyleSolid

	// BorderStyleDashed represents CSS border-style: dashed.
	BorderStyleDashed

	// BorderStyleDotted represents CSS border-style: dotted.
	BorderStyleDotted

	// BorderStyleDouble represents CSS border-style: double.
	BorderStyleDouble

	// BorderStyleGroove represents CSS border-style: groove.
	BorderStyleGroove

	// BorderStyleRidge represents CSS border-style: ridge.
	BorderStyleRidge

	// BorderStyleInset represents CSS border-style: inset.
	BorderStyleInset

	// BorderStyleOutset represents CSS border-style: outset.
	BorderStyleOutset
)

// BoxSizingType represents the CSS box-sizing property.
type BoxSizingType int

const (
	// BoxSizingContentBox means width/height set the content
	// area size; padding and border are added outside.
	BoxSizingContentBox BoxSizingType = iota

	// BoxSizingBorderBox means width/height set the border-box
	// size; padding and border are subtracted to derive the
	// content area.
	BoxSizingBorderBox
)

// DirectionType represents the CSS direction property.
type DirectionType int

const (
	// DirectionLTR represents CSS direction: ltr.
	DirectionLTR DirectionType = iota

	// DirectionRTL represents CSS direction: rtl.
	DirectionRTL
)

// UnicodeBidiType represents the CSS unicode-bidi property.
type UnicodeBidiType int

const (
	// UnicodeBidiNormal represents CSS unicode-bidi: normal.
	UnicodeBidiNormal UnicodeBidiType = iota

	// UnicodeBidiEmbed represents CSS unicode-bidi: embed.
	UnicodeBidiEmbed

	// UnicodeBidiIsolate represents CSS unicode-bidi: isolate.
	UnicodeBidiIsolate

	// UnicodeBidiBidiOverride represents CSS unicode-bidi: bidi-override.
	UnicodeBidiBidiOverride

	// UnicodeBidiIsolateOverride represents CSS unicode-bidi: isolate-override.
	UnicodeBidiIsolateOverride

	// UnicodeBidiPlaintext represents CSS unicode-bidi: plaintext.
	UnicodeBidiPlaintext
)

// HyphensType represents the CSS hyphens property.
type HyphensType int

const (
	// HyphensNone represents CSS hyphens: none.
	HyphensNone HyphensType = iota

	// HyphensManual represents CSS hyphens: manual.
	HyphensManual

	// HyphensAuto represents CSS hyphens: auto.
	HyphensAuto
)

// displayTypeNames maps DisplayType values to their CSS keyword strings.
var displayTypeNames = [...]string{
	DisplayBlock:            "block",
	DisplayInline:           "inline",
	DisplayInlineBlock:      "inline-block",
	DisplayFlex:             "flex",
	DisplayInlineFlex:       "inline-flex",
	DisplayTable:            "table",
	DisplayTableRow:         "table-row",
	DisplayTableCell:        "table-cell",
	DisplayTableRowGroup:    "table-row-group",
	DisplayTableHeaderGroup: "table-header-group",
	DisplayTableFooterGroup: "table-footer-group",
	DisplayTableCaption:     "table-caption",
	DisplayListItem:         "list-item",
	DisplayGrid:             "grid",
	DisplayInlineGrid:       "inline-grid",
	DisplayNone:             cssKeywordNone,
	DisplayContents:         "contents",
}

// positionTypeNames maps PositionType values to their CSS keyword strings.
var positionTypeNames = [...]string{
	PositionStatic:   "static",
	PositionRelative: "relative",
	PositionAbsolute: "absolute",
	PositionFixed:    "fixed",
}

// floatTypeNames maps FloatType values to their CSS keyword strings.
var floatTypeNames = [...]string{
	FloatNone:  cssKeywordNone,
	FloatLeft:  cssKeywordLeft,
	FloatRight: cssKeywordRight,
}

// clearTypeNames maps ClearType values to their CSS keyword strings.
var clearTypeNames = [...]string{
	ClearNone:  cssKeywordNone,
	ClearLeft:  cssKeywordLeft,
	ClearRight: cssKeywordRight,
	ClearBoth:  "both",
}

// textAlignTypeNames maps TextAlignType values to their CSS keyword strings.
var textAlignTypeNames = [...]string{
	TextAlignLeft:    cssKeywordLeft,
	TextAlignCentre:  cssKeywordCentre,
	TextAlignRight:   cssKeywordRight,
	TextAlignJustify: "justify",
	TextAlignStart:   "start",
	TextAlignEnd:     "end",
}

// textDecorationStyleTypeNames maps TextDecorationStyleType
// values to their CSS keyword strings.
var textDecorationStyleTypeNames = [...]string{
	TextDecorationStyleSolid:  "solid",
	TextDecorationStyleDashed: "dashed",
	TextDecorationStyleDotted: "dotted",
	TextDecorationStyleDouble: "double",
	TextDecorationStyleWavy:   "wavy",
}

// textTransformTypeNames maps TextTransformType values to their CSS keyword strings.
var textTransformTypeNames = [...]string{
	TextTransformNone:       cssKeywordNone,
	TextTransformUppercase:  "uppercase",
	TextTransformLowercase:  "lowercase",
	TextTransformCapitalise: "capitalise",
}

// whiteSpaceTypeNames maps WhiteSpaceType values to their CSS keyword strings.
var whiteSpaceTypeNames = [...]string{
	WhiteSpaceNormal:  "normal",
	WhiteSpacePre:     "pre",
	WhiteSpaceNowrap:  "nowrap",
	WhiteSpacePreWrap: "pre-wrap",
	WhiteSpacePreLine: "pre-line",
}

// wordBreakTypeNames maps WordBreakType values to their CSS keyword strings.
var wordBreakTypeNames = [...]string{
	WordBreakNormal:   "normal",
	WordBreakBreakAll: "break-all",
	WordBreakKeepAll:  "keep-all",
}

// overflowWrapTypeNames maps OverflowWrapType values to their CSS keyword strings.
var overflowWrapTypeNames = [...]string{
	OverflowWrapNormal:    "Normal",
	OverflowWrapBreakWord: "BreakWord",
	OverflowWrapAnywhere:  "Anywhere",
}

// overflowTypeNames maps OverflowType values to their CSS keyword strings.
var overflowTypeNames = [...]string{
	OverflowVisible: "visible",
	OverflowHidden:  "hidden",
	OverflowScroll:  "scroll",
	OverflowAuto:    cssKeywordAuto,
}

// visibilityTypeNames maps VisibilityType values to their CSS keyword strings.
var visibilityTypeNames = [...]string{
	VisibilityVisible:  "visible",
	VisibilityHidden:   "hidden",
	VisibilityCollapse: "collapse",
}

// borderStyleTypeNames maps BorderStyleType values to their CSS keyword strings.
var borderStyleTypeNames = [...]string{
	BorderStyleNone:   cssKeywordNone,
	BorderStyleSolid:  "solid",
	BorderStyleDashed: "dashed",
	BorderStyleDotted: "dotted",
	BorderStyleDouble: "double",
	BorderStyleGroove: "groove",
	BorderStyleRidge:  "ridge",
	BorderStyleInset:  "inset",
	BorderStyleOutset: "outset",
}

// boxSizingTypeNames maps BoxSizingType values to their CSS keyword strings.
var boxSizingTypeNames = [...]string{
	BoxSizingContentBox: "content-box",
	BoxSizingBorderBox:  "border-box",
}

// directionTypeNames maps DirectionType values to their CSS keyword strings.
var directionTypeNames = [...]string{
	DirectionLTR: "ltr",
	DirectionRTL: "rtl",
}

// unicodeBidiTypeNames maps UnicodeBidiType values to their CSS keyword strings.
var unicodeBidiTypeNames = [...]string{
	UnicodeBidiNormal:          "normal",
	UnicodeBidiEmbed:           "embed",
	UnicodeBidiIsolate:         "isolate",
	UnicodeBidiBidiOverride:    "bidi-override",
	UnicodeBidiIsolateOverride: "isolate-override",
	UnicodeBidiPlaintext:       "plaintext",
}

// hyphensTypeNames maps HyphensType values to their CSS keyword strings.
var hyphensTypeNames = [...]string{
	HyphensNone:   cssKeywordNone,
	HyphensManual: "manual",
	HyphensAuto:   cssKeywordAuto,
}

// String returns the CSS keyword for this display type.
//
// Returns string which is the CSS keyword.
func (d DisplayType) String() string {
	if int(d) < len(displayTypeNames) {
		return displayTypeNames[d]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this position type.
//
// Returns string which is the CSS keyword.
func (p PositionType) String() string {
	if int(p) < len(positionTypeNames) {
		return positionTypeNames[p]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this float type.
//
// Returns string which is the CSS keyword.
func (f FloatType) String() string {
	if int(f) < len(floatTypeNames) {
		return floatTypeNames[f]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this clear type.
//
// Returns string which is the CSS keyword.
func (c ClearType) String() string {
	if int(c) < len(clearTypeNames) {
		return clearTypeNames[c]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this text-align type.
//
// Returns string which is the CSS keyword.
func (t TextAlignType) String() string {
	if int(t) < len(textAlignTypeNames) {
		return textAlignTypeNames[t]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this text-decoration-style type.
//
// Returns string which is the CSS keyword.
func (t TextDecorationStyleType) String() string {
	if int(t) < len(textDecorationStyleTypeNames) {
		return textDecorationStyleTypeNames[t]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this text-transform type.
//
// Returns string which is the CSS keyword.
func (t TextTransformType) String() string {
	if int(t) < len(textTransformTypeNames) {
		return textTransformTypeNames[t]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this white-space type.
//
// Returns string which is the CSS keyword.
func (w WhiteSpaceType) String() string {
	if int(w) < len(whiteSpaceTypeNames) {
		return whiteSpaceTypeNames[w]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this word-break type.
//
// Returns string which is the CSS keyword.
func (w WordBreakType) String() string {
	if int(w) < len(wordBreakTypeNames) {
		return wordBreakTypeNames[w]
	}
	return cssKeywordUnknown
}

// String returns the Go constant name suffix for this overflow-wrap type.
//
// Returns string which is the constant name suffix.
func (o OverflowWrapType) String() string {
	if int(o) < len(overflowWrapTypeNames) {
		return overflowWrapTypeNames[o]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this overflow type.
//
// Returns string which is the CSS keyword.
func (o OverflowType) String() string {
	if int(o) < len(overflowTypeNames) {
		return overflowTypeNames[o]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this visibility type.
//
// Returns string which is the CSS keyword.
func (v VisibilityType) String() string {
	if int(v) < len(visibilityTypeNames) {
		return visibilityTypeNames[v]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this border-style type.
//
// Returns string which is the CSS keyword.
func (b BorderStyleType) String() string {
	if int(b) < len(borderStyleTypeNames) {
		return borderStyleTypeNames[b]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this box-sizing type.
//
// Returns string which is the CSS keyword.
func (b BoxSizingType) String() string {
	if int(b) < len(boxSizingTypeNames) {
		return boxSizingTypeNames[b]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this direction type.
//
// Returns string which is the CSS keyword.
func (d DirectionType) String() string {
	if int(d) < len(directionTypeNames) {
		return directionTypeNames[d]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this unicode-bidi type.
//
// Returns string which is the CSS keyword.
func (u UnicodeBidiType) String() string {
	if int(u) < len(unicodeBidiTypeNames) {
		return unicodeBidiTypeNames[u]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this hyphens type.
//
// Returns string which is the CSS keyword.
func (h HyphensType) String() string {
	if int(h) < len(hyphensTypeNames) {
		return hyphensTypeNames[h]
	}
	return cssKeywordUnknown
}
