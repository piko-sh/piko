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
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/inventory"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

func TestSuiteSelection(t *testing.T) {
	for _, testCase := range []struct {
		suite   string
		browser string
		n       uint64
	}{
		{suite: "smoke", n: 1, browser: "sample"},
		{suite: "extended", n: extendedCases, browser: "every"},
		{suite: "soak", n: 100, browser: "sample"},
	} {
		options, _, err := parse("preview", []string{"-suite", testCase.suite, "-mode", "interp"})
		require.NoError(t, err, testCase.suite)
		assert.Equal(t, testCase.n, options.n, testCase.suite)
		assert.Equal(t, testCase.browser, options.browser, testCase.suite)
		assert.Equal(t, "all", options.target, testCase.suite)
		assert.Equal(t, testCase.suite == "soak", options.duration > 0, testCase.suite)
		list, err := selected(&options)
		require.NoError(t, err, testCase.suite)
		assert.Contains(t, list, "incremental", testCase.suite)
		assert.NotContains(t, list, "embedded-deploy", testCase.suite)
	}
	options, _, err := parse("preview", []string{"-suite", "extended", "-n", "2", "-browser", "off", "-target", "markup"})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), options.n)
	assert.Equal(t, "off", options.browser)
	assert.Equal(t, "markup", options.target)
}

func TestSuiteRejectsInvalidPresets(t *testing.T) {
	_, _, err := parse("preview", []string{"-suite", "unknown"})
	require.Error(t, err)
	_, _, err = parse("replay", []string{"-suite", "smoke"})
	require.Error(t, err)
	_, _, err = parse("campaign", []string{"-suite", "smoke", "-engine", "go"})
	require.Error(t, err)
}

func TestCompareKeysCountsMissingFeatures(t *testing.T) {
	gap := compareKeys(map[string]inventory.Frequency{"a": {}, "b": {}, "c": {}}, map[string]inventory.Frequency{"b": {}})
	assert.Equal(t, sourceGap{Shared: 1, Missing: 2, MissingKeys: []string{"a", "c"}}, gap)
}

func TestBoundedCountSaturates(t *testing.T) {
	assert.Equal(t, 7, boundedCount(7))
	assert.Equal(t, math.MaxInt, boundedCount(math.MaxUint64))
}

func TestPreviewOneDescribesPackageTarget(t *testing.T) {
	options, _, err := parse(commandPreview, []string{"-target=binder"})
	require.NoError(t, err)

	row := previewOne(&options, "binder", 3)

	assert.Equal(t, "binder", row.Target)
	assert.Equal(t, []string{"package"}, row.Modes)
	assert.Equal(t, 3, row.PreviewCases)
}

func TestPreviewDescribesEveryTargetKind(t *testing.T) {
	fakeLauncher(t)

	output, err := captureStdout(t, func() error {
		return run(t.Context(), []string{"preview", "-target=all", "-n=1", "-browser=every", "-seed=5"})
	})

	require.NoError(t, err)
	var report previewReport
	require.NoError(t, json.Unmarshal([]byte(output), &report))
	assert.Equal(t, uint64(5), report.Seed)
	assert.Equal(t, browserEvery, report.Browser)
	rows := map[string]previewTarget{}
	for _, row := range report.Targets {
		rows[row.Target] = row
	}
	assert.Equal(t, []string{"warm interpreter fixture"}, rows[targetIncremental].Modes)
	assert.Contains(t, rows["wasm"].Tools, "node")
	assert.Contains(t, rows["generated-artifacts"].Tools, "tsc")
	assert.Contains(t, rows["lsp"].Tools, "pikopls")
	assert.Contains(t, rows["database"].Tools, "local Seagull and SQLite")
	assert.Equal(t, []string{"compiled authored example"}, rows[targetExampleContracts].Modes)
	assert.Equal(t, []string{modeCompiled}, rows[targetEmbeddedDeploy].Modes)
	pages := rows["pages"]
	assert.Equal(t, []string{modeInterp, modeCompiled}, pages.Modes)
	assert.Contains(t, pages.Tools, localPipit)
	assert.Contains(t, pages.Tools, chromium)
	assert.Equal(t, 1, pages.BrowserSelected)
	assert.NotEmpty(t, pages.HTTPRequests)
	assert.NotEmpty(t, pages.Matrices)
	assert.Contains(t, rows["checkout-workflow"].Tools, "pdftotext")
	assert.Contains(t, rows["firefox-markup"].Tools, "Firefox (WebDriver BiDi)")
	assert.NotEmpty(t, rows["source-diagnostics"].Matrices)
}

func TestPreviewUsesDurationWindowAndRejectsOverflow(t *testing.T) {
	fakeLauncher(t)

	output, err := captureStdout(t, func() error {
		return run(t.Context(), []string{"preview", "-target=binder", "-duration=1m"})
	})
	require.NoError(t, err)
	var report previewReport
	require.NoError(t, json.Unmarshal([]byte(output), &report))
	require.Len(t, report.Targets, 1)
	assert.Equal(t, previewDurationCases, report.Targets[0].PreviewCases)

	err = run(t.Context(), []string{"preview", "-target=binder", "-n=2", "-index=18446744073709551615"})
	assert.ErrorIs(t, err, errPreviewOverflow)
}

func TestInventoryComparesAuthoredAndGeneratedSources(t *testing.T) {
	t.Setenv("PIKO_FUZZ_ROOT", "")
	state := t.TempDir()
	t.Setenv("PIKO_FUZZ_STATE", state)
	directory := filepath.Join(state, "saved")
	report := seededReport()
	report.LastStop = "budget reached"
	writeSeededReport(t, directory, report)

	output, err := captureStdout(t, func() error {
		return run(t.Context(), []string{"inventory", directory, "-target=pages", "-n=1"})
	})

	require.NoError(t, err)
	var printed inventoryOutput
	require.NoError(t, json.Unmarshal([]byte(output), &printed))
	assert.Equal(t, []string{"pages"}, printed.Targets)
	assert.Equal(t, uint64(1), printed.CasesPerTarget)
	assert.NotEmpty(t, printed.SourceComparison)
	require.NotNil(t, printed.Campaign)
	assert.Equal(t, "budget reached", printed.Campaign.LastStop)
	assert.Equal(t, uint64(1), printed.Campaign.CompletedPositions)
}

func TestInventoryRejectsInvalidSelection(t *testing.T) {
	fakeLauncher(t)

	assert.ErrorContains(t, run(t.Context(), []string{"inventory", "-target=binder"}), "inventory needs distinct application targets")
	assert.ErrorContains(t, run(t.Context(), []string{"inventory", "-target=pages,pages"}), "unknown or repeated")
	assert.ErrorContains(t, run(t.Context(), []string{"inventory", "-n=513"}), "inventory accepts between 1 and")
	assert.Error(t, run(t.Context(), []string{"inventory", t.TempDir()}), "a directory without a report was accepted")
	assert.Error(t, run(t.Context(), []string{"inventory", "-target=pages", "-n=1"}), "an empty repository was inventoried")
}

func TestInventoryTargetsDefaultsToEveryApplication(t *testing.T) {
	options, _, err := parse("inventory", []string{"-target=all"})
	require.NoError(t, err)

	names, err := inventoryTargets(&options)

	require.NoError(t, err)
	assert.Equal(t, targets.ApplicationNames(), names)
}
