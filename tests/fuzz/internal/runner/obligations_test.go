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
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestShimProofsRequireMatchedResponses(t *testing.T) {
	testCase := generate.New("auth-contracts", 7, 0)
	page, action := testCase.Expected.Auth.Steps[0], testCase.Expected.Auth.Actions[0]
	o := Observation{Auth: &AuthReceipt{
		Steps:   []AuthObservation{{Name: page.Name, Actor: page.Actor, Path: page.Path, Response: page.Want}},
		Actions: []AuthActionObservation{{Name: action.Name, Actor: action.Actor, Result: action.Want, Ledger: action.Ledger, LedgerStatus: http.StatusOK}},
	}}
	assert.Equal(t, Verified, AuthPageProof(&testCase, &o, page.Name), "complete observations lost credit")
	assert.Equal(t, Verified, AuthActionProof(&testCase, &o, action.Name), "complete observations lost credit")
	o.Auth.Steps[0].Actor = "other"
	o.Auth.Actions[0].Ledger.Writes++
	assert.Equal(t, Unseen, AuthPageProof(&testCase, &o, page.Name), "wrong actor received credit")
	assert.Equal(t, Observed, AuthActionProof(&testCase, &o, action.Name), "wrong write received credit")
	o.Auth.Actions[0].LedgerStatus = 0
	assert.Equal(t, Reached, AuthActionProof(&testCase, &o, action.Name), "absent ledger received credit")

	testCase = generate.New("form-shim-contracts", 7, 0)
	step := testCase.Expected.FormShim.Steps[0]
	o = Observation{FormShim: &FormShimReceipt{Steps: []FormShimRead{{Name: step.Name, Status: step.Status, Code: step.Code, Write: step.Write, After: step.After}}}}
	assert.Equal(t, Verified, FormShimProof(&testCase, &o, step.Name), "complete form observation lost credit")
	o.FormShim.Steps[0].After = casefile.FormShimState{}
	assert.Equal(t, Observed, FormShimProof(&testCase, &o, step.Name), "missing state received credit")
	assert.Equal(t, Unseen, FormShimProof(&testCase, &o, testCase.Expected.FormShim.Steps[1].Name), "missing attempt received credit")
}

func modelledObservation(c *casefile.Case) Observation {
	o := Observation{Mode: modeInterpreted}
	if model := c.Expected.Client; model != nil {
		client := *model
		client.BrowserRoutes = []casefile.BrowserRouteVisit{{Path: c.Request, Landed: c.Request, Status: c.Expected.Status}}
		if client.Events != nil {
			trace := *client.Events
			trace.BrowserPath = "/events"
			client.Events = &trace
		}
		o.Client = &client
	}
	if plan := c.Expected.DOM; plan != nil {
		if o.Client == nil {
			o.Client = &casefile.ClientTransition{}
		}
		o.Client.DOM = &casefile.DOMTrace{Engine: "chromium", Version: "test"}
		for _, step := range plan.Steps {
			state := casefile.DOMState{Step: step.ID, Path: plan.Path, Values: map[string][]string{}}
			for _, field := range step.Fields {
				state.Values[field.ID] = slices.Clone(field.Want)
			}
			o.Client.DOM.States = append(o.Client.DOM.States, state)
		}
	}
	for _, probe := range c.Probes {
		o.Probes = append(o.Probes, ProbeObservation{Path: probe.Path, Actor: probe.Actor, Status: probe.Status, Location: probe.Location, Text: probe.Text, Program: probe.Program, ProgramRoot: probe.ProgramRoot, Output: slices.Clone(probe.Output), Semantic: probe.Semantic, ResponseHeaders: probe.ResponseHeaders})
	}
	return o
}

func TestWorkflowProofsCreditModelledJourneys(t *testing.T) {
	for _, target := range []string{"markup", "browser-events", "browser-lifecycle", "partial-components", "checkout-workflow"} {
		t.Run(target, func(t *testing.T) {
			c := generate.New(target, 7, 0)
			o := modelledObservation(&c)

			proofs := WorkflowProofs(&c, &o)

			require.NotEmpty(t, proofs)
			for index, proof := range proofs {
				assert.Equal(t, Verified, proof, "step %d", index)
			}
			assert.Equal(t, Verified, aggregateProof(proofs))
		})
	}
}

func TestWorkflowProofsDemoteACorruptedState(t *testing.T) {
	c := generate.New("browser-lifecycle", 7, 0)
	o := modelledObservation(&c)
	trace := *o.Client.Lifecycle
	trace.States = slices.Clone(trace.States)
	trace.States[1].Draft += "changed"
	o.Client.Lifecycle = &trace

	proofs := WorkflowProofs(&c, &o)

	assert.Equal(t, Verified, proofs[0])
	assert.Equal(t, Observed, proofs[1])
	assert.Equal(t, Observed, BrowserHistoryProof(&c, &o, proofs))
}

func TestBrowserHistoryProofVerifiesModelledHistories(t *testing.T) {
	for _, target := range []string{"browser-events", "browser-lifecycle", "partial-components"} {
		t.Run(target, func(t *testing.T) {
			c := generate.New(target, 7, 0)
			o := modelledObservation(&c)

			assert.Equal(t, Verified, BrowserHistoryProof(&c, &o, WorkflowProofs(&c, &o)))
		})
	}
}

func TestBrowserHistoryProofWithoutEvidence(t *testing.T) {
	c := generate.New("browser-events", 7, 0)

	assert.Equal(t, Unseen, BrowserHistoryProof(&c, &Observation{}, nil))
	assert.Equal(t, Unseen, aggregateProof(nil))
	assert.Equal(t, Unseen, aggregateProof([]Proof{Unseen, Unseen}))
	assert.Equal(t, Reached, aggregateProof([]Proof{Verified, Reached}))
}

func TestDOMFamilyProofMeasuresOneFamily(t *testing.T) {
	c := generate.New("markup", 7, 0)
	o := modelledObservation(&c)
	var family string
	for _, step := range c.Expected.DOM.Steps {
		if step.ID == "content-models" {
			id := step.Fields[0].ID
			family = id[:strings.LastIndex(id, "-")]
		}
	}
	require.NotEmpty(t, family)

	assert.Equal(t, Verified, DOMFamilyProof(&c, &o, family))
	assert.Equal(t, Reached, DOMFamilyProof(&c, &o, "absent-family"))
	o.Client.DOM.Engine = "firefox"
	assert.Equal(t, Unseen, DOMFamilyProof(&c, &o, family))
}

func TestProgramProofsCreditMatchingRows(t *testing.T) {
	c := generate.New("pages", 7, 0)
	o := modelledObservation(&c)

	proofs := ProgramProofs(&c, &o)

	for index, probe := range c.Probes {
		if !probe.Program {
			continue
		}
		require.Len(t, proofs[index].Rows, len(probe.Output))
		assert.NotEqual(t, Unseen, proofs[index].Request)
	}
}

func TestComponentHistoryProofFollowsThePrimaryRoute(t *testing.T) {
	c := generate.New("components", 7, 0)
	o := modelledObservation(&c)
	require.NotNil(t, o.Client)

	assert.Equal(t, Verified, BrowserHistoryProof(&c, &o, nil))
	o.Client.BrowserRoutes[0].DriverError = "crashed"
	assert.Equal(t, Reached, BrowserHistoryProof(&c, &o, nil))
	o.Client.BrowserRoutes = nil
	assert.Equal(t, Unseen, BrowserHistoryProof(&c, &o, nil))
}

func TestWorkflowProofsMeasureEachFormCommand(t *testing.T) {
	workflow, got := formCheckFixture(t)
	c := &casefile.Case{Expected: casefile.Expected{FormWorkflow: workflow}}

	proofs := WorkflowProofs(c, &Observation{FormWorkflow: got})

	require.Len(t, proofs, len(workflow.Steps))
	for index, proof := range proofs {
		assert.Equal(t, Verified, proof, "step %d", index)
	}

	got.Steps[1].Command.Value = "other"
	got.Steps[2].FreshRead = false
	got.DriverError, got.FailedStep = "click failed", 1
	proofs = WorkflowProofs(c, &Observation{FormWorkflow: got})
	assert.Equal(t, Reached, proofs[1])
	assert.Equal(t, Reached, proofs[2])

	got.PartialPath = "/elsewhere"
	assert.Equal(t, make([]Proof, len(workflow.Steps)), WorkflowProofs(c, &Observation{FormWorkflow: got}))
	assert.Equal(t, make([]Proof, len(workflow.Steps)), WorkflowProofs(c, &Observation{}))
}
