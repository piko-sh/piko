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

package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func clearRecipeVariables(t *testing.T) {
	t.Helper()
	for _, name := range recipeVariables {
		t.Setenv(name, "")
	}
}

func TestProbeRecipeSizesBudgetFromSelectedJourneys(t *testing.T) {
	clearRecipeVariables(t)
	empty, err := readProbeRecipe()
	require.NoError(t, err)
	base := empty.budget(0)
	assert.Equal(t, sessionMargin+primaryBudget, base)
	assert.Equal(t, 2*actionBudget, empty.budget(2)-base)

	t.Setenv(envCheckout, `{"Customer":"Alice","Quantity":2}`)
	t.Setenv(envEvents, `{}`)
	t.Setenv(envLifecycle, `{}`)
	t.Setenv(envPartialComponents, `{"ReloadModes":true}`)
	t.Setenv(envDOM, `{"Steps":[{"ID":"a"},{"ID":"b"}]}`)
	t.Setenv(envIntake, "true")
	t.Setenv(envJSCoverage, "/coverage.json")
	t.Setenv(envProgramPaths, `["/one","/two"]`)
	full, err := readProbeRecipe()
	require.NoError(t, err)
	require.NotNil(t, full.checkout)
	assert.Equal(t, "Alice", full.checkout.Customer)
	reloads := time.Duration(len(full.partialComponents.ReloadOrder()))
	want := base + checkoutBudget() + eventsBudget() + lifecycleBudget() + (partialStepCount+reloads)*partialStepBudget +
		domReadyWait + 2*domStepBudget + intakeBudget() + coverageBudget + 2*routeBudget
	assert.Equal(t, want, full.budget(0))
}

func TestProbeRecipeRejectsMalformedVariables(t *testing.T) {
	for _, name := range []string{envCheckout, envEvents, envLifecycle, envPartialComponents, envDOM} {
		t.Run(name, func(t *testing.T) {
			clearRecipeVariables(t)
			t.Setenv(name, "{")
			_, err := readProbeRecipe()
			require.Error(t, err)
			assert.Contains(t, err.Error(), name)
		})
	}
	clearRecipeVariables(t)
	t.Setenv(envProgramPaths, "{")
	_, err := readProbeRecipe()
	require.ErrorIs(t, err, errInvalidProgramPaths)
}

func TestParseActionsBoundsRecipe(t *testing.T) {
	actions, err := parseActions(`[{"Op":"set-input","Value":"x"}]`)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	_, err = parseActions(strings.Repeat(" ", maxActionRecipeBytes+1))
	require.Error(t, err)
	_, err = parseActions("[" + strings.TrimSuffix(strings.Repeat("{},", maxActions+1), ",") + "]")
	require.Error(t, err)
	_, err = parseActions("{")
	require.Error(t, err)
}

func TestInteractionSelectsBrowserAction(t *testing.T) {
	for _, operation := range []casefile.ClientOp{casefile.SetInput, casefile.SetLabel, casefile.ReverseRows, casefile.RemoveRow, casefile.ModifiedClick, casefile.ResetControls, casefile.ParentClick, casefile.ToggleCheck, casefile.ChooseRadio, casefile.SeekTimeline} {
		drive, err := interaction("hosts", casefile.ClientAction{Op: operation, Value: "first"}, nil)
		require.NoError(t, err, string(operation))
		assert.NotNil(t, drive, string(operation))
	}
	_, err := interaction("hosts", casefile.ClientAction{Op: casefile.SelectOption, Value: "middle"}, nil)
	require.Error(t, err)
	_, err = interaction("hosts", casefile.ClientAction{Op: "unknown"}, nil)
	require.Error(t, err)
}

func TestExitCodeMarksRuntimeCoverageFailures(t *testing.T) {
	assert.Equal(t, casefile.RuntimeCoverageExitCode, exitCode(fmt.Errorf("collecting %w: %w", errRuntimeCoverage, errors.New("profiler gone"))))
	assert.Equal(t, 1, exitCode(errors.New("navigation failed")))
}

func TestExecuteRejectsUsage(t *testing.T) {
	assert.Equal(t, usageExitCode, execute(t.Context(), []string{"http://127.0.0.1"}))
	assert.Equal(t, usageExitCode, execute(t.Context(), []string{"http://127.0.0.1", "x-tag", "result.json", "{"}))
}

func TestReportSelectsExitStatus(t *testing.T) {
	assert.Zero(t, report(nil, 3))
	assert.Equal(t, 3, report(errors.New("failed"), 3))
}

func TestWriteProbeReceiptBoundsSize(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "receipt.json")
	require.NoError(t, writeProbeReceipt(path, &casefile.ClientTransition{}))
	assert.FileExists(t, path)

	oversized := filepath.Join(directory, "oversized.json")
	err := writeProbeReceipt(oversized, &casefile.ClientTransition{PrimaryBrowserVersion: strings.Repeat("x", casefile.MaxBrowserReceiptBytes)})
	require.ErrorContains(t, err, "browser-result exceeds")
	assert.NoFileExists(t, oversized)
}

func TestNewProbeScriptsQuotesTag(t *testing.T) {
	scripts := newProbeScripts(`x-"tag"`)

	assert.Equal(t, `"x-\"tag\""`, scripts.tag)
	assert.Contains(t, scripts.hosts, `document.querySelectorAll("x-\"tag\"")`)
	assert.True(t, strings.HasPrefix(scripts.snapshot, scripts.hosts))
}

func TestExecuteRoutesFormWorkflowMode(t *testing.T) {
	result := filepath.Join(t.TempDir(), "workflow.json")

	assert.Equal(t, 1, execute(t.Context(), []string{"http://127.0.0.1", "--form-workflow", result, "{"}))
	assert.NoFileExists(t, result)
}

func TestRunActionsRejectsOutOfBoundsAndUnknownActions(t *testing.T) {
	scripts := newProbeScripts("x-counter")
	var empty casefile.ClientTransition
	err := runActions(t.Context(), nil, &scripts, []casefile.ClientAction{{Op: casefile.SetInput}}, &empty)
	require.ErrorContains(t, err, "interaction 0 outside recipe bounds")

	one := casefile.ClientTransition{Before: []casefile.ClientInstance{{Count: "1"}}}
	err = runActions(t.Context(), nil, &scripts, []casefile.ClientAction{{Op: casefile.SetInput, Value: strings.Repeat("x", maxActionValueBytes+1)}}, &one)
	require.ErrorContains(t, err, "outside recipe bounds")
	err = runActions(t.Context(), nil, &scripts, []casefile.ClientAction{{Op: "unknown"}}, &one)
	require.ErrorContains(t, err, `unknown client operation "unknown"`)
	assert.Empty(t, one.Steps)
}
