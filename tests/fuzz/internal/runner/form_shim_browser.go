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
	"fmt"
	"net/http"
	"slices"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// formActionModule is the generated public browser action module path.
	formActionModule = "/_piko/assets/pk-js/pk/actions.gen.js"

	// formBrowserSubject prefixes every browser form-shim failure subject.
	formBrowserSubject = "form-shim/browser"
)

// checkIntakeState compares an independent form-state read with its model.
//
// Takes subject (string) which identifies the checked property.
// Takes want (casefile.FormShimState) which holds the independently expected result.
// Takes got (*casefile.FormShimState) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkIntakeState(subject string, want casefile.FormShimState, got *casefile.FormShimState) error {
	if got == nil {
		return responseMismatchf(subject+"/missing", "%s: missing intake.Read result", subject)
	}
	if want.Checks != got.Checks {
		return responseMismatchf(subject+"/checks", "%s: got %d provider checks, want %d", subject, got.Checks, want.Checks)
	}
	if want.Writes != got.Writes {
		return responseMismatchf(subject+"/writes", "%s: got %d writes, want %d", subject, got.Writes, want.Writes)
	}
	if !slices.Equal(want.Events, got.Events) {
		return responseMismatchf(subject+"/events", "%s: got %v, want %v", subject, got.Events, want.Events)
	}
	return nil
}

// checkBrowserIntake checks the browser's form submission and real network evidence.
//
// Takes model (*casefile.FormShimContract) which holds the independent expected
// interaction model.
// Takes server (*FormShimReceipt) which holds the independent server-side form receipt.
// Takes client (*casefile.ClientTransition) which holds the saved or observed component
// interactions.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkBrowserIntake(model *casefile.FormShimContract, server *FormShimReceipt, client *casefile.ClientTransition) error {
	if model == nil || !model.BrowserSubmission || client == nil {
		return nil
	}
	const subject = formBrowserSubject
	browserReceipt := client.FormShim
	if browserReceipt == nil {
		return responseMismatchf(subject+"/missing", "%s: missing Chromium receipt", subject)
	}
	if browserReceipt.DriverError != "" {
		return responseMismatchf(subject+"/driver", "%s: %s", subject, browserReceipt.DriverError)
	}
	if len(model.Steps) == 0 || server == nil || len(server.Steps) != len(model.Steps) {
		return responseMismatchf(subject+"/http-history", "%s: missing final HTTP ledger", subject)
	}
	quota := &model.Steps[len(model.Steps)-1]
	if quota.Status != http.StatusTooManyRequests || len(server.Steps) == 0 || server.Steps[len(server.Steps)-1].Status != http.StatusTooManyRequests {
		return responseMismatchf(subject+"/http-quota", "%s: HTTP history did not end with a quota refusal", subject)
	}
	if err := checkBrowserIntakeFields(model, quota, browserReceipt); err != nil {
		return err
	}
	if err := checkBrowserIntakeRequests(browserReceipt.Requests); err != nil {
		return err
	}
	if err := checkIntakeState(subject+"/initial-model", quota.After, browserReceipt.Before); err != nil {
		return err
	}
	if err := checkIntakeState(subject+"/http-ledger", server.Steps[len(server.Steps)-1].After, browserReceipt.Before); err != nil {
		return err
	}
	return checkIntakeState(subject+"/after", *browserReceipt.Before, browserReceipt.After)
}

// checkBrowserIntakeFields compares the browser's form fields and quota rejection.
//
// Takes model (*casefile.FormShimContract) which holds the expected page.
// Takes quota (*casefile.FormShimStep) which is the modelled quota-refused step.
// Takes browserReceipt (*casefile.FormShimBrowser) which holds the browser receipt.
//
// Returns error which reports the first mismatched field or rejection property.
func checkBrowserIntakeFields(model *casefile.FormShimContract, quota *casefile.FormShimStep, browserReceipt *casefile.FormShimBrowser) error {
	for _, comparison := range []struct{ name, got, want string }{
		{name: "page", got: browserReceipt.Page, want: model.Page},
		{name: "module", got: browserReceipt.Module, want: formActionModule},
		{name: "action", got: browserReceipt.FormAction, want: "/_piko/actions/intake.Submit"},
		{name: "label", got: browserReceipt.Label, want: quota.Label},
		{name: "ticket", got: browserReceipt.Ticket, want: quota.Ticket},
	} {
		if comparison.got != comparison.want {
			return responseMismatchf(formBrowserSubject+"/"+comparison.name, "%s: got %q, want %q", comparison.name, comparison.got, comparison.want)
		}
	}
	if !browserReceipt.Rejected || browserReceipt.Status != http.StatusTooManyRequests || browserReceipt.ErrorKind != "http" {
		return responseMismatchf(formBrowserSubject+"/quota-error", "%s: public client rejected=%t HTTP %d type %q",
			formBrowserSubject, browserReceipt.Rejected, browserReceipt.Status, browserReceipt.ErrorKind)
	}
	return nil
}

// checkBrowserIntakeRequests checks the browser's read, submit and read action requests.
//
// Takes requests ([]casefile.FormShimBrowserRequest) which lists the observed requests.
//
// Returns error which reports a wrong count, route, status or missing binding cookie.
func checkBrowserIntakeRequests(requests []casefile.FormShimBrowserRequest) error {
	routes := []string{"Read", "Submit", "Read"}
	if len(requests) != len(routes) {
		return responseMismatchf(formBrowserSubject+"/requests", "%s: got %d action requests, want three", formBrowserSubject, len(requests))
	}
	for index, route := range routes {
		request := &requests[index]
		name := fmt.Sprintf("%s/request-%d", formBrowserSubject, index+1)
		status := http.StatusOK
		if route == "Submit" {
			status = http.StatusTooManyRequests
		}
		if request.Method != http.MethodPost || request.Path != "/_piko/actions/intake."+route || request.Status != status {
			return responseMismatchf(name+"/http", "%s: got %s %s HTTP %d", name, request.Method, request.Path, request.Status)
		}
		if !request.BindingCookie {
			return responseMismatchf(name+"/cookie", "%s: Chromium did not send its binding cookie", name)
		}
	}
	return nil
}
