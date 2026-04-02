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

func TestRenderCursor(t *testing.T) {
	testCases := []struct {
		name     string
		selected bool
		focused  bool
	}{
		{name: "not selected", selected: false, focused: false},
		{name: "selected not focused", selected: true, focused: false},
		{name: "selected and focused", selected: true, focused: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderCursor(tc.selected, tc.focused)
			if result == "" {
				t.Error("expected non-empty cursor string")
			}
		})
	}
}

func TestRenderCursorWithIndent(t *testing.T) {
	testCases := []struct {
		name           string
		inactiveIndent string
		activeIndent   string
		selected       bool
		focused        bool
	}{
		{
			name:           "not selected uses inactive indent",
			selected:       false,
			focused:        false,
			inactiveIndent: "    ",
			activeIndent:   "  ",
		},
		{
			name:           "selected not focused",
			selected:       true,
			focused:        false,
			inactiveIndent: "    ",
			activeIndent:   "  ",
		},
		{
			name:           "selected and focused",
			selected:       true,
			focused:        true,
			inactiveIndent: "    ",
			activeIndent:   "  ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderCursorWithIndent(tc.selected, tc.focused, tc.inactiveIndent, tc.activeIndent)
			if result == "" {
				t.Error("expected non-empty result")
			}
			if !tc.selected && result != tc.inactiveIndent {
				t.Errorf("not selected should return inactive indent, got %q", result)
			}
		})
	}
}

func TestRenderExpandIndicator(t *testing.T) {
	testCases := []struct {
		name     string
		expanded bool
	}{
		{name: "collapsed", expanded: false},
		{name: "expanded", expanded: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderExpandIndicator(tc.expanded)
			if result == "" {
				t.Error("expected non-empty indicator")
			}
		})
	}
}

func TestRenderName(t *testing.T) {
	testCases := []struct {
		name        string
		inputName   string
		maxWidth    int
		selected    bool
		focused     bool
		expectTrunc bool
	}{
		{
			name:        "short name not truncated",
			inputName:   "test",
			maxWidth:    10,
			selected:    false,
			focused:     false,
			expectTrunc: false,
		},
		{
			name:        "long name truncated",
			inputName:   "very long name that exceeds width",
			maxWidth:    10,
			selected:    false,
			focused:     false,
			expectTrunc: true,
		},
		{
			name:        "selected and focused",
			inputName:   "test",
			maxWidth:    10,
			selected:    true,
			focused:     true,
			expectTrunc: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderName(tc.inputName, tc.maxWidth, tc.selected, tc.focused)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderDimText(t *testing.T) {
	result := RenderDimText("test text")
	if result == "" {
		t.Error("expected non-empty result")
	}

	if !strings.Contains(result, "test text") {
		t.Error("expected result to contain original text")
	}
}

func TestRenderItalicDimText(t *testing.T) {
	result := RenderItalicDimText("test text")
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderErrorText(t *testing.T) {
	result := RenderErrorText("error message")
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderInfoText(t *testing.T) {
	result := RenderInfoText("info message")
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderEmptyState(t *testing.T) {
	testCases := []struct {
		name      string
		itemName  string
		hasFilter bool
	}{
		{name: "no filter", hasFilter: false, itemName: "items"},
		{name: "with filter", hasFilter: true, itemName: "items"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var content strings.Builder
			RenderEmptyState(&content, tc.hasFilter, tc.itemName)
			result := content.String()

			if result == "" {
				t.Error("expected non-empty result")
			}
			if tc.hasFilter && !strings.Contains(result, "filter") {
				t.Error("with filter should mention filter")
			}
		})
	}
}

func TestRenderErrorState(t *testing.T) {
	var content strings.Builder
	RenderErrorState(&content, ErrNoProviders)
	result := content.String()

	if result == "" {
		t.Error("expected non-empty result")
	}
	if !strings.Contains(result, "Error") {
		t.Error("expected error message")
	}
}

func TestScrollContext_WriteLineIfVisible(t *testing.T) {
	testCases := []struct {
		name         string
		scrollOffset int
		visibleLines int
		lineToWrite  int
		shouldRender bool
	}{
		{
			name:         "visible line rendered",
			scrollOffset: 0,
			visibleLines: 5,
			lineToWrite:  2,
			shouldRender: true,
		},
		{
			name:         "line above visible not rendered",
			scrollOffset: 5,
			visibleLines: 5,
			lineToWrite:  2,
			shouldRender: false,
		},
		{
			name:         "line below visible not rendered",
			scrollOffset: 0,
			visibleLines: 3,
			lineToWrite:  5,
			shouldRender: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var content strings.Builder
			ctx := NewScrollContext(&content, tc.scrollOffset, tc.visibleLines)

			for range tc.lineToWrite {
				ctx.WriteLineIfVisible(func() string { return "" })
			}

			called := false
			ctx.WriteLineIfVisible(func() string {
				called = true
				return "test line"
			})

			if called != tc.shouldRender {
				t.Errorf("expected called=%v, got %v", tc.shouldRender, called)
			}
		})
	}
}

func TestScrollContext_WriteLine(t *testing.T) {
	var content strings.Builder
	ctx := NewScrollContext(&content, 0, 3)

	ctx.WriteLine("line 0")
	ctx.WriteLine("line 1")
	ctx.WriteLine("line 2")
	ctx.WriteLine("line 3")

	result := content.String()
	if !strings.Contains(result, "line 0") {
		t.Error("expected line 0")
	}
	if !strings.Contains(result, "line 2") {
		t.Error("expected line 2")
	}
	if strings.Contains(result, "line 3") {
		t.Error("line 3 should not be visible")
	}
}

func TestScrollContext_SkipLines(t *testing.T) {
	var content strings.Builder
	ctx := NewScrollContext(&content, 0, 5)

	ctx.SkipLines(3)
	if ctx.LineIndex() != 3 {
		t.Errorf("expected lineIndex 3, got %d", ctx.LineIndex())
	}

	ctx.SkipLines(2)
	if ctx.LineIndex() != 5 {
		t.Errorf("expected lineIndex 5, got %d", ctx.LineIndex())
	}
}

func TestScrollContext_IsVisible(t *testing.T) {
	var content strings.Builder
	ctx := NewScrollContext(&content, 2, 3)

	testCases := []struct {
		lineIndex int
		visible   bool
	}{
		{lineIndex: 0, visible: false},
		{lineIndex: 1, visible: false},
		{lineIndex: 2, visible: true},
		{lineIndex: 3, visible: true},
		{lineIndex: 4, visible: true},
		{lineIndex: 5, visible: false},
	}

	for _, tc := range testCases {
		ctx.lineIndex = tc.lineIndex
		if ctx.IsVisible() != tc.visible {
			t.Errorf("at lineIndex %d: expected visible=%v, got %v",
				tc.lineIndex, tc.visible, ctx.IsVisible())
		}
	}
}

func TestSimpleRenderer(t *testing.T) {
	type testItem struct {
		id   string
		name string
	}

	renderer := &SimpleRenderer[testItem]{
		GetIDFunction:         func(item testItem) string { return item.id },
		MatchesFilterFunction: func(item testItem, q string) bool { return strings.Contains(item.name, q) },
		RenderRowFunction: func(item testItem, _ int, selected, focused bool, _ int) string {
			return RenderCursor(selected, focused) + item.name
		},
		IsExpandableFunction:  func(_ testItem) bool { return true },
		ExpandedCountFunction: func(_ testItem) int { return 2 },
		RenderExpandedFunction: func(item testItem, _ int) []string {
			return []string{"detail 1", "detail 2"}
		},
	}

	item := testItem{id: "1", name: "test item"}

	t.Run("GetID", func(t *testing.T) {
		if renderer.GetID(item) != "1" {
			t.Error("expected ID '1'")
		}
	})

	t.Run("MatchesFilter", func(t *testing.T) {
		if !renderer.MatchesFilter(item, "test") {
			t.Error("expected to match 'test'")
		}
		if renderer.MatchesFilter(item, "xyz") {
			t.Error("expected not to match 'xyz'")
		}
	})

	t.Run("RenderRow", func(t *testing.T) {
		result := renderer.RenderRow(item, 0, true, true, 80)
		if !strings.Contains(result, "test item") {
			t.Error("expected row to contain item name")
		}
	})

	t.Run("IsExpandable", func(t *testing.T) {
		if !renderer.IsExpandable(item) {
			t.Error("expected expandable")
		}
	})

	t.Run("ExpandedLineCount", func(t *testing.T) {
		if renderer.ExpandedLineCount(item) != 2 {
			t.Error("expected 2 expanded lines")
		}
	})

	t.Run("RenderExpanded", func(t *testing.T) {
		lines := renderer.RenderExpanded(item, 80)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
	})
}

func TestSimpleRenderer_NilOptionalFunctions(t *testing.T) {
	type testItem struct{}

	renderer := &SimpleRenderer[testItem]{
		GetIDFunction:         func(_ testItem) string { return "id" },
		MatchesFilterFunction: func(_ testItem, _ string) bool { return true },
		RenderRowFunction:     func(_ testItem, _ int, _, _ bool, _ int) string { return "row" },
	}

	item := testItem{}

	t.Run("IsExpandable nil returns false", func(t *testing.T) {
		if renderer.IsExpandable(item) {
			t.Error("expected false when IsExpandableFunction is nil")
		}
	})

	t.Run("ExpandedLineCount nil returns 0", func(t *testing.T) {
		if renderer.ExpandedLineCount(item) != 0 {
			t.Error("expected 0 when ExpandedCountFunction is nil")
		}
	})

	t.Run("RenderExpanded nil returns nil", func(t *testing.T) {
		if renderer.RenderExpanded(item, 80) != nil {
			t.Error("expected nil when RenderExpandedFunction is nil")
		}
	})
}
