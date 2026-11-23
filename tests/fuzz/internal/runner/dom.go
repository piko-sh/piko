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
	"slices"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// checkDOMEngine checks the observed browser engine against the saved case.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes actual (*casefile.DOMTrace) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDOMEngine(c *casefile.Case, actual *casefile.DOMTrace) error {
	if c.Expected.DOM == nil || actual == nil {
		return nil
	}
	engine := "chromium"
	if c.Target == "firefox-markup" {
		engine = "firefox"
	}
	if actual.Engine != engine || actual.Version == "" {
		return responseMismatchf(c.Expected.DOM.Path+"/browser-engine", "browser/engine: requested %s, got %q version %q", engine, actual.Engine, actual.Version)
	}
	return nil
}

// VerifyDOM checks public DOM fields against the saved operation plan.
//
// Takes plan (*casefile.DOMPlan) which holds the saved DOM operations and expected
// values.
// Takes trace (*casefile.DOMTrace) which holds the browser observations being checked.
//
// Returns int which counts the verified prefix before the first failing DOM step.
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func VerifyDOM(plan *casefile.DOMPlan, trace *casefile.DOMTrace) (int, error) {
	verified, failures := domFailures(plan, trace)
	if len(failures) > 0 {
		return verified, failures[0]
	}
	return verified, nil
}

// domFailures checks DOM fields and retains independent step failures.
//
// Takes plan (*casefile.DOMPlan) which holds the saved DOM operations and expected
// values.
// Takes trace (*casefile.DOMTrace) which holds the browser observations being checked.
//
// Returns int which counts the verified prefix before the first failing DOM step.
// Returns []error which holds independent DOM failures.
func domFailures(plan *casefile.DOMPlan, trace *casefile.DOMTrace) (int, []error) {
	if plan == nil {
		return 0, nil
	}
	first := len(plan.Steps)
	var failures []error
	fail := func(index int, subject, detail string) {
		first = min(first, index)
		failures = append(failures, responseMismatchf(plan.Path+"/"+subject, "browser/dom: %s/%s: %s", plan.Path, subject, detail))
	}
	if trace == nil {
		fail(0, "start", "missing browser receipt")
		return first, failures
	}
	for index, step := range plan.Steps {
		if index >= len(trace.States) {
			fail(index, step.ID, "missing observation: "+trace.DriverError)
			return first, failures
		}
		state := trace.States[index]
		if state.Step != step.ID || state.Path != plan.Path {
			fail(index, step.ID+"/location", fmt.Sprintf("got %s at %s", state.Step, state.Path))
			continue
		}
		checkDOMFields(index, step, state.Values, fail)
	}
	if len(trace.States) != len(plan.Steps) || trace.DriverError != "" {
		fail(len(plan.Steps), "finish", "incomplete or additional operation: "+trace.DriverError)
	}
	return first, failures
}

// checkDOMFields compares one step's observed field values with the plan.
//
// Takes index (int) which is the step's position in the plan.
// Takes step (casefile.DOMStep) which holds the step's expected fields.
// Takes values (map[string][]string) which maps observed field identifiers to values.
// Takes fail (func(int, string, string)) which records one field failure.
func checkDOMFields(index int, step casefile.DOMStep, values map[string][]string, fail func(int, string, string)) {
	for _, field := range step.Fields {
		actual, ok := values[field.ID]
		if !ok || !slices.Equal(actual, field.Want) {
			fail(index, step.ID+"/"+field.ID, fmt.Sprintf("got %q (present=%t), want %q", actual, ok, field.Want))
		}
	}
}
