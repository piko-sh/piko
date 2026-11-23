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
	"fmt"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// eventMismatch identifies a component event mismatch.
type eventMismatch struct {
	// step identifies the event step.
	step string

	// field names the mismatched property.
	field string

	// detail describes the mismatch.
	detail string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e eventMismatch) Error() string {
	return fmt.Sprintf("events/%s: %s: %s", e.field, e.step, e.detail)
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e eventMismatch) FailureSubject() string { return "/events/" + e.step + "/" + e.field }

// checkEvents compares the complete component event journey with its model.
//
// Takes got (*casefile.EventsTrace) which holds the observed result.
// Takes want (*casefile.EventsTrace) which holds the independently expected result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkEvents(got, want *casefile.EventsTrace) error {
	_, err := verifiedEvents(got, want)
	return err
}

// VerifiedEventSteps counts the verified prefix of the component event journey.
//
// Takes got (*casefile.EventsTrace) which holds the observed result.
// Takes want (*casefile.EventsTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of event states.
func VerifiedEventSteps(got, want *casefile.EventsTrace) int {
	verified, _ := verifiedEvents(got, want)
	return verified
}

// verifiedEvents checks event states and counts the verified prefix.
//
// Takes got (*casefile.EventsTrace) which holds the observed result.
// Takes want (*casefile.EventsTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of event states.
// Returns error which reports the first journey mismatch or driver failure, or nil on
// success.
func verifiedEvents(got, want *casefile.EventsTrace) (int, error) {
	if got == nil && want == nil {
		return 0, nil
	}
	if want == nil || got == nil || got.Input != want.Input {
		return 0, eventMismatch{step: "initial", field: "recipe", detail: "missing or mismatched recipe"}
	}
	for index := range got.States {
		observed := &got.States[index]
		if index >= len(want.States) {
			return index, eventMismatch{step: "final", field: "count", detail: "extra observation"}
		}
		expected := &want.States[index]
		if field := eventStateField(observed, expected); field != "" {
			return index, eventMismatch{step: expected.Step, field: field, detail: fmt.Sprintf("got %+v, want %+v", *observed, *expected)}
		}
	}
	if recorded := len(got.States); recorded != len(want.States) || got.DriverError != "" {
		step := "final"
		if recorded < len(want.States) {
			step = want.States[recorded].Step
		}
		return recorded, eventMismatch{step: step, field: "driver", detail: fmt.Sprintf("%d/%d recorded: %s", recorded, len(want.States), got.DriverError)}
	}
	return len(got.States), nil
}

// eventStateField names the first field where an observed event state departs from the
// model.
//
// Takes observed (*casefile.EventState) which holds the observed state.
// Takes expected (*casefile.EventState) which holds the independently expected state.
//
// Returns string which names the mismatched field, or empty when the states agree.
func eventStateField(observed, expected *casefile.EventState) string {
	switch {
	case observed.Step != expected.Step:
		return "step"
	case observed.Initialisations != expected.Initialisations:
		return "module-initialisation"
	case observed.LeafSerial != expected.LeafSerial || observed.LeftSerial != expected.LeftSerial || observed.RightSerial != expected.RightSerial:
		return "shared-module-counter"
	case observed.Owner != expected.Owner || observed.SameLeaf != expected.SameLeaf:
		return "slot-owner"
	case observed.Local != expected.Local || observed.Left != expected.Left || observed.Right != expected.Right || observed.PK != expected.PK:
		return "listener-count"
	case observed.Log != expected.Log:
		return "delivery-order"
	case observed.Payload != expected.Payload:
		return "pk-event-payload"
	}
	return ""
}
