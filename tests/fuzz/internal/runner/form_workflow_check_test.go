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
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestFormWorkflowDetectsIndependentFailures(t *testing.T) {
	workflow, got := formCheckFixture(t)
	require.NoError(t, verifyFormWorkflow(workflow, got))
	require.Equal(t, len(workflow.Steps), VerifiedFormSteps(workflow, got))
	tests := []struct {
		breakIt func(*casefile.FormWorkflowObservation)
		field   string
		prefix  int
	}{
		{field: "disabled-control", prefix: 0, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[0].State.DisabledControlPresent = false }},
		{field: "native-value", prefix: 1, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[1].State.Input = "one" }},
		{field: "form-data", prefix: 1, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[1].State.Entries = o.Steps[1].State.Entries[:1] }},
		{field: "invalid-events", prefix: 2, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[2].State.InvalidEvents = 0 }},
		{field: "requests", prefix: 2, breakIt: func(o *casefile.FormWorkflowObservation) {
			o.Steps[2].Requests = append(o.Steps[2].Requests, casefile.FormRequest{Method: http.MethodPost, Status: http.StatusOK})
		}},
		{field: "partial-rows", prefix: 5, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[5].State.Rows[1].Done = false }},
		{field: "action-result", prefix: 5, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[5].State.Result = nil }},
		{field: "request-fields", prefix: 3, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[3].Requests[0].InputFields = `{"label":"one"}` }},
		{field: "response-json", prefix: 3, breakIt: func(o *casefile.FormWorkflowObservation) {
			o.Steps[3].Requests[0].JSONResult = `{"id":1,"label":"one","done":"false"}`
		}},
		{field: "response-result", prefix: 3, breakIt: func(o *casefile.FormWorkflowObservation) {
			o.Steps[3].Requests[0].JSONResult = `{"id":1,"label":"wrong","done":false}`
		}},
		{field: "fresh-client-rows", prefix: 4, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[4].FreshRows = nil }},
		{field: "sqlite-rows", prefix: 4, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[4].StoreRows = o.Steps[4].StoreRows[:1] }},
		{field: "sqlite-rows", prefix: 0, breakIt: func(o *casefile.FormWorkflowObservation) { o.Steps[0].StoreRead = false }},
		{field: "sqlite-rows", prefix: len(workflow.Steps), breakIt: func(o *casefile.FormWorkflowObservation) {
			o.FinalStore = []casefile.FormRow{{ID: 3, Label: "ghost"}}
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.field, func(t *testing.T) {
			workflow, got := formCheckFixture(t)
			testCase.breakIt(got)
			err := verifyFormWorkflow(workflow, got)
			bad, ok := errors.AsType[formMismatch](err)
			require.True(t, ok, "expected %s, got %v", testCase.field, err)
			assert.Equal(t, testCase.field, bad.field)
			assert.NotEmpty(t, bad.FailureSubject())
			assert.Equal(t, testCase.prefix, VerifiedFormSteps(workflow, got))
		})
	}
}

func TestFormPartialReceipt(t *testing.T) {
	workflow, observation := formCheckFixture(t)
	observation.Steps = observation.Steps[:3]
	observation.DriverError = "failed to read database"
	observation.FailedStep = 3
	assert.Error(t, verifyFormWorkflow(workflow, observation))
	assert.Equal(t, 3, VerifiedFormSteps(workflow, observation))
}

func TestRecoverPartialFormAfterCoverageError(t *testing.T) {
	workflow, receipt := formCheckFixture(t)
	receipt.FailedStep, receipt.DriverError = 2, "browser disconnected"
	receipt.Steps = receipt.Steps[:2]
	data, err := json.Marshal(receipt)
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	var o Observation
	err = recoverPartialForm(work, workflow, &o)
	bad, ok := errors.AsType[formMismatch](err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "/workflow/invalid/driver", bad.FailureSubject())
	require.NotNil(t, o.FormWorkflow)
	assert.Equal(t, 2, VerifiedFormSteps(workflow, o.FormWorkflow))
}

func TestFormDriverFailureSubjectIgnoresWhetherTheStepWasRecorded(t *testing.T) {
	workflow := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "toggle", ID: 3}}}

	_, err := finishFormWorkflow(workflow, nil, &casefile.FormWorkflowObservation{DriverError: "browser disconnected", FailedStep: 0})

	mismatch, ok := errors.AsType[formMismatch](err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "/workflow/toggle-3/driver", mismatch.FailureSubject())
}

func TestRecoverPartialFormRecordsACleanReceipt(t *testing.T) {
	workflow, receipt := formCheckFixture(t)
	data, err := json.Marshal(receipt)
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	var o Observation

	require.NoError(t, recoverPartialForm(work, workflow, &o))

	require.NotNil(t, o.FormWorkflow)
	assert.Equal(t, len(workflow.Steps), VerifiedFormSteps(workflow, o.FormWorkflow))
}

func TestRecoverPartialFormToleratesAnAbsentReceipt(t *testing.T) {
	workflow, _ := formCheckFixture(t)
	var o Observation

	require.NoError(t, recoverPartialForm(t.TempDir(), workflow, &o))

	assert.Nil(t, o.FormWorkflow)
}

func formCheckFixture(t *testing.T) (*casefile.FormWorkflow, *casefile.FormWorkflowObservation) {
	t.Helper()
	workflow := casefile.NewFormWorkflow("one", "two", true)
	states, err := workflow.Model()
	require.NoError(t, err)
	got := &casefile.FormWorkflowObservation{Initial: casefile.EmptyFormState(), PartialPath: "/_piko/partials/task-list", FinalStore: slices.Clone(states[len(states)-1].Rows), FinalRead: true}
	for index, command := range workflow.Steps {
		state := states[index]
		requests := expectedFormRequests(command.Op, got.PartialPath)
		if len(requests) > 0 {
			wire, err := json.Marshal(state.Result)
			require.NoError(t, err)
			requests[0].JSONResult = string(wire)
			fields := map[string]any{"id": command.ID}
			if command.Op == "create" {
				fields = map[string]any{"label": command.Value, "repeat": []string{"one", "two"}}
			}
			encoded, err := json.Marshal(fields)
			require.NoError(t, err)
			requests[0].InputFields = string(encoded)
		}
		got.Steps = append(got.Steps, casefile.FormObservation{Command: command, State: state, Requests: requests, FreshRows: slices.Clone(state.Rows), StoreRows: slices.Clone(state.Rows), FreshRead: true, StoreRead: true})
	}
	return workflow, got
}
