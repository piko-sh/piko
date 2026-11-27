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
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestLifecycleChecksEachCompletionOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		input := casefile.LifecycleInput{Draft: "東京", Reverse: reverse}
		want := casefile.ExpectedLifecycle(input)
		require.NoError(t, checkLifecycle(casefile.ExpectedLifecycle(input), want))
		for index, step := range want.States {
			got := casefile.ExpectedLifecycle(input)
			got.States[index].Deliveries++
			mismatch, ok := errors.AsType[lifecycleMismatch](checkLifecycle(got, want))
			if assert.True(t, ok, "missed duplicate listener at %s", step.Step) {
				assert.Equal(t, "/journey/"+step.Step, mismatch.FailureSubject())
			}
		}
		got := casefile.ExpectedLifecycle(input)
		got.States[1].SameRow = false
		assert.Error(t, checkLifecycle(got, want), "accepted replacement of a keyed row")
		got = casefile.ExpectedLifecycle(input)
		got.States[len(got.States)-1].SameDocument = false
		assert.Error(t, checkLifecycle(got, want), "accepted a hard page load during browser history navigation")
	}
}

func TestLifecycleRejectsPageScriptAndElementReuseFaults(t *testing.T) {
	editor := casefile.LifecycleInput{Draft: "note", LayoutLinks: true}
	reuse := casefile.LifecycleInput{ElementReuse: true}
	testCases := []struct {
		name   string
		step   string
		detail string
		fault  func(*casefile.LifecycleState)
		input  casefile.LifecycleInput
	}{
		{"onDisconnected missed on leave", "navigate-away", "Journey.Disconnected (got 0, want 1)",
			func(s *casefile.LifecycleState) { s.Journey.Disconnected-- }, editor},
		{"cleanup missed on leave", "link-away", "Journey.Cleaned (got 1, want 2)",
			func(s *casefile.LifecycleState) { s.Journey.Cleaned-- }, editor},
		{"onConnected missed on return", "back-and-dispatch", "Journey.Connected (got 1, want 2)",
			func(s *casefile.LifecycleState) { s.Journey.Connected-- }, editor},
		{"other page script never connects", "navigate-away", "Away.Connected (got 0, want 1)",
			func(s *casefile.LifecycleState) { s.Away.Connected = 0 }, editor},
		{"stale binding after return", "link-back-and-click", "Clicks (got 2, want 3)",
			func(s *casefile.LifecycleState) { s.Clicks-- }, editor},
		{"double binding after return", "history-forward-and-click", "Clicks (got 5, want 4)",
			func(s *casefile.LifecycleState) { s.Clicks++ }, editor},
		{"full reload instead of SPA navigation", "link-away", "SameDocument (got false, want true)",
			func(s *casefile.LifecycleState) { s.SameDocument = false }, editor},
		{"pager keeps the previous listing's configuration", "reuse-link-invoices", `Applied (got "orders,orders", want "orders,invoices")`,
			func(s *casefile.LifecycleState) { s.Applied = "orders,orders" }, reuse},
		{"pager never reconnects", "reuse-link-invoices", "Connected (got 1, want 2), Cleaned (got 0, want 1)",
			func(s *casefile.LifecycleState) { s.Connected, s.Cleaned = 1, 0 }, reuse},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			want := casefile.ExpectedLifecycle(tc.input)
			got := casefile.ExpectedLifecycle(tc.input)
			index := slices.IndexFunc(got.States, func(state casefile.LifecycleState) bool { return state.Step == tc.step })
			require.GreaterOrEqual(t, index, 0, "unknown step %s", tc.step)
			tc.fault(&got.States[index])

			err := checkLifecycle(got, want)

			mismatch, ok := errors.AsType[lifecycleMismatch](err)
			require.True(t, ok, "%v", err)
			assert.Equal(t, "/journey/"+tc.step, mismatch.FailureSubject())
			assert.Contains(t, err.Error(), "differs in "+tc.detail+";")
			assert.Equal(t, index, VerifiedLifecycleSteps(got, want))
		})
	}
}

func TestLifecyclePartialReceiptKeepsEarliestFailureAndVerifiedPrefix(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		input := casefile.LifecycleInput{Draft: "local", Reverse: reverse}
		want := casefile.ExpectedLifecycle(input)
		for index, state := range want.States {
			got := casefile.ExpectedLifecycle(input)
			got.States = got.States[:index]
			got.FailedStep, got.DriverError = state.Step, "local browser action failed"
			err := checkLifecycle(got, want)
			bad, ok := errors.AsType[lifecycleMismatch](err)
			require.True(t, ok, "%s: got %v", state.Step, err)
			assert.Equal(t, "/journey/"+state.Step, bad.FailureSubject())
			assert.Equal(t, index, VerifiedLifecycleSteps(got, want))
			assert.Contains(t, err.Error(), got.DriverError)
			if index > 0 {
				got.States[index-1].Deliveries++
				bad, ok = errors.AsType[lifecycleMismatch](checkLifecycle(got, want))
				require.True(t, ok, "earlier observer discrepancy hidden by later driver error")
				assert.Equal(t, "/journey/"+want.States[index-1].Step, bad.FailureSubject())
				assert.Equal(t, index-1, VerifiedLifecycleSteps(got, want))
			}
		}
	}
}

func TestRecoverPartialLifecycleAfterCoverageError(t *testing.T) {
	want := casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "x"})
	got := casefile.ExpectedLifecycle(want.Input)
	got.States = got.States[:3]
	got.FailedStep, got.DriverError = "cancel-navigation", "browser disconnected"
	client := casefile.ClientTransition{Before: []casefile.ClientInstance{{Count: "3"}}, After: []casefile.ClientInstance{{Count: "4"}}, Lifecycle: got}
	data, err := json.Marshal(client)
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	var o Observation
	err = recoverPartialLifecycle(work, want, &o)
	bad, ok := errors.AsType[lifecycleMismatch](err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "/journey/cancel-navigation", bad.FailureSubject())
	require.NotNil(t, o.Client)
	assert.Len(t, o.Client.Lifecycle.States, 3)
	assert.Equal(t, []string{"3", "4"}, o.Counts)
}

func TestRecoverPartialLifecycleRecordsACleanReceipt(t *testing.T) {
	want := casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "x"})
	client := casefile.ClientTransition{Before: []casefile.ClientInstance{{Count: "1"}}, After: []casefile.ClientInstance{{Count: "2"}}, Lifecycle: casefile.ExpectedLifecycle(want.Input)}
	data, err := json.Marshal(client)
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	var o Observation

	require.NoError(t, recoverPartialLifecycle(work, want, &o))

	require.NotNil(t, o.Client)
	assert.Equal(t, []string{"1", "2"}, o.Counts)
}

func TestDifferingFieldsComparesFieldsThatCannotUseEquals(t *testing.T) {
	type nested struct {
		Count int
	}
	type record struct {
		Names  []string
		Labels map[string]int
		Inner  nested
		Step   string
	}
	got := record{Names: []string{"first"}, Labels: map[string]int{"a": 1}, Inner: nested{Count: 2}, Step: "same"}
	want := record{Names: []string{"second"}, Labels: map[string]int{"a": 1}, Inner: nested{Count: 3}, Step: "same"}

	var fields []string
	require.NotPanics(t, func() { fields = differingFields("", reflect.ValueOf(got), reflect.ValueOf(want)) })

	require.Len(t, fields, 2)
	assert.Contains(t, fields[0], "Names")
	assert.Contains(t, fields[1], "Inner.Count")
}
