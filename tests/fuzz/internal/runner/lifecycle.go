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
	"reflect"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// lifecycleMismatch identifies a lifecycle step mismatch.
type lifecycleMismatch struct {
	// step identifies the lifecycle step.
	step string

	// detail describes the mismatch.
	detail string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e lifecycleMismatch) Error() string { return "browser/lifecycle: " + e.step + " " + e.detail }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e lifecycleMismatch) FailureSubject() string { return "/journey/" + e.step }

// recoverPartialLifecycle records whatever lifecycle receipt was written before V8
// coverage failed, whether the journey completed or stopped at a driver failure.
//
// Takes work (string) which is the invocation-private working directory.
// Takes want (*casefile.LifecycleTrace) which holds the independently expected result.
// Takes o (*Observation) which receives the application observation.
//
// Returns error which reports an unreadable or invalid receipt or the independently
// checked journey failure, or nil when the receipt is absent, incomplete or passes.
func recoverPartialLifecycle(work string, want *casefile.LifecycleTrace, o *Observation) error {
	data, err := readBrowserReceipt(work, browserReceiptName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("partial lifecycle receipt: %w", err)
	}
	var client casefile.ClientTransition
	if err := json.Unmarshal(data, &client); err != nil {
		return fmt.Errorf("partial lifecycle receipt: %w", err)
	}
	if client.Lifecycle == nil || len(client.Before) == 0 || len(client.After) == 0 {
		return nil
	}
	o.Client = &client
	o.Counts = []string{client.Before[0].Count, client.After[0].Count}
	return checkLifecycle(client.Lifecycle, want)
}

// checkLifecycle checks the complete component navigation and cleanup journey.
//
// Takes got (*casefile.LifecycleTrace) which holds the observed result.
// Takes want (*casefile.LifecycleTrace) which holds the independently expected result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkLifecycle(got, want *casefile.LifecycleTrace) error {
	_, err := verifiedLifecycle(got, want)
	return err
}

// VerifiedLifecycleSteps counts the matching prefix even when a later browser action
// fails.
//
// Takes got (*casefile.LifecycleTrace) which holds the observed result.
// Takes want (*casefile.LifecycleTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of lifecycle states.
func VerifiedLifecycleSteps(got, want *casefile.LifecycleTrace) int {
	count, _ := verifiedLifecycle(got, want)
	return count
}

// verifiedLifecycle checks lifecycle states and counts the verified prefix.
//
// Takes got (*casefile.LifecycleTrace) which holds the observed result.
// Takes want (*casefile.LifecycleTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of lifecycle states.
// Returns error which reports the first journey mismatch or driver failure, or nil on
// success.
func verifiedLifecycle(got, want *casefile.LifecycleTrace) (int, error) {
	if got == nil && want == nil {
		return 0, nil
	}
	if got == nil || want == nil {
		return 0, lifecycleMismatch{step: "trace", detail: "missing lifecycle trace"}
	}
	if got.Input != want.Input {
		return 0, lifecycleMismatch{step: "schedule", detail: "observed a different schedule"}
	}
	for index := range got.States {
		if index >= len(want.States) {
			return index, lifecycleMismatch{step: "final", detail: "unexpected extra observation"}
		}
		if actual, state := &got.States[index], &want.States[index]; *actual != *state {
			return index, lifecycleMismatch{step: state.Step, detail: lifecycleDifference(actual, state)}
		}
	}
	recorded := len(got.States)
	if recorded < len(want.States) || got.DriverError != "" || got.FailedStep != "" {
		step := "final"
		if recorded < len(want.States) {
			step = want.States[recorded].Step
		}
		return recorded, lifecycleMismatch{step: step, detail: fmt.Sprintf("recorded %d/%d snapshots, failed step %q: %s", recorded, len(want.States), got.FailedStep, got.DriverError)}
	}
	return recorded, nil
}

// lifecycleDifference describes how an observed lifecycle state differs from its model.
// It gives each differing field first, then both complete states.
//
// Takes got (*casefile.LifecycleState) which holds the observed state.
// Takes want (*casefile.LifecycleState) which holds the expected state.
//
// Returns string which names the differing fields and holds both states.
func lifecycleDifference(got, want *casefile.LifecycleState) string {
	fields := differingFields("", reflect.ValueOf(*got), reflect.ValueOf(*want))
	return fmt.Sprintf("differs in %s; got %+v want %+v", strings.Join(fields, ", "), *got, *want)
}

// differingFields lists the fields, nested structs included, in which two values of one
// struct type differ.
//
// Takes prefix (string) which is the dotted path of the struct within the state.
// Takes got (reflect.Value) which holds the observed struct.
// Takes want (reflect.Value) which holds the expected struct.
//
// Returns []string which describes each differing field with both values.
func differingFields(prefix string, got, want reflect.Value) []string {
	var fields []string
	for field, actual := range got.Fields() {
		name := prefix + field.Name
		expected := want.FieldByIndex(field.Index)
		if actual.Kind() == reflect.Struct {
			fields = append(fields, differingFields(name+".", actual, expected)...)
			continue
		}
		if !valuesEqual(actual, expected) {
			fields = append(fields, fmt.Sprintf("%s (got %#v, want %#v)", name, actual, expected))
		}
	}
	return fields
}

// valuesEqual reports whether two field values are equal. Values that cannot be compared
// with ==, such as slices and maps, are compared in depth.
//
// Takes actual (reflect.Value) which holds the observed field.
// Takes expected (reflect.Value) which holds the expected field.
//
// Returns bool which is true when both values are equal.
func valuesEqual(actual, expected reflect.Value) bool {
	if actual.Comparable() && expected.Comparable() {
		return actual.Equal(expected)
	}
	return reflect.DeepEqual(actual.Interface(), expected.Interface())
}
