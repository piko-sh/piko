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

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// partialStepBack is the partial-component step that navigates back.
	partialStepBack = "back"
)

// recordMismatch identifies a partial-component field mismatch.
type recordMismatch struct {
	// step identifies the partial-component step.
	step string

	// field names the mismatched property.
	field string

	// detail describes the mismatch.
	detail string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e recordMismatch) Error() string {
	return fmt.Sprintf("partial-components/%s/%s: %s", e.step, e.field, e.detail)
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e recordMismatch) FailureSubject() string {
	return "/partial-components/" + e.step + "/" + e.field
}

// VerifiedPartialComponentSteps counts the verified prefix of the partial-component
// journey.
//
// Takes got (*casefile.PartialComponentsTrace) which holds the observed result.
// Takes want (*casefile.PartialComponentsTrace) which holds the independently expected
// result.
//
// Returns int which counts the matching prefix of partial-component states.
func VerifiedPartialComponentSteps(got, want *casefile.PartialComponentsTrace) int {
	verified, _ := checkPartialComponents(got, want)
	return verified
}

// checkPartialComponents checks rendered rows, identities and reload-mode state.
//
// Takes got (*casefile.PartialComponentsTrace) which holds the observed result.
// Takes want (*casefile.PartialComponentsTrace) which holds the independently expected
// result.
//
// Returns int which counts the matching prefix of partial-component states.
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkPartialComponents(got, want *casefile.PartialComponentsTrace) (int, error) {
	if want == nil && got == nil {
		return 0, nil
	}
	if want == nil || got == nil || want.Input != got.Input {
		return 0, recordMismatch{step: "mount", field: "recipe", detail: "missing or mismatched trace"}
	}
	for index := range got.States {
		if index >= len(want.States) {
			return index, recordMismatch{step: "final", field: "count", detail: "extra state"}
		}
		observed, expected := &got.States[index], &want.States[index]
		if field := partialStateField(observed, expected); field != "" {
			return index, recordMismatch{step: expected.Step, field: field, detail: fmt.Sprintf("got %+v, want %+v", *observed, *expected)}
		}
	}
	recorded := len(got.States)
	if recorded != len(want.States) || got.DriverError != "" {
		step := "final"
		if recorded < len(want.States) {
			step = want.States[recorded].Step
		}
		return recorded, recordMismatch{step: step, field: "driver", detail: got.DriverError}
	}
	return recorded, nil
}

// recoverPartialComponents records whatever partial-component receipt was written before
// V8 coverage failed, whether the journey completed or stopped at a driver failure.
//
// Takes work (string) which is the invocation-private working directory.
// Takes want (*casefile.PartialComponentsTrace) which holds the independently expected
// result.
// Takes o (*Observation) which receives the application observation.
//
// Returns error which reports an unreadable or invalid receipt or the independently
// checked journey failure, or nil when the receipt is absent or passes.
func recoverPartialComponents(work string, want *casefile.PartialComponentsTrace, o *Observation) error {
	data, err := readBrowserReceipt(work, browserReceiptName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("partial components receipt: %w", err)
	}
	var transition casefile.ClientTransition
	if err := json.Unmarshal(data, &transition); err != nil {
		return fmt.Errorf("partial components receipt: %w", err)
	}
	o.Client = &transition
	_, err = checkPartialComponents(transition.PartialComponents, want)
	return err
}

// partialStateField names the first field where an observed partial-component state
// departs from the model.
//
// Identity, action and local state are not compared after a back navigation.
//
// Takes observed (*casefile.PartialComponentState) which holds the observed state.
// Takes expected (*casefile.PartialComponentState) which holds the expected state.
//
// Returns string which names the mismatched field, or empty when the states agree.
func partialStateField(observed, expected *casefile.PartialComponentState) string {
	back := observed.Step == partialStepBack
	switch {
	case observed.Step != expected.Step:
		return "step"
	case observed.Path != expected.Path:
		return "destination"
	case observed.Locale != expected.Locale:
		return "locale"
	}
	if field := compareReload(observed.Reload, expected.Reload); field != "" {
		return field
	}
	switch {
	case observed.Revision != expected.Revision:
		return "service-revision"
	case !back && observed.ActionRevision != expected.ActionRevision:
		return "action-result"
	case observed.Detail != expected.Detail:
		return "detail"
	case !back && observed.Log != expected.Log:
		return "recipient"
	case len(observed.Rows) != len(expected.Rows):
		return "row-count"
	}
	for rowIndex := range observed.Rows {
		if field := partialRowField(&observed.Rows[rowIndex], &expected.Rows[rowIndex], back); field != "" {
			return field
		}
	}
	return ""
}

// partialRowField names the first field where an observed partial-component row departs
// from the model.
//
// Takes row (*casefile.PartialComponentRow) which holds the observed row.
// Takes expected (*casefile.PartialComponentRow) which holds the expected row.
// Takes back (bool) which reports a back navigation, where identity and local state are
// not compared.
//
// Returns string which names the mismatched field, or empty when the rows agree.
func partialRowField(row, expected *casefile.PartialComponentRow, back bool) string {
	switch {
	case row.Key != expected.Key:
		return "row-order"
	case row.Label != expected.Label || row.Score != expected.Score:
		return "host-props"
	case row.Shadow != expected.Shadow || row.ShadowScore != expected.ShadowScore:
		return "shadow-props"
	case row.Caption != expected.Caption:
		return "slot"
	case row.Link != expected.Link:
		return "link"
	case row.Name != expected.Name:
		return "link-name"
	case row.Icon != expected.Icon:
		return "svg"
	case !back && (row.HostIdentity != expected.HostIdentity || row.RowIdentity != expected.RowIdentity):
		return "identity"
	case !back && row.Count != expected.Count:
		return "local-state"
	}
	return ""
}
