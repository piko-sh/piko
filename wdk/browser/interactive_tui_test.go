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

package browser

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTUIModel() (tuiModel, chan Step, chan struct{}, chan struct{}) {
	steps := make(chan Step, stepChannelBufferSize)
	continues := make(chan struct{}, 1)
	quit := make(chan struct{})
	return newTUIModel("TestFlow", steps, continues, quit), steps, continues, quit
}

func updateModel(t *testing.T, model tuiModel, message tea.Msg) (tuiModel, tea.Cmd) {
	t.Helper()
	updated, command := model.Update(message)
	next, ok := updated.(tuiModel)
	require.True(t, ok)
	return next, command
}

func keyPress(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: text, Code: []rune(text)[0]}
}

func drainContinue(continues chan struct{}) bool {
	select {
	case <-continues:
		return true
	default:
		return false
	}
}

func TestTUIModelKeys(t *testing.T) {
	model, _, continues, quit := newTestTUIModel()
	assert.Equal(t, tuiModeAutoplay, model.mode)

	model, _ = updateModel(t, model, keyPress("0"))
	assert.Equal(t, tuiModePaused, model.mode)
	model, command := updateModel(t, model, tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, command)
	assert.True(t, drainContinue(continues), "enter continues a paused run")

	model, _ = updateModel(t, model, keyPress("4"))
	assert.Equal(t, tuiModeAutoplay, model.mode)
	assert.Equal(t, 4*time.Second, model.autoplayGap)
	assert.True(t, drainContinue(continues))
	model, _ = updateModel(t, model, tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.False(t, drainContinue(continues), "enter does not continue during autoplay")
	model, _ = updateModel(t, model, keyPress("x"))
	assert.Equal(t, tuiModeAutoplay, model.mode)

	model, _ = updateModel(t, model, tuiTickMessage(time.Now()))
	assert.True(t, drainContinue(continues), "a tick continues autoplay")
	model, _ = updateModel(t, model, "unrelated")

	model, command = updateModel(t, model, keyPress("q"))
	assert.True(t, model.quitting)
	require.NotNil(t, command)
	_, isQuit := command().(tea.QuitMsg)
	assert.True(t, isQuit)
	_, open := <-quit
	assert.False(t, open, "quitting closes the quit channel")
}

func TestTUIModelStepUpdates(t *testing.T) {
	model, steps, continues, _ := newTestTUIModel()
	assert.Contains(t, model.View().Content, "Waiting for first step")
	steps <- Step{Action: "Click", State: StepRunning}
	message := model.Init()()
	model, command := updateModel(t, model, message)
	require.NotNil(t, command)
	assert.True(t, drainContinue(continues), "autoplay starts on the first step")
	require.Len(t, model.steps, 1)

	model, _ = updateModel(t, model, stepUpdateMessage{Action: "Click", State: StepPassed, Duration: 1500 * time.Microsecond})
	assert.Equal(t, StepPassed, model.steps[0].State)
	assert.Contains(t, model.View().Content, "AUTOPLAY")

	model, _ = updateModel(t, model, stepUpdateMessage{Action: "Fill", Detail: strings.Repeat("x", 80), State: StepRunning})
	model, _ = updateModel(t, model, stepUpdateMessage{Action: "Fill", State: StepFailed})
	assert.True(t, model.hasFailed())
	view := model.View()
	assert.True(t, view.AltScreen)
	assert.Contains(t, view.Content, "Step failed")
	assert.Contains(t, view.Content, "...")

	model.mode = tuiModePaused
	model.steps[1].State = StepPassed
	assert.Contains(t, model.View().Content, "PAUSED")

	model, command = updateModel(t, model, stepUpdateMessage{Action: "__done__"})
	assert.True(t, model.completed)
	require.NotNil(t, command)
	assert.Contains(t, model.View().Content, "Test completed")
	_, command = updateModel(t, model, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, command)
}

func TestTUIModelStepIcons(t *testing.T) {
	model, _, _, _ := newTestTUIModel()
	for state, icon := range map[StepState]string{StepPassed: "✓", StepFailed: "✗", StepRunning: "▶", StepPending: "○"} {
		assert.Contains(t, model.stepIcon(state), icon)
	}
	assert.Equal(t, "Click            #go", strings.TrimRight(model.formatStepDescription(Step{Action: "Click", Detail: "#go", State: StepRunning}), " "))
	assert.Contains(t, model.formatStepDescription(Step{Action: "Click", Detail: "#go", State: StepPassed, Duration: 2 * time.Millisecond}), "2ms")
}

func TestTUIRunnerChannels(t *testing.T) {
	runner, ok := NewTUIRunner().(*tuiRunner)
	require.True(t, ok)
	runner.BeforeStep("Click", "#go")
	runner.AfterStep("Click", "#go", true, time.Millisecond)
	assert.Equal(t, Step{Action: "Click", Detail: "#go", State: StepRunning}, <-runner.stepChan)
	assert.Equal(t, StepFailed, (<-runner.stepChan).State)

	go func() { runner.continueChan <- struct{}{} }()
	runner.WaitForContinue()

	close(runner.doneChan)
	runner.WaitForContinue()
	runner.Close()
	assert.Equal(t, "__done__", (<-runner.stepChan).Action)

	close(runner.quitChan)
	assert.Panics(t, func() {
		runner.doneChan = make(chan struct{})
		runner.WaitForContinue()
	})
}
