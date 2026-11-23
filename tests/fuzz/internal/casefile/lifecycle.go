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

package casefile

const (
	// lifecycleSecondOwner names the second asynchronous request owner.
	lifecycleSecondOwner = "second"

	// lifecycleReturnConnections counts connections after navigating back to the journey.
	lifecycleReturnConnections = 4
)

// LifecycleInput controls the user's draft and the order of two asynchronous completions
// in a single browser document.
type LifecycleInput struct {
	// Draft is the user's unsaved text.
	Draft string

	// Reverse reverses the asynchronous completion order.
	Reverse bool
}

// LifecycleState records public observations at a named point in the journey.
type LifecycleState struct {
	// Step names the journey point.
	Step string

	// Path is the browser route.
	Path string

	// Draft is the user's unsaved text.
	Draft string

	// Focus names the focused control.
	Focus string

	// Applied names the request owner whose result was applied.
	Applied string

	// ActiveValue is the active request's displayed value.
	ActiveValue string

	// Prompts counts leave prompts.
	Prompts int

	// Connected counts component connections.
	Connected int

	// Cleaned counts component cleanups.
	Cleaned int

	// Deliveries counts subscription deliveries.
	Deliveries int

	// SameDocument reports whether the browser document was retained.
	SameDocument bool

	// SameRow reports whether the row element identity was retained.
	SameRow bool
}

// LifecycleTrace records a component's navigation and cleanup journey.
type LifecycleTrace struct {
	// FailedStep identifies the last attempted step on failure.
	FailedStep string `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// Input holds the supplied interaction values.
	Input LifecycleInput

	// States holds the ordered interaction snapshots.
	States []LifecycleState
}

// ExpectedLifecycle models the navigation and cleanup journey.
//
// Takes input (LifecycleInput) which selects draft text and completion order.
//
// Returns *LifecycleTrace which holds the expected navigation, request and cleanup
// states.
func ExpectedLifecycle(input LifecycleInput) *LifecycleTrace {
	state := LifecycleState{
		Step:         "",
		Path:         "/journey",
		Draft:        "",
		Focus:        "",
		Applied:      "",
		ActiveValue:  "",
		Prompts:      0,
		Connected:    2,
		Cleaned:      0,
		Deliveries:   0,
		SameDocument: true,
		SameRow:      true,
	}
	trace := &LifecycleTrace{Input: input, States: nil, FailedStep: "", DriverError: ""}
	record := func(step string) { state.Step = step; trace.States = append(trace.States, state) }
	record("mount")
	state.Draft, state.Focus = input.Draft, "draft"
	record("edit-and-refresh")
	state.Prompts++
	record("cancel-navigation")
	state.Deliveries = 2
	record("start-requests")
	state.Cleaned = 1
	record("detach-first")
	for _, owner := range lifecycleOrder(input.Reverse) {
		if owner == lifecycleSecondOwner {
			state.Applied, state.ActiveValue = lifecycleSecondOwner, lifecycleSecondOwner
		}
		record("release-" + owner)
	}
	state.Prompts++
	state.Path, state.Focus, state.Draft, state.SameRow = "/journey-away", "", "", false
	state.Cleaned, state.ActiveValue = 2, ""
	record("navigate-away")
	state.Path, state.Connected = "/journey", lifecycleReturnConnections
	state.Deliveries += 2
	record("back-and-dispatch")
	return trace
}

// lifecycleOrder selects the asynchronous completion order.
//
// Takes reverse (bool) which reverses the asynchronous completion order.
//
// Returns []string which lists the request owners in completion order.
func lifecycleOrder(reverse bool) []string {
	if reverse {
		return []string{lifecycleSecondOwner, "first"}
	}
	return []string{"first", lifecycleSecondOwner}
}
