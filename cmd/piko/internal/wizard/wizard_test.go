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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/cmd/piko/internal/wizard/templates"
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

func TestUpdate_ModulePath_InvalidStaysOnStep(t *testing.T) {
	t.Parallel()

	m := InitialModel()
	m.Step = StepModulePath
	m.Config.ProjectName = "My App"
	m.Inputs = append(m.Inputs, m.Inputs[0])
	m.Inputs[1].Placeholder = "My App"
	m.Inputs[1].SetValue("")

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model := toModel(t, result)

	assert.Equal(t, StepModulePath, model.Step)
	require.ErrorIs(t, model.ModulePathErr, templates.ErrInvalidModuleName)
	assert.Empty(t, model.Config.ModuleName)
	assert.Contains(t, model.View().Content, "invalid module name")

	model.Inputs[1].SetValue("example.com/my-app")
	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = toModel(t, result)

	assert.Equal(t, StepFeatures, model.Step)
	assert.NoError(t, model.ModulePathErr)
	assert.Equal(t, "example.com/my-app", model.Config.ModuleName)
}

func pressKeys(t *testing.T, model *Model, keys ...tea.KeyPressMsg) (*Model, tea.Cmd) {
	t.Helper()

	var command tea.Cmd
	for _, key := range keys {
		var result tea.Model
		result, command = model.Update(key)
		next, ok := result.(*Model)
		require.Truef(t, ok, "unexpected tea.Model type: %T", result)
		model = next
	}
	return model, command
}

func typeText(text string) []tea.KeyPressMsg {
	keys := make([]tea.KeyPressMsg, 0, len(text))
	for _, character := range text {
		keys = append(keys, tea.KeyPressMsg{Code: character, Text: string(character)})
	}
	return keys
}

func TestUpdate_WalksEveryStepWithKeyPresses(t *testing.T) {
	t.Parallel()

	model := newInitialModel()
	require.NotNil(t, model.Init())
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	down := tea.KeyPressMsg{Code: tea.KeyDown}
	up := tea.KeyPressMsg{Code: tea.KeyUp}
	space := tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

	model, _ = pressKeys(t, model, typeText("demo")...)
	model, _ = pressKeys(t, model, enter)
	require.Equal(t, StepDestination, model.Step)
	assert.Equal(t, "demo", model.Config.ProjectName)
	assert.Contains(t, model.View().Content, "./demo/")

	model, _ = pressKeys(t, model, down, up, enter)
	require.Equal(t, StepModulePath, model.Step)
	assert.Equal(t, "demo", model.Config.DestinationPath)

	model, _ = pressKeys(t, model, typeText("example.com/demo")...)
	model, _ = pressKeys(t, model, enter)
	require.Equal(t, StepFeatures, model.Step)
	assert.Equal(t, "example.com/demo", model.Config.ModuleName)
	assert.Contains(t, model.View().Content, "space to toggle")

	model, _ = pressKeys(t, model, up, space, up, up, space, down, down, down)
	model, command := pressKeys(t, model, enter)
	require.Equal(t, StepScaffolding, model.Step)
	require.NotNil(t, command)
	assert.Equal(t, templates.ScaffoldData{
		ProjectName:       "demo",
		ModuleName:        "example.com/demo",
		DestinationPath:   "demo",
		EnableValidator:   true,
		EnableAgents:      true,
		EnableSonicJSON:   true,
		EnableInterpreted: true,
	}, model.Config)

	_, command = model.Update(model.Spinner.Tick())
	assert.NotNil(t, command, "the spinner keeps ticking while scaffolding")

	_, command = model.Update(scaffoldDoneMessage{versionWarning: "offline"})
	require.NotNil(t, command)
	assert.Equal(t, StepTidying, model.Step)
	assert.Equal(t, "offline", model.VersionWarning)
	assert.Contains(t, model.View().Content, "go mod tidy")

	_, _ = model.Update(tidyDoneMessage{warning: "tidy failed"})
	assert.True(t, model.Done)
	assert.Equal(t, "tidy failed", model.TidyWarning)

	_, command = pressKeys(t, model, enter)
	require.NotNil(t, command)
	assert.IsType(t, tea.QuitMsg{}, command())
}

func TestEnterFeatures_ScaffoldsWithTheResolvedVersion(t *testing.T) {
	t.Parallel()

	destination := filepath.Join(t.TempDir(), "demo")
	model := newInitialModel()
	model.Step = StepFeatures
	model.Config = templates.ScaffoldData{ProjectName: "demo", ModuleName: "example.com/demo", DestinationPath: destination}
	model.Choices = make([]string, 4)
	model.Selected = make([]bool, 4)
	model.Cursor = len(model.Choices)
	model.resolveVersion = func() (string, error) { return "v1.2.3", nil }

	_, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, command)
	message := command()

	assert.Equal(t, scaffoldDoneMessage{}, message)
	assert.Equal(t, "v1.2.3", model.Config.PikoVersion)
	goMod, err := os.ReadFile(filepath.Join(destination, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(goMod), "v1.2.3")
}

func TestEnterFeatures_FallsBackWhenTheVersionCannotBeResolved(t *testing.T) {
	t.Parallel()

	model := newInitialModel()
	model.Step = StepFeatures
	model.Config = templates.ScaffoldData{ProjectName: "demo", ModuleName: "example.com/demo", DestinationPath: t.TempDir()}
	model.Choices = make([]string, 4)
	model.Selected = make([]bool, 4)
	model.Cursor = len(model.Choices)
	model.resolveVersion = func() (string, error) { return "", errors.New("offline") }

	_, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, command)
	message := command()

	failure, ok := message.(errMessage)
	require.Truef(t, ok, "unexpected message %T", message)
	assert.ErrorContains(t, failure.err, "already exists")
	assert.Equal(t, fallbackVersion, model.Config.PikoVersion)
}

func TestRunGoModTidy_ReportsAFailureAsAWarning(t *testing.T) {
	t.Parallel()

	destination := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(destination, "go.mod"), []byte("not a module file\n"), 0o600))
	model := newInitialModel()
	model.Config.DestinationPath = destination

	message := model.runGoModTidy()()

	done, ok := message.(tidyDoneMessage)
	require.Truef(t, ok, "unexpected message %T", message)
	assert.Contains(t, done.warning, "'go mod tidy' failed")
}

func TestReportResult(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		model       *Model
		name        string
		wantContain []string
		wantAbsent  []string
		wantCode    int
	}{
		{
			name:        "cancelled",
			model:       &Model{Aborted: true},
			wantCode:    1,
			wantContain: []string{"Project creation cancelled."},
		},
		{
			name:        "failed",
			model:       &Model{Err: errors.New("disk full")},
			wantCode:    1,
			wantContain: []string{"An error occurred during project creation: disk full"},
		},
		{
			name:        "created",
			model:       &Model{Config: templates.ScaffoldData{ProjectName: "demo"}},
			wantCode:    0,
			wantContain: []string{"Piko project 'demo' created successfully!", "1. cd demo", "2. Run 'piko generate'", "3. Run 'air'"},
			wantAbsent:  []string{"Warning:", "AGENTS.md", "go mod tidy"},
		},
		{
			name: "created with warnings and agents",
			model: &Model{
				Config:         templates.ScaffoldData{ProjectName: "demo", EnableAgents: true},
				VersionWarning: "using v0.0.0",
				TidyWarning:    "tidy failed",
			},
			wantCode: 0,
			wantContain: []string{
				"Warning: using v0.0.0",
				"Warning: tidy failed",
				"AGENTS.md + references/",
				"2. Run 'go mod tidy' to fetch dependencies.",
				"3. Run 'piko generate'",
				"4. Run 'air'",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			code := reportResult(&out, testCase.model)

			assert.Equal(t, testCase.wantCode, code)
			for _, want := range testCase.wantContain {
				assert.Contains(t, out.String(), want)
			}
			for _, absent := range testCase.wantAbsent {
				assert.NotContains(t, out.String(), absent)
			}
		})
	}
}

func TestGoVersionWarning(t *testing.T) {
	t.Parallel()

	assert.Empty(t, goVersionWarning(), "the test toolchain meets the required Go version")
}
