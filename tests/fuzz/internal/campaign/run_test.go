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

package campaign

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
)

type testBudget struct{}

func (testBudget) Admit(context.Context, uint64) error { return nil }

func (testBudget) Sample(context.Context) (resource.Sample, error) { return resource.Sample{}, nil }

func (testBudget) SampleNow(context.Context) (resource.Sample, error) { return resource.Sample{}, nil }

type scriptedBudget struct {
	admitErr error
	before   resource.Sample
	after    resource.Sample
}

func (budget scriptedBudget) Admit(context.Context, uint64) error { return budget.admitErr }

func (budget scriptedBudget) Sample(context.Context) (resource.Sample, error) {
	return budget.before, nil
}

func (budget scriptedBudget) SampleNow(context.Context) (resource.Sample, error) {
	return budget.after, nil
}

func TestExecuteResumesAtTheInterruptedCase(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	report := cleanReport()
	var visited []uint64
	stopped := false
	h := Harness{Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: &bytes.Buffer{}, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result {
		visited = append(visited, c.Index)
		if c.Index == 1 && !stopped {
			stopped = true
			cancel()
			return Result{Kind: "interrupted"}
		}
		return Result{Target: c.Target, Index: c.Index, Kind: "pass"}
	}}
	require.ErrorIs(t, h.Execute(ctx, dir, &report), context.Canceled)
	saved, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), saved.Next)
	assert.NotEmpty(t, saved.Active)
	output := &bytes.Buffer{}
	h.Output = output
	require.NoError(t, h.Execute(t.Context(), dir, &saved))
	assert.Equal(t, uint64(3), saved.Next)
	assert.Equal(t, "complete", saved.Status)
	assert.Equal(t, uint64(3), saved.ByTarget["pages"])
	require.Len(t, visited, 4)
	assert.Equal(t, visited[1], visited[2])
	assert.Contains(t, output.String(), "3 cases")
}

func TestExecuteResumesConfirmsAndReducesAnInterruptedFinding(t *testing.T) {
	dir := t.TempDir()
	report := cleanReport()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := true
	finding := Result{Kind: "expectation", Mode: "interp", Target: "pages", Phase: "rows", Subject: "pages/main.pk", Message: "rows: wrong order"}
	progress := &bytes.Buffer{}
	h := Harness{Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: progress, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result {
		if first {
			first = false
			cancel()
		}
		if len(c.Data) == 0 {
			return Result{Kind: "resource", Mode: "interp", Target: c.Target}
		}
		return finding
	}}
	require.ErrorIs(t, h.Execute(ctx, dir, &report), context.Canceled)
	saved, err := Load(dir)
	require.NoError(t, err)
	assert.Zero(t, saved.Next)
	assert.NotEmpty(t, saved.Active)
	assert.Zero(t, saved.Metrics.Cases)
	report = saved
	require.Error(t, h.Execute(t.Context(), dir, &report), "passed an injected mismatch")
	assert.Contains(t, progress.String(), "pages[0]: expectation (rows)")
	entries, err := os.ReadDir(filepath.Join(dir, "failures"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	bundle := filepath.Join(dir, "failures", entries[0].Name())
	for _, name := range []string{"case.json", "src/pages/main.pk", "confirmation.json", "original.json", "result.json", "options.json"} {
		assert.FileExists(t, filepath.Join(bundle, name))
	}
	assert.Equal(t, uint64(1), report.Metrics.Cases)
	assert.Len(t, report.Metrics.ConfirmedFingerprints, 1)
	original, err := casefile.Read(bundle)
	require.NoError(t, err)
	out := filepath.Join(bundle, "minimal")
	_, err = h.Reduce(t.Context(), bundle, out, &original, report.Options)
	require.NoError(t, err)
	minimal, err := casefile.Read(out)
	require.NoError(t, err)
	assert.Len(t, minimal.Data, 1)
	assert.True(t, finding.matches(h.RunOne(t.Context(), &minimal, report.Options.Mode, report.Options.Browser)))
}

func TestExecuteRejectsSourceOrGeneratorDrift(t *testing.T) {
	testCases := []struct {
		name   string
		change func(*Report, *casefile.Case)
	}{
		{name: "active bytes", change: func(_ *Report, c *casefile.Case) { c.Files["pages/main.pk"] += " " }},
		{name: "case generator", change: func(_ *Report, c *casefile.Case) { c.Generator = "future" }},
		{name: "case profile", change: func(_ *Report, c *casefile.Case) { c.ProfileSHA256 = "different" }},
		{name: "report profile", change: func(report *Report, _ *casefile.Case) { report.GeneratorProfile = "different" }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			report := cleanReport()
			report.Generator = generate.Version
			report.GeneratorProfile = generate.ProfileSHA256
			c := generate.New("pages", report.Options.Seed, 0)
			report.Active = c.Hash()
			testCase.change(&report, &c)
			dir := t.TempDir()
			require.NoError(t, casefile.Write(filepath.Join(dir, "active"), &c))
			ran := false
			h := Harness{Budget: testBudget{}, trial: func(context.Context, *casefile.Case, string, string) Result {
				ran = true
				return Result{}
			}}
			require.Error(t, h.Execute(t.Context(), dir, &report), "accepted a stale checkpoint")
			assert.False(t, ran, "ran a stale active case")
		})
	}
}

func TestExecuteRetriesActiveCaseAfterInfrastructureStop(t *testing.T) {
	report := cleanReport()
	report.Options.Cases = 1
	dir := t.TempDir()
	failed := false
	h := Harness{Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: &bytes.Buffer{}, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result {
		if !failed {
			failed = true
			return Result{Kind: "harness", Message: "worker unavailable", Target: c.Target, Checks: map[string]CheckMetric{"compiled": {Status: "pending", Admission: AdmissionMetric{EstimatedBytes: 1 << 30, Elapsed: time.Second}}}}
		}
		return Result{Kind: "pass", Target: c.Target}
	}}
	require.Error(t, h.Execute(t.Context(), dir, &report), "missed worker failure")
	assert.Zero(t, report.Metrics.Cases)
	assert.Zero(t, report.Next)
	assert.NotEmpty(t, report.Active)
	assert.Equal(t, uint64(1), report.IncompleteAttempts["harness"])
	assert.Equal(t, uint64(1), report.IncompleteChecks["compiled"].AdmissionDenied)
	assert.Equal(t, time.Second, report.IncompleteChecks["compiled"].AdmissionElapsed)
	require.NoError(t, h.Execute(t.Context(), dir, &report))
	assert.Equal(t, uint64(1), report.Metrics.Cases)
	assert.Equal(t, uint64(1), report.Next)
	assert.Equal(t, "complete", report.Status)
}

func TestExecuteCommitsAConfirmedTimeoutAndAdvances(t *testing.T) {
	report := cleanReport()
	report.Options.KeepGoing = true
	dir := t.TempDir()
	h := Harness{Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: &bytes.Buffer{}, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result {
		if c.Index == 0 {
			return failure("timeout", "interp", errPhaseTimeout, Result{Target: c.Target, Index: c.Index, Kind: "pass"})
		}
		return Result{Kind: "pass", Target: c.Target}
	}}
	require.Error(t, h.Execute(t.Context(), dir, &report), "confirmed findings must be reported")
	assert.Equal(t, uint64(3), report.Next, "stream did not advance past the hang")
	assert.Empty(t, report.Active)
	assert.Equal(t, uint64(1), report.Outcomes["timeout"])
	assert.Len(t, report.Metrics.ConfirmedFingerprints, 1)
	assert.Equal(t, uint64(1), report.Metrics.Targets["pages"].Failed)
	entries, err := os.ReadDir(filepath.Join(dir, "failures"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.FileExists(t, filepath.Join(dir, "failures", entries[0].Name(), "confirmation.json"))
}

func TestExecuteKeepsCompletedAndActiveTrialsSeparateAtTheDurationLimit(t *testing.T) {
	report := cleanReport()
	report.Options.Duration = 50 * time.Millisecond
	h := Harness{Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: &bytes.Buffer{}, trial: func(ctx context.Context, c *casefile.Case, _, _ string) Result {
		<-ctx.Done()
		return Result{Kind: "interrupted", Target: c.Target}
	}}
	dir := t.TempDir()
	require.NoError(t, h.Execute(t.Context(), dir, &report))
	assert.Equal(t, "time-limit", report.Status)
	assert.GreaterOrEqual(t, report.Elapsed, report.Options.Duration)
	assert.Zero(t, report.Next)
	assert.NotEmpty(t, report.Active)
	require.NoError(t, h.Execute(t.Context(), dir, &report))
	report.Options.Duration = 0
	h.trial = func(_ context.Context, c *casefile.Case, _, _ string) Result {
		return Result{Kind: "pass", Target: c.Target}
	}
	require.NoError(t, h.Execute(t.Context(), dir, &report))
	assert.Equal(t, uint64(3), report.Next, "resume failed to consume the pending case")
	assert.Equal(t, uint64(3), report.Metrics.Cases)
}

func TestExecuteRejectsSoakWithoutDiskBudget(t *testing.T) {
	t.Setenv(diskBudgetVariable, "")
	report := cleanReport()
	report.Options.Suite = "soak"
	h := Harness{Budget: testBudget{}, trial: func(context.Context, *casefile.Case, string, string) Result {
		return Result{Kind: "pass"}
	}}
	require.ErrorIs(t, h.Execute(t.Context(), t.TempDir(), &report), errDiskBudget)
}

func TestClassifySeparatesInfrastructureFromProductFindings(t *testing.T) {
	for _, message := range []string{"infrastructure: binder worker: exit status 1", "stage: invalid workspace", "worker-build/generate-worker: compile startup error", "listen: operation not permitted", "resource measurement: cgroup disappeared"} {
		result := classify(t.Context(), "worker", "interp", errors.New(message), Result{Checks: map[string]CheckMetric{"interp": {Status: "running"}}})
		assert.Equal(t, "harness", result.Kind, message)
		assert.False(t, confirmable(result), message)
		assert.Equal(t, "incomplete", result.Checks["interp"].Status, message)
	}
	result := classify(t.Context(), "target", "package", errors.New("binder/field-count: mismatch"), Result{})
	assert.Equal(t, "target", result.Kind)
	assert.True(t, confirmable(result))
}

func TestClassifyTreatsAnExhaustedBrowserBudgetAsHarness(t *testing.T) {
	message := "checkout/submit: driver: " + casefile.BrowserBudgetMarker + ": browser session: draft: context deadline exceeded"

	result := classify(t.Context(), "browser", "interp", errors.New(message), Result{Checks: map[string]CheckMetric{"interp": {Status: "running"}}})

	assert.Equal(t, "harness", result.Kind, "a worker that ran out of its own budget is not a product failure")
	assert.False(t, confirmable(result))
}

func TestClassifyConfirmsTimeoutsAndKeepsResourceStops(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		kind        string
		confirmable bool
	}{
		{name: "phase timeout", err: fmt.Errorf("worker: %w", errPhaseTimeout), kind: "timeout", confirmable: true},
		{name: "inner deadline", err: fmt.Errorf("probe: %w", context.DeadlineExceeded), kind: "resource"},
		{name: "readiness deadline", err: fmt.Errorf("startup: host startup timed out: %w", context.DeadlineExceeded), kind: "resource"},
		{name: "cgroup oom", err: fmt.Errorf("%w: interp cgroup OOM", resource.ErrLimit), kind: "resource"},
		{name: "disk full", err: fmt.Errorf("write: %w", syscall.ENOSPC), kind: "resource"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := classify(t.Context(), "worker", "interp", testCase.err, Result{Checks: map[string]CheckMetric{"interp": {Status: "failed"}}})
			assert.Equal(t, testCase.kind, result.Kind)
			assert.Equal(t, testCase.confirmable, confirmable(result))
		})
	}
}

func TestClassifyReportsTheParentCause(t *testing.T) {
	budgetStop, cancelBudget := context.WithCancelCause(t.Context())
	cancelBudget(fmt.Errorf("%w: host memory", resource.ErrLimit))
	result := classify(budgetStop, "worker", "interp", errPhaseTimeout, Result{})
	assert.Equal(t, "resource", result.Kind)
	stopped, cancel := context.WithCancel(t.Context())
	cancel()
	result = classify(stopped, "worker", "interp", errors.New("killed"), Result{})
	assert.Equal(t, "interrupted", result.Kind)
}

func TestRunPhaseRecordsAPassingCheck(t *testing.T) {
	h := Harness{Budget: testBudget{}, seenModes: map[string]bool{}}
	result := Result{Checks: map[string]CheckMetric{}}
	var directory string
	err := h.runPhase(t.Context(), phaseRequest{scratch: t.TempDir(), name: "interp", estimate: 1 << 20, action: func(_ context.Context, dir string) error {
		directory = dir
		return nil
	}}, &result)
	require.NoError(t, err)
	check := result.Checks["interp"]
	assert.Equal(t, "pass", check.Status)
	assert.True(t, check.Admission.Granted)
	assert.Equal(t, uint64(1<<20), check.Admission.EstimatedBytes)
	assert.True(t, check.FirstInInvocation)
	assert.DirExists(t, filepath.Join(directory, "tmp"))
	err = h.runPhase(t.Context(), phaseRequest{scratch: t.TempDir(), name: "interp", action: func(context.Context, string) error { return nil }}, &result)
	require.NoError(t, err)
	assert.False(t, result.Checks["interp"].FirstInInvocation)
}

func TestRunPhaseReportsATimeoutAsAConfirmableFailure(t *testing.T) {
	h := Harness{Budget: testBudget{}, phaseTimeout: time.Millisecond}
	result := Result{Checks: map[string]CheckMetric{}}
	err := h.runPhase(t.Context(), phaseRequest{scratch: t.TempDir(), name: "compiled", action: func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}}, &result)
	require.ErrorIs(t, err, errPhaseTimeout)
	assert.Equal(t, "failed", result.Checks["compiled"].Status)
	classified := classify(t.Context(), "worker", "compiled", err, result)
	assert.Equal(t, "timeout", classified.Kind)
	assert.True(t, confirmable(classified))
	assert.Equal(t, "failed", classified.Checks["compiled"].Status)
}

func TestRunPhaseReportsRefusedAdmissionAndOutOfMemory(t *testing.T) {
	refused := errors.New("admission refused")
	h := Harness{Budget: scriptedBudget{admitErr: refused}}
	result := Result{Checks: map[string]CheckMetric{}}
	err := h.runPhase(t.Context(), phaseRequest{scratch: t.TempDir(), name: "interp", action: func(context.Context, string) error { return nil }}, &result)
	require.ErrorIs(t, err, refused)
	assert.False(t, result.Checks["interp"].Admission.Granted)
	group := t.TempDir()
	h = Harness{Budget: scriptedBudget{before: resource.Sample{Cgroup: group}, after: resource.Sample{Cgroup: group, OOMKills: 1}}}
	err = h.runPhase(t.Context(), phaseRequest{scratch: t.TempDir(), name: "interp", action: func(context.Context, string) error { return errors.New("signal: killed") }}, &result)
	require.ErrorIs(t, err, resource.ErrLimit)
	assert.Equal(t, "resource", classify(t.Context(), "worker", "interp", err, result).Kind)
}

func TestSaveFailureConfirmsMatchingReplays(t *testing.T) {
	c := generate.New("pages", 7, 0)
	original := failure("expectation", "interp", errors.New("rows: wrong order"), Result{Target: c.Target, Index: c.Index, Kind: "pass", Checks: map[string]CheckMetric{}})
	testCases := []struct {
		name      string
		replay    Result
		kind      string
		confirmed bool
	}{
		{name: "same failure", replay: original, kind: "expectation", confirmed: true},
		{name: "different failure", replay: failure("expectation", "interp", errors.New("title: wrong"), Result{Target: c.Target, Kind: "pass"}), kind: "flaky"},
		{name: "resource stop", replay: Result{Kind: "resource", Message: "memory"}, kind: "resource"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			h := Harness{Budget: testBudget{}, trial: func(context.Context, *casefile.Case, string, string) Result { return testCase.replay }}
			result := original
			require.NoError(t, h.saveFailure(t.Context(), dir, cleanReport().Options, &c, &result))
			assert.Equal(t, testCase.kind, result.Kind)
			assert.Equal(t, testCase.confirmed, result.Confirmed)
			bundle := failurePath(dir, &c, &original)
			for _, name := range []string{"case.json", "options.json", "original.json", "confirmation.json", "result.json"} {
				assert.FileExists(t, filepath.Join(bundle, name))
			}
		})
	}
}

func TestSaveFailureSkipsConfirmationForHarnessFailures(t *testing.T) {
	c := generate.New("pages", 7, 0)
	dir := t.TempDir()
	ran := false
	h := Harness{Budget: testBudget{}, trial: func(context.Context, *casefile.Case, string, string) Result {
		ran = true
		return Result{}
	}}
	result := Result{Target: c.Target, Kind: "harness", Message: "worker unavailable"}
	require.NoError(t, h.saveFailure(t.Context(), dir, cleanReport().Options, &c, &result))
	assert.False(t, ran)
	assert.NoFileExists(t, filepath.Join(failurePath(dir, &c, &result), "confirmation.json"))
}

func TestSaveFailureReportsAnUninspectableBundle(t *testing.T) {
	c := generate.New("pages", 7, 0)
	dir := t.TempDir()
	result := Result{Target: c.Target, Kind: "harness"}
	bundle := failurePath(dir, &c, &result)
	require.NoError(t, os.MkdirAll(filepath.Dir(bundle), 0o700))
	require.NoError(t, os.WriteFile(bundle, []byte("not a directory"), 0o600))
	err := Harness{Budget: testBudget{}}.saveFailure(t.Context(), dir, cleanReport().Options, &c, &result)
	require.ErrorContains(t, err, "inspecting failure bundle")
}

func TestRecoverGeneratorKeepsTheStackOutOfTheError(t *testing.T) {
	progress := &bytes.Buffer{}
	h := Harness{Progress: progress}
	generateFailing := func() (err error) {
		defer func() { h.recoverGenerator(recover(), &err, "pages", 7, 3) }()
		panic("unknown production")
	}
	err := generateFailing()
	require.EqualError(t, err, "generating pages seed 7 index 3: panic: unknown production")
	assert.Contains(t, progress.String(), "generator panic for pages seed 7 index 3")
	assert.Contains(t, progress.String(), "goroutine")
}

func TestGenerateCaseMatchesTheGenerator(t *testing.T) {
	c, err := Harness{}.generateCase("pages", 7, 1, "")
	require.NoError(t, err)
	regenerated := generate.New("pages", 7, 1)
	assert.Equal(t, regenerated.Hash(), c.Hash())
	replayed, err := Harness{}.replayCase(c.Target, c.Seed, c.Index, c.Data, c.Size)
	require.NoError(t, err)
	assert.Equal(t, c.Hash(), replayed.Hash())
}

func TestPlannedBrowserRoutesMarkSelectionWithoutObservations(t *testing.T) {
	c := generate.New("pages", 7, 1)
	routes, err := plannedBrowserRoutes(&c, true)
	require.NoError(t, err)
	for path, route := range routes {
		assert.True(t, route.Selected, path)
		assert.False(t, route.Visited, path)
		assert.False(t, route.Verified, path)
	}
}

func TestFailureKeepsIndependentSiblingFailures(t *testing.T) {
	c := casefile.Case{Target: "pages", Expected: casefile.Expected{Status: 200}, Probes: []casefile.Probe{{Path: "/left", Status: 200, Text: "left"}, {Path: "/right", Status: 200, Text: "right"}}}
	observation := runner.Observation{Status: 200, ContentType: "text/html", MetadataTitle: "Piko Page", Probes: []runner.ProbeObservation{{Path: "/left", Status: 200}, {Path: "/right", Status: 200}}}
	result := Result{Target: c.Target, Kind: "pass"}
	for _, issue := range runner.IndependentFailures(&c, &observation) {
		result = failure("expectation", "compiled", issue, result)
	}
	require.Len(t, result.Failures, 2)
	assert.NotEqual(t, result.Failures[0].Subject, result.Failures[1].Subject)
}

func TestFailureKeepsParityDifferencesDistinctFromExpectations(t *testing.T) {
	c := casefile.Case{Target: "pages", Expected: casefile.Expected{Status: 200}}
	interpreted := runner.Observation{Status: 200, ContentType: "text/html", MetadataTitle: "Piko Page", ApplicationDOM: "<main></main>"}
	compiled := interpreted
	compiled.ApplicationDOM = "<main data-mode=compiled></main>"
	for _, observation := range []runner.Observation{interpreted, compiled} {
		require.Empty(t, runner.IndependentFailures(&c, &observation), "invalid parity-only fixture")
	}
	difference := runner.CompareCase(&c, &interpreted, &compiled)
	require.Error(t, difference, "parity-only change escaped")
	result := failure("disagreement", "both", difference, Result{Target: c.Target, Kind: "pass"})
	assert.Equal(t, "disagreement", result.Kind)
	assert.Equal(t, "dom", result.Phase)
	require.Len(t, result.Failures, 1)
	assert.Equal(t, "both", result.Failures[0].Mode)
}

func TestLoadActiveRejectsAnUnreadableActiveCase(t *testing.T) {
	report := cleanReport()
	report.Active = "missing"
	_, err := loadActive(t.TempDir(), &report)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestBrowserScheduleVisitsFourStrata(t *testing.T) {
	for _, seed := range []uint64{0, 1, 7, 17, 39, ^uint64(0)} {
		c := casefile.Case{Target: "pages", Seed: seed, BrowserSampleVersion: 1}
		for window := range uint64(3) {
			var strata [4]int
			for decade := range uint64(4) {
				selections := 0
				for offset := range uint64(10) {
					c.Index = 40*window + 10*decade + offset
					if BrowserCase(&c, "sample") {
						selections++
						strata[c.Index%4]++
					}
				}
				require.Equal(t, 1, selections, "seed %d window %d decade %d", seed, window, decade)
			}
			require.Equal(t, [4]int{1, 1, 1, 1}, strata, "seed %d window %d", seed, window)
		}
		c.BrowserSampleVersion = 0
		for index := range uint64(100) {
			c.Index = index
			assert.Equal(t, index%10 == seed%10, BrowserCase(&c, "sample"), "legacy seed %d index %d changed", seed, index)
		}
	}
	for _, seed := range []uint64{2, 7} {
		routes := map[string]bool{}
		for index := range uint64(40) {
			c := generate.New("pages", seed, index)
			if !BrowserCase(&c, "sample") {
				continue
			}
			for _, probe := range c.Probes {
				if probe.BrowserProgram {
					routes[probe.Path] = true
				}
			}
		}
		for _, path := range []string{"/typed?mode=missing", "/typed?mode=nil", "/typed?mode=empty", "/typed?mode=full", "/scoped?view=first", "/scoped?view=other"} {
			require.True(t, routes[path], "seed %d missed %s", seed, path)
		}
	}
}

func TestRouteMetricDenominators(t *testing.T) {
	c := generate.New("pages", 7, 1)
	var counts ModeMetrics
	for _, selected := range []bool{true, false} {
		check := CheckMetric{BrowserSelected: selected}
		visits := []casefile.BrowserRouteVisit{{Path: c.Request, Landed: c.Request, Status: c.Expected.Status}}
		for _, probe := range c.Probes {
			if probe.BrowserProgram {
				visits = append(visits, casefile.BrowserRouteVisit{Path: probe.Path, Landed: probe.Path, Status: probe.Status, Program: true, ProgramRoot: probe.ProgramRoot, Output: probe.Output})
			}
		}
		client := *c.Expected.Client
		client.BrowserRoutes = visits
		client.BrowserRoutes[len(client.BrowserRoutes)-1].Output = nil
		observation := runner.Observation{Client: &client, Counts: []string{client.Before[0].Count, client.After[0].Count}}
		if !selected {
			observation = runner.Observation{}
		}
		require.NoError(t, check.recordObservation(&c, &observation, nil))
		counts.observe("interp", &check)
	}
	want := map[string]BrowserRouteCounts{
		"/main":              {Eligible: 2, Selected: 1, Visited: 1, Verified: 1},
		"/scoped?view=first": {Eligible: 2, Selected: 1, Visited: 1, Verified: 1},
		"/scoped?view=other": {Eligible: 2, Selected: 1, Visited: 1, Verified: 0},
	}
	assert.Equal(t, want, counts.BrowserRoutes)
}

func TestDedicatedBrowserRoutes(t *testing.T) {
	for _, target := range []string{"browser-events", "browser-lifecycle", "partial-components"} {
		c := generate.New(target, 7, 0)
		observation := runner.Observation{Client: c.Expected.Client}
		want := map[string]bool{}
		switch target {
		case "browser-events":
			expected := *c.Expected.Client.Events
			actualClient := *c.Expected.Client
			actualClient.Events = &expected
			actualClient.Events.BrowserPath = "/events"
			observation.Client = &actualClient
			want["/events"] = true
		case "browser-lifecycle":
			for _, state := range c.Expected.Client.Lifecycle.States {
				want[state.Path] = true
			}
		case "partial-components":
			for _, state := range c.Expected.Client.PartialComponents.States {
				want[state.Path] = true
			}
		}
		evidence, err := runner.BrowserRouteResults(&c, &observation, true)
		if err != nil {
			assert.Contains(t, err.Error(), "browser/route: primary/", "%s failed outside the primary route", target)
		}
		for path := range want {
			route := evidence[path]
			assert.Equal(t, "workflow", route.Role, "%s %s", target, path)
			assert.True(t, route.Selected && route.Visited && route.Verified, "%s %s got %+v", target, path, route)
		}
		require.Len(t, evidence, len(want)+1, "repeated visits inflated %s", target)
		if target == "browser-events" {
			observation.Client.Events.BrowserPath = "/elsewhere"
			evidence, err := runner.BrowserRouteResults(&c, &observation, true)
			require.Error(t, err)
			assert.False(t, evidence["/events"].Visited, "wrong route received credit")
			assert.False(t, evidence["/events"].Verified, "wrong route received credit")
		}
	}
	c := generate.New("form-workflow", 7, 0)
	evidence, err := runner.BrowserRouteResults(&c, &runner.Observation{}, true)
	require.NoError(t, err, "an unobserved workflow route is not a navigation failure")
	require.Len(t, evidence, 1)
	assert.True(t, evidence["/workflow"].Selected)
	assert.False(t, evidence["/workflow"].Visited)
	actual := runner.Observation{FormWorkflow: &casefile.FormWorkflowObservation{BrowserPath: "/workflow"}}
	evidence, err = runner.BrowserRouteResults(&c, &actual, true)
	require.NoError(t, err)
	assert.True(t, evidence["/workflow"].Visited)
	assert.False(t, evidence["/workflow"].Verified, "URL-only form falsely credited")
}

func cleanReport() Report {
	return Report{Version: 1, Options: Options{Targets: []string{"pages"}, Mode: "both", Browser: "off", Cases: 3, Seed: 42}, Outcomes: map[string]uint64{}, ByFeature: map[string]uint64{}, ByTarget: map[string]uint64{}, Modes: map[string]uint64{}}
}

func TestRunOneMeasuresOracleSensitivityInThePackageCheck(t *testing.T) {
	h := Harness{Budget: testBudget{}, Progress: &bytes.Buffer{}}
	c := generate.New("oracle-sensitivity", 7, 0)

	r := h.RunOne(t.Context(), &c, "both", "off")

	require.Equal(t, checkStatusPass, r.Kind, "%+v", r.Failures)
	assert.Equal(t, c.Hash(), r.Hash)
	assert.Positive(t, r.Elapsed)
	check := r.Checks[packageCheck]
	assert.Equal(t, checkStatusPass, check.Status)
	assert.NotEmpty(t, check.Mutations)
	assert.Equal(t, uint64(packagePhaseBytes), check.Admission.EstimatedBytes)
	assert.True(t, check.Admission.Granted)
}

func TestRunOneReportsAnUnmeasurableSensitivityCase(t *testing.T) {
	h := Harness{Budget: testBudget{}, Progress: &bytes.Buffer{}}
	c := casefile.Case{Target: "oracle-sensitivity"}

	r := h.RunOne(t.Context(), &c, "both", "off")

	assert.Equal(t, "target", r.Kind)
	assert.Equal(t, packageCheck, r.Mode)
	assert.Equal(t, "sensitivity", r.Phase)
	assert.Equal(t, StatusFailed, r.Checks[packageCheck].Status)
}

func TestRunOneClassifiesAWorkerFailurePerMode(t *testing.T) {
	h := Harness{Budget: testBudget{}, Progress: &bytes.Buffer{}}
	c := generate.New("pages", 7, 1)

	r := h.RunOne(t.Context(), &c, "unsupported", "off")

	assert.Equal(t, "worker", r.Kind)
	assert.Equal(t, "unsupported", r.Mode)
	assert.Equal(t, "mode", r.Phase)
	assert.Equal(t, StatusFailed, r.Checks["unsupported"].Status)
	assert.Equal(t, uint64(packagePhaseBytes), r.Checks["unsupported"].Admission.EstimatedBytes)
	assert.Contains(t, r.Observations, "unsupported")
}

func TestRunOneStopsWhenAdmissionIsRefused(t *testing.T) {
	h := Harness{Budget: scriptedBudget{admitErr: fmt.Errorf("%w: memory", resource.ErrLimit)}, Progress: &bytes.Buffer{}}
	c := generate.New("pages", 7, 1)

	r := h.RunOne(t.Context(), &c, "both", "off")

	assert.Equal(t, kindResource, r.Kind)
	assert.Equal(t, modeInterp, r.Mode)
	assert.False(t, r.Checks[modeInterp].Admission.Granted)
	assert.Equal(t, statusPending, r.Checks[modeCompiled].Status, "a stopped attempt ran the second mode")
}

func TestWithRuntimeCoverageRequiresADiskBudget(t *testing.T) {
	t.Setenv(diskBudgetVariable, "")
	_, err := Harness{}.WithRuntimeCoverage(t.TempDir())
	require.ErrorContains(t, err, "runtime coverage needs a verified disk budget")

	t.Setenv(diskBudgetVariable, "1073741824")
	h, err := Harness{Budget: testBudget{}, Progress: &bytes.Buffer{}}.WithRuntimeCoverage(filepath.Join(t.TempDir(), "runtime-coverage"))
	require.NoError(t, err)
	require.NotNil(t, h.coverage)
	c := generate.New("pages", 7, 1)

	r := h.RunOne(t.Context(), &c, "unsupported", "off")

	assert.Equal(t, "worker", r.Kind)
	assert.Equal(t, "mode", r.Phase)
	assert.Empty(t, r.RuntimeCoverage, "a failed measurement published coverage")
}

func TestRecordHostReadsTheHostBinaryOnlyWhenRequired(t *testing.T) {
	workerFailure := errors.New("worker failed")
	testCases := []struct {
		receiptErr error
		runErr     error
		c          casefile.Case
		name       string
		mode       string
		want       string
		message    string
	}{
		{name: "compiled host", c: casefile.Case{Target: "pages"}, mode: modeCompiled, want: "native"},
		{name: "interpreted project host", c: casefile.Case{Target: "checkout-workflow"}, mode: modeInterp, want: "interp-app"},
		{name: "interpreted without host", c: casefile.Case{Target: "pages"}, mode: modeInterp, want: ""},
		{name: "absent after worker failure", c: casefile.Case{Target: "pages"}, mode: modeCompiled, want: "native", receiptErr: fs.ErrNotExist, runErr: workerFailure},
		{name: "absent after success", c: casefile.Case{Target: "pages"}, mode: modeCompiled, want: "native", receiptErr: fs.ErrNotExist, message: "infrastructure: reading compiled host provenance"},
		{name: "unreadable after failure", c: casefile.Case{Target: "pages"}, mode: modeCompiled, want: "native", receiptErr: errors.New("corrupt"), runErr: workerFailure, message: "corrupt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var read []string
			trial := caseTrial{c: &testCase.c, harness: Harness{recordHostModules: func(path string) error {
				read = append(read, path)
				return testCase.receiptErr
			}}}
			dir := t.TempDir()

			err := trial.recordHost(dir, testCase.mode, testCase.runErr)

			if testCase.message != "" {
				require.ErrorContains(t, err, testCase.message)
			} else {
				require.NoError(t, err)
			}
			if testCase.want == "" {
				assert.Empty(t, read)
				return
			}
			assert.Equal(t, []string{filepath.Join(dir, testCase.want)}, read)
		})
	}
}

func TestRecordWorkersRecordsEachPresentWorkerOnce(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	workers := t.TempDir()
	require.NoError(t, os.Symlink(executable, filepath.Join(workers, "generate-worker")))
	run := &campaignRun{harness: Harness{WorkersDir: workers}, r: &Report{}, workersRecorded: map[string]bool{}}

	require.NoError(t, run.recordWorkers())
	require.NoError(t, run.recordWorkers())

	assert.Len(t, run.r.Provenance.WorkerModules, 1)
	assert.Contains(t, run.r.Provenance.WorkerModules, "generate-worker")
	assert.True(t, run.workersRecorded["generate-worker"])
	assert.False(t, run.workersRecorded["interp-worker"], "an absent worker was recorded")

	require.NoError(t, os.WriteFile(filepath.Join(workers, "interp-worker"), []byte("not a binary"), 0o600))
	require.Error(t, run.recordWorkers(), "an unreadable worker binary was accepted")

	unconfigured := &campaignRun{harness: Harness{}, r: &Report{}, workersRecorded: map[string]bool{}}
	require.NoError(t, unconfigured.recordWorkers())
	assert.Empty(t, unconfigured.r.Provenance.WorkerModules)
}

func TestReportRecordCountsRuntimeCoverageRuns(t *testing.T) {
	var report Report
	c := casefile.Case{Target: "pages", Features: []string{"loop"}}
	browser := runtimecoverage.Counts{}
	report.record(&c, &Result{Kind: checkStatusPass, RuntimeCoverage: map[string]runtimecoverage.Receipt{
		modeInterp:   {Artefact: "first"},
		modeCompiled: {Artefact: "second", JavaScriptFunctions: &browser},
	}, Checks: map[string]CheckMetric{modeInterp: {Status: checkStatusPass, BrowserObserved: true}, modeCompiled: {Status: "incomplete"}}})

	assert.Equal(t, RuntimeCoverageCount{Runs: 1, LastArtefact: filepath.Join("runtime-coverage", "first")}, report.RuntimeCoverage[modeInterp])
	assert.Equal(t, uint64(1), report.RuntimeCoverage[modeCompiled].BrowserRuns)
	assert.Equal(t, uint64(1), report.Outcomes[checkStatusPass])
	assert.Equal(t, uint64(1), report.ByFeature["loop"])
	assert.Equal(t, uint64(1), report.Modes[modeInterp])
	assert.Zero(t, report.Modes[modeCompiled], "an incomplete check counted as a run mode")
	assert.Equal(t, uint64(1), report.Browsers)
}

func TestSelectedModesAndPhaseEstimates(t *testing.T) {
	t.Setenv(pipitpending.LocalEnv, "1")
	pages := casefile.Case{Target: "pages", Index: 1, Seed: 2}
	assert.Equal(t, []string{modeInterp, modeCompiled}, selectedModes(&pages, "both"))
	assert.Equal(t, []string{modeCompiled}, selectedModes(&casefile.Case{Target: "embedded-deploy"}, "both"))
	assert.Equal(t, []string{modeInterp}, selectedModes(&pages, modeInterp))

	t.Setenv(pipitpending.LocalEnv, "")
	assert.Equal(t, []string{modeCompiled}, selectedModes(&pages, "both"))
	assert.Empty(t, selectedModes(&pages, modeInterp))

	assert.Equal(t, uint64(compiledPhaseBytes), phaseEstimate(&pages, modeCompiled, "off"))
	assert.Equal(t, uint64(interpretedPhaseBytes), phaseEstimate(&pages, modeInterp, "off"))
	assert.Equal(t, uint64(compiledPhaseBytes), phaseEstimate(&pages, modeInterp, "every"))
	assert.Equal(t, uint64(packagePhaseBytes), phaseEstimate(&casefile.Case{Target: "binder"}, packageCheck, "every"))
}
