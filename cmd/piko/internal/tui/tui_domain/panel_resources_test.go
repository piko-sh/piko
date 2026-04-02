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
	"errors"
	"strings"
	"testing"
)

func TestNewResourcesPanel(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "resources" {
		t.Errorf("expected ID 'resources', got %q", panel.ID())
	}
	if panel.Title() != "Resources" {
		t.Errorf("expected Title 'Resources', got %q", panel.Title())
	}
}

func TestNewResourcesPanel_NilClock(t *testing.T) {
	panel := NewResourcesPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestResourcesPanel_CategoryIcon(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	testCases := []struct {
		category string
		expected string
	}{
		{category: "file", expected: "📄"},
		{category: "tcp", expected: "🌐"},
		{category: "udp", expected: "📡"},
		{category: "unix", expected: "🔌"},
		{category: "pipe", expected: "📥"},
		{category: "socket", expected: "🔗"},
		{category: "other", expected: "❓"},
		{category: "unknown", expected: "❓"},
	}

	for _, tc := range testCases {
		t.Run(tc.category, func(t *testing.T) {
			result := panel.categoryIcon(tc.category)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestResourcesPanel_CategoryDisplayName(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	testCases := []struct {
		category string
		expected string
	}{
		{category: "file", expected: "Files"},
		{category: "tcp", expected: "TCP Connections"},
		{category: "udp", expected: "UDP Sockets"},
		{category: "unix", expected: "Unix Sockets"},
		{category: "pipe", expected: "Pipes"},
		{category: "socket", expected: "Other Sockets"},
		{category: "other", expected: "Other"},
		{category: "unknown", expected: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.category, func(t *testing.T) {
			result := panel.categoryDisplayName(tc.category)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestResourcesPanel_FormatAge(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	testCases := []struct {
		name    string
		ageMs   int64
		warning bool
		error   bool
	}{
		{
			name:    "recent",
			ageMs:   5000,
			warning: false,
			error:   false,
		},
		{
			name:    "warning threshold",
			ageMs:   35 * 60 * 1000,
			warning: true,
			error:   false,
		},
		{
			name:    "error threshold",
			ageMs:   2 * 60 * 60 * 1000,
			warning: false,
			error:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := panel.formatAge(tc.ageMs)

			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestResourcesPanel_FDMatchesFilter(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	fd := FDInfo{
		FD:     42,
		Target: "/var/log/app.log",
	}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "log", expected: true},
		{query: "var", expected: true},
		{query: "app", expected: true},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := panel.fdMatchesFilter(fd, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestResourcesPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	data := &FDsData{
		Total: 50,
		Categories: []FDCategory{
			{Category: "file", Count: 30, FDs: []FDInfo{{FD: 1, Target: "/file1"}}},
			{Category: "tcp", Count: 20, FDs: []FDInfo{{FD: 2, Target: "192.168.1.1:8080"}}},
		},
	}
	panel.handleRefreshMessage(ResourcesRefreshMessage{Data: data})

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	panel.handleRefreshMessage(ResourcesRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestResourcesPanel_FDCountHistory(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{Total: 50, Categories: []FDCategory{}},
	})

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{Total: 60, Categories: []FDCategory{}},
	})

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{Total: 55, Categories: []FDCategory{}},
	})

	panel.stateMutex.RLock()
	history := panel.fdCountHistory.Values()
	panel.stateMutex.RUnlock()

	if len(history) != 3 {
		t.Errorf("expected 3 history values, got %d", len(history))
	}
	if history[0] != 50 || history[1] != 60 || history[2] != 55 {
		t.Errorf("unexpected history values: %v", history)
	}
}

func TestResourcesPanel_View(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	view := panel.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "No resources") {
		t.Error("expected empty state message")
	}
}

func TestResourcesPanel_ViewWithData(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{
			Total: 10,
			Categories: []FDCategory{
				{Category: "file", Count: 5, FDs: []FDInfo{}},
				{Category: "tcp", Count: 5, FDs: []FDInfo{}},
			},
		},
	})

	view := panel.View(80, 24)
	if !strings.Contains(view, "Files") {
		t.Error("expected 'Files' in view")
	}
}

func TestResourcesPanel_ToggleCategoryExpansion(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{
			Total: 10,
			Categories: []FDCategory{
				{Category: "file", Count: 3, FDs: []FDInfo{{FD: 1}, {FD: 2}, {FD: 3}}},
				{Category: "tcp", Count: 2, FDs: []FDInfo{{FD: 4}, {FD: 5}}},
			},
		},
	})

	if panel.IsExpanded("file") || panel.IsExpanded("tcp") {
		t.Error("expected no category expanded initially")
	}

	panel.toggleCategoryExpansion()
	if !panel.IsExpanded("file") {
		t.Error("expected file category to be expanded")
	}

	panel.toggleCategoryExpansion()
	if panel.IsExpanded("file") {
		t.Error("expected file category to be collapsed after second toggle")
	}
}

func TestResourcesRenderer_GetID(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	renderer := &resourcesRenderer{panel: panel}

	cat := FDCategory{Category: "file"}
	id := renderer.GetID(cat)

	if id != "file" {
		t.Errorf("expected 'file', got %q", id)
	}
}

func TestResourcesRenderer_MatchesFilter(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	renderer := &resourcesRenderer{panel: panel}

	cat := FDCategory{
		Category: "tcp",
		Count:    2,
		FDs: []FDInfo{
			{FD: 1, Target: "192.168.1.1:8080"},
			{FD: 2, Target: "10.0.0.1:3000"},
		},
	}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "tcp", expected: true},
		{query: "connections", expected: true},
		{query: "192.168", expected: true},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(cat, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestResourcesRenderer_IsExpandable(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	renderer := &resourcesRenderer{panel: panel}

	testCases := []struct {
		name     string
		cat      FDCategory
		expected bool
	}{
		{
			name:     "with FDs",
			cat:      FDCategory{Category: "file", FDs: []FDInfo{{FD: 1}}},
			expected: true,
		},
		{
			name:     "empty FDs",
			cat:      FDCategory{Category: "file", FDs: []FDInfo{}},
			expected: false,
		},
		{
			name:     "nil FDs",
			cat:      FDCategory{Category: "file"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.cat)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestResourcesRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	renderer := &resourcesRenderer{panel: panel}

	cat := FDCategory{
		Category: "file",
		FDs: []FDInfo{
			{FD: 1, Target: "/file1"},
			{FD: 2, Target: "/file2"},
			{FD: 3, Target: "/file3"},
		},
	}

	count := renderer.ExpandedLineCount(cat)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestResourcesRenderer_RenderExpanded(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	renderer := &resourcesRenderer{panel: panel}

	cat := FDCategory{
		Category: "file",
		FDs: []FDInfo{
			{FD: 1, Target: "/var/log/app.log", AgeMs: 5000},
			{FD: 2, Target: "/etc/config", AgeMs: 10000},
		},
	}

	lines := renderer.RenderExpanded(cat, 80)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestResourcesPanel_RenderCategoryRow(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	cat := FDCategory{
		Category: "file",
		Count:    10,
	}

	row := panel.renderCategoryRow(cat, true, false)
	if !strings.Contains(row, "Files") {
		t.Error("expected 'Files' in row")
	}
	if !strings.Contains(row, "10") {
		t.Error("expected count in row")
	}
}

func TestResourcesPanel_RenderFDRow(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	fd := FDInfo{
		FD:     42,
		Target: "/var/log/application.log",
		AgeMs:  60000,
	}

	row := panel.renderFDRow(fd, false)
	if !strings.Contains(row, "42") {
		t.Error("expected FD number in row")
	}
	if !strings.Contains(row, "application") {
		t.Error("expected target in row")
	}
}

func TestResourcesPanel_SummaryLine(t *testing.T) {
	panel := NewResourcesPanel(nil, newTestClock())
	panel.SetSize(80, 24)

	summaryEmpty := panel.renderSummaryLine()
	if !strings.Contains(summaryEmpty, "Loading") {
		t.Error("expected 'Loading' when no data")
	}

	panel.handleRefreshMessage(ResourcesRefreshMessage{
		Data: &FDsData{Total: 100, Categories: []FDCategory{}},
	})

	summary := panel.renderSummaryLine()
	if !strings.Contains(summary, "100 file descriptors") {
		t.Error("expected FD count in summary")
	}
}
