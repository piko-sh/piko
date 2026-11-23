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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestPartialComponentObservationsRejectIndependentCorruption(t *testing.T) {
	input := casefile.PartialComponentsInput{Labels: [2]string{"same", "same"}, Edited: "<new>", Query: "a & b", Score: 3, Locale: "fr"}
	want := casefile.ExpectedPartialComponents(input)
	for _, testCase := range []struct {
		change func(*casefile.PartialComponentState)
		name   string
	}{
		{name: "host-props", change: func(state *casefile.PartialComponentState) { state.Rows[0].Score++ }},
		{name: "shadow-props", change: func(state *casefile.PartialComponentState) { state.Rows[0].Shadow = "same" }},
		{name: "slot", change: func(state *casefile.PartialComponentState) { state.Rows[0].Caption = state.Rows[1].Caption }},
		{name: "identity", change: func(state *casefile.PartialComponentState) { state.Rows[0].HostIdentity = state.Rows[1].HostIdentity }},
		{name: "recipient", change: func(state *casefile.PartialComponentState) {
			state.Log = strings.ReplaceAll(state.Log, "amber=", "blue=")
		}},
		{name: "local-state", change: func(state *casefile.PartialComponentState) { state.Rows[1].Count++ }},
		{name: "link", change: func(state *casefile.PartialComponentState) { state.Rows[0].Link = state.Rows[1].Link }},
		{name: "link-name", change: func(state *casefile.PartialComponentState) { state.Rows[0].Name = state.Rows[1].Name }},
		{name: "svg", change: func(state *casefile.PartialComponentState) { state.Rows[0].Icon = "missing" }},
		{name: "locale", change: func(state *casefile.PartialComponentState) { state.Locale = "en" }},
		{name: "action-result", change: func(state *casefile.PartialComponentState) { state.ActionRevision = 0 }},
		{name: "service-revision", change: func(state *casefile.PartialComponentState) { state.Revision = 0 }},
		{name: "row-order", change: func(state *casefile.PartialComponentState) {
			state.Rows[0], state.Rows[1] = state.Rows[1], state.Rows[0]
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := casefile.ExpectedPartialComponents(input)
			testCase.change(&got.States[2])
			checked, err := checkPartialComponents(got, want)
			assert.Equal(t, 2, checked)
			assertRecordSubject(t, err, "/partial-components/edit/"+testCase.name)
		})
	}
	got := casefile.ExpectedPartialComponents(input)
	got.States = got.States[:4]
	got.DriverError = "browser closed"
	checked, err := checkPartialComponents(got, want)
	assert.Equal(t, 4, checked)
	assertRecordSubject(t, err, "/partial-components/reorder/driver")
}

func TestPartialReloadModesDistinguishOwnership(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		input := casefile.PartialComponentsInput{Edited: "draft", ReloadModes: true, ReverseReload: reverse}
		want := casefile.ExpectedPartialComponents(input)
		checked, err := checkPartialComponents(want, want)
		require.NoError(t, err, "clean control")
		require.Equal(t, 16, checked, "clean control")
		for modeIndex, mode := range input.ReloadOrder() {
			index := 12 + modeIndex
			for _, mutation := range []struct {
				alter func(*casefile.PartialReloadState)
				field string
			}{
				{field: "reload-result", alter: func(state *casefile.PartialReloadState) { state.Result = "stale" }},
				{field: "root-attribute", alter: func(state *casefile.PartialReloadState) { state.RootStage = "wrong" }},
				{field: "child-content", alter: func(state *casefile.PartialReloadState) { state.ChildStage = "wrong" }},
				{field: "child-content", alter: func(state *casefile.PartialReloadState) { state.Default = state.Draft }},
				{field: "root-ownership", alter: func(state *casefile.PartialReloadState) { state.LiveOnly = "wrong" }},
				{field: "root-ownership", alter: func(state *casefile.PartialReloadState) { state.Preserved = state.RootStage }},
				{field: "owner-identity", alter: func(state *casefile.PartialReloadState) { state.InputRetained = false }},
				{field: "owner-identity", alter: func(state *casefile.PartialReloadState) { state.RootRetained = false }},
				{field: "owner-identity", alter: func(state *casefile.PartialReloadState) { state.Slotted = false }},
				{field: "focused-draft", alter: func(state *casefile.PartialReloadState) { state.Focused = false }},
				{field: "focused-draft", alter: func(state *casefile.PartialReloadState) { state.Selection = "0:0:none" }},
				{field: "focused-draft", alter: func(state *casefile.PartialReloadState) { state.Draft = state.Default }},
				{field: "sibling", alter: func(state *casefile.PartialReloadState) { state.SiblingUntouched = false }},
				{field: "sibling", alter: func(state *casefile.PartialReloadState) { state.SiblingRetained = false }},
			} {
				got := casefile.ExpectedPartialComponents(input)
				mutation.alter(got.States[index].Reload)
				checked, err := checkPartialComponents(got, want)
				assert.Equal(t, index, checked, "%s: %s corrupted", mode, mutation.field)
				assertRecordSubject(t, err, "/partial-components/reload-"+mode+"/"+mutation.field)
			}
		}
	}
}

func assertRecordSubject(t *testing.T, err error, want string) {
	t.Helper()
	mismatch, ok := errors.AsType[recordMismatch](err)
	if assert.True(t, ok, "unexpected failure %v", err) {
		assert.Equal(t, want, mismatch.FailureSubject())
	}
}

func TestRecoverPartialComponentsFollowsTheSharedPolicy(t *testing.T) {
	want := casefile.ExpectedPartialComponents(casefile.PartialComponentsInput{Labels: [2]string{"a", "b"}, Edited: "c", Query: "d", Score: 1, Locale: "en"})

	var absent Observation
	require.NoError(t, recoverPartialComponents(t.TempDir(), want, &absent))
	assert.Nil(t, absent.Client)

	data, err := json.Marshal(casefile.ClientTransition{PartialComponents: casefile.ExpectedPartialComponents(want.Input)})
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	var clean Observation
	require.NoError(t, recoverPartialComponents(work, want, &clean))
	require.NotNil(t, clean.Client)

	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), []byte("{"), 0600))
	assert.ErrorContains(t, recoverPartialComponents(work, want, &Observation{}), "partial components receipt")
}

func TestVerifiedPartialComponentStepsCreditsTheModelledJourney(t *testing.T) {
	want := casefile.ExpectedPartialComponents(casefile.PartialComponentsInput{Labels: [2]string{"a", "b"}, Edited: "c"})
	got := casefile.ExpectedPartialComponents(want.Input)

	assert.Equal(t, len(want.States), VerifiedPartialComponentSteps(got, want))
	got.States = got.States[:3]
	assert.Equal(t, 3, VerifiedPartialComponentSteps(got, want))
}
