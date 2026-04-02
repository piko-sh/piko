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

package tui_domain

import (
	"strings"
	"testing"
)

func TestHistoryRing(t *testing.T) {
	t.Run("NewHistoryRing creates empty ring", func(t *testing.T) {
		h := NewHistoryRing(10)
		if h.Len() != 0 {
			t.Errorf("expected length 0, got %d", h.Len())
		}
		if h.Capacity() != 10 {
			t.Errorf("expected capacity 10, got %d", h.Capacity())
		}
	})

	t.Run("NewHistoryRing defaults to 1800 for invalid capacity", func(t *testing.T) {
		h := NewHistoryRing(0)
		if h.Capacity() != 1800 {
			t.Errorf("expected capacity 1800, got %d", h.Capacity())
		}
		h = NewHistoryRing(-5)
		if h.Capacity() != 1800 {
			t.Errorf("expected capacity 1800, got %d", h.Capacity())
		}
	})

	t.Run("Append adds values", func(t *testing.T) {
		h := NewHistoryRing(5)
		h.Append(1.0)
		h.Append(2.0)
		h.Append(3.0)

		if h.Len() != 3 {
			t.Errorf("expected length 3, got %d", h.Len())
		}

		values := h.Values()
		expected := []float64{1.0, 2.0, 3.0}
		if len(values) != len(expected) {
			t.Fatalf("expected %d values, got %d", len(expected), len(values))
		}
		for i, v := range expected {
			if values[i] != v {
				t.Errorf("expected values[%d] = %f, got %f", i, v, values[i])
			}
		}
	})

	t.Run("Append evicts oldest when at capacity", func(t *testing.T) {
		h := NewHistoryRing(3)
		h.Append(1.0)
		h.Append(2.0)
		h.Append(3.0)
		h.Append(4.0)

		if h.Len() != 3 {
			t.Errorf("expected length 3, got %d", h.Len())
		}

		values := h.Values()
		expected := []float64{2.0, 3.0, 4.0}
		for i, v := range expected {
			if values[i] != v {
				t.Errorf("expected values[%d] = %f, got %f", i, v, values[i])
			}
		}
	})

	t.Run("Latest returns most recent value", func(t *testing.T) {
		h := NewHistoryRing(5)
		if h.Latest() != 0 {
			t.Errorf("expected 0 for empty ring, got %f", h.Latest())
		}

		h.Append(1.0)
		h.Append(2.0)
		h.Append(3.0)

		if h.Latest() != 3.0 {
			t.Errorf("expected 3.0, got %f", h.Latest())
		}
	})

	t.Run("Stats returns correct values", func(t *testing.T) {
		h := NewHistoryRing(10)
		h.Append(1.0)
		h.Append(5.0)
		h.Append(3.0)

		minVal, maxVal, avg := h.Stats()
		if minVal != 1.0 {
			t.Errorf("expected min 1.0, got %f", minVal)
		}
		if maxVal != 5.0 {
			t.Errorf("expected max 5.0, got %f", maxVal)
		}
		if avg != 3.0 {
			t.Errorf("expected avg 3.0, got %f", avg)
		}
	})

	t.Run("Stats returns zeros for empty ring", func(t *testing.T) {
		h := NewHistoryRing(10)
		minVal, maxVal, avg := h.Stats()
		if minVal != 0 || maxVal != 0 || avg != 0 {
			t.Errorf("expected all zeros, got min=%f, max=%f, avg=%f", minVal, maxVal, avg)
		}
	})

	t.Run("Clear removes all values", func(t *testing.T) {
		h := NewHistoryRing(5)
		h.Append(1.0)
		h.Append(2.0)
		h.Clear()

		if h.Len() != 0 {
			t.Errorf("expected length 0 after clear, got %d", h.Len())
		}
	})

	t.Run("Values returns a copy", func(t *testing.T) {
		h := NewHistoryRing(5)
		h.Append(1.0)
		h.Append(2.0)

		values := h.Values()
		values[0] = 999.0

		original := h.Values()
		if original[0] != 1.0 {
			t.Errorf("modifying returned slice affected original, got %f", original[0])
		}
	})
}

func TestNavigablePositions(t *testing.T) {
	t.Run("Simple navigation returns all positions", func(t *testing.T) {
		isExpanded := func(index int) bool { return false }
		expandedCount := func(index int) int { return 0 }

		positions := NavigablePositions(3, isExpanded, expandedCount, NavigationSimple)
		expected := []int{0, 1, 2}

		if len(positions) != len(expected) {
			t.Fatalf("expected %d positions, got %d", len(expected), len(positions))
		}
		for i, position := range expected {
			if positions[i] != position {
				t.Errorf("expected positions[%d] = %d, got %d", i, position, positions[i])
			}
		}
	})

	t.Run("Simple navigation includes expanded lines", func(t *testing.T) {
		isExpanded := func(index int) bool { return index == 1 }
		expandedCount := func(index int) int {
			if index == 1 {
				return 2
			}
			return 0
		}

		positions := NavigablePositions(3, isExpanded, expandedCount, NavigationSimple)

		expected := []int{0, 1, 2, 3, 4}

		if len(positions) != len(expected) {
			t.Fatalf("expected %d positions, got %d", len(expected), len(positions))
		}
		for i, position := range expected {
			if positions[i] != position {
				t.Errorf("expected positions[%d] = %d, got %d", i, position, positions[i])
			}
		}
	})

	t.Run("Skip-line navigation skips expanded lines", func(t *testing.T) {
		isExpanded := func(index int) bool { return index == 1 }
		expandedCount := func(index int) int {
			if index == 1 {
				return 2
			}
			return 0
		}

		positions := NavigablePositions(3, isExpanded, expandedCount, NavigationSkipLine)

		expected := []int{0, 1, 4}

		if len(positions) != len(expected) {
			t.Fatalf("expected %d positions, got %d", len(expected), len(positions))
		}
		for i, position := range expected {
			if positions[i] != position {
				t.Errorf("expected positions[%d] = %d, got %d", i, position, positions[i])
			}
		}
	})

	t.Run("Empty list returns nil", func(t *testing.T) {
		positions := NavigablePositions(0, nil, nil, NavigationSimple)
		if positions != nil {
			t.Errorf("expected nil, got %v", positions)
		}
	})
}

func TestNextPreviousNavigablePosition(t *testing.T) {
	positions := []int{0, 2, 5, 8}

	t.Run("NextNavigablePosition finds next position", func(t *testing.T) {
		tests := []struct {
			cursor   int
			expected int
		}{
			{cursor: 0, expected: 2},
			{cursor: 1, expected: 2},
			{cursor: 2, expected: 5},
			{cursor: 7, expected: 8},
			{cursor: 8, expected: -1},
			{cursor: 10, expected: -1},
		}

		for _, tc := range tests {
			result := NextNavigablePosition(tc.cursor, positions)
			if result != tc.expected {
				t.Errorf("NextNavigablePosition(%d) = %d, expected %d", tc.cursor, result, tc.expected)
			}
		}
	})

	t.Run("PreviousNavigablePosition finds previous position", func(t *testing.T) {
		tests := []struct {
			cursor   int
			expected int
		}{
			{cursor: 0, expected: -1},
			{cursor: 1, expected: 0},
			{cursor: 2, expected: 0},
			{cursor: 3, expected: 2},
			{cursor: 8, expected: 5},
			{cursor: 10, expected: 8},
		}

		for _, tc := range tests {
			result := PreviousNavigablePosition(tc.cursor, positions)
			if result != tc.expected {
				t.Errorf("PreviousNavigablePosition(%d) = %d, expected %d", tc.cursor, result, tc.expected)
			}
		}
	})
}

func TestAdjustScrollForCursor(t *testing.T) {
	tests := []struct {
		name          string
		cursor        int
		scrollOffset  int
		visibleHeight int
		lineCount     int
		expected      int
	}{
		{
			name:          "cursor visible, no change",
			cursor:        5,
			scrollOffset:  3,
			visibleHeight: 10,
			lineCount:     20,
			expected:      3,
		},
		{
			name:          "cursor above visible, scroll up",
			cursor:        2,
			scrollOffset:  5,
			visibleHeight: 10,
			lineCount:     20,
			expected:      2,
		},
		{
			name:          "cursor below visible, scroll down",
			cursor:        15,
			scrollOffset:  3,
			visibleHeight: 10,
			lineCount:     20,
			expected:      6,
		},
		{
			name:          "cursor at end of list",
			cursor:        18,
			scrollOffset:  0,
			visibleHeight: 10,
			lineCount:     20,
			expected:      9,
		},
		{
			name:          "clamp to max scroll",
			cursor:        19,
			scrollOffset:  0,
			visibleHeight: 10,
			lineCount:     20,
			expected:      10,
		},
		{
			name:          "handle zero visible height",
			cursor:        5,
			scrollOffset:  3,
			visibleHeight: 0,
			lineCount:     20,
			expected:      3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AdjustScrollForCursor(tc.cursor, tc.scrollOffset, tc.visibleHeight, tc.lineCount)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestScrollContext(t *testing.T) {
	t.Run("writes visible lines only", func(t *testing.T) {
		var content strings.Builder
		ctx := NewScrollContext(&content, 2, 3)

		lines := []string{"line0", "line1", "line2", "line3", "line4", "line5"}
		for _, line := range lines {
			l := line
			ctx.WriteLineIfVisible(func() string { return l })
		}

		result := content.String()
		if !strings.Contains(result, "line2") {
			t.Error("expected line2 to be visible")
		}
		if !strings.Contains(result, "line3") {
			t.Error("expected line3 to be visible")
		}
		if !strings.Contains(result, "line4") {
			t.Error("expected line4 to be visible")
		}
		if strings.Contains(result, "line0") {
			t.Error("expected line0 to be invisible")
		}
		if strings.Contains(result, "line1") {
			t.Error("expected line1 to be invisible")
		}
		if strings.Contains(result, "line5") {
			t.Error("expected line5 to be invisible")
		}
	})

	t.Run("handles newlines correctly", func(t *testing.T) {
		var content strings.Builder
		ctx := NewScrollContext(&content, 0, 3)

		ctx.WriteLine("a")
		ctx.WriteLine("b")
		ctx.WriteLine("c")

		result := content.String()
		expected := "a\nb\nc"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("IsVisible returns correct value", func(t *testing.T) {
		var content strings.Builder
		ctx := NewScrollContext(&content, 2, 3)

		ctx.SkipLines(1)
		if ctx.IsVisible() {
			t.Error("line 1 should not be visible")
		}

		ctx.SkipLines(1)
		if !ctx.IsVisible() {
			t.Error("line 2 should be visible")
		}
	})
}

type testItem struct {
	id      string
	name    string
	details []string
}

type testRenderer struct{}

func (r *testRenderer) RenderRow(item testItem, lineIndex int, selected, focused bool, width int) string {
	cursor := RenderCursor(selected, focused)
	return cursor + item.name
}

func (r *testRenderer) RenderExpanded(item testItem, width int) []string {
	return item.details
}

func (r *testRenderer) GetID(item testItem) string {
	return item.id
}

func (r *testRenderer) MatchesFilter(item testItem, query string) bool {
	return strings.Contains(strings.ToLower(item.name), query)
}

func (r *testRenderer) IsExpandable(item testItem) bool {
	return len(item.details) > 0
}

func (r *testRenderer) ExpandedLineCount(item testItem) int {
	return len(item.details)
}

func TestAssetViewer(t *testing.T) {
	t.Run("SetItems updates items", func(t *testing.T) {
		v := NewAssetViewer(AssetViewerConfig[testItem]{
			ID:       "test",
			Title:    "Test",
			Renderer: &testRenderer{},
			NavMode:  NavigationSimple,
		})

		items := []testItem{
			{id: "1", name: "Item 1"},
			{id: "2", name: "Item 2"},
		}
		v.SetItems(items)

		if v.ItemCount() != 2 {
			t.Errorf("expected 2 items, got %d", v.ItemCount())
		}
	})

	t.Run("Expansion state management", func(t *testing.T) {
		v := NewAssetViewer(AssetViewerConfig[testItem]{
			ID:       "test",
			Title:    "Test",
			Renderer: &testRenderer{},
			NavMode:  NavigationSimple,
		})

		if v.IsExpanded("1") {
			t.Error("item should not be expanded initially")
		}

		v.ToggleExpanded("1")
		if !v.IsExpanded("1") {
			t.Error("item should be expanded after toggle")
		}

		v.ToggleExpanded("1")
		if v.IsExpanded("1") {
			t.Error("item should not be expanded after second toggle")
		}

		v.SetExpanded("2", true)
		v.SetExpanded("3", true)
		v.CollapseAll()
		if v.IsExpanded("2") || v.IsExpanded("3") {
			t.Error("all items should be collapsed after CollapseAll")
		}
	})

	t.Run("GetDisplayItems returns correct indices", func(t *testing.T) {
		v := NewAssetViewer(AssetViewerConfig[testItem]{
			ID:           "test",
			Title:        "Test",
			Renderer:     &testRenderer{},
			NavMode:      NavigationSimple,
			EnableSearch: true,
		})

		items := []testItem{
			{id: "1", name: "Apple"},
			{id: "2", name: "Banana"},
			{id: "3", name: "Cherry"},
		}
		v.SetItems(items)

		indices := v.GetDisplayItems()
		if len(indices) != 3 {
			t.Errorf("expected 3 indices, got %d", len(indices))
		}
	})

	t.Run("CalculateLineCount includes expanded content", func(t *testing.T) {
		v := NewAssetViewer(AssetViewerConfig[testItem]{
			ID:       "test",
			Title:    "Test",
			Renderer: &testRenderer{},
			NavMode:  NavigationSimple,
		})

		items := []testItem{
			{id: "1", name: "Item 1", details: []string{"detail1", "detail2"}},
			{id: "2", name: "Item 2", details: []string{"detail3"}},
		}
		v.SetItems(items)

		lineCount := v.CalculateLineCount()
		if lineCount != 2 {
			t.Errorf("expected 2 lines, got %d", lineCount)
		}

		v.ToggleExpanded("1")
		lineCount = v.CalculateLineCount()
		if lineCount != 4 {
			t.Errorf("expected 4 lines, got %d", lineCount)
		}

		v.ToggleExpanded("2")
		lineCount = v.CalculateLineCount()
		if lineCount != 5 {
			t.Errorf("expected 5 lines, got %d", lineCount)
		}
	})
}

func TestSearchMixin(t *testing.T) {
	t.Run("UpdateFilter filters items", func(t *testing.T) {
		m := NewSearchMixin(nil)

		items := []string{"Apple", "Banana", "Cherry", "Apricot"}
		m.SetQuery("ap")
		m.UpdateFilter(len(items), func(index int, query string) bool {
			return strings.Contains(strings.ToLower(items[index]), query)
		})

		filtered := m.FilteredItems()
		if len(filtered) != 2 {
			t.Errorf("expected 2 filtered items, got %d", len(filtered))
		}
		if filtered[0] != 0 || filtered[1] != 3 {
			t.Errorf("expected [0, 3], got %v", filtered)
		}
	})

	t.Run("GetDisplayIndices returns all when no filter", func(t *testing.T) {
		m := NewSearchMixin(nil)

		indices := m.GetDisplayIndices(5)
		if len(indices) != 5 {
			t.Errorf("expected 5 indices, got %d", len(indices))
		}
		for i, index := range indices {
			if index != i {
				t.Errorf("expected indices[%d] = %d, got %d", i, i, index)
			}
		}
	})

	t.Run("HasQuery returns correct value", func(t *testing.T) {
		m := NewSearchMixin(nil)

		if m.HasQuery() {
			t.Error("should not have query initially")
		}

		m.SetQuery("test")
		if !m.HasQuery() {
			t.Error("should have query after SetQuery")
		}

		m.ClearQuery()
		if m.HasQuery() {
			t.Error("should not have query after ClearQuery")
		}
	})
}

func TestStatusFilterMixin(t *testing.T) {
	t.Run("CycleFilter cycles through statuses", func(t *testing.T) {
		m := NewStatusFilterMixin()

		if m.HasFilter() {
			t.Error("should not have filter initially")
		}

		m.CycleFilter()
		if *m.FilterStatus() != ResourceStatusHealthy {
			t.Errorf("expected Healthy, got %v", *m.FilterStatus())
		}

		m.CycleFilter()
		if *m.FilterStatus() != ResourceStatusDegraded {
			t.Errorf("expected Degraded, got %v", *m.FilterStatus())
		}

		m.CycleFilter()
		if *m.FilterStatus() != ResourceStatusUnhealthy {
			t.Errorf("expected Unhealthy, got %v", *m.FilterStatus())
		}

		m.CycleFilter()
		if *m.FilterStatus() != ResourceStatusPending {
			t.Errorf("expected Pending, got %v", *m.FilterStatus())
		}

		m.CycleFilter()
		if m.HasFilter() {
			t.Error("expected nil filter after full cycle")
		}
	})

	t.Run("MatchesFilter returns true when no filter", func(t *testing.T) {
		m := NewStatusFilterMixin()

		if !m.MatchesFilter(ResourceStatusHealthy) {
			t.Error("should match when no filter")
		}
		if !m.MatchesFilter(ResourceStatusUnhealthy) {
			t.Error("should match when no filter")
		}
	})

	t.Run("MatchesFilter returns correct value with filter", func(t *testing.T) {
		m := NewStatusFilterMixin()
		m.CycleFilter()

		if !m.MatchesFilter(ResourceStatusHealthy) {
			t.Error("Healthy should match Healthy filter")
		}
		if m.MatchesFilter(ResourceStatusUnhealthy) {
			t.Error("Unhealthy should not match Healthy filter")
		}
	})
}
