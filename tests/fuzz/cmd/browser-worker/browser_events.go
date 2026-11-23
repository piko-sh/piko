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
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
	"piko.sh/piko/wdk/browser"
)

const (
	// eventsStepBudget bounds one sampled event step.
	eventsStepBudget = 6 * time.Second

	// eventsStepCount is the number of sampled steps in the event journey.
	eventsStepCount = 12

	// eventsEmissions is the number of sampled emissions per owner.
	eventsEmissions = 4

	// eventsMountWait bounds the wait for the event page's controls.
	eventsMountWait = 5 * time.Second

	// eventsDiagnosticLimit is the largest number of retained browser diagnostics.
	eventsDiagnosticLimit = 6

	// eventsDiagnosticBytes bounds the byte length of one retained diagnostic.
	eventsDiagnosticBytes = 320

	// eventsReady reports that the event page's controls and bindings are present.
	eventsReady = `document.querySelector('#leaf')?.shadowRoot?.querySelector('#emit-3') && ` +
		`document.querySelector('#right')?.shadowRoot?.querySelector('#tick') && ` +
		`document.querySelector('#event-page')?.hasAttribute('pk-ev-bound')`

	// eventsShot reads the event journey's public counters.
	eventsShot = `(() => {
	const el=id=>document.getElementById(id), leaf=el('leaf');
	const counter=node=>{ const text=node.textContent; if (!/^\d+$/.test(text)) throw new Error('invalid counter '+node.id+': '+text); return Number(text); };
	const num=(host,id)=>counter(host.shadowRoot.querySelector('#'+id));
	return {Owner:leaf.parentElement.id,SameLeaf:leaf===window.__fuzzFirstLeaf,Initializations:Number(window.__pikoEventInits||0),Local:num(leaf,'local'),
		LeafSerial:num(leaf,'serial'),LeftSerial:num(el('left'),'serial'),RightSerial:num(el('right'),'serial'),Left:num(el('left'),'seen'),Right:num(el('right'),'seen'),
		PK:counter(el('event-count')),Payload:el('event-payload').textContent,Log:el('event-log').textContent};
})()`
)

// eventDiagnostics collects bounded same-origin browser failures seen during the journey.
type eventDiagnostics struct {
	// host is the application's host.
	host string

	// messages lists the retained diagnostics in observed order.
	messages []string

	// mu guards messages against concurrent browser callbacks.
	mu sync.Mutex
}

// eventsJourney drives the component event journey and records its states.
type eventsJourney struct {
	// session is the root browser session.
	session *browser.Session

	// trace receives the sampled states and any driver failure.
	trace *casefile.EventsTrace

	// diagnostics collects browser failures quoted in a driver error.
	diagnostics *eventDiagnostics
}

// observe records a same-origin HTTP failure or a JavaScript exception.
//
// Takes event (browser.NetworkEvent) which is the observed browser event.
//
// Concurrency: Safe for concurrent callbacks. The mutex guards the messages.
func (d *eventDiagnostics) observe(event browser.NetworkEvent) {
	var message string
	switch event.Kind {
	case browser.ResponseReceived:
		if event.Status >= http.StatusBadRequest {
			location, err := url.Parse(event.URL)
			if err == nil && location.Host == d.host {
				message = fmt.Sprintf("HTTP %d %s", event.Status, location.Path)
			}
		}
	case browser.JavaScriptException:
		message = event.Exception
	default:
		return
	}
	if message == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.messages) < eventsDiagnosticLimit {
		d.messages = append(d.messages, textlimit.Truncate(message, eventsDiagnosticBytes))
	}
}

// summary joins the retained diagnostics for a driver error.
//
// Returns string which is the parenthesised diagnostics, or empty when none were seen.
//
// Concurrency: Safe for concurrent use. The mutex protects the snapshot.
func (d *eventDiagnostics) summary() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.messages) == 0 {
		return ""
	}
	return " (" + strings.Join(d.messages, ", ") + ")"
}

// sample runs one step's actions, waits for rendering and records the counters.
//
// Takes name (string) which names the step.
// Takes actions (...browserAction) which lists the step's browser operations.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *eventsJourney) sample(ctx context.Context, name string, actions ...browserAction) bool {
	var state casefile.EventState
	actions = append(actions,
		await("new Promise(ok=>requestAnimationFrame(()=>requestAnimationFrame(()=>ok(true))))", nil),
		evaluate(eventsShot, &state))
	stepCtx, cancel := stepContext(ctx, eventsStepBudget)
	defer cancel()
	if err := runBrowser(stepCtx, journey.session, actions...); err != nil {
		journey.trace.DriverError = sessionFailure(stepCtx, name, err) + journey.diagnostics.summary()
		return false
	}
	state.Step = name
	journey.trace.States = append(journey.trace.States, state)
	return true
}

// owner moves the leaf to an owner when needed, ticks the owner and samples each
// emission.
//
// Takes input (casefile.EventsInput) which selects the emission flags.
// Takes ownerIndex (int) which is the owner's position in the journey.
// Takes owner (string) which is the owner element's ID.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *eventsJourney) owner(ctx context.Context, input casefile.EventsInput, ownerIndex int, owner string) bool {
	if ownerIndex == 1 && !journey.sample(ctx, "move-right", evaluate("document.getElementById('right').append(document.getElementById('leaf'))", nil)) {
		return false
	}
	if !journey.sample(ctx, "tick-"+owner, evaluate(fmt.Sprintf("document.getElementById('%s').shadowRoot.querySelector('#tick').click()", owner), nil)) {
		return false
	}
	for emission := range eventsEmissions {
		flag := casefile.EventFlags(input, emission)
		click := fmt.Sprintf("document.getElementById('leaf').shadowRoot.querySelector('#emit-%d').click()", flag)
		if !journey.sample(ctx, fmt.Sprintf("%s-flags-%d", owner, flag), evaluate(click, nil)) {
			return false
		}
	}
	return true
}

// browserEvents runs the component event journey in Chromium.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes input (casefile.EventsInput) which selects the component event journey values.
//
// Returns *casefile.EventsTrace which records event states reached before completion or
// driver failure.
func browserEvents(ctx context.Context, session *browser.Session, address string, input casefile.EventsInput) *casefile.EventsTrace {
	trace := new(casefile.EventsTrace)
	trace.Input = input
	base, err := url.Parse(address)
	if err != nil {
		trace.DriverError = err.Error()
		return trace
	}
	diagnostics := new(eventDiagnostics)
	diagnostics.host = base.Host
	stop, err := session.ObserveNetwork(ctx, diagnostics.observe)
	if err != nil {
		trace.DriverError = sessionFailure(ctx, "mount", err)
		return trace
	}
	defer stop()
	journey := &eventsJourney{session: session, trace: trace, diagnostics: diagnostics}
	if !journey.sample(ctx, "mount", navigate(base.ResolveReference(&url.URL{Path: "/events"}).String()),
		evaluate("location.pathname+location.search+location.hash", &trace.BrowserPath),
		waitExpression(eventsReady, eventsMountWait),
		evaluate("window.__fuzzFirstLeaf=document.getElementById('leaf'); true", nil)) {
		return trace
	}
	for ownerIndex, owner := range []string{"left", "right"} {
		if !journey.owner(ctx, input, ownerIndex, owner) {
			return trace
		}
	}
	return trace
}

// eventsBudget sizes the session time the event journey needs.
//
// Returns time.Duration which is the sum of the journey's step budgets.
func eventsBudget() time.Duration {
	return eventsStepCount * eventsStepBudget
}
