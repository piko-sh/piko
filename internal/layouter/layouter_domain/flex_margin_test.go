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
	"github.com/stretchr/testify/require"
)

func marginTestFlex(direction FlexDirectionType, styles ...func(*ComputedStyle)) (*LayoutBox, *LayoutBox, []*LayoutBox) {
	root := makeRoot(300)
	flexStyle := DefaultComputedStyle()
	flexStyle.Display = DisplayFlex
	flexStyle.FlexDirection = direction
	container := newLayoutBox(BoxFlex, &flexStyle, root)
	root.Children = []*LayoutBox{container}
	items := make([]*LayoutBox, 0, len(styles))
	for _, apply := range styles {
		style := DefaultComputedStyle()
		style.Display = DisplayBlock
		apply(&style)
		item := newLayoutBox(BoxFlexItem, &style, container)
		container.Children = append(container.Children, item)
		items = append(items, item)
	}
	return root, container, items
}

func TestFlexLayout_MainAxisMarginSitsOutsideTheItem(t *testing.T) {
	root, _, items := marginTestFlex(FlexDirectionRow,
		func(s *ComputedStyle) {
			s.Width, s.Height, s.MarginRight = DimensionPt(8), DimensionPt(10), DimensionPt(20)
		},
		func(s *ComputedStyle) { s.Width, s.Height = DimensionPt(30), DimensionPt(10) },
	)
	runLayout(root)

	assert.InDelta(t, 8, items[0].ContentWidth, 0.001)
	assert.InDelta(t, 0, items[0].ContentX, 0.001)
	assert.InDelta(t, 28, items[1].ContentX, 0.001)
	assert.InDelta(t, 30, items[1].ContentWidth, 0.001)
}

func TestFlexLayout_MarginsCountAgainstFreeSpace(t *testing.T) {
	root, _, items := marginTestFlex(FlexDirectionRow,
		func(s *ComputedStyle) {
			s.FlexGrow, s.FlexBasis, s.Height, s.MarginRight = 1, DimensionPt(0), DimensionPt(10), DimensionPt(100)
		},
		func(s *ComputedStyle) { s.FlexGrow, s.FlexBasis, s.Height = 1, DimensionPt(0), DimensionPt(10) },
	)
	runLayout(root)

	assert.InDelta(t, 100, items[0].ContentWidth, 0.001)
	assert.InDelta(t, 200, items[1].ContentX, 0.001)
	assert.InDelta(t, 100, items[1].ContentWidth, 0.001)
}

func TestFlexLayout_ColumnHeightIncludesMarginsAndGaps(t *testing.T) {
	item := func(s *ComputedStyle) { s.Height, s.MarginBottom = DimensionPt(20), DimensionPt(6) }
	root, column, items := marginTestFlex(FlexDirectionColumn, item, item)
	column.Style.RowGap = 10
	runLayout(root)

	require.Len(t, items, 2)
	assert.InDelta(t, 0, items[0].ContentY, 0.001)
	assert.InDelta(t, 36, items[1].ContentY, 0.001, "20 high, a 6 margin and a 10 gap")
	assert.InDelta(t, 62, column.ContentHeight, 0.001)
}

func TestMeasureFlexIntrinsicWidth_IncludesItemMargins(t *testing.T) {
	_, row, _ := marginTestFlex(FlexDirectionRow,
		func(s *ComputedStyle) { s.Width, s.MarginLeft = DimensionPt(40), DimensionPt(5) },
		func(s *ComputedStyle) { s.Width, s.MarginRight = DimensionPt(30), DimensionPt(15) },
	)

	assert.InDelta(t, 90, measureMaxContentWidth(row, &mockFontMetrics{}), 0.001)
}
