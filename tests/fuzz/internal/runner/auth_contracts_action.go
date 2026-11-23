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
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// publicationAttemptMax bounds the publication actions of one recipe.
	publicationAttemptMax = 8
)

// AuthActionObservation records an action reply and an independent ledger read.
type AuthActionObservation struct {
	// Name identifies the recipe step.
	Name string

	// Actor names the session that submitted the action.
	Actor string

	// Result holds the action result.
	Result casefile.AuthActionResult

	// Ledger holds an independent publication-ledger read.
	Ledger casefile.AuthLedger

	// LedgerStatus is the independent ledger read's HTTP status.
	LedgerStatus int
}

// publicationClient holds a publication client's transport and CSRF state.
type publicationClient struct {
	// http is the publication HTTP client.
	http *http.Client

	// token is the action CSRF token.
	token string

	// ephemeral is the form's ephemeral CSRF token.
	ephemeral string
}

// publicationReply is the decoded public publication action result.
type publicationReply struct {
	// By is the stored publication owner.
	By string

	// Headline is the stored publication headline.
	Headline string

	// Error is the public error message.
	Error string

	// Code is the public error code.
	Code string

	// Login is the login redirect target.
	Login string
}

// post submits a publication action and decodes its public result.
//
// Takes base (string) which is the application base URL.
// Takes name (string) which names the public action to invoke.
// Takes fields (map[string]string) which supplies action fields in a fresh map. The
// function adds the CSRF ephemeral token to the map.
// Takes value (any) which receives the decoded public action result.
//
// Returns int which is the action response HTTP status.
// Returns error which reports request construction, HTTP transport or response-decoding
// failure, or nil on success.
func (c publicationClient) post(ctx context.Context, base, name string, fields map[string]string, value any) (int, error) {
	fields["_csrf_ephemeral_token"] = c.ephemeral
	body, err := json.Marshal(fields)
	if err != nil {
		return 0, fmt.Errorf("auth/action-encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/publication."+name, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, decodeAction(response, value)
}

// observePublicationActions runs publication actions and reads the ledger independently.
//
// Takes sessions (map[string]*http.Client) which maps actor names to their independent
// HTTP clients.
// Takes base (string) which is the application base URL.
// Takes steps ([]casefile.AuthActionStep) which holds ordered proof or interaction
// observations.
//
// Returns []AuthActionObservation which lists auth action observation values in their
// recorded order.
// Returns error which reports action dispatch or an independent ledger-read failure, or
// nil on success.
func observePublicationActions(ctx context.Context, sessions map[string]*http.Client, base string, steps []casefile.AuthActionStep) ([]AuthActionObservation, error) {
	if len(steps) == 0 {
		return nil, nil
	}
	if len(steps) > publicationAttemptMax {
		return nil, errors.New("auth/action-recipe: at most eight publication attempts")
	}
	clients, err := openPublicationClients(ctx, sessions, base)
	if err != nil {
		return nil, err
	}
	var trace []AuthActionObservation
	for index := range steps {
		recipe := &steps[index]
		client, ok := clients[recipe.Actor]
		if !ok {
			return trace, fmt.Errorf("auth/action-recipe: unknown session %q", recipe.Actor)
		}
		step, err := publishAndReadLedger(ctx, client, clients["editor"], base, recipe)
		trace = append(trace, step)
		if err != nil {
			return trace, err
		}
	}
	return trace, nil
}

// openPublicationClients reads each session's action tokens from the front page.
//
// Takes sessions (map[string]*http.Client) which maps actor names to their independent
// HTTP clients.
// Takes base (string) which is the application base URL.
//
// Returns map[string]publicationClient which maps actor names to token-bearing clients.
// Returns error which reports a failed session page or missing tokens.
func openPublicationClients(ctx context.Context, sessions map[string]*http.Client, base string) (map[string]publicationClient, error) {
	clients := map[string]publicationClient{}
	for _, name := range []string{"guest", "reader", "editor"} {
		code, _, body, err := get(ctx, sessions[name], base, "/publication/front")
		if err != nil {
			return nil, err
		}
		if code != http.StatusOK {
			return nil, fmt.Errorf("auth/action-session/%s: HTTP %d", name, code)
		}
		token, ephemeral, err := pageActionTokens(string(body))
		if err != nil {
			return nil, fmt.Errorf("auth/action-session/%s: %w", name, err)
		}
		clients[name] = publicationClient{http: sessions[name], token: token, ephemeral: ephemeral}
	}
	return clients, nil
}

// publishAndReadLedger submits one publication attempt and reads the ledger as the
// editor.
//
// Takes client (publicationClient) which is the attempting session.
// Takes editor (publicationClient) which reads the ledger independently.
// Takes base (string) which is the application base URL.
// Takes recipe (*casefile.AuthActionStep) which names the attempt and its headline.
//
// Returns AuthActionObservation which records the reply and the ledger read so far.
// Returns error which reports a failed attempt or ledger read, or nil on success.
func publishAndReadLedger(ctx context.Context, client, editor publicationClient, base string, recipe *casefile.AuthActionStep) (AuthActionObservation, error) {
	var reply publicationReply
	code, err := client.post(ctx, base, "Publish", map[string]string{"Headline": recipe.Headline}, &reply)
	step := AuthActionObservation{
		Name:         recipe.Name,
		Actor:        recipe.Actor,
		Result:       casefile.AuthActionResult{Status: code, Code: cmp.Or(reply.Code, reply.Error), Login: reply.Login, By: reply.By, Headline: reply.Headline},
		Ledger:       casefile.AuthLedger{},
		LedgerStatus: 0,
	}
	if err != nil {
		return step, fmt.Errorf("auth/action/%s: HTTP %d: %w", recipe.Name, code, err)
	}
	step.LedgerStatus, err = editor.post(ctx, base, "ReadLedger", map[string]string{}, &step.Ledger)
	if err != nil {
		return step, fmt.Errorf("auth/read-ledger/%s: HTTP %d: %w", recipe.Name, step.LedgerStatus, err)
	}
	return step, nil
}

// checkPublicationActions compares action replies and independent ledger reads.
//
// Takes want ([]casefile.AuthActionStep) which holds the independently expected result.
// Takes got ([]AuthActionObservation) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkPublicationActions(want []casefile.AuthActionStep, got []AuthActionObservation) error {
	for index := range want {
		expected := &want[index]
		if index >= len(got) {
			return authMismatch{step: expected.Name, actor: expected.Actor, field: "action-missing", detail: "no HTTP or ledger receipt"}
		}
		observed := &got[index]
		for _, comparison := range []struct {
			got, want any
			name      string
		}{
			{name: "action-name", got: observed.Name, want: expected.Name}, {name: "action-session", got: observed.Actor, want: expected.Actor},
			{name: "action-status", got: observed.Result.Status, want: expected.Want.Status}, {name: "action-code", got: observed.Result.Code, want: expected.Want.Code},
			{name: "action-login", got: observed.Result.Login, want: expected.Want.Login}, {name: "action-owner", got: observed.Result.By, want: expected.Want.By},
			{name: "action-headline", got: observed.Result.Headline, want: expected.Want.Headline}, {name: "ledger-status", got: observed.LedgerStatus, want: http.StatusOK},
			{name: "handler-calls", got: observed.Ledger.Calls, want: expected.Ledger.Calls}, {name: "published-writes", got: observed.Ledger.Writes, want: expected.Ledger.Writes},
			{name: "stored-owner", got: observed.Ledger.Owner, want: expected.Ledger.Owner}, {name: "stored-headline", got: observed.Ledger.Headline, want: expected.Ledger.Headline},
		} {
			if comparison.got != comparison.want {
				return authMismatch{step: expected.Name, actor: expected.Actor, field: comparison.name, detail: fmt.Sprintf("got %v, want %v", comparison.got, comparison.want)}
			}
		}
	}
	if len(got) != len(want) {
		return authMismatch{step: "actions", actor: "all", field: "length", detail: "unexpected additional action attempts"}
	}
	return nil
}
