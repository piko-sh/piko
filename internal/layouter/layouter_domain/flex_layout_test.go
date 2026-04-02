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

func makeFlexItem(baseSize, grow, shrink float64, order int) *flexItem {
	box := &LayoutBox{}
	box.Style.Order = order
	box.Style.AlignSelf = AlignSelfAuto
	box.Style.Width = DimensionAuto()
	box.Style.Height = DimensionAuto()
	box.Style.MarginLeft = DimensionPt(0)
	box.Style.MarginRight = DimensionPt(0)
	box.Style.MarginTop = DimensionPt(0)
	box.Style.MarginBottom = DimensionPt(0)

	return &flexItem{
		box:          box,
		flexBaseSize: baseSize,
		mainSize:     baseSize,
		flexGrow:     grow,
		flexShrink:   shrink,
	}
}

func TestSortItemsByOrder(t *testing.T) {
	tests := []struct {
		name           string
		orders         []int
		expected_order []int
	}{
		{
			name:           "already sorted",
			orders:         []int{1, 2, 3},
			expected_order: []int{1, 2, 3},
		},
		{
			name:           "reverse order",
			orders:         []int{3, 2, 1},
			expected_order: []int{1, 2, 3},
		},
		{
			name:           "equal orders preserve original order (stable sort)",
			orders:         []int{0, 0, 0},
			expected_order: []int{0, 0, 0},
		},
		{
			name:           "negative orders sort first",
			orders:         []int{1, -1, 0},
			expected_order: []int{-1, 0, 1},
		},
		{
			name:           "single item",
			orders:         []int{5},
			expected_order: []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]*flexItem, len(tt.orders))
			for i, order := range tt.orders {
				items[i] = makeFlexItem(100, 0, 1, order)
			}

			sortItemsByOrder(items)

			for i, item := range items {
				assert.Equal(t, tt.expected_order[i], item.box.Style.Order,
					"item at index %d should have order %d", i, tt.expected_order[i])
			}
		})
	}
}

func TestCollectFlexLines(t *testing.T) {
	tests := []struct {
		name              string
		base_sizes        []float64
		expected_per_line []int
		container_main    float64
		main_gap          float64
		expected_lines    int
		is_wrap           bool
	}{
		{
			name:              "nowrap puts all items on a single line",
			base_sizes:        []float64{100, 100, 100},
			container_main:    200,
			main_gap:          0,
			is_wrap:           false,
			expected_lines:    1,
			expected_per_line: []int{3},
		},
		{
			name:              "wrap creates multiple lines when items exceed container",
			base_sizes:        []float64{100, 100, 100},
			container_main:    250,
			main_gap:          0,
			is_wrap:           true,
			expected_lines:    2,
			expected_per_line: []int{2, 1},
		},
		{
			name:              "wrap with gaps causes earlier line breaks",
			base_sizes:        []float64{100, 100, 100},
			container_main:    210,
			main_gap:          20,
			is_wrap:           true,
			expected_lines:    3,
			expected_per_line: []int{1, 1, 1},
		},
		{
			name:              "wrap with all items fitting on one line",
			base_sizes:        []float64{50, 50, 50},
			container_main:    200,
			main_gap:          0,
			is_wrap:           true,
			expected_lines:    1,
			expected_per_line: []int{3},
		},
		{
			name:              "empty items returns empty",
			base_sizes:        []float64{},
			container_main:    200,
			main_gap:          0,
			is_wrap:           false,
			expected_lines:    1,
			expected_per_line: []int{0},
		},
		{
			name:              "single item always fits",
			base_sizes:        []float64{300},
			container_main:    200,
			main_gap:          0,
			is_wrap:           true,
			expected_lines:    1,
			expected_per_line: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]*flexItem, len(tt.base_sizes))
			for i, size := range tt.base_sizes {
				items[i] = makeFlexItem(size, 0, 1, 0)
			}

			lines := collectFlexLines(items, tt.container_main, tt.main_gap, tt.is_wrap)

			assert.Equal(t, tt.expected_lines, len(lines))
			for i, line := range lines {
				assert.Equal(t, tt.expected_per_line[i], len(line.items),
					"line %d should have %d items", i, tt.expected_per_line[i])
			}
		})
	}
}

func TestResolveFlexibleLengths_Grow(t *testing.T) {
	tests := []struct {
		name           string
		base_sizes     []float64
		grow_factors   []float64
		expected_sizes []float64
		free_space     float64
	}{
		{
			name:           "distribute proportionally by flex-grow",
			base_sizes:     []float64{100, 100},
			grow_factors:   []float64{1, 3},
			free_space:     200,
			expected_sizes: []float64{150, 250},
		},
		{
			name:           "no flex-grow keeps base sizes",
			base_sizes:     []float64{100, 100},
			grow_factors:   []float64{0, 0},
			free_space:     200,
			expected_sizes: []float64{100, 100},
		},
		{
			name:           "single item with flex-grow takes all free space",
			base_sizes:     []float64{50},
			grow_factors:   []float64{1},
			free_space:     150,
			expected_sizes: []float64{200},
		},
		{
			name:           "equal flex-grow distributes evenly",
			base_sizes:     []float64{100, 100, 100},
			grow_factors:   []float64{1, 1, 1},
			free_space:     90,
			expected_sizes: []float64{130, 130, 130},
		},
		{
			name:           "mixed grow and no-grow",
			base_sizes:     []float64{100, 100},
			grow_factors:   []float64{0, 2},
			free_space:     100,
			expected_sizes: []float64{100, 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]*flexItem, len(tt.base_sizes))
			totalBase := 0.0
			for i := range tt.base_sizes {
				items[i] = makeFlexItem(tt.base_sizes[i], tt.grow_factors[i], 1, 0)
				totalBase += tt.base_sizes[i]
			}
			line := &flexLine{items: items}

			resolveFlexibleLengths(line, totalBase+tt.free_space, 0, true)

			for i, item := range items {
				assert.InDelta(t, tt.expected_sizes[i], item.mainSize, floatEpsilon,
					"item %d mainSize", i)
				assert.InDelta(t, tt.expected_sizes[i], item.targetMainSize, floatEpsilon,
					"item %d targetMainSize", i)
			}
		})
	}
}

func TestResolveFlexibleLengths_Shrink(t *testing.T) {
	tests := []struct {
		name           string
		base_sizes     []float64
		shrink_factors []float64
		expected_sizes []float64
		free_space     float64
	}{
		{

			name:           "shrink proportionally by flex-shrink x base size",
			base_sizes:     []float64{200, 200},
			shrink_factors: []float64{1, 1},
			free_space:     -100,
			expected_sizes: []float64{150, 150},
		},
		{
			name:           "no flex-shrink keeps base sizes",
			base_sizes:     []float64{200, 200},
			shrink_factors: []float64{0, 0},
			free_space:     -100,
			expected_sizes: []float64{200, 200},
		},
		{

			name:           "different shrink factors and base sizes",
			base_sizes:     []float64{100, 200},
			shrink_factors: []float64{1, 2},
			free_space:     -100,
			expected_sizes: []float64{80, 120},
		},
		{

			name:           "size floors at zero",
			base_sizes:     []float64{50, 50},
			shrink_factors: []float64{1, 1},
			free_space:     -200,
			expected_sizes: []float64{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]*flexItem, len(tt.base_sizes))
			totalBase := 0.0
			for i := range tt.base_sizes {
				items[i] = makeFlexItem(tt.base_sizes[i], 0, tt.shrink_factors[i], 0)
				totalBase += tt.base_sizes[i]
			}
			line := &flexLine{items: items}

			resolveFlexibleLengths(line, totalBase+tt.free_space, 0, true)

			for i, item := range items {
				assert.InDelta(t, tt.expected_sizes[i], item.mainSize, floatEpsilon,
					"item %d mainSize", i)
				assert.InDelta(t, tt.expected_sizes[i], item.targetMainSize, floatEpsilon,
					"item %d targetMainSize", i)
			}
		})
	}
}

func TestDistributeAlignContent(t *testing.T) {
	tests := []struct {
		name             string
		line_cross_sizes []float64
		expected_sizes   []float64
		container_cross  float64
		total_cross_gaps float64
		align_content    AlignContentType
		expected_offset  float64
		expected_spacing float64
	}{
		{

			name:             "stretch distributes extra cross space to lines",
			line_cross_sizes: []float64{50, 50},
			container_cross:  400,
			total_cross_gaps: 0,
			align_content:    AlignContentStretch,
			expected_offset:  0,
			expected_spacing: 0,
			expected_sizes:   []float64{200, 200},
		},
		{
			name:             "flex-start has zero offset",
			line_cross_sizes: []float64{50, 50},
			container_cross:  200,
			total_cross_gaps: 0,
			align_content:    AlignContentFlexStart,
			expected_offset:  0,
			expected_spacing: 0,
			expected_sizes:   []float64{50, 50},
		},
		{

			name:             "flex-end offsets by remaining space",
			line_cross_sizes: []float64{50, 50},
			container_cross:  200,
			total_cross_gaps: 0,
			align_content:    AlignContentFlexEnd,
			expected_offset:  100,
			expected_spacing: 0,
			expected_sizes:   []float64{50, 50},
		},
		{

			name:             "centre offsets by half remaining space",
			line_cross_sizes: []float64{50, 50},
			container_cross:  200,
			total_cross_gaps: 0,
			align_content:    AlignContentCentre,
			expected_offset:  50,
			expected_spacing: 0,
			expected_sizes:   []float64{50, 50},
		},
		{

			name:             "space-between distributes space between lines",
			line_cross_sizes: []float64{40, 30, 30},
			container_cross:  300,
			total_cross_gaps: 0,
			align_content:    AlignContentSpaceBetween,
			expected_offset:  0,
			expected_spacing: 100,
			expected_sizes:   []float64{40, 30, 30},
		},
		{

			name:             "space-around distributes space around lines",
			line_cross_sizes: []float64{50, 50},
			container_cross:  300,
			total_cross_gaps: 0,
			align_content:    AlignContentSpaceAround,
			expected_offset:  50,
			expected_spacing: 100,
			expected_sizes:   []float64{50, 50},
		},
		{

			name:             "no remaining space returns zero result",
			line_cross_sizes: []float64{100, 100},
			container_cross:  200,
			total_cross_gaps: 0,
			align_content:    AlignContentFlexEnd,
			expected_offset:  0,
			expected_spacing: 0,
			expected_sizes:   []float64{100, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := make([]*flexLine, len(tt.line_cross_sizes))
			for i, size := range tt.line_cross_sizes {
				lines[i] = &flexLine{crossSize: size}
			}

			result := distributeAlignContent(lines, tt.container_cross, tt.total_cross_gaps, tt.align_content)

			assert.InDelta(t, tt.expected_offset, result.initialOffset, floatEpsilon, "initialOffset")
			assert.InDelta(t, tt.expected_spacing, result.lineSpacing, floatEpsilon, "lineSpacing")

			for i, line := range lines {
				assert.InDelta(t, tt.expected_sizes[i], line.crossSize, floatEpsilon,
					"line %d crossSize", i)
			}
		})
	}
}

func TestComputeJustifyOffset(t *testing.T) {

	make_line := func(sizes ...float64) *flexLine {
		items := make([]*flexItem, len(sizes))
		for i, s := range sizes {
			items[i] = &flexItem{mainSize: s}
		}
		return &flexLine{items: items}
	}

	tests := []struct {
		line            *flexLine
		name            string
		justify         JustifyContentType
		container_main  float64
		main_gap        float64
		expected_offset float64
	}{
		{
			name:            "flex-start offset is zero",
			justify:         JustifyFlexStart,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 0,
		},
		{

			name:            "flex-end offset equals free space",
			justify:         JustifyFlexEnd,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 200,
		},
		{

			name:            "centre offset is half free space",
			justify:         JustifyCentre,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 100,
		},
		{
			name:            "space-between offset is zero",
			justify:         JustifySpaceBetween,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 0,
		},
		{

			name:            "space-around offset is half per-item spacing",
			justify:         JustifySpaceAround,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 50,
		},
		{

			name:            "space-evenly offset is free_space/(count+1)",
			justify:         JustifySpaceEvenly,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        0,
			expected_offset: 66.667,
		},
		{

			name:            "flex-end accounts for gaps",
			justify:         JustifyFlexEnd,
			container_main:  300,
			line:            make_line(50, 50),
			main_gap:        10,
			expected_offset: 190,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := computeJustifyOffset(tt.justify, tt.container_main, tt.line, tt.main_gap)
			assert.InDelta(t, tt.expected_offset, offset, floatEpsilon)
		})
	}
}

func TestComputeJustifySpacing(t *testing.T) {
	make_line := func(sizes ...float64) *flexLine {
		items := make([]*flexItem, len(sizes))
		for i, s := range sizes {
			items[i] = &flexItem{mainSize: s}
		}
		return &flexLine{items: items}
	}

	tests := []struct {
		line             *flexLine
		name             string
		justify          JustifyContentType
		container_main   float64
		main_gap         float64
		expected_spacing float64
	}{
		{
			name:             "flex-start spacing is zero",
			justify:          JustifyFlexStart,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         0,
			expected_spacing: 0,
		},
		{
			name:             "flex-end spacing is zero",
			justify:          JustifyFlexEnd,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         0,
			expected_spacing: 0,
		},
		{
			name:             "centre spacing is zero",
			justify:          JustifyCentre,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         0,
			expected_spacing: 0,
		},
		{

			name:             "space-between distributes among gaps",
			justify:          JustifySpaceBetween,
			container_main:   300,
			line:             make_line(30, 30, 30),
			main_gap:         0,
			expected_spacing: 105,
		},
		{

			name:             "space-between with one item returns zero",
			justify:          JustifySpaceBetween,
			container_main:   300,
			line:             make_line(50),
			main_gap:         0,
			expected_spacing: 0,
		},
		{

			name:             "space-around distributes per item",
			justify:          JustifySpaceAround,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         0,
			expected_spacing: 100,
		},
		{

			name:             "space-evenly distributes with extra slot",
			justify:          JustifySpaceEvenly,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         0,
			expected_spacing: 66.667,
		},
		{

			name:             "space-between accounts for gaps",
			justify:          JustifySpaceBetween,
			container_main:   300,
			line:             make_line(50, 50),
			main_gap:         20,
			expected_spacing: 180,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spacing := computeJustifySpacing(tt.justify, tt.container_main, tt.line, tt.main_gap)
			assert.InDelta(t, tt.expected_spacing, spacing, floatEpsilon)
		})
	}
}

func TestComputeCrossPosition(t *testing.T) {

	make_cross_item := func(cross_size float64, align_self AlignSelfType) *flexItem {
		box := &LayoutBox{}
		box.Style.AlignSelf = align_self
		box.Style.Width = DimensionAuto()
		box.Style.Height = DimensionAuto()
		box.Style.MarginLeft = DimensionPt(0)
		box.Style.MarginRight = DimensionPt(0)
		box.Style.MarginTop = DimensionPt(0)
		box.Style.MarginBottom = DimensionPt(0)

		return &flexItem{
			box:       box,
			crossSize: cross_size,
			fragment:  &Fragment{},
		}
	}

	tests := []struct {
		item            *flexItem
		name            string
		line_cross_size float64
		cross_offset    float64
		container_align AlignItemsType
		expected_pos    float64
		is_row          bool
	}{
		{

			name:            "stretch returns cross offset",
			item:            make_cross_item(30, AlignSelfStretch),
			line_cross_size: 100,
			cross_offset:    10,
			container_align: AlignItemsStretch,
			is_row:          true,
			expected_pos:    10,
		},
		{

			name:            "flex-start returns cross offset",
			item:            make_cross_item(30, AlignSelfFlexStart),
			line_cross_size: 100,
			cross_offset:    10,
			container_align: AlignItemsStretch,
			is_row:          true,
			expected_pos:    10,
		},
		{

			name:            "flex-end aligns to end of line",
			item:            make_cross_item(30, AlignSelfFlexEnd),
			line_cross_size: 100,
			cross_offset:    10,
			container_align: AlignItemsStretch,
			is_row:          true,
			expected_pos:    80,
		},
		{

			name:            "centre aligns to middle of line",
			item:            make_cross_item(30, AlignSelfCentre),
			line_cross_size: 100,
			cross_offset:    10,
			container_align: AlignItemsStretch,
			is_row:          true,
			expected_pos:    45,
		},
		{

			name:            "auto falls back to container align-items",
			item:            make_cross_item(20, AlignSelfAuto),
			line_cross_size: 80,
			cross_offset:    5,
			container_align: AlignItemsFlexEnd,
			is_row:          true,
			expected_pos:    65,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := computeCrossPosition(tt.item, tt.line_cross_size, tt.cross_offset, tt.container_align, tt.is_row)
			assert.InDelta(t, tt.expected_pos, pos, floatEpsilon)
		})
	}
}

func TestResolveAlignSelf(t *testing.T) {
	tests := []struct {
		name            string
		align_self      AlignSelfType
		container_align AlignItemsType
		expected        AlignItemsType
	}{
		{
			name:            "auto falls back to container align-items flex-start",
			align_self:      AlignSelfAuto,
			container_align: AlignItemsFlexStart,
			expected:        AlignItemsFlexStart,
		},
		{
			name:            "auto falls back to container align-items centre",
			align_self:      AlignSelfAuto,
			container_align: AlignItemsCentre,
			expected:        AlignItemsCentre,
		},
		{
			name:            "flex-start maps to AlignItemsFlexStart",
			align_self:      AlignSelfFlexStart,
			container_align: AlignItemsStretch,
			expected:        AlignItemsFlexStart,
		},
		{
			name:            "flex-end maps to AlignItemsFlexEnd",
			align_self:      AlignSelfFlexEnd,
			container_align: AlignItemsStretch,
			expected:        AlignItemsFlexEnd,
		},
		{
			name:            "centre maps to AlignItemsCentre",
			align_self:      AlignSelfCentre,
			container_align: AlignItemsStretch,
			expected:        AlignItemsCentre,
		},
		{
			name:            "stretch maps to AlignItemsStretch",
			align_self:      AlignSelfStretch,
			container_align: AlignItemsFlexStart,
			expected:        AlignItemsStretch,
		},
		{
			name:            "baseline maps to AlignItemsBaseline",
			align_self:      AlignSelfBaseline,
			container_align: AlignItemsStretch,
			expected:        AlignItemsBaseline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAlignSelf(tt.align_self, tt.container_align)
			assert.Equal(t, tt.expected, result)
		})
	}
}
