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
	got.States = got.States[:2]
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
	assert.Len(t, o.Client.Lifecycle.States, 2)
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
