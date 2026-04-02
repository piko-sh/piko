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

func TestDimensionConstructors(t *testing.T) {
	tests := []struct {
		name         string
		dimension    Dimension
		expectedVal  float64
		expectedUnit DimensionUnit
	}{
		{
			name:         "auto",
			dimension:    DimensionAuto(),
			expectedVal:  0,
			expectedUnit: DimensionUnitAuto,
		},
		{
			name:         "points",
			dimension:    DimensionPt(10),
			expectedVal:  10,
			expectedUnit: DimensionUnitPoints,
		},
		{
			name:         "percentage",
			dimension:    DimensionPct(50),
			expectedVal:  50,
			expectedUnit: DimensionUnitPercentage,
		},
		{
			name:         "min-content",
			dimension:    DimensionMinContent(),
			expectedVal:  0,
			expectedUnit: DimensionUnitMinContent,
		},
		{
			name:         "max-content",
			dimension:    DimensionMaxContent(),
			expectedVal:  0,
			expectedUnit: DimensionUnitMaxContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expectedVal, tt.dimension.Value, 0.001)
			assert.Equal(t, tt.expectedUnit, tt.dimension.Unit)
		})
	}
}

func TestDimension_IsAuto(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		expected  bool
	}{
		{name: "auto", dimension: DimensionAuto(), expected: true},
		{name: "points", dimension: DimensionPt(10), expected: false},
		{name: "percentage", dimension: DimensionPct(50), expected: false},
		{name: "min-content", dimension: DimensionMinContent(), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dimension.IsAuto())
		})
	}
}

func TestDimension_IsMinContent(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		expected  bool
	}{
		{name: "min-content", dimension: DimensionMinContent(), expected: true},
		{name: "max-content", dimension: DimensionMaxContent(), expected: false},
		{name: "points", dimension: DimensionPt(10), expected: false},
		{name: "auto", dimension: DimensionAuto(), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dimension.IsMinContent())
		})
	}
}

func TestDimension_IsMaxContent(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		expected  bool
	}{
		{name: "max-content", dimension: DimensionMaxContent(), expected: true},
		{name: "min-content", dimension: DimensionMinContent(), expected: false},
		{name: "points", dimension: DimensionPt(10), expected: false},
		{name: "auto", dimension: DimensionAuto(), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dimension.IsMaxContent())
		})
	}
}

func TestDimension_IsIntrinsic(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		expected  bool
	}{
		{name: "min-content is intrinsic", dimension: DimensionMinContent(), expected: true},
		{name: "max-content is intrinsic", dimension: DimensionMaxContent(), expected: true},
		{name: "points is not intrinsic", dimension: DimensionPt(10), expected: false},
		{name: "auto is not intrinsic", dimension: DimensionAuto(), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dimension.IsIntrinsic())
		})
	}
}

func TestDimension_Resolve(t *testing.T) {
	tests := []struct {
		name                string
		dimension           Dimension
		containingBlockSize float64
		fallback            float64
		expected            float64
	}{
		{

			name:                "points resolve directly",
			dimension:           DimensionPt(10),
			containingBlockSize: 200,
			fallback:            0,
			expected:            10,
		},
		{

			name:                "percentage resolves against containing block",
			dimension:           DimensionPct(50),
			containingBlockSize: 200,
			fallback:            0,
			expected:            100,
		},
		{

			name:                "auto returns fallback",
			dimension:           DimensionAuto(),
			containingBlockSize: 200,
			fallback:            42,
			expected:            42,
		},
		{

			name:                "min-content returns fallback",
			dimension:           DimensionMinContent(),
			containingBlockSize: 200,
			fallback:            0,
			expected:            0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dimension.Resolve(tt.containingBlockSize, tt.fallback)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestDimension_String(t *testing.T) {
	tests := []struct {
		name      string
		expected  string
		dimension Dimension
	}{
		{name: "auto", dimension: DimensionAuto(), expected: "auto"},
		{name: "points", dimension: DimensionPt(10), expected: "10.00pt"},
		{name: "percentage", dimension: DimensionPct(50), expected: "50.00%"},
		{name: "min-content", dimension: DimensionMinContent(), expected: "min-content"},
		{name: "max-content", dimension: DimensionMaxContent(), expected: "max-content"},
		{

			name:      "unknown unit",
			dimension: Dimension{Value: 0, Unit: DimensionUnit(99)},
			expected:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dimension.String())
		})
	}
}

func TestDefaultComputedStyle(t *testing.T) {
	style := DefaultComputedStyle()

	assert.Equal(t, DisplayInline, style.Display)
	assert.Equal(t, PositionStatic, style.Position)

	assert.Equal(t, "serif", style.FontFamily)
	assert.InDelta(t, 12.0, style.FontSize, 0.001)
	assert.Equal(t, 400, style.FontWeight)
	assert.Equal(t, FontStyleNormal, style.FontStyle)
	assert.Equal(t, TextAlignStart, style.TextAlign)

	assert.InDelta(t, 16.8, style.LineHeight, 0.001)
	assert.True(t, style.LineHeightAuto)

	assert.Equal(t, ColourBlack, style.Colour)
	assert.Equal(t, ColourTransparent, style.BackgroundColour)

	assert.Equal(t, FlexDirectionRow, style.FlexDirection)
	assert.InDelta(t, 0.0, style.FlexGrow, 0.001)
	assert.InDelta(t, 1.0, style.FlexShrink, 0.001)
	assert.True(t, style.FlexBasis.IsAuto())

	assert.InDelta(t, 1.0, style.Opacity, 0.001)

	assert.True(t, style.Width.IsAuto())
	assert.True(t, style.Height.IsAuto())
	assert.InDelta(t, 0.0, style.MinWidth.Value, 0.001)
	assert.InDelta(t, 0.0, style.MinHeight.Value, 0.001)
	assert.True(t, style.MaxWidth.IsAuto())
	assert.True(t, style.MaxHeight.IsAuto())

	assert.True(t, style.GridColumnStart.IsAuto)
	assert.True(t, style.GridColumnEnd.IsAuto)
	assert.True(t, style.GridRowStart.IsAuto)
	assert.True(t, style.GridRowEnd.IsAuto)

	assert.InDelta(t, 8.0, style.TabSize, 0.001)

	assert.True(t, style.ZIndexAuto)

	assert.Equal(t, 2, style.Orphans)
	assert.Equal(t, 2, style.Widows)
}

func TestInheritedComputedStyle(t *testing.T) {
	parent := DefaultComputedStyle()
	parent.Colour = NewRGBA(1, 0, 0, 1)
	parent.FontSize = 24
	parent.FontFamily = "monospace"
	parent.FontWeight = 700
	parent.TextAlign = TextAlignCentre
	parent.Visibility = VisibilityHidden

	child := parent.InheritedComputedStyle()

	assert.InDelta(t, 1.0, child.Colour.Red, 0.001)
	assert.InDelta(t, 0.0, child.Colour.Green, 0.001)
	assert.InDelta(t, 0.0, child.Colour.Blue, 0.001)
	assert.InDelta(t, 24.0, child.FontSize, 0.001)
	assert.Equal(t, "monospace", child.FontFamily)
	assert.Equal(t, 700, child.FontWeight)
	assert.Equal(t, TextAlignCentre, child.TextAlign)
	assert.Equal(t, VisibilityHidden, child.Visibility)

	assert.Equal(t, DisplayInline, child.Display)
	assert.Equal(t, PositionStatic, child.Position)
	assert.True(t, child.Width.IsAuto())
	assert.True(t, child.Height.IsAuto())
	assert.InDelta(t, 0.0, child.PaddingTop, 0.001)
	assert.InDelta(t, 0.0, child.BorderTopWidth, 0.001)
	assert.InDelta(t, 1.0, child.Opacity, 0.001)
	assert.Equal(t, ColourTransparent, child.BackgroundColour)
}

func TestDefaultGridLine(t *testing.T) {
	gl := DefaultGridLine()
	assert.True(t, gl.IsAuto)
	assert.Equal(t, 0, gl.Line)
	assert.Equal(t, 0, gl.Span)
}
