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

import "time"

// StepState represents the current state of a test step during its run.
type StepState int

const (
	// StepPending is the state when a step has not yet started.
	StepPending StepState = iota

	// StepRunning indicates the step is currently executing.
	StepRunning

	// StepPassed indicates the step completed successfully.
	StepPassed

	// StepFailed indicates the step failed with an error.
	StepFailed
)

// Step represents a single test step for tracking progress and results.
type Step struct {
	// Error holds any error if the step failed; nil means success.
	Error error

	// Action is the operation being done, shown as a label in output.
	Action string

	// Detail provides extra context about the step.
	Detail string

	// Duration is how long the step took to complete.
	Duration time.Duration

	// State is the current status of the step (running, passed, or failed).
	State StepState
}

// InteractiveRunner provides the interface for interactive test execution.
// Implementations control how test steps are presented to users and how
// user input is gathered for step-through debugging.
type InteractiveRunner interface {
	// Start sets up the runner with the given test name.
	//
	// Takes testName (string) which identifies the test to run.
	//
	// Returns error when setup fails.
	//
	// This should be called once before any steps are run.
	Start(testName string) error

	// BeforeStep is called before each action runs.
	// The runner should show the step as running and wait for user input
	// if in paused mode.
	//
	// Takes action (string) which is the name of the step to run.
	// Takes detail (string) which gives extra information about the step.
	BeforeStep(action, detail string)

	// AfterStep is called after each action finishes.
	//
	// Takes action (string) which names the completed step.
	// Takes detail (string) which gives extra information about the result.
	// Takes failed (bool) which shows whether the step did not succeed.
	// Takes duration (time.Duration) which is how long the step took.
	AfterStep(action, detail string, failed bool, duration time.Duration)

	// WaitForContinue blocks until the user signals to continue.
	// In autoplay mode, this returns at once after a short delay.
	WaitForContinue()

	// Close releases all resources held by the runner.
	Close()
}
