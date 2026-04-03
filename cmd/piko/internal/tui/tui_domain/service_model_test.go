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
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestNewModel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !model.lastRefresh.IsZero() {
		t.Error("expected zero lastRefresh")
	}
	if len(model.panels) != 0 {
		t.Error("expected no panels initially")
	}
	if model.activePanelIndex != 0 {
		t.Error("expected activePanelIndex 0")
	}
	if model.showHelp {
		t.Error("expected showHelp false")
	}
	if model.quitting {
		t.Error("expected quitting false")
	}
}

func TestModel_AddPanel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")

	model.AddPanel(panel1)
	if len(model.panels) != 1 {
		t.Errorf("expected 1 panel, got %d", len(model.panels))
	}
	if !panel1.Focused() {
		t.Error("first panel should be focused")
	}

	model.AddPanel(panel2)
	if len(model.panels) != 2 {
		t.Errorf("expected 2 panels, got %d", len(model.panels))
	}
	if panel2.Focused() {
		t.Error("second panel should not be focused initially")
	}
}

func TestModel_ActivePanel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	if model.ActivePanel() != nil {
		t.Error("expected nil when no panels")
	}

	panel := newMockPanel("test")
	model.AddPanel(panel)

	active := model.ActivePanel()
	if active == nil {
		t.Fatal("expected active panel")
	}
	if active.ID() != "test" {
		t.Errorf("expected ID 'test', got %q", active.ID())
	}
}

func TestModel_UpdateResourceData(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	summary := map[string]map[ResourceStatus]int{
		"pod": {
			ResourceStatusHealthy:   5,
			ResourceStatusUnhealthy: 2,
		},
	}
	resources := map[string][]Resource{
		"pod": {
			{ID: "pod-1", Name: "nginx-1"},
			{ID: "pod-2", Name: "nginx-2"},
		},
	}

	model.UpdateResourceData(summary, resources)

	if model.resourceSummary["pod"][ResourceStatusHealthy] != 5 {
		t.Error("expected healthy count of 5")
	}

	if len(model.resourcesByKind["pod"]) != 2 {
		t.Errorf("expected 2 pods, got %d", len(model.resourcesByKind["pod"]))
	}
}

func TestModel_UpdateResourceData_RemovesStaleKinds(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.UpdateResourceData(
		map[string]map[ResourceStatus]int{
			"pod": {ResourceStatusHealthy: 3},
		},
		map[string][]Resource{
			"pod": {{ID: "pod-1", Name: "nginx"}},
		},
	)

	model.UpdateResourceData(
		map[string]map[ResourceStatus]int{
			"service": {ResourceStatusHealthy: 1},
		},
		map[string][]Resource{
			"service": {{ID: "svc-1", Name: "api"}},
		},
	)

	if _, exists := model.resourceSummary["pod"]; exists {
		t.Error("expected stale 'pod' kind to be removed")
	}
	if _, exists := model.resourcesByKind["pod"]; exists {
		t.Error("expected stale 'pod' resources to be removed")
	}
	if model.resourceSummary["service"][ResourceStatusHealthy] != 1 {
		t.Error("expected 'service' kind to be present")
	}
}

func TestModel_Init(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel := newMockPanel("test")
	model.AddPanel(panel)

	command := model.Init()
	if command == nil {
		t.Error("expected non-nil command from Init")
	}
}

func TestModel_View_Quitting(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	model.quitting = true

	view := model.View()
	if view.Content != "Goodbye!\n" {
		t.Errorf("expected goodbye message, got %q", view.Content)
	}
}

func TestModel_View_Initialising(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	view := model.View()
	if view.Content != "Initialising..." {
		t.Errorf("expected initialising message, got %q", view.Content)
	}
}

func TestModel_View_NoPanels(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	model.width = 80
	model.height = 24

	view := model.View()
	if view.Content != "No panels available" {
		t.Errorf("expected 'No panels available', got %q", view.Content)
	}
}

func TestModel_HandleKeyMessage(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")
	model.AddPanel(panel1)
	model.AddPanel(panel2)

	testCases := []struct {
		name      string
		key       string
		expectCmd bool
	}{
		{name: "q quits", key: "q", expectCmd: true},
		{name: "? toggles help", key: "?", expectCmd: true},
		{name: "tab next panel", key: "tab", expectCmd: true},
		{name: "shift+tab prev panel", key: "shift+tab", expectCmd: true},
		{name: "r force refresh", key: "r", expectCmd: true},
		{name: "1 jumps to panel", key: "1", expectCmd: true},
		{name: "unknown key no command", key: "x", expectCmd: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := createTestKeyMessage(tc.key)
			command := model.handleKeyMessage(message)

			if tc.expectCmd && command == nil {
				t.Error("expected command")
			}
			if !tc.expectCmd && command != nil {
				t.Error("expected no command")
			}
		})
	}
}

func TestModel_FocusPanelByID(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")
	model.AddPanel(panel1)
	model.AddPanel(panel2)

	model.focusPanelByID("panel2")

	if model.activePanelIndex != 1 {
		t.Errorf("expected activePanelIndex 1, got %d", model.activePanelIndex)
	}
	if !panel2.Focused() {
		t.Error("panel2 should be focused")
	}
	if panel1.Focused() {
		t.Error("panel1 should not be focused")
	}
}

func TestModel_FocusNextPrevPanel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")
	panel3 := newMockPanel("panel3")
	model.AddPanel(panel1)
	model.AddPanel(panel2)
	model.AddPanel(panel3)

	model.focusNextPanel()
	if model.activePanelIndex != 1 {
		t.Errorf("expected 1 after next, got %d", model.activePanelIndex)
	}

	model.focusNextPanel()
	if model.activePanelIndex != 2 {
		t.Errorf("expected 2 after second next, got %d", model.activePanelIndex)
	}

	model.focusNextPanel()
	if model.activePanelIndex != 0 {
		t.Errorf("expected 0 after wrap, got %d", model.activePanelIndex)
	}

	model.focusPreviousPanel()
	if model.activePanelIndex != 2 {
		t.Errorf("expected 2 after prev wrap, got %d", model.activePanelIndex)
	}
}

func TestModel_FocusNextPanel_NoPanels(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.focusNextPanel()
	model.focusPreviousPanel()
}

func TestModel_DispatchMessage_WindowSize(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	message := tea.WindowSizeMsg{Width: 100, Height: 50}
	model.dispatchMessage(message)

	if model.width != 100 {
		t.Errorf("expected width 100, got %d", model.width)
	}
	if model.height != 50 {
		t.Errorf("expected height 50, got %d", model.height)
	}
}

func TestModel_DispatchMessage_ToggleHelp(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.dispatchMessage(toggleHelpMessage{})
	if !model.showHelp {
		t.Error("expected showHelp true")
	}

	model.dispatchMessage(toggleHelpMessage{})
	if model.showHelp {
		t.Error("expected showHelp false")
	}
}

func TestModel_DispatchMessage_Quit(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.dispatchMessage(quitMessage{})
	if !model.quitting {
		t.Error("expected quitting true")
	}
}

func TestModel_DispatchMessage_FocusPanel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")
	model.AddPanel(panel1)
	model.AddPanel(panel2)

	model.dispatchMessage(focusPanelMessage{panelID: "panel2"})

	if model.activePanelIndex != 1 {
		t.Errorf("expected panel2 focused (index 1), got %d", model.activePanelIndex)
	}
}

func TestModel_DispatchMessage_NextPrevPanel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel1 := newMockPanel("panel1")
	panel2 := newMockPanel("panel2")
	model.AddPanel(panel1)
	model.AddPanel(panel2)

	model.dispatchMessage(nextPanelMessage{})
	if model.activePanelIndex != 1 {
		t.Errorf("expected 1 after next, got %d", model.activePanelIndex)
	}

	model.dispatchMessage(previousPanelMessage{})
	if model.activePanelIndex != 0 {
		t.Errorf("expected 0 after prev, got %d", model.activePanelIndex)
	}
}

func TestModel_DispatchMessage_ErrorMessage(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.dispatchMessage(errorMessage{err: ErrProviderNotReady})

	if !errors.Is(model.lastError, ErrProviderNotReady) {
		t.Error("expected lastError to be set")
	}
}

func TestModel_HandleTickMessage(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel := newMockPanel("test")
	model.AddPanel(panel)

	testTime := time.Now()
	cmds := model.handleTickMessage(tickMessage{time: testTime})

	if !model.lastRefresh.Equal(testTime) {
		t.Error("expected lastRefresh to be updated")
	}
	if len(cmds) != 1 {
		t.Errorf("expected 1 command (tick only), got %d", len(cmds))
	}
}

func TestModel_HandleTickMessage_CollectsPanelCommands(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	commandCalled := false
	panel := newMockPanel("test")
	panel.updateFunc = func(message tea.Msg) (Panel, tea.Cmd) {
		if _, ok := message.(TickMessage); ok {
			return panel, func() tea.Msg {
				commandCalled = true
				return nil
			}
		}
		return panel, nil
	}
	model.AddPanel(panel)

	cmds := model.handleTickMessage(tickMessage{time: time.Now()})

	if len(cmds) != 2 {
		t.Errorf("expected 2 commands (tick + panel), got %d", len(cmds))
	}

	for _, command := range cmds[1:] {
		if command != nil {
			command()
		}
	}
	if !commandCalled {
		t.Error("expected panel command to be collected and executable")
	}
}

func TestModel_HandleDataUpdatedMessage(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	testTime := time.Now()
	cmds := model.handleDataUpdatedMessage(dataUpdatedMessage{time: testTime})

	if !model.lastRefresh.Equal(testTime) {
		t.Error("expected lastRefresh to be updated")
	}
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}
}

func TestModel_HandleProviderStatusMessage(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.providerInfo["test-provider"] = &ProviderInfo{
		Name:   "test-provider",
		Status: ProviderStatusConnecting,
	}

	model.handleProviderStatusMessage(providerStatusMessage{
		name:   "test-provider",
		status: ProviderStatusConnected,
		err:    nil,
	})

	if model.providerInfo["test-provider"].Status != ProviderStatusConnected {
		t.Error("expected status to be updated")
	}

	model.handleProviderStatusMessage(providerStatusMessage{
		name:   "test-provider",
		status: ProviderStatusError,
		err:    ErrConnectionFailed,
	})

	if model.providerInfo["test-provider"].ErrorCount != 1 {
		t.Error("expected error count to increment")
	}
}

func TestModel_HandleProviderStatusMessage_UnknownProvider(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	model.handleProviderStatusMessage(providerStatusMessage{
		name:   "unknown",
		status: ProviderStatusConnected,
	})
}

func TestModel_Update(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	panel := newMockPanel("test")
	model.AddPanel(panel)
	model.width = 80
	model.height = 24

	newModel, command := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if newModel == nil {
		t.Error("expected non-nil model")
	}

	_ = command

	updatedModel, ok := newModel.(*Model)
	if !ok {
		t.Fatal("expected newModel to be *Model")
	}
	if updatedModel.width != 100 {
		t.Errorf("expected width 100, got %d", updatedModel.width)
	}
}

func TestModel_Update_Quitting(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	model.quitting = true

	_, command := model.Update(nil)

	if command == nil {
		t.Error("expected quit command")
	}
}

func TestModel_RenderHelp(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	model.showHelp = true
	model.width = 80
	model.height = 24

	panel := newMockPanel("test")
	panel.keyMap = []KeyBinding{{Key: "j", Description: "Down"}}
	model.AddPanel(panel)

	result := model.renderHelp()

	if result == "" {
		t.Error("expected non-empty help")
	}
}
