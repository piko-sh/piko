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
	"slices"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// intakeStepBudget bounds the intake navigation and form submission.
	intakeStepBudget = 10 * time.Second

	// intakeLateEvidence bounds the wait for request metadata after submission.
	intakeLateEvidence = time.Second

	// intakeRequestLimit is the largest number of retained intake requests.
	intakeRequestLimit = 8

	// intakeCookieLimit is the largest number of retained cookie observations.
	intakeCookieLimit = 64

	// intakeExpectedRequests is the number of action requests one submission makes.
	intakeExpectedRequests = 3

	// intakeReady waits for the public form helper and the intake form.
	intakeReady = `!!window.piko?.form?.data && !!document.querySelector('form[data-case="quota"]')`

	// browserIntake submits the form using Piko's shipped action client.
	browserIntake = `(async () => {
	const form = document.querySelector('form[data-case="quota"]');
	const url = new URL(document.querySelector('script[type="module"][src$="/pk/actions.gen.js"]').src);
	if (url.origin !== location.origin || form.action !== location.origin + '/_piko/actions/intake.Submit' || form.method !== 'post')
		throw Error("intake page published an unexpected module or form");
	const {action} = await import(url.href);
	const fields = window.piko.form.data(form);
	const state = async () => {
		const value = await action.intake.Read({}).call();
		if (!Number.isInteger(value.checks) || !Number.isInteger(value.writes) || !Array.isArray(value.events) ||
			value.events.length>32 || !value.events.every(x=>typeof x==='string' && x.length<=256))
			throw Error("intake.Read returned invalid counters or journal");
		return {Checks: value.checks, Writes: value.writes, Events: value.events};
	};
	const out = {Page:location.pathname, Module:url.pathname, FormAction:new URL(form.action).pathname,
		Label:fields.get("label"), Ticket:fields.get("_captcha_token"), Before:await state(), Rejected:false};
	try { await action.intake.Submit(fields).call(); }
	catch (e) {
		out.Rejected=true;
		out.Status = e?.status;
		out.ErrorKind = e?.isNetworkError ? "network" : e?.isValidationError ? "validation" : e?.isAuthError ? "auth" : "http";
	}
	out.After=await state();
	return out;
})()`
)

// intakeRequests collects network evidence for a form submission.
type intakeRequests struct {
	// cookies tracks binding-cookie evidence by request ID.
	cookies map[browser.RequestID]bool

	// complete signals that late request metadata has arrived.
	complete chan struct{}

	// origin is the application's origin.
	origin string

	// ids lists submission request IDs in observed order.
	ids []browser.RequestID

	// rows holds the submission's HTTP evidence.
	rows []casefile.FormShimBrowserRequest

	// mu guards every other field against concurrent browser callbacks.
	mu sync.Mutex

	// overflow reports that the receipt's request bound was exceeded.
	overflow bool
}

// observe records the Chromium request and response event.
//
// Takes event (browser.NetworkEvent) which holds the incoming browser event.
//
// Concurrency: Safe for concurrent callbacks. The mutex guards the evidence.
func (calls *intakeRequests) observe(event browser.NetworkEvent) {
	calls.mu.Lock()
	defer calls.mu.Unlock()
	switch event.Kind {
	case browser.RequestStarted:
		if !calls.requestStarted(event) {
			return
		}
	case browser.RequestCookies:
		if !calls.requestCookies(event) {
			return
		}
	case browser.ResponseReceived:
		if index := slices.Index(calls.ids, event.ID); index >= 0 {
			calls.rows[index].Status = event.Status
		}
	default:
	}
	calls.signalComplete()
}

// requestStarted records a same-origin action request. The caller must hold mu.
//
// Takes event (browser.NetworkEvent) which is the request event.
//
// Returns bool which is false when the event was ignored or overflowed the receipt.
func (calls *intakeRequests) requestStarted(event browser.NetworkEvent) bool {
	location, err := url.Parse(event.URL)
	if err != nil || location.Scheme+"://"+location.Host != calls.origin || !strings.HasPrefix(location.Path, "/_piko/actions/") {
		return false
	}
	if len(calls.rows) >= intakeRequestLimit {
		calls.overflow = true
		return false
	}
	calls.ids = append(calls.ids, event.ID)
	calls.rows = append(calls.rows, casefile.FormShimBrowserRequest{Method: event.Method, Path: location.Path, Status: 0, BindingCookie: false})
	return true
}

// requestCookies records whether a request carried the binding cookie.
//
// Chromium can send this before the request itself. The caller must hold mu.
//
// Takes event (browser.NetworkEvent) which is the request cookie event.
//
// Returns bool which is false when the event overflowed the receipt.
func (calls *intakeRequests) requestCookies(event browser.NetworkEvent) bool {
	if len(calls.cookies) >= intakeCookieLimit {
		calls.overflow = true
		return false
	}
	calls.cookies[event.ID] = slices.ContainsFunc(event.Cookies, func(cookie browser.RequestCookie) bool {
		return cookie.Name == "_piko_csrf_token" && !cookie.Blocked
	})
	return true
}

// signalComplete closes the completion channel once every expected request has its cookie
// and response evidence. The caller must hold mu.
func (calls *intakeRequests) signalComplete() {
	if calls.complete == nil || len(calls.rows) != intakeExpectedRequests {
		return
	}
	for index, id := range calls.ids {
		if _, seen := calls.cookies[id]; !seen || calls.rows[index].Status == 0 {
			return
		}
	}
	close(calls.complete)
	calls.complete = nil
}

// receipt copies the collected request evidence into the browser receipt.
//
// Takes out (*casefile.FormShimBrowser) which receives the requests.
//
// Concurrency: Safe for concurrent use. The mutex protects the snapshot.
func (calls *intakeRequests) receipt(out *casefile.FormShimBrowser) {
	calls.mu.Lock()
	defer calls.mu.Unlock()
	out.Requests = slices.Clone(calls.rows)
	for index, id := range calls.ids {
		out.Requests[index].BindingCookie = calls.cookies[id]
	}
	if calls.overflow {
		out.DriverError = fmt.Sprintf("intake action count exceeded %d", intakeRequestLimit)
	}
}

// driveIntake submits the form through Piko's public browser action client.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes origin (*url.URL) which is the parsed application origin.
//
// Returns *casefile.FormShimBrowser which records the form-client result and independent
// HTTP evidence.
func driveIntake(ctx context.Context, session *browser.Session, origin *url.URL) *casefile.FormShimBrowser {
	stepCtx, cancel := stepContext(ctx, intakeStepBudget)
	defer cancel()
	out := new(casefile.FormShimBrowser)
	page := origin.ResolveReference(&url.URL{Path: "/intake"})
	if err := runBrowser(stepCtx, session, navigate(page.String()), evaluate("location.pathname", &out.Page),
		waitExpression(intakeReady, intakeStepBudget)); err != nil {
		out.DriverError = sessionFailure(stepCtx, "intake navigation", err)
		return out
	}
	calls := new(intakeRequests)
	calls.origin = page.Scheme + "://" + page.Host
	calls.cookies = make(map[browser.RequestID]bool)
	calls.complete = make(chan struct{})
	completed := calls.complete
	stop, err := session.ObserveNetwork(stepCtx, calls.observe)
	if err != nil {
		out.DriverError = sessionFailure(stepCtx, "intake action client", err)
		return out
	}
	defer stop()
	if err := session.Await(stepCtx, browserIntake, out); err != nil {
		out.DriverError = sessionFailure(stepCtx, "intake action client", err)
	} else {
		timer := time.NewTimer(intakeLateEvidence)
		defer timer.Stop()
		select {
		case <-completed:
		case <-timer.C:
		case <-stepCtx.Done():
		}
	}
	calls.receipt(out)
	return out
}

// intakeBudget sizes the session time the intake submission needs.
//
// Returns time.Duration which is the submission budget plus the late-evidence wait.
func intakeBudget() time.Duration {
	return intakeStepBudget + intakeLateEvidence
}
