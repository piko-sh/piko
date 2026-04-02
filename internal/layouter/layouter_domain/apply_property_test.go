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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyProperty_GlobalKeywords(t *testing.T) {
	t.Run("inherit copies value from parent", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parent := DefaultComputedStyle()
		parent.Display = DisplayFlex

		applyProperty(&style, "display", "inherit", ctx, &parent)
		assert.Equal(t, DisplayFlex, style.Display)
	})

	t.Run("initial resets to default value", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.Display = DisplayBlock
		ctx := defaultResolutionContext()

		applyProperty(&style, "display", "initial", ctx, nil)
		assert.Equal(t, DisplayInline, style.Display)
	})

	t.Run("unset on inheritable property inherits from parent", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parent := DefaultComputedStyle()
		red, _ := ParseColour("red")
		parent.Colour = red

		applyProperty(&style, "color", "unset", ctx, &parent)
		assert.Equal(t, red, style.Colour)
	})

	t.Run("unset on non-inheritable property resets to initial", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parent := DefaultComputedStyle()
		parent.Display = DisplayFlex

		applyProperty(&style, "display", "unset", ctx, &parent)
		assert.Equal(t, DisplayInline, style.Display)
	})
}

func TestApplyProperty_Display(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected DisplayType
	}{
		{"block", "block", DisplayBlock},
		{"flex", "flex", DisplayFlex},
		{"inline-block", "inline-block", DisplayInlineBlock},
		{"grid", "grid", DisplayGrid},
		{"none", "none", DisplayNone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			style := DefaultComputedStyle()
			ctx := defaultResolutionContext()
			applyProperty(&style, "display", tc.value, ctx, nil)
			assert.Equal(t, tc.expected, style.Display)
		})
	}
}

func TestApplyProperty_Position(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected PositionType
	}{
		{"static", "static", PositionStatic},
		{"relative", "relative", PositionRelative},
		{"absolute", "absolute", PositionAbsolute},
		{"fixed", "fixed", PositionFixed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			style := DefaultComputedStyle()
			ctx := defaultResolutionContext()
			applyProperty(&style, "position", tc.value, ctx, nil)
			assert.Equal(t, tc.expected, style.Position)
		})
	}
}

func TestApplyProperty_BoxModel(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("width 100px resolves to 75pt", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "width", "100px", ctx, nil)
		assert.Equal(t, DimensionPt(75), style.Width)
	})

	t.Run("height 50% stored as percentage dimension", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "height", "50%", ctx, nil)
		assert.Equal(t, DimensionPct(50), style.Height)
	})

	t.Run("margin-top 10px resolves to 7.5pt dimension", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "margin-top", "10px", ctx, nil)
		assert.Equal(t, DimensionPt(7.5), style.MarginTop)
	})

	t.Run("padding-left 20px resolves to 15pt", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "padding-left", "20px", ctx, nil)
		assert.InDelta(t, 15.0, style.PaddingLeft, 0.001)
	})

	t.Run("border-top-width 2px resolves to 1.5pt", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "border-top-width", "2px", ctx, nil)
		assert.InDelta(t, 1.5, style.BorderTopWidth, 0.001)
	})
}

func TestApplyProperty_Colours(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("color red matches ParseColour result", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "color", "red", ctx, nil)

		expected, ok := ParseColour("red")
		assert.True(t, ok)
		assert.Equal(t, expected, style.Colour)
	})

	t.Run("background-color #00ff00 matches ParseColour result", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "background-color", "#00ff00", ctx, nil)

		expected, ok := ParseColour("#00ff00")
		assert.True(t, ok)
		assert.Equal(t, expected, style.BackgroundColour)
	})
}

func TestApplyProperty_FlexProperties(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("flex-direction column", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "flex-direction", "column", ctx, nil)
		assert.Equal(t, FlexDirectionColumn, style.FlexDirection)
	})

	t.Run("flex-wrap wrap", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "flex-wrap", "wrap", ctx, nil)
		assert.Equal(t, FlexWrapWrap, style.FlexWrap)
	})

	t.Run("justify-content center", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "justify-content", "center", ctx, nil)
		assert.Equal(t, JustifyCentre, style.JustifyContent)
	})

	t.Run("align-items stretch", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "align-items", "stretch", ctx, nil)
		assert.Equal(t, AlignItemsStretch, style.AlignItems)
	})

	t.Run("flex-grow 2", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "flex-grow", "2", ctx, nil)
		assert.InDelta(t, 2.0, style.FlexGrow, 0.001)
	})

	t.Run("flex-shrink 0.5", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "flex-shrink", "0.5", ctx, nil)
		assert.InDelta(t, 0.5, style.FlexShrink, 0.001)
	})
}

func TestApplyProperty_TextProperties(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("text-align center", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-align", "center", ctx, nil)
		assert.Equal(t, TextAlignCentre, style.TextAlign)
	})

	t.Run("text-decoration underline", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-decoration", "underline", ctx, nil)
		assert.Equal(t, TextDecorationUnderline, style.TextDecoration)
	})

	t.Run("text-decoration-line underline", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-decoration-line", "underline", ctx, nil)
		assert.Equal(t, TextDecorationUnderline, style.TextDecoration)
	})

	t.Run("text-decoration-line line-through", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-decoration-line", "line-through", ctx, nil)
		assert.Equal(t, TextDecorationLineThrough, style.TextDecoration)
	})

	t.Run("text-transform uppercase", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-transform", "uppercase", ctx, nil)
		assert.Equal(t, TextTransformUppercase, style.TextTransform)
	})

	t.Run("white-space nowrap", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "white-space", "nowrap", ctx, nil)
		assert.Equal(t, WhiteSpaceNowrap, style.WhiteSpace)
	})
}

func TestApplyProperty_ZIndex(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("auto sets ZIndexAuto flag", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "z-index", "auto", ctx, nil)
		assert.True(t, style.ZIndexAuto)
	})

	t.Run("numeric value sets ZIndex and clears auto flag", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "z-index", "5", ctx, nil)
		assert.Equal(t, 5, style.ZIndex)
		assert.False(t, style.ZIndexAuto)
	})
}

func TestApplyProperty_LineHeight(t *testing.T) {
	ctx := defaultResolutionContext()
	fontSize := 12.0

	t.Run("normal sets auto flag and applies default multiplier", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.FontSize = fontSize
		applyProperty(&style, "line-height", "normal", ctx, nil)
		assert.True(t, style.LineHeightAuto)
		assert.InDelta(t, fontSize*1.4, style.LineHeight, 0.001)
	})

	t.Run("unitless multiplier is applied to font size", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.FontSize = fontSize
		applyProperty(&style, "line-height", "1.5", ctx, nil)
		assert.False(t, style.LineHeightAuto)
		assert.InDelta(t, fontSize*1.5, style.LineHeight, 0.001)
	})

	t.Run("pixel value is converted to points", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.FontSize = fontSize
		applyProperty(&style, "line-height", "24px", ctx, nil)
		assert.False(t, style.LineHeightAuto)

		assert.InDelta(t, 18.0, style.LineHeight, 0.001)
	})
}

func TestApplyProperty_Gap(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("single value sets both row and column gap", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "gap", "10px", ctx, nil)

		assert.InDelta(t, 7.5, style.RowGap, 0.001)
		assert.InDelta(t, 7.5, style.ColumnGap, 0.001)
	})

	t.Run("two values set row and column gap independently", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "gap", "10px 20px", ctx, nil)
		assert.InDelta(t, 7.5, style.RowGap, 0.001)
		assert.InDelta(t, 15.0, style.ColumnGap, 0.001)
	})
}

func TestApplyProperty_Transform(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("rotate sets HasTransform true", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "transform", "rotate(45deg)", ctx, nil)
		assert.True(t, style.HasTransform)
	})

	t.Run("none sets HasTransform false", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.HasTransform = true
		applyProperty(&style, "transform", "none", ctx, nil)
		assert.False(t, style.HasTransform)
	})

	t.Run("empty string sets HasTransform false", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.HasTransform = true
		applyProperty(&style, "transform", "", ctx, nil)
		assert.False(t, style.HasTransform)
	})
}

func TestApplyProperty_Overflow(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("overflow hidden sets both axes", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "overflow", "hidden", ctx, nil)
		assert.Equal(t, OverflowHidden, style.OverflowX)
		assert.Equal(t, OverflowHidden, style.OverflowY)
	})

	t.Run("overflow-x scroll sets only horizontal axis", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "overflow-x", "scroll", ctx, nil)
		assert.Equal(t, OverflowScroll, style.OverflowX)
	})
}

func TestApplyProperty_GridProperties(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("grid-column shorthand sets start and end lines", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "grid-column", "1 / 3", ctx, nil)
		assert.Equal(t, 1, style.GridColumnStart.Line)
		assert.Equal(t, 3, style.GridColumnEnd.Line)
	})

	t.Run("grid-row shorthand with span", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "grid-row", "2 / span 3", ctx, nil)
		assert.Equal(t, 2, style.GridRowStart.Line)
		assert.Equal(t, 3, style.GridRowEnd.Span)
	})
}

func TestApplyProperty_ListStyle(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("list-style-type disc", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "list-style-type", "disc", ctx, nil)
		assert.Equal(t, ListStyleTypeDisc, style.ListStyleType)
	})

	t.Run("list-style-type decimal", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "list-style-type", "decimal", ctx, nil)
		assert.Equal(t, ListStyleTypeDecimal, style.ListStyleType)
	})

	t.Run("list-style-position inside", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "list-style-position", "inside", ctx, nil)
		assert.Equal(t, ListStylePositionInside, style.ListStylePosition)
	})

	t.Run("list-style shorthand with type and position", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "list-style", "square inside", ctx, nil)
		assert.Equal(t, ListStyleTypeSquare, style.ListStyleType)
		assert.Equal(t, ListStylePositionInside, style.ListStylePosition)
	})
}

func TestApplyProperty_ColumnProperties(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("column-count numeric", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "column-count", "3", ctx, nil)
		assert.Equal(t, 3, style.ColumnCount)
	})

	t.Run("column-count auto resolves to zero", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "column-count", "auto", ctx, nil)
		assert.Equal(t, 0, style.ColumnCount)
	})

	t.Run("column-fill auto", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "column-fill", "auto", ctx, nil)
		assert.Equal(t, ColumnFillAuto, style.ColumnFill)
	})

	t.Run("column-fill balance", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "column-fill", "balance", ctx, nil)
		assert.Equal(t, ColumnFillBalance, style.ColumnFill)
	})

	t.Run("column-span all", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "column-span", "all", ctx, nil)
		assert.Equal(t, ColumnSpanAll, style.ColumnSpan)
	})
}

func TestApplyProperty_MiscProperties(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("opacity", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "opacity", "0.5", ctx, nil)
		assert.InDelta(t, 0.5, style.Opacity, 0.001)
	})

	t.Run("tab-size", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "tab-size", "4", ctx, nil)
		assert.InDelta(t, 4.0, style.TabSize, 0.001)
	})

	t.Run("orphans", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "orphans", "3", ctx, nil)
		assert.Equal(t, 3, style.Orphans)
	})

	t.Run("widows", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "widows", "2", ctx, nil)
		assert.Equal(t, 2, style.Widows)
	})

	t.Run("text-overflow ellipsis", func(t *testing.T) {
		style := DefaultComputedStyle()
		applyProperty(&style, "text-overflow", "ellipsis", ctx, nil)
		assert.Equal(t, TextOverflowEllipsis, style.TextOverflow)
	})
}

func TestCopyPropertyFromStyle(t *testing.T) {
	tests := []struct {
		setup    func(src *ComputedStyle)
		verify   func(t *testing.T, dst *ComputedStyle)
		name     string
		property string
	}{
		{
			name:     "display is copied",
			property: "display",
			setup:    func(src *ComputedStyle) { src.Display = DisplayGrid },
			verify:   func(t *testing.T, dst *ComputedStyle) { assert.Equal(t, DisplayGrid, dst.Display) },
		},
		{
			name:     "color is copied",
			property: "color",
			setup: func(src *ComputedStyle) {
				red, _ := ParseColour("red")
				src.Colour = red
			},
			verify: func(t *testing.T, dst *ComputedStyle) {
				red, _ := ParseColour("red")
				assert.Equal(t, red, dst.Colour)
			},
		},
		{
			name:     "width is copied",
			property: "width",
			setup:    func(src *ComputedStyle) { src.Width = DimensionPt(200) },
			verify:   func(t *testing.T, dst *ComputedStyle) { assert.Equal(t, DimensionPt(200), dst.Width) },
		},
		{
			name:     "flex-grow is copied",
			property: "flex-grow",
			setup:    func(src *ComputedStyle) { src.FlexGrow = 3.0 },
			verify:   func(t *testing.T, dst *ComputedStyle) { assert.InDelta(t, 3.0, dst.FlexGrow, 0.001) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := DefaultComputedStyle()
			tc.setup(&src)

			dst := DefaultComputedStyle()
			copyPropertyFromStyle(&dst, tc.property, &src)
			tc.verify(t, &dst)
		})
	}
}

func TestInheritFromParent(t *testing.T) {
	parent := DefaultComputedStyle()

	red, _ := ParseColour("red")
	parent.Colour = red
	parent.FontSize = 20
	parent.TextAlign = TextAlignCentre
	parent.WhiteSpace = WhiteSpacePre
	parent.ListStyleType = ListStyleTypeDecimal
	parent.Orphans = 5
	parent.Widows = 4

	parent.Display = DisplayFlex
	parent.Position = PositionAbsolute

	child := DefaultComputedStyle()
	inheritFromParent(&child, &parent)

	assert.Equal(t, red, child.Colour)
	assert.InDelta(t, 20.0, child.FontSize, 0.001)
	assert.Equal(t, TextAlignCentre, child.TextAlign)
	assert.Equal(t, WhiteSpacePre, child.WhiteSpace)
	assert.Equal(t, ListStyleTypeDecimal, child.ListStyleType)
	assert.Equal(t, 5, child.Orphans)
	assert.Equal(t, 4, child.Widows)

	assert.Equal(t, DisplayInline, child.Display)
	assert.Equal(t, PositionStatic, child.Position)
}

func TestCollectCustomProperties(t *testing.T) {
	properties := map[string]string{
		"--my-var": "red",
		"--other":  "blue",
		"color":    "green",
	}

	style := DefaultComputedStyle()
	collectCustomProperties(&style, properties)

	assert.Equal(t, "red", style.CustomProperties["--my-var"])
	assert.Equal(t, "blue", style.CustomProperties["--other"])

	_, found := style.CustomProperties["color"]
	assert.False(t, found)
}

func TestCopyPropertyFromStyle_Comprehensive(t *testing.T) {

	testColour := NewRGBA(0.5, 0.5, 0.5, 1.0)

	testCases := []struct {
		setup    func(src *ComputedStyle)
		verify   func(t *testing.T, dst *ComputedStyle)
		name     string
		property string
	}{

		{
			name:     "display",
			property: "display",
			setup:    func(s *ComputedStyle) { s.Display = DisplayFlex },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DisplayFlex, d.Display) },
		},
		{
			name:     "position",
			property: "position",
			setup:    func(s *ComputedStyle) { s.Position = PositionAbsolute },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PositionAbsolute, d.Position) },
		},
		{
			name:     "transform",
			property: "transform",
			setup:    func(s *ComputedStyle) { s.HasTransform = true },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.True(t, d.HasTransform) },
		},
		{
			name:     "transform-origin",
			property: "transform-origin",
			setup:    func(s *ComputedStyle) { s.TransformOrigin = "top left" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "top left", d.TransformOrigin) },
		},
		{
			name:     "box-sizing",
			property: "box-sizing",
			setup:    func(s *ComputedStyle) { s.BoxSizing = BoxSizingBorderBox },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BoxSizingBorderBox, d.BoxSizing) },
		},
		{
			name:     "float",
			property: "float",
			setup:    func(s *ComputedStyle) { s.Float = FloatLeft },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, FloatLeft, d.Float) },
		},
		{
			name:     "clear",
			property: "clear",
			setup:    func(s *ComputedStyle) { s.Clear = ClearBoth },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ClearBoth, d.Clear) },
		},
		{
			name:     "visibility",
			property: "visibility",
			setup:    func(s *ComputedStyle) { s.Visibility = VisibilityHidden },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, VisibilityHidden, d.Visibility) },
		},

		{
			name:     "overflow-x",
			property: "overflow-x",
			setup:    func(s *ComputedStyle) { s.OverflowX = OverflowScroll },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, OverflowScroll, d.OverflowX) },
		},
		{
			name:     "overflow-y",
			property: "overflow-y",
			setup:    func(s *ComputedStyle) { s.OverflowY = OverflowScroll },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, OverflowScroll, d.OverflowY) },
		},

		{
			name:     "z-index",
			property: "z-index",
			setup: func(s *ComputedStyle) {
				s.ZIndex = 42
				s.ZIndexAuto = true
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Equal(t, 42, d.ZIndex)
				assert.True(t, d.ZIndexAuto)
			},
		},

		{
			name:     "width",
			property: "width",
			setup:    func(s *ComputedStyle) { s.Width = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Width) },
		},
		{
			name:     "height",
			property: "height",
			setup:    func(s *ComputedStyle) { s.Height = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Height) },
		},
		{
			name:     "min-width",
			property: "min-width",
			setup:    func(s *ComputedStyle) { s.MinWidth = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MinWidth) },
		},
		{
			name:     "min-height",
			property: "min-height",
			setup:    func(s *ComputedStyle) { s.MinHeight = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MinHeight) },
		},
		{
			name:     "max-width",
			property: "max-width",
			setup:    func(s *ComputedStyle) { s.MaxWidth = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MaxWidth) },
		},
		{
			name:     "max-height",
			property: "max-height",
			setup:    func(s *ComputedStyle) { s.MaxHeight = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MaxHeight) },
		},

		{
			name:     "margin-top",
			property: "margin-top",
			setup:    func(s *ComputedStyle) { s.MarginTop = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MarginTop) },
		},
		{
			name:     "margin-right",
			property: "margin-right",
			setup:    func(s *ComputedStyle) { s.MarginRight = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MarginRight) },
		},
		{
			name:     "margin-bottom",
			property: "margin-bottom",
			setup:    func(s *ComputedStyle) { s.MarginBottom = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MarginBottom) },
		},
		{
			name:     "margin-left",
			property: "margin-left",
			setup:    func(s *ComputedStyle) { s.MarginLeft = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.MarginLeft) },
		},

		{
			name:     "padding-top",
			property: "padding-top",
			setup:    func(s *ComputedStyle) { s.PaddingTop = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.PaddingTop, 0.001) },
		},
		{
			name:     "padding-right",
			property: "padding-right",
			setup:    func(s *ComputedStyle) { s.PaddingRight = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.PaddingRight, 0.001) },
		},
		{
			name:     "padding-bottom",
			property: "padding-bottom",
			setup:    func(s *ComputedStyle) { s.PaddingBottom = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.PaddingBottom, 0.001) },
		},
		{
			name:     "padding-left",
			property: "padding-left",
			setup:    func(s *ComputedStyle) { s.PaddingLeft = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.PaddingLeft, 0.001) },
		},

		{
			name:     "border-top-width",
			property: "border-top-width",
			setup:    func(s *ComputedStyle) { s.BorderTopWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderTopWidth, 0.001) },
		},
		{
			name:     "border-right-width",
			property: "border-right-width",
			setup:    func(s *ComputedStyle) { s.BorderRightWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderRightWidth, 0.001) },
		},
		{
			name:     "border-bottom-width",
			property: "border-bottom-width",
			setup:    func(s *ComputedStyle) { s.BorderBottomWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderBottomWidth, 0.001) },
		},
		{
			name:     "border-left-width",
			property: "border-left-width",
			setup:    func(s *ComputedStyle) { s.BorderLeftWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderLeftWidth, 0.001) },
		},

		{
			name:     "border-top-style",
			property: "border-top-style",
			setup:    func(s *ComputedStyle) { s.BorderTopStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.BorderTopStyle) },
		},
		{
			name:     "border-right-style",
			property: "border-right-style",
			setup:    func(s *ComputedStyle) { s.BorderRightStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.BorderRightStyle) },
		},
		{
			name:     "border-bottom-style",
			property: "border-bottom-style",
			setup:    func(s *ComputedStyle) { s.BorderBottomStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.BorderBottomStyle) },
		},
		{
			name:     "border-left-style",
			property: "border-left-style",
			setup:    func(s *ComputedStyle) { s.BorderLeftStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.BorderLeftStyle) },
		},

		{
			name:     "border-top-left-radius",
			property: "border-top-left-radius",
			setup:    func(s *ComputedStyle) { s.BorderTopLeftRadius = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderTopLeftRadius, 0.001) },
		},
		{
			name:     "border-top-right-radius",
			property: "border-top-right-radius",
			setup:    func(s *ComputedStyle) { s.BorderTopRightRadius = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderTopRightRadius, 0.001) },
		},
		{
			name:     "border-bottom-right-radius",
			property: "border-bottom-right-radius",
			setup:    func(s *ComputedStyle) { s.BorderBottomRightRadius = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderBottomRightRadius, 0.001) },
		},
		{
			name:     "border-bottom-left-radius",
			property: "border-bottom-left-radius",
			setup:    func(s *ComputedStyle) { s.BorderBottomLeftRadius = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderBottomLeftRadius, 0.001) },
		},

		{
			name:     "top",
			property: "top",
			setup:    func(s *ComputedStyle) { s.Top = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Top) },
		},
		{
			name:     "right",
			property: "right",
			setup:    func(s *ComputedStyle) { s.Right = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Right) },
		},
		{
			name:     "bottom",
			property: "bottom",
			setup:    func(s *ComputedStyle) { s.Bottom = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Bottom) },
		},
		{
			name:     "left",
			property: "left",
			setup:    func(s *ComputedStyle) { s.Left = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.Left) },
		},

		{
			name:     "font-family",
			property: "font-family",
			setup:    func(s *ComputedStyle) { s.FontFamily = "test-value" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "test-value", d.FontFamily) },
		},
		{
			name:     "font-size",
			property: "font-size",
			setup:    func(s *ComputedStyle) { s.FontSize = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.FontSize, 0.001) },
		},
		{
			name:     "font-weight",
			property: "font-weight",
			setup:    func(s *ComputedStyle) { s.FontWeight = 700 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 700.0, d.FontWeight, 0.001) },
		},
		{
			name:     "font-style",
			property: "font-style",
			setup:    func(s *ComputedStyle) { s.FontStyle = FontStyleItalic },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, FontStyleItalic, d.FontStyle) },
		},

		{
			name:     "line-height",
			property: "line-height",
			setup: func(s *ComputedStyle) {
				s.LineHeight = 42.0
				s.LineHeightAuto = true
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.InDelta(t, 42.0, d.LineHeight, 0.001)
				assert.True(t, d.LineHeightAuto)
			},
		},

		{
			name:     "text-align",
			property: "text-align",
			setup:    func(s *ComputedStyle) { s.TextAlign = TextAlignCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, TextAlignCentre, d.TextAlign) },
		},
		{
			name:     "text-decoration",
			property: "text-decoration",
			setup:    func(s *ComputedStyle) { s.TextDecoration = TextDecorationUnderline },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, TextDecorationUnderline, d.TextDecoration) },
		},
		{
			name:     "text-decoration-color",
			property: "text-decoration-color",
			setup: func(s *ComputedStyle) {
				s.TextDecorationColour = testColour
				s.TextDecorationColourSet = true
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Equal(t, testColour, d.TextDecorationColour)
				assert.True(t, d.TextDecorationColourSet)
			},
		},
		{
			name:     "text-decoration-style",
			property: "text-decoration-style",
			setup:    func(s *ComputedStyle) { s.TextDecorationStyle = TextDecorationStyleDashed },
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Equal(t, TextDecorationStyleDashed, d.TextDecorationStyle)
			},
		},
		{
			name:     "text-transform",
			property: "text-transform",
			setup:    func(s *ComputedStyle) { s.TextTransform = TextTransformUppercase },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, TextTransformUppercase, d.TextTransform) },
		},
		{
			name:     "letter-spacing",
			property: "letter-spacing",
			setup:    func(s *ComputedStyle) { s.LetterSpacing = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.LetterSpacing, 0.001) },
		},
		{
			name:     "word-spacing",
			property: "word-spacing",
			setup:    func(s *ComputedStyle) { s.WordSpacing = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.WordSpacing, 0.001) },
		},
		{
			name:     "white-space",
			property: "white-space",
			setup:    func(s *ComputedStyle) { s.WhiteSpace = WhiteSpaceNowrap },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, WhiteSpaceNowrap, d.WhiteSpace) },
		},
		{
			name:     "word-break",
			property: "word-break",
			setup:    func(s *ComputedStyle) { s.WordBreak = WordBreakBreakAll },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, WordBreakBreakAll, d.WordBreak) },
		},
		{
			name:     "overflow-wrap",
			property: "overflow-wrap",
			setup:    func(s *ComputedStyle) { s.OverflowWrap = OverflowWrapBreakWord },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, OverflowWrapBreakWord, d.OverflowWrap) },
		},
		{
			name:     "word-wrap aliases to overflow-wrap",
			property: "word-wrap",
			setup:    func(s *ComputedStyle) { s.OverflowWrap = OverflowWrapBreakWord },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, OverflowWrapBreakWord, d.OverflowWrap) },
		},
		{
			name:     "text-indent",
			property: "text-indent",
			setup:    func(s *ComputedStyle) { s.TextIndent = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.TextIndent, 0.001) },
		},

		{
			name:     "color",
			property: "color",
			setup:    func(s *ComputedStyle) { s.Colour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.Colour) },
		},
		{
			name:     "background-color",
			property: "background-color",
			setup:    func(s *ComputedStyle) { s.BackgroundColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.BackgroundColour) },
		},
		{
			name:     "border-top-color",
			property: "border-top-color",
			setup:    func(s *ComputedStyle) { s.BorderTopColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.BorderTopColour) },
		},
		{
			name:     "border-right-color",
			property: "border-right-color",
			setup:    func(s *ComputedStyle) { s.BorderRightColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.BorderRightColour) },
		},
		{
			name:     "border-bottom-color",
			property: "border-bottom-color",
			setup:    func(s *ComputedStyle) { s.BorderBottomColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.BorderBottomColour) },
		},
		{
			name:     "border-left-color",
			property: "border-left-color",
			setup:    func(s *ComputedStyle) { s.BorderLeftColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.BorderLeftColour) },
		},

		{
			name:     "flex-direction",
			property: "flex-direction",
			setup:    func(s *ComputedStyle) { s.FlexDirection = FlexDirectionColumn },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, FlexDirectionColumn, d.FlexDirection) },
		},
		{
			name:     "flex-wrap",
			property: "flex-wrap",
			setup:    func(s *ComputedStyle) { s.FlexWrap = FlexWrapWrap },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, FlexWrapWrap, d.FlexWrap) },
		},
		{
			name:     "justify-content",
			property: "justify-content",
			setup:    func(s *ComputedStyle) { s.JustifyContent = JustifyCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, JustifyCentre, d.JustifyContent) },
		},
		{
			name:     "align-items",
			property: "align-items",
			setup:    func(s *ComputedStyle) { s.AlignItems = AlignItemsCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, AlignItemsCentre, d.AlignItems) },
		},
		{
			name:     "align-self",
			property: "align-self",
			setup:    func(s *ComputedStyle) { s.AlignSelf = AlignSelfCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, AlignSelfCentre, d.AlignSelf) },
		},
		{
			name:     "align-content",
			property: "align-content",
			setup:    func(s *ComputedStyle) { s.AlignContent = AlignContentCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, AlignContentCentre, d.AlignContent) },
		},
		{
			name:     "justify-items",
			property: "justify-items",
			setup:    func(s *ComputedStyle) { s.JustifyItems = JustifyItemsCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, JustifyItemsCentre, d.JustifyItems) },
		},
		{
			name:     "justify-self",
			property: "justify-self",
			setup:    func(s *ComputedStyle) { s.JustifySelf = JustifySelfCentre },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, JustifySelfCentre, d.JustifySelf) },
		},
		{
			name:     "flex-grow",
			property: "flex-grow",
			setup:    func(s *ComputedStyle) { s.FlexGrow = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.FlexGrow, 0.001) },
		},
		{
			name:     "flex-shrink",
			property: "flex-shrink",
			setup:    func(s *ComputedStyle) { s.FlexShrink = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.FlexShrink, 0.001) },
		},
		{
			name:     "flex-basis",
			property: "flex-basis",
			setup:    func(s *ComputedStyle) { s.FlexBasis = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.FlexBasis) },
		},
		{
			name:     "order",
			property: "order",
			setup:    func(s *ComputedStyle) { s.Order = 7 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 7, d.Order) },
		},

		{
			name:     "row-gap",
			property: "row-gap",
			setup:    func(s *ComputedStyle) { s.RowGap = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.RowGap, 0.001) },
		},
		{
			name:     "column-gap",
			property: "column-gap",
			setup:    func(s *ComputedStyle) { s.ColumnGap = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.ColumnGap, 0.001) },
		},

		{
			name:     "table-layout",
			property: "table-layout",
			setup:    func(s *ComputedStyle) { s.TableLayout = TableLayoutFixed },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, TableLayoutFixed, d.TableLayout) },
		},
		{
			name:     "border-collapse",
			property: "border-collapse",
			setup:    func(s *ComputedStyle) { s.BorderCollapse = BorderCollapseCollapse },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderCollapseCollapse, d.BorderCollapse) },
		},
		{
			name:     "border-spacing",
			property: "border-spacing",
			setup:    func(s *ComputedStyle) { s.BorderSpacing = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderSpacing, 0.001) },
		},
		{
			name:     "caption-side",
			property: "caption-side",
			setup:    func(s *ComputedStyle) { s.CaptionSide = CaptionSideBottom },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, CaptionSideBottom, d.CaptionSide) },
		},
		{
			name:     "vertical-align",
			property: "vertical-align",
			setup:    func(s *ComputedStyle) { s.VerticalAlign = VerticalAlignMiddle },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, VerticalAlignMiddle, d.VerticalAlign) },
		},

		{
			name:     "list-style-type",
			property: "list-style-type",
			setup:    func(s *ComputedStyle) { s.ListStyleType = ListStyleTypeDecimal },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ListStyleTypeDecimal, d.ListStyleType) },
		},
		{
			name:     "list-style-position",
			property: "list-style-position",
			setup:    func(s *ComputedStyle) { s.ListStylePosition = ListStylePositionInside },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ListStylePositionInside, d.ListStylePosition) },
		},

		{
			name:     "page-break-before",
			property: "page-break-before",
			setup:    func(s *ComputedStyle) { s.PageBreakBefore = PageBreakAlways },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAlways, d.PageBreakBefore) },
		},
		{
			name:     "page-break-after",
			property: "page-break-after",
			setup:    func(s *ComputedStyle) { s.PageBreakAfter = PageBreakAlways },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAlways, d.PageBreakAfter) },
		},
		{
			name:     "page-break-inside",
			property: "page-break-inside",
			setup:    func(s *ComputedStyle) { s.PageBreakInside = PageBreakAvoid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAvoid, d.PageBreakInside) },
		},
		{
			name:     "break-before aliases to page-break-before",
			property: "break-before",
			setup:    func(s *ComputedStyle) { s.PageBreakBefore = PageBreakAlways },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAlways, d.PageBreakBefore) },
		},
		{
			name:     "break-after aliases to page-break-after",
			property: "break-after",
			setup:    func(s *ComputedStyle) { s.PageBreakAfter = PageBreakAlways },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAlways, d.PageBreakAfter) },
		},
		{
			name:     "break-inside aliases to page-break-inside",
			property: "break-inside",
			setup:    func(s *ComputedStyle) { s.PageBreakInside = PageBreakAvoid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, PageBreakAvoid, d.PageBreakInside) },
		},

		{
			name:     "orphans",
			property: "orphans",
			setup:    func(s *ComputedStyle) { s.Orphans = 7 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 7, d.Orphans) },
		},
		{
			name:     "widows",
			property: "widows",
			setup:    func(s *ComputedStyle) { s.Widows = 7 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 7, d.Widows) },
		},

		{
			name:     "opacity",
			property: "opacity",
			setup:    func(s *ComputedStyle) { s.Opacity = 0.42 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 0.42, d.Opacity, 0.001) },
		},

		{
			name:     "box-shadow",
			property: "box-shadow",
			setup: func(s *ComputedStyle) {
				s.BoxShadow = []BoxShadowValue{{OffsetX: 1, OffsetY: 2, Colour: ColourBlack}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.BoxShadow, 1)
				assert.InDelta(t, 1.0, d.BoxShadow[0].OffsetX, 0.001)
				assert.InDelta(t, 2.0, d.BoxShadow[0].OffsetY, 0.001)
				assert.Equal(t, ColourBlack, d.BoxShadow[0].Colour)
			},
		},

		{
			name:     "grid-template-columns",
			property: "grid-template-columns",
			setup: func(s *ComputedStyle) {
				s.GridTemplateColumns = []GridTrack{{Value: 100, Unit: GridTrackPoints}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.GridTemplateColumns, 1)
				assert.InDelta(t, 100.0, d.GridTemplateColumns[0].Value, 0.001)
				assert.Equal(t, GridTrackPoints, d.GridTemplateColumns[0].Unit)
			},
		},
		{
			name:     "grid-template-rows",
			property: "grid-template-rows",
			setup: func(s *ComputedStyle) {
				s.GridTemplateRows = []GridTrack{{Value: 100, Unit: GridTrackPoints}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.GridTemplateRows, 1)
				assert.InDelta(t, 100.0, d.GridTemplateRows[0].Value, 0.001)
			},
		},
		{
			name:     "grid-auto-columns",
			property: "grid-auto-columns",
			setup: func(s *ComputedStyle) {
				s.GridAutoColumns = []GridTrack{{Value: 100, Unit: GridTrackPoints}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.GridAutoColumns, 1)
				assert.InDelta(t, 100.0, d.GridAutoColumns[0].Value, 0.001)
			},
		},
		{
			name:     "grid-auto-rows",
			property: "grid-auto-rows",
			setup: func(s *ComputedStyle) {
				s.GridAutoRows = []GridTrack{{Value: 100, Unit: GridTrackPoints}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.GridAutoRows, 1)
				assert.InDelta(t, 100.0, d.GridAutoRows[0].Value, 0.001)
			},
		},

		{
			name:     "grid-column-start",
			property: "grid-column-start",
			setup:    func(s *ComputedStyle) { s.GridColumnStart = GridLine{Line: 2} },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 2, d.GridColumnStart.Line) },
		},
		{
			name:     "grid-column-end",
			property: "grid-column-end",
			setup:    func(s *ComputedStyle) { s.GridColumnEnd = GridLine{Line: 2} },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 2, d.GridColumnEnd.Line) },
		},
		{
			name:     "grid-row-start",
			property: "grid-row-start",
			setup:    func(s *ComputedStyle) { s.GridRowStart = GridLine{Line: 2} },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 2, d.GridRowStart.Line) },
		},
		{
			name:     "grid-row-end",
			property: "grid-row-end",
			setup:    func(s *ComputedStyle) { s.GridRowEnd = GridLine{Line: 2} },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 2, d.GridRowEnd.Line) },
		},

		{
			name:     "grid-template-areas",
			property: "grid-template-areas",
			setup: func(s *ComputedStyle) {
				s.GridTemplateAreas = [][]string{{"header", "header"}, {"main", "sidebar"}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.GridTemplateAreas, 2)
				assert.Equal(t, []string{"header", "header"}, d.GridTemplateAreas[0])
				assert.Equal(t, []string{"main", "sidebar"}, d.GridTemplateAreas[1])
			},
		},
		{
			name:     "grid-area",
			property: "grid-area",
			setup:    func(s *ComputedStyle) { s.GridArea = "test-value" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "test-value", d.GridArea) },
		},

		{
			name:     "writing-mode",
			property: "writing-mode",
			setup:    func(s *ComputedStyle) { s.WritingMode = WritingModeVerticalRL },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, WritingModeVerticalRL, d.WritingMode) },
		},
		{
			name:     "grid-auto-flow",
			property: "grid-auto-flow",
			setup:    func(s *ComputedStyle) { s.GridAutoFlow = GridAutoFlowColumn },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, GridAutoFlowColumn, d.GridAutoFlow) },
		},

		{
			name:     "aspect-ratio",
			property: "aspect-ratio",
			setup: func(s *ComputedStyle) {
				s.AspectRatio = 1.5
				s.AspectRatioAuto = false
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.InDelta(t, 1.5, d.AspectRatio, 0.001)
				assert.False(t, d.AspectRatioAuto)
			},
		},

		{
			name:     "text-overflow",
			property: "text-overflow",
			setup:    func(s *ComputedStyle) { s.TextOverflow = TextOverflowEllipsis },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, TextOverflowEllipsis, d.TextOverflow) },
		},

		{
			name:     "content",
			property: "content",
			setup:    func(s *ComputedStyle) { s.Content = "test-value" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "test-value", d.Content) },
		},

		{
			name:     "column-count",
			property: "column-count",
			setup:    func(s *ComputedStyle) { s.ColumnCount = 7 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, 7, d.ColumnCount) },
		},
		{
			name:     "column-width",
			property: "column-width",
			setup:    func(s *ComputedStyle) { s.ColumnWidth = DimensionPt(99) },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DimensionPt(99), d.ColumnWidth) },
		},
		{
			name:     "column-fill",
			property: "column-fill",
			setup:    func(s *ComputedStyle) { s.ColumnFill = ColumnFillAuto },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ColumnFillAuto, d.ColumnFill) },
		},
		{
			name:     "column-rule-width",
			property: "column-rule-width",
			setup:    func(s *ComputedStyle) { s.ColumnRuleWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.ColumnRuleWidth, 0.001) },
		},
		{
			name:     "column-rule-style",
			property: "column-rule-style",
			setup:    func(s *ComputedStyle) { s.ColumnRuleStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.ColumnRuleStyle) },
		},
		{
			name:     "column-rule-color",
			property: "column-rule-color",
			setup:    func(s *ComputedStyle) { s.ColumnRuleColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.ColumnRuleColour) },
		},
		{
			name:     "column-span",
			property: "column-span",
			setup:    func(s *ComputedStyle) { s.ColumnSpan = ColumnSpanAll },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ColumnSpanAll, d.ColumnSpan) },
		},

		{
			name:     "text-shadow",
			property: "text-shadow",
			setup: func(s *ComputedStyle) {
				s.TextShadow = []TextShadowValue{{OffsetX: 1, OffsetY: 2, Colour: ColourBlack}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.TextShadow, 1)
				assert.InDelta(t, 1.0, d.TextShadow[0].OffsetX, 0.001)
				assert.InDelta(t, 2.0, d.TextShadow[0].OffsetY, 0.001)
				assert.Equal(t, ColourBlack, d.TextShadow[0].Colour)
			},
		},

		{
			name:     "outline-width",
			property: "outline-width",
			setup:    func(s *ComputedStyle) { s.OutlineWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.OutlineWidth, 0.001) },
		},
		{
			name:     "outline-style",
			property: "outline-style",
			setup:    func(s *ComputedStyle) { s.OutlineStyle = BorderStyleSolid },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderStyleSolid, d.OutlineStyle) },
		},
		{
			name:     "outline-color",
			property: "outline-color",
			setup:    func(s *ComputedStyle) { s.OutlineColour = testColour },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, testColour, d.OutlineColour) },
		},
		{
			name:     "outline-offset",
			property: "outline-offset",
			setup:    func(s *ComputedStyle) { s.OutlineOffset = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.OutlineOffset, 0.001) },
		},

		{
			name:     "background-image",
			property: "background-image",
			setup: func(s *ComputedStyle) {
				s.BgImages = []BackgroundImage{{Type: BackgroundImageURL, URL: "test.png"}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Equal(t, 1, len(d.BgImages))
				assert.Equal(t, BackgroundImageURL, d.BgImages[0].Type)
				assert.Equal(t, "test.png", d.BgImages[0].URL)
			},
		},
		{
			name:     "background-size",
			property: "background-size",
			setup:    func(s *ComputedStyle) { s.BgSize = "cover" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "cover", d.BgSize) },
		},
		{
			name:     "background-position",
			property: "background-position",
			setup:    func(s *ComputedStyle) { s.BgPosition = "center top" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "center top", d.BgPosition) },
		},
		{
			name:     "background-repeat",
			property: "background-repeat",
			setup:    func(s *ComputedStyle) { s.BgRepeat = "no-repeat" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "no-repeat", d.BgRepeat) },
		},
		{
			name:     "background-attachment",
			property: "background-attachment",
			setup:    func(s *ComputedStyle) { s.BgAttachment = "fixed" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "fixed", d.BgAttachment) },
		},
		{
			name:     "background-origin",
			property: "background-origin",
			setup:    func(s *ComputedStyle) { s.BgOrigin = "border-box" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "border-box", d.BgOrigin) },
		},
		{
			name:     "background-clip",
			property: "background-clip",
			setup:    func(s *ComputedStyle) { s.BgClip = "padding-box" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "padding-box", d.BgClip) },
		},

		{
			name:     "object-fit",
			property: "object-fit",
			setup:    func(s *ComputedStyle) { s.ObjectFit = ObjectFitCover },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, ObjectFitCover, d.ObjectFit) },
		},
		{
			name:     "object-position",
			property: "object-position",
			setup:    func(s *ComputedStyle) { s.ObjectPosition = "center bottom" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "center bottom", d.ObjectPosition) },
		},

		{
			name:     "border-image-source",
			property: "border-image-source",
			setup:    func(s *ComputedStyle) { s.BorderImageSource = "url(test.png)" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "url(test.png)", d.BorderImageSource) },
		},
		{
			name:     "border-image-slice",
			property: "border-image-slice",
			setup:    func(s *ComputedStyle) { s.BorderImageSlice = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderImageSlice, 0.001) },
		},
		{
			name:     "border-image-width",
			property: "border-image-width",
			setup:    func(s *ComputedStyle) { s.BorderImageWidth = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderImageWidth, 0.001) },
		},
		{
			name:     "border-image-outset",
			property: "border-image-outset",
			setup:    func(s *ComputedStyle) { s.BorderImageOutset = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.BorderImageOutset, 0.001) },
		},
		{
			name:     "border-image-repeat",
			property: "border-image-repeat",
			setup:    func(s *ComputedStyle) { s.BorderImageRepeat = BorderImageRepeatRepeat },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, BorderImageRepeatRepeat, d.BorderImageRepeat) },
		},

		{
			name:     "clip-path",
			property: "clip-path",
			setup:    func(s *ComputedStyle) { s.ClipPath = "circle(50%)" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "circle(50%)", d.ClipPath) },
		},
		{
			name:     "mask-image",
			property: "mask-image",
			setup:    func(s *ComputedStyle) { s.MaskImage = "url(mask.svg)" },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, "url(mask.svg)", d.MaskImage) },
		},

		{
			name:     "direction",
			property: "direction",
			setup:    func(s *ComputedStyle) { s.Direction = DirectionRTL },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, DirectionRTL, d.Direction) },
		},
		{
			name:     "hyphens",
			property: "hyphens",
			setup:    func(s *ComputedStyle) { s.Hyphens = HyphensAuto },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.Equal(t, HyphensAuto, d.Hyphens) },
		},
		{
			name:     "tab-size",
			property: "tab-size",
			setup:    func(s *ComputedStyle) { s.TabSize = 42.0 },
			verify:   func(t *testing.T, d *ComputedStyle) { assert.InDelta(t, 42.0, d.TabSize, 0.001) },
		},

		{
			name:     "counter-reset",
			property: "counter-reset",
			setup: func(s *ComputedStyle) {
				s.CounterReset = []CounterEntry{{Name: "test", Value: 1}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.CounterReset, 1)
				assert.Equal(t, "test", d.CounterReset[0].Name)
				assert.Equal(t, 1, d.CounterReset[0].Value)
			},
		},
		{
			name:     "counter-increment",
			property: "counter-increment",
			setup: func(s *ComputedStyle) {
				s.CounterIncrement = []CounterEntry{{Name: "test", Value: 1}}
			},
			verify: func(t *testing.T, d *ComputedStyle) {
				assert.Len(t, d.CounterIncrement, 1)
				assert.Equal(t, "test", d.CounterIncrement[0].Name)
				assert.Equal(t, 1, d.CounterIncrement[0].Value)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := DefaultComputedStyle()
			tc.setup(&src)
			dst := DefaultComputedStyle()
			copyPropertyFromStyle(&dst, tc.property, &src)
			tc.verify(t, &dst)
		})
	}
}

func TestApplyProperty_AllBranches(t *testing.T) {
	testCases := []struct {
		property string
		value    string
		verify   func(t *testing.T, s *ComputedStyle)
	}{

		{"transform-origin", "center top", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "center top", s.TransformOrigin)
		}},

		{"min-width", "50px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(37.5), s.MinWidth)
		}},
		{"min-height", "30px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(22.5), s.MinHeight)
		}},
		{"max-width", "200px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(150), s.MaxWidth)
		}},
		{"max-height", "100px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(75), s.MaxHeight)
		}},

		{"margin-right", "10px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(7.5), s.MarginRight)
		}},
		{"margin-bottom", "20px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(15), s.MarginBottom)
		}},
		{"margin-left", "5px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(3.75), s.MarginLeft)
		}},

		{"padding-top", "10px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 7.5, s.PaddingTop, 0.001)
		}},
		{"padding-right", "8px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 6.0, s.PaddingRight, 0.001)
		}},
		{"padding-bottom", "12px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 9.0, s.PaddingBottom, 0.001)
		}},

		{"border-right-width", "3px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 2.25, s.BorderRightWidth, 0.001)
		}},
		{"border-bottom-width", "4px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 3.0, s.BorderBottomWidth, 0.001)
		}},
		{"border-left-width", "1px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 0.75, s.BorderLeftWidth, 0.001)
		}},

		{"border-top-style", "solid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleSolid, s.BorderTopStyle)
		}},
		{"border-right-style", "dashed", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleDashed, s.BorderRightStyle)
		}},
		{"border-bottom-style", "dotted", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleDotted, s.BorderBottomStyle)
		}},
		{"border-left-style", "double", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleDouble, s.BorderLeftStyle)
		}},

		{"border-top-left-radius", "5px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 3.75, s.BorderTopLeftRadius, 0.001)
		}},
		{"border-top-right-radius", "6px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 4.5, s.BorderTopRightRadius, 0.001)
		}},
		{"border-bottom-right-radius", "7px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 5.25, s.BorderBottomRightRadius, 0.001)
		}},
		{"border-bottom-left-radius", "8px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 6.0, s.BorderBottomLeftRadius, 0.001)
		}},

		{"top", "10px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(7.5), s.Top)
		}},
		{"right", "20px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(15), s.Right)
		}},
		{"bottom", "30px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(22.5), s.Bottom)
		}},
		{"left", "40px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(30), s.Left)
		}},

		{"font-family", "Arial, sans-serif", func(t *testing.T, s *ComputedStyle) {
			assert.NotEqual(t, "serif", s.FontFamily)
		}},
		{"font-size", "16px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 12.0, s.FontSize, 0.001)
		}},
		{"font-weight", "bold", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 700, s.FontWeight)
		}},
		{"font-style", "italic", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, FontStyleItalic, s.FontStyle)
		}},

		{"text-decoration-color", "red", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("red")
			assert.Equal(t, expected, s.TextDecorationColour)
			assert.True(t, s.TextDecorationColourSet)
		}},

		{"text-decoration-style", "dashed", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, TextDecorationStyleDashed, s.TextDecorationStyle)
		}},

		{"letter-spacing", "2px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 1.5, s.LetterSpacing, 0.001)
		}},

		{"word-spacing", "4px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 3.0, s.WordSpacing, 0.001)
		}},

		{"word-break", "break-all", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, WordBreakBreakAll, s.WordBreak)
		}},

		{"overflow-wrap", "break-word", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, OverflowWrapBreakWord, s.OverflowWrap)
		}},

		{"word-wrap", "anywhere", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, OverflowWrapAnywhere, s.OverflowWrap)
		}},

		{"text-indent", "20px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 15.0, s.TextIndent, 0.001)
		}},

		{"border-top-color", "red", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("red")
			assert.Equal(t, expected, s.BorderTopColour)
		}},
		{"border-right-color", "green", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("green")
			assert.Equal(t, expected, s.BorderRightColour)
		}},
		{"border-bottom-color", "blue", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("blue")
			assert.Equal(t, expected, s.BorderBottomColour)
		}},
		{"border-left-color", "#ff0000", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("#ff0000")
			assert.Equal(t, expected, s.BorderLeftColour)
		}},

		{"align-self", "center", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, AlignSelfCentre, s.AlignSelf)
		}},
		{"align-content", "space-between", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, AlignContentSpaceBetween, s.AlignContent)
		}},
		{"justify-items", "start", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, JustifyItemsStart, s.JustifyItems)
		}},
		{"justify-self", "end", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, JustifySelfEnd, s.JustifySelf)
		}},

		{"flex-basis", "100px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(75), s.FlexBasis)
		}},

		{"order", "3", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 3, s.Order)
		}},

		{"row-gap", "10px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 7.5, s.RowGap, 0.001)
		}},
		{"column-gap", "20px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 15.0, s.ColumnGap, 0.001)
		}},

		{"table-layout", "fixed", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, TableLayoutFixed, s.TableLayout)
		}},
		{"border-collapse", "collapse", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderCollapseCollapse, s.BorderCollapse)
		}},
		{"border-spacing", "6px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 4.5, s.BorderSpacing, 0.001)
		}},
		{"caption-side", "bottom", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, CaptionSideBottom, s.CaptionSide)
		}},
		{"vertical-align", "middle", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, VerticalAlignMiddle, s.VerticalAlign)
		}},

		{"page-break-before", "always", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAlways, s.PageBreakBefore)
		}},
		{"page-break-after", "avoid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAvoid, s.PageBreakAfter)
		}},
		{"page-break-inside", "avoid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAvoid, s.PageBreakInside)
		}},
		{"break-before", "always", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAlways, s.PageBreakBefore)
		}},
		{"break-after", "avoid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAvoid, s.PageBreakAfter)
		}},
		{"break-inside", "avoid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, PageBreakAvoid, s.PageBreakInside)
		}},

		{"box-shadow", "2px 2px 4px black", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.BoxShadow)
		}},

		{"grid-template-columns", "100px 200px", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridTemplateColumns)
		}},
		{"grid-template-rows", "50px auto", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridTemplateRows)
		}},

		{"grid-auto-columns", "1fr", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridAutoColumns)
		}},
		{"grid-auto-rows", "min-content", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridAutoRows)
		}},

		{"grid-column-start", "2", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 2, s.GridColumnStart.Line)
		}},
		{"grid-column-end", "4", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 4, s.GridColumnEnd.Line)
		}},
		{"grid-row-start", "1", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 1, s.GridRowStart.Line)
		}},
		{"grid-row-end", "3", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 3, s.GridRowEnd.Line)
		}},

		{"grid-template-areas", `"header header" "main sidebar"`, func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridTemplateAreas)
		}},

		{"grid-area", "header", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.GridArea)
		}},

		{"writing-mode", "vertical-rl", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, WritingModeVerticalRL, s.WritingMode)
		}},

		{"grid-auto-flow", "column", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, GridAutoFlowColumn, s.GridAutoFlow)
		}},

		{"aspect-ratio", "16 / 9", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 16.0/9.0, s.AspectRatio, 0.001)
		}},

		{"content", `"hello"`, func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.Content)
		}},

		{"column-width", "200px", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DimensionPt(150), s.ColumnWidth)
		}},

		{"columns", "3 200px", func(t *testing.T, s *ComputedStyle) {

			has_change := s.ColumnCount != 0 || s.ColumnWidth.Unit != DimensionUnitAuto
			assert.True(t, has_change)
		}},

		{"column-rule-width", "2px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 1.5, s.ColumnRuleWidth, 0.001)
		}},

		{"column-rule-style", "solid", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleSolid, s.ColumnRuleStyle)
		}},

		{"column-rule-color", "red", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("red")
			assert.Equal(t, expected, s.ColumnRuleColour)
		}},

		{"text-shadow", "1px 1px 2px black", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.TextShadow)
		}},

		{"outline", "2px solid red", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 1.5, s.OutlineWidth, 0.001)
		}},

		{"outline-width", "3px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 2.25, s.OutlineWidth, 0.001)
		}},
		{"outline-style", "dashed", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderStyleDashed, s.OutlineStyle)
		}},
		{"outline-color", "blue", func(t *testing.T, s *ComputedStyle) {
			expected, _ := ParseColour("blue")
			assert.Equal(t, expected, s.OutlineColour)
		}},
		{"outline-offset", "4px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 3.0, s.OutlineOffset, 0.001)
		}},

		{"background-image", "url(test.png)", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, 1, len(s.BgImages))
			assert.Equal(t, BackgroundImageURL, s.BgImages[0].Type)
		}},

		{"background-size", "cover", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "cover", s.BgSize)
		}},
		{"background-position", "center center", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "center center", s.BgPosition)
		}},
		{"background-repeat", "no-repeat", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "no-repeat", s.BgRepeat)
		}},
		{"background-attachment", "fixed", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "fixed", s.BgAttachment)
		}},
		{"background-origin", "border-box", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "border-box", s.BgOrigin)
		}},
		{"background-clip", "padding-box", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "padding-box", s.BgClip)
		}},

		{"object-fit", "cover", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, ObjectFitCover, s.ObjectFit)
		}},

		{"object-position", "center top", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "center top", s.ObjectPosition)
		}},

		{"border-image-source", "url(border.png)", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "url(border.png)", s.BorderImageSource)
		}},
		{"border-image-slice", "30", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 30.0, s.BorderImageSlice, 0.001)
		}},
		{"border-image-width", "10px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 7.5, s.BorderImageWidth, 0.001)
		}},
		{"border-image-outset", "5px", func(t *testing.T, s *ComputedStyle) {
			assert.InDelta(t, 3.75, s.BorderImageOutset, 0.001)
		}},
		{"border-image-repeat", "round", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, BorderImageRepeatRound, s.BorderImageRepeat)
		}},

		{"clip-path", "circle(50%)", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "circle(50%)", s.ClipPath)
		}},

		{"mask-image", "url(mask.svg)", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, "url(mask.svg)", s.MaskImage)
		}},

		{"direction", "rtl", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, DirectionRTL, s.Direction)
		}},

		{"hyphens", "auto", func(t *testing.T, s *ComputedStyle) {
			assert.Equal(t, HyphensAuto, s.Hyphens)
		}},

		{"counter-reset", "section 0", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.CounterReset)
			assert.Equal(t, "section", s.CounterReset[0].Name)
		}},

		{"counter-increment", "section 1", func(t *testing.T, s *ComputedStyle) {
			assert.NotEmpty(t, s.CounterIncrement)
			assert.Equal(t, "section", s.CounterIncrement[0].Name)
		}},
	}

	ctx := defaultResolutionContext()
	for _, tc := range testCases {
		t.Run(tc.property, func(t *testing.T) {
			style := DefaultComputedStyle()
			applyProperty(&style, tc.property, tc.value, ctx, nil)
			tc.verify(t, &style)
		})
	}
}
