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
	"piko.sh/piko/tests/fuzz/internal/textlimit"
	"piko.sh/piko/wdk/browser"
)

const (
	// checkoutStepBudget bounds one checkout step.
	checkoutStepBudget = 12 * time.Second

	// checkoutStepCount is the largest number of steps in the checkout journey.
	checkoutStepCount = 8

	// checkoutWait bounds one readiness wait inside a checkout step.
	checkoutWait = 10 * time.Second

	// checkoutPlace is the ID of the button that places the order.
	checkoutPlace = "place"

	// checkoutSettled waits until the review screen leaves the pending status.
	checkoutSettled = "document.getElementById('checkout-review').dataset.status !== 'pending'"

	// checkoutDraftReady waits for the draft form and its next-step handler.
	checkoutDraftReady = "!!document.getElementById('customer') && window.piko?.context.get().hasFunction('next') === true"

	// checkoutReviewReady waits for the review screen and its restore handler after the
	// navigation completes.
	checkoutReviewReady = "window.__pikoFuzzCheckoutReady === true && !!document.getElementById('checkout-review') && " +
		"window.piko.context.get().hasFunction('restore')"

	// checkoutWatchNavigation observes the public navigation completion hook for the review
	// screen.
	checkoutWatchNavigation = `(() => {
		window.__pikoFuzzCheckoutReady=false;
		const stop=window.piko.nav.guard({afterNavigate(url){
			if(new URL(url,location.href).pathname==='/checkout-review'){
				window.__pikoFuzzCheckoutReady=true; stop();
			}
		}});
	})()`

	// checkoutSnapshot reads the checkout screen's public state. The error text is cut to
	// 256 UTF-16 code units without splitting a surrogate pair, and a cut is marked.
	checkoutSnapshot = `(() => {
	const el = id => document.getElementById(id);
	const clip = (text, limit) => {
		if (text.length <= limit) return text;
		let end = limit;
		const unit = text.charCodeAt(end - 1);
		if (unit >= 0xD800 && unit <= 0xDBFF) end--;
		return text.slice(0, end) + "` + textlimit.Marker + `";
	};
	if (el("checkout-form")) return {Path:location.pathname, Customer:el("customer").value, Quantity:Number(el("quantity").value)};
	const s=el("checkout-review").dataset;
	const integer=v=>{if (typeof v !== 'string' || !/^\d+$/.test(v)) throw Error('unavailable numeric checkout field');return Number(v)};
	return {Path:location.pathname,Customer:el("review-customer").textContent,Quantity:integer(el("review-quantity").textContent),
		Unit:integer(s.unit),Attempts:integer(s.attempts),Total:integer(s.total),Committed:integer(s.committed),
		Error:clip(el("error").textContent, 256), Status:s.status, Disabled:el("place").disabled,Display:el("receipt").textContent};
})()`
)

// checkoutPublicError is the public action error the checkout screen renders as JSON.
type checkoutPublicError struct {
	// ValidationErrors maps field names to validation messages.
	ValidationErrors map[string]string

	// Status is the action's HTTP status.
	Status int

	// IsValidationError reports a validation failure.
	IsValidationError bool

	// IsNetworkError reports a transport failure.
	IsNetworkError bool

	// IsAuthError reports an authentication or authorisation failure.
	IsAuthError bool
}

// checkoutJourney drives the checkout journey and records each screen.
type checkoutJourney struct {
	// session is the root browser session.
	session *browser.Session

	// trace receives the recorded screens and any driver failure.
	trace *casefile.CheckoutTrace
}

// run performs one checkout step and records the resulting screen.
//
// Takes step (string) which names the step.
// Takes actions (...browserAction) which lists the step's browser operations.
//
// Returns bool which is false when the step failed and the trace holds the failure.
func (journey *checkoutJourney) run(ctx context.Context, step string, actions ...browserAction) bool {
	stepCtx, cancel := stepContext(ctx, checkoutStepBudget)
	defer cancel()
	var state casefile.CheckoutState
	if err := runBrowser(stepCtx, journey.session, append(actions, evaluate(checkoutSnapshot, &state))...); err != nil {
		journey.trace.DriverError = sessionFailure(stepCtx, step, err)
		return false
	}
	state.Step = step
	if state.Error != "" {
		var public checkoutPublicError
		if err := json.Unmarshal([]byte(state.Error), &public); err == nil {
			state.Validation = &casefile.CheckoutValidation{
				Status:       public.Status,
				Fields:       public.ValidationErrors,
				IsValidation: public.IsValidationError,
				IsNetwork:    public.IsNetworkError,
				IsAuth:       public.IsAuthError,
			}
		}
	}
	journey.trace.States = append(journey.trace.States, state)
	return true
}

// review opens the draft, fills it and moves to the review screen.
//
// Takes base (*url.URL) which is the parsed application URL.
// Takes input (casefile.CheckoutInput) which supplies the checkout customer and order
// values.
//
// Returns bool which is false when a step failed and the trace holds the failure.
func (journey *checkoutJourney) review(ctx context.Context, base *url.URL, input casefile.CheckoutInput) bool {
	customer, err := scriptString(input.Customer)
	if err != nil {
		journey.trace.DriverError = "draft: " + err.Error()
		return false
	}
	fill := "document.getElementById('customer').value=" + customer +
		";document.getElementById('quantity').value=" + fmt.Sprint(input.Quantity)

	if !journey.run(ctx, "draft", navigate(base.ResolveReference(&url.URL{Path: "/checkout"}).String()),
		checkoutCondition(checkoutDraftReady), evaluate(fill, nil)) {
		return false
	}

	return journey.run(ctx, "review", evaluate(checkoutWatchNavigation, nil), checkoutClick("next"),
		checkoutCondition(checkoutReviewReady), checkoutClick("restore"),
		checkoutCondition("!!document.getElementById('review-quantity')?.textContent"))
}

// driveCheckout runs the checkout journey and records each screen.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes input (casefile.CheckoutInput) which supplies the checkout customer and order
// values.
//
// Returns *casefile.CheckoutTrace which records checkout screens reached before
// completion or driver failure.
func driveCheckout(ctx context.Context, session *browser.Session, address string, input casefile.CheckoutInput) *casefile.CheckoutTrace {
	trace := new(casefile.CheckoutTrace)
	trace.Input = input
	base, err := url.Parse(address)
	if err != nil {
		trace.DriverError = err.Error()
		return trace
	}
	journey := &checkoutJourney{session: session, trace: trace}
	if !journey.review(ctx, base, input) {
		return trace
	}
	if !journey.run(ctx, "first-pending", checkoutClick(checkoutPlace), checkoutStatus("pending")) ||
		!journey.run(ctx, "duplicate-while-pending", checkoutClick(checkoutPlace)) ||
		!journey.run(ctx, "denied", checkoutClick("deny"), checkoutStatus("retry")) {
		return trace
	}
	if !journey.run(ctx, "retry-pending", checkoutClick(checkoutPlace), checkoutStatus("pending")) ||
		!journey.run(ctx, "confirmed", checkoutClick("approve"), checkoutCondition(checkoutSettled)) {
		return trace
	}
	if trace.States[len(trace.States)-1].Status == "saved" {
		journey.run(ctx, "resubmit", checkoutClick(checkoutPlace), checkoutCondition(checkoutSettled))
	}
	return trace
}

// checkoutClick builds an action that clicks a checkout element by ID.
//
// Takes id (string) which is the element's ID.
//
// Returns browserAction which performs the click.
func checkoutClick(id string) browserAction {
	return evaluate("document.getElementById('"+id+"').click()", nil)
}

// checkoutCondition builds an action that waits for a checkout readiness expression.
//
// Takes expression (string) which is the JavaScript condition.
//
// Returns browserAction which performs the bounded wait.
func checkoutCondition(expression string) browserAction {
	return waitExpression(expression, checkoutWait)
}

// checkoutStatus builds an action that waits for the review screen's status.
//
// Takes value (string) which is the expected status.
//
// Returns browserAction which performs the bounded wait.
func checkoutStatus(value string) browserAction {
	return checkoutCondition("document.getElementById('checkout-review')?.dataset.status === '" + value + "'")
}

// checkoutBudget sizes the session time the checkout journey needs.
//
// Returns time.Duration which is the sum of the journey's step budgets.
func checkoutBudget() time.Duration {
	return checkoutStepCount * checkoutStepBudget
}
