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

func TestMeasureTextIntrinsicWidth_MinContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "hello world",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	result := measureTextIntrinsicWidth(box, SizingModeMinContent, font_metrics)
	assert.InDelta(t, 30.0, result, 0.001)
}

func TestMeasureTextIntrinsicWidth_MaxContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "hello world",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	result := measureTextIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 66.0, result, 0.001)
}

func TestMeasureBlockIntrinsicWidth_ExplicitWidth(t *testing.T) {
	test_cases := []struct {
		name     string
		style    ComputedStyle
		expected float64
	}{
		{
			name: "content-box adds padding and border",
			style: ComputedStyle{
				Width:            DimensionPt(200),
				BoxSizing:        BoxSizingContentBox,
				PaddingLeft:      10,
				PaddingRight:     10,
				BorderLeftWidth:  2,
				BorderRightWidth: 2,
			},

			expected: 224,
		},
		{
			name: "border-box returns declared width directly",
			style: ComputedStyle{
				Width:            DimensionPt(200),
				BoxSizing:        BoxSizingBorderBox,
				PaddingLeft:      10,
				PaddingRight:     10,
				BorderLeftWidth:  2,
				BorderRightWidth: 2,
			},

			expected: 200,
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.name, func(t *testing.T) {
			box := &LayoutBox{
				Type:  BoxBlock,
				Style: tc.style,
			}
			font_metrics := &mockFontMetrics{}

			result := measureBlockIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestMeasureBlockIntrinsicWidth_Children(t *testing.T) {

	font_metrics := &mockFontMetrics{}

	child_a := &LayoutBox{
		Type: BoxTextRun,
		Text: "short",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}
	child_b := &LayoutBox{
		Type: BoxTextRun,
		Text: "a longer piece of text",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	box := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			Width:        DimensionAuto(),
			PaddingLeft:  5,
			PaddingRight: 5,
		},
	}

	result := measureBlockIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 142.0, result, 0.001)
}

func TestMeasureFlexIntrinsicWidth_Row(t *testing.T) {

	font_metrics := &mockFontMetrics{}

	child_a := &LayoutBox{
		Type: BoxTextRun,
		Text: "hello",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}
	child_b := &LayoutBox{
		Type: BoxTextRun,
		Text: "world",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	box := &LayoutBox{
		Type:     BoxFlex,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			FlexDirection: FlexDirectionRow,
			ColumnGap:     10,
			PaddingLeft:   4,
			PaddingRight:  4,
		},
	}

	result := measureFlexIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 78.0, result, 0.001)
}

func TestMeasureIntrinsicWidth_Dispatch(t *testing.T) {

	font_metrics := &mockFontMetrics{}

	test_cases := []struct {
		box      *LayoutBox
		name     string
		mode     SizingMode
		expected float64
	}{
		{
			name: "dispatches text run to measureTextIntrinsicWidth",
			box: &LayoutBox{
				Type: BoxTextRun,
				Text: "abc",
				Style: ComputedStyle{
					FontSize: 10,
				},
			},
			mode: SizingModeMaxContent,

			expected: 15,
		},
		{
			name: "dispatches flex to measureFlexIntrinsicWidth",
			box: &LayoutBox{
				Type: BoxFlex,
				Style: ComputedStyle{
					FlexDirection: FlexDirectionRow,
				},
			},
			mode: SizingModeMaxContent,

			expected: 0,
		},
		{
			name: "dispatches block to measureBlockIntrinsicWidth",
			box: &LayoutBox{
				Type: BoxBlock,
				Style: ComputedStyle{
					Width: DimensionAuto(),
				},
			},
			mode: SizingModeMaxContent,

			expected: 0,
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.name, func(t *testing.T) {
			result := measureIntrinsicWidth(tc.box, tc.mode, font_metrics)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}
