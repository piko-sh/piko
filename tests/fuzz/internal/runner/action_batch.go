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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// collectActionName is the public name of the collecting fuzz action.
	collectActionName = "fuzz.Collect"
)

// actionBatch records a batch dispatch and its individual results.
type actionBatch struct {
	// Results holds individual batch results.
	Results []batchResult

	// Status is the HTTP response status.
	Status int

	// Parallel records whether the batch ran in parallel.
	Parallel bool

	// Success records the batch result flag.
	Success bool
}

// batchResult records one action in a batch response.
type batchResult struct {
	// Data holds the typed action result.
	Data *casefile.ActionValue

	// Errors holds public field-validation errors.
	Errors map[string]string

	// Name is the dispatched action name.
	Name string

	// Code is the public error code, empty on success.
	Code string

	// Status is the HTTP response status.
	Status int
}

// batchEnvelope is the decoded public batch response.
type batchEnvelope struct {
	// Success is the batch success flag, nil when the response omitted it.
	Success *bool

	// Results holds individual batch results.
	Results []batchResult
}

// observeBatch submits a sequential or parallel action batch.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes token (string) which is the action CSRF token.
// Takes ephemeral (string) which is the form's ephemeral CSRF token.
// Takes in (casefile.ActionValue) which supplies the action label and quantity.
// Takes parallel (bool) which selects parallel batch execution.
//
// Returns actionBatch which records the batch status and individual action results.
// Returns error which reports batch-request or response-decoding failure, or nil on
// success.
func observeBatch(ctx context.Context, client *http.Client, base, token, ephemeral string, in casefile.ActionValue, parallel bool) (actionBatch, error) {
	out := actionBatch{Results: nil, Status: 0, Parallel: parallel, Success: false}
	item := func(label, operation string, quantity int) map[string]any {
		return map[string]any{"name": collectActionName, "args": map[string]any{"Label": label, "Operation": operation, "Quantity": quantity}}
	}
	actions := []map[string]any{
		item(in.Label+"-first", "", in.Quantity),
		item(in.Label+"-invalid", "validate", 0),
		item(in.Label+"-last", "", in.Quantity+2),
	}
	body, err := json.Marshal(map[string]any{"actions": actions, "parallel": parallel, "_csrf_ephemeral_token": ephemeral})
	if err != nil {
		return out, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/_batch", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, token)
	response, err := client.Do(request)
	if err != nil {
		return out, err
	}
	out.Status = response.StatusCode
	var wire batchEnvelope
	if err = decodeAction(response, &wire); err != nil {
		return out, fmt.Errorf("action/batch-json: %w", err)
	}
	if wire.Success == nil {
		return out, fmt.Errorf("action/batch-success: no success flag (HTTP %d)", out.Status)
	}
	out.Success, out.Results = *wire.Success, wire.Results
	return out, nil
}

// expectedBatch models batch results for the action contract.
//
// Takes want (*casefile.ActionContract) which holds the saved action contract.
// Takes parallel (bool) which selects parallel batch execution.
//
// Returns actionBatch which holds the independent batch status and per-action results.
func expectedBatch(want *casefile.ActionContract, parallel bool) actionBatch {
	first := casefile.ActionValue{Label: want.Input.Label + "-first", Quantity: want.Result.Quantity}
	last := casefile.ActionValue{Label: want.Input.Label + "-last", Quantity: want.Result.Quantity + 2}
	return actionBatch{Parallel: parallel, Success: false, Status: http.StatusOK, Results: []batchResult{
		{Name: collectActionName, Status: http.StatusOK, Data: &first, Code: "", Errors: nil},
		{Name: collectActionName, Status: http.StatusUnprocessableEntity, Code: "VALIDATION_FAILED", Errors: map[string]string{"Quantity": "must be positive"}, Data: nil},
		{Name: collectActionName, Status: http.StatusOK, Data: &last, Code: "", Errors: nil},
	}}
}

// checkBatches checks observed batches against the action contract.
//
// Takes want (*casefile.ActionContract) which holds the independently expected result.
// Takes got ([]actionBatch) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkBatches(want *casefile.ActionContract, got []actionBatch) error {
	if len(got) != 2 {
		return fmt.Errorf("action/batch-count: got %d batches, want serial and parallel", len(got))
	}
	for batchIndex, parallel := range []bool{false, true} {
		expected := expectedBatch(want, parallel)
		observed := got[batchIndex]
		if observed.Status != expected.Status || observed.Parallel != expected.Parallel || observed.Success != expected.Success || len(observed.Results) != len(expected.Results) {
			return fmt.Errorf("action/batch-envelope: parallel %t got %+v", parallel, observed)
		}
		for entryIndex, entry := range expected.Results {
			if !reflect.DeepEqual(entry, observed.Results[entryIndex]) {
				return fmt.Errorf("action/batch-entry: parallel %t index %d got %+v want %+v", parallel, entryIndex, observed.Results[entryIndex], entry)
			}
		}
	}
	return nil
}
