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

package pikotest_domain

import (
	"testing"

	"piko.sh/piko/internal/pikotest/pikotest_dto"
)

// ActionResultView wraps a pikotest_dto.ActionResult and provides assertion
// methods for testing server action outcomes.
type ActionResultView struct {
	// tb is the testing interface for reporting failures and logging.
	tb testing.TB

	// result holds the raw action result data from the DTO layer.
	result *pikotest_dto.ActionResult
}

// AssertSuccess asserts that the action completed without error.
func (v *ActionResultView) AssertSuccess() {
	v.tb.Helper()
	if v.result.Err != nil {
		v.tb.Fatalf("expected action to succeed, got error: %v", v.result.Err)
	}
}

// AssertError asserts that the action returned an error.
func (v *ActionResultView) AssertError() {
	v.tb.Helper()
	if v.result.Err == nil {
		v.tb.Fatal("expected action to return an error, but it succeeded")
	}
}

// AssertErrorContains asserts that the action returned an error containing
// the given substring.
//
// Takes substr (string) which is the expected substring within the error
// message.
func (v *ActionResultView) AssertErrorContains(substr string) {
	v.tb.Helper()
	if v.result.Err == nil {
		v.tb.Fatalf("expected action to return an error containing %q, but it succeeded", substr)
	}
	if got := v.result.Err.Error(); !containsSubstring(got, substr) {
		v.tb.Fatalf("expected error to contain %q, got %q", substr, got)
	}
}

// AssertHelper asserts that the response includes a helper call with the
// given name.
//
// Takes name (string) which is the helper name to look for.
func (v *ActionResultView) AssertHelper(name string) {
	v.tb.Helper()
	for _, h := range v.result.Response.Helpers {
		if h.Name == name {
			return
		}
	}
	v.tb.Fatalf("expected helper %q in response, got helpers: %v", name, v.result.Response.Helpers)
}

// AssertNoHelpers asserts that the response has no helper calls.
func (v *ActionResultView) AssertNoHelpers() {
	v.tb.Helper()
	if len(v.result.Response.Helpers) > 0 {
		v.tb.Fatalf("expected no helpers, got %d: %v", len(v.result.Response.Helpers), v.result.Response.Helpers)
	}
}

// Data returns the action's return data for custom assertions.
//
// Returns any which holds the raw response data from the action.
func (v *ActionResultView) Data() any {
	return v.result.Response.Data
}

// Err returns the error from the action invocation, or nil if it succeeded.
//
// Returns error which is the action's error result.
func (v *ActionResultView) Err() error {
	return v.result.Err
}

// Result returns the underlying ActionResult DTO for advanced assertions.
//
// Returns *pikotest_dto.ActionResult which contains the raw response and
// error.
func (v *ActionResultView) Result() *pikotest_dto.ActionResult {
	return v.result
}

// newActionResultView creates a new ActionResultView from a DTO ActionResult.
//
// Takes tb (testing.TB) which provides the test context for assertions.
// Takes result (*pikotest_dto.ActionResult) which holds the action outcome.
//
// Returns *ActionResultView ready for assertions.
func newActionResultView(tb testing.TB, result *pikotest_dto.ActionResult) *ActionResultView {
	return &ActionResultView{
		tb:     tb,
		result: result,
	}
}
