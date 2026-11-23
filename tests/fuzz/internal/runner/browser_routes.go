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
	"slices"
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// BrowserRouteEvidence records selection, navigation and verification of a route.
type BrowserRouteEvidence struct {
	// Role identifies the route's role in the saved case.
	Role string

	// Selected reports whether this route was selected for browser observation.
	Selected bool

	// Visited reports whether the browser visited this route.
	Visited bool

	// Verified reports whether the route's independent contract passed.
	Verified bool
}

// routeMismatch attaches a stable route subject to a failure.
type routeMismatch struct {
	// err holds the underlying failure.
	err error

	// subject identifies the failed property independently of generated values.
	subject string
}

// browserRouteRequest names one route the browser must visit and its saved model.
type browserRouteRequest struct {
	// model is the saved program probe, nil for the primary route.
	model *casefile.Probe

	// path is the requested route.
	path string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (r routeMismatch) Error() string { return "browser/route: " + r.subject + ": " + r.err.Error() }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (r routeMismatch) FailureSubject() string { return r.subject }

// BrowserRouteResults retains the requested URLs and independent result per route,
// including successes after a different route's output failed.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes selected (bool) which reports whether a browser was selected.
//
// Returns map[string]BrowserRouteEvidence which maps route identities to their
// independently measured proof.
// Returns error which reports an unobserved or incorrect saved browser route, or nil on
// success.
func BrowserRouteResults(c *casefile.Case, o *Observation, selected bool) (map[string]BrowserRouteEvidence, error) {
	evidence, failures := browserRouteFailures(c, o, selected)
	if len(failures) > 0 {
		return evidence, failures[0]
	}
	return evidence, nil
}

// browserRouteFailures records route proof and independent navigation failures.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
// Takes selected (bool) which reports whether a browser was selected.
//
// Returns map[string]BrowserRouteEvidence which records per-route navigation proof.
// Returns []error which holds independent route failures.
func browserRouteFailures(c *casefile.Case, o *Observation, selected bool) (map[string]BrowserRouteEvidence, []error) {
	if c.BrowserSampleVersion == 0 {
		return nil, nil
	}
	evidence, workflowErr := browserWorkflowRoutes(c, o, selected)
	var failures []error
	if workflowErr != nil {
		failures = append(failures, workflowErr)
	}
	if c.Expected.FormWorkflow != nil || c.Expected.Tag == "" {
		return evidence, failures
	}
	wanted := wantedBrowserRoutes(c)
	var actual []casefile.BrowserRouteVisit
	if o.Client != nil {
		actual = o.Client.BrowserRoutes
	}
	for index, route := range wanted {
		routeEvidence := BrowserRouteEvidence{Role: "program", Selected: selected, Visited: false, Verified: false}
		if route.model == nil {
			routeEvidence.Role = "primary"
		}
		if selected {
			if err := checkBrowserRoute(c, o, route, actual, index, &routeEvidence); err != nil {
				failures = append(failures, err)
			}
		}
		evidence[route.path] = routeEvidence
	}
	if selected && len(actual) > len(wanted) {
		failures = append(failures, routeMismatch{subject: "extra", err: fmt.Errorf("received %d documents, want %d", len(actual), len(wanted))})
	}
	return evidence, failures
}

// wantedBrowserRoutes lists the primary route followed by every browser program probe.
//
// Takes c (*casefile.Case) which holds the saved request and probes.
//
// Returns []browserRouteRequest which lists the routes in visit order.
func wantedBrowserRoutes(c *casefile.Case) []browserRouteRequest {
	wanted := []browserRouteRequest{{path: c.Request, model: nil}}
	for index := range c.Probes {
		probe := &c.Probes[index]
		if probe.BrowserProgram {
			wanted = append(wanted, browserRouteRequest{path: probe.Path, model: probe})
		}
	}
	return wanted
}

// checkBrowserRoute verifies one visited route against its saved model.
//
// Takes c (*casefile.Case) which holds the saved expected results.
// Takes o (*Observation) which holds the application observation.
// Takes route (browserRouteRequest) which names the route and its model.
// Takes actual ([]casefile.BrowserRouteVisit) which lists the browser's visits in order.
// Takes index (int) which is the route's position in visit order.
// Takes evidence (*BrowserRouteEvidence) which receives the visited and verified flags.
//
// Returns error which is a routeMismatch for the failed property, or nil when the route
// passes.
func checkBrowserRoute(c *casefile.Case, o *Observation, route browserRouteRequest, actual []casefile.BrowserRouteVisit, index int, evidence *BrowserRouteEvidence) error {
	primary := route.model == nil
	subject := route.path
	if primary {
		subject = "primary"
	}
	fail := func(field string, err error) error {
		return routeMismatch{subject: subject + "/" + field, err: err}
	}
	if index >= len(actual) || actual[index].Path != route.path {
		return fail("missing", errors.New("missing matching document observation"))
	}
	got := &actual[index]
	evidence.Visited = got.Landed == route.path && got.Status > 0
	if got.DriverError != "" {
		return fail("driver", errors.New(got.DriverError))
	}
	if !evidence.Visited {
		return fail("navigation", fmt.Errorf("landed at %q with HTTP %d", got.Landed, got.Status))
	}
	status := c.Expected.Status
	if !primary {
		status = route.model.Status
	}
	if got.Status != status {
		return fail("status", fmt.Errorf("got %d, want %d", got.Status, status))
	}
	if primary {
		if err := checkPrimaryBrowserState(c, o); err != nil {
			return fail("state", err)
		}
	} else if err := checkBrowserProgram(route.model, got); err != nil {
		if located, ok := errors.AsType[subjectError](err); ok {
			return routeMismatch{subject: located.FailureSubject(), err: err}
		}
		return fail("output", err)
	}
	evidence.Verified = true
	return nil
}

// checkBrowserProgram checks a visited route's program output against its probe.
//
// Takes model (*casefile.Probe) which holds the saved program probe.
// Takes got (*casefile.BrowserRouteVisit) which holds the browser's visit.
//
// Returns error which reports a program mismatch, or nil when the output matches.
func checkBrowserProgram(model *casefile.Probe, got *casefile.BrowserRouteVisit) error {
	var observed ProbeObservation
	observed.Program, observed.ProgramRoot, observed.RootHidden, observed.Output = got.Program, got.ProgramRoot, got.RootHidden, got.Output
	return checkProgram(*model, observed)
}

// checkPrimaryBrowserState checks that secondary visits preserved the primary browser
// state.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkPrimaryBrowserState(c *casefile.Case, o *Observation) error {
	if expected := c.Expected.Client; expected != nil {
		if o.Client == nil {
			return errors.New("missing primary browser states")
		}
		if err := compareInstances("initial", o.Client.Before, expected.Before); err != nil {
			return err
		}
		if len(expected.Before) == 0 || len(expected.After) == 0 {
			return errors.New("empty expected primary browser states")
		}
		return compareInstances("after-click", o.Client.After, expected.After)
	}
	if !slices.Equal(o.Counts, []string{strconv.Itoa(c.Expected.Count), strconv.Itoa(c.Expected.Count + 1)}) {
		return fmt.Errorf("unexpected count transitions %v", o.Counts)
	}
	return nil
}

// readBrowserReceipt reads a browser receipt under the fixed byte limit.
//
// Takes work (string) which is the invocation-private working directory.
// Takes name (string) which is the receipt file, relative to work.
//
// Returns []byte which holds the bounded receipt bytes.
// Returns error which reports an unreadable or oversized browser receipt, wrapping
// fs.ErrNotExist when the receipt is absent, or nil on success.
func readBrowserReceipt(work, name string) ([]byte, error) {
	data, err := readBoundedFile(work, name, casefile.MaxBrowserReceiptBytes)
	if err != nil {
		return nil, fmt.Errorf("browser receipt: %w", err)
	}
	return data, nil
}
