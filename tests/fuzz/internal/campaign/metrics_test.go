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
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

func TestMetricsKeepIncompleteAndObservedDistinct(t *testing.T) {
	c := casefile.Case{Target: "pages", Features: []string{"loop", "text", "loop"}}
	result := Result{Target: "pages", Mode: "interp", Phase: "title", Kind: "expectation", Elapsed: time.Second,
		Checks: map[string]CheckMetric{
			"interp":   {Status: "failed", Elapsed: time.Second, FrontendAccepted: true, BrowserSelected: true, BrowserObserved: true},
			"compiled": {Status: "pending", BrowserSelected: true},
		}}
	var metrics RunMetrics
	metrics.Observe(&c, &result, true)
	metrics.Observe(&c, &Result{Target: "pages", Kind: "resource"}, false)
	assert.Zero(t, metrics.Checks["compiled"].Attempted)
	assert.Equal(t, uint64(1), metrics.Checks["compiled"].Skipped)
	assert.Equal(t, uint64(1), metrics.Checks["interp"].Failed)
	assert.Equal(t, uint64(1), metrics.Checks["interp"].FrontendAccepted)
	assert.Equal(t, uint64(1), metrics.Checks["compiled"].FrontendUnknown)
	assert.Equal(t, uint64(2), metrics.BrowserSelected)
	assert.Equal(t, uint64(1), metrics.BrowserObserved)
	assert.Equal(t, uint64(1), metrics.CasesWithoutModeData)
	assert.Equal(t, map[string]uint64{"loop + text": 2}, metrics.FeaturePairs)
	assert.Equal(t, uint64(1), metrics.ConfirmedFingerprints[result.Fingerprint()])
	assert.Equal(t, uint64(1), metrics.Targets["pages"].Incomplete)
	assert.Equal(t, uint64(2), metrics.Targets["pages"].Cases)
	assert.Contains(t, metrics.Summary(), "browser observations: 1/2")
}

func TestBinderMetricsDoNotCreditPageFeatures(t *testing.T) {
	var metrics RunMetrics
	metrics.Observe(&casefile.Case{Target: "binder", Features: []string{"counter", "loop"}}, &Result{Target: "binder", Kind: "pass", Checks: map[string]CheckMetric{"package": {Status: "pass"}}}, false)
	assert.Empty(t, metrics.FeaturePairs)
	assert.Equal(t, uint64(1), metrics.Targets["binder"].Passed)
}

func TestFirstAndSubsequentModeMemoryAndAdmission(t *testing.T) {
	var metrics RunMetrics
	c := casefile.Case{Target: "pages"}
	for _, sample := range []struct {
		first bool
		base  uint64
		peak  uint64
	}{{first: true, base: 10, peak: 90}, {first: false, base: 30, peak: 55}, {first: false, base: 40, peak: 53}, {first: true, base: 51, peak: 70}} {
		metrics.Observe(&c, &Result{Kind: "pass", Checks: map[string]CheckMetric{"compiled": {
			Status: "pass", FirstInInvocation: sample.first, Admission: AdmissionMetric{Granted: true, EstimatedBytes: 75, Elapsed: time.Millisecond},
			Memory: resource.PhaseMemory{StartKnown: true, StartBytes: sample.base, SampledPeakBytes: sample.peak, Samples: 2},
		}}}, false)
	}
	got := metrics.Checks["compiled"]
	assert.Equal(t, uint64(2), got.FirstInInvocationMemory.MeasuredChecks)
	assert.Equal(t, uint64(90), got.FirstInInvocationMemory.MaxSampledBytes)
	assert.Equal(t, uint64(55), got.SubsequentMemory.MaxSampledBytes)
	assert.Equal(t, uint64(80), got.FirstInInvocationMemory.MaxRiseFromStartBytes)
	assert.Equal(t, uint64(25), got.SubsequentMemory.MaxRiseFromStartBytes)
	assert.Equal(t, uint64(4), got.AdmissionGranted)
	assert.Equal(t, 4*time.Millisecond, got.AdmissionElapsed)
	assert.Equal(t, uint64(75), got.LargestEstimateBytes)
	assert.Contains(t, metrics.Summary(), "sampled cgroup memory.current peak")
}

func TestSelectedBrowserRequiresACompleteObservation(t *testing.T) {
	c := casefile.Case{Expected: casefile.Expected{Client: &casefile.ClientTransition{}}}
	observation := runner.Observation{Status: 200, Counts: []string{"0", "1"}}
	check := CheckMetric{BrowserSelected: true}
	require.Error(t, check.recordObservation(&c, &observation, nil), "accepted counter-only evidence for a shadow DOM scenario")
	assert.False(t, check.BrowserObserved)
	observation.Client = &casefile.ClientTransition{Before: []casefile.ClientInstance{{}}, After: []casefile.ClientInstance{{}}}
	require.NoError(t, check.recordObservation(&c, &observation, nil))
	assert.True(t, check.BrowserObserved, "complete observation rejected")
	action := casefile.ClientAction{Op: casefile.ResetControls}
	c.Expected.Client.Steps = []casefile.ClientStep{{Action: action}}
	require.Error(t, check.recordObservation(&c, &observation, nil), "incomplete action sequence earned browser credit")
	assert.False(t, check.BrowserObserved)
	observation.Client.Steps = []casefile.ClientStep{{Action: action}}
	require.Error(t, check.recordObservation(&c, &observation, nil), "missing post-action snapshot earned browser credit")
	assert.False(t, check.BrowserObserved)
	observation.Client.Steps[0].After = []casefile.ClientInstance{{}}
	require.NoError(t, check.recordObservation(&c, &observation, nil))
	assert.True(t, check.BrowserObserved, "complete action sequence rejected")
	c.Expected.Client = nil
	observation.Client = nil
	require.NoError(t, check.recordObservation(&c, &observation, nil))
	assert.True(t, check.BrowserObserved, "legacy counter evidence rejected")
	observation.Counts = nil
	require.Error(t, check.recordObservation(&c, &observation, nil), "missing selected counters passed")
	assert.False(t, check.BrowserObserved)
	check.BrowserSelected = false
	require.NoError(t, check.recordObservation(&c, &observation, nil))
	assert.False(t, check.BrowserObserved)
}

func TestProgramCoverageRequiresIndependentRuntimeCheck(t *testing.T) {
	c := casefile.Case{Target: "pages", Probes: []casefile.Probe{{Path: "/plain"}, {Path: "/empty", Program: true}, {Path: "/rows", Program: true, Output: make([]casefile.ProgramOutput, 7)}}}
	for probeIndex := range c.Probes {
		c.Probes[probeIndex].Status = 200
		for outputIndex := range c.Probes[probeIndex].Output {
			c.Probes[probeIndex].Output[outputIndex].ID = "row"
		}
	}
	var metrics RunMetrics
	metrics.Observe(&c, &Result{Kind: "expectation", Observations: map[string]runner.Observation{"interp": modelledReceipt(c)}, Checks: map[string]CheckMetric{"interp": {Status: "failed"}, "compiled": {Status: "pass", ProgramVerified: true}}}, false)
	interp := metrics.Checks["interp"]
	assert.Equal(t, uint64(2), interp.VerifiedProgramRequests)
	assert.Equal(t, uint64(7), interp.VerifiedProgramOutputs)
	compiled := metrics.Checks["compiled"]
	assert.Equal(t, uint64(2), compiled.ProgramRequests)
	assert.Equal(t, uint64(7), compiled.ProgramOutputs)
	assert.Zero(t, compiled.VerifiedProgramRequests)
	assert.Zero(t, compiled.VerifiedProgramOutputs)
	assert.Contains(t, metrics.Summary(), "0/2 requests and 0/7 output rows verified")
}

func TestWorkflowMetricsDistinguishScheduledAndObserved(t *testing.T) {
	c := casefile.Case{Target: "browser-lifecycle"}
	lifecycle := casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "unchanged"})
	c.Expected.Client = &casefile.ClientTransition{Lifecycle: lifecycle}
	result := Result{Kind: "pass", Checks: map[string]CheckMetric{
		"interp":   {Status: "pass"},
		"compiled": {Status: "failed", BrowserSelected: true, WorkflowObserved: 3, WorkflowVerified: 2},
	}}
	partial := *lifecycle
	partial.States = slices.Clone(lifecycle.States[:3])
	partial.States[2].Connected++
	result.Observations = map[string]runner.Observation{"compiled": {Client: &casefile.ClientTransition{Lifecycle: &partial}}}
	var metrics RunMetrics
	metrics.Observe(&c, &result, false)
	for index, step := range lifecycle.States {
		name := "navigation/browser/" + step.Step
		assert.Equal(t, TransitionCounts{Planned: 1}, metrics.Checks["interp"].Workflows[name], "%s falsely credited off-mode browser", name)
		want := TransitionCounts{Planned: 1}
		if index < 3 {
			want.Executed, want.Reached, want.Observed = 1, 1, 1
		}
		if index < 2 {
			want.Verified = 1
		}
		assert.Equal(t, want, metrics.Checks["compiled"].Workflows[name], name)
	}
}

func TestLifecycleDriverFailureMetricsCreditOnlyVerifiedPrefix(t *testing.T) {
	c := casefile.Case{Target: "browser-lifecycle"}
	lifecycle := casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "local"})
	c.Expected.Client = &casefile.ClientTransition{Lifecycle: lifecycle}
	actual := casefile.ExpectedLifecycle(lifecycle.Input)
	actual.States = actual.States[:3]
	actual.States[2].Prompts++
	actual.FailedStep, actual.DriverError = lifecycle.States[3].Step, "local forced failure"
	observation := runner.Observation{Counts: []string{"0", "1"}, Client: &casefile.ClientTransition{Before: []casefile.ClientInstance{{}}, After: []casefile.ClientInstance{{}}, Lifecycle: actual}}
	check := CheckMetric{BrowserSelected: true}
	require.NoError(t, check.recordObservation(&c, &observation, nil))
	assert.False(t, check.BrowserObserved)
	assert.Equal(t, 3, check.WorkflowObserved)
	assert.Equal(t, 2, check.WorkflowVerified)
	var totals ModeMetrics
	totals.observeWorkflows(&c, &check, runner.WorkflowProofs(&c, &observation))
	assert.Equal(t, TransitionCounts{Planned: 1, Executed: 1, Reached: 1, Observed: 1}, totals.Workflows["navigation/browser/cancel-navigation"])
	assert.Equal(t, TransitionCounts{Planned: 1, Reached: 1}, totals.Workflows["navigation/browser/start-requests"])
}

func TestPartialFormAndEventsUseExpectationFailurePath(t *testing.T) {
	events := casefile.ExpectedEvents(casefile.EventsInput{Label: "x", Delta: 1})
	form := casefile.NewFormWorkflow("x", "y", false)
	for _, name := range []string{"form", "events"} {
		t.Run(name, func(t *testing.T) {
			var c casefile.Case
			observation := runner.Observation{ContentType: "text/html", MetadataTitle: "Piko Page"}
			switch name {
			case "events":
				c.Expected.Client = &casefile.ClientTransition{Events: events}
				partial := *events
				partial.States = events.States[:1]
				partial.DriverError = "synthetic browser stop"
				observation.Client = &casefile.ClientTransition{Events: &partial}
			case "form":
				c.Expected.FormWorkflow = form
				observation.FormWorkflow = &casefile.FormWorkflowObservation{Initial: casefile.EmptyFormState(), PartialPath: "/_piko/partials/task-list", DriverError: "synthetic browser stop"}
			}
			check := CheckMetric{Status: "pass", BrowserSelected: true}
			require.NoError(t, check.recordObservation(&c, &observation, nil), "receipt was classified as a failed worker")
			assert.False(t, check.BrowserObserved)
			located, ok := errors.AsType[subjectError](runner.Check(&c, &observation))
			require.True(t, ok, "failure has no expectation subject")
			assert.NotEmpty(t, located.FailureSubject())
		})
	}
}

func TestPartialComponentJourneyCreditsOnlyACompleteTrace(t *testing.T) {
	expected := casefile.ExpectedPartialComponents(casefile.PartialComponentsInput{Labels: [2]string{"a", "b"}, Edited: "c", Query: "q", Locale: "en", Score: 1})
	require.Greater(t, len(expected.States), 1)
	c := casefile.Case{Target: "partial-components", Expected: casefile.Expected{Client: &casefile.ClientTransition{PartialComponents: expected}}}

	check := CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.False(t, check.recordPartialComponents(&c, &runner.Observation{}))
	assert.False(t, check.BrowserObserved, "a missing trace earned browser credit")

	partial := *expected
	partial.States = slices.Clone(expected.States[:1])
	check = CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.True(t, check.recordPartialComponents(&c, &runner.Observation{Client: &casefile.ClientTransition{PartialComponents: &partial}}))
	assert.False(t, check.BrowserObserved)
	assert.Equal(t, 1, check.WorkflowObserved)

	complete := *expected
	check = CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.False(t, check.recordPartialComponents(&c, &runner.Observation{Client: &casefile.ClientTransition{PartialComponents: &complete}}))
	assert.True(t, check.BrowserObserved)
	assert.Equal(t, len(expected.States), check.WorkflowObserved)

	unselected := CheckMetric{BrowserObserved: true}
	assert.False(t, unselected.recordPartialComponents(&c, &runner.Observation{}))
	assert.True(t, unselected.BrowserObserved, "an unselected browser check was rewritten")
}

func TestCheckoutJourneyCreditsOnlyACompleteTrace(t *testing.T) {
	expected := casefile.ExpectedCheckout(casefile.CheckoutInput{Customer: "c", SKU: "s", Quantity: 2, Unit: 3})
	require.NotEmpty(t, expected.States)
	c := casefile.Case{Target: "checkout-workflow", Expected: casefile.Expected{Client: &casefile.ClientTransition{Checkout: expected}}}

	check := CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.False(t, check.recordCheckout(&c, &runner.Observation{}))
	assert.False(t, check.BrowserObserved)

	failed := *expected
	failed.DriverError = "synthetic browser stop"
	check = CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.True(t, check.recordCheckout(&c, &runner.Observation{Client: &casefile.ClientTransition{Checkout: &failed}}))
	assert.False(t, check.BrowserObserved)
	assert.Equal(t, len(expected.States), check.WorkflowObserved)

	complete := *expected
	check = CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.False(t, check.recordCheckout(&c, &runner.Observation{Client: &casefile.ClientTransition{Checkout: &complete}}))
	assert.True(t, check.BrowserObserved)
}

func TestDOMPlanRecordsEngineAndVerification(t *testing.T) {
	plan := &casefile.DOMPlan{Path: "/dom", ReadyElement: "main", Steps: []casefile.DOMStep{
		{ID: "first", Op: "read", Fields: []casefile.DOMField{{ID: "title"}}},
		{ID: "second", Op: "read"},
	}}
	c := casefile.Case{Target: "components", Expected: casefile.Expected{DOM: plan}}

	check := CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.False(t, check.recordDOM(&c, &runner.Observation{}))
	assert.False(t, check.BrowserObserved)

	trace := &casefile.DOMTrace{Engine: "firefox", Version: "140", DriverError: "synthetic browser stop", States: []casefile.DOMState{{Step: "first"}}}
	check = CheckMetric{BrowserSelected: true, BrowserObserved: true}
	assert.True(t, check.recordDOM(&c, &runner.Observation{Client: &casefile.ClientTransition{DOM: trace}}))
	assert.False(t, check.BrowserObserved)
	assert.Equal(t, "firefox", check.BrowserEngine)
	assert.Equal(t, "140", check.BrowserVersion)
	assert.Equal(t, 1, check.WorkflowObserved)
	assert.NotEmpty(t, check.Verification, "an incomplete DOM trace verified")

	unselected := CheckMetric{}
	assert.False(t, unselected.recordDOM(&c, &runner.Observation{Client: &casefile.ClientTransition{DOM: trace}}))
	assert.Empty(t, unselected.BrowserEngine)
}

func TestRecordVerificationKeepsTheFirstFailure(t *testing.T) {
	var check CheckMetric
	check.recordVerification(nil)
	assert.Empty(t, check.Verification)
	check.recordVerification(errors.New("first"))
	check.recordVerification(errors.New("second"))
	assert.Equal(t, "first", check.Verification)
}

func TestModeMetricsAccumulateEvidenceSensitivityAndEditions(t *testing.T) {
	c := casefile.Case{Target: "content-editions", Expected: casefile.Expected{Editions: []casefile.Edition{{ID: "one"}, {ID: "two"}}}}
	check := CheckMetric{
		Status:                  checkStatusPass,
		BrowserEngine:           "firefox",
		BrowserVersion:          "140",
		PrimaryBrowserVersion:   "130",
		GeneratedArtifactChecks: []string{"manifest", "manifest"},
		GoHost:                  map[string]runner.HostEvidence{"cache": {Planned: 2, Executed: 2, Verified: 1}},
		Mutations: []runner.MutationCheck{
			{Name: "control", Control: true, Passed: true},
			{Name: "drop-row", Passed: true},
			{Name: "swap-row", Passed: false},
		},
		Editions:      []runner.EditionReceipt{{ID: "one", Reached: true, Verified: true}, {ID: "two", Reached: true}},
		Cost:          resource.PhaseCost{CPUKnown: true, CPUUsec: 2_000_000, WorkDirKnown: true, EndWorkDirBytes: 4096},
		Memory:        resource.PhaseMemory{SampledPeakBytes: 3 * mebibyte, Samples: 1},
		BrowserRoutes: map[string]runner.BrowserRouteEvidence{"/page?x=1": {Role: "primary", Selected: true, Visited: true, Verified: true}, "/other?y=2": {Role: "secondary"}},
	}
	var metrics RunMetrics
	metrics.Observe(&c, &Result{Kind: checkStatusPass, Elapsed: time.Hour, Checks: map[string]CheckMetric{modeInterp: check}}, false)
	metrics.Observe(&c, &Result{Kind: checkStatusPass, Elapsed: time.Hour, Checks: map[string]CheckMetric{modeInterp: check}}, false)

	got := metrics.Checks[modeInterp]
	assert.Equal(t, map[string]uint64{"manifest": 4}, got.GeneratedArtifactChecks)
	assert.Equal(t, map[string]uint64{"firefox 140": 2, "chromium 130": 2}, got.BrowserEngines)
	assert.Equal(t, runner.HostEvidence{Planned: 4, Executed: 4, Verified: 2}, got.GoHost["cache"])
	require.NotNil(t, got.Sensitivity)
	assert.Equal(t, uint64(2), got.Sensitivity.Cases)
	assert.Equal(t, uint64(4_000_000), got.Sensitivity.CPUUsec)
	assert.Equal(t, uint64(3*mebibyte), got.Sensitivity.MaxSampledMemoryBytes)
	assert.Equal(t, MutationCounts{Control: true, Attempts: 2, ExpectedResults: 2}, got.Sensitivity.Checks["control"])
	assert.Equal(t, MutationCounts{Attempts: 2}, got.Sensitivity.Checks["swap-row"])
	assert.Equal(t, TransitionCounts{Planned: 2, Executed: 2, Verified: 2}, got.Editions["one"])
	assert.Equal(t, TransitionCounts{Planned: 2, Executed: 2}, got.Editions["two"])
	assert.Equal(t, BrowserRouteCounts{Eligible: 2, Selected: 2, Visited: 2, Verified: 2}, got.BrowserRoutes["/page"])
	assert.Equal(t, BrowserRouteCounts{Eligible: 2}, got.BrowserRoutes["/other?y=2"])
	assert.Equal(t, uint64(2), got.MeasuredWorkDirs)
	assert.Equal(t, uint64(4096), got.LargestEndWorkDirBytes)

	summary := metrics.Summary()
	for _, want := range []string{
		"  cache: 4 planned, 4 observed, 2 verified",
		"browser routes 4 eligible/2 selected/2 visited/2 verified",
		"4s cgroup CPU over 2 checks",
		"killed 2/4 observation mutations; 2/2 clean controls passed, 0.5 killed/check CPU-second",
		"largest end-of-check work directory 4096 bytes",
		"2 cases, 0 confirmed fingerprints, 1.0 cases/hour of case execution",
	} {
		assert.Contains(t, summary, want)
	}
}

func TestModeMetricsCountEveryCheckStatus(t *testing.T) {
	var metrics ModeMetrics
	for _, status := range []string{checkStatusPass, StatusFailed, "running", "incomplete", "pending", "skipped", "mystery"} {
		metrics.observeStatus("package", &CheckMetric{Status: status})
	}
	metrics.observeStatus(modeCompiled, &CheckMetric{Status: checkStatusPass, FrontendRejected: true})
	assert.Equal(t, uint64(8), metrics.Selected)
	assert.Equal(t, uint64(5), metrics.Attempted)
	assert.Equal(t, uint64(3), metrics.Completed)
	assert.Equal(t, uint64(1), metrics.Failed)
	assert.Equal(t, uint64(2), metrics.Incomplete)
	assert.Equal(t, uint64(2), metrics.Skipped)
	assert.Equal(t, uint64(1), metrics.Unknown)
	assert.Equal(t, uint64(1), metrics.FrontendRejected)
}

func TestSummaryReportsWorkflowMatrixAndProgramTotals(t *testing.T) {
	metrics := RunMetrics{
		Cases:         1,
		InvocationCPU: CampaignCPU{Usec: 1500, MeasuredInvocations: 1},
		Checks: map[string]ModeMetrics{modeCompiled: {
			Workflows: map[string]TransitionCounts{
				"form/browser/add":    {Planned: 2, Executed: 2, Reached: 2, Observed: 2, Verified: 1},
				"form/browser/remove": {Planned: 1, LegacyVerified: 1},
			},
			Matrices: map[string]*MatrixEvidence{"locale": {Feasible: 3, Cells: map[string]CellEvidence{
				"en": {Generated: 1, Selected: 1, Reached: 1, Observed: 1, Verified: 1},
				"fr": {Generated: 1, LegacyVerified: 1},
			}}},
			ProgramRequests:               2,
			VerifiedProgramRequests:       1,
			LegacyVerifiedProgramRequests: 1,
		}},
	}

	summary := metrics.Summary()

	for _, want := range []string{
		"2ms whole-campaign cgroup CPU",
		"workflow transitions: 3 planned, 2 executed, 1 independently verified, 2 reached and 2 observed, 1 historical prefix verifications",
		"locale: 2 generated, 1 selected, 1 reached, 1 observed, 1/3 distinct cells verified, 2 outstanding, 1 cells with historical whole-mode credit",
		"program model: 1/2 requests",
		"historical whole-mode credit 1 requests and 0 rows",
	} {
		assert.Contains(t, summary, want)
	}
	assert.NotContains(t, summary, "sampled cgroup memory.current peak")
}

func TestMemoryPeakReportsUnmeasuredChecks(t *testing.T) {
	assert.Equal(t, "unmeasured", memoryPeak(&MemoryMetrics{}))
	assert.Equal(t, "2.0 MiB (1 checks, 3 samples)", memoryPeak(&MemoryMetrics{MeasuredChecks: 1, Samples: 3, MaxSampledBytes: 2 * mebibyte}))
}

func TestWorkflowSequencesListEveryModelledJourney(t *testing.T) {
	client := &casefile.ClientTransition{
		Checkout:          casefile.ExpectedCheckout(casefile.CheckoutInput{Customer: "c", SKU: "s", Quantity: 1, Unit: 1}),
		PartialComponents: casefile.ExpectedPartialComponents(casefile.PartialComponentsInput{Labels: [2]string{"a", "b"}, Edited: "c"}),
		Events:            casefile.ExpectedEvents(casefile.EventsInput{Label: "x", Delta: 1}),
		Lifecycle:         casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "unchanged"}),
	}
	c := casefile.Case{Target: "components", Expected: casefile.Expected{
		DOM:          &casefile.DOMPlan{Steps: []casefile.DOMStep{{ID: "open"}}},
		FormWorkflow: casefile.NewFormWorkflow("x", "y", false),
		Client:       client,
	}}

	sequences := workflowSequences(&c)

	require.Len(t, sequences, 6)
	assert.Equal(t, []string{"components/browser/open"}, sequences[0])
	prefixes := []string{"form/browser/", "checkout/browser/", "partial-components/browser/", "events/browser/", "navigation/browser/"}
	for index, prefix := range prefixes {
		require.NotEmpty(t, sequences[index+1])
		for _, name := range sequences[index+1] {
			assert.True(t, strings.HasPrefix(name, prefix), "%s lacks %s", name, prefix)
		}
	}
}
