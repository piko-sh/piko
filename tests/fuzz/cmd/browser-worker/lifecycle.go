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
	"net/url"
	"strconv"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// lifecycleStepBudget bounds one lifecycle step.
	lifecycleStepBudget = 6 * time.Second

	// lifecycleStepCount is the number of steps in the lifecycle journey.
	lifecycleStepCount = 9

	// lifecycleWait bounds one readiness wait inside a lifecycle step.
	lifecycleWait = 30 * time.Second

	// lifecycleBootstrap installs the journey's public event log before each document.
	lifecycleBootstrap = `window.__journey={connected:0,cleaned:0,deliveries:0,prompts:0,release:{},ready:[],applied:[],finished:[]};`

	// lifecycleSnapshot reads the lifecycle journey's public state.
	lifecycleSnapshot = `(() => {
		const log=window.__journey, draft=document.querySelector('#draft'), row=document.querySelector('#retained-row');
		return {Path:location.pathname,Draft:draft?.value || '',Focus:document.activeElement?.id || '',
			Connected:log.connected,Cleaned:log.cleaned,Deliveries:log.deliveries,Prompts:log.prompts,
			Applied:log.applied.join(','),ActiveValue:document.querySelector('#second')?.shadowRoot?.querySelector('#received')?.textContent || '',
			SameDocument:!!log.original,SameRow:!!row && row===log.row};
	})()`
)

// lifecycleJourney drives the component navigation and cleanup journey.
type lifecycleJourney struct {
	// session is the root browser session.
	session *browser.Session

	// trace receives the recorded states and any failed step.
	trace *casefile.LifecycleTrace
}

// step performs one lifecycle step and records the resulting state.
//
// Takes name (string) which names the step.
// Takes actions (...browserAction) which lists the step's browser operations.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *lifecycleJourney) step(ctx context.Context, name string, actions ...browserAction) bool {
	var state casefile.LifecycleState
	actions = append(actions, evaluate(lifecycleSnapshot, &state))
	stepCtx, cancel := stepContext(ctx, lifecycleStepBudget)
	defer cancel()
	if err := runBrowser(stepCtx, journey.session, actions...); err != nil {
		journey.trace.FailedStep, journey.trace.DriverError = name, sessionFailure(stepCtx, "", err)
		return false
	}
	state.Step = name
	journey.trace.States = append(journey.trace.States, state)
	return true
}

// prepare mounts the journey, edits and refreshes the draft, cancels a navigation and
// starts the owners' requests.
//
// Takes start (string) which is the journey page URL.
// Takes input (casefile.LifecycleInput) which selects the draft text.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *lifecycleJourney) prepare(ctx context.Context, start string, input casefile.LifecycleInput) bool {
	register := func(stepCtx context.Context, target *browser.Session) error {
		return target.ScriptOnNewDocument(stepCtx, lifecycleBootstrap)
	}
	if !journey.step(ctx, "mount", register, navigate(start),
		lifecycleWaitFor("window.__journey.connected === 2 && !!window.piko?.partials"),
		lifecycleRun("window.__journey.original=true; window.__journey.row=document.querySelector('#retained-row')")) {
		return false
	}
	draft, err := scriptString(input.Draft)
	if err != nil {
		journey.trace.FailedStep, journey.trace.DriverError = "edit-and-refresh", err.Error()
		return false
	}
	if !journey.step(ctx, "edit-and-refresh", lifecycleRun(`const field=document.querySelector('#draft'); field.focus(); field.value=`+draft+`;
		field.dispatchEvent(new Event('input',{bubbles:true})); document.querySelector('#retained-row').textContent='pending';
		window.piko.partials.reload('journey-list',{mode:'merge'})`), lifecycleWaitFor("document.querySelector('#retained-row')?.textContent==='saved'")) {
		return false
	}
	if !journey.step(ctx, "cancel-navigation", lifecycleRun(`window.confirm=()=>{window.__journey.prompts++;return false};
		window.piko.nav.navigate('/journey-away')`)) {
		return false
	}
	return journey.step(ctx, "start-requests", lifecycleRun("window.dispatchEvent(new Event('journey-request')); window.__journey.started=true"),
		lifecycleWaitFor("window.__journey.ready.length === 2"))
}

// release resolves the owners' pending requests in the selected order.
//
// Takes input (casefile.LifecycleInput) which selects the completion order.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *lifecycleJourney) release(ctx context.Context, input casefile.LifecycleInput) bool {
	order := []string{"first", "second"}
	if input.Reverse {
		order[0], order[1] = order[1], order[0]
	}
	for index, owner := range order {
		quoted := strconv.Quote(owner)
		ready := fmt.Sprintf("window.__journey.finished.length === %d", index+1)
		if owner == "second" {
			ready += " && document.querySelector('#second')?.shadowRoot?.querySelector('#received')?.textContent === 'second'"
		}
		if !journey.step(ctx, "release-"+owner, lifecycleRun("window.__journey.release["+quoted+"]()"), lifecycleWaitFor(ready)) {
			return false
		}
	}
	return true
}

// browserLifecycle runs the component navigation and cleanup journey.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes input (casefile.LifecycleInput) which selects draft text and completion order.
//
// Returns *casefile.LifecycleTrace which records lifecycle states reached before
// completion or driver failure.
func browserLifecycle(ctx context.Context, session *browser.Session, address string, input casefile.LifecycleInput) *casefile.LifecycleTrace {
	trace := new(casefile.LifecycleTrace)
	trace.Input = input
	base, err := url.Parse(address)
	if err != nil {
		trace.FailedStep, trace.DriverError = "mount", err.Error()
		return trace
	}
	journey := &lifecycleJourney{session: session, trace: trace}
	if !journey.prepare(ctx, base.ResolveReference(&url.URL{Path: "/journey"}).String(), input) {
		return trace
	}
	if !journey.step(ctx, "detach-first", lifecycleRun("document.querySelector('#first').remove()")) || !journey.release(ctx, input) {
		return trace
	}
	if !journey.step(ctx, "navigate-away", lifecycleRun(`window.confirm=()=>{window.__journey.prompts++;return true};
		window.piko.nav.navigate('/journey-away')`), lifecycleWaitFor("!!document.querySelector('#journey-away')")) {
		return trace
	}
	journey.step(ctx, "back-and-dispatch", lifecycleRun("history.back()"),
		lifecycleWaitFor("!!document.querySelector('#second')?.shadowRoot?.querySelector('#received')"),
		lifecycleRun("window.dispatchEvent(new Event('journey-request'))"))
	return trace
}

// lifecycleWaitFor builds an action that waits for a lifecycle readiness expression.
//
// Takes expression (string) which is the JavaScript condition.
//
// Returns browserAction which performs the bounded wait.
func lifecycleWaitFor(expression string) browserAction {
	return waitExpression(expression, lifecycleWait)
}

// lifecycleRun builds an action that awaits a lifecycle script.
//
// Takes expression (string) which is the JavaScript to run.
//
// Returns browserAction which awaits the script.
func lifecycleRun(expression string) browserAction {
	return await(expression, nil)
}

// lifecycleBudget sizes the session time the lifecycle journey needs.
//
// Returns time.Duration which is the sum of the journey's step budgets.
func lifecycleBudget() time.Duration {
	return lifecycleStepCount * lifecycleStepBudget
}
