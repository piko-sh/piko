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

func TestResolveDimensionFromOffsets(t *testing.T) {
	test_cases := []struct {
		name     string
		input    dimensionFromOffsetsInput
		expected float64
	}{
		{
			name: "explicit size set returns resolved size (content-box)",
			input: dimensionFromOffsetsInput{
				style:          &ComputedStyle{BoxSizing: BoxSizingContentBox},
				explicitSize:   DimensionPt(200),
				startOffset:    DimensionAuto(),
				endOffset:      DimensionAuto(),
				current:        50,
				containingSize: 800,
				edgesSum:       0,
				horizontal:     true,
			},
			expected: 200,
		},
		{
			name: "both offsets set derives size from containing block",
			input: dimensionFromOffsetsInput{
				style:          &ComputedStyle{BoxSizing: BoxSizingContentBox},
				explicitSize:   DimensionAuto(),
				startOffset:    DimensionPt(50),
				endOffset:      DimensionPt(30),
				current:        100,
				containingSize: 800,
				edgesSum:       20,
				horizontal:     true,
			},

			expected: 700,
		},
		{
			name: "neither explicit size nor both offsets returns fallback",
			input: dimensionFromOffsetsInput{
				style:          &ComputedStyle{BoxSizing: BoxSizingContentBox},
				explicitSize:   DimensionAuto(),
				startOffset:    DimensionPt(50),
				endOffset:      DimensionAuto(),
				current:        123,
				containingSize: 800,
				edgesSum:       0,
				horizontal:     true,
			},
			expected: 123,
		},
		{
			name: "border-box with explicit size subtracts padding and border",
			input: dimensionFromOffsetsInput{
				style: &ComputedStyle{
					BoxSizing:        BoxSizingBorderBox,
					PaddingLeft:      10,
					PaddingRight:     10,
					BorderLeftWidth:  2,
					BorderRightWidth: 2,
				},
				explicitSize:   DimensionPt(200),
				startOffset:    DimensionAuto(),
				endOffset:      DimensionAuto(),
				current:        50,
				containingSize: 800,
				edgesSum:       0,
				horizontal:     true,
			},

			expected: 176,
		},
		{
			name: "both offsets set but result would be negative returns zero",
			input: dimensionFromOffsetsInput{
				style:          &ComputedStyle{BoxSizing: BoxSizingContentBox},
				explicitSize:   DimensionAuto(),
				startOffset:    DimensionPt(400),
				endOffset:      DimensionPt(400),
				current:        100,
				containingSize: 500,
				edgesSum:       50,
				horizontal:     true,
			},

			expected: 0,
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveDimensionFromOffsets(tc.input)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestResolvePositionFromOffset(t *testing.T) {
	test_cases := []struct {
		name              string
		start_offset      Dimension
		end_offset        Dimension
		containing_origin float64
		containing_size   float64
		content_size      float64
		start_edges       float64
		end_edges         float64
		expected          float64
	}{
		{
			name:              "start offset set positions from start",
			start_offset:      DimensionPt(20),
			end_offset:        DimensionAuto(),
			containing_origin: 100,
			containing_size:   800,
			content_size:      200,
			start_edges:       5,
			end_edges:         5,

			expected: 125,
		},
		{
			name:              "end offset set positions from end",
			start_offset:      DimensionAuto(),
			end_offset:        DimensionPt(20),
			containing_origin: 100,
			containing_size:   800,
			content_size:      200,
			start_edges:       5,
			end_edges:         10,

			expected: 670,
		},
		{
			name:              "both auto defaults to start-aligned position",
			start_offset:      DimensionAuto(),
			end_offset:        DimensionAuto(),
			containing_origin: 100,
			containing_size:   800,
			content_size:      200,
			start_edges:       15,
			end_edges:         10,

			expected: 115,
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolvePositionFromOffset(
				tc.start_offset,
				tc.end_offset,
				tc.containing_origin,
				tc.containing_size,
				tc.content_size,
				tc.start_edges,
				tc.end_edges,
			)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}
