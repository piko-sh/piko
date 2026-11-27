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
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// Proof records the strongest independently measured stage at a saved model location.
type Proof uint8

const (
	// Unseen marks an operation without worker evidence.
	Unseen Proof = iota

	// Reached marks an operation reached by the worker.
	Reached

	// Observed marks a receipt at the expected operation location.
	Observed

	// Verified marks an operation whose independent contract passed.
	Verified
)

const (
	// domFamilyFieldCount is the number of content-model fields one DOM family keeps.
	domFamilyFieldCount = 3
)

// ProgramProof records proof stages for a request and its output rows.
type ProgramProof struct {
	// Rows holds the proof stage for each output row.
	Rows []Proof

	// Request is the owning request's measured proof stage.
	Request Proof
}

// ProgramProofs uses the ordered HTTP worker receipts.
//
// An observed row must carry the intended output ID at its expected position. Every
// checked field must agree for verification. A sibling row can fail without erasing that
// evidence.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []ProgramProof which lists program proof values in their recorded order.
func ProgramProofs(c *casefile.Case, o *Observation) []ProgramProof {
	out := make([]ProgramProof, len(c.Probes))
	for probeIndex := range c.Probes {
		probe := &c.Probes[probeIndex]
		if !probe.Program || probeIndex >= len(o.Probes) {
			continue
		}
		out[probeIndex] = programProof(c, o.Mode, probe, &o.Probes[probeIndex])
	}
	return out
}

// programProof measures one program request and its output rows.
//
// Status, location and root identify the document that owns a row. An unrelated header or
// page metadata error belongs to the request oracle.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes mode (string) which selects the server mode.
// Takes probe (*casefile.Probe) which holds the saved program request.
// Takes got (*ProbeObservation) which holds the observed response.
//
// Returns ProgramProof which records the request and row proof stages.
func programProof(c *casefile.Case, mode string, probe *casefile.Probe, got *ProbeObservation) ProgramProof {
	var proof ProgramProof
	if got.Path != probe.Path || got.Actor != probe.Actor || got.Status <= 0 {
		return proof
	}
	proof.Request = Reached
	if !got.Program {
		return proof
	}
	proof.Request = Observed
	proof.Rows = make([]Proof, len(probe.Output))
	status := probe.Status
	modeOK := true
	if len(probe.ByMode) > 0 {
		var contract casefile.ModeResponse
		contract, modeOK = probe.ByMode[mode]
		status = contract.Status
	}
	valid := modeOK && got.Status == status && got.Location == probe.Location && (probe.ProgramRoot == "" || (got.ProgramRoot == probe.ProgramRoot && !got.RootHidden))
	envelope := *probe
	envelope.Program = false
	all := valid && len(probe.Output) == len(got.Output) && checkProbe(c, mode, envelope, got) == nil
	for rowIndex := range probe.Output {
		stage := programRowProof(probe, rowIndex, got, valid)
		proof.Rows[rowIndex] = stage
		all = all && stage == Verified
	}
	if all {
		proof.Request = Verified
	}
	return proof
}

// programRowProof measures one program output row.
//
// Takes probe (*casefile.Probe) which holds the saved program request.
// Takes rowIndex (int) which is the row's position.
// Takes got (*ProbeObservation) which holds the observed response.
// Takes valid (bool) which reports whether the row's owning document matched.
//
// Returns Proof which is Reached, Observed or Verified.
func programRowProof(probe *casefile.Probe, rowIndex int, got *ProbeObservation, valid bool) Proof {
	want := probe.Output[rowIndex]
	if rowIndex >= len(got.Output) || want.ID == "" || got.Output[rowIndex].ID != want.ID {
		return Reached
	}
	if valid && checkProgramOutput(*probe, rowIndex, want, got.Output[rowIndex]) == nil {
		return Verified
	}
	return Observed
}

// DOMFamilyProof measures proof for one DOM family independently of sibling families.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes id (string) which identifies the DOM family to measure.
//
// Returns Proof which is the strongest independently measured proof for the selected DOM
// family.
func DOMFamilyProof(c *casefile.Case, o *Observation, id string) Proof {
	plan := c.Expected.DOM
	if plan == nil || o.Client == nil || o.Client.DOM == nil {
		return Unseen
	}
	trace := o.Client.DOM
	if checkDOMEngine(c, trace) != nil {
		return Unseen
	}
	for index := range plan.Steps {
		step := &plan.Steps[index]
		if step.ID != "content-models" || index >= len(trace.States) {
			continue
		}
		state := &trace.States[index]
		if state.Step != step.ID || state.Path != plan.Path {
			return Unseen
		}
		return domFamilyStepProof(step, state.Values, id)
	}
	return Unseen
}

// domFamilyStepProof measures one DOM family's fields within the content-models step.
//
// A source reduction can remove parts of a family's contract, so the family must keep all
// of its fields to count.
//
// Takes step (*casefile.DOMStep) which holds the step's expected fields.
// Takes values (map[string][]string) which maps observed field identifiers to values.
// Takes id (string) which identifies the DOM family.
//
// Returns Proof which is Reached, Observed or Verified.
func domFamilyStepProof(step *casefile.DOMStep, values map[string][]string, id string) Proof {
	matched := 0
	stage := Verified
	for _, field := range step.Fields {
		if !strings.HasPrefix(field.ID, id+"-") && (id != "multimedia" || field.ID != "media-defaults") {
			continue
		}
		matched++
		value, ok := values[field.ID]
		if !ok {
			return Reached
		}
		if !slices.Equal(value, field.Want) {
			stage = Observed
		}
	}
	need := domFamilyFieldCount
	if id == "multimedia" {
		need++
	}
	if matched != need {
		return Reached
	}
	return stage
}

// snapshotProof classifies a snapshot by its location and matched fields.
//
// Takes location (bool) which reports whether the snapshot reached the expected location.
// Takes matched (bool) which reports whether the observed fields match their model.
//
// Returns Proof which is Unseen, Observed or Verified according to location and field
// agreement.
func snapshotProof(location, matched bool) Proof {
	if !location {
		return Unseen
	}
	if matched {
		return Verified
	}
	return Observed
}

// WorkflowProofs follows the saved action indices, without assuming later snapshots are
// wrong because an earlier property failed.
//
// An explicit driver failure identifies a reached step even when no complete snapshot was
// saved.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []Proof which lists proof values in their recorded order.
func WorkflowProofs(c *casefile.Case, o *Observation) []Proof {
	if plan := c.Expected.DOM; plan != nil {
		return domWorkflowProofs(c, plan, o.Client)
	}
	if workflow := c.Expected.FormWorkflow; workflow != nil {
		return formWorkflowProofs(c, workflow, o.FormWorkflow)
	}
	model, actual := c.Expected.Client, o.Client
	if model == nil || actual == nil {
		return nil
	}
	switch {
	case model.Events != nil && actual.Events != nil:
		return eventProofs(c, model.Events, actual.Events)
	case model.Lifecycle != nil && actual.Lifecycle != nil:
		return lifecycleProofs(model.Lifecycle, actual.Lifecycle)
	case model.PartialComponents != nil && actual.PartialComponents != nil:
		return partialComponentProofs(model.PartialComponents, actual.PartialComponents)
	case model.Checkout != nil && actual.Checkout != nil:
		return checkoutProofs(model.Checkout, actual.Checkout)
	}
	return nil
}

// domWorkflowProofs measures each DOM step independently.
//
// Takes c (*casefile.Case) which holds the saved case.
// Takes plan (*casefile.DOMPlan) which holds the saved DOM operations.
// Takes client (*casefile.ClientTransition) which holds the browser observation, if any.
//
// Returns []Proof which lists the step proofs in plan order.
func domWorkflowProofs(c *casefile.Case, plan *casefile.DOMPlan, client *casefile.ClientTransition) []Proof {
	out := make([]Proof, len(plan.Steps))
	if client == nil || client.DOM == nil || checkDOMEngine(c, client.DOM) != nil {
		return out
	}
	states := client.DOM.States
	for index := range min(len(out), len(states)) {
		got, step := &states[index], &plan.Steps[index]
		if got.Step != step.ID || got.Path != plan.Path {
			continue
		}
		out[index] = domStepProof(step, got.Values)
	}
	return out
}

// domStepProof measures one DOM step's observed fields.
//
// Takes step (*casefile.DOMStep) which holds the step's expected fields.
// Takes values (map[string][]string) which maps observed field identifiers to values.
//
// Returns Proof which is Reached, Observed or Verified.
func domStepProof(step *casefile.DOMStep, values map[string][]string) Proof {
	stage := Verified
	for _, field := range step.Fields {
		value, ok := values[field.ID]
		if !ok {
			return Reached
		}
		if !slices.Equal(value, field.Want) {
			stage = Observed
		}
	}
	return stage
}

// formWorkflowProofs measures each form workflow command independently.
//
// Takes c (*casefile.Case) which holds the saved case.
// Takes workflow (*casefile.FormWorkflow) which holds the saved commands.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed journey, if any.
//
// Returns []Proof which lists the command proofs in recipe order.
func formWorkflowProofs(c *casefile.Case, workflow *casefile.FormWorkflow, got *casefile.FormWorkflowObservation) []Proof {
	out := make([]Proof, len(workflow.Steps))
	if got == nil || (c.BrowserSampleVersion > 0 && got.BrowserPath != "/workflow") || !strings.HasPrefix(got.PartialPath, "/_piko/partials/") {
		return out
	}
	want, err := workflow.Model()
	if err != nil {
		return out
	}
	for index := range min(len(out), len(got.Steps), len(want)) {
		step := &got.Steps[index]
		if workflow.Steps[index] != step.Command {
			continue
		}
		out[index] = Reached
		if step.FreshRead && step.StoreRead {
			out[index] = snapshotProof(true, checkFormStep(workflow.Steps[index], want[index], *step, got.PartialPath) == nil)
		}
	}
	if got.DriverError != "" && got.FailedStep >= 0 && got.FailedStep < len(out) {
		out[got.FailedStep] = Reached
	}
	return out
}

// eventProofs measures each browser event state independently.
//
// Takes c (*casefile.Case) which holds the saved case.
// Takes want (*casefile.EventsTrace) which holds the expected journey.
// Takes got (*casefile.EventsTrace) which holds the observed journey.
//
// Returns []Proof which lists the state proofs in journey order.
func eventProofs(c *casefile.Case, want, got *casefile.EventsTrace) []Proof {
	proofs := make([]Proof, len(want.States))
	if want.Input != got.Input || ((c.BrowserSampleVersion > 0 || got.BrowserPath != "") && got.BrowserPath != "/events") {
		return proofs
	}
	for index := range min(len(proofs), len(got.States)) {
		state, expected := &got.States[index], &want.States[index]
		proofs[index] = snapshotProof(state.Step == expected.Step, *state == *expected)
	}
	return proofs
}

// lifecycleProofs measures each lifecycle state independently.
//
// Takes want (*casefile.LifecycleTrace) which holds the expected journey.
// Takes got (*casefile.LifecycleTrace) which holds the observed journey.
//
// Returns []Proof which lists the state proofs in journey order.
func lifecycleProofs(want, got *casefile.LifecycleTrace) []Proof {
	proofs := make([]Proof, len(want.States))
	if want.Input != got.Input {
		return proofs
	}
	for index := range min(len(proofs), len(got.States)) {
		state, expected := &got.States[index], &want.States[index]
		proofs[index] = snapshotProof(state.Step == expected.Step && state.Path == expected.Path, *state == *expected)
	}
	if next := len(got.States); next < len(proofs) && got.FailedStep == want.States[next].Step {
		proofs[next] = Reached
	}
	return proofs
}

// partialComponentProofs measures each partial-component state independently.
//
// Takes want (*casefile.PartialComponentsTrace) which holds the expected journey.
// Takes got (*casefile.PartialComponentsTrace) which holds the observed journey.
//
// Returns []Proof which lists the state proofs in journey order.
func partialComponentProofs(want, got *casefile.PartialComponentsTrace) []Proof {
	proofs := make([]Proof, len(want.States))
	if want.Input != got.Input {
		return proofs
	}
	for index := range min(len(proofs), len(got.States)) {
		expected, observed := *want, *got
		expected.States, observed.States, observed.DriverError = want.States[index:index+1], got.States[index:index+1], ""
		_, err := checkPartialComponents(&observed, &expected)
		state := &got.States[index]
		proofs[index] = snapshotProof(state.Step == expected.States[0].Step && state.Path == expected.States[0].Path, err == nil)
	}
	return proofs
}

// checkoutProofs measures each checkout state independently.
//
// Takes want (*casefile.CheckoutTrace) which holds the expected journey.
// Takes got (*casefile.CheckoutTrace) which holds the observed journey.
//
// Returns []Proof which lists the state proofs in journey order.
func checkoutProofs(want, got *casefile.CheckoutTrace) []Proof {
	proofs := make([]Proof, len(want.States))
	if want.Input != got.Input {
		return proofs
	}
	for index := range min(len(proofs), len(got.States)) {
		expected, observed := *want, *got
		expected.States, observed.States, observed.DriverError, expected.Documents = want.States[index:index+1], got.States[index:index+1], "", nil
		_, err := checkCheckout(&observed, &expected)
		state := &got.States[index]
		proofs[index] = snapshotProof(state.Step == expected.States[0].Step && state.Path == expected.States[0].Path, err == nil)
	}
	return proofs
}

// BrowserHistoryProof measures proof for the selected browser history.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes steps ([]Proof) which holds ordered proof or interaction observations.
//
// Returns Proof which is the aggregate independently observed proof stage for the
// selected browser history.
func BrowserHistoryProof(c *casefile.Case, o *Observation, steps []Proof) Proof {
	want, got := c.Expected.Client, o.Client
	if want == nil || got == nil {
		return Unseen
	}
	if c.Target == "components" {
		return componentHistoryProof(c, want, got)
	}
	stage := aggregateProof(steps)
	if stage != Verified {
		return stage
	}
	var err error
	switch c.Target {
	case "partial-components":
		_, err = checkPartialComponents(got.PartialComponents, want.PartialComponents)
	case "browser-events":
		err = checkEvents(got.Events, want.Events)
	case "browser-lifecycle", "browser-lifecycle-inline", "browser-element-reuse":
		err = checkLifecycle(got.Lifecycle, want.Lifecycle)
	default:
		return Unseen
	}
	if err != nil {
		return Observed
	}
	return Verified
}

// aggregateProof combines step proofs into the weakest stage a history reached.
//
// Takes steps ([]Proof) which holds the per-step proofs.
//
// Returns Proof which is Unseen without evidence, Reached below Observed, or the weakest
// step stage.
func aggregateProof(steps []Proof) Proof {
	if len(steps) == 0 {
		return Unseen
	}
	stage := Verified
	present := false
	for _, step := range steps {
		stage = min(stage, step)
		present = present || step != Unseen
	}
	if !present {
		return Unseen
	}
	if stage < Observed {
		return Reached
	}
	return stage
}

// componentHistoryProof measures a component case's primary browser history.
//
// Takes c (*casefile.Case) which holds the saved request and expected status.
// Takes want (*casefile.ClientTransition) which holds the expected history.
// Takes got (*casefile.ClientTransition) which holds the observed history.
//
// Returns Proof which is the strongest stage the history reached.
func componentHistoryProof(c *casefile.Case, want, got *casefile.ClientTransition) Proof {
	if c.BrowserSampleVersion > 0 {
		if proof, settled := primaryRouteProof(c, got.BrowserRoutes); settled {
			return proof
		}
	}
	if len(want.Steps) == 0 || len(got.Before) == 0 {
		return Unseen
	}
	if len(got.After) == 0 || len(got.Steps) < len(want.Steps) {
		return Reached
	}
	for index := range want.Steps {
		if got.Steps[index].Action != want.Steps[index].Action || len(got.Steps[index].After) == 0 {
			return Reached
		}
	}
	var expected, observed casefile.ClientTransition
	expected.Before, expected.After, expected.Steps = want.Before, want.After, want.Steps
	observed.Before, observed.After, observed.Steps = got.Before, got.After, got.Steps
	if compareClient(&observed, &expected) == nil {
		return Verified
	}
	return Observed
}

// primaryRouteProof checks the primary browser route of a sampled component case.
//
// Takes c (*casefile.Case) which holds the saved request and expected status.
// Takes routes ([]casefile.BrowserRouteVisit) which lists the browser's visits.
//
// Returns Proof which is the route's stage when it settles the history proof.
// Returns bool which reports whether the route settles the proof on its own.
func primaryRouteProof(c *casefile.Case, routes []casefile.BrowserRouteVisit) (Proof, bool) {
	if len(routes) == 0 {
		return Unseen, true
	}
	route := &routes[0]
	if route.Path != c.Request || route.Landed != c.Request || route.Status <= 0 {
		return Unseen, true
	}
	if route.DriverError != "" || route.Status != c.Expected.Status {
		return Reached, true
	}
	return Unseen, false
}

// AuthPageProof measures proof for one authentication page request.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes id (string) which is the saved authentication step name.
//
// Returns Proof which is the independently measured proof for the authentication page
// step.
func AuthPageProof(c *casefile.Case, o *Observation, id string) Proof {
	if c.Expected.Auth == nil || o.Auth == nil {
		return Unseen
	}
	for index := range c.Expected.Auth.Steps {
		expected := &c.Expected.Auth.Steps[index]
		if expected.Name != id || index >= len(o.Auth.Steps) {
			continue
		}
		observed := &o.Auth.Steps[index]
		if observed.Name != id || expected.Path != observed.Path || expected.Actor != observed.Actor {
			return Unseen
		}
		if observed.Response.Status == 0 {
			return Reached
		}
		return snapshotProof(true, checkAuthResponse(id, expected.Actor, expected.Want, observed.Response) == nil)
	}
	return Unseen
}

// AuthActionProof measures proof for one publication action.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes id (string) which is the saved publication action name.
//
// Returns Proof which is the independently measured proof for the publication action.
func AuthActionProof(c *casefile.Case, o *Observation, id string) Proof {
	if c.Expected.Auth == nil || o.Auth == nil {
		return Unseen
	}
	for index := range c.Expected.Auth.Actions {
		expected := &c.Expected.Auth.Actions[index]
		if expected.Name != id || index >= len(o.Auth.Actions) {
			continue
		}
		observed := &o.Auth.Actions[index]
		if observed.Name != id || observed.Actor != expected.Actor {
			return Unseen
		}
		if observed.Result.Status == 0 || observed.LedgerStatus == 0 {
			return Reached
		}
		return snapshotProof(true, checkPublicationActions(c.Expected.Auth.Actions[index:index+1], o.Auth.Actions[index:index+1]) == nil)
	}
	return Unseen
}

// FormShimProof measures proof for one form-client interaction.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes id (string) which is the saved form-step name.
//
// Returns Proof which is the independently measured proof for the form-client step.
func FormShimProof(c *casefile.Case, o *Observation, id string) Proof {
	if c.Expected.FormShim == nil || o.FormShim == nil {
		return Unseen
	}
	for index, expected := range c.Expected.FormShim.Steps {
		if expected.Name != id || index >= len(o.FormShim.Steps) {
			continue
		}
		observed := o.FormShim.Steps[index]
		if observed.Name != id {
			return Unseen
		}
		if observed.Status == 0 {
			return Reached
		}
		return snapshotProof(true, checkFormShimStep(expected, observed) == nil)
	}
	return Unseen
}
