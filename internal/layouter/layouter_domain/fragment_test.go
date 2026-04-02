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

func TestFragment_BorderBoxWidth(t *testing.T) {
	frag := &Fragment{
		ContentWidth: 100,
		Padding:      BoxEdges{Left: 10, Right: 10},
		Border:       BoxEdges{Left: 5, Right: 5},
	}

	assert.InDelta(t, 130.0, frag.BorderBoxWidth(), 0.001)
}

func TestFragment_BorderBoxHeight(t *testing.T) {
	frag := &Fragment{
		ContentHeight: 100,
		Padding:       BoxEdges{Top: 10, Bottom: 10},
		Border:        BoxEdges{Top: 5, Bottom: 5},
	}

	assert.InDelta(t, 130.0, frag.BorderBoxHeight(), 0.001)
}

func TestFragment_MarginBoxWidth(t *testing.T) {
	frag := &Fragment{
		ContentWidth: 100,
		Padding:      BoxEdges{Left: 10, Right: 10},
		Border:       BoxEdges{Left: 5, Right: 5},
		Margin:       BoxEdges{Left: 8, Right: 12},
	}

	assert.InDelta(t, 150.0, frag.MarginBoxWidth(), 0.001)
}

func TestFragment_MarginBoxHeight(t *testing.T) {
	frag := &Fragment{
		ContentHeight: 100,
		Padding:       BoxEdges{Top: 10, Bottom: 10},
		Border:        BoxEdges{Top: 5, Bottom: 5},
		Margin:        BoxEdges{Top: 8, Bottom: 12},
	}

	assert.InDelta(t, 150.0, frag.MarginBoxHeight(), 0.001)
}

func TestFragment_InlineSize(t *testing.T) {
	tests := []struct {
		name        string
		frag        *Fragment
		writingMode WritingModeType
		expected    float64
	}{
		{
			"horizontal-tb returns content width",
			&Fragment{ContentWidth: 100, ContentHeight: 50},
			WritingModeHorizontalTB,
			100,
		},
		{
			"vertical-rl returns content height",
			&Fragment{ContentWidth: 100, ContentHeight: 50},
			WritingModeVerticalRL,
			50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.frag.InlineSize(tt.writingMode), 0.001)
		})
	}
}

func TestFragment_BlockSize(t *testing.T) {
	tests := []struct {
		name        string
		frag        *Fragment
		writingMode WritingModeType
		expected    float64
	}{
		{
			"horizontal-tb returns content height",
			&Fragment{ContentWidth: 100, ContentHeight: 50},
			WritingModeHorizontalTB,
			50,
		},
		{
			"vertical-rl returns content width",
			&Fragment{ContentWidth: 100, ContentHeight: 50},
			WritingModeVerticalRL,
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.frag.BlockSize(tt.writingMode), 0.001)
		})
	}
}

func TestFragment_InlineOffset(t *testing.T) {
	tests := []struct {
		name        string
		frag        *Fragment
		writingMode WritingModeType
		expected    float64
	}{
		{
			"horizontal-tb returns offset X",
			&Fragment{OffsetX: 10, OffsetY: 20},
			WritingModeHorizontalTB,
			10,
		},
		{
			"vertical-rl returns offset Y",
			&Fragment{OffsetX: 10, OffsetY: 20},
			WritingModeVerticalRL,
			20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.frag.InlineOffset(tt.writingMode), 0.001)
		})
	}
}

func TestFragment_BlockOffset(t *testing.T) {
	tests := []struct {
		name        string
		frag        *Fragment
		writingMode WritingModeType
		expected    float64
	}{
		{
			"horizontal-tb returns offset Y",
			&Fragment{OffsetX: 10, OffsetY: 20},
			WritingModeHorizontalTB,
			20,
		},
		{
			"vertical-rl returns offset X",
			&Fragment{OffsetX: 10, OffsetY: 20},
			WritingModeVerticalRL,
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.frag.BlockOffset(tt.writingMode), 0.001)
		})
	}
}

func TestWriteFragmentsToBoxTree(t *testing.T) {

	parentBox := &LayoutBox{Type: BoxBlock}
	childBox := &LayoutBox{Type: BoxBlock}

	childFragment := &Fragment{
		Box:           childBox,
		OffsetX:       5,
		OffsetY:       15,
		ContentWidth:  50,
		ContentHeight: 30,
		Padding:       BoxEdges{Top: 2, Right: 3, Bottom: 4, Left: 5},
		Border:        BoxEdges{Top: 1, Right: 1, Bottom: 1, Left: 1},
		Margin:        BoxEdges{Top: 3, Right: 3, Bottom: 3, Left: 3},
	}

	parentFragment := &Fragment{
		Box:           parentBox,
		OffsetX:       10,
		OffsetY:       20,
		ContentWidth:  200,
		ContentHeight: 100,
		Children:      []*Fragment{childFragment},
	}

	writeFragmentsToBoxTree(parentFragment, 100, 200)

	assert.InDelta(t, 110.0, parentBox.ContentX, 0.001)
	assert.InDelta(t, 220.0, parentBox.ContentY, 0.001)
	assert.InDelta(t, 200.0, parentBox.ContentWidth, 0.001)
	assert.InDelta(t, 100.0, parentBox.ContentHeight, 0.001)

	assert.InDelta(t, 115.0, childBox.ContentX, 0.001)
	assert.InDelta(t, 235.0, childBox.ContentY, 0.001)
	assert.InDelta(t, 50.0, childBox.ContentWidth, 0.001)
	assert.InDelta(t, 30.0, childBox.ContentHeight, 0.001)

	assert.InDelta(t, 2.0, childBox.Padding.Top, 0.001)
	assert.InDelta(t, 3.0, childBox.Padding.Right, 0.001)
	assert.InDelta(t, 1.0, childBox.Border.Top, 0.001)
	assert.InDelta(t, 3.0, childBox.Margin.Top, 0.001)
}
