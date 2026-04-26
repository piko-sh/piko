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
	"time"
)

func TestNewOrchestratorPanel(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "orchestrator" {
		t.Errorf("expected ID 'orchestrator', got %q", panel.ID())
	}
	if panel.Title() != "Orchestrator" {
		t.Errorf("expected Title 'Orchestrator', got %q", panel.Title())
	}
}

func TestNewOrchestratorPanel_NilClock(t *testing.T) {
	panel := NewOrchestratorPanel(nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestOrchestratorPanel_SetTasks(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())

	tasks := []Resource{
		{ID: "task-1", Name: "ProcessOrder", Status: ResourceStatusHealthy},
		{ID: "task-2", Name: "SendEmail", Status: ResourceStatusPending},
	}

	panel.SetTasks(tasks)

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestOrchestratorPanel_SetWorkflows(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())

	panel.switchToViewMode(ViewModeWorkflows)

	workflows := []Resource{
		{ID: "wf-1", Name: "OrderProcessing", Status: ResourceStatusHealthy},
		{ID: "wf-2", Name: "UserOnboarding", Status: ResourceStatusPending},
	}

	panel.SetWorkflows(workflows)

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestOrchestratorPanel_SwitchToViewMode(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	panel.SetSize(80, 24)

	panel.tasks = []Resource{
		{ID: "task-1", Name: "Task1"},
		{ID: "task-2", Name: "Task2"},
	}
	panel.workflows = []Resource{
		{ID: "wf-1", Name: "Workflow1"},
	}

	if panel.viewMode != ViewModeTasks {
		t.Error("expected initial viewMode to be ViewModeTasks")
	}

	panel.SetCursor(1)

	panel.switchToViewMode(ViewModeWorkflows)

	if panel.viewMode != ViewModeWorkflows {
		t.Error("expected viewMode to be ViewModeWorkflows after switch")
	}
	if panel.Cursor() != 0 {
		t.Errorf("expected cursor to reset to 0, got %d", panel.Cursor())
	}

	panel.SetCursor(1)
	panel.switchToViewMode(ViewModeWorkflows)
	if panel.Cursor() != 1 {
		t.Error("switching to same mode should not reset cursor")
	}
}

func TestOrchestratorPanel_ApplyFilters(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())

	tasks := []Resource{
		{ID: "1", Name: "Task1", Status: ResourceStatusHealthy},
		{ID: "2", Name: "Task2", Status: ResourceStatusHealthy},
		{ID: "3", Name: "Task3", Status: ResourceStatusUnhealthy},
	}
	panel.tasks = tasks

	panel.applyFilters()
	if len(panel.Items()) != 3 {
		t.Errorf("expected 3 items without filter, got %d", len(panel.Items()))
	}

	panel.CycleFilter()
	panel.applyFilters()

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 healthy items, got %d", len(items))
	}

	panel.CycleFilter()
	panel.applyFilters()

	items = panel.Items()
	if len(items) != 0 {
		t.Errorf("expected 0 degraded items, got %d", len(items))
	}

	panel.CycleFilter()
	panel.applyFilters()

	items = panel.Items()
	if len(items) != 1 {
		t.Errorf("expected 1 unhealthy item, got %d", len(items))
	}
}

func TestOrchestratorPanel_View(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	view := panel.View(100, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "No items") {
		t.Error("expected empty state message")
	}
}

func TestOrchestratorPanel_ViewWithTasks(t *testing.T) {
	mockClock := newTestClock()
	panel := NewOrchestratorPanel(mockClock)
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	panel.SetTasks([]Resource{
		{
			ID:         "task-1",
			Name:       "ProcessOrder",
			Status:     ResourceStatusHealthy,
			StatusText: "Running",
			UpdatedAt:  mockClock.Now().Add(-5 * time.Minute),
			Metadata: map[string]string{
				"priority": "high",
				"attempt":  "1",
			},
		},
	})

	view := panel.View(100, 24)
	if !strings.Contains(view, "ProcessOrder") {
		t.Error("expected 'ProcessOrder' in view")
	}
	if !strings.Contains(view, "[Tasks]") {
		t.Error("expected '[Tasks]' indicator in view")
	}
}

func TestOrchestratorPanel_ViewWithWorkflows(t *testing.T) {
	mockClock := newTestClock()
	panel := NewOrchestratorPanel(mockClock)
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	panel.switchToViewMode(ViewModeWorkflows)
	panel.SetWorkflows([]Resource{
		{
			ID:     "wf-1",
			Name:   "OrderProcessing",
			Status: ResourceStatusHealthy,
			Metadata: map[string]string{
				"task_count":     "10",
				"complete_count": "8",
				"failed_count":   "1",
				"progress":       "80%",
			},
		},
	})

	view := panel.View(100, 24)
	if !strings.Contains(view, "OrderProcessing") {
		t.Error("expected 'OrderProcessing' in view")
	}
	if !strings.Contains(view, "[Workflows]") {
		t.Error("expected '[Workflows]' indicator in view")
	}
}

func TestOrchestratorRenderer_GetID(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	renderer := &orchestratorRenderer{panel: panel}

	resource := Resource{ID: "task-123"}
	id := renderer.GetID(resource)

	if id != "task-123" {
		t.Errorf("expected 'task-123', got %q", id)
	}
}

func TestOrchestratorRenderer_MatchesFilter(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	renderer := &orchestratorRenderer{panel: panel}

	resource := Resource{ID: "task-123", Name: "ProcessOrder"}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "process", expected: true},
		{query: "order", expected: true},
		{query: "task-123", expected: true},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(resource, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestOrchestratorRenderer_IsExpandable(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	renderer := &orchestratorRenderer{panel: panel}

	testCases := []struct {
		name     string
		resource Resource
		expected bool
	}{
		{
			name:     "with non-empty metadata",
			resource: Resource{Metadata: map[string]string{"key": "value"}},
			expected: true,
		},
		{
			name:     "with empty metadata values",
			resource: Resource{Metadata: map[string]string{"key": ""}},
			expected: false,
		},
		{
			name:     "without metadata",
			resource: Resource{Metadata: nil},
			expected: false,
		},
		{
			name:     "empty metadata map",
			resource: Resource{Metadata: map[string]string{}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.resource)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestOrchestratorRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	renderer := &orchestratorRenderer{panel: panel}

	testCases := []struct {
		name     string
		resource Resource
		expected int
	}{
		{
			name: "counts non-empty values only",
			resource: Resource{
				Metadata: map[string]string{
					"key1": "value1",
					"key2": "",
					"key3": "value3",
				},
			},
			expected: 2,
		},
		{
			name:     "empty metadata",
			resource: Resource{Metadata: map[string]string{}},
			expected: 0,
		},
		{
			name:     "nil metadata",
			resource: Resource{Metadata: nil},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.ExpandedLineCount(tc.resource)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestOrchestratorRenderer_RenderExpanded(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())
	panel.SetSize(80, 24)
	renderer := &orchestratorRenderer{panel: panel}

	resource := Resource{
		Metadata: map[string]string{
			"priority": "high",
			"attempt":  "2",
			"empty":    "",
		},
	}

	lines := renderer.RenderExpanded(resource, 80)

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestOrchestratorPanel_RenderTaskRow(t *testing.T) {
	mockClock := newTestClock()
	panel := NewOrchestratorPanel(mockClock)
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	task := Resource{
		ID:         "task-1",
		Name:       "ProcessOrder",
		Status:     ResourceStatusHealthy,
		StatusText: "Running",
		UpdatedAt:  mockClock.Now().Add(-30 * time.Second),
		Metadata: map[string]string{
			"priority": "high",
			"attempt":  "2",
		},
	}

	row := panel.renderTaskRow(task, true)
	if !strings.Contains(row, "ProcessOrder") {
		t.Error("expected 'ProcessOrder' in row")
	}
	if !strings.Contains(row, "Running") {
		t.Error("expected 'Running' in row")
	}
	if !strings.Contains(row, "P:h") {
		t.Error("expected priority 'P:h' in row")
	}
	if !strings.Contains(row, "A:2") {
		t.Error("expected attempt 'A:2' in row")
	}
}

func TestOrchestratorPanel_RenderWorkflowRow(t *testing.T) {
	mockClock := newTestClock()
	panel := NewOrchestratorPanel(mockClock)
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	workflow := Resource{
		ID:     "wf-1",
		Name:   "OrderProcessing",
		Status: ResourceStatusHealthy,
		Metadata: map[string]string{
			"task_count":     "10",
			"complete_count": "8",
			"failed_count":   "1",
			"progress":       "80%",
		},
	}

	row := panel.renderWorkflowRow(workflow, true)
	if !strings.Contains(row, "OrderProcessing") {
		t.Error("expected 'OrderProcessing' in row")
	}
	if !strings.Contains(row, "8/10") {
		t.Error("expected '8/10' progress in row")
	}
}

func TestOrchestratorPanel_CurrentItems(t *testing.T) {
	panel := NewOrchestratorPanel(newTestClock())

	tasks := []Resource{{ID: "task-1"}, {ID: "task-2"}}
	workflows := []Resource{{ID: "wf-1"}}

	panel.tasks = tasks
	panel.workflows = workflows

	panel.viewMode = ViewModeTasks
	items := panel.currentItems()
	if len(items) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(items))
	}

	panel.viewMode = ViewModeWorkflows
	items = panel.currentItems()
	if len(items) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(items))
	}
}
