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
	"fmt"
	"strings"
	"testing"
)

type coreTestItem struct {
	ID         string
	Name       string
	Expandable bool
	Lines      int
}

func newTestRenderer() *SimpleRenderer[coreTestItem] {
	return &SimpleRenderer[coreTestItem]{
		GetIDFunction: func(item coreTestItem) string { return item.ID },
		MatchesFilterFunction: func(item coreTestItem, query string) bool {
			return strings.Contains(strings.ToLower(item.Name), query)
		},
		RenderRowFunction: func(item coreTestItem, _ int, _, _ bool, _ int) string {
			return item.Name
		},
		IsExpandableFunction: func(item coreTestItem) bool {
			return item.Expandable
		},
		ExpandedCountFunction: func(item coreTestItem) int {
			return item.Lines
		},
		RenderExpandedFunction: func(item coreTestItem, _ int) []string {
			lines := make([]string, item.Lines)
			for i := range lines {
				lines[i] = fmt.Sprintf("  detail-%d", i)
			}
			return lines
		},
	}
}

func newTestViewer(items []coreTestItem) *AssetViewer[coreTestItem] {
	v := NewAssetViewer(AssetViewerConfig[coreTestItem]{
		ID:           "test",
		Title:        "Test",
		Renderer:     newTestRenderer(),
		NavMode:      NavigationSkipLine,
		EnableSearch: true,
	})
	v.SetItems(items)
	v.SetSize(80, 24)
	v.SetFocused(true)
	return v
}

func sampleItems() []coreTestItem {
	return []coreTestItem{
		{ID: "a", Name: "Alpha", Expandable: true, Lines: 2},
		{ID: "b", Name: "Bravo", Expandable: true, Lines: 1},
		{ID: "c", Name: "Charlie", Expandable: false, Lines: 0},
	}
}

func TestAssetViewer_SetItems_Items(t *testing.T) {
	t.Parallel()

	v := newTestViewer(nil)
	if v.ItemCount() != 0 {
		t.Errorf("ItemCount() = %d, want 0", v.ItemCount())
	}

	items := sampleItems()
	v.SetItems(items)
	if v.ItemCount() != 3 {
		t.Errorf("ItemCount() = %d, want 3", v.ItemCount())
	}

	got := v.Items()
	if len(got) != 3 {
		t.Fatalf("Items() len = %d, want 3", len(got))
	}
	if got[0].ID != "a" {
		t.Errorf("Items()[0].ID = %q, want %q", got[0].ID, "a")
	}
}

func TestAssetViewer_GetDisplayItems(t *testing.T) {
	t.Parallel()

	t.Run("no search returns all indices", func(t *testing.T) {
		t.Parallel()
		v := NewAssetViewer(AssetViewerConfig[coreTestItem]{
			ID:       "test",
			Renderer: newTestRenderer(),
		})
		v.SetItems(sampleItems())

		got := v.GetDisplayItems()
		want := []int{0, 1, 2}
		assertIntSlice(t, got, want)
	})

	t.Run("with search returns filtered", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())

		v.Search().SetQuery("al")
		v.updateFilter()

		got := v.GetDisplayItems()

		want := []int{0}
		assertIntSlice(t, got, want)
	})
}

func TestAssetViewer_Expansion(t *testing.T) {
	t.Parallel()

	t.Run("toggle expansion", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())

		if v.IsExpanded("a") {
			t.Error("should not be expanded initially")
		}

		v.ToggleExpanded("a")
		if !v.IsExpanded("a") {
			t.Error("should be expanded after toggle")
		}

		v.ToggleExpanded("a")
		if v.IsExpanded("a") {
			t.Error("should be collapsed after second toggle")
		}
	})

	t.Run("set expansion", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())

		v.SetExpanded("b", true)
		if !v.IsExpanded("b") {
			t.Error("should be expanded after SetExpanded(true)")
		}

		v.SetExpanded("b", false)
		if v.IsExpanded("b") {
			t.Error("should be collapsed after SetExpanded(false)")
		}
	})

	t.Run("collapse all", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())
		v.SetExpanded("a", true)
		v.SetExpanded("b", true)

		v.CollapseAll()
		if v.IsExpanded("a") || v.IsExpanded("b") {
			t.Error("all items should be collapsed after CollapseAll")
		}
	})

	t.Run("expanded map round-trip", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())
		v.SetExpanded("a", true)
		v.SetExpanded("c", true)

		m := v.ExpandedMap()
		if !m["a"] || !m["c"] {
			t.Error("ExpandedMap should reflect set state")
		}

		v.CollapseAll()
		v.SetExpandedMap(m)
		if !v.IsExpanded("a") || !v.IsExpanded("c") {
			t.Error("SetExpandedMap should restore expansion state")
		}
	})
}

func TestAssetViewer_UpdateFilter(t *testing.T) {
	t.Parallel()

	v := newTestViewer(sampleItems())

	v.cursor = 2
	v.Search().SetQuery("bravo")
	v.updateFilter()

	got := v.GetDisplayItems()
	want := []int{1}
	assertIntSlice(t, got, want)

	if v.cursor > 0 {
		t.Errorf("cursor = %d, want 0 (clamped to filtered len - 1)", v.cursor)
	}
}

func TestAssetViewer_Update_EnterToggle(t *testing.T) {
	t.Parallel()

	v := newTestViewer(sampleItems())

	message := createTestKeyMessage("space")
	handled, _ := v.Update(message)
	if !handled {
		t.Error("space key should be handled")
	}
	if !v.IsExpanded("a") {
		t.Error("item 'a' should be expanded after space")
	}
}

func TestAssetViewer_Update_EnterNonExpandable(t *testing.T) {
	t.Parallel()

	items := []coreTestItem{
		{ID: "x", Name: "X", Expandable: false},
	}
	v := newTestViewer(items)

	message := createTestKeyMessage("enter")
	handled, _ := v.Update(message)
	if handled {
		t.Error("enter on non-expandable item should not be handled")
	}
}

func TestAssetViewer_Update_EscClearsSearchFirst(t *testing.T) {
	t.Parallel()

	v := newTestViewer(sampleItems())
	v.Search().SetQuery("test")
	v.SetExpanded("a", true)

	message := createTestKeyMessage("esc")
	handled, _ := v.Update(message)
	if !handled {
		t.Error("esc should be handled")
	}
	if v.Search().HasQuery() {
		t.Error("esc should clear search query first")
	}
	if !v.IsExpanded("a") {
		t.Error("expansion should NOT be cleared on first esc (search cleared first)")
	}
}

func TestAssetViewer_Update_EscCollapsesAll(t *testing.T) {
	t.Parallel()

	v := newTestViewer(sampleItems())
	v.SetExpanded("a", true)
	v.SetExpanded("b", true)

	message := createTestKeyMessage("esc")
	handled, _ := v.Update(message)
	if !handled {
		t.Error("esc should be handled")
	}
	if v.IsExpanded("a") || v.IsExpanded("b") {
		t.Error("esc should collapse all when no search is active")
	}
}

func TestAssetViewer_GetItemAtCursor(t *testing.T) {
	t.Parallel()

	t.Run("returns item at cursor", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(sampleItems())
		item := v.GetItemAtCursor()
		if item == nil {
			t.Fatal("GetItemAtCursor() returned nil")
		}
		if item.ID != "a" {
			t.Errorf("GetItemAtCursor().ID = %q, want %q", item.ID, "a")
		}
	})

	t.Run("returns nil when empty", func(t *testing.T) {
		t.Parallel()
		v := newTestViewer(nil)
		item := v.GetItemAtCursor()
		if item != nil {
			t.Errorf("GetItemAtCursor() = %v, want nil", item)
		}
	})
}

func TestAssetViewer_WithMutex(t *testing.T) {
	t.Parallel()

	v := NewAssetViewer(AssetViewerConfig[coreTestItem]{
		ID:       "test",
		Title:    "Test",
		Renderer: newTestRenderer(),
		UseMutex: true,
	})

	if v.Mutex() == nil {
		t.Fatal("Mutex() should not be nil when UseMutex is true")
	}

	v.SetItems(sampleItems())
	if v.ItemCount() != 3 {
		t.Errorf("ItemCount() = %d, want 3", v.ItemCount())
	}
}

func TestAssetViewer_CalculateLineCount(t *testing.T) {
	t.Parallel()

	v := newTestViewer(sampleItems())

	t.Run("all collapsed", func(t *testing.T) {
		got := v.CalculateLineCount()
		if got != 3 {
			t.Errorf("CalculateLineCount() = %d, want 3", got)
		}
	})

	t.Run("one expanded", func(t *testing.T) {
		v.SetExpanded("a", true)
		got := v.CalculateLineCount()

		if got != 5 {
			t.Errorf("CalculateLineCount() = %d, want 5", got)
		}
		v.SetExpanded("a", false)
	})
}
