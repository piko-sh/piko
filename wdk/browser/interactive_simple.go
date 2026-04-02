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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// simpleRunner implements InteractiveRunner using basic ANSI terminal output.
// It provides a fallback mode for terminals that do not support the full TUI.
type simpleRunner struct {
	// reader reads user input from standard input.
	reader *bufio.Reader

	// testName is the name of the test that is running.
	testName string

	// steps holds the completed steps for rendering.
	steps []Step

	// autoplayGap is the delay between automatic steps in seconds (1-9).
	autoplayGap time.Duration

	// mode is the current runner state (paused or autoplay).
	mode simpleRunMode
}

// simpleRunMode represents the current run mode for a test runner.
type simpleRunMode int

const (
	// simpleModePaused is the starting state where the runner waits for user input.
	simpleModePaused simpleRunMode = iota

	// simpleModeAutoplay is the mode where slides advance automatically after a delay.
	simpleModeAutoplay
)

const (
	// autoplayMinSeconds is the smallest number of seconds allowed for autoplay delay.
	autoplayMinSeconds = 1

	// autoplayMaxSeconds is the largest allowed autoplay delay in seconds.
	autoplayMaxSeconds = 9

	// displaySeparatorWidth is the width of the separator lines in the simple TUI.
	displaySeparatorWidth = 50
)

// Start initialises the runner with the test name.
//
// Takes testName (string) which identifies the test being run.
//
// Returns error when initialisation fails.
func (r *simpleRunner) Start(testName string) error {
	r.testName = testName
	return nil
}

// BeforeStep is called before each action executes.
//
// Takes action (string) which names the action about to run.
// Takes detail (string) which provides additional context for the action.
func (r *simpleRunner) BeforeStep(action, detail string) {
	r.render(action, detail)
}

// AfterStep is called after each action completes.
//
// Takes action (string) which identifies the step that was performed.
// Takes detail (string) which provides additional information about the step.
// Takes failed (bool) which indicates whether the step failed.
// Takes duration (time.Duration) which records how long the step took.
func (r *simpleRunner) AfterStep(action, detail string, failed bool, duration time.Duration) {
	state := StepPassed
	if failed {
		state = StepFailed
	}
	r.steps = append(r.steps, Step{
		Action:   action,
		Detail:   detail,
		State:    state,
		Error:    nil,
		Duration: duration,
	})
}

// WaitForContinue blocks until the user signals to continue.
//
// In autoplay mode, it waits for the set delay then returns. Otherwise, it
// reads user input: an empty line continues, "q" exits the test, and a number
// from 1 to 9 switches to autoplay mode with that many seconds of delay.
func (r *simpleRunner) WaitForContinue() {
	if r.mode == simpleModeAutoplay {
		time.Sleep(r.autoplayGap)
		return
	}

	for {
		input, _ := r.reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "q" || input == "Q" {
			fmt.Println("\n[e2e] Test aborted by user")
			os.Exit(1)
		}

		if input == "" {
			return
		}

		if n, err := strconv.Atoi(input); err == nil && n >= autoplayMinSeconds && n <= autoplayMaxSeconds {
			r.mode = simpleModeAutoplay
			r.autoplayGap = time.Duration(n) * time.Second
			return
		}
	}
}

// Close releases resources and prints a completion message.
func (r *simpleRunner) Close() {
	r.render("", "")
	fmt.Println("\nTest completed.")
}

// render displays the current state of the interactive test runner.
//
// Takes currentAction (string) which is the label for the step in progress,
// or empty if no step is active.
// Takes currentDetail (string) which gives extra context for the current
// action.
func (r *simpleRunner) render(currentAction, currentDetail string) {
	_, _ = fmt.Print("\033[2J\033[H")

	fmt.Printf("E2E: %s (interactive - simple mode)\n", r.testName)
	fmt.Println(strings.Repeat("-", displaySeparatorWidth))
	fmt.Println()

	for _, step := range r.steps {
		icon := "\u2713"
		if step.State == StepFailed {
			icon = "\u2717"
		}
		fmt.Printf("  %s %-14s %s\n", icon, step.Action, step.Detail)
	}

	if currentAction != "" {
		fmt.Printf("  \u25b6 %-14s %s\n", currentAction, currentDetail)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", displaySeparatorWidth))

	if r.mode == simpleModePaused {
		fmt.Println("Mode: PAUSED")
		fmt.Println("[Enter] Continue  [1-9] Autoplay  [q] Quit")
	} else {
		fmt.Printf("Mode: AUTOPLAY (%v delay)\n", r.autoplayGap)
		fmt.Println("[Enter] Pause  [q] Quit")
	}
}

// newSimpleRunner creates a new simple interactive runner.
//
// Returns *simpleRunner which is ready to use in paused mode with stdin input.
func newSimpleRunner() *simpleRunner {
	return &simpleRunner{
		reader:      bufio.NewReader(os.Stdin),
		testName:    "",
		steps:       make([]Step, 0),
		autoplayGap: 0,
		mode:        simpleModePaused,
	}
}
