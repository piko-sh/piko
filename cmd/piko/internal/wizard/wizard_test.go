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

package wizard

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInitialModel(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	if m.Step != StepProjectName {
		t.Errorf("Step = %d, want StepProjectName (%d)", m.Step, StepProjectName)
	}
	if m.Done {
		t.Error("Done should be false")
	}
	if m.Aborted {
		t.Error("Aborted should be false")
	}
	if len(m.Inputs) != 1 {
		t.Errorf("len(Inputs) = %d, want 1", len(m.Inputs))
	}
}

func TestUpdate_CtrlC_Aborts(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	message := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if !model.Aborted {
		t.Error("Aborted should be true after Ctrl+C")
	}
}

func TestUpdate_Esc_Aborts(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	message := tea.KeyPressMsg{Code: tea.KeyEscape}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if !model.Aborted {
		t.Error("Aborted should be true after Esc")
	}
}

func TestUpdate_ProjectName_Enter(t *testing.T) {
	t.Parallel()

	m := InitialModel()

	m.Inputs[0].SetValue("my-app")

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Step != StepDestination {
		t.Errorf("Step = %d, want StepDestination (%d)", model.Step, StepDestination)
	}
	if model.Config.ProjectName != "my-app" {
		t.Errorf("ProjectName = %q, want %q", model.Config.ProjectName, "my-app")
	}
	if len(model.Choices) != 2 {
		t.Errorf("len(Choices) = %d, want 2", len(model.Choices))
	}
}

func TestUpdate_ProjectName_EmptyUsesPlaceholder(t *testing.T) {
	t.Parallel()

	m := InitialModel()

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Config.ProjectName != "my-piko-app" {
		t.Errorf("ProjectName = %q, want placeholder %q", model.Config.ProjectName, "my-piko-app")
	}
}

func TestUpdate_Destination_NewFolder(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Config.ProjectName = "test-proj"
	m.Step = StepDestination
	m.Choices = []string{"new folder", "current folder"}
	m.Cursor = 0

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Step != StepModulePath {
		t.Errorf("Step = %d, want StepModulePath (%d)", model.Step, StepModulePath)
	}
	if model.Config.DestinationPath != "test-proj" {
		t.Errorf("DestinationPath = %q, want %q", model.Config.DestinationPath, "test-proj")
	}
}

func TestUpdate_Destination_CurrentFolder(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Config.ProjectName = "test-proj"
	m.Step = StepDestination
	m.Choices = []string{"new folder", "current folder"}
	m.Cursor = 1

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Config.DestinationPath != "." {
		t.Errorf("DestinationPath = %q, want %q", model.Config.DestinationPath, ".")
	}
}

func TestUpdate_DestinationNavigation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		key           string
		startPosition int
		wantPos       int
		numChoices    int
	}{
		{name: "down from 0", key: "down", startPosition: 0, wantPos: 1, numChoices: 2},
		{name: "j from 0", key: "j", startPosition: 0, wantPos: 1, numChoices: 2},
		{name: "up from 1", key: "up", startPosition: 1, wantPos: 0, numChoices: 2},
		{name: "k from 1", key: "k", startPosition: 1, wantPos: 0, numChoices: 2},
		{name: "up at 0 stays", key: "up", startPosition: 0, wantPos: 0, numChoices: 2},
		{name: "down at last stays", key: "down", startPosition: 1, wantPos: 1, numChoices: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := InitialModel()
			m.Step = StepDestination
			m.Choices = make([]string, tc.numChoices)
			m.Cursor = tc.startPosition

			message := tea.KeyPressMsg{Code: []rune(tc.key)[0], Text: tc.key}
			result, _ := m.Update(message)
			model := toModel(t, result)

			if model.Cursor != tc.wantPos {
				t.Errorf("Cursor = %d, want %d", model.Cursor, tc.wantPos)
			}
		})
	}
}

func TestUpdate_ModulePath_Enter(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepModulePath
	m.Config.ProjectName = "test-proj"
	m.Config.DestinationPath = "test-proj"

	m.Inputs = append(m.Inputs, m.Inputs[0])
	m.Inputs[1].SetValue("github.com/user/test-proj")

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Step != StepFeatures {
		t.Errorf("Step = %d, want StepFeatures (%d)", model.Step, StepFeatures)
	}
	if model.Config.ModuleName != "github.com/user/test-proj" {
		t.Errorf("ModuleName = %q, want %q", model.Config.ModuleName, "github.com/user/test-proj")
	}
	if len(model.Choices) == 0 {
		t.Error("Choices should be populated for features step")
	}
	if len(model.Selected) != len(model.Choices) {
		t.Errorf("Selected length = %d, want %d", len(model.Selected), len(model.Choices))
	}
}

func TestUpdate_ModulePath_EmptyUsesPlaceholder(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepModulePath
	m.Config.ProjectName = "test-proj"

	m.Inputs = append(m.Inputs, m.Inputs[0])
	m.Inputs[1].Placeholder = "test-proj"
	m.Inputs[1].SetValue("")

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Config.ModuleName != "test-proj" {
		t.Errorf("ModuleName = %q, want placeholder %q", model.Config.ModuleName, "test-proj")
	}
}

func TestUpdate_Features_Toggle(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepFeatures
	m.Choices = []string{
		"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
		"Experimental interpreted mode",
	}
	m.Selected = []bool{false, false}
	m.Cursor = 0

	message := tea.KeyPressMsg{Code: tea.KeySpace}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if !model.Selected[0] {
		t.Error("Selected[0] should be true after space toggle")
	}

	result, _ = model.Update(message)
	model = toModel(t, result)
	if model.Selected[0] {
		t.Error("Selected[0] should be false after second toggle")
	}
}

func TestUpdate_Features_Enter(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepFeatures
	m.Config.ProjectName = "test-proj"
	m.Config.DestinationPath = "test-proj"
	m.Config.ModuleName = "test-proj"
	m.Choices = []string{
		"Struct validation (go-playground/validator)",
		"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
		"Sonic JSON provider (faster JSON encoding via bytedance/sonic)",
		"Experimental interpreted mode",
	}
	m.Selected = []bool{true, true, false, true}
	m.Cursor = len(m.Choices)

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Step != StepScaffolding {
		t.Errorf("Step = %d, want StepScaffolding (%d)", model.Step, StepScaffolding)
	}
	if !model.Config.EnableValidator {
		t.Error("EnableValidator should be true when selected")
	}
	if !model.Config.EnableAgents {
		t.Error("EnableAgents should be true when selected")
	}
	if model.Config.EnableSonicJSON {
		t.Error("EnableSonicJSON should be false when not selected")
	}
	if !model.Config.EnableInterpreted {
		t.Error("EnableInterpreted should be true when selected")
	}
}

func TestUpdate_Features_EnterTogglesCheckbox(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepFeatures
	m.Choices = []string{
		"Struct validation (go-playground/validator)",
		"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
		"Sonic JSON provider (faster JSON encoding via bytedance/sonic)",
		"Experimental interpreted mode",
	}
	m.Selected = []bool{false, false, false, false}
	m.Cursor = 0

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Step != StepFeatures {
		t.Errorf("Step = %d, want StepFeatures (%d) - enter on checkbox should toggle, not advance", model.Step, StepFeatures)
	}
	if !model.Selected[0] {
		t.Error("Selected[0] should be true after enter toggle")
	}
}

func TestUpdate_Features_AgentsOnly(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepFeatures
	m.Config.ProjectName = "test-proj"
	m.Config.DestinationPath = "test-proj"
	m.Config.ModuleName = "test-proj"
	m.Choices = []string{
		"Struct validation (go-playground/validator)",
		"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
		"Sonic JSON provider (faster JSON encoding via bytedance/sonic)",
		"Experimental interpreted mode",
	}
	m.Selected = []bool{false, true, false, false}
	m.Cursor = len(m.Choices)

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	if model.Config.EnableValidator {
		t.Error("EnableValidator should be false when not selected")
	}
	if !model.Config.EnableAgents {
		t.Error("EnableAgents should be true when selected")
	}
	if model.Config.EnableSonicJSON {
		t.Error("EnableSonicJSON should be false when not selected")
	}
	if model.Config.EnableInterpreted {
		t.Error("EnableInterpreted should be false when not selected")
	}
}

func TestUpdate_ErrorMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	testErr := errors.New("scaffold failed")
	result, _ := m.Update(errMessage{err: testErr})
	model := toModel(t, result)

	if model.Err == nil {
		t.Fatal("Err should not be nil")
	}
	if model.Err.Error() != "scaffold failed" {
		t.Errorf("Err = %v, want %q", model.Err, "scaffold failed")
	}
}

func TestUpdate_ScaffoldDoneMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepScaffolding
	m.Config.DestinationPath = "/tmp/test"

	result, _ := m.Update(scaffoldDoneMessage{})
	model := toModel(t, result)

	if model.Step != StepTidying {
		t.Errorf("Step = %d, want StepTidying (%d)", model.Step, StepTidying)
	}
}

func TestUpdate_TidyDoneMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepTidying

	result, _ := m.Update(tidyDoneMessage{})
	model := toModel(t, result)

	if !model.Done {
		t.Error("Done should be true")
	}
	if model.Step != StepFinished {
		t.Errorf("Step = %d, want StepFinished (%d)", model.Step, StepFinished)
	}
}

func TestView_EachStep(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		step     int
		setup    func(*Model)
		mustHave string
	}{
		{
			name:     "project name prompt",
			step:     StepProjectName,
			mustHave: "name of your new Piko project",
		},
		{
			name: "destination prompt",
			step: StepDestination,
			setup: func(m *Model) {
				m.Choices = []string{"new folder", "current folder"}
			},
			mustHave: "Where should we create",
		},
		{
			name: "module path prompt",
			step: StepModulePath,
			setup: func(m *Model) {
				m.Inputs = append(m.Inputs, m.Inputs[0])
			},
			mustHave: "Go module path",
		},
		{
			name: "features checkboxes",
			step: StepFeatures,
			setup: func(m *Model) {
				m.Choices = []string{
					"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
					"Experimental interpreted mode",
				}
				m.Selected = []bool{false, false}
			},
			mustHave: "optional features",
		},
		{
			name:     "scaffolding spinner",
			step:     StepScaffolding,
			mustHave: "Scaffolding",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := InitialModel()
			m.Step = tc.step
			if tc.setup != nil {
				tc.setup(&m)
			}
			view := m.View()
			if !strings.Contains(view.Content, tc.mustHave) {
				t.Errorf("View() should contain %q, got:\n%s", tc.mustHave, view.Content)
			}
		})
	}
}

func TestView_Error(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Err = errors.New("something broke")
	view := m.View()

	if !strings.Contains(view.Content, "something broke") {
		t.Errorf("View() should show error, got:\n%s", view.Content)
	}
}

func toModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	v, ok := m.(*Model)
	if !ok {
		t.Fatalf("unexpected tea.Model type: %T", m)
		return Model{}
	}
	return *v
}
