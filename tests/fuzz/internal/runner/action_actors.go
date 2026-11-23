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
	"sync"
	"time"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// actorRequestTimeout bounds the paired actor requests.
	actorRequestTimeout = 6 * time.Second
)

var (
	// errActorRequestTimeout is the cause recorded when the paired actor requests overrun.
	errActorRequestTimeout = errors.New("action/actor-request: rendezvous timed out")

	// actorNames names the two isolated actors in request order.
	actorNames = [2]string{"left", "right"}

	// actorHeaders holds the per-actor request header values in request order.
	actorHeaders = [2]string{"first", "second"}
)

// ActorRead records an actor's cookie, receiver and counter state.
type ActorRead struct {
	// Label is the actor-local input label.
	Label string

	// Before is the actor header value read before the rendezvous.
	Before string

	// After is the actor header value read after the rendezvous.
	After string

	// Cookie is the actor's cookie value.
	Cookie string

	// Number is the bound argument value.
	Number int64

	// Receiver is the actor receiver value.
	Receiver int64
}

// expectedActors models independent actor cookies, headers and receiver values.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns []ActorRead which lists actor read values in their recorded order.
func expectedActors(input casefile.ActionValue) []ActorRead {
	quantity := int64(input.Quantity)
	return []ActorRead{
		{Label: input.Label + "/left", Before: "first", After: "first", Cookie: "jar-left", Number: quantity, Receiver: quantity},
		{Label: input.Label + "/right", Before: "second", After: "second", Cookie: "jar-right", Number: quantity + 1, Receiver: quantity + 1},
	}
}

// checkActors compares actor observations with the independent model.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
// Takes got ([]ActorRead) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkActors(input casefile.ActionValue, got []ActorRead) error {
	expected := expectedActors(input)
	if len(got) != len(expected) {
		return fmt.Errorf("action/actor-count: got %d, want %d", len(got), len(expected))
	}
	for index, want := range expected {
		actual := got[index]
		switch {
		case actual.Label != want.Label || actual.Number != want.Number:
			return fmt.Errorf("action/actor-body: actor %d got %q %d, want %q %d", index, actual.Label, actual.Number, want.Label, want.Number)
		case actual.Before != want.Before || actual.After != want.After:
			return fmt.Errorf("action/actor-header: actor %d got %q then %q, want %q", index, actual.Before, actual.After, want.Before)
		case actual.Cookie != want.Cookie:
			return fmt.Errorf("action/actor-cookie: actor %d got %q, want %q", index, actual.Cookie, want.Cookie)
		case actual.Receiver != want.Receiver:
			return fmt.Errorf("action/actor-receiver: actor %d got %d, want %d", index, actual.Receiver, want.Receiver)
		}
	}
	return nil
}

// observeActors runs two actor-isolated actions through a provider rendezvous.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns []ActorRead which lists actor read values in their recorded order.
// Returns error which reports session setup, action dispatch or actor-response decoding
// failure, or nil on success.
func observeActors(ctx context.Context, client *http.Client, base string, input casefile.ActionValue) ([]ActorRead, error) {
	actors := map[string]*http.Client{}
	var requests [2]*http.Request
	var clients [2]*http.Client
	for index, name := range actorNames {
		request, actor, err := actorRequest(ctx, client, actors, base, input, index, name)
		if err != nil {
			return nil, err
		}
		requests[index], clients[index] = request, actor
	}
	return dispatchActors(ctx, requests, clients)
}

// actorRequest opens one actor's session and builds its rendezvous request.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes actors (map[string]*http.Client) which caches the per-actor clients.
// Takes base (string) which is the application base URL.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
// Takes index (int) which is the actor's position in request order.
// Takes name (string) which names the actor.
//
// Returns *http.Request which is the prepared rendezvous request.
// Returns *http.Client which is the actor's isolated client.
// Returns error which reports session setup or request construction failure.
func actorRequest(ctx context.Context, client *http.Client, actors map[string]*http.Client, base string, input casefile.ActionValue, index int, name string) (*http.Request, *http.Client, error) {
	var probe casefile.Probe
	probe.Actor = name
	probe.Cookies = map[string]string{"case-actor": "jar-" + name}
	actor, err := actorClient(client, actors, base, probe)
	if err != nil {
		return nil, nil, err
	}
	status, _, page, err := get(ctx, actor, base, "/main")
	if err != nil {
		return nil, nil, fmt.Errorf("action/actor-session: %w", err)
	}
	if status != http.StatusOK {
		return nil, nil, fmt.Errorf("action/actor-session: HTTP %d", status)
	}
	token, ephemeral, err := pageActionTokens(string(page))
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(map[string]any{"Label": input.Label + "/" + name, "Number": input.Quantity + index, "_csrf_ephemeral_token": ephemeral})
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/fuzz.Rendezvous", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, token)
	request.Header.Set("X-Fuzz-Actor", actorHeaders[index])
	return request, actor, nil
}

// dispatchActors sends the paired actor requests concurrently and decodes each reply.
//
// Takes requests ([2]*http.Request) which holds the prepared requests in actor order.
// Takes clients ([2]*http.Client) which holds the matching actor clients.
//
// Returns []ActorRead which lists actor read values in their recorded order.
// Returns error which reports the first actor request failure, or nil on success.
func dispatchActors(ctx context.Context, requests [2]*http.Request, clients [2]*http.Client) ([]ActorRead, error) {
	requestContext, cancel := context.WithTimeoutCause(ctx, actorRequestTimeout, errActorRequestTimeout)
	defer cancel()
	results := make([]ActorRead, len(requests))
	var failures [2]error
	var pair sync.WaitGroup
	for index, request := range requests {
		pair.Go(func() {
			response, err := clients[index].Do(request.WithContext(requestContext))
			if err != nil {
				failures[index] = err
				return
			}
			if response.StatusCode != http.StatusOK {
				failures[index] = errors.Join(fmt.Errorf("HTTP %d", response.StatusCode), closeResponse(response))
				return
			}
			failures[index] = decodeAction(response, &results[index])
		})
	}
	pair.Wait()
	for index, err := range failures {
		if err == nil {
			continue
		}
		if cause := context.Cause(requestContext); cause != nil {
			err = errors.Join(err, cause)
		}
		return results, fmt.Errorf("action/actor-request: actor %d: %w", index, err)
	}
	return results, nil
}
