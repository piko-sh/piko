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
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialModel(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	assert.Equalf(t, StepProjectName, m.Step, "Step = %d, want StepProjectName (%d)", m.Step, StepProjectName)
	assert.False(t, m.Done, "Done should be false")
	assert.False(t, m.Aborted, "Aborted should be false")
	assert.Lenf(t, m.Inputs, 1, "len(Inputs) = %d, want 1", len(m.Inputs))
}

func TestUpdate_CtrlC_Aborts(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	message := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	result, _ := m.Update(message)
	model := toModel(t, result)

	assert.True(t, model.Aborted, "Aborted should be true after Ctrl+C")
}

func TestUpdate_Esc_Aborts(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	message := tea.KeyPressMsg{Code: tea.KeyEscape}
	result, _ := m.Update(message)
	model := toModel(t, result)

	assert.True(t, model.Aborted, "Aborted should be true after Esc")
}

func TestUpdate_ProjectName_Enter(t *testing.T) {
	t.Parallel()

	m := InitialModel()

	m.Inputs[0].SetValue("my-app")

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	assert.Equalf(t, StepDestination, model.Step, "Step = %d, want StepDestination (%d)", model.Step, StepDestination)
	assert.Equalf(t, "my-app", model.Config.ProjectName, "ProjectName = %q, want %q", model.Config.ProjectName, "my-app")
	assert.Lenf(t, model.Choices, 2, "len(Choices) = %d, want 2", len(model.Choices))
}

func TestUpdate_ProjectName_EmptyUsesPlaceholder(t *testing.T) {
	t.Parallel()

	m := InitialModel()

	message := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, _ := m.Update(message)
	model := toModel(t, result)

	assert.Equalf(t, "my-piko-app", model.Config.ProjectName, "ProjectName = %q, want placeholder %q", model.Config.ProjectName, "my-piko-app")
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

	assert.Equalf(t, StepModulePath, model.Step, "Step = %d, want StepModulePath (%d)", model.Step, StepModulePath)
	assert.Equalf(t, "test-proj", model.Config.DestinationPath, "DestinationPath = %q, want %q", model.Config.DestinationPath, "test-proj")
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

	assert.Equalf(t, ".", model.Config.DestinationPath, "DestinationPath = %q, want %q", model.Config.DestinationPath, ".")
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

			assert.Equalf(t, tc.wantPos, model.Cursor, "Cursor = %d, want %d", model.Cursor, tc.wantPos)
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

	assert.Equalf(t, StepFeatures, model.Step, "Step = %d, want StepFeatures (%d)", model.Step, StepFeatures)
	assert.Equalf(t, "github.com/user/test-proj", model.Config.ModuleName, "ModuleName = %q, want %q", model.Config.ModuleName, "github.com/user/test-proj")
	assert.NotEmpty(t, model.Choices, "Choices should be populated for features step")
	assert.Lenf(t, model.Selected, len(model.Choices), "Selected length = %d, want %d", len(model.Selected), len(model.Choices))
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

	assert.Equalf(t, "test-proj", model.Config.ModuleName, "ModuleName = %q, want placeholder %q", model.Config.ModuleName, "test-proj")
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

	assert.True(t, model.Selected[0], "Selected[0] should be true after space toggle")

	result, _ = model.Update(message)
	model = toModel(t, result)
	assert.False(t, model.Selected[0], "Selected[0] should be false after second toggle")
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

	assert.Equalf(t, StepScaffolding, model.Step, "Step = %d, want StepScaffolding (%d)", model.Step, StepScaffolding)
	assert.True(t, model.Config.EnableValidator, "EnableValidator should be true when selected")
	assert.True(t, model.Config.EnableAgents, "EnableAgents should be true when selected")
	assert.False(t, model.Config.EnableSonicJSON, "EnableSonicJSON should be false when not selected")
	assert.True(t, model.Config.EnableInterpreted, "EnableInterpreted should be true when selected")
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

	assert.Equalf(t, StepFeatures, model.Step, "Step = %d, want StepFeatures (%d) - enter on checkbox should toggle, not advance", model.Step, StepFeatures)
	assert.True(t, model.Selected[0], "Selected[0] should be true after enter toggle")
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

	assert.False(t, model.Config.EnableValidator, "EnableValidator should be false when not selected")
	assert.True(t, model.Config.EnableAgents, "EnableAgents should be true when selected")
	assert.False(t, model.Config.EnableSonicJSON, "EnableSonicJSON should be false when not selected")
	assert.False(t, model.Config.EnableInterpreted, "EnableInterpreted should be false when not selected")
}

func TestUpdate_ErrorMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	testErr := errors.New("scaffold failed")
	result, _ := m.Update(errMessage{err: testErr})
	model := toModel(t, result)

	require.Error(t, model.Err, "Err should not be nil")
	assert.EqualErrorf(t, model.Err, "scaffold failed", "Err = %v, want %q", model.Err, "scaffold failed")
}

func TestUpdate_ScaffoldDoneMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepScaffolding
	m.Config.DestinationPath = "/tmp/test"

	result, _ := m.Update(scaffoldDoneMessage{})
	model := toModel(t, result)

	assert.Equalf(t, StepTidying, model.Step, "Step = %d, want StepTidying (%d)", model.Step, StepTidying)
}

func TestUpdate_TidyDoneMessage(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepTidying

	result, _ := m.Update(tidyDoneMessage{})
	model := toModel(t, result)

	assert.True(t, model.Done, "Done should be true")
	assert.Equalf(t, StepFinished, model.Step, "Step = %d, want StepFinished (%d)", model.Step, StepFinished)
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
			assert.Containsf(t, view.Content, tc.mustHave, "View() should contain %q, got:\n%s", tc.mustHave, view.Content)
		})
	}
}

func TestView_Error(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Err = errors.New("something broke")
	view := m.View()

	assert.Containsf(t, view.Content, "something broke", "View() should show error, got:\n%s", view.Content)
}

func toModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	v, ok := m.(*Model)
	require.Truef(t, ok, "unexpected tea.Model type: %T", m)
	return *v
}
