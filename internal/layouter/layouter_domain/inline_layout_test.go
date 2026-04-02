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

const inlineEpsilon = 0.001

func TestAllowsWrapping(t *testing.T) {
	tests := []struct {
		name       string
		whiteSpace WhiteSpaceType
		expected   bool
	}{
		{
			name:       "normal allows wrapping",
			whiteSpace: WhiteSpaceNormal,
			expected:   true,
		},
		{
			name:       "pre disallows wrapping",
			whiteSpace: WhiteSpacePre,
			expected:   false,
		},
		{
			name:       "nowrap disallows wrapping",
			whiteSpace: WhiteSpaceNowrap,
			expected:   false,
		},
		{
			name:       "pre-wrap allows wrapping",
			whiteSpace: WhiteSpacePreWrap,
			expected:   true,
		},
		{
			name:       "pre-line allows wrapping",
			whiteSpace: WhiteSpacePreLine,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowsWrapping(tt.whiteSpace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreservesWhitespace(t *testing.T) {
	tests := []struct {
		name       string
		whiteSpace WhiteSpaceType
		expected   bool
	}{
		{
			name:       "normal does not preserve whitespace",
			whiteSpace: WhiteSpaceNormal,
			expected:   false,
		},
		{
			name:       "pre preserves whitespace",
			whiteSpace: WhiteSpacePre,
			expected:   true,
		},
		{
			name:       "nowrap does not preserve whitespace",
			whiteSpace: WhiteSpaceNowrap,
			expected:   false,
		},
		{
			name:       "pre-wrap preserves whitespace",
			whiteSpace: WhiteSpacePreWrap,
			expected:   true,
		},
		{
			name:       "pre-line preserves whitespace",
			whiteSpace: WhiteSpacePreLine,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preservesWhitespace(tt.whiteSpace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "two words separated by space",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "leading trailing and multiple spaces are collapsed",
			input:    "  multiple   spaces  ",
			expected: []string{"multiple", "spaces"},
		},
		{
			name:     "empty string returns empty slice",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single word returns single-element slice",
			input:    "single",
			expected: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandSoftHyphens(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "word without soft hyphens is unchanged",
			input:    []string{"hello"},
			expected: []string{"hello"},
		},
		{
			name:     "word with soft hyphen splits into fragments with trailing markers",
			input:    []string{"butter\u00ADfly"},
			expected: []string{"butter\u00AD", "fly"},
		},
		{
			name:     "multiple soft hyphens produce multiple fragments",
			input:    []string{"un\u00ADbreak\u00ADable"},
			expected: []string{"un\u00AD", "break\u00AD", "able"},
		},
		{
			name:     "mixed words with and without soft hyphens",
			input:    []string{"hello", "butter\u00ADfly"},
			expected: []string{"hello", "butter\u00AD", "fly"},
		},
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "leading soft hyphen skips empty part",
			input:    []string{"\u00ADword"},
			expected: []string{"word"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandSoftHyphens(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyJustifyToLine(t *testing.T) {
	tests := []struct {
		name             string
		items            []lineItem
		expected_offsets []float64
		expected_widths  []float64
		line_index       int
		line_count       int
		available_width  float64
		content_offset_x float64
	}{
		{
			name: "last line is not justified",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0, ContentWidth: 50}, x: 0, width: 50},
				{fragment: &Fragment{OffsetX: 50, ContentWidth: 50}, x: 50, width: 50},
			},
			line_index:      1,
			line_count:      2,
			available_width: 200,

			expected_offsets: []float64{0, 50},
			expected_widths:  []float64{50, 50},
		},
		{
			name: "two items distribute free space evenly",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0, ContentWidth: 40}, x: 0, width: 40},
				{fragment: &Fragment{OffsetX: 40, ContentWidth: 60}, x: 40, width: 60},
			},
			line_index:       0,
			line_count:       2,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{0, 140},
			expected_widths:  []float64{140, 60},
		},
		{
			name: "three items with content offset",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 10, ContentWidth: 30}, x: 0, width: 30},
				{fragment: &Fragment{OffsetX: 40, ContentWidth: 30}, x: 30, width: 30},
				{fragment: &Fragment{OffsetX: 70, ContentWidth: 30}, x: 60, width: 30},
			},
			line_index:       0,
			line_count:       3,
			available_width:  200,
			content_offset_x: 10,

			expected_offsets: []float64{10, 95, 180},
			expected_widths:  []float64{85, 85, 30},
		},
		{
			name: "single item stretches to fill available width",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0, ContentWidth: 50}, x: 0, width: 50},
			},
			line_index:       0,
			line_count:       2,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{0},
			expected_widths:  []float64{200},
		},
		{
			name: "no free space does not modify items",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0, ContentWidth: 100}, x: 0, width: 100},
				{fragment: &Fragment{OffsetX: 100, ContentWidth: 100}, x: 100, width: 100},
			},
			line_index:       0,
			line_count:       2,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{0, 100},
			expected_widths:  []float64{100, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line_width := 0.0
			for _, item := range tt.items {
				line_width += item.width
			}
			line := lineBox{items: tt.items, width: line_width}

			applyJustifyToLine(line, tt.line_index, tt.line_count, tt.available_width, tt.content_offset_x)

			for i, item := range line.items {
				assert.InDelta(t, tt.expected_offsets[i], item.fragment.OffsetX, inlineEpsilon, "item %d OffsetX", i)
				assert.InDelta(t, tt.expected_widths[i], item.fragment.ContentWidth, inlineEpsilon, "item %d ContentWidth", i)
			}
		})
	}
}

func TestApplyOffsetToLine(t *testing.T) {
	tests := []struct {
		name             string
		items            []lineItem
		expected_offsets []float64
		line_width       float64
		text_align       TextAlignType
		available_width  float64
		content_offset_x float64
	}{
		{
			name: "centre alignment shifts items by half the free space",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0}, x: 0, width: 40},
				{fragment: &Fragment{OffsetX: 40}, x: 40, width: 60},
			},
			line_width:       100,
			text_align:       TextAlignCentre,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{50, 90},
		},
		{
			name: "right alignment shifts items by full free space",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0}, x: 0, width: 40},
				{fragment: &Fragment{OffsetX: 40}, x: 40, width: 60},
			},
			line_width:       100,
			text_align:       TextAlignRight,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{100, 140},
		},
		{
			name: "centre with content offset applies offset to all items",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 10}, x: 0, width: 100},
			},
			line_width:       100,
			text_align:       TextAlignCentre,
			available_width:  300,
			content_offset_x: 10,

			expected_offsets: []float64{110},
		},
		{
			name: "no free space does not shift items",
			items: []lineItem{
				{fragment: &Fragment{OffsetX: 0}, x: 0, width: 200},
			},
			line_width:       200,
			text_align:       TextAlignRight,
			available_width:  200,
			content_offset_x: 0,

			expected_offsets: []float64{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := lineBox{items: tt.items, width: tt.line_width}

			applyOffsetToLine(line, tt.text_align, tt.available_width, tt.content_offset_x)

			for i, item := range line.items {
				assert.InDelta(t, tt.expected_offsets[i], item.fragment.OffsetX, inlineEpsilon, "item %d OffsetX", i)
			}
		})
	}
}

func TestApplyInlineVerticalAlign(t *testing.T) {
	tests := []struct {
		name            string
		items           []lineItem
		expected_deltay []float64
		line_height     float64
	}{
		{
			name: "baseline items align to maximum baseline",
			items: []lineItem{
				{
					fragment: &Fragment{
						Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBaseline}},
						OffsetY: 0,
						Margin:  BoxEdges{Top: 5},
					},
				},
				{
					fragment: &Fragment{
						Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBaseline}},
						OffsetY: 0,
						Margin:  BoxEdges{Top: 10},
					},
				},
			},
			line_height: 50,

			expected_deltay: []float64{5, 0},
		},
		{
			name: "top-aligned item is not shifted",
			items: []lineItem{
				{
					fragment: &Fragment{
						Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignTop}},
						OffsetY: 0,
					},
				},
			},
			line_height:     50,
			expected_deltay: []float64{0},
		},
		{
			name: "middle-aligned item is centred within line height",
			items: []lineItem{
				{
					fragment: &Fragment{
						Box:           &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignMiddle}},
						OffsetY:       0,
						ContentHeight: 20,
					},
				},
			},
			line_height: 50,

			expected_deltay: []float64{15},
		},
		{
			name: "bottom-aligned item is pushed to line bottom",
			items: []lineItem{
				{
					fragment: &Fragment{
						Box:           &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBottom}},
						OffsetY:       0,
						ContentHeight: 20,
					},
				},
			},
			line_height: 50,

			expected_deltay: []float64{30},
		},
		{
			name: "nil box is skipped without panic",
			items: []lineItem{
				{
					fragment: &Fragment{Box: nil, OffsetY: 0},
				},
			},
			line_height:     50,
			expected_deltay: []float64{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original_offsets := make([]float64, len(tt.items))
			for i, item := range tt.items {
				original_offsets[i] = item.fragment.OffsetY
			}

			applyInlineVerticalAlign(tt.items, tt.line_height)

			for i, item := range tt.items {
				actual_delta := item.fragment.OffsetY - original_offsets[i]
				assert.InDelta(t, tt.expected_deltay[i], actual_delta, inlineEpsilon, "item %d vertical shift", i)
			}
		})
	}
}
