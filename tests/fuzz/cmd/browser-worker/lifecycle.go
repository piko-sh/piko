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
	"strconv"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// lifecycleStepBudget bounds one lifecycle step.
	lifecycleStepBudget = 6 * time.Second

	// editorMountStep names the editor journey's first step.
	editorMountStep = "mount"

	// reuseMountStep names the element-reuse journey's first step.
	reuseMountStep = "reuse-mount"

	// lifecycleWait bounds one readiness wait inside a lifecycle step.
	lifecycleWait = 30 * time.Second

	// lifecycleSettlePeriod bounds the wait for the lifecycle callbacks of a navigation to
	// reach the counts the model expects. The step records its state when the period ends,
	// so a lifecycle callback that never fires shows as a model mismatch rather than as a
	// driver timeout.
	lifecycleSettlePeriod = 2 * time.Second

	// lifecycleBootstrap installs the journey's public event log before each document. A
	// full page load replaces the log, which clears the marks SameDocument reads.
	lifecycleBootstrap = `window.__journey={connected:0,cleaned:0,deliveries:0,prompts:0,release:{},ready:[],applied:[],finished:[],
		pages:{journey:{connected:0,disconnected:0,cleaned:0,clicks:0},away:{connected:0,disconnected:0,cleaned:0,clicks:0}}};`

	// lifecycleSnapshot reads the lifecycle journey's public state.
	lifecycleSnapshot = `(() => {
		const log=window.__journey, draft=document.querySelector('#draft'), row=document.querySelector('#retained-row');
		const page=counts=>({Connected:counts.connected,Disconnected:counts.disconnected,Cleaned:counts.cleaned});
		return {Path:location.pathname,Draft:draft?.value || '',Focus:document.activeElement?.id || '',
			Connected:log.connected,Cleaned:log.cleaned,Deliveries:log.deliveries,Prompts:log.prompts,
			Applied:log.applied.join(','),ActiveValue:document.querySelector('#second')?.shadowRoot?.querySelector('#received')?.textContent || '',
			Journey:page(log.pages.journey),Away:page(log.pages.away),Clicks:log.pages.journey.clicks,
			SameDocument:!!log.original,SameRow:!!row && row===log.row};
	})()`

	// lifecycleSettleScript resolves once a condition holds or a number of milliseconds has
	// passed, whichever comes first.
	lifecycleSettleScript = `new Promise(resolve => {
		const end = performance.now() + %d;
		const poll = () => (%s) || performance.now() > end ? resolve(true) : setTimeout(poll, 20);
		poll();
	})`

	// lifecycleSettledCondition formats the condition that holds once the page shows the
	// route and lifecycle counts the model expects. It checks the path, the component
	// connections and cleanups, then the journey and away page scripts' counts.
	lifecycleSettledCondition = `location.pathname === %s && window.__journey.connected === %d && window.__journey.cleaned === %d &&
		%s && %s`

	// pageScriptSettledCondition formats the condition that holds once one page script's
	// lifecycle counts match the model.
	pageScriptSettledCondition = `window.__journey.pages.%s.connected === %d && window.__journey.pages.%[1]s.disconnected === %d &&
		window.__journey.pages.%[1]s.cleaned === %d`

	// componentConnections counts the journey's component connections.
	componentConnections = "window.__journey.connected"

	// journeyShown holds once the journey page has rendered its second watcher.
	journeyShown = "!!document.querySelector('#second')?.shadowRoot?.querySelector('#received')"

	// awayShown holds once the away page has rendered.
	awayShown = "!!document.querySelector('#journey-away')"

	// clickBound clicks the element the journey page script bound its listener to.
	clickBound = "document.querySelector('#journey-bound').click()"
)

var (
	// errUnmodelledLifecycleStep is returned for a lifecycle step the model has no state
	// for.
	errUnmodelledLifecycleStep = errors.New("lifecycle step has no modelled state")
)

// lifecycleJourney drives the component navigation and cleanup journey.
type lifecycleJourney struct {
	// session is the root browser session.
	session *browser.Session

	// trace receives the recorded states and any failed step.
	trace *casefile.LifecycleTrace

	// expected maps each step name to the state the model expects it to record.
	expected map[string]casefile.LifecycleState
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

// mount loads a journey's first page with a fresh event log and waits for it to be ready,
// then marks the document and the elements later steps compare against.
//
// Takes name (string) which names the step.
// Takes start (string) which is the absolute URL of the journey's first page.
// Takes ready (string) which is the condition that holds once the page is ready.
// Takes marks (string) which is the JavaScript that marks the loaded document.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *lifecycleJourney) mount(ctx context.Context, name, start, ready, marks string) bool {
	register := func(stepCtx context.Context, target *browser.Session) error {
		return target.ScriptOnNewDocument(stepCtx, lifecycleBootstrap)
	}
	return journey.step(ctx, name, register, navigate(start), lifecycleWaitFor(ready),
		journey.settle(name), lifecycleRun(marks))
}

// visit builds the actions that start a navigation, wait for the destination page to
// render and give its scripts the settle period to reach the counts the model expects for
// the step.
//
// Takes name (string) which names the step the visit belongs to.
// Takes navigation (string) which is the JavaScript that starts the navigation.
// Takes shown (string) which is the condition that holds once the destination renders.
//
// Returns []browserAction which performs the visit.
func (journey *lifecycleJourney) visit(name, navigation, shown string) []browserAction {
	return []browserAction{lifecycleRun(navigation), lifecycleWaitFor(shown), journey.settle(name)}
}

// reuseVisit builds the actions that navigate to an element-reuse listing.
//
// Takes name (string) which names the step the visit belongs to.
// Takes navigation (string) which is the JavaScript that starts the navigation.
// Takes listing (string) which names the destination listing.
//
// Returns []browserAction which performs the visit.
func (journey *lifecycleJourney) reuseVisit(name, navigation, listing string) []browserAction {
	return journey.visit(name, navigation, "!!document.querySelector('#"+listing+"-filter')")
}

// settle builds an action that waits until the page shows the route and lifecycle counts
// the model expects for a step, or the settle period ends.
//
// Takes name (string) which names the step.
//
// Returns browserAction which performs the bounded wait, or fails when the model has no
// state for the step.
func (journey *lifecycleJourney) settle(name string) browserAction {
	expected, found := journey.expected[name]
	if !found {
		return func(context.Context, *browser.Session) error {
			return fmt.Errorf("%w: %s", errUnmodelledLifecycleStep, name)
		}
	}
	return lifecycleSettle(settledCondition(expected))
}

// prepare mounts the journey, clicks the page script's bound element, edits and refreshes
// the draft, cancels a navigation and starts the owners' requests.
//
// Takes start (string) which is the journey page URL.
// Takes input (casefile.LifecycleInput) which selects the draft text.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *lifecycleJourney) prepare(ctx context.Context, start string, input casefile.LifecycleInput) bool {
	if !journey.mount(ctx, editorMountStep, start, "window.__journey.connected === 2 && !!window.piko?.partials",
		"window.__journey.original=true; window.__journey.row=document.querySelector('#retained-row')") {
		return false
	}
	if !journey.step(ctx, "click-bound", lifecycleRun(clickBound)) {
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

// leaveAndReturn confirms the leave prompt and navigates away, returns through browser
// history and clicks the page script's bound element again.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *lifecycleJourney) leaveAndReturn(ctx context.Context) bool {
	leave := journey.visit("navigate-away", `window.confirm=()=>{window.__journey.prompts++;return true};
		window.piko.nav.navigate('/journey-away')`, awayShown)
	if !journey.step(ctx, "navigate-away", leave...) {
		return false
	}
	back := append(journey.visit("back-and-dispatch", "history.back()", journeyShown),
		lifecycleRun("window.dispatchEvent(new Event('journey-request'))"))
	if !journey.step(ctx, "back-and-dispatch", back...) {
		return false
	}
	return journey.step(ctx, "click-after-back", lifecycleRun(clickBound))
}

// revisit makes SPA round trips through the `<piko:a>` links and then through browser
// history, clicking the page script's bound element after each return.
func (journey *lifecycleJourney) revisit(ctx context.Context) {
	if !journey.step(ctx, "link-away", journey.visit("link-away", lifecycleLink("away"), awayShown)...) {
		return
	}
	back := append(journey.visit("link-back-and-click", lifecycleLink("journey"), journeyShown), lifecycleRun(clickBound))
	if !journey.step(ctx, "link-back-and-click", back...) {
		return
	}
	if !journey.step(ctx, "history-back", journey.visit("history-back", "history.back()", awayShown)...) {
		return
	}
	forward := append(journey.visit("history-forward-and-click", "history.forward()", journeyShown), lifecycleRun(clickBound))
	journey.step(ctx, "history-forward-and-click", forward...)
}

// edit runs the editor journey. The journey has edits, a cancelled leave and detached
// asynchronous completions, then SPA visits between the journey and away page scripts.
//
// Takes base (*url.URL) which is the parsed application URL.
// Takes input (casefile.LifecycleInput) which selects draft text and completion order.
func (journey *lifecycleJourney) edit(ctx context.Context, base *url.URL, input casefile.LifecycleInput) {
	if !journey.prepare(ctx, base.ResolveReference(&url.URL{Path: "/journey"}).String(), input) {
		return
	}
	if !journey.step(ctx, "detach-first", lifecycleRun("document.querySelector('#first').remove()")) || !journey.release(ctx, input) {
		return
	}
	if !journey.leaveAndReturn(ctx) {
		return
	}
	journey.revisit(ctx)
}

// reuseListings runs the element-reuse journey. It makes SPA navigation between two
// listings whose filter input and pager component sit at the same position, and changes
// the shown listing's filter after each navigation.
//
// Takes base (*url.URL) which is the parsed application URL.
func (journey *lifecycleJourney) reuseListings(ctx context.Context, base *url.URL) {
	first, second := casefile.ReuseFirstListing, casefile.ReuseSecondListing
	start := base.ResolveReference(&url.URL{Path: casefile.ReuseListingPath(first)}).String()
	if !journey.mount(ctx, reuseMountStep, start, componentConnections+" === 1", "window.__journey.original=true") {
		return
	}
	if !journey.step(ctx, "reuse-filter-"+first, reuseFilter(first)) {
		return
	}
	for _, listing := range []string{second, first} {
		name := "reuse-link-" + listing
		visit := append(journey.reuseVisit(name, lifecycleLink(listing), listing), reuseFilter(listing))
		if !journey.step(ctx, name, visit...) {
			return
		}
	}
	journey.step(ctx, "reuse-history-back", append(journey.reuseVisit("reuse-history-back", "history.back()", second), reuseFilter(second))...)
}

// browserLifecycle runs the lifecycle journey the input selects.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes input (casefile.LifecycleInput) which selects the journey, draft text and
// completion order.
//
// Returns *casefile.LifecycleTrace which records lifecycle states reached before
// completion or driver failure.
func browserLifecycle(ctx context.Context, session *browser.Session, address string, input casefile.LifecycleInput) *casefile.LifecycleTrace {
	trace := new(casefile.LifecycleTrace)
	trace.Input = input
	base, err := url.Parse(address)
	if err != nil {
		trace.FailedStep, trace.DriverError = mountStep(input), err.Error()
		return trace
	}
	journey := &lifecycleJourney{session: session, trace: trace, expected: expectedStates(input)}
	if input.ElementReuse {
		journey.reuseListings(ctx, base)
		return trace
	}
	journey.edit(ctx, base, input)
	return trace
}

// mountStep names the first step of the selected lifecycle journey.
//
// Takes input (casefile.LifecycleInput) which selects the journey.
//
// Returns string which is the name of the journey's mount step.
func mountStep(input casefile.LifecycleInput) string {
	if input.ElementReuse {
		return reuseMountStep
	}
	return editorMountStep
}

// expectedStates indexes the states the model expects by step name.
//
// Takes input (casefile.LifecycleInput) which selects the journey.
//
// Returns map[string]casefile.LifecycleState which maps each step name to its state.
func expectedStates(input casefile.LifecycleInput) map[string]casefile.LifecycleState {
	states := casefile.ExpectedLifecycle(input).States
	expected := make(map[string]casefile.LifecycleState, len(states))
	for index := range states {
		expected[states[index].Step] = states[index]
	}
	return expected
}

// settledCondition builds the condition that holds once the page shows the route and
// lifecycle counts of an expected state. The counts a step changes after its navigation,
// such as clicks and deliveries, are left out.
//
// Takes expected (casefile.LifecycleState) which is the state the model expects.
//
// Returns string which is the JavaScript condition.
func settledCondition(expected casefile.LifecycleState) string {
	return fmt.Sprintf(lifecycleSettledCondition, strconv.Quote(expected.Path), expected.Connected, expected.Cleaned,
		pageScriptCondition("journey", expected.Journey), pageScriptCondition("away", expected.Away))
}

// pageScriptCondition builds the condition that holds once one page script's lifecycle
// counts match the model.
//
// Takes key (string) which names the page's counters in the journey's event log.
// Takes counts (casefile.PageScriptCounts) which holds the expected counts.
//
// Returns string which is the JavaScript condition.
func pageScriptCondition(key string, counts casefile.PageScriptCounts) string {
	return fmt.Sprintf(pageScriptSettledCondition, key, counts.Connected, counts.Disconnected, counts.Cleaned)
}

// lifecycleLink builds the script that clicks the `<piko:a>` link to a journey page.
//
// Takes page (string) which names the destination page.
//
// Returns string which is the JavaScript that clicks the link.
func lifecycleLink(page string) string {
	return "document.querySelector('#link-" + page + "').click()"
}

// reuseFilter builds an action that changes a listing's filter input.
//
// Takes listing (string) which names the listing shown.
//
// Returns browserAction which dispatches the change event.
func reuseFilter(listing string) browserAction {
	return lifecycleRun("document.querySelector('#" + listing + "-filter').dispatchEvent(new Event('change',{bubbles:true}))")
}

// lifecycleWaitFor builds an action that waits for a lifecycle readiness expression.
//
// Takes expression (string) which is the JavaScript condition.
//
// Returns browserAction which performs the bounded wait.
func lifecycleWaitFor(expression string) browserAction {
	return waitExpression(expression, lifecycleWait)
}

// lifecycleSettle builds an action that waits until a condition holds or the settle
// period ends, and succeeds either way.
//
// Takes expression (string) which is the JavaScript condition.
//
// Returns browserAction which performs the bounded wait.
func lifecycleSettle(expression string) browserAction {
	return lifecycleRun(fmt.Sprintf(lifecycleSettleScript, lifecycleSettlePeriod.Milliseconds(), expression))
}

// lifecycleRun builds an action that awaits a lifecycle script.
//
// Takes expression (string) which is the JavaScript to run.
//
// Returns browserAction which awaits the script.
func lifecycleRun(expression string) browserAction {
	return await(expression, nil)
}

// lifecycleBudget sizes the session time the selected lifecycle journey needs.
//
// Takes input (casefile.LifecycleInput) which selects the journey.
//
// Returns time.Duration which is the sum of the budgets of the steps the model expects.
func lifecycleBudget(input casefile.LifecycleInput) time.Duration {
	return time.Duration(len(casefile.ExpectedLifecycle(input).States)) * lifecycleStepBudget
}
