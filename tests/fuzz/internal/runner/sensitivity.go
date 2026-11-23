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
	"net/http"
	"slices"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// sensitivityLeafCount is the number of output leaves a scoped reference needs.
	sensitivityLeafCount = 3
)

// MutationCheck describes a controlled change to a recorded observation.
type MutationCheck struct {
	// Name identifies the checker mutation or control.
	Name string

	// Subject identifies the failed property independently of generated values.
	Subject string `json:",omitempty"`

	// Control marks an unmodified checker control.
	Control bool

	// Detected reports whether the checker rejected the mutation.
	Detected bool

	// Passed reports whether the checker produced the expected control outcome.
	Passed bool
}

// routeMutation is one controlled change to a recorded browser route.
type routeMutation struct {
	// change applies the mutation to the route.
	change func(*casefile.BrowserRouteVisit)

	// name identifies the mutation.
	name string
}

// sensitivityRecorder accumulates mutation outcomes and the first unexpected one.
type sensitivityRecorder struct {
	// first holds the first control whose outcome differed from its expectation.
	first error

	// out lists the recorded checks in order.
	out []MutationCheck
}

// sensitivityFailure identifies a failed checker-sensitivity control.
type sensitivityFailure struct {
	// name identifies the fixture or operation.
	name string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e sensitivityFailure) Error() string { return "sensitivity: unexpected outcome for " + e.name }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e sensitivityFailure) FailureSubject() string { return e.name }

// record stores one mutation outcome.
//
// Takes name (string) which identifies the mutation or control.
// Takes positive (bool) which reports whether the checker is expected to pass.
// Takes err (error) which is the checker's result.
func (r *sensitivityRecorder) record(name string, positive bool, err error) {
	check := MutationCheck{Name: name, Subject: "", Control: positive, Detected: err != nil, Passed: (err == nil) == positive}
	if located, ok := errors.AsType[subjectError](err); ok {
		check.Subject = located.FailureSubject()
	}
	if !check.Passed && r.first == nil {
		r.first = sensitivityFailure{name: name}
	}
	r.out = append(r.out, check)
}

// MeasureSensitivity checks controlled mutations of the independent observations.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns []MutationCheck which lists mutation check values in their recorded order.
// Returns error which reports a checker control whose detection result differs from its
// expectation, or nil on success.
func MeasureSensitivity(c *casefile.Case) ([]MutationCheck, error) {
	paths, err := c.BrowserProgramPaths()
	if err != nil {
		return nil, err
	}
	if len(paths) != 2 || c.Expected.Client == nil {
		return nil, errors.New("sensitivity: missing saved scoped contracts")
	}
	var probes []*casefile.Probe
	for index := range c.Probes {
		if c.Probes[index].BrowserProgram {
			probes = append(probes, &c.Probes[index])
		}
	}
	if len(probes) < 2 || len(probes[0].Output) != sensitivityLeafCount || len(probes[1].Output) != sensitivityLeafCount {
		return nil, errors.New("sensitivity: scoped reference needs three output leaves")
	}
	recorder := &sensitivityRecorder{out: nil, first: nil}
	check := func(name string, positive bool, mutate func(*Observation)) {
		observation := sensitivityControl(c, probes)
		if mutate != nil {
			mutate(&observation)
		}
		_, err := BrowserRouteResults(c, &observation, true)
		recorder.record(name, positive, err)
	}
	check("browser-route/unmodified", true, nil)
	for _, mutation := range browserRouteMutations() {
		check("browser-route/"+mutation.name, false, func(o *Observation) { mutation.change(&o.Client.BrowserRoutes[1]) })
	}
	check("browser-route/stale-second-query", false, func(o *Observation) {
		o.Client.BrowserRoutes[2].Output = slices.Clone(o.Client.BrowserRoutes[1].Output)
	})
	if len(c.Expected.Client.After) == 0 {
		return recorder.out, errors.New("sensitivity: missing clickable component")
	}
	check("primary/unchanged-after-click", false, func(o *Observation) { o.Client.After[0].Count = "uncommitted" })
	var left, right Observation
	left.ApplicationDOM, right.ApplicationDOM = "A", "B"
	recorder.record("parity/equal-pair", true, compareObservations(&left, &left))
	recorder.record("parity/one-sided-dom-change", false, compareObservations(&left, &right))
	return recorder.out, recorder.first
}

// sensitivityControl builds an unmodified observation that satisfies the case's browser
// routes.
//
// Takes c (*casefile.Case) which holds the saved request and client model.
// Takes probes ([]*casefile.Probe) which lists the browser program probes.
//
// Returns Observation which is the passing control observation.
func sensitivityControl(c *casefile.Case, probes []*casefile.Probe) Observation {
	model := *c.Expected.Client
	model.Before = slices.Clone(model.Before)
	model.After = slices.Clone(model.After)
	var primary casefile.BrowserRouteVisit
	primary.Path, primary.Landed, primary.Status = c.Request, c.Request, c.Expected.Status
	model.BrowserRoutes = []casefile.BrowserRouteVisit{primary}
	for _, probe := range probes {
		var route casefile.BrowserRouteVisit
		route.Path, route.Landed, route.Status = probe.Path, probe.Path, probe.Status
		route.Program, route.ProgramRoot, route.Output = true, probe.ProgramRoot, slices.Clone(probe.Output)
		model.BrowserRoutes = append(model.BrowserRoutes, route)
	}
	var observation Observation
	observation.Client = &model
	return observation
}

// browserRouteMutations lists the controlled changes applied to the first program route.
//
// Returns []routeMutation which lists the mutations in check order.
func browserRouteMutations() []routeMutation {
	return []routeMutation{
		{name: "landing", change: func(route *casefile.BrowserRouteVisit) { route.Landed = "/different-document" }},
		{name: "http-status", change: func(route *casefile.BrowserRouteVisit) { route.Status = http.StatusNotFound }},
		{name: "root-missing", change: func(route *casefile.BrowserRouteVisit) { route.Program = false }},
		{name: "root-hidden", change: func(route *casefile.BrowserRouteVisit) { route.RootHidden = true }},
		{name: "wrong-owner", change: func(route *casefile.BrowserRouteVisit) { route.Output[0].Owner = "aside#unrelated" }},
		{name: "leaf-hidden", change: func(route *casefile.BrowserRouteVisit) { route.Output[0].Hidden = true }},
		{name: "key-namespace", change: func(route *casefile.BrowserRouteVisit) { route.Output[0].ScopedKey = "second/detail" }},
		{name: "full-key", change: func(route *casefile.BrowserRouteVisit) { route.Output[0].FullKey += ".stale" }},
		{name: "leaf-order", change: func(route *casefile.BrowserRouteVisit) {
			route.Output[0], route.Output[1] = route.Output[1], route.Output[0]
		}},
		{name: "attribute-missing", change: func(route *casefile.BrowserRouteVisit) { route.Output[2].HasValue = false }},
		{name: "attribute-value", change: func(route *casefile.BrowserRouteVisit) { route.Output[2].Value += "!" }},
		{name: "leaf-text", change: func(route *casefile.BrowserRouteVisit) { route.Output[2].Text += "!" }},
	}
}
