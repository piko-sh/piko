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
	"strings"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// partialStepBudget bounds one partial-component step.
	partialStepBudget = 7 * time.Second

	// partialStepCount is the number of steps before the optional reload modes.
	partialStepCount = 12

	// partialWait bounds one readiness wait inside a partial-component step.
	partialWait = 5 * time.Second

	// revisionEdit is the action revision after the edit mutation.
	revisionEdit = 1

	// revisionReorder is the action revision after the reorder mutation.
	revisionReorder = 2

	// revisionRemove is the action revision after the remove mutation.
	revisionRemove = 3

	// revisionRestore is the action revision after the restore mutation.
	revisionRestore = 4

	// partialReady waits for the record rows and every record chip's controls.
	partialReady = `document.getElementById('record-rows') && Array.from(document.querySelectorAll('record-chip')).length>0 && ` +
		`Array.from(document.querySelectorAll('record-chip')).every(h=>h.shadowRoot?.querySelector('#pick'))`

	// partialDetachedEvent dispatches a record event from the detached record chip.
	partialDetachedEvent = "window.__detachedRecord.dispatchEvent(new CustomEvent('record-picked'," +
		"{bubbles:true,composed:true,detail:{label:'detached',count:99}}))"

	// recordSnapshot reads partial-component rows and their node identities.
	recordSnapshot = `(() => {
	const el=id=>document.getElementById(id), path=location.pathname+location.search+location.hash;
	const number=s=>{if(typeof s!='string'||!/^\d+$/.test(s))throw new Error('missing integer: '+s);return Number(s)};
	const detail=el('record-detail');
	if(detail) return {Path:path,Locale:document.documentElement.lang,Detail:el('summary').textContent,
		Revision:number(detail.dataset.revision),ActionRevision:-1};
	const holder=el('record-rows');
	const ids=window.__recordIDs??={rows:new WeakMap(),hosts:new WeakMap(),rowSeq:0,hostSeq:0};
	const id=(node,map,next)=>{if(!map.has(node))map.set(node,++ids[next]);return map.get(node);};
	const rows=Array.from(holder.querySelectorAll('article[data-record]'), row=>{
		const host=row.querySelector('record-chip'),root=host.shadowRoot;
		const link=row.querySelector('a'),href=new URL(link.href),svg=link.querySelector('svg'),use=svg?.querySelector('use');
		const ref=use?.getAttribute('href')||'',symbol=ref.startsWith('#')?document.getElementById(ref.slice(1)):null,p=symbol?.querySelector('path');
		const point=n=>{const a=p.getPointAtLength(n);return Math.round(a.x)+','+Math.round(a.y);};
		const icon=svg && p && symbol.localName==='symbol' && use.namespaceURI===p.namespaceURI && p.namespaceURI==='http://www.w3.org/2000/svg' ?
			[ref,svg.getAttribute('viewBox'),symbol.getAttribute('viewBox'),point(0),point(p.getTotalLength()),Math.round(1000*p.getTotalLength())].join('|') :
			'missing SVG';
		return {Key:row.dataset.record,Label:host.getAttribute('label'),Score:number(host.getAttribute('score')),
			Shadow:root.querySelector('#label').textContent,ShadowScore:number(root.querySelector('#score').textContent),
			Count:number(root.querySelector('#count').textContent),Caption:root.querySelector('slot[name=caption]').assignedElements().map(x=>x.textContent).join(''),
			HostIdentity:id(host,ids.hosts,'hostSeq'),RowIdentity:id(row,ids.rows,'rowSeq'),
			Link:href.pathname+href.search+href.hash,Name:link.getAttribute('aria-label'),Icon:icon};
	});
	return {Path:path,Locale:document.documentElement.lang,Log:el('record-log').textContent,Revision:number(holder.dataset.revision),
		ActionRevision:number(el('record-action').dataset.revision),Rows:rows};
})()`
)

// partialJourney drives the partial-component journey and records its states.
type partialJourney struct {
	// session is the root browser session.
	session *browser.Session

	// trace receives the recorded states and any driver failure.
	trace *casefile.PartialComponentsTrace
}

// run performs one journey step, waits for rendering and records the resulting state.
//
// Takes step (string) which names the step. Reload steps read the reload snapshot.
// Takes actions (...browserAction) which lists the step's browser operations.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *partialJourney) run(ctx context.Context, step string, actions ...browserAction) bool {
	stepCtx, cancel := stepContext(ctx, partialStepBudget)
	defer cancel()
	var state casefile.PartialComponentState
	snapshot := recordSnapshot
	if strings.HasPrefix(step, "reload-") {
		snapshot = reloadSnapshot
	}
	actions = append(actions, await("new Promise(ok=>requestAnimationFrame(()=>requestAnimationFrame(()=>ok(true))))", nil), evaluate(snapshot, &state))
	if err := runBrowser(stepCtx, journey.session, actions...); err != nil {
		journey.trace.DriverError = sessionFailure(stepCtx, step, err)
		return false
	}
	state.Step = step
	journey.trace.States = append(journey.trace.States, state)
	return true
}

// click selects a record by clicking its chip's pick button.
//
// Takes step (string) which names the step.
// Takes key (string) which is the record's key.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *partialJourney) click(ctx context.Context, step, key string) bool {
	return journey.run(ctx, step, evaluate(partialHost(key)+".shadowRoot.querySelector('#pick').click()", nil))
}

// mutate runs a record mutation and waits for its action revision and the re-rendered
// rows.
//
// Takes operation (string) which names the mutation and its button.
// Takes revision (int) which is the action revision the mutation produces.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *partialJourney) mutate(ctx context.Context, operation string, revision int) bool {
	changed := fmt.Sprintf("document.getElementById('record-action').dataset.revision!==%q", fmt.Sprint(revision-1))
	return journey.run(ctx, operation, evaluate("document.getElementById('record-"+operation+"').click()", nil),
		partialWaitFor(changed), partialWaitFor(partialReady))
}

// edit selects, edits, reorders, removes and restores the records.
//
// Takes first (string) which is the key of the edited record.
// Takes other (string) which is the key of the sibling record.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *partialJourney) edit(ctx context.Context, first, other string) bool {
	if !journey.click(ctx, "select", first) || !journey.mutate(ctx, "edit", revisionEdit) || !journey.click(ctx, "select-edited", first) ||
		!journey.mutate(ctx, "reorder", revisionReorder) || !journey.click(ctx, "select-sibling", other) {
		return false
	}
	if err := journey.session.Evaluate(ctx, "window.__detachedRecord="+partialHost(other)+"; true", nil); err != nil {
		journey.trace.DriverError = sessionFailure(ctx, "remove", err)
		return false
	}
	return journey.mutate(ctx, "remove", revisionRemove) && journey.run(ctx, "detached-event", evaluate(partialDetachedEvent, nil)) &&
		journey.mutate(ctx, "restore", revisionRestore) && journey.click(ctx, "select-restored", other)
}

// browserPartialComponents runs the partial-component journey in Chromium.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes input (casefile.PartialComponentsInput) which selects the partial-component
// journey values.
//
// Returns *casefile.PartialComponentsTrace which records partial-component states reached
// before completion or driver failure.
func browserPartialComponents(ctx context.Context, session *browser.Session, address string, input casefile.PartialComponentsInput) *casefile.PartialComponentsTrace {
	trace := new(casefile.PartialComponentsTrace)
	trace.Input = input
	base, err := url.Parse(address)
	if err != nil {
		trace.DriverError = err.Error()
		return trace
	}
	journey := &partialJourney{session: session, trace: trace}
	if !journey.run(ctx, "mount", navigate(base.ResolveReference(&url.URL{Path: input.ListPath()}).String()), partialWaitFor(partialReady)) {
		return trace
	}
	first, other := "amber", "blue"
	if input.EditSecond {
		first, other = other, first
	}
	if !journey.edit(ctx, first, other) {
		return trace
	}
	if !journey.run(ctx, "detail", evaluate(fmt.Sprintf("document.querySelector('article[data-record=%s] a').click()", other), nil),
		partialWaitFor("!!document.getElementById('record-detail')")) {
		return trace
	}
	if journey.run(ctx, "back", evaluate("history.back()", nil), partialWaitFor(partialReady)) && input.ReloadModes {
		run := func(step string, actions ...browserAction) bool { return journey.run(ctx, step, actions...) }
		if err := driveReloadModes(base, input, run, partialWaitFor); err != nil {
			trace.DriverError = err.Error()
		}
	}
	return trace
}

// partialHost selects a record's chip element.
//
// Takes key (string) which is the record's key.
//
// Returns string which is the chip selector expression.
func partialHost(key string) string {
	return fmt.Sprintf("document.querySelector('article[data-record=%s] record-chip')", key)
}

// partialWaitFor builds an action that waits for a partial-component readiness
// expression.
//
// Takes expression (string) which is the JavaScript condition.
//
// Returns browserAction which performs the bounded wait.
func partialWaitFor(expression string) browserAction {
	return waitExpression(expression, partialWait)
}

// partialComponentsBudget sizes the session time the partial-component journey needs.
//
// Takes input (casefile.PartialComponentsInput) which selects whether reload modes run.
//
// Returns time.Duration which is the sum of the journey's step budgets.
func partialComponentsBudget(input casefile.PartialComponentsInput) time.Duration {
	steps := partialStepCount
	if input.ReloadModes {
		steps += len(input.ReloadOrder())
	}
	return time.Duration(steps) * partialStepBudget
}
