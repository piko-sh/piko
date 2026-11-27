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
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// domReadyWait bounds the wait for the DOM plan's ready element.
	domReadyWait = 30 * time.Second

	// domStepBudget bounds one DOM plan step and its observation.
	domStepBudget = 10 * time.Second

	// domViewportHeight is the fixed viewport height used by viewport steps.
	domViewportHeight = 720

	// scriptGroupEnd closes a JavaScript call or bracketed expression built as a string.
	scriptGroupEnd = ")"

	// queryDOM reads the fixed public DOM query vocabulary.
	queryDOM = `(step => {
 const root = host => host ? document.querySelector(host)?.shadowRoot : document;
 const nodes = q => Array.from(root(q.Host)?.querySelectorAll(q.Selector) ?? []);
 const ids = globalThis.__pikoFuzzDOMIDs ??= {nodes:new WeakMap(),next:1};
 const read = (n,q) => {
  switch(q.Read) {
   case "text": return n.textContent;
   case "attr": if(!n.hasAttribute(q.Argument)) throw Error("missing attribute "+q.Argument); return n.getAttribute(q.Argument);
   case "present": return String(n.hasAttribute(q.Argument));
   case "value": return n.value;
   case "namespace": return n.namespaceURI;
   case "tag": return n.localName;
   case "parent": return n.parentElement?.localName ?? "";
   case "identity": if(!ids.nodes.has(n)) ids.nodes.set(n,ids.next++); return String(ids.nodes.get(n));
   case "focus": return String(n.getRootNode().activeElement===n);
   case "selection": return n.selectionStart+"/"+n.selectionEnd;
   case "open": return String(n.open);
   case "checked": return String(n.checked);
   case "valid": return String(n.validity.valid);
   case "slot": return JSON.stringify(n.assignedNodes({flatten:true}).map(x=>x.textContent));
   case "cell": return n.parentElement.rowIndex+"/"+n.cellIndex;
   case "rows": return String(n.rows.length);
   case "style": return getComputedStyle(n).getPropertyValue(q.Argument);
   case "form": return JSON.stringify(Array.from(new FormData(n)));
   case "image": return String(n.complete && n.naturalWidth>0 && !!n.currentSrc);
   case "pixel": return String(n.getContext("2d").getImageData(0,0,1,1).data);
   case "media": return [n.paused,n.controls,n.preload].join("/");
   case "visible": return String(n.checkVisibility());
  }
 };
 const Values={}; for(const f of step.Fields) Values[f.ID]=nodes(f.Query).map(n=>read(n,f.Query));
 return {Step:step.ID,Path:location.pathname+location.search+location.hash,Values};
})`
)

// domJourney runs a DOM plan and retains partial evidence on failure.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes base (string) which is the application base URL.
// Takes plan (casefile.DOMPlan) which holds the saved DOM operations and expected values.
// Takes settle (browserAction) which waits for queued browser rendering.
//
// Returns *casefile.DOMTrace which records DOM observations reached before completion or
// driver failure.
func domJourney(ctx context.Context, session *browser.Session, base string, plan casefile.DOMPlan, settle browserAction) *casefile.DOMTrace {
	trace := new(casefile.DOMTrace)
	trace.Engine = "chromium"
	if err := runDOM(ctx, session, base, plan, settle, trace); err != nil {
		trace.DriverError = sessionFailure(ctx, "", err)
	}
	return trace
}

// runDOM performs a DOM plan and appends its observations.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes base (string) which is the application base URL.
// Takes plan (casefile.DOMPlan) which holds the saved DOM operations and expected values.
// Takes settle (browserAction) which waits for queued browser rendering.
// Takes trace (*casefile.DOMTrace) which receives the browser's observations, including
// partial progress.
//
// Returns error which reports a browser operation or DOM observation failure, or nil on
// success.
func runDOM(ctx context.Context, session *browser.Session, base string, plan casefile.DOMPlan, settle browserAction, trace *casefile.DOMTrace) error {
	plan, err := plan.DriverPlan()
	if err != nil {
		return err
	}
	origin, err := url.Parse(base)
	if err != nil {
		return err
	}
	destination, err := origin.Parse(plan.Path)
	if err != nil {
		return err
	}
	ready, err := domSelector(plan.ReadyHost, plan.ReadyElement)
	if err != nil {
		return err
	}
	if err := runBrowser(ctx, session, navigate(destination.String()), waitExpression("!!("+ready+scriptGroupEnd, domReadyWait)); err != nil {
		return fmt.Errorf("visit %s: %w", plan.Path, err)
	}
	if err := session.Evaluate(ctx, "navigator.userAgent", &trace.Version); err != nil {
		return err
	}
	for _, step := range plan.Steps {
		state, err := runDOMStep(ctx, session, step, settle)
		if err != nil {
			return fmt.Errorf("%s: %w", step.ID, err)
		}
		trace.States = append(trace.States, state)
	}
	return nil
}

// runDOMStep performs one DOM plan step within its own budget and observes the result.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes step (casefile.DOMStep) which is the operation and its queries.
// Takes settle (browserAction) which waits for queued browser rendering.
//
// Returns casefile.DOMState which is the step's observation.
// Returns error which reports an unsupported key, browser operation or observation
// failure, or nil on success.
func runDOMStep(ctx context.Context, session *browser.Session, step casefile.DOMStep, settle browserAction) (casefile.DOMState, error) {
	var state casefile.DOMState
	target, err := domSelector(step.Host, step.Target)
	if err != nil {
		return state, err
	}
	var tasks browserActions
	switch step.Op {
	case "viewport":
		tasks = append(tasks, viewport(int64(step.ViewportWidth()), domViewportHeight))
	case "click":
		tasks = append(tasks, evaluate(target+".click()", nil))
	case "focus":
		tasks = append(tasks, evaluate(target+".focus()", nil))
	case "type":
		tasks = append(tasks, evaluate(target+".focus()", nil), typeKeys(step.Value))
	case "key":
		switch step.Value {
		case "End", "Home", "Enter", "Escape", "Tab":
			tasks = append(tasks, evaluate(target+".focus()", nil), pressKey(step.Value))
		default:
			return state, fmt.Errorf("unsupported key %q", step.Value)
		}
	}
	input, err := json.Marshal(step)
	if err != nil {
		return state, fmt.Errorf("encoding DOM step: %w", err)
	}
	tasks = append(tasks, settle, evaluate(queryDOM+"("+string(input)+scriptGroupEnd, &state))
	stepCtx, cancel := stepContext(ctx, domStepBudget)
	defer cancel()
	return state, tasks.run(stepCtx, session)
}

// domSelector selects an element in a document or component shadow root.
//
// Takes host (string) which selects the component shadow root.
// Takes selector (string) which selects the target element.
//
// Returns string which is the document or shadow-root selector expression.
// Returns error which reports a selector that cannot be encoded, or nil on success.
func domSelector(host, selector string) (string, error) {
	quotedSelector, err := scriptString(selector)
	if err != nil {
		return "", err
	}
	if host == "" {
		return "document.querySelector(" + quotedSelector + scriptGroupEnd, nil
	}
	quotedHost, err := scriptString(host)
	if err != nil {
		return "", err
	}
	return "document.querySelector(" + quotedHost + ")?.shadowRoot?.querySelector(" + quotedSelector + scriptGroupEnd, nil
}

// domBudget sizes the session time a DOM plan needs.
//
// Takes plan (casefile.DOMPlan) which holds the saved DOM operations.
//
// Returns time.Duration which is the readiness wait plus one step budget per operation.
func domBudget(plan casefile.DOMPlan) time.Duration {
	return domReadyWait + time.Duration(len(plan.Steps))*domStepBudget
}
