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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

func TestFiniteWindowsAndRuntimeCredit(t *testing.T) {
	for target, window := range map[string]int{"components": 64, "pages": 4, "graph-rebuild": 32, "markup": 4, "source-diagnostics": 4, "partial-components": 8, "browser-events": 4, "browser-lifecycle": 2} {
		for _, seed := range []uint64{7, 93} {
			var credited, uncredited ModeMetrics
			for index := range uint64(window) {
				c := generate.New(target, seed, index)
				observation := modelledReceipt(c)
				passed := CheckMetric{Status: "pass", BrowserSelected: true}
				credited.observeMatrices(newMatrixProofs(&c, &passed, &observation, runner.ProgramProofs(&c, &observation), runner.WorkflowProofs(&c, &observation)))
				empty := runner.Observation{}
				incomplete := CheckMetric{Status: "incomplete"}
				uncredited.observeMatrices(newMatrixProofs(&c, &incomplete, &empty, runner.ProgramProofs(&c, &empty), nil))
			}
			for name, matrix := range credited.Matrices {
				require.NotZero(t, matrix.Feasible, "%s/%s seed %d", target, name, seed)
				require.Len(t, matrix.Cells, matrix.Feasible, "%s/%s seed %d", target, name, seed)
				for identity, count := range matrix.Cells {
					require.NotZero(t, count.Verified, "%s/%s/%s unverified", target, name, identity)
					require.Zero(t, uncredited.Matrices[name].Cells[identity].Verified, "%s/%s/%s credited without evidence", target, name, identity)
				}
			}
		}
	}
}

func TestObservationReductionPreservesOriginalModel(t *testing.T) {
	c := generate.New("markup", 7, 0)
	original := c.Hash()
	last := c.Expected.DOM.Steps[len(c.Expected.DOM.Steps)-1]
	identity := last.Fields[0].ID
	reduced := c
	shrinkObservations(t.Context(), &reduced, func(candidate *casefile.Case) bool {
		for _, step := range candidate.Expected.DOM.Steps {
			for _, field := range step.Fields {
				if step.ID == last.ID && field.ID == identity {
					return true
				}
			}
		}
		return false
	})
	assert.Equal(t, original, c.Hash(), "source model changed")
	assert.Empty(t, reduced.Probes)
	require.Len(t, reduced.Expected.DOM.Steps, 1)
	assert.Len(t, reduced.Expected.DOM.Steps[0].Fields, 1)
}

func TestLegacyMatrixCountsRemainDistinct(t *testing.T) {
	var evidence CellEvidence
	require.NoError(t, json.Unmarshal([]byte(`{"Generated":2,"Selected":2,"Verified":1}`), &evidence))
	assert.Equal(t, CellEvidence{Generated: 2, Selected: 2, LegacyVerified: 1}, evidence)
	evidence.Reached, evidence.Observed, evidence.Verified = 1, 1, 1
	data, err := json.Marshal(evidence)
	require.NoError(t, err)
	var round CellEvidence
	require.NoError(t, json.Unmarshal(data, &round))
	assert.Equal(t, evidence, round)
}

func TestFailedHTTPCellDoesNotEraseSiblingReceipts(t *testing.T) {
	c := generate.New("pages", 7, 0)
	observation := modelledReceipt(c)
	index := slices.IndexFunc(c.Probes, func(probe casefile.Probe) bool { return probe.Path == "/arithmetic" })
	require.GreaterOrEqual(t, index, 0, "missing matrix")
	want := c.Probes[index]
	observation.Probes[index].Output[1].Text += " wrong"
	observation.Probes = observation.Probes[:index+1]
	result := Result{Kind: "expectation", Checks: map[string]CheckMetric{"interp": {Status: "failed"}}, Observations: map[string]runner.Observation{"interp": observation}}
	var totals RunMetrics
	totals.Observe(&c, &result, false)
	metrics := totals.Checks["interp"]
	cells := metrics.Matrices["pages/"+generate.ArithmeticMatrix].Cells
	for row, output := range want.Output {
		_, identity := generate.ProgramMatrixCell(want, output)
		expected := CellEvidence{Generated: 1, Selected: 1, Reached: 1, Observed: 1, Verified: 1}
		if row == 1 {
			expected.Verified = 0
		}
		assert.Equal(t, expected, cells[identity], "row %d", row)
	}
	for _, cell := range metrics.Matrices["pages/"+generate.TypedMatrix].Cells {
		if cell.Reached == 0 {
			assert.Zero(t, cell.Observed)
			assert.Zero(t, cell.Verified)
		}
	}
	observation.Probes[index].Path = "/other"
	audits := runner.ProgramProofs(&c, &observation)
	assert.Equal(t, runner.Unseen, audits[index].Request, "wrong response credited")
	assert.Empty(t, audits[index].Rows, "wrong response credited")
	observation.Probes[index].Path = want.Path
	observation.Probes[index].Output = observation.Probes[index].Output[:2]
	audit := runner.ProgramProofs(&c, &observation)[index]
	assert.Equal(t, runner.Observed, audit.Request)
	assert.Equal(t, []runner.Proof{runner.Verified, runner.Observed, runner.Reached}, audit.Rows[:3])
	observation = modelledReceipt(c)
	c.Probes[index].ResponseHeaders = map[string][]string{"X-Test": {"yes"}}
	audit = runner.ProgramProofs(&c, &observation)[index]
	assert.Equal(t, runner.Observed, audit.Request)
	assert.Equal(t, runner.Verified, audit.Rows[0], "unrelated header failure erased the row check")
}

func TestFailedBrowserFamilyRetainsIndependentFamiliesAndLaterSteps(t *testing.T) {
	c := generate.New("markup", 7, 0)
	observation := modelledReceipt(c)
	stepIndex := slices.IndexFunc(c.Expected.DOM.Steps, func(step casefile.DOMStep) bool { return step.ID == "content-models" })
	families := generate.Matrices(&c)[0].Cells
	require.GreaterOrEqual(t, stepIndex, 0)
	require.GreaterOrEqual(t, len(families), 2, "missing families: %v", families)
	bad, good := families[0], families[1]
	observation.Client.DOM.States[stepIndex].Values[bad+"-text"] = []string{"shared wrong"}
	check := CheckMetric{Status: "failed", BrowserSelected: true, ProgramVerified: false}
	var metrics RunMetrics
	metrics.Observe(&c, &Result{Checks: map[string]CheckMetric{"interp": check}, Observations: map[string]runner.Observation{"interp": observation}}, false)
	cells := metrics.Checks["interp"].Matrices["markup/HTML content families"].Cells
	assert.Equal(t, uint64(1), cells[bad].Observed)
	assert.Zero(t, cells[bad].Verified)
	assert.Equal(t, uint64(1), cells[good].Verified)
	delete(observation.Client.DOM.States[stepIndex].Values, good+"-tag")
	assert.Equal(t, runner.Reached, runner.DOMFamilyProof(&c, &observation, good), "unread family counted as observed")
	observation.Client.DOM.Engine = "firefox"
	assert.Equal(t, runner.Unseen, runner.DOMFamilyProof(&c, &observation, bad), "unselected engine earned credit")

	workflow := generate.New("browser-lifecycle", 7, 0)
	observation = modelledReceipt(workflow)
	trace := *observation.Client.Lifecycle
	trace.States = slices.Clone(trace.States)
	trace.States[1].Prompts++
	observation.Client.Lifecycle = &trace
	var histories RunMetrics
	histories.Observe(&workflow, &Result{Checks: map[string]CheckMetric{"interp": check, "compiled": {Status: "pass"}}, Observations: map[string]runner.Observation{"interp": observation}}, false)
	modeMetrics := histories.Checks["interp"]
	assert.Zero(t, modeMetrics.Workflows["navigation/browser/"+trace.States[1].Step].Verified)
	assert.Equal(t, uint64(1), modeMetrics.Workflows["navigation/browser/"+trace.States[2].Step].Verified)
	for _, cell := range modeMetrics.Matrices["browser-lifecycle/request release order"].Cells {
		assert.Equal(t, uint64(1), cell.Observed)
		assert.Zero(t, cell.Verified, "history containing a bad step credited")
	}
}

func TestReportResumeSeparatesHistoricalTotals(t *testing.T) {
	const historical = `{"ProgramRequests":4,"VerifiedProgramRequests":4,"ProgramOutputs":10,"VerifiedProgramOutputs":10,"Workflows":{"old":{"Planned":2,"Executed":2,"Verified":1}}}`
	var totals ModeMetrics
	require.NoError(t, json.Unmarshal([]byte(historical), &totals))
	workflow := totals.Workflows["old"]
	assert.Equal(t, uint64(4), totals.LegacyVerifiedProgramRequests)
	assert.Equal(t, uint64(10), totals.LegacyVerifiedProgramOutputs)
	assert.Zero(t, totals.VerifiedProgramOutputs)
	assert.Equal(t, uint64(2), workflow.LegacyExecuted)
	assert.Equal(t, uint64(1), workflow.LegacyVerified)
	assert.Zero(t, workflow.Reached)
	assert.Zero(t, workflow.Observed)
	totals.ReachedProgramRequests, totals.ObservedProgramRequests, totals.VerifiedProgramRequests = 2, 2, 1
	data, err := json.Marshal(totals)
	require.NoError(t, err)
	var again ModeMetrics
	require.NoError(t, json.Unmarshal(data, &again))
	assert.Equal(t, uint64(4), again.LegacyVerifiedProgramRequests)
	assert.Equal(t, uint64(2), again.ReachedProgramRequests)
	assert.Equal(t, uint64(1), again.VerifiedProgramRequests)
	assert.Equal(t, workflow, again.Workflows["old"])
}

func TestGraphCellUsesItsOwnRequests(t *testing.T) {
	c := generate.New("graph-rebuild", 7, 0)
	observation := modelledReceipt(c)
	check := CheckMetric{Status: "failed"}
	record := func() CellEvidence {
		t.Helper()
		var metrics ModeMetrics
		metrics.observeCaseObligations(&c, "interp", &check, &observation)
		cells := metrics.Matrices["graph-rebuild/partial graph and edit"].Cells
		require.NotEmpty(t, cells, "missing graph matrix")
		for _, cell := range cells {
			return cell
		}
		return CellEvidence{}
	}
	assert.Equal(t, uint64(1), record().Verified, "unrelated mode failure erased verified graph")
	index := slices.IndexFunc(c.Probes, func(probe casefile.Probe) bool { return probe.Path == "/twin?run=second" })
	require.GreaterOrEqual(t, index, 0)
	observation.Probes[index].Output[0].Text = "wrong edit"
	edited := record()
	assert.Equal(t, uint64(1), edited.Observed)
	assert.Zero(t, edited.Verified, "bad twin passed")
	observation.Probes = observation.Probes[:index]
	missing := record()
	assert.Equal(t, uint64(1), missing.Reached)
	assert.Zero(t, missing.Observed, "missing twin passed")
}

func modelledReceipt(c casefile.Case) runner.Observation {
	observation := runner.Observation{Mode: "interp", Client: c.Expected.Client}
	if observation.Client != nil {
		client := *observation.Client
		client.BrowserRoutes = []casefile.BrowserRouteVisit{{Path: c.Request, Landed: c.Request, Status: c.Expected.Status}}
		observation.Client = &client
		if client.Events != nil {
			trace := *client.Events
			trace.BrowserPath = "/events"
			client.Events = &trace
		}
	}
	if plan := c.Expected.DOM; plan != nil {
		if observation.Client == nil {
			observation.Client = &casefile.ClientTransition{}
		}
		engine := "chromium"
		if c.Target == "firefox-markup" {
			engine = "firefox"
		}
		observation.Client.DOM = &casefile.DOMTrace{Engine: engine, Version: "test"}
		for _, step := range plan.Steps {
			state := casefile.DOMState{Step: step.ID, Path: plan.Path, Values: map[string][]string{}}
			for _, field := range step.Fields {
				state.Values[field.ID] = slices.Clone(field.Want)
			}
			observation.Client.DOM.States = append(observation.Client.DOM.States, state)
		}
	}
	for _, probe := range c.Probes {
		observation.Probes = append(observation.Probes, runner.ProbeObservation{Path: probe.Path, Actor: probe.Actor, Status: probe.Status, Location: probe.Location, Text: probe.Text, Program: probe.Program, ProgramRoot: probe.ProgramRoot, Output: slices.Clone(probe.Output), Semantic: probe.Semantic, ResponseHeaders: probe.ResponseHeaders})
	}
	return observation
}
