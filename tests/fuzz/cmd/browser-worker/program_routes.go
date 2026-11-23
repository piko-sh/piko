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
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// routeBudget bounds one browser program route visit and its observation.
	routeBudget = 10 * time.Second

	// maxProgramPathsBytes bounds the encoded browser route list.
	maxProgramPathsBytes = 16 << 10

	// programSnapshot reads bounded program leaves from the live DOM.
	programSnapshot = `(() => {
	const root = document.getElementById('fuzz-program');
	if (!root) return {Program:false};
	const nodes = root.querySelectorAll('output');
	if (nodes.length > 128) throw new Error('browser program has more than 128 output nodes');
	const hidden = el => {
		for (; el; el=el.parentElement) {
			const style = getComputedStyle(el);
			if (el.hidden || el.getAttribute('aria-hidden')?.trim().toLowerCase()==='true' || style.display==='none' || ['hidden','collapse'].includes(style.visibility)) return true;
		}
		return false;
	};
	const Output = Array.from(nodes, node => {
		const owners=[];
		for(let parent=node.parentElement;parent;parent=parent.parentElement){
			owners.push(parent.localName+(parent.id?'#'+parent.id:''));
			if(parent===root)break;
		}
		const row = {Owner:owners.reverse().join('/'), FullKey:node.getAttribute('p-key') || '', Hidden:hidden(node), ID:node.getAttribute('data-fuzz-output') || '', Text:node.textContent,
			HasValue:node.hasAttribute('data-value'), Value:node.getAttribute('data-value') || ''};
		if (node.getAttribute('data-fuzz-check-key') === 'true') {
			const key = (node.getAttribute('p-key') || '').split(/[.:]/).filter(Boolean);
			if (key.length > 1) row.ScopedKey = key[0] + '/' + key.at(-1);
		}
		if (Object.values(row).some(v => typeof v === 'string' && v.length > 1024)) throw new Error('browser program output exceeds 1024 characters');
		return row;
	});
	return {Program:true, ProgramRoot:root.localName, RootHidden:hidden(root), Output};
})()`
)

var (
	// errInvalidProgramPaths reports a malformed or excessive browser route list.
	errInvalidProgramPaths = errors.New("invalid browser program route list")
)

// programDOM is the decoded program snapshot of one route.
type programDOM struct {
	// ProgramRoot is the local name of the program's root element.
	ProgramRoot string

	// Output lists the program's output leaves.
	Output []casefile.ProgramOutput

	// RootHidden reports whether the program root is hidden.
	RootHidden bool

	// Program reports whether the page contains a program root.
	Program bool
}

// browserProgramPaths decodes and validates the saved browser route list.
//
// Takes raw (string) which holds the JSON-encoded route list.
//
// Returns []string which lists validated local browser destinations.
// Returns error which reports an invalid, excessive or nonlocal route list, or nil on
// success.
func browserProgramPaths(raw string) ([]string, error) {
	if len(raw) > maxProgramPathsBytes {
		return nil, fmt.Errorf("%w: exceeds %d bytes", errInvalidProgramPaths, maxProgramPathsBytes)
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidProgramPaths, err)
	}
	if len(paths) > casefile.MaxBrowserProgramRoutes {
		return nil, fmt.Errorf("%w: more than %d routes", errInvalidProgramPaths, casefile.MaxBrowserProgramRoutes)
	}
	for _, path := range paths {
		if !casefile.LocalBrowserPath(path) {
			return nil, fmt.Errorf("invalid browser program path %q", path)
		}
	}
	return paths, nil
}

// navigateRoute visits a same-origin route and records its program output.
//
// Takes target (*browser.Session) which is the tab that performs the visit.
// Takes origin (*url.URL) which is the parsed application origin.
// Takes path (string) which is the same-origin browser route to visit.
//
// Returns casefile.BrowserRouteVisit which records the final route and observed program
// output.
func navigateRoute(ctx context.Context, target *browser.Session, origin *url.URL, path string) casefile.BrowserRouteVisit {
	var visit casefile.BrowserRouteVisit
	visit.Path = path
	status, landed, err := target.NavigateResponse(ctx, origin.Scheme+"://"+origin.Host+path)
	if err != nil {
		visit.DriverError = sessionFailure(ctx, "", err)
		return visit
	}
	if status == 0 {
		visit.DriverError = "no HTTP document response"
		return visit
	}
	visit.Status = status
	final, err := url.Parse(landed)
	if err != nil || final.Scheme != origin.Scheme || final.Host != origin.Host {
		visit.DriverError = "document landed outside the local application"
		return visit
	}
	visit.Landed = final.RequestURI()
	return visit
}

// visitBrowserPrograms visits saved program routes in a separate browser target.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes origin (*url.URL) which is the parsed application origin.
// Takes paths ([]string) which lists the same-origin program routes.
//
// Returns []casefile.BrowserRouteVisit which lists browser route visit values in their
// recorded order.
func visitBrowserPrograms(ctx context.Context, session *browser.Session, origin *url.URL, paths []string) []casefile.BrowserRouteVisit {
	if len(paths) == 0 {
		return nil
	}

	tab, err := session.NewTab(ctx)
	if err != nil {
		var failed casefile.BrowserRouteVisit
		failed.Path = paths[0]
		failed.DriverError = sessionFailure(ctx, "", err)
		return []casefile.BrowserRouteVisit{failed}
	}
	defer func() {
		if err := tab.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "closing program tab:", err)
		}
	}()
	var visits []casefile.BrowserRouteVisit
	for _, path := range paths {
		visit := visitProgram(ctx, tab, origin, path)
		visits = append(visits, visit)
		if visit.DriverError != "" {
			break
		}
	}
	return visits
}

// visitProgram visits one program route within its own budget and reads its output.
//
// Takes tab (*browser.Session) which is the secondary tab.
// Takes origin (*url.URL) which is the parsed application origin.
// Takes path (string) which is the same-origin browser route to visit.
//
// Returns casefile.BrowserRouteVisit which records the visit and its program output.
func visitProgram(ctx context.Context, tab *browser.Session, origin *url.URL, path string) casefile.BrowserRouteVisit {
	routeCtx, cancel := stepContext(ctx, routeBudget)
	defer cancel()
	visit := navigateRoute(routeCtx, tab, origin, path)
	if visit.DriverError != "" {
		return visit
	}
	var dom programDOM
	if err := tab.Evaluate(routeCtx, programSnapshot, &dom); err != nil {
		visit.DriverError = sessionFailure(routeCtx, "", err)
	}
	visit.Program, visit.ProgramRoot, visit.RootHidden, visit.Output = dom.Program, dom.ProgramRoot, dom.RootHidden, dom.Output
	return visit
}

// programRoutesBudget sizes the session time the browser program routes need.
//
// Takes paths ([]string) which lists the program routes.
//
// Returns time.Duration which is one route budget per route.
func programRoutesBudget(paths []string) time.Duration {
	return time.Duration(len(paths)) * routeBudget
}
