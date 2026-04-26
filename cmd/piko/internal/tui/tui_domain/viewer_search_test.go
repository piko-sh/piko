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

func TestSearchMixin_QueryLifecycle(t *testing.T) {
	t.Parallel()

	m := NewSearchMixin(nil)

	if m.HasQuery() {
		t.Error("HasQuery() should be false initially")
	}
	if m.Query() != "" {
		t.Errorf("Query() = %q, want empty", m.Query())
	}

	m.SetQuery("hello")
	if !m.HasQuery() {
		t.Error("HasQuery() should be true after SetQuery")
	}
	if m.Query() != "hello" {
		t.Errorf("Query() = %q, want %q", m.Query(), "hello")
	}

	m.ClearQuery()
	if m.HasQuery() {
		t.Error("HasQuery() should be false after ClearQuery")
	}
	if m.Query() != "" {
		t.Errorf("Query() = %q, want empty after ClearQuery", m.Query())
	}
}

func TestSearchMixin_UpdateFilter(t *testing.T) {
	t.Parallel()

	items := []string{"alpha", "bravo", "charlie", "alpine"}

	t.Run("matches subset", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		m.SetQuery("alp")
		m.UpdateFilter(len(items), func(index int, query string) bool {
			return strings.Contains(strings.ToLower(items[index]), query)
		})

		got := m.FilteredItems()
		want := []int{0, 3}
		assertIntSlice(t, got, want)
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		m.SetQuery("zzz")
		m.UpdateFilter(len(items), func(index int, query string) bool {
			return strings.Contains(strings.ToLower(items[index]), query)
		})

		got := m.FilteredItems()
		if len(got) != 0 {
			t.Errorf("FilteredItems() = %v, want empty", got)
		}
	})

	t.Run("empty query clears filter", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		m.SetQuery("alp")
		m.UpdateFilter(len(items), func(index int, query string) bool {
			return strings.Contains(strings.ToLower(items[index]), query)
		})
		if m.FilteredItems() == nil {
			t.Fatal("FilteredItems() should not be nil with active query")
		}

		m.ClearQuery()
		m.UpdateFilter(len(items), func(index int, query string) bool {
			return strings.Contains(strings.ToLower(items[index]), query)
		})
		if m.FilteredItems() != nil {
			t.Errorf("FilteredItems() = %v, want nil after clearing query", m.FilteredItems())
		}
	})
}

func TestSearchMixin_GetDisplayIndices(t *testing.T) {
	t.Parallel()

	t.Run("no filter returns all", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		got := m.GetDisplayIndices(4)
		want := []int{0, 1, 2, 3}
		assertIntSlice(t, got, want)
	})

	t.Run("with filter returns filtered", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		m.SetQuery("x")
		m.UpdateFilter(5, func(index int, _ string) bool {
			return index%2 == 0
		})
		got := m.GetDisplayIndices(5)
		want := []int{0, 2, 4}
		assertIntSlice(t, got, want)
	})

	t.Run("query with no matches returns empty", func(t *testing.T) {
		t.Parallel()
		m := NewSearchMixin(nil)
		m.SetQuery("nothing")
		m.UpdateFilter(5, func(_ int, _ string) bool { return false })
		got := m.GetDisplayIndices(5)
		if len(got) != 0 {
			t.Errorf("GetDisplayIndices() = %v, want empty", got)
		}
	})
}

func TestSearchMixin_AdjustCursorAfterFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		query      string
		filtered   []int
		cursor     int
		wantCursor int
	}{
		{name: "no filter", query: "", cursor: 5, wantCursor: 5},
		{name: "cursor within bounds", query: "x", filtered: []int{0, 1, 2}, cursor: 1, wantCursor: 1},
		{name: "cursor beyond filtered", query: "x", filtered: []int{0, 1}, cursor: 5, wantCursor: 1},
		{name: "empty filtered set", query: "x", filtered: []int{}, cursor: 3, wantCursor: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewSearchMixin(nil)
			m.SetQuery(tc.query)
			m.filteredItems = tc.filtered
			got := m.AdjustCursorAfterFilter(tc.cursor)
			if got != tc.wantCursor {
				t.Errorf("AdjustCursorAfterFilter(%d) = %d, want %d", tc.cursor, got, tc.wantCursor)
			}
		})
	}
}

func TestSearchMixin_OnFilterCallback(t *testing.T) {
	t.Parallel()

	callCount := 0
	m := NewSearchMixin(func() { callCount++ })

	m.searchBox.onClose("test", true)

	if callCount != 1 {
		t.Errorf("onFilter called %d times, want 1", callCount)
	}
	if m.Query() != "test" {
		t.Errorf("Query() = %q, want %q", m.Query(), "test")
	}
}

func TestSearchMixin_OnFilterCallback_Cancelled(t *testing.T) {
	t.Parallel()

	callCount := 0
	m := NewSearchMixin(func() { callCount++ })
	m.SetQuery("previous")

	m.searchBox.onClose("ignored", false)

	if callCount != 1 {
		t.Errorf("onFilter called %d times, want 1", callCount)
	}
	if m.Query() != "" {
		t.Errorf("Query() = %q, want empty after cancel", m.Query())
	}
}

func TestStatusFilterMixin_CycleFilter(t *testing.T) {
	t.Parallel()

	m := NewStatusFilterMixin()

	m.CycleFilter()
	assertStatusFilter(t, m, new(ResourceStatusHealthy), "after first cycle")

	m.CycleFilter()
	assertStatusFilter(t, m, new(ResourceStatusDegraded), "after second cycle")

	m.CycleFilter()
	assertStatusFilter(t, m, new(ResourceStatusUnhealthy), "after third cycle")

	m.CycleFilter()
	assertStatusFilter(t, m, new(ResourceStatusPending), "after fourth cycle")

	m.CycleFilter()
	if m.FilterStatus() != nil {
		t.Errorf("FilterStatus() = %v, want nil after full cycle", m.FilterStatus())
	}
}

func TestStatusFilterMixin_MatchesFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		filter *ResourceStatus
		name   string
		status ResourceStatus
		want   bool
	}{
		{name: "nil filter matches all", filter: nil, status: ResourceStatusHealthy, want: true},
		{name: "nil filter matches unknown", filter: nil, status: ResourceStatusUnknown, want: true},
		{name: "healthy matches healthy", filter: new(ResourceStatusHealthy), status: ResourceStatusHealthy, want: true},
		{name: "healthy rejects degraded", filter: new(ResourceStatusHealthy), status: ResourceStatusDegraded, want: false},
		{name: "pending matches pending", filter: new(ResourceStatusPending), status: ResourceStatusPending, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewStatusFilterMixin()
			m.SetFilterStatus(tc.filter)
			got := m.MatchesFilter(tc.status)
			if got != tc.want {
				t.Errorf("MatchesFilter(%v) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestStatusFilterMixin_HasFilter(t *testing.T) {
	t.Parallel()

	m := NewStatusFilterMixin()
	if m.HasFilter() {
		t.Error("HasFilter() should be false initially")
	}

	m.SetFilterStatus(new(ResourceStatusHealthy))
	if !m.HasFilter() {
		t.Error("HasFilter() should be true after SetFilterStatus")
	}

	m.ClearFilter()
	if m.HasFilter() {
		t.Error("HasFilter() should be false after ClearFilter")
	}
}

func assertStatusFilter(t *testing.T, m *StatusFilterMixin, want *ResourceStatus, context string) {
	t.Helper()
	got := m.FilterStatus()
	if got == nil {
		t.Fatalf("FilterStatus() = nil %s, want %v", context, *want)
	}
	if *got != *want {
		t.Errorf("FilterStatus() = %v %s, want %v", *got, context, *want)
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d = %d, want %d", i, got[i], want[i])
		}
	}
}
