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

func TestAdjustForBoxSizing(t *testing.T) {
	tests := []struct {
		name       string
		style      ComputedStyle
		declared   float64
		expected   float64
		horizontal bool
	}{
		{
			name:     "content-box returns declared value unchanged",
			declared: 200,
			style: ComputedStyle{
				BoxSizing: BoxSizingContentBox,
			},
			horizontal: true,
			expected:   200,
		},
		{
			name:     "border-box horizontal subtracts padding and border",
			declared: 200,
			style: ComputedStyle{
				BoxSizing:        BoxSizingBorderBox,
				PaddingLeft:      10,
				PaddingRight:     10,
				BorderLeftWidth:  5,
				BorderRightWidth: 5,
			},
			horizontal: true,
			expected:   170,
		},
		{
			name:     "border-box vertical subtracts padding and border",
			declared: 200,
			style: ComputedStyle{
				BoxSizing:         BoxSizingBorderBox,
				PaddingTop:        10,
				PaddingBottom:     10,
				BorderTopWidth:    5,
				BorderBottomWidth: 5,
			},
			horizontal: false,
			expected:   170,
		},
		{
			name:     "border-box edges exceed declared floors at zero",
			declared: 20,
			style: ComputedStyle{
				BoxSizing:        BoxSizingBorderBox,
				PaddingLeft:      10,
				PaddingRight:     10,
				BorderLeftWidth:  10,
				BorderRightWidth: 10,
			},
			horizontal: true,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adjustForBoxSizing(tt.declared, &tt.style, tt.horizontal)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestCollapseMargins(t *testing.T) {
	tests := []struct {
		name     string
		marginA  float64
		marginB  float64
		expected float64
	}{
		{
			name:     "both positive takes the larger",
			marginA:  10,
			marginB:  20,
			expected: 20,
		},
		{
			name:     "both negative takes the most negative",
			marginA:  -10,
			marginB:  -20,
			expected: -20,
		},
		{
			name:     "positive and negative sums them",
			marginA:  10,
			marginB:  -5,
			expected: 5,
		},
		{
			name:     "negative and positive sums them",
			marginA:  -10,
			marginB:  5,
			expected: -5,
		},
		{
			name:     "both zero returns zero",
			marginA:  0,
			marginB:  0,
			expected: 0,
		},
		{
			name:     "one zero one positive takes the positive",
			marginA:  0,
			marginB:  10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collapseMargins(tt.marginA, tt.marginB)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestResolveAutoWidthFromStyle(t *testing.T) {
	tests := []struct {
		name            string
		style           ComputedStyle
		availableWidth  float64
		horizontalEdges float64
		expected        resolvedWidth
	}{
		{
			name: "width auto with zero margins fills available space",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MarginLeft = DimensionPt(0)
				s.MarginRight = DimensionPt(0)
				return s
			}(),
			availableWidth:  500,
			horizontalEdges: 30,
			expected: resolvedWidth{
				ContentWidth: 470,
				MarginLeft:   0,
				MarginRight:  0,
			},
		},
		{
			name: "width auto with explicit margins subtracts them",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MarginLeft = DimensionPt(20)
				s.MarginRight = DimensionPt(20)
				return s
			}(),
			availableWidth:  500,
			horizontalEdges: 30,
			expected: resolvedWidth{
				ContentWidth: 430,
				MarginLeft:   20,
				MarginRight:  20,
			},
		},
		{
			name: "width auto content would be negative floors at zero",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MarginLeft = DimensionPt(300)
				s.MarginRight = DimensionPt(300)
				return s
			}(),
			availableWidth:  500,
			horizontalEdges: 30,
			expected: resolvedWidth{
				ContentWidth: 0,
				MarginLeft:   300,
				MarginRight:  300,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAutoWidthFromStyle(&tt.style, tt.availableWidth, tt.horizontalEdges)
			assert.InDelta(t, tt.expected.ContentWidth, result.ContentWidth, 0.001)
			assert.InDelta(t, tt.expected.MarginLeft, result.MarginLeft, 0.001)
			assert.InDelta(t, tt.expected.MarginRight, result.MarginRight, 0.001)
		})
	}
}

func TestResolveExplicitWidthFromStyle(t *testing.T) {
	tests := []struct {
		name                     string
		style                    ComputedStyle
		availableWidth           float64
		horizontalEdges          float64
		containingBlockDirection DirectionType
		expected                 resolvedWidth
	}{
		{
			name: "both margins auto splits remaining space equally",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Width = DimensionPt(200)
				s.MarginLeft = DimensionAuto()
				s.MarginRight = DimensionAuto()
				return s
			}(),
			availableWidth:           500,
			horizontalEdges:          30,
			containingBlockDirection: DirectionLTR,

			expected: resolvedWidth{
				ContentWidth: 200,
				MarginLeft:   135,
				MarginRight:  135,
			},
		},
		{
			name: "left margin auto right margin explicit absorbs remaining space",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Width = DimensionPt(200)
				s.MarginLeft = DimensionAuto()
				s.MarginRight = DimensionPt(20)
				return s
			}(),
			availableWidth:           500,
			horizontalEdges:          30,
			containingBlockDirection: DirectionLTR,

			expected: resolvedWidth{
				ContentWidth: 200,
				MarginLeft:   250,
				MarginRight:  20,
			},
		},
		{
			name: "both margins explicit with zero values",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Width = DimensionPt(200)
				s.MarginLeft = DimensionPt(0)
				s.MarginRight = DimensionPt(0)
				return s
			}(),
			availableWidth:           500,
			horizontalEdges:          30,
			containingBlockDirection: DirectionLTR,
			expected: resolvedWidth{
				ContentWidth: 200,
				MarginLeft:   0,
				MarginRight:  0,
			},
		},
		{
			name: "RTL direction with both margins set adds remaining to left margin",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Width = DimensionPt(200)
				s.MarginLeft = DimensionPt(10)
				s.MarginRight = DimensionPt(10)
				return s
			}(),
			availableWidth:           500,
			horizontalEdges:          30,
			containingBlockDirection: DirectionRTL,

			expected: resolvedWidth{
				ContentWidth: 200,
				MarginLeft:   260,
				MarginRight:  10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveExplicitWidthFromStyle(&tt.style, tt.availableWidth, tt.horizontalEdges, tt.containingBlockDirection)
			assert.InDelta(t, tt.expected.ContentWidth, result.ContentWidth, 0.001)
			assert.InDelta(t, tt.expected.MarginLeft, result.MarginLeft, 0.001)
			assert.InDelta(t, tt.expected.MarginRight, result.MarginRight, 0.001)
		})
	}
}

func TestResolveEdgesFromStyle(t *testing.T) {
	tests := []struct {
		name                  string
		style                 ComputedStyle
		marginPercentageBasis float64
		expectedPadding       BoxEdges
		expectedBorder        BoxEdges
		expectedMarginTop     float64
		expectedMarginBottom  float64
	}{
		{
			name: "resolves all edge values from style",
			style: ComputedStyle{
				PaddingTop:        10,
				PaddingRight:      20,
				PaddingBottom:     30,
				PaddingLeft:       40,
				BorderTopWidth:    1,
				BorderRightWidth:  2,
				BorderBottomWidth: 3,
				BorderLeftWidth:   4,
				MarginTop:         DimensionPt(5),
				MarginBottom:      DimensionPt(15),
			},
			marginPercentageBasis: 500,
			expectedPadding:       BoxEdges{Top: 10, Right: 20, Bottom: 30, Left: 40},
			expectedBorder:        BoxEdges{Top: 1, Right: 2, Bottom: 3, Left: 4},
			expectedMarginTop:     5,
			expectedMarginBottom:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEdgesFromStyle(&tt.style, tt.marginPercentageBasis)
			assert.InDelta(t, tt.expectedPadding.Top, result.Padding.Top, 0.001)
			assert.InDelta(t, tt.expectedPadding.Right, result.Padding.Right, 0.001)
			assert.InDelta(t, tt.expectedPadding.Bottom, result.Padding.Bottom, 0.001)
			assert.InDelta(t, tt.expectedPadding.Left, result.Padding.Left, 0.001)
			assert.InDelta(t, tt.expectedBorder.Top, result.Border.Top, 0.001)
			assert.InDelta(t, tt.expectedBorder.Right, result.Border.Right, 0.001)
			assert.InDelta(t, tt.expectedBorder.Bottom, result.Border.Bottom, 0.001)
			assert.InDelta(t, tt.expectedBorder.Left, result.Border.Left, 0.001)
			assert.InDelta(t, tt.expectedMarginTop, result.MarginTop, 0.001)
			assert.InDelta(t, tt.expectedMarginBottom, result.MarginBottom, 0.001)
		})
	}
}

func TestCanCollapseParentChildTop(t *testing.T) {
	tests := []struct {
		box      *LayoutBox
		name     string
		edges    resolvedEdges
		expected bool
	}{
		{
			name: "zero padding and border with no BFC collapses",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			edges:    resolvedEdges{},
			expected: true,
		},
		{
			name: "top padding prevents collapsing",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			edges: resolvedEdges{
				Padding: BoxEdges{Top: 1},
			},
			expected: false,
		},
		{
			name: "top border prevents collapsing",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			edges: resolvedEdges{
				Border: BoxEdges{Top: 1},
			},
			expected: false,
		},
		{
			name: "box establishing BFC prevents collapsing",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.OverflowX = OverflowHidden
				return &LayoutBox{
					Type:  BoxBlock,
					Style: s,
				}
			}(),
			edges:    resolvedEdges{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canCollapseParentChildTop(tt.box, tt.edges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanCollapseParentChildBottom(t *testing.T) {
	tests := []struct {
		box      *LayoutBox
		name     string
		edges    resolvedEdges
		expected bool
	}{
		{
			name: "zero padding and border with height auto and no BFC collapses",
			box: &LayoutBox{
				Type: BoxBlock,
				Style: func() ComputedStyle {
					s := DefaultComputedStyle()

					return s
				}(),
			},
			edges:    resolvedEdges{},
			expected: true,
		},
		{
			name: "bottom padding prevents collapsing",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			edges: resolvedEdges{
				Padding: BoxEdges{Bottom: 1},
			},
			expected: false,
		},
		{
			name: "bottom border prevents collapsing",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			edges: resolvedEdges{
				Border: BoxEdges{Bottom: 1},
			},
			expected: false,
		},
		{
			name: "explicit height prevents collapsing",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Height = DimensionPt(100)
				return &LayoutBox{
					Type:  BoxBlock,
					Style: s,
				}
			}(),
			edges:    resolvedEdges{},
			expected: false,
		},
		{
			name: "non-zero min-height prevents collapsing",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.MinHeight = DimensionPt(10)
				return &LayoutBox{
					Type:  BoxBlock,
					Style: s,
				}
			}(),
			edges:    resolvedEdges{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canCollapseParentChildBottom(tt.box, tt.edges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEstablishesBlockFormattingContext(t *testing.T) {
	tests := []struct {
		box      *LayoutBox
		name     string
		expected bool
	}{
		{
			name: "default block box does not establish BFC",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			expected: false,
		},
		{
			name: "overflow-x hidden establishes BFC",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.OverflowX = OverflowHidden
				return &LayoutBox{Type: BoxBlock, Style: s}
			}(),
			expected: true,
		},
		{
			name: "float left establishes BFC",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Float = FloatLeft
				return &LayoutBox{Type: BoxBlock, Style: s}
			}(),
			expected: true,
		},
		{
			name: "position absolute establishes BFC",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionAbsolute
				return &LayoutBox{Type: BoxBlock, Style: s}
			}(),
			expected: true,
		},
		{
			name: "display inline-block establishes BFC",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Display = DisplayInlineBlock
				return &LayoutBox{Type: BoxInlineBlock, Style: s}
			}(),
			expected: true,
		},
		{
			name: "display flex establishes BFC",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Display = DisplayFlex
				return &LayoutBox{Type: BoxFlex, Style: s}
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := establishesBlockFormattingContext(tt.box)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasOnlyInlineChildren(t *testing.T) {
	tests := []struct {
		box      *LayoutBox
		name     string
		expected bool
	}{
		{
			name: "inline children only returns true",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
				Children: []*LayoutBox{
					{Type: BoxInline, Style: DefaultComputedStyle()},
					{Type: BoxTextRun, Style: DefaultComputedStyle()},
					{Type: BoxAnonymousInline, Style: DefaultComputedStyle()},
				},
			},
			expected: true,
		},
		{
			name: "block child among inline children returns false",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
				Children: []*LayoutBox{
					{Type: BoxInline, Style: DefaultComputedStyle()},
					{Type: BoxBlock, Style: DefaultComputedStyle()},
				},
			},
			expected: false,
		},
		{
			name: "no children returns false",
			box: &LayoutBox{
				Type:     BoxBlock,
				Style:    DefaultComputedStyle(),
				Children: nil,
			},
			expected: false,
		},
		{
			name: "inline and list-marker children returns true because markers are skipped",
			box: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
				Children: []*LayoutBox{
					{Type: BoxListMarker, Style: DefaultComputedStyle()},
					{Type: BoxInline, Style: DefaultComputedStyle()},
					{Type: BoxTextRun, Style: DefaultComputedStyle()},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOnlyInlineChildren(tt.box)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveContentHeight(t *testing.T) {
	tests := []struct {
		name               string
		style              ComputedStyle
		intrinsicHeight    float64
		availableBlockSize float64
		expected           float64
	}{
		{
			name:            "height auto returns intrinsic height",
			intrinsicHeight: 150,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				return s
			}(),
			availableBlockSize: 500,
			expected:           150,
		},
		{
			name:            "explicit height 100pt returns 100",
			intrinsicHeight: 150,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Height = DimensionPt(100)
				return s
			}(),
			availableBlockSize: 500,
			expected:           100,
		},
		{
			name:            "height 50 percent with available block size 200 returns 100",
			intrinsicHeight: 150,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Height = DimensionPct(50)
				return s
			}(),
			availableBlockSize: 200,
			expected:           100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveContentHeight(tt.intrinsicHeight, &tt.style, tt.availableBlockSize)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestApplyAspectRatio(t *testing.T) {
	tests := []struct {
		box           *LayoutBox
		name          string
		style         ComputedStyle
		contentWidth  float64
		contentHeight float64
		expected      float64
	}{
		{
			name:          "aspect ratio zero or negative returns content height unchanged",
			contentWidth:  200,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.AspectRatio = 0
				return s
			}(),
			box:      &LayoutBox{Type: BoxBlock},
			expected: 100,
		},
		{
			name:          "height not auto returns content height unchanged",
			contentWidth:  200,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.AspectRatio = 2.0
				s.Height = DimensionPt(100)
				return s
			}(),
			box:      &LayoutBox{Type: BoxBlock},
			expected: 100,
		},
		{
			name:          "aspect ratio auto with replaced box and intrinsic height returns content height unchanged",
			contentWidth:  200,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.AspectRatio = 2.0
				s.AspectRatioAuto = true
				return s
			}(),
			box:      &LayoutBox{Type: BoxReplaced, IntrinsicHeight: 50},
			expected: 100,
		},
		{
			name:          "valid aspect ratio 2.0 with height auto and width 200 returns 100",
			contentWidth:  200,
			contentHeight: 50,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.AspectRatio = 2.0
				return s
			}(),
			box:      &LayoutBox{Type: BoxBlock},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyAspectRatio(tt.contentWidth, tt.contentHeight, &tt.style, tt.box)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestClampDimensions(t *testing.T) {
	tests := []struct {
		name                 string
		style                ComputedStyle
		contentWidth         float64
		contentHeight        float64
		containingBlockWidth float64
		expectedWidth        float64
		expectedHeight       float64
	}{
		{
			name:          "no min or max constraints returns inputs unchanged",
			contentWidth:  150,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()

				return s
			}(),
			containingBlockWidth: 500,
			expectedWidth:        150,
			expectedHeight:       100,
		},
		{
			name:          "min-width 100 clamps content width 50 to 100",
			contentWidth:  50,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MinWidth = DimensionPt(100)
				return s
			}(),
			containingBlockWidth: 500,
			expectedWidth:        100,
			expectedHeight:       100,
		},
		{
			name:          "max-width 100 clamps content width 200 to 100",
			contentWidth:  200,
			contentHeight: 100,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MaxWidth = DimensionPt(100)
				return s
			}(),
			containingBlockWidth: 500,
			expectedWidth:        100,
			expectedHeight:       100,
		},
		{
			name:          "min-height 50 clamps content height 20 to 50",
			contentWidth:  150,
			contentHeight: 20,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MinHeight = DimensionPt(50)
				return s
			}(),
			containingBlockWidth: 500,
			expectedWidth:        150,
			expectedHeight:       50,
		},
		{
			name:          "max-height 200 clamps content height 300 to 200",
			contentWidth:  150,
			contentHeight: 300,
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.MaxHeight = DimensionPt(200)
				return s
			}(),
			containingBlockWidth: 500,
			expectedWidth:        150,
			expectedHeight:       200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultWidth, resultHeight := clampDimensions(tt.contentWidth, tt.contentHeight, &tt.style, tt.containingBlockWidth, 0, &LayoutBox{Style: tt.style}, &mockFontMetrics{})
			assert.InDelta(t, tt.expectedWidth, resultWidth, 0.001, "clamped width")
			assert.InDelta(t, tt.expectedHeight, resultHeight, 0.001, "clamped height")
		})
	}
}

func TestComputeRelativeOffset(t *testing.T) {
	tests := []struct {
		box             *LayoutBox
		name            string
		expectedOffsetX float64
		expectedOffsetY float64
	}{
		{
			name: "position not relative sets no offset",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionStatic
				return &LayoutBox{Style: s}
			}(),
			expectedOffsetX: 0,
			expectedOffsetY: 0,
		},
		{
			name: "left 10pt sets offset x to 10",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionRelative
				s.Left = DimensionPt(10)
				return &LayoutBox{Style: s}
			}(),
			expectedOffsetX: 10,
			expectedOffsetY: 0,
		},
		{
			name: "right 10pt sets offset x to -10",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionRelative
				s.Right = DimensionPt(10)
				return &LayoutBox{Style: s}
			}(),
			expectedOffsetX: -10,
			expectedOffsetY: 0,
		},
		{
			name: "top 5pt sets offset y to 5",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionRelative
				s.Top = DimensionPt(5)
				return &LayoutBox{Style: s}
			}(),
			expectedOffsetX: 0,
			expectedOffsetY: 5,
		},
		{
			name: "bottom 5pt sets offset y to -5",
			box: func() *LayoutBox {
				s := DefaultComputedStyle()
				s.Position = PositionRelative
				s.Bottom = DimensionPt(5)
				return &LayoutBox{Style: s}
			}(),
			expectedOffsetX: 0,
			expectedOffsetY: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computeRelativeOffset(tt.box)
			assert.InDelta(t, tt.expectedOffsetX, tt.box.OffsetX, 0.001, "OffsetX")
			assert.InDelta(t, tt.expectedOffsetY, tt.box.OffsetY, 0.001, "OffsetY")
		})
	}
}

func TestResolveAvailableBlockSize(t *testing.T) {
	tests := []struct {
		name            string
		style           ComputedStyle
		parentBlockSize float64
		expected        float64
	}{
		{
			name: "height auto returns zero",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				return s
			}(),
			parentBlockSize: 500,
			expected:        0,
		},
		{
			name: "explicit height 100pt returns 100",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Height = DimensionPt(100)
				return s
			}(),
			parentBlockSize: 500,
			expected:        100,
		},
		{
			name: "height 50 percent with parent block size 200 returns 100",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Height = DimensionPct(50)
				return s
			}(),
			parentBlockSize: 200,
			expected:        100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAvailableBlockSize(&tt.style, tt.parentBlockSize)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestApplyAllRelativeOffsets(t *testing.T) {
	t.Run("shifts box and descendants by relative offset", func(t *testing.T) {
		child_style := DefaultComputedStyle()
		child := &LayoutBox{
			Type:     BoxBlock,
			Style:    child_style,
			ContentX: 50,
			ContentY: 50,
		}

		parent_style := DefaultComputedStyle()
		parent_style.Position = PositionRelative
		parent_style.Left = DimensionPt(10)
		parent := &LayoutBox{
			Type:     BoxBlock,
			Style:    parent_style,
			ContentX: 0,
			ContentY: 0,
			Children: []*LayoutBox{child},
		}

		applyAllRelativeOffsets(parent)

		assert.InDelta(t, 10.0, parent.ContentX, 0.001, "parent ContentX")
		assert.InDelta(t, 0.0, parent.ContentY, 0.001, "parent ContentY")

		assert.InDelta(t, 60.0, child.ContentX, 0.001, "child ContentX")
		assert.InDelta(t, 50.0, child.ContentY, 0.001, "child ContentY")
	})
}
