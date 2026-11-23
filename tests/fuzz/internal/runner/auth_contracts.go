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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// authStepMax bounds the publication requests of one authentication recipe.
	authStepMax = 24
)

// AuthReceipt contains only the fixed publication's HTTP and document evidence.
type AuthReceipt struct {
	// Steps holds the ordered interaction steps.
	Steps []AuthObservation

	// Actions holds the ordered action interactions.
	Actions []AuthActionObservation
}

// AuthObservation records one publication request's response.
type AuthObservation struct {
	// Name identifies the publication request.
	Name string

	// Actor selects the request's session.
	Actor string

	// Path is the requested publication route.
	Path string

	// Response holds the observed authentication response.
	Response casefile.AuthResponse
}

// authMismatch identifies an authentication field mismatch.
type authMismatch struct {
	// step identifies the failed request.
	step string

	// actor identifies the request's session.
	actor string

	// field identifies the mismatched authentication property.
	field string

	// detail describes the mismatch.
	detail string
}

// authSession pairs a publication actor with its session ticket.
type authSession struct {
	// name identifies the actor.
	name string

	// ticket is the actor's session cookie value, or empty for a guest.
	ticket string
}

// authTextField pairs a publication element ID with the response field it fills.
type authTextField struct {
	// destination receives the element text.
	destination *string

	// id is the publication element ID.
	id string
}

// authField is one compared public authentication property.
type authField struct {
	// got holds the observed value.
	got any

	// want holds the expected value.
	want any

	// name identifies the property.
	name string

	// mismatch reports whether the values differ.
	mismatch bool
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e authMismatch) Error() string {
	return fmt.Sprintf("auth/%s/%s/%s: %s", e.step, e.actor, e.field, e.detail)
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e authMismatch) FailureSubject() string {
	return "/auth/" + e.step + "/" + e.actor + "/" + e.field
}

// readAuthResponse reads one publication response without following redirects.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes step (casefile.AuthStep) which identifies the current history step.
//
// Returns AuthObservation which records the publication request's public response fields.
// Returns error which reports session setup, HTTP read or public authentication response
// failure, or nil on success.
func readAuthResponse(ctx context.Context, client *http.Client, base string, step casefile.AuthStep) (AuthObservation, error) {
	o := AuthObservation{Name: step.Name, Actor: step.Actor, Path: step.Path, Response: casefile.AuthResponse{}}
	switch step.Path {
	case "/publication/front", "/publication/desk", "/publication/profile", "/publication/profile?meeting=paired", "/publication/profile?edition=renewal":
	default:
		return o, fmt.Errorf("auth/route: unknown publication route %q", step.Path)
	}
	code, header, body, err := get(ctx, client, base, step.Path)
	if err != nil {
		return o, err
	}
	response := &o.Response
	response.Status = code
	response.Location = header.Get("Location")
	for _, cookie := range (&http.Response{Header: header}).Cookies() {
		if cookie.Name == "publication_visit" {
			response.IssuedVisit = append(response.IssuedVisit, cookie.Value)
		}
	}

	if code == http.StatusOK && strings.HasPrefix(step.Path, "/publication/profile") {
		response.CacheControl = header.Get("Cache-Control")
	}
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return o, err
	}
	publication := findID(document, "publication")
	if publication == nil {
		return o, nil
	}
	response.Page = attr(publication, "data-page")
	fields := []authTextField{
		{id: "user", destination: &response.User},
		{id: "role", destination: &response.Role},
		{id: "meeting", destination: &response.Rendezvous},
		{id: "last-visit", destination: &response.PreviousVisit},
	}
	for _, field := range fields {
		if node := findID(publication, field.id); node != nil {
			*field.destination = contents(node)
		}
	}
	return o, nil
}

// observeAuth runs the saved publication session and action history.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes want (*casefile.AuthContract) which holds the independently expected result.
//
// Returns *AuthReceipt which records authentication pages and publication actions reached
// before failure.
// Returns error which reports session setup, HTTP read or public authentication response
// failure, or nil on success.
func observeAuth(ctx context.Context, client *http.Client, base string, want *casefile.AuthContract) (*AuthReceipt, error) {
	if want == nil {
		return nil, nil
	}
	if len(want.Steps) == 0 || len(want.Steps) > authStepMax {
		return nil, fmt.Errorf("auth/recipe: expected 1 to %d publication requests", authStepMax)
	}
	sessions, err := openAuthSessions(client, base, want)
	if err != nil {
		return nil, err
	}
	o := &AuthReceipt{Steps: nil, Actions: nil}
	o.Steps, err = observeAuthSteps(ctx, sessions, base, want.Steps)
	if err != nil {
		return o, err
	}
	o.Actions, err = observePublicationActions(ctx, sessions, base, want.Actions)
	return o, err
}

// openAuthSessions creates the guest, reader and editor sessions the recipe names.
//
// Takes client (*http.Client) which is the HTTP client the sessions derive from.
// Takes base (string) which is the application base URL.
// Takes want (*casefile.AuthContract) which supplies the session tickets and steps.
//
// Returns map[string]*http.Client which maps each actor name to its session client.
// Returns error which reports session setup failure or a step naming an unknown session.
func openAuthSessions(client *http.Client, base string, want *casefile.AuthContract) (map[string]*http.Client, error) {
	actors := []authSession{
		{name: "guest", ticket: ""},
		{name: "reader", ticket: want.ReaderSession},
		{name: "editor", ticket: want.EditorSession},
	}
	sessions := make(map[string]*http.Client, len(actors))
	for _, actor := range actors {
		var probe casefile.Probe
		probe.Actor = actor.name
		if actor.ticket != "" {
			probe.Cookies = map[string]string{"publication_session": actor.ticket}
		}
		if _, err := actorClient(client, sessions, base, probe); err != nil {
			return nil, err
		}
	}
	for index := range want.Steps {
		if sessions[want.Steps[index].Actor] == nil {
			return nil, fmt.Errorf("auth/recipe: unknown publication session %q", want.Steps[index].Actor)
		}
	}
	return sessions, nil
}

// observeAuthSteps reads the recipe's publication requests, overlapping paired steps.
//
// Takes sessions (map[string]*http.Client) which maps actor names to session clients.
// Takes base (string) which is the application base URL.
// Takes steps ([]casefile.AuthStep) which lists the publication requests in order.
//
// Returns []AuthObservation which records the responses read before any failure.
// Returns error which reports an invalid overlap or a failed request, or nil on success.
func observeAuthSteps(ctx context.Context, sessions map[string]*http.Client, base string, steps []casefile.AuthStep) ([]AuthObservation, error) {
	observed := make([]AuthObservation, 0, len(steps))
	for index := 0; index < len(steps); {
		width, err := authBatchWidth(steps, index)
		if err != nil {
			return observed, err
		}
		results, failures := readAuthBatch(ctx, sessions, base, steps[index:index+width])
		for offset := range results {
			observed = append(observed, results[offset])
			if failures[offset] != nil {
				return observed, fmt.Errorf("auth/request/%s/%s: %w", results[offset].Name, results[offset].Actor, failures[offset])
			}
		}
		index += width
	}
	return observed, nil
}

// authBatchWidth returns how many steps starting at index run together.
//
// Takes steps ([]casefile.AuthStep) which lists the publication requests in order.
// Takes index (int) which is the first step of the batch.
//
// Returns int which is 2 for an overlapping pair and 1 otherwise.
// Returns error which reports an overlap without two distinct sessions.
func authBatchWidth(steps []casefile.AuthStep, index int) (int, error) {
	current := &steps[index]
	if !current.Parallel {
		return 1, nil
	}
	if index+1 >= len(steps) || !steps[index+1].Parallel || steps[index+1].Actor == current.Actor {
		return 0, errors.New("auth/recipe: overlap requires two distinct sessions")
	}
	return 2, nil
}

// readAuthBatch reads one batch of publication requests concurrently.
//
// Takes sessions (map[string]*http.Client) which maps actor names to session clients.
// Takes base (string) which is the application base URL.
// Takes batch ([]casefile.AuthStep) which lists the requests that overlap.
//
// Returns []AuthObservation which holds each request's response in batch order.
// Returns []error which holds each request's failure in batch order.
func readAuthBatch(ctx context.Context, sessions map[string]*http.Client, base string, batch []casefile.AuthStep) ([]AuthObservation, []error) {
	results := make([]AuthObservation, len(batch))
	failures := make([]error, len(batch))
	var group sync.WaitGroup
	for offset := range batch {
		step := &batch[offset]
		session := sessions[step.Actor]
		group.Go(func() { results[offset], failures[offset] = readAuthResponse(ctx, session, base, *step) })
	}
	group.Wait()
	return results, failures
}

// checkAuthResponse compares the public authentication fields for one request.
//
// Takes step (string) which identifies the current history step.
// Takes actor (string) which identifies the request's session.
// Takes want (casefile.AuthResponse) which holds the independently expected result.
// Takes got (casefile.AuthResponse) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAuthResponse(step, actor string, want, got casefile.AuthResponse) error {
	fields := []authField{
		{name: "status", mismatch: got.Status != want.Status, got: got.Status, want: want.Status},
		{name: "location", mismatch: got.Location != want.Location, got: got.Location, want: want.Location},
		{name: "page", mismatch: got.Page != want.Page, got: got.Page, want: want.Page},
		{name: "identity", mismatch: got.User != want.User, got: got.User, want: want.User},
		{name: "role", mismatch: got.Role != want.Role, got: got.Role, want: want.Role},
		{name: "overlap", mismatch: got.Rendezvous != want.Rendezvous, got: got.Rendezvous, want: want.Rendezvous},
		{name: "issued-visit", mismatch: !slices.Equal(got.IssuedVisit, want.IssuedVisit), got: got.IssuedVisit, want: want.IssuedVisit},
		{name: "previous-visit", mismatch: got.PreviousVisit != want.PreviousVisit, got: got.PreviousVisit, want: want.PreviousVisit},
		{name: "cache-control", mismatch: got.CacheControl != want.CacheControl, got: got.CacheControl, want: want.CacheControl},
	}
	for _, field := range fields {
		if field.mismatch {
			return authMismatch{step: step, actor: actor, field: field.name, detail: fmt.Sprintf("got %v, want %v", field.got, field.want)}
		}
	}
	return nil
}

// checkAuth compares publication responses and writes with the saved contract.
//
// Takes want (*casefile.AuthContract) which holds the independently expected result.
// Takes got (*AuthReceipt) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAuth(want *casefile.AuthContract, got *AuthReceipt) error {
	if want == nil && got == nil {
		return nil
	}
	if want == nil || got == nil {
		return authMismatch{step: "trace", actor: "all", field: "presence", detail: "missing or unexpected publication trace"}
	}
	for index := range want.Steps {
		expected := &want.Steps[index]
		if index >= len(got.Steps) {
			return authMismatch{step: expected.Name, actor: expected.Actor, field: "missing", detail: "missing HTTP observation"}
		}
		observed := got.Steps[index]
		if observed.Name != expected.Name || observed.Actor != expected.Actor || observed.Path != expected.Path {
			detail := fmt.Sprintf("got %s %s %s, want %s %s %s", observed.Name, observed.Actor, observed.Path, expected.Name, expected.Actor, expected.Path)
			return authMismatch{step: expected.Name, actor: expected.Actor, field: "request", detail: detail}
		}
		if err := checkAuthResponse(expected.Name, expected.Actor, expected.Want, observed.Response); err != nil {
			return err
		}
	}
	if len(got.Steps) != len(want.Steps) {
		return authMismatch{step: "trace", actor: "all", field: "length", detail: "unexpected additional requests"}
	}
	return checkPublicationActions(want.Actions, got.Actions)
}

// compareAuth compares publication observations across server modes.
//
// Takes a (*AuthReceipt) which holds the first value to compare.
// Takes b (*AuthReceipt) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareAuth(a, b *AuthReceipt) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil {
		return authMismatch{step: "trace", actor: "all", field: "presence", detail: "one host omitted the trace"}
	}
	want := &casefile.AuthContract{}
	for index := range a.Steps {
		step := &a.Steps[index]
		want.Steps = append(want.Steps, casefile.AuthStep{Name: step.Name, Actor: step.Actor, Path: step.Path, Want: step.Response, Parallel: false})
	}
	for index := range a.Actions {
		action := &a.Actions[index]
		if index < len(b.Actions) && b.Actions[index].LedgerStatus != action.LedgerStatus {
			detail := fmt.Sprintf("got %d, want %d", b.Actions[index].LedgerStatus, action.LedgerStatus)
			return authMismatch{step: action.Name, actor: action.Actor, field: "ledger-status", detail: detail}
		}
		want.Actions = append(want.Actions, casefile.AuthActionStep{Name: action.Name, Actor: action.Actor, Want: action.Result, Ledger: action.Ledger, Headline: ""})
	}
	return checkAuth(want, b)
}
