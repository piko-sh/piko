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

// FlexDirectionType represents the CSS flex-direction property.
type FlexDirectionType int

const (
	// FlexDirectionRow represents CSS flex-direction: row.
	FlexDirectionRow FlexDirectionType = iota

	// FlexDirectionRowReverse represents CSS flex-direction: row-reverse.
	FlexDirectionRowReverse

	// FlexDirectionColumn represents CSS flex-direction: column.
	FlexDirectionColumn

	// FlexDirectionColumnReverse represents CSS flex-direction: column-reverse.
	FlexDirectionColumnReverse
)

// FlexWrapType represents the CSS flex-wrap property.
type FlexWrapType int

const (
	// FlexWrapNowrap represents CSS flex-wrap: nowrap.
	FlexWrapNowrap FlexWrapType = iota

	// FlexWrapWrap represents CSS flex-wrap: wrap.
	FlexWrapWrap

	// FlexWrapWrapReverse represents CSS flex-wrap: wrap-reverse.
	FlexWrapWrapReverse
)

// JustifyContentType represents the CSS justify-content property.
type JustifyContentType int

const (
	// JustifyFlexStart represents CSS justify-content: flex-start.
	JustifyFlexStart JustifyContentType = iota

	// JustifyFlexEnd represents CSS justify-content: flex-end.
	JustifyFlexEnd

	// JustifyCentre represents CSS justify-content: centre.
	JustifyCentre

	// JustifySpaceBetween represents CSS justify-content: space-between.
	JustifySpaceBetween

	// JustifySpaceAround represents CSS justify-content: space-around.
	JustifySpaceAround

	// JustifySpaceEvenly represents CSS justify-content: space-evenly.
	JustifySpaceEvenly
)

// AlignItemsType represents the CSS align-items property.
type AlignItemsType int

const (
	// AlignItemsStretch represents CSS align-items: stretch.
	AlignItemsStretch AlignItemsType = iota

	// AlignItemsFlexStart represents CSS align-items: flex-start.
	AlignItemsFlexStart

	// AlignItemsFlexEnd represents CSS align-items: flex-end.
	AlignItemsFlexEnd

	// AlignItemsCentre represents CSS align-items: centre.
	AlignItemsCentre

	// AlignItemsBaseline represents CSS align-items: baseline.
	AlignItemsBaseline
)

// AlignSelfType represents the CSS align-self property.
type AlignSelfType int

const (
	// AlignSelfAuto represents CSS align-self: auto.
	AlignSelfAuto AlignSelfType = iota

	// AlignSelfFlexStart represents CSS align-self: flex-start.
	AlignSelfFlexStart

	// AlignSelfFlexEnd represents CSS align-self: flex-end.
	AlignSelfFlexEnd

	// AlignSelfCentre represents CSS align-self: centre.
	AlignSelfCentre

	// AlignSelfBaseline represents CSS align-self: baseline.
	AlignSelfBaseline

	// AlignSelfStretch represents CSS align-self: stretch.
	AlignSelfStretch
)

// AlignContentType represents the CSS align-content property.
type AlignContentType int

const (
	// AlignContentStretch represents CSS align-content: stretch.
	AlignContentStretch AlignContentType = iota

	// AlignContentFlexStart represents CSS align-content: flex-start.
	AlignContentFlexStart

	// AlignContentFlexEnd represents CSS align-content: flex-end.
	AlignContentFlexEnd

	// AlignContentCentre represents CSS align-content: centre.
	AlignContentCentre

	// AlignContentSpaceBetween represents CSS align-content: space-between.
	AlignContentSpaceBetween

	// AlignContentSpaceAround represents CSS align-content: space-around.
	AlignContentSpaceAround
)

// JustifyItemsType represents the CSS justify-items property
// for grid containers.
type JustifyItemsType int

const (
	// JustifyItemsStretch represents CSS justify-items: stretch.
	JustifyItemsStretch JustifyItemsType = iota

	// JustifyItemsStart represents CSS justify-items: start.
	JustifyItemsStart

	// JustifyItemsEnd represents CSS justify-items: end.
	JustifyItemsEnd

	// JustifyItemsCentre represents CSS justify-items: centre.
	JustifyItemsCentre
)

// JustifySelfType represents the CSS justify-self property
// for grid items.
type JustifySelfType int

const (
	// JustifySelfAuto represents CSS justify-self: auto.
	JustifySelfAuto JustifySelfType = iota

	// JustifySelfStretch represents CSS justify-self: stretch.
	JustifySelfStretch

	// JustifySelfStart represents CSS justify-self: start.
	JustifySelfStart

	// JustifySelfEnd represents CSS justify-self: end.
	JustifySelfEnd

	// JustifySelfCentre represents CSS justify-self: centre.
	JustifySelfCentre
)

// GridAutoFlowType represents the CSS grid-auto-flow property.
type GridAutoFlowType int

const (
	// GridAutoFlowRow places items row by row (default).
	GridAutoFlowRow GridAutoFlowType = iota

	// GridAutoFlowColumn places items column by column.
	GridAutoFlowColumn

	// GridAutoFlowRowDense places items row by row, filling gaps.
	GridAutoFlowRowDense

	// GridAutoFlowColumnDense places items column by column, filling gaps.
	GridAutoFlowColumnDense
)

// GridAutoRepeatType distinguishes between auto-fill and auto-fit
// repetition in CSS grid track lists.
type GridAutoRepeatType int

const (
	// GridAutoRepeatFill creates as many tracks as fit in the
	// container, leaving empty tracks at their resolved size.
	GridAutoRepeatFill GridAutoRepeatType = iota

	// GridAutoRepeatFit creates tracks like auto-fill but
	// collapses empty tracks to zero width.
	GridAutoRepeatFit
)

// PageBreakType represents the CSS page-break-before/after/inside property.
type PageBreakType int

const (
	// PageBreakAuto represents CSS page-break: auto.
	PageBreakAuto PageBreakType = iota

	// PageBreakAlways represents CSS page-break: always.
	PageBreakAlways

	// PageBreakAvoid represents CSS page-break: avoid.
	PageBreakAvoid

	// PageBreakLeft represents CSS page-break: left.
	PageBreakLeft

	// PageBreakRight represents CSS page-break: right.
	PageBreakRight
)

// WritingModeType represents the CSS writing-mode property.
type WritingModeType int

const (
	// WritingModeHorizontalTB represents CSS writing-mode: horizontal-tb.
	WritingModeHorizontalTB WritingModeType = iota

	// WritingModeVerticalRL represents CSS writing-mode: vertical-rl.
	WritingModeVerticalRL

	// WritingModeVerticalLR represents CSS writing-mode: vertical-lr.
	WritingModeVerticalLR
)

// ColumnSpanType represents the CSS column-span property.
type ColumnSpanType int

const (
	// ColumnSpanNone means the element does not span columns.
	ColumnSpanNone ColumnSpanType = iota

	// ColumnSpanAll means the element spans all columns.
	ColumnSpanAll
)

// CaptionSideType represents the CSS caption-side property.
type CaptionSideType int

const (
	// CaptionSideTop represents CSS caption-side: top.
	CaptionSideTop CaptionSideType = iota

	// CaptionSideBottom represents CSS caption-side: bottom.
	CaptionSideBottom
)

// ListStylePositionType represents the CSS list-style-position property.
type ListStylePositionType int

const (
	// ListStylePositionOutside represents CSS list-style-position: outside.
	ListStylePositionOutside ListStylePositionType = iota

	// ListStylePositionInside represents CSS list-style-position: inside.
	ListStylePositionInside
)

// VerticalAlignType represents the CSS vertical-align property for table
// cells.
type VerticalAlignType int

const (
	// VerticalAlignBaseline represents CSS vertical-align: baseline.
	VerticalAlignBaseline VerticalAlignType = iota

	// VerticalAlignTop represents CSS vertical-align: top.
	VerticalAlignTop

	// VerticalAlignMiddle represents CSS vertical-align: middle.
	VerticalAlignMiddle

	// VerticalAlignBottom represents CSS vertical-align: bottom.
	VerticalAlignBottom

	// VerticalAlignSuper represents CSS vertical-align: super.
	VerticalAlignSuper

	// VerticalAlignSub represents CSS vertical-align: sub.
	VerticalAlignSub

	// VerticalAlignTextTop represents CSS vertical-align: text-top.
	VerticalAlignTextTop

	// VerticalAlignTextBottom represents CSS vertical-align: text-bottom.
	VerticalAlignTextBottom
)

// TableLayoutType represents the CSS table-layout property.
type TableLayoutType int

const (
	// TableLayoutAuto represents CSS table-layout: auto.
	TableLayoutAuto TableLayoutType = iota

	// TableLayoutFixed represents CSS table-layout: fixed.
	TableLayoutFixed
)

// BorderCollapseType represents the CSS border-collapse property.
type BorderCollapseType int

const (
	// BorderCollapseSeparate represents CSS border-collapse: separate.
	BorderCollapseSeparate BorderCollapseType = iota

	// BorderCollapseCollapse represents CSS border-collapse: collapse.
	BorderCollapseCollapse
)

// ListStyleType represents the CSS list-style-type property.
type ListStyleType int

const (
	// ListStyleTypeDisc represents CSS list-style-type: disc.
	ListStyleTypeDisc ListStyleType = iota

	// ListStyleTypeCircle represents CSS list-style-type: circle.
	ListStyleTypeCircle

	// ListStyleTypeSquare represents CSS list-style-type: square.
	ListStyleTypeSquare

	// ListStyleTypeDecimal represents CSS list-style-type: decimal.
	ListStyleTypeDecimal

	// ListStyleTypeNone represents CSS list-style-type: none.
	ListStyleTypeNone
)

// TextOverflowType represents the CSS text-overflow property.
type TextOverflowType int

const (
	// TextOverflowClip clips overflowing text (default).
	TextOverflowClip TextOverflowType = iota

	// TextOverflowEllipsis renders an ellipsis for overflowing text.
	TextOverflowEllipsis
)

// ColumnFillType represents the CSS column-fill property.
type ColumnFillType int

const (
	// ColumnFillBalance distributes content equally across columns.
	ColumnFillBalance ColumnFillType = iota

	// ColumnFillAuto fills columns sequentially.
	ColumnFillAuto
)

var flexDirectionTypeNames = [...]string{
	FlexDirectionRow:           "row",
	FlexDirectionRowReverse:    "row-reverse",
	FlexDirectionColumn:        "column",
	FlexDirectionColumnReverse: "column-reverse",
}

var flexWrapTypeNames = [...]string{
	FlexWrapNowrap:      "nowrap",
	FlexWrapWrap:        "wrap",
	FlexWrapWrapReverse: "wrap-reverse",
}

var justifyContentTypeNames = [...]string{
	JustifyFlexStart:    cssKeywordFlexStart,
	JustifyFlexEnd:      cssKeywordFlexEnd,
	JustifyCentre:       cssKeywordCentre,
	JustifySpaceBetween: "space-between",
	JustifySpaceAround:  "space-around",
	JustifySpaceEvenly:  "space-evenly",
}

var alignItemsTypeNames = [...]string{
	AlignItemsStretch:   "stretch",
	AlignItemsFlexStart: cssKeywordFlexStart,
	AlignItemsFlexEnd:   cssKeywordFlexEnd,
	AlignItemsCentre:    cssKeywordCentre,
	AlignItemsBaseline:  "baseline",
}

var alignSelfTypeNames = [...]string{
	AlignSelfAuto:      cssKeywordAuto,
	AlignSelfFlexStart: cssKeywordFlexStart,
	AlignSelfFlexEnd:   cssKeywordFlexEnd,
	AlignSelfCentre:    cssKeywordCentre,
	AlignSelfBaseline:  "baseline",
	AlignSelfStretch:   "stretch",
}

var alignContentTypeNames = [...]string{
	AlignContentStretch:      "stretch",
	AlignContentFlexStart:    cssKeywordFlexStart,
	AlignContentFlexEnd:      cssKeywordFlexEnd,
	AlignContentCentre:       cssKeywordCentre,
	AlignContentSpaceBetween: "space-between",
	AlignContentSpaceAround:  "space-around",
}

var justifyItemsTypeNames = [...]string{
	JustifyItemsStretch: "stretch",
	JustifyItemsStart:   "start",
	JustifyItemsEnd:     "end",
	JustifyItemsCentre:  cssKeywordCentre,
}

var justifySelfTypeNames = [...]string{
	JustifySelfAuto:    cssKeywordAuto,
	JustifySelfStretch: "stretch",
	JustifySelfStart:   "start",
	JustifySelfEnd:     "end",
	JustifySelfCentre:  cssKeywordCentre,
}

var gridAutoFlowTypeNames = [...]string{
	GridAutoFlowRow:         "row",
	GridAutoFlowColumn:      "column",
	GridAutoFlowRowDense:    "row dense",
	GridAutoFlowColumnDense: "column dense",
}

var pageBreakTypeNames = [...]string{
	PageBreakAuto:   cssKeywordAuto,
	PageBreakAlways: "always",
	PageBreakAvoid:  "avoid",
	PageBreakLeft:   cssKeywordLeft,
	PageBreakRight:  cssKeywordRight,
}

var writingModeTypeNames = [...]string{
	WritingModeHorizontalTB: "horizontal-tb",
	WritingModeVerticalRL:   "vertical-rl",
	WritingModeVerticalLR:   "vertical-lr",
}

var columnSpanTypeNames = [...]string{
	ColumnSpanNone: "none",
	ColumnSpanAll:  "all",
}

var captionSideTypeNames = [...]string{
	CaptionSideTop:    "top",
	CaptionSideBottom: "bottom",
}

var listStylePositionTypeNames = [...]string{
	ListStylePositionOutside: "outside",
	ListStylePositionInside:  "inside",
}

var verticalAlignTypeNames = [...]string{
	VerticalAlignBaseline:   "baseline",
	VerticalAlignTop:        "top",
	VerticalAlignMiddle:     "middle",
	VerticalAlignBottom:     "bottom",
	VerticalAlignSuper:      "super",
	VerticalAlignSub:        "sub",
	VerticalAlignTextTop:    "text-top",
	VerticalAlignTextBottom: "text-bottom",
}

var tableLayoutTypeNames = [...]string{
	TableLayoutAuto:  cssKeywordAuto,
	TableLayoutFixed: "fixed",
}

var borderCollapseTypeNames = [...]string{
	BorderCollapseSeparate: "separate",
	BorderCollapseCollapse: "collapse",
}

var listStyleTypeNames = [...]string{
	ListStyleTypeDisc:    "disc",
	ListStyleTypeCircle:  "circle",
	ListStyleTypeSquare:  "square",
	ListStyleTypeDecimal: "decimal",
	ListStyleTypeNone:    cssKeywordNone,
}

var textOverflowTypeNames = [...]string{
	TextOverflowClip:     "clip",
	TextOverflowEllipsis: "ellipsis",
}

var columnFillTypeNames = [...]string{
	ColumnFillBalance: "balance",
	ColumnFillAuto:    "auto",
}

// String returns the CSS keyword for this flex-direction type.
//
// Returns string which is the CSS keyword.
func (f FlexDirectionType) String() string {
	if int(f) < len(flexDirectionTypeNames) {
		return flexDirectionTypeNames[f]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this flex-wrap type.
//
// Returns string which is the CSS keyword.
func (f FlexWrapType) String() string {
	if int(f) < len(flexWrapTypeNames) {
		return flexWrapTypeNames[f]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this justify-content type.
//
// Returns string which is the CSS keyword.
func (j JustifyContentType) String() string {
	if int(j) < len(justifyContentTypeNames) {
		return justifyContentTypeNames[j]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this align-items type.
//
// Returns string which is the CSS keyword.
func (a AlignItemsType) String() string {
	if int(a) < len(alignItemsTypeNames) {
		return alignItemsTypeNames[a]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this align-self type.
//
// Returns string which is the CSS keyword.
func (a AlignSelfType) String() string {
	if int(a) < len(alignSelfTypeNames) {
		return alignSelfTypeNames[a]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this align-content type.
//
// Returns string which is the CSS keyword.
func (a AlignContentType) String() string {
	if int(a) < len(alignContentTypeNames) {
		return alignContentTypeNames[a]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this justify-items type.
//
// Returns string which is the CSS keyword.
func (j JustifyItemsType) String() string {
	if int(j) < len(justifyItemsTypeNames) {
		return justifyItemsTypeNames[j]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this justify-self type.
//
// Returns string which is the CSS keyword.
func (j JustifySelfType) String() string {
	if int(j) < len(justifySelfTypeNames) {
		return justifySelfTypeNames[j]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this grid-auto-flow type.
//
// Returns string which is the CSS keyword.
func (g GridAutoFlowType) String() string {
	if int(g) < len(gridAutoFlowTypeNames) {
		return gridAutoFlowTypeNames[g]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this page-break type.
//
// Returns string which is the CSS keyword.
func (p PageBreakType) String() string {
	if int(p) < len(pageBreakTypeNames) {
		return pageBreakTypeNames[p]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this writing-mode type.
//
// Returns string which is the CSS keyword.
func (w WritingModeType) String() string {
	if int(w) < len(writingModeTypeNames) {
		return writingModeTypeNames[w]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this column-span type.
//
// Returns string which is the CSS keyword.
func (c ColumnSpanType) String() string {
	if int(c) < len(columnSpanTypeNames) {
		return columnSpanTypeNames[c]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this caption-side type.
//
// Returns string which is the CSS keyword.
func (c CaptionSideType) String() string {
	if int(c) < len(captionSideTypeNames) {
		return captionSideTypeNames[c]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this list-style-position type.
//
// Returns string which is the CSS keyword.
func (l ListStylePositionType) String() string {
	if int(l) < len(listStylePositionTypeNames) {
		return listStylePositionTypeNames[l]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this vertical-align type.
//
// Returns string which is the CSS keyword.
func (v VerticalAlignType) String() string {
	if int(v) < len(verticalAlignTypeNames) {
		return verticalAlignTypeNames[v]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this table-layout type.
//
// Returns string which is the CSS keyword.
func (t TableLayoutType) String() string {
	if int(t) < len(tableLayoutTypeNames) {
		return tableLayoutTypeNames[t]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this border-collapse type.
//
// Returns string which is the CSS keyword.
func (b BorderCollapseType) String() string {
	if int(b) < len(borderCollapseTypeNames) {
		return borderCollapseTypeNames[b]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this list-style-type.
//
// Returns string which is the CSS keyword.
func (l ListStyleType) String() string {
	if int(l) < len(listStyleTypeNames) {
		return listStyleTypeNames[l]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this text-overflow type.
//
// Returns string which is the CSS keyword.
func (t TextOverflowType) String() string {
	if int(t) < len(textOverflowTypeNames) {
		return textOverflowTypeNames[t]
	}
	return cssKeywordUnknown
}

// String returns the CSS keyword for this column-fill type.
//
// Returns string which is the CSS keyword.
func (c ColumnFillType) String() string {
	if int(c) < len(columnFillTypeNames) {
		return columnFillTypeNames[c]
	}
	return cssKeywordUnknown
}
