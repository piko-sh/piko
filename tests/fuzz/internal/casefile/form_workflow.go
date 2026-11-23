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

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// MaxFormSteps bounds saved form-workflow commands.
	MaxFormSteps = 12

	// maxFormValueBytes bounds one form command's text value.
	maxFormValueBytes = 128
)

// FormCommand is a browser operation.
type FormCommand struct {
	// Op selects the operation.
	Op string

	// Value supplies the edited input value.
	Value string `json:",omitempty"`

	// ID selects a persisted SQLite row.
	ID int `json:",omitempty"`
}

// FormWorkflow holds a sequence of form commands.
type FormWorkflow struct {
	// Steps holds the ordered interaction steps.
	Steps []FormCommand
}

// FormRow holds a persisted form row.
type FormRow struct {
	// Label is the displayed label.
	Label string

	// ID is the persisted SQLite row identifier.
	ID int

	// Done is the row's completion state.
	Done bool
}

// FormEntry holds one native form-data entry.
type FormEntry struct {
	// Name is the form-data entry's field name.
	Name string

	// Value is the form-data entry's value.
	Value string
}

// FormState records native form validity and rendered row state.
type FormState struct {
	// Result is the rendered submission result.
	Result *FormRow `json:",omitempty"`

	// Input is the native input value.
	Input string

	// Entries holds the native form-data entries.
	Entries []FormEntry

	// Rows holds the ordered row values.
	Rows []FormRow

	// Revision is the rendered persistence revision.
	Revision int

	// InvalidEvents counts delivered invalid events.
	InvalidEvents int

	// DisabledControlPresent reports whether the disabled fixture input exists.
	DisabledControlPresent bool

	// Valid reports whether the rendered state marks the input valid.
	Valid bool

	// NativeValid reports whether the native input reports itself valid.
	NativeValid bool

	// FormValid reports whether the native form reports itself valid.
	FormValid bool
}

// FormRequest records one workflow request and its public response.
type FormRequest struct {
	// Method is the request's HTTP method.
	Method string

	// Path is the request path.
	Path string

	// InputFields records the submitted public workflow fields.
	InputFields string `json:",omitempty"`

	// JSONResult records the bounded action response body.
	JSONResult string `json:",omitempty"`

	// Status is the HTTP response status.
	Status int
}

// FormObservation records the state and independent reads after one command.
type FormObservation struct {
	// Command is the performed form operation.
	Command FormCommand

	// Requests holds mutation requests made by this command.
	Requests []FormRequest

	// FreshRows holds an independent HTTP row read.
	FreshRows []FormRow

	// StoreRows holds an independent SQLite row read.
	StoreRows []FormRow

	// State holds the rendered form state.
	State FormState

	// FreshRead reports whether the independent HTTP read succeeded.
	FreshRead bool

	// StoreRead reports whether the independent SQLite read succeeded.
	StoreRead bool
}

// FormWorkflowObservation records the form journey and its final database read.
type FormWorkflowObservation struct {
	// BrowserPath records the browser's final route.
	BrowserPath string `json:",omitempty"`

	// PartialPath records the partial reload route.
	PartialPath string

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// Steps holds the ordered interaction steps.
	Steps []FormObservation

	// FinalStore holds the final independent SQLite rows.
	FinalStore []FormRow

	// Initial holds the initial rendered state.
	Initial FormState

	// FailedStep identifies the last attempted step on failure.
	FailedStep int `json:",omitempty"`

	// FinalRead reports whether the final SQLite read succeeded.
	FinalRead bool
}

// NewFormWorkflow models a form journey with two persisted rows.
//
// Takes first (string) which is the first generated row label.
// Takes second (string) which is the second generated row label.
// Takes toggleSecond (bool) which selects whether the second row is toggled.
//
// Returns *FormWorkflow which holds the ordered edit, reset and persisted-row commands.
func NewFormWorkflow(first, second string, toggleSecond bool) *FormWorkflow {
	toggle, remove := 1, 2
	if toggleSecond {
		toggle, remove = 2, 1
	}
	return &FormWorkflow{Steps: []FormCommand{
		{Op: "edit", Value: first, ID: 0}, {Op: "reset", Value: "", ID: 0}, {Op: "invalid", Value: "", ID: 0},
		{Op: "create", Value: first, ID: 0}, {Op: "create", Value: second, ID: 0},
		{Op: "toggle", Value: "", ID: toggle}, {Op: "delete", Value: "", ID: remove}, {Op: "delete", Value: "", ID: toggle},
	}}
}

// formEntries models native form-data entries for the supplied input.
//
// Takes value (string) which is the label field value.
//
// Returns []FormEntry which lists form entry values in their recorded order.
func formEntries(value string) []FormEntry {
	return []FormEntry{{Name: "label", Value: value}, {Name: "repeat", Value: "one"}, {Name: "repeat", Value: "two"}}
}

// EmptyFormState models the initial empty form.
//
// Returns FormState which holds empty label entries and the disabled control.
func EmptyFormState() FormState {
	var state FormState
	state.Entries = formEntries("")
	state.DisabledControlPresent = true
	return state
}

// Model checks operation preconditions and derives every expected state from independent
// inputs.
//
// Returns []FormState which lists form state values in their recorded order.
// Returns error which reports a form history whose command prerequisites are invalid, or
// nil on success.
func (w *FormWorkflow) Model() ([]FormState, error) {
	if w == nil || len(w.Steps) == 0 || len(w.Steps) > MaxFormSteps {
		return nil, fmt.Errorf("expected 1 to %d form operations", MaxFormSteps)
	}
	state := EmptyFormState()
	nextID := 1
	var states []FormState
	for step, command := range w.Steps {
		if !utf8.ValidString(command.Value) || len(command.Value) > maxFormValueBytes || strings.ContainsAny(command.Value, "\x00\r\n") {
			return nil, fmt.Errorf("invalid text at step %d", step)
		}
		state.Rows = slices.Clone(state.Rows)
		if err := applyFormCommand(&state, command, step, &nextID); err != nil {
			return nil, err
		}
		state.Valid = state.Input != ""
		state.NativeValid = state.Valid
		state.FormValid = state.Valid
		state.Entries = formEntries(state.Input)
		states = append(states, state)
	}
	return states, nil
}

// ReduceFormWorkflow proposes shorter valid form histories.
//
// Takes w (*FormWorkflow) which holds the saved form workflow.
//
// Returns []*FormWorkflow which lists form workflow values in their recorded order.
func ReduceFormWorkflow(w *FormWorkflow) []*FormWorkflow {
	if w == nil {
		return nil
	}
	var candidates []*FormWorkflow
	for index := range w.Steps {
		candidate := &FormWorkflow{Steps: slices.Delete(slices.Clone(w.Steps), index, index+1)}
		if _, err := candidate.Model(); err == nil {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

// applyFormCommand applies one form command to the modelled state.
//
// Takes state (*FormState) which is the modelled state updated in place.
// Takes command (FormCommand) which is the operation to apply.
// Takes step (int) which is the command's zero-based position.
// Takes nextID (*int) which is the next persisted row identifier, advanced on create.
//
// Returns error which reports a command whose prerequisites are invalid, or nil on
// success.
func applyFormCommand(state *FormState, command FormCommand, step int, nextID *int) error {
	switch command.Op {
	case "edit":
		state.Input = command.Value
	case "reset":
		state.Input = ""
	case "invalid":
		if state.Input != "" {
			return fmt.Errorf("valid label at invalid submit %d", step)
		}
		state.InvalidEvents++
	case "create":
		if command.Value == "" {
			return fmt.Errorf("missing create label at %d", step)
		}
		state.Input = command.Value
		row := FormRow{ID: *nextID, Label: command.Value, Done: false}
		*nextID++
		state.Rows = append(state.Rows, row)
		state.Result = &row
		state.Revision++
	case "toggle", "delete":
		return changeFormRow(state, command, step)
	default:
		return fmt.Errorf("unknown form operation %q", command.Op)
	}
	return nil
}

// changeFormRow toggles or deletes one persisted row in the modelled state.
//
// Takes state (*FormState) which is the modelled state updated in place.
// Takes command (FormCommand) which is the toggle or delete operation.
// Takes step (int) which is the command's zero-based position.
//
// Returns error which reports an absent row, or nil on success.
func changeFormRow(state *FormState, command FormCommand, step int) error {
	position := slices.IndexFunc(state.Rows, func(row FormRow) bool { return row.ID == command.ID })
	if position < 0 {
		return fmt.Errorf("absent task %d at step %d", command.ID, step)
	}
	row := state.Rows[position]
	if command.Op == "toggle" {
		row.Done = !row.Done
		state.Rows[position] = row
	} else {
		state.Rows = slices.Delete(state.Rows, position, position+1)
	}
	state.Result = &row
	state.Revision++
	return nil
}
