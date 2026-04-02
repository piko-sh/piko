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

func TestIsMultiColumnContainer(t *testing.T) {
	tests := []struct {
		box      *LayoutBox
		name     string
		expected bool
	}{
		{
			name: "column count greater than 1 is multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 3,
					ColumnWidth: DimensionAuto(),
				},
			},
			expected: true,
		},
		{
			name: "explicit column width with auto count is multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 0,
					ColumnWidth: DimensionPt(200),
				},
			},
			expected: true,
		},
		{
			name: "both column count and width set is multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 2,
					ColumnWidth: DimensionPt(150),
				},
			},
			expected: true,
		},
		{
			name: "default box with no column properties is not multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 0,
					ColumnWidth: DimensionAuto(),
				},
			},
			expected: false,
		},
		{
			name: "column count of 1 is not multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 1,
					ColumnWidth: DimensionAuto(),
				},
			},
			expected: false,
		},
		{
			name: "column width of zero is not multi-column",
			box: &LayoutBox{
				Style: ComputedStyle{
					ColumnCount: 0,
					ColumnWidth: DimensionPt(0),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMultiColumnContainer(tt.box)
			assert.Equal(t, tt.expected, result)
		})
	}
}
