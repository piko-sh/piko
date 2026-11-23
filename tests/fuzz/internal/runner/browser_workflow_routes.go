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
	"fmt"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// workflowRouteTally accumulates per-route evidence for workflow journeys.
type workflowRouteTally struct {
	// first holds the first navigation failure recorded.
	first error

	// routes maps each workflow route to its evidence.
	routes map[string]BrowserRouteEvidence

	// incomplete records routes with at least one unverified or misdirected visit.
	incomplete map[string]bool

	// selected reports whether a browser was selected.
	selected bool
}

// record folds one expected visit into the route's evidence.
//
// Takes want (string) which is the expected route.
// Takes got (string) which is the observed route, empty when none was recorded.
// Takes checked (bool) which reports whether the visit's contract passed.
func (t *workflowRouteTally) record(want, got string, checked bool) {
	route := t.routes[want]
	route.Role = "workflow"
	route.Selected = t.selected
	route.Visited = route.Visited || t.selected && got == want
	t.incomplete[want] = t.incomplete[want] || !checked || got != want
	route.Verified = t.selected && !t.incomplete[want]
	t.routes[want] = route
	if t.selected && got != "" && got != want && t.first == nil {
		t.first = routeMismatch{subject: want + "/navigation", err: fmt.Errorf("landed at %q", got)}
	}
}

// recordStates folds an ordered state journey into the route evidence.
//
// Takes expected ([]string) which lists the modelled state paths.
// Takes observed ([]string) which lists the observed state paths.
// Takes verified (int) which counts the verified prefix of states.
func (t *workflowRouteTally) recordStates(expected, observed []string, verified int) {
	for index, path := range expected {
		t.record(path, pathAt(observed, index), index < verified)
	}
}

// recordDOM folds the DOM journey into the route evidence.
//
// Takes model (*casefile.DOMPlan) which holds the modelled journey.
// Takes client (*casefile.ClientTransition) which holds the browser observation, if any.
func (t *workflowRouteTally) recordDOM(model *casefile.DOMPlan, client *casefile.ClientTransition) {
	var trace *casefile.DOMTrace
	if client != nil {
		trace = client.DOM
	}
	path := ""
	if trace != nil && len(trace.States) > 0 {
		path = trace.States[0].Path
	}
	_, err := VerifyDOM(model, trace)
	t.record(model.Path, path, err == nil)
}

// recordFormShim folds the browser form-shim submission into the route evidence.
//
// Takes model (*casefile.FormShimContract) which holds the modelled submission.
// Takes o (*Observation) which holds the application observation.
func (t *workflowRouteTally) recordFormShim(model *casefile.FormShimContract, o *Observation) {
	path := ""
	var got *casefile.FormShimBrowser
	if o.Client != nil {
		got = o.Client.FormShim
	}
	if got != nil {
		path = got.Page
	}
	t.record(model.Page, path, got != nil && checkBrowserIntake(model, o.FormShim, o.Client) == nil)
	if t.selected && got != nil && path == "" && got.DriverError == "" && t.first == nil {
		t.first = routeMismatch{subject: model.Page + "/navigation", err: errors.New("missing observed location")}
	}
}

// recordFormWorkflow folds the form workflow journey into the route evidence.
//
// Takes model (*casefile.FormWorkflow) which holds the modelled journey.
// Takes got (*casefile.FormWorkflowObservation) which holds the observed journey, if any.
func (t *workflowRouteTally) recordFormWorkflow(model *casefile.FormWorkflow, got *casefile.FormWorkflowObservation) {
	path := ""
	if got != nil {
		path = got.BrowserPath
	}
	t.record("/workflow", path, got != nil && len(model.Steps) > 0 && VerifiedFormSteps(model, got) == len(model.Steps))
	if t.selected && got != nil && path == "" && got.DriverError == "" && t.first == nil {
		t.first = routeMismatch{subject: "/workflow/navigation", err: errors.New("missing observed location")}
	}
}

// recordEvents folds the browser event journey into the route evidence.
//
// Takes model (*casefile.EventsTrace) which holds the modelled journey.
// Takes client (*casefile.ClientTransition) which holds the browser observation, if any.
func (t *workflowRouteTally) recordEvents(model *casefile.EventsTrace, client *casefile.ClientTransition) {
	path := ""
	var got *casefile.EventsTrace
	if client != nil {
		got = client.Events
	}
	if got != nil {
		path = got.BrowserPath
	}
	t.record("/events", path, got != nil && len(model.States) > 0 && VerifiedEventSteps(got, model) == len(model.States))
	if t.selected && got != nil && path == "" && got.DriverError == "" && t.first == nil {
		t.first = routeMismatch{subject: "/events/navigation", err: errors.New("missing observed location")}
	}
}

// recordClientJourneys folds the lifecycle, checkout and partial-component journeys into
// the route evidence.
//
// Takes expected (*casefile.ClientTransition) which holds the modelled client journeys.
// Takes client (*casefile.ClientTransition) which holds the browser observation, if any.
func (t *workflowRouteTally) recordClientJourneys(expected, client *casefile.ClientTransition) {
	if client == nil {
		var empty casefile.ClientTransition
		client = &empty
	}
	if model := expected.Lifecycle; model != nil {
		t.recordLifecycle(model, client.Lifecycle)
	}
	if model := expected.Checkout; model != nil {
		t.recordCheckout(model, client.Checkout)
	}
	if model := expected.PartialComponents; model != nil {
		t.recordPartialComponents(model, client.PartialComponents)
	}
}

// recordLifecycle folds the lifecycle journey into the route evidence.
//
// Takes model (*casefile.LifecycleTrace) which holds the modelled journey.
// Takes got (*casefile.LifecycleTrace) which holds the observed journey, if any.
func (t *workflowRouteTally) recordLifecycle(model, got *casefile.LifecycleTrace) {
	path := func(state *casefile.LifecycleState) string { return state.Path }
	verified := 0
	var observed []string
	if got != nil {
		verified = VerifiedLifecycleSteps(got, model)
		observed = statePaths(got.States, path)
	}
	t.recordStates(statePaths(model.States, path), observed, verified)
}

// recordCheckout folds the checkout journey into the route evidence.
//
// Takes model (*casefile.CheckoutTrace) which holds the modelled journey.
// Takes got (*casefile.CheckoutTrace) which holds the observed journey, if any.
func (t *workflowRouteTally) recordCheckout(model, got *casefile.CheckoutTrace) {
	path := func(state *casefile.CheckoutState) string { return state.Path }
	verified := 0
	var observed []string
	if got != nil {
		verified = VerifiedCheckoutSteps(got, model)
		observed = statePaths(got.States, path)
	}
	t.recordStates(statePaths(model.States, path), observed, verified)
}

// recordPartialComponents folds the partial-component journey into the route evidence.
//
// Takes model (*casefile.PartialComponentsTrace) which holds the modelled journey.
// Takes got (*casefile.PartialComponentsTrace) which holds the observed journey, if any.
func (t *workflowRouteTally) recordPartialComponents(model, got *casefile.PartialComponentsTrace) {
	path := func(state *casefile.PartialComponentState) string { return state.Path }
	verified := 0
	var observed []string
	if got != nil {
		verified = VerifiedPartialComponentSteps(got, model)
		observed = statePaths(got.States, path)
	}
	t.recordStates(statePaths(model.States, path), observed, verified)
}

// browserWorkflowRoutes checks the real routes recorded by workflow journeys.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes selected (bool) which reports whether a browser was selected.
//
// Returns map[string]BrowserRouteEvidence which records workflow-route proof.
// Returns error which reports an unobserved or incorrect saved browser route, or nil on
// success.
func browserWorkflowRoutes(c *casefile.Case, o *Observation, selected bool) (map[string]BrowserRouteEvidence, error) {
	tally := &workflowRouteTally{first: nil, routes: map[string]BrowserRouteEvidence{}, incomplete: map[string]bool{}, selected: selected}
	if model := c.Expected.DOM; model != nil {
		tally.recordDOM(model, o.Client)
	}
	if model := c.Expected.FormShim; model != nil && model.BrowserSubmission {
		tally.recordFormShim(model, o)
	}
	if model := c.Expected.FormWorkflow; model != nil {
		tally.recordFormWorkflow(model, o.FormWorkflow)
	}
	expectedClient := c.Expected.Client
	if expectedClient == nil {
		return tally.routes, tally.first
	}
	if model := expectedClient.Events; model != nil {
		tally.recordEvents(model, o.Client)
	}
	tally.recordClientJourneys(expectedClient, o.Client)
	return tally.routes, tally.first
}

// statePaths lists the route path of each recorded state in order.
//
// Takes states ([]T) which holds the recorded states.
// Takes path (func(*T) string) which reads one state's path.
//
// Returns []string which lists the paths in state order.
func statePaths[T any](states []T, path func(*T) string) []string {
	paths := make([]string, 0, len(states))
	for index := range states {
		paths = append(paths, path(&states[index]))
	}
	return paths
}

// pathAt returns the observed path at a position, or an empty path past the end.
//
// Takes paths ([]string) which holds observed paths in recorded order.
// Takes index (int) which is the position to read.
//
// Returns string which is the observed path, or empty when none was recorded.
func pathAt(paths []string, index int) string {
	if index < len(paths) {
		return paths[index]
	}
	return ""
}
