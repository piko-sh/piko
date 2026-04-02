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

func TestNewRegistryPanel(t *testing.T) {
	panel := NewRegistryPanel()

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "registry" {
		t.Errorf("expected ID 'registry', got %q", panel.ID())
	}
	if panel.Title() != "Registry" {
		t.Errorf("expected Title 'Registry', got %q", panel.Title())
	}
}

func TestRegistryPanel_SetSummary(t *testing.T) {
	panel := NewRegistryPanel()

	summary := map[string]map[ResourceStatus]int{
		"pod": {
			ResourceStatusHealthy:   5,
			ResourceStatusUnhealthy: 2,
		},
		"service": {
			ResourceStatusHealthy: 10,
		},
	}

	panel.SetSummary(summary)

	if len(panel.kinds) != 2 {
		t.Errorf("expected 2 kinds, got %d", len(panel.kinds))
	}
	if panel.kinds[0] != "pod" || panel.kinds[1] != "service" {
		t.Errorf("expected kinds to be sorted, got %v", panel.kinds)
	}
}

func TestRegistryPanel_SetResources(t *testing.T) {
	panel := NewRegistryPanel()
	panel.SetSummary(map[string]map[ResourceStatus]int{
		"pod": {ResourceStatusHealthy: 2},
	})
	panel.selectedKind = "pod"

	resources := []Resource{
		{ID: "pod-1", Name: "nginx-1"},
		{ID: "pod-2", Name: "nginx-2"},
	}
	panel.SetResources(resources)

	if len(panel.resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(panel.resources))
	}
}

func TestRegistryPanel_SelectedKind(t *testing.T) {
	panel := NewRegistryPanel()

	if panel.SelectedKind() != "" {
		t.Error("expected empty selectedKind initially")
	}

	panel.selectedKind = "pod"
	if panel.SelectedKind() != "pod" {
		t.Errorf("expected 'pod', got %q", panel.SelectedKind())
	}
}

func TestDetermineKindOverallStatus(t *testing.T) {
	testCases := []struct {
		counts   map[ResourceStatus]int
		name     string
		expected ResourceStatus
	}{
		{
			name:     "unhealthy takes precedence",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 5, ResourceStatusUnhealthy: 1},
			expected: ResourceStatusUnhealthy,
		},
		{
			name:     "degraded takes precedence over pending",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 5, ResourceStatusDegraded: 1, ResourceStatusPending: 1},
			expected: ResourceStatusDegraded,
		},
		{
			name:     "pending takes precedence over healthy",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 5, ResourceStatusPending: 1},
			expected: ResourceStatusPending,
		},
		{
			name:     "healthy only",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 10},
			expected: ResourceStatusHealthy,
		},
		{
			name:     "empty counts",
			counts:   map[ResourceStatus]int{},
			expected: ResourceStatusUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := determineKindOverallStatus(tc.counts)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestBuildKindCountString(t *testing.T) {
	testCases := []struct {
		name     string
		counts   map[ResourceStatus]int
		contains []string
	}{
		{
			name:     "healthy only",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 5},
			contains: []string{"(5"},
		},
		{
			name:     "with unhealthy",
			counts:   map[ResourceStatus]int{ResourceStatusHealthy: 3, ResourceStatusUnhealthy: 2},
			contains: []string{"(5", "2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildKindCountString(tc.counts)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestRegistryPanel_View(t *testing.T) {
	panel := NewRegistryPanel()
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

func TestRegistryPanel_ViewWithData(t *testing.T) {
	panel := NewRegistryPanel()
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	panel.SetSummary(map[string]map[ResourceStatus]int{
		"pod":     {ResourceStatusHealthy: 5},
		"service": {ResourceStatusHealthy: 3},
	})

	view := panel.View(80, 24)
	if !strings.Contains(view, "Pod") {
		t.Error("expected 'Pod' in view")
	}
}

func TestRegistryRenderer_GetID(t *testing.T) {
	panel := NewRegistryPanel()
	renderer := &registryRenderer{panel: panel}

	testCases := []struct {
		name     string
		expected string
		item     registryDisplayItem
	}{
		{
			name:     "kind item",
			item:     registryDisplayItem{kind: "pod", itemType: registryItemKind},
			expected: "kind:pod",
		},
		{
			name:     "resource item",
			item:     registryDisplayItem{kind: "pod", resource: &Resource{ID: "r1"}, itemType: registryItemResource},
			expected: "resource:r1",
		},
		{
			name:     "metadata item",
			item:     registryDisplayItem{resourceID: "r1", metadataKey: "key1", itemType: registryItemMetadata},
			expected: "meta:r1:key1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.GetID(tc.item)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestRegistryRenderer_MatchesFilter(t *testing.T) {
	panel := NewRegistryPanel()
	renderer := &registryRenderer{panel: panel}

	testCases := []struct {
		name     string
		query    string
		item     registryDisplayItem
		expected bool
	}{
		{
			name:     "kind matches",
			item:     registryDisplayItem{kind: "pod", itemType: registryItemKind},
			query:    "pod",
			expected: true,
		},
		{
			name:     "resource name matches",
			item:     registryDisplayItem{resource: &Resource{ID: "r1", Name: "nginx"}, itemType: registryItemResource},
			query:    "nginx",
			expected: true,
		},
		{
			name:     "metadata key matches",
			item:     registryDisplayItem{metadataKey: "version", metadataVal: "1.0", itemType: registryItemMetadata},
			query:    "version",
			expected: true,
		},
		{
			name:     "no match",
			item:     registryDisplayItem{kind: "pod", itemType: registryItemKind},
			query:    "xyz",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.MatchesFilter(tc.item, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestRegistryRenderer_IsExpandable(t *testing.T) {
	panel := NewRegistryPanel()
	renderer := &registryRenderer{panel: panel}

	testCases := []struct {
		name     string
		item     registryDisplayItem
		expected bool
	}{
		{
			name:     "kind is expandable",
			item:     registryDisplayItem{itemType: registryItemKind},
			expected: true,
		},
		{
			name:     "resource with metadata is expandable",
			item:     registryDisplayItem{resource: &Resource{Metadata: map[string]string{"k": "v"}}, itemType: registryItemResource},
			expected: true,
		},
		{
			name:     "resource without metadata not expandable",
			item:     registryDisplayItem{resource: &Resource{Metadata: nil}, itemType: registryItemResource},
			expected: false,
		},
		{
			name:     "metadata not expandable",
			item:     registryDisplayItem{itemType: registryItemMetadata},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.item)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestRegistryRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewRegistryPanel()
	renderer := &registryRenderer{panel: panel}

	item := registryDisplayItem{kind: "pod", itemType: registryItemKind}
	if renderer.ExpandedLineCount(item) != 0 {
		t.Error("expected 0 for all items")
	}
}
