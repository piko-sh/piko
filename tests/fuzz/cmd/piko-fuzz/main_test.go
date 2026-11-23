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
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/tests/fuzz/internal/campaign"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
)

func TestPackageSelectionRejectsUnrelatedBrowserAndMode(t *testing.T) {
	for _, argument := range []string{"-mode=both", "-browser=every"} {
		options, _, err := parse("campaign", []string{"-target=binder", argument})
		require.NoError(t, err)
		_, err = selected(&options)
		require.Error(t, err, argument)
	}
}

func TestNativeTargetsUseNativeRegistry(t *testing.T) {
	t.Setenv("PIKO_FUZZ_ROOT", t.TempDir())
	t.Setenv("PIKO_FUZZ_MEMORY_BYTES", "")
	err := run(t.Context(), []string{"campaign", "-engine", "go", "-target", "raw-sfc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign budget")
}

func TestLatestSkipsUnpublishedReceipt(t *testing.T) {
	state := t.TempDir()
	root := filepath.Join(state, "campaigns")
	for _, name := range []string{"a", "b"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name), 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "a", "native.json"), []byte("{}"), 0o600))
	got, err := latest(state)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "a"), got)
	_, err = latest(t.TempDir())
	require.Error(t, err)
}

func TestCoverageWindowAndTargetBounds(t *testing.T) {
	for _, arguments := range [][]string{{"-n=513"}, {"-target=binder"}, {"-n=2", "-index=18446744073709551615"}} {
		require.Error(t, run(t.Context(), append([]string{"coverage"}, arguments...)), "unbounded or unrelated inventory %v", arguments)
	}
}

func TestCollectionTargetAcceptsHostInstrumentation(t *testing.T) {
	t.Setenv("PIKO_FUZZ_ROOT", t.TempDir())
	t.Setenv("PIKO_FUZZ_MEMORY_BYTES", "")
	for _, mode := range []string{"compiled", "interp", "both"} {
		err := run(t.Context(), []string{"campaign", "-target", "collection-routes", "-mode", mode, "-browser", "off", "-runtime-coverage"})
		require.Error(t, err, "mode %s instrumentation selection", mode)
		assert.Contains(t, err.Error(), "campaign budget", "mode %s instrumentation selection", mode)
	}
}

func TestCompiledOnlyExcludesWarmInterpreter(t *testing.T) {
	for _, mode := range []string{"interp", "both", "compiled"} {
		options, _, err := parse("campaign", []string{"-target=all", "-mode=" + mode})
		require.NoError(t, err)
		list, err := selected(&options)
		require.NoError(t, err, mode)
		assert.Contains(t, list, "graph-rebuild", mode)
		assert.Equal(t, mode != "compiled", slices.Contains(list, "incremental"), mode)
		options.target = "incremental"
		_, err = selected(&options)
		assert.Equal(t, mode == "compiled", err != nil, mode)
	}
}

func TestAuthoredExamplesRespectHostAndBrowserSelection(t *testing.T) {
	for _, testCase := range []struct {
		mode     string
		browser  string
		included bool
	}{
		{mode: "interp", browser: "every", included: false},
		{mode: "both", browser: "off", included: false},
		{mode: "compiled", browser: "off", included: false},
		{mode: "compiled", browser: "every", included: true},
		{mode: "both", browser: "sample", included: true},
	} {
		options, _, err := parse("campaign", []string{"-target=all", "-mode=" + testCase.mode, "-browser=" + testCase.browser})
		require.NoError(t, err)
		list, err := selected(&options)
		require.NoError(t, err)
		assert.Equal(t, testCase.included, slices.Contains(list, "example-contracts"), "%+v", testCase)
		options.target = "example-contracts"
		_, err = selected(&options)
		assert.Equal(t, testCase.included, err == nil, "%+v", testCase)
	}
}

func TestSelectedRejectsUnknownAndRepeatedTargets(t *testing.T) {
	for _, target := range []string{"no-such-target", "pages,pages"} {
		options, _, err := parse("campaign", []string{"-target=" + target})
		require.NoError(t, err)
		_, err = selected(&options)
		require.Error(t, err, target)
	}
}

func TestPrepareToolsDirectoryRemovesStaleOwners(t *testing.T) {
	state := t.TempDir()
	root := filepath.Join(state, toolsDirectoryName)
	stale := []string{"999999999-dead", "not-a-process", "plain", strconv.Itoa(os.Getpid()) + "-reused"}
	for _, name := range stale {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name), 0o700))
	}
	directory, err := prepareToolsDirectory(state)
	require.NoError(t, err)
	assert.Equal(t, root, filepath.Dir(directory))
	assert.True(t, strings.HasPrefix(filepath.Base(directory), strconv.Itoa(os.Getpid())+"-"))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, filepath.Base(directory), entries[0].Name())
}

func TestToolsOwnerRunningRecognisesLiveParent(t *testing.T) {
	assert.True(t, toolsOwnerRunning(strconv.Itoa(os.Getppid())+"-x"))
	assert.False(t, toolsOwnerRunning(strconv.Itoa(os.Getpid())+"-x"))
	assert.False(t, toolsOwnerRunning("-1-x"))
	assert.False(t, toolsOwnerRunning("nodash"))
}

func TestReplayModeSupported(t *testing.T) {
	for _, testCase := range []struct {
		target  string
		mode    string
		wantErr string
	}{
		{target: "pages", mode: modeInterp, wantErr: ""},
		{target: targetIncremental, mode: modeInterp, wantErr: ""},
		{target: targetExampleContracts, mode: modeCompiled, wantErr: ""},
		{target: targetExampleContracts, mode: modeInterp, wantErr: exampleMessage},
		{target: targetEmbeddedDeploy, mode: modeInterp, wantErr: deploymentMessage},
		{target: targetIncremental, mode: modeCompiled, wantErr: warmInterpreterMessage},
		{target: "binder", mode: modeBoth, wantErr: "-mode requires an application target or incremental"},
	} {
		t.Run(testCase.target+"/"+testCase.mode, func(t *testing.T) {
			err := replayModeSupported(testCase.target, testCase.mode)
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, testCase.wantErr)
		})
	}
}

func TestOverrideReplayOptionsAppliesExplicitFlags(t *testing.T) {
	options, _, err := parse(commandReplay, []string{"-mode=interp", "-browser=off"})
	require.NoError(t, err)
	var savedOptions campaign.Options

	require.NoError(t, overrideReplayOptions(&casefile.Case{Target: "pages"}, &options, &savedOptions))

	assert.Equal(t, modeInterp, savedOptions.Mode)
	assert.Equal(t, browserOff, savedOptions.Browser)
}

func TestOverrideReplayOptionsRejectsUnsupportedFlags(t *testing.T) {
	for _, testCase := range []struct {
		target    string
		arguments []string
	}{
		{target: targetExampleContracts, arguments: []string{"-browser=off"}},
		{target: "binder", arguments: []string{"-browser=every"}},
		{target: targetIncremental, arguments: []string{"-mode=compiled"}},
	} {
		options, _, err := parse(commandReplay, testCase.arguments)
		require.NoError(t, err)
		var savedOptions campaign.Options

		err = overrideReplayOptions(&casefile.Case{Target: testCase.target}, &options, &savedOptions)

		require.Error(t, err, "%s %v", testCase.target, testCase.arguments)
		assert.Equal(t, campaign.Options{}, savedOptions)
	}
}

func TestValidateNativeOptions(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		arguments []string
		wantErr   string
	}{
		{name: "size", arguments: []string{"-fuzzer=binder", "-size=tiny"}, wantErr: "-size applies to seeded generation"},
		{name: "mode", arguments: []string{"-fuzzer=binder", "-mode=interp"}, wantErr: "native Go fuzz targets run independently of app mode and browser"},
		{name: "seeded engine", arguments: []string{"-fuzzer=binder", "-engine=seeded"}, wantErr: "-fuzzer selects the Go engine"},
		{name: "conflicting target", arguments: []string{"-fuzzer=binder", "-target=cache"}, wantErr: "-target and -fuzzer select different Go fuzz targets"},
		{name: "unregistered", arguments: []string{"-fuzzer=no-such-fuzzer"}, wantErr: "choose a registered -fuzzer from ./hack/fuzz/fuzz.sh help"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			options, _, err := parse(commandCampaign, testCase.arguments)
			require.NoError(t, err)
			require.EqualError(t, validateNativeOptions(&options), testCase.wantErr)
		})
	}
}

func TestValidateNativeOptionsDefaultsFuzzerToTarget(t *testing.T) {
	options, _, err := parse(commandCampaign, []string{"-engine=go", "-target=binder"})
	require.NoError(t, err)

	require.NoError(t, validateNativeOptions(&options))

	assert.Equal(t, "binder", options.fuzzer)
}

func TestDefaultNativeEngineSelectsGoForTimedFuzzer(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{name: "timed fuzzer", arguments: []string{"-target=binder", "-duration=1m"}, want: engineGo},
		{name: "untimed", arguments: []string{"-target=binder"}, want: engineSeeded},
		{name: "explicit engine", arguments: []string{"-target=binder", "-duration=1m", "-engine=seeded"}, want: engineSeeded},
		{name: "seeded-only target", arguments: []string{"-target=pages", "-duration=1m"}, want: engineSeeded},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			options, _, err := parse(commandCampaign, testCase.arguments)
			require.NoError(t, err)

			defaultNativeEngine(&options)

			assert.Equal(t, testCase.want, options.engine)
		})
	}
}

func TestCommandPath(t *testing.T) {
	path, err := commandPath(nil)
	require.NoError(t, err)
	assert.Empty(t, path)

	path, err = commandPath([]string{"relative"})
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(path))

	_, err = commandPath([]string{"one", "two"})
	require.Error(t, err)
}

func TestResumeNativeHandlesRecordedOutcomes(t *testing.T) {
	var harness campaign.Harness
	options, _, err := parse(commandResume, nil)
	require.NoError(t, err)

	require.NoError(t, resumeNative(t.Context(), harness, commandResume, &options, t.TempDir(), &campaign.NativeRun{Status: statusPass}))
	err = resumeNative(t.Context(), harness, commandResume, &options, t.TempDir(), &campaign.NativeRun{Status: statusFailed})
	require.ErrorContains(t, err, "native campaign recorded a failure")
	err = resumeNative(t.Context(), harness, commandReduce, &options, t.TempDir(), &campaign.NativeRun{})
	require.ErrorContains(t, err, "native Go fuzz minimises failures")
	options.runtimeCoverage = true
	err = resumeNative(t.Context(), harness, commandResume, &options, t.TempDir(), &campaign.NativeRun{})
	require.ErrorContains(t, err, "runtime coverage uses application hosts")
}

func TestCreatePrivateDirectoryConfinesToState(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")

	require.NoError(t, createPrivateDirectory(state, "campaigns"))

	info, err := os.Stat(filepath.Join(state, "campaigns"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(privateDirectoryMode), info.Mode().Perm())
	require.Error(t, createPrivateDirectory(state, "../escape"))
	assert.NoDirExists(t, filepath.Join(filepath.Dir(state), "escape"))
}

func TestReplayCoverageHarnessRejectsPackageTargets(t *testing.T) {
	var harness campaign.Harness
	harness.State = t.TempDir()

	_, err := replayCoverageHarness(harness, &casefile.Case{Target: "binder"})

	require.EqualError(t, err, "runtime coverage uses application hosts")
}

func TestGenerateCaseWritesOneTarget(t *testing.T) {
	out := filepath.Join(t.TempDir(), "case")
	options, _, err := parse(commandGenerate, []string{"-target=binder", "-out=" + out})
	require.NoError(t, err)

	require.NoError(t, generateCase(&options, []string{"binder"}))

	saved, err := casefile.Read(out)
	require.NoError(t, err)
	assert.Equal(t, "binder", saved.Target)
	require.EqualError(t, generateCase(&options, []string{"binder", "cache"}), "gen takes one target")
}

func TestStatusReportRequiresSavedCampaign(t *testing.T) {
	require.Error(t, statusReport(commandStatus, t.TempDir(), ""))
	require.Error(t, statusReport(commandStatus, t.TempDir(), t.TempDir()))
}

func captureStdout(t *testing.T, action func() error) (string, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = writer
	collected := make(chan []byte, 1)
	go func() {
		data, readErr := io.ReadAll(reader)
		assert.NoError(t, readErr)
		collected <- data
	}()
	actionErr := action()
	os.Stdout = original
	require.NoError(t, writer.Close())
	output := <-collected
	require.NoError(t, reader.Close())
	return string(output), actionErr
}

func fakeLauncher(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	state := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "tests", "fuzz"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tests", "fuzz", "go.mod"), []byte("module example.test/fuzz\n"), 0o600))
	sample := resource.Sample{Cgroup: "campaign", Available: 1 << 40, Headroom: 1 << 40, DiskFree: 1 << 40}
	original := openBudget
	openBudget = func(context.Context) (*resource.Budget, error) {
		return resource.NewBudget(func(context.Context, bool) (resource.Sample, error) { return sample, nil }, 0, 1<<40), nil
	}
	t.Cleanup(func() { openBudget = original })
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("PIKO_FUZZ_ROOT", root)
	t.Setenv("PIKO_FUZZ_STATE", state)
	t.Setenv("PIKO_FUZZ_CPUS", "2")
	t.Setenv("PIKO_FUZZ_MEMORY_BYTES", "1073741824")
	t.Setenv("PIKO_FUZZ_MIN_FREE_BYTES", "0")
	t.Setenv("PIKO_FUZZ_DISK_BYTES", "1099511627776")
	return state
}

func writeNativeReceipt(t *testing.T, directory string, native *campaign.NativeRun) {
	t.Helper()
	data, err := json.Marshal(native)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "native.json"), data, 0o600))
}

func writeSeededReport(t *testing.T, directory string, report *campaign.Report) {
	t.Helper()
	data, err := json.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "report.json"), data, 0o600))
}

func seededReport() *campaign.Report {
	return &campaign.Report{
		Version: 1,
		Status:  "paused",
		Next:    1,
		Elapsed: 1500 * time.Millisecond,
		Options: campaign.Options{Targets: []string{"pages"}, Mode: modeBoth, Browser: browserSample, Seed: 7, Cases: 3},
	}
}

func TestRunPrintsHelpWithoutCommand(t *testing.T) {
	for _, arguments := range [][]string{nil, {"help"}} {
		output, err := captureStdout(t, func() error { return run(t.Context(), arguments) })

		require.NoError(t, err)
		assert.Contains(t, output, "Use ./hack/fuzz/fuzz.sh")
		assert.Contains(t, output, "Targets:")
		assert.Contains(t, output, "Go fuzz functions")
	}
}

func TestRunRejectsUnknownCommandAndBadFlags(t *testing.T) {
	fakeLauncher(t)

	assert.EqualError(t, run(t.Context(), []string{"explode"}), `unknown command "explode"`)
	assert.Error(t, run(t.Context(), []string{"status", "-no-such-flag"}))
	assert.ErrorContains(t, run(t.Context(), []string{"status", "-mode=sideways"}), "invalid mode")
	assert.ErrorContains(t, run(t.Context(), []string{"status", "-jobs=2"}), "one job")
	assert.ErrorContains(t, run(t.Context(), []string{"status", "-size=huge"}), "unknown generator size")
	assert.ErrorContains(t, run(t.Context(), []string{"status", "-size=tiny"}), "status uses the saved generation size")
	assert.ErrorContains(t, run(t.Context(), []string{"status", "one", "two"}), "at most one directory")
	assert.ErrorContains(t, run(t.Context(), []string{"preview", t.TempDir()}), "preview accepts flags only")
	assert.ErrorContains(t, run(t.Context(), []string{"gen", "-target=no-such-target"}), "unknown or repeated target")
	assert.ErrorContains(t, run(t.Context(), []string{"status", "-runtime-coverage"}), "-runtime-coverage applies to seeded application campaigns")
}

func TestRunGeneratePrintsCaseWithSources(t *testing.T) {
	fakeLauncher(t)

	output, err := captureStdout(t, func() error { return run(t.Context(), []string{"gen", "-target=binder", "-seed=3"}) })

	require.NoError(t, err)
	var printed struct {
		Target  string
		Seed    uint64
		Sources map[string]string
	}
	require.NoError(t, json.Unmarshal([]byte(output), &printed))
	assert.Equal(t, "binder", printed.Target)
	assert.Equal(t, uint64(3), printed.Seed)
	assert.NotEmpty(t, printed.Sources)
}

func TestRunCoverageInventoriesGeneratedWindow(t *testing.T) {
	fakeLauncher(t)

	output, err := captureStdout(t, func() error {
		return run(t.Context(), []string{"coverage", "-target=pages", "-n=2", "-browser=off"})
	})

	require.NoError(t, err)
	var printed map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &printed))
	assert.NotEmpty(t, printed)
}

func TestRunCoverageInventoriesSavedApplicationCase(t *testing.T) {
	fakeLauncher(t)
	directory := filepath.Join(t.TempDir(), "case")
	_, err := captureStdout(t, func() error {
		return run(t.Context(), []string{"gen", "-target=pages", "-out=" + directory})
	})
	require.NoError(t, err)

	output, err := captureStdout(t, func() error { return run(t.Context(), []string{"coverage", directory}) })

	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(output))
}

func TestRunCoverageRejectsSavedPackageCase(t *testing.T) {
	fakeLauncher(t)
	directory := filepath.Join(t.TempDir(), "case")
	require.NoError(t, run(t.Context(), []string{"gen", "-target=binder", "-out=" + directory}))

	assert.EqualError(t, run(t.Context(), []string{"coverage", directory}), "coverage takes a saved application case")
	assert.Error(t, run(t.Context(), []string{"coverage", filepath.Join(t.TempDir(), "absent")}))
}

func TestStatusReportDescribesNativeCampaign(t *testing.T) {
	directory := t.TempDir()
	native := &campaign.NativeRun{Fuzzer: "raw-sfc", Status: statusPass, Cases: 4}
	native.Resources.QuotaCPUs = 2
	native.Resources.MemoryPeak = 4096
	writeNativeReceipt(t, directory, native)

	status, err := captureStdout(t, func() error { return statusReport(commandStatus, t.TempDir(), directory) })
	require.NoError(t, err)
	assert.Equal(t, directory+"\npass, Go fuzzer raw-sfc, quota 2.00 CPUs, cumulative kernel memory.peak 4096 bytes\n", status)

	report, err := captureStdout(t, func() error { return statusReport("report", t.TempDir(), directory) })
	require.NoError(t, err)
	var printed campaign.NativeRun
	require.NoError(t, json.Unmarshal([]byte(report), &printed))
	assert.Equal(t, "raw-sfc", printed.Fuzzer)
}

func TestStatusReportDescribesLatestSeededCampaign(t *testing.T) {
	state := t.TempDir()
	directory := filepath.Join(state, "campaigns", "20260101T000000Z-1")
	report := seededReport()
	report.LastStop = "stopped by operator"
	report.Resources.Cgroup = "campaign"
	report.Resources.MemoryPeak = 2048
	report.Resources.MemoryMax = 8192
	report.Resources.QuotaCPUs = 1.5
	report.RuntimeCoverage = map[string]campaign.RuntimeCoverageCount{
		modeInterp: {
			LastArtefact:            "interp-1",
			LastGoStatements:        runtimecoverage.Counts{Covered: 5, Total: 10},
			LastJavaScriptFunctions: &runtimecoverage.Counts{Covered: 2, Total: 3},
			Runs:                    1,
			BrowserRuns:             1,
		},
		modeCompiled: {LastArtefact: "compiled-1", LastGoStatements: runtimecoverage.Counts{Covered: 1, Total: 2}, Runs: 1},
	}
	writeSeededReport(t, directory, report)

	status, err := captureStdout(t, func() error { return statusReport(commandStatus, state, "") })

	require.NoError(t, err)
	assert.Contains(t, status, directory+"\npaused, next trial 1, active runtime 1.5s\n")
	assert.Contains(t, status, "Cumulative kernel memory.peak 2048 bytes, memory.max 8192 bytes, CPU quota 1.50\n")
	assert.Contains(t, status, "interp runtime coverage: 1 Go receipts, 1 Chromium receipts, latest interp-1 (5/10 Go statements, 2/3 JS functions)\n")
	assert.Contains(t, status, "compiled runtime coverage: 1 Go receipts, 0 Chromium receipts, latest compiled-1 (1/2 Go statements)\n")
	assert.True(t, strings.HasSuffix(status, "stopped by operator\n"))

	full, err := captureStdout(t, func() error { return statusReport("report", state, directory) })
	require.NoError(t, err)
	var printed campaign.Report
	require.NoError(t, json.Unmarshal([]byte(full), &printed))
	assert.Equal(t, uint64(7), printed.Options.Seed)
}

func TestStatusReportRejectsCorruptNativeReceipt(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "native.json"), []byte("{"), 0o600))

	assert.ErrorContains(t, statusReport(commandStatus, t.TempDir(), directory), "decoding native campaign receipt")
}

func TestRunCampaignCommandRejectsDirectoryForNewCampaign(t *testing.T) {
	fakeLauncher(t)

	err := run(t.Context(), []string{"campaign", t.TempDir(), "-target=binder"})

	assert.EqualError(t, err, "campaign accepts flags; use replay to run a case directory")
}

func TestRunCampaignCommandRemovesToolDirectory(t *testing.T) {
	state := fakeLauncher(t)

	require.Error(t, run(t.Context(), []string{"campaign", t.TempDir(), "-target=binder"}))

	entries, err := os.ReadDir(filepath.Join(state, toolsDirectoryName))
	require.NoError(t, err)
	assert.Empty(t, entries, "tool directory outlived the command")
}

func TestRunResumeRequiresSavedCampaign(t *testing.T) {
	fakeLauncher(t)

	assert.ErrorIs(t, run(t.Context(), []string{"resume"}), fs.ErrNotExist)
	assert.EqualError(t, run(t.Context(), []string{"replay"}), "a saved case directory is required")
	assert.EqualError(t, run(t.Context(), []string{"reduce"}), "a saved case directory is required")
}

func TestRunResumeReportsCompletedAndFailedNativeCampaigns(t *testing.T) {
	state := fakeLauncher(t)
	completed := filepath.Join(state, "campaigns", "20260101T000000Z-go-1")
	writeNativeReceipt(t, completed, &campaign.NativeRun{Fuzzer: "raw-sfc", Status: statusPass, Cases: 1})

	output, err := captureStdout(t, func() error { return run(t.Context(), []string{"resume"}) })
	require.NoError(t, err)
	assert.Equal(t, "Native campaign already completed "+completed+"\n", output)

	failed := filepath.Join(t.TempDir(), "failed")
	writeNativeReceipt(t, failed, &campaign.NativeRun{Fuzzer: "raw-sfc", Status: statusFailed, Cases: 1})
	assert.ErrorContains(t, run(t.Context(), []string{"resume", failed}), "native campaign recorded a failure")
}

func TestRunResumeRejectsCorruptNativeReceipt(t *testing.T) {
	fakeLauncher(t)
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "native.json"), []byte("{"), 0o600))

	assert.ErrorContains(t, run(t.Context(), []string{"resume", directory}), "decoding native campaign receipt")
}

func TestRunResumeSeededRejectsSelectionOverrides(t *testing.T) {
	fakeLauncher(t)
	directory := t.TempDir()
	writeSeededReport(t, directory, seededReport())

	for _, key := range []string{"target=binder", "mode=interp", "browser=off", "seed=9", "engine=go", "fuzzer=raw-sfc"} {
		flagName, _, _ := strings.Cut(key, "=")
		err := run(t.Context(), []string{"resume", directory, "-" + key})
		assert.EqualError(t, err, "resume uses the saved -"+flagName+" setting", key)
	}
	assert.Error(t, run(t.Context(), []string{"resume", t.TempDir()}), "a directory without a report resumed")
}

func TestRunReplayRejectsUnsupportedOverridesAndMissingCase(t *testing.T) {
	fakeLauncher(t)
	directory := filepath.Join(t.TempDir(), "case")
	require.NoError(t, run(t.Context(), []string{"gen", "-target=binder", "-out=" + directory}))

	assert.ErrorContains(t, run(t.Context(), []string{"replay", directory, "-mode=interp"}), "-mode requires an application target")
	assert.EqualError(t, run(t.Context(), []string{"replay", directory, "-runtime-coverage"}), "runtime coverage uses application hosts")
	assert.Error(t, run(t.Context(), []string{"replay", t.TempDir()}))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "options.json"), []byte("{"), 0o600))
	assert.ErrorContains(t, run(t.Context(), []string{"replay", directory}), "decoding")
}

func TestInterpretedModeNeedsLocalPipit(t *testing.T) {
	t.Setenv(pipitpending.LocalEnv, "")
	var options flags
	options.mode = modeInterp
	options.browser = browserOff
	options.engine = engineSeeded
	options.n = 1
	options.jobs = 1
	options.cpus = 1

	assert.ErrorIs(t, validateOptions(commandCampaign, &options), errInterpretedNeedsLocalPipit)
}
