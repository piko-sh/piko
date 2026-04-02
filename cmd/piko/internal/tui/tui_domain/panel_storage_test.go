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

func TestNewStoragePanel(t *testing.T) {
	panel := NewStoragePanel()

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "storage" {
		t.Errorf("expected ID 'storage', got %q", panel.ID())
	}
	if panel.Title() != "Storage" {
		t.Errorf("expected Title 'Storage', got %q", panel.Title())
	}
}

func TestStoragePanel_SetArtefacts(t *testing.T) {
	panel := NewStoragePanel()

	artefacts := []Resource{
		{ID: "artefact-1", Name: "image.png", Status: ResourceStatusHealthy},
		{ID: "artefact-2", Name: "style.css", Status: ResourceStatusHealthy},
	}

	panel.SetArtefacts(artefacts)

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestStoragePanel_StatusFilter(t *testing.T) {
	panel := NewStoragePanel()

	artefacts := []Resource{
		{ID: "1", Name: "healthy1", Status: ResourceStatusHealthy},
		{ID: "2", Name: "healthy2", Status: ResourceStatusHealthy},
		{ID: "3", Name: "unhealthy", Status: ResourceStatusUnhealthy},
	}

	panel.SetArtefacts(artefacts)

	if len(panel.Items()) != 3 {
		t.Errorf("expected 3 items without filter, got %d", len(panel.Items()))
	}

	panel.CycleFilter()
	panel.applyStatusFilter()

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 healthy items, got %d", len(items))
	}

	panel.CycleFilter()
	panel.applyStatusFilter()

	items = panel.Items()
	if len(items) != 0 {
		t.Errorf("expected 0 degraded items, got %d", len(items))
	}

	panel.CycleFilter()
	panel.applyStatusFilter()

	items = panel.Items()
	if len(items) != 1 {
		t.Errorf("expected 1 unhealthy item, got %d", len(items))
	}
}

func TestStoragePanel_View(t *testing.T) {
	panel := NewStoragePanel()
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	view := panel.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "No artefacts") {
		t.Error("expected empty state message")
	}
}

func TestStoragePanel_ViewWithData(t *testing.T) {
	panel := NewStoragePanel()
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	panel.SetArtefacts([]Resource{
		{
			ID:     "1",
			Name:   "image.png",
			Status: ResourceStatusHealthy,
			Metadata: map[string]string{
				"variant_count": "3",
				"total_size":    "1.5MB",
			},
		},
	})

	view := panel.View(80, 24)
	if !strings.Contains(view, "image.png") {
		t.Error("expected 'image.png' in view")
	}
}

func TestStorageRenderer_GetID(t *testing.T) {
	panel := NewStoragePanel()
	renderer := &storageRenderer{panel: panel}

	artefact := Resource{ID: "artefact-123"}
	id := renderer.GetID(artefact)

	if id != "artefact-123" {
		t.Errorf("expected 'artefact-123', got %q", id)
	}
}

func TestStorageRenderer_MatchesFilter(t *testing.T) {
	panel := NewStoragePanel()
	renderer := &storageRenderer{panel: panel}

	artefact := Resource{ID: "art-123", Name: "background-image.png"}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "background", expected: true},
		{query: "image", expected: true},
		{query: "123", expected: true},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(artefact, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestStorageRenderer_IsExpandable(t *testing.T) {
	panel := NewStoragePanel()
	renderer := &storageRenderer{panel: panel}

	testCases := []struct {
		name     string
		artefact Resource
		expected bool
	}{
		{
			name:     "with metadata",
			artefact: Resource{Metadata: map[string]string{"key": "value"}},
			expected: true,
		},
		{
			name:     "without metadata",
			artefact: Resource{Metadata: nil},
			expected: false,
		},
		{
			name:     "empty metadata",
			artefact: Resource{Metadata: map[string]string{}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.artefact)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestStorageRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewStoragePanel()
	renderer := &storageRenderer{panel: panel}

	artefact := Resource{
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	count := renderer.ExpandedLineCount(artefact)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestStorageRenderer_RenderExpanded(t *testing.T) {
	panel := NewStoragePanel()
	panel.SetSize(80, 24)
	renderer := &storageRenderer{panel: panel}

	artefact := Resource{
		Metadata: map[string]string{
			"format": "png",
			"size":   "1.5MB",
		},
	}

	lines := renderer.RenderExpanded(artefact, 80)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestStoragePanel_RenderArtefactRow(t *testing.T) {
	panel := NewStoragePanel()
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	artefact := Resource{
		ID:     "1",
		Name:   "image.png",
		Status: ResourceStatusHealthy,
		Metadata: map[string]string{
			"variant_count": "3",
			"total_size":    "1.5MB",
		},
	}

	row := panel.renderArtefactRow(artefact, true, false)
	if !strings.Contains(row, "image.png") {
		t.Error("expected 'image.png' in row")
	}
	if !strings.Contains(row, "1.5MB") {
		t.Error("expected '1.5MB' in row")
	}
}
