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
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// tuiRunMode represents the current mode of the text user interface.
type tuiRunMode int

const (
	// tuiModePaused is the paused state where auto-play has stopped.
	tuiModePaused tuiRunMode = iota

	// tuiModeAutoplay is the mode where tests advance automatically after a delay.
	tuiModeAutoplay
)

const (
	// stepChannelBufferSize is the buffer size for the channel that sends step
	// updates to the TUI.
	stepChannelBufferSize = 10

	// stepDetailMaxDisplayLen is the maximum length for step details before
	// truncation.
	stepDetailMaxDisplayLen = 50

	// truncationSuffixLen is the length of the "..." suffix used when truncating.
	truncationSuffixLen = 3

	// tuiSectionSeparator is a blank line used to separate sections in the TUI
	// output.
	tuiSectionSeparator = "\n\n"
)

// tuiRunner implements InteractiveRunner using a Bubble Tea TUI.
type tuiRunner struct {
	// stepChan sends step updates to the TUI for display.
	stepChan chan Step

	// continueChan signals when to proceed after a step completes.
	continueChan chan struct{}

	// doneChan signals when the TUI has finished running.
	doneChan chan struct{}

	// quitChan signals when the user wants to stop the test.
	quitChan chan struct{}

	// program is the Bubble Tea program that runs the interactive TUI.
	program *tea.Program

	// testName is the name of the test being run.
	testName string
}

// Start initialises the runner with the test name.
//
// Takes testName (string) which identifies the test being run.
//
// Returns error when initialisation fails.
//
// Safe for concurrent use. Spawns a goroutine that runs
// the TUI until the program exits.
func (r *tuiRunner) Start(testName string) error {
	r.testName = testName

	model := newTUIModel(testName, r.stepChan, r.continueChan, r.quitChan)
	r.program = tea.NewProgram(model)

	go func() {
		defer close(r.doneChan)
		_, _ = r.program.Run()
	}()

	return nil
}

// BeforeStep is called before each action runs.
//
// Takes action (string) which describes the step being done.
// Takes detail (string) which gives more context about the step.
func (r *tuiRunner) BeforeStep(action, detail string) {
	r.stepChan <- Step{
		Action:   action,
		Detail:   detail,
		State:    StepRunning,
		Error:    nil,
		Duration: 0,
	}
}

// AfterStep is called after each action completes.
//
// Takes action (string) which names the step that was performed.
// Takes detail (string) which provides additional context about the step.
// Takes failed (bool) which indicates whether the step failed.
// Takes duration (time.Duration) which records how long the step took.
func (r *tuiRunner) AfterStep(action, detail string, failed bool, duration time.Duration) {
	state := StepPassed
	if failed {
		state = StepFailed
	}
	r.stepChan <- Step{
		Action:   action,
		Detail:   detail,
		State:    state,
		Error:    nil,
		Duration: duration,
	}
}

// WaitForContinue blocks until the user signals to continue.
//
// Panics when the user quits via the quit channel, aborting the test.
func (r *tuiRunner) WaitForContinue() {
	select {
	case <-r.continueChan:
		return
	case <-r.quitChan:
		panic("browser: test aborted by user")
	case <-r.doneChan:
		return
	}
}

// Close releases resources held by the runner.
func (r *tuiRunner) Close() {
	r.stepChan <- Step{
		Action:   "__done__",
		Detail:   "",
		State:    StepPending,
		Error:    nil,
		Duration: 0,
	}

	<-r.doneChan
}

// tuiModel is the Bubble Tea model for the interactive test display.
type tuiModel struct {
	// stepChan receives step updates from the linter for display in the TUI.
	stepChan chan Step

	// continueChan signals the processor to move to the next step.
	continueChan chan struct{}

	// quitChan signals when the user wants to quit the application.
	quitChan chan struct{}

	// testName is the name of the current E2E test shown in the display.
	testName string

	// steps holds the processing steps in the order they run.
	steps []Step

	// autoplayGap is the delay between steps in autoplay mode.
	autoplayGap time.Duration

	// currentStep is the index of the step that is running or about to run.
	currentStep int

	// mode tracks the current playback state: paused or autoplay.
	mode tuiRunMode

	// quitting indicates whether the user has requested to exit.
	quitting bool

	// completed indicates whether the test run has finished.
	completed bool
}

// stepUpdateMessage is sent when a step update is received from the channel.
type stepUpdateMessage Step

// tuiTickMessage is sent when the autoplay timer fires.
type tuiTickMessage time.Time

var (
	// tuiTitleStyle defines the Lip Gloss style for the test name heading.
	tuiTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	// tuiPassedStyle defines the Lip Gloss style for passed step indicators.
	tuiPassedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// tuiFailedStyle defines the Lip Gloss style for failed step indicators.
	tuiFailedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// tuiRunningStyle defines the Lip Gloss style for currently running step indicators.
	tuiRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	// tuiPendingStyle defines the Lip Gloss style for pending
	// step indicators and secondary text.
	tuiPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// tuiHelpStyle defines the Lip Gloss style for help text at the bottom of the TUI.
	tuiHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// tuiModeStyle defines the Lip Gloss style for the playback mode label.
	tuiModeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))

	// tuiBoxStyle defines the Lip Gloss style for the outer border of the TUI display.
	tuiBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

// Init implements tea.Model.
//
// Returns tea.Cmd which starts listening for step updates.
func (m tuiModel) Init() tea.Cmd {
	return m.waitForStep()
}

// Update handles incoming messages and updates the model state.
// Implements tea.Model.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the next command to run, or nil if none.
func (m tuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(message)

	case stepUpdateMessage:
		return m.handleStepUpdate(Step(message))

	case tuiTickMessage:
		if m.mode == tuiModeAutoplay && !m.completed {
			select {
			case m.continueChan <- struct{}{}:
			default:
			}
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
//
// Returns tea.View which is the rendered TUI display.
func (m tuiModel) View() tea.View {
	var b strings.Builder

	b.WriteString(tuiTitleStyle.Render(fmt.Sprintf("E2E: %s", m.testName)))
	b.WriteString(tuiSectionSeparator)

	if len(m.steps) == 0 {
		b.WriteString(tuiPendingStyle.Render("  Waiting for first step..."))
		b.WriteString("\n")
	} else {
		for i, step := range m.steps {
			icon := m.stepIcon(step.State)
			stepNum := fmt.Sprintf("%2d.", i+1)
			description := m.formatStepDescription(step)

			_, _ = fmt.Fprintf(&b, "  %s %s %s\n", icon, stepNum, description)
		}
	}

	b.WriteString("\n")

	if m.completed {
		b.WriteString(tuiPassedStyle.Render("  Test completed!"))
		b.WriteString(tuiSectionSeparator)
		b.WriteString(tuiHelpStyle.Render("  Press any key to exit"))
	} else if m.hasFailed() {
		b.WriteString(tuiFailedStyle.Render("  ✗ Step failed"))
		b.WriteString(tuiSectionSeparator)
		b.WriteString(tuiHelpStyle.Render("  Press Enter to continue, or inspect browser state"))
	} else {
		if m.mode == tuiModePaused {
			b.WriteString(tuiModeStyle.Render("  Mode: PAUSED"))
			b.WriteString(tuiSectionSeparator)
			b.WriteString(tuiHelpStyle.Render("  [Enter] Next step  [1-9] Autoplay (Ns delay)  [q] Quit"))
		} else {
			b.WriteString(tuiModeStyle.Render(fmt.Sprintf("  Mode: AUTOPLAY (%s delay)", m.autoplayGap)))
			b.WriteString(tuiSectionSeparator)
			b.WriteString(tuiHelpStyle.Render("  [0/Space] Pause  [1-9] Change speed  [q] Quit"))
		}
	}

	b.WriteString("\n")

	v := tea.NewView(tuiBoxStyle.Render(b.String()))
	v.AltScreen = true
	return v
}

// waitForStep returns a command that waits for the next step update.
//
// Returns tea.Cmd which waits until a step arrives on the channel.
func (m tuiModel) waitForStep() tea.Cmd {
	return func() tea.Msg {
		step := <-m.stepChan
		return stepUpdateMessage(step)
	}
}

// handleKeyPress processes keyboard input and updates the model state.
//
// Takes message (tea.KeyMessage) which contains the key event to handle.
//
// Returns tea.Model which is the updated model after handling the key.
// Returns tea.Cmd which is a command to run, or nil if no action is needed.
func (m tuiModel) handleKeyPress(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		close(m.quitChan)
		return m, tea.Quit

	case "enter":
		if m.completed {
			return m, tea.Quit
		}
		if m.mode == tuiModePaused {
			select {
			case m.continueChan <- struct{}{}:
			default:
			}
		}
		return m, nil

	case "0", " ":
		m.mode = tuiModePaused
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		seconds := int(message.String()[0] - '0')
		m.autoplayGap = time.Duration(seconds) * time.Second
		m.mode = tuiModeAutoplay

		select {
		case m.continueChan <- struct{}{}:
		default:
		}

		return m, nil
	}

	return m, nil
}

// handleStepUpdate processes a step update from the Page.
//
// Takes step (Step) which contains the step data to process.
//
// Returns tea.Model which is the updated model after processing the step.
// Returns tea.Cmd which provides the next command to run.
func (m tuiModel) handleStepUpdate(step Step) (tea.Model, tea.Cmd) {
	if step.Action == "__done__" {
		m.completed = true
		return m, tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
			return tea.QuitMsg{}
		})
	}

	if step.State == StepRunning {
		isFirstStep := len(m.steps) == 0

		m.steps = append(m.steps, step)
		m.currentStep = len(m.steps) - 1

		if isFirstStep && m.mode == tuiModeAutoplay {
			select {
			case m.continueChan <- struct{}{}:
			default:
			}
		}

		return m, m.waitForStep()
	}

	if len(m.steps) > 0 {
		index := len(m.steps) - 1
		m.steps[index].State = step.State
		m.steps[index].Duration = step.Duration
	}

	if m.mode == tuiModeAutoplay && step.State == StepPassed {
		return m, tea.Batch(
			m.waitForStep(),
			tea.Tick(m.autoplayGap, func(t time.Time) tea.Msg {
				return tuiTickMessage(t)
			}),
		)
	}

	return m, m.waitForStep()
}

// stepIcon returns the icon for a given step state.
//
// Takes state (StepState) which specifies the current step state.
//
// Returns string which is the styled icon for the state.
func (tuiModel) stepIcon(state StepState) string {
	switch state {
	case StepPassed:
		return tuiPassedStyle.Render("✓")
	case StepFailed:
		return tuiFailedStyle.Render("✗")
	case StepRunning:
		return tuiRunningStyle.Render("▶")
	default:
		return tuiPendingStyle.Render("○")
	}
}

// formatStepDescription creates a readable description of a step.
//
// Takes step (Step) which contains the action, detail, state, and duration.
//
// Returns string which is the formatted description with shortened details and
// duration shown for finished steps.
func (tuiModel) formatStepDescription(step Step) string {
	detail := truncateRunes(step.Detail, stepDetailMaxDisplayLen-truncationSuffixLen)

	if step.State == StepPassed || step.State == StepFailed {
		if step.Duration > 0 {
			return fmt.Sprintf("%-16s %-50s %s", step.Action, detail, tuiPendingStyle.Render(step.Duration.Round(time.Millisecond).String()))
		}
	}

	return fmt.Sprintf("%-16s %s", step.Action, detail)
}

// hasFailed returns true if any step has failed.
//
// Returns bool which is true when at least one step has a failed state.
func (m tuiModel) hasFailed() bool {
	for _, step := range m.steps {
		if step.State == StepFailed {
			return true
		}
	}
	return false
}

// NewTUIRunner creates a new TUI-based interactive runner.
//
// Returns InteractiveRunner which is set up with buffered channels for step
// processing, continuation signals, completion, and quit handling.
func NewTUIRunner() InteractiveRunner {
	return &tuiRunner{
		stepChan:     make(chan Step, stepChannelBufferSize),
		continueChan: make(chan struct{}),
		doneChan:     make(chan struct{}),
		quitChan:     make(chan struct{}),
		program:      nil,
		testName:     "",
	}
}

// newTUIModel creates a new TUI model.
//
// Takes testName (string) which identifies the test being displayed.
// Takes stepChan (chan Step) which receives steps to display.
// Takes continueChan (chan struct{}) which signals when to continue.
// Takes quitChan (chan struct{}) which signals when to quit.
//
// Returns tuiModel which is initialised in autoplay mode with a one second
// gap between steps.
func newTUIModel(testName string, stepChan chan Step, continueChan chan struct{}, quitChan chan struct{}) tuiModel {
	return tuiModel{
		stepChan:     stepChan,
		continueChan: continueChan,
		quitChan:     quitChan,
		testName:     testName,
		steps:        make([]Step, 0),
		autoplayGap:  1 * time.Second,
		currentStep:  0,
		mode:         tuiModeAutoplay,
		quitting:     false,
		completed:    false,
	}
}
