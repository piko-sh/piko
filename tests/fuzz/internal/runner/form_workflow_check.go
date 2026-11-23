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

package runner

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// formInitialSubject names the workflow's initial-state failures.
	formInitialSubject = "initial"
)

// formMismatch identifies a workflow field mismatch.
type formMismatch struct {
	// subject identifies the workflow step.
	subject string

	// field names the mismatched property.
	field string

	// detail describes the mismatch.
	detail string
}

// formCreateInputs is the expected wire input of a create action.
type formCreateInputs struct {
	// Label is the submitted row label.
	Label string `json:"label"`

	// Repeat holds the submitted repeated field values.
	Repeat []string `json:"repeat"`
}

// formRowInputs is the expected wire input of a row-targeted action.
type formRowInputs struct {
	// ID selects the targeted row.
	ID int `json:"id"`
}

// formActionRow is the decoded action result, with absent fields left nil.
type formActionRow struct {
	// ID is the row identity.
	ID *int `json:"id"`

	// Label is the row label.
	Label *string `json:"label"`

	// Done reports whether the row is complete.
	Done *bool `json:"done"`
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e formMismatch) Error() string {
	return fmt.Sprintf("workflow/%s: %s: %s", e.field, e.subject, e.detail)
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e formMismatch) FailureSubject() string { return "/workflow/" + e.subject + "/" + e.field }

// formDifference creates a workflow failure with a stable field identity.
//
// Takes subject (string) which identifies the checked property.
// Takes field (string) which identifies the mismatched field.
// Takes got (any) which is the observed field value.
// Takes want (any) which is the independently expected field value.
//
// Returns error which is the non-nil mismatch for the selected workflow field.
func formDifference(subject, field string, got, want any) error {
	return formMismatch{subject: subject, field: field, detail: fmt.Sprintf("got %+v, want %+v", got, want)}
}

// checkFormState compares native form and rendered state with the model.
//
// Takes subject (string) which identifies the checked property.
// Takes got (casefile.FormState) which holds the observed result.
// Takes want (casefile.FormState) which holds the independently expected result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormState(subject string, got, want casefile.FormState) error {
	switch {
	case got.DisabledControlPresent != want.DisabledControlPresent:
		return formDifference(subject, "disabled-control", got.DisabledControlPresent, want.DisabledControlPresent)
	case got.Input != want.Input:
		return formDifference(subject, "native-value", got.Input, want.Input)
	case got.Valid != want.Valid:
		return formDifference(subject, "validity", got.Valid, want.Valid)
	case got.NativeValid != want.NativeValid:
		return formDifference(subject, "shadow-validity", got.NativeValid, want.NativeValid)
	case got.FormValid != want.FormValid:
		return formDifference(subject, "form-validity", got.FormValid, want.FormValid)
	case !slices.Equal(got.Entries, want.Entries):
		return formDifference(subject, "form-data", got.Entries, want.Entries)
	case got.InvalidEvents != want.InvalidEvents:
		return formDifference(subject, "invalid-events", got.InvalidEvents, want.InvalidEvents)
	case !slices.Equal(got.Rows, want.Rows):
		return formDifference(subject, "partial-rows", got.Rows, want.Rows)
	case got.Revision != want.Revision:
		return formDifference(subject, "revision", got.Revision, want.Revision)
	case !reflect.DeepEqual(got.Result, want.Result):
		return formDifference(subject, "action-result", got.Result, want.Result)
	}
	return nil
}

// expectedFormRequests models the mutation requests made by a form operation.
//
// Takes operation (string) which selects the browser or expression operation.
// Takes partial (string) which is the form's partial reload path.
//
// Returns []casefile.FormRequest which lists form request values in their recorded order.
func expectedFormRequests(operation, partial string) []casefile.FormRequest {
	var action string
	switch operation {
	case "create":
		action = "Create"
	case "toggle":
		action = "Toggle"
	case "delete":
		action = "Delete"
	default:
		return nil
	}
	return []casefile.FormRequest{
		{Method: http.MethodPost, Path: "/_piko/actions/tasks." + action, Status: http.StatusOK, InputFields: "", JSONResult: ""},
		{Method: http.MethodGet, Path: partial, Status: http.StatusOK, InputFields: "", JSONResult: ""},
	}
}

// checkRequests checks mutation requests and their public action results.
//
// Takes subject (string) which identifies the checked property.
// Takes command (casefile.FormCommand) which selects the saved form operation.
// Takes partial (string) which is the form's partial reload path.
// Takes got ([]casefile.FormRequest) which holds the observed result.
// Takes result (*casefile.FormRow) which is the independently expected action-result row.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkRequests(subject string, command casefile.FormCommand, partial string, got []casefile.FormRequest, result *casefile.FormRow) error {
	want := expectedFormRequests(command.Op, partial)
	if len(got) != len(want) {
		return formDifference(subject, "requests", got, want)
	}
	for index, observed := range got {
		body := observed.JSONResult
		input := observed.InputFields
		observed.JSONResult = ""
		observed.InputFields = ""
		if observed != want[index] {
			return formDifference(subject, "requests", observed, want[index])
		}
		if observed.Method != http.MethodPost {
			continue
		}
		if err := checkMutationRequest(subject, command, input, body, result); err != nil {
			return err
		}
	}
	return nil
}

// checkMutationRequest checks one mutation request's encoded inputs and typed result.
//
// Takes subject (string) which identifies the checked property.
// Takes command (casefile.FormCommand) which selects the saved form operation.
// Takes input (string) which is the request's encoded input fields.
// Takes body (string) which is the action's JSON result.
// Takes result (*casefile.FormRow) which is the independently expected action-result row.
//
// Returns error which reports an input, decoding or result mismatch.
func checkMutationRequest(subject string, command casefile.FormCommand, input, body string, result *casefile.FormRow) error {
	var expectedInputs any = formRowInputs{ID: command.ID}
	if command.Op == "create" {
		expectedInputs = formCreateInputs{Label: command.Value, Repeat: []string{"one", "two"}}
	}
	encoded, err := json.Marshal(expectedInputs)
	if err != nil {
		return fmt.Errorf("workflow/request-fields: encode expected inputs: %w", err)
	}
	if input != string(encoded) {
		return formDifference(subject, "request-fields", input, string(encoded))
	}
	var value formActionRow
	err = json.Unmarshal([]byte(body), &value)
	if err != nil || value.ID == nil || value.Label == nil || value.Done == nil {
		return formDifference(subject, "response-json", body, "typed id/label/done")
	}
	row := casefile.FormRow{ID: *value.ID, Label: *value.Label, Done: *value.Done}
	if result == nil || row != *result {
		return formDifference(subject, "response-result", row, result)
	}
	return nil
}

// verifyFormWorkflow compares browser, fresh HTTP client and direct SQLite observations
// against the recorded model.
//
// Takes workflow (*casefile.FormWorkflow) which holds the saved form workflow.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func verifyFormWorkflow(workflow *casefile.FormWorkflow, got *casefile.FormWorkflowObservation) error {
	_, err := checkFormWorkflow(workflow, got)
	return err
}

// VerifiedFormSteps gives credit only to the independently verified prefix of the
// receipt.
//
// Takes workflow (*casefile.FormWorkflow) which holds the saved form workflow.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed result.
//
// Returns int which counts the independently verified prefix of form steps.
func VerifiedFormSteps(workflow *casefile.FormWorkflow, got *casefile.FormWorkflowObservation) int {
	verified, _ := checkFormWorkflow(workflow, got)
	return verified
}

// checkFormWorkflow checks form steps and independent HTTP and SQLite reads.
//
// Takes workflow (*casefile.FormWorkflow) which holds the saved form workflow.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed result.
//
// Returns int which counts the matching prefix of form commands.
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormWorkflow(workflow *casefile.FormWorkflow, got *casefile.FormWorkflowObservation) (int, error) {
	states, err := workflow.Model()
	if err != nil {
		return 0, fmt.Errorf("workflow/recipe: %w", err)
	}
	if err := checkFormWorkflowStart(got); err != nil {
		return 0, err
	}
	for index := range got.Steps {
		if index >= len(states) {
			return index, formDifference("steps", "count", len(got.Steps), len(states))
		}
		command := workflow.Steps[index]
		if err := checkFormStep(command, states[index], got.Steps[index], got.PartialPath); err != nil {
			return index, err
		}
		if got.DriverError != "" && got.FailedStep == index {
			return index, formMismatch{subject: formStepName(command), field: "driver", detail: got.DriverError}
		}
	}
	return finishFormWorkflow(workflow, states, got)
}

// finishFormWorkflow checks the workflow's driver outcome, step count and final store.
//
// Takes workflow (*casefile.FormWorkflow) which holds the saved form workflow.
// Takes states ([]casefile.FormState) which holds the modelled state after each step.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed result.
//
// Returns int which counts the recorded steps.
// Returns error which reports a driver failure, a count mismatch or a final-store
// mismatch, or nil when the workflow completed as modelled.
func finishFormWorkflow(workflow *casefile.FormWorkflow, states []casefile.FormState, got *casefile.FormWorkflowObservation) (int, error) {
	recorded := len(got.Steps)
	if got.DriverError != "" {
		subject := "final"
		if got.FailedStep >= 0 && got.FailedStep < len(workflow.Steps) {
			subject = formStepName(workflow.Steps[got.FailedStep])
		}
		return recorded, formMismatch{subject: subject, field: "driver", detail: got.DriverError}
	}
	if recorded != len(states) {
		return recorded, formDifference("steps", "count", recorded, len(states))
	}
	if !got.FinalRead || !slices.Equal(got.FinalStore, states[len(states)-1].Rows) {
		return recorded, formDifference("final", "sqlite-rows", got.FinalStore, states[len(states)-1].Rows)
	}
	return recorded, nil
}

// checkFormWorkflowStart checks the workflow receipt's presence, driver, partial source
// and initial state.
//
// Takes got (*casefile.FormWorkflowObservation) which holds the observed result.
//
// Returns error which reports a missing receipt, an early driver failure, a wrong partial
// source or a non-empty initial state.
func checkFormWorkflowStart(got *casefile.FormWorkflowObservation) error {
	if got == nil {
		return formMismatch{subject: formInitialSubject, field: "missing", detail: "browser did not supply a workflow observation"}
	}
	if got.DriverError != "" && got.FailedStep < 0 {
		return formMismatch{subject: formInitialSubject, field: "driver", detail: got.DriverError}
	}
	if !strings.HasPrefix(got.PartialPath, "/_piko/partials/") {
		return formDifference(formInitialSubject, "partial-source", got.PartialPath, "mounted public partial path")
	}
	return checkFormState(formInitialSubject, got.Initial, casefile.EmptyFormState())
}

// compareFormWorkflows compares form observations across server modes.
//
// Takes a (*casefile.FormWorkflowObservation) which holds the first value to compare.
// Takes b (*casefile.FormWorkflowObservation) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareFormWorkflows(a, b *casefile.FormWorkflowObservation) error {
	if !reflect.DeepEqual(a, b) {
		return formDifference("modes", "parity", a, b)
	}
	return nil
}

// recoverPartialForm records whatever form receipt was written before V8 coverage failed,
// whether the journey completed or stopped at a driver failure.
//
// Takes work (string) which is the invocation-private working directory.
// Takes want (*casefile.FormWorkflow) which holds the independently expected result.
// Takes o (*Observation) which receives the application observation.
//
// Returns error which reports an unreadable or invalid receipt or the independently
// checked journey failure, or nil when the receipt is absent or passes.
func recoverPartialForm(work string, want *casefile.FormWorkflow, o *Observation) error {
	data, err := readBrowserReceipt(work, browserReceiptName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("partial form receipt: %w", err)
	}
	var got casefile.FormWorkflowObservation
	if err := json.Unmarshal(data, &got); err != nil {
		return fmt.Errorf("partial form receipt: %w", err)
	}
	o.FormWorkflow = &got
	return verifyFormWorkflow(want, &got)
}

// checkFormStep checks one form command's state and independent evidence.
//
// Takes command (casefile.FormCommand) which selects the saved form operation.
// Takes want (casefile.FormState) which holds the independently expected result.
// Takes step (casefile.FormObservation) which holds one form command's state and
// independent reads.
// Takes partialPath (string) which is the form's partial reload path.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormStep(command casefile.FormCommand, want casefile.FormState, step casefile.FormObservation, partialPath string) error {
	subject := formStepName(command)
	if step.Command != command {
		return formDifference(subject, "command", step.Command, command)
	}
	if err := checkFormState(subject, step.State, want); err != nil {
		return err
	}
	if err := checkRequests(subject, command, partialPath, step.Requests, want.Result); err != nil {
		return err
	}
	if !step.FreshRead || !slices.Equal(step.FreshRows, want.Rows) {
		return formDifference(subject, "fresh-client-rows", step.FreshRows, want.Rows)
	}
	if !step.StoreRead || !slices.Equal(step.StoreRows, want.Rows) {
		return formDifference(subject, "sqlite-rows", step.StoreRows, want.Rows)
	}
	return nil
}

// formStepName formats a stable identity for a form command.
//
// Takes command (casefile.FormCommand) which selects the saved form operation.
//
// Returns string which identifies the command without generated field values.
func formStepName(command casefile.FormCommand) string {
	subject := command.Op
	if command.ID != 0 {
		subject = fmt.Sprintf("%s-%d", subject, command.ID)
	}
	return subject
}
