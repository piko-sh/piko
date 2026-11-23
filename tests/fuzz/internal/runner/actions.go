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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// actionResponseLimit bounds one action response body in bytes.
	actionResponseLimit = 64 << 10

	// responseDrainLimit bounds the bytes discarded before a response body is closed.
	responseDrainLimit = 64 << 10

	// mediaJSON is the JSON media type.
	mediaJSON = "application/json"

	// mediaEventStream is the server-sent event media type.
	mediaEventStream = "text/event-stream"

	// headerContentType is the Content-Type header name.
	headerContentType = "Content-Type"

	// actionStreamLimit bounds the stream bytes read before the first waiting event.
	actionStreamLimit = 64 << 10

	// actionWatchTimeout bounds one watching action stream.
	actionWatchTimeout = 3 * time.Second

	// actionCancelSettle bounds the wait for cancelled stream state to settle.
	actionCancelSettle = 2 * time.Second

	// actionCancelPoll separates state polls after a stream is cancelled.
	actionCancelPoll = 10 * time.Millisecond
)

var (
	// errActionWatchTimeout is the cancellation cause when an action stream watch exceeds
	// its deadline.
	errActionWatchTimeout = errors.New("action stream watch timed out")

	// errActionCancelSettle is the cause recorded when cancelled stream state does not
	// change within its settle window.
	errActionCancelSettle = errors.New("cancelled stream state settle window elapsed")
)

// ActionResponse records an action's status, content type and body.
type ActionResponse struct {
	// ContentType is the response media type.
	ContentType string

	// Raw holds the bounded response body.
	Raw string

	// Status is the HTTP response status.
	Status int
}

// ActionObservation records action results across the selected transports.
type ActionObservation struct {
	// Batches holds batch-dispatch receipts.
	Batches []actionBatch `json:",omitempty"`

	// Notifications holds notification delivery evidence.
	Notifications *NotificationTrace `json:",omitempty"`

	// Actors holds per-actor state evidence.
	Actors []ActorRead `json:",omitempty"`

	// Upload holds upload evidence.
	Upload *UploadRead `json:",omitempty"`

	// Cache holds action-cache evidence.
	Cache []CacheRead `json:",omitempty"`

	// Refusals holds expected action refusals.
	Refusals []casefile.ActionRefusal `json:",omitempty"`

	// Cancel holds action-stream cancellation evidence.
	Cancel *StreamCancellation `json:",omitempty"`

	// JSON records the action result over the JSON transport.
	JSON ActionResponse

	// SSE records the action result over the server-sent event transport.
	SSE ActionResponse

	// Result holds the action result.
	Result casefile.ActionValue

	// Events holds the emitted events.
	Events []casefile.ActionEvent
}

// StreamCancellation records state before and after cancelling an action stream.
type StreamCancellation struct {
	// First holds the first observed stream values.
	First casefile.ActionEvent

	// Before holds the independent state read before cancellation.
	Before casefile.ActionValue

	// After holds the independent state read after cancellation.
	After casefile.ActionValue

	// NextRequest holds the result of the request following cancellation.
	NextRequest casefile.ActionValue
}

// actionSession holds the transport and CSRF tokens one action observation shares.
type actionSession struct {
	// client is the HTTP client used for every action request.
	client *http.Client

	// base is the application base URL.
	base string

	// token is the action CSRF token.
	token string

	// ephemeral is the form's ephemeral CSRF token.
	ephemeral string

	// label is the generated action label.
	label string
}

// post submits the collecting fixture action with the session's label.
//
// Takes media (string) which is the requested response media type.
// Takes operation (string) which selects the fixture operation, empty for the default.
// Takes quantity (int) which is the bound quantity argument.
//
// Returns *http.Response which is the unread action response.
// Returns error which reports encoding or transport failure, or nil on success.
func (s *actionSession) post(ctx context.Context, media, operation string, quantity int) (*http.Response, error) {
	arguments := map[string]any{"Label": s.label, "Quantity": quantity, "_csrf_ephemeral_token": s.ephemeral}
	if operation != "" {
		arguments["Operation"] = operation
	}
	payload, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("action/encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/_piko/actions/fuzz.Collect", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	setActionHeaders(request, media, mediaJSON, s.token)
	return s.client.Do(request)
}

// pageActionTokens extracts action and ephemeral CSRF tokens from a page.
//
// Takes page (string) which holds the server-rendered page markup.
//
// Returns string which is the action CSRF token.
// Returns string which is the form's ephemeral CSRF token.
// Returns error which reports missing or malformed action tokens, or nil on success.
func pageActionTokens(page string) (token, ephemeral string, err error) {
	document, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return "", "", err
	}
	for node := range document.Descendants() {
		if node.Type != html.ElementNode || node.Data != "meta" {
			continue
		}
		switch attr(node, "name") {
		case "csrf-token":
			token = attr(node, "content")
		case "csrf-ephemeral":
			ephemeral = attr(node, "content")
		}
	}
	if token == "" || ephemeral == "" {
		return "", "", errors.New("action/csrf: page lacks the action token pair")
	}
	return token, ephemeral, nil
}

// observeAction runs the selected action transport and state checks.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes page (string) which holds the rendered page markup.
// Takes want (*casefile.ActionContract) which holds the independently expected result.
//
// Returns *ActionObservation which records action transport results and independent state
// observations.
// Returns error which reports action dispatch or an independent transport or state-check
// failure, or nil on success.
func observeAction(ctx context.Context, client *http.Client, base, page string, want *casefile.ActionContract) (*ActionObservation, error) {
	o := &ActionObservation{}
	token, ephemeral, err := pageActionTokens(page)
	if err != nil {
		return o, err
	}
	session := &actionSession{client: client, base: base, token: token, ephemeral: ephemeral, label: want.Input.Label}
	for _, transport := range []struct {
		destination *ActionResponse
		media       string
	}{{media: mediaJSON, destination: &o.JSON}, {media: mediaEventStream, destination: &o.SSE}} {
		if err := observeActionTransport(ctx, session, transport.media, want.Input.Quantity, transport.destination); err != nil {
			return o, err
		}
	}
	if err := json.Unmarshal([]byte(o.JSON.Raw), &o.Result); err != nil {
		return o, fmt.Errorf("action/json: %w", err)
	}
	o.Events, err = actionEvents(o.SSE.Raw)
	if err != nil {
		return o, err
	}
	o.Refusals, err = observeRefusals(ctx, session, want.Refusals)
	if err != nil {
		return o, err
	}
	return o, observeActionFeatures(ctx, session, want, o)
}

// observeActionTransport submits the action over one transport and records its response.
//
// Takes session (*actionSession) which submits the action.
// Takes media (string) which is the requested response media type.
// Takes quantity (int) which is the bound quantity argument.
// Takes destination (*ActionResponse) which receives the status, media type and body.
//
// Returns error which reports a failed, oversized or mistyped response.
func observeActionTransport(ctx context.Context, session *actionSession, media string, quantity int, destination *ActionResponse) error {
	response, err := session.post(ctx, media, "", quantity)
	if err != nil {
		return fmt.Errorf("action/request: %w", err)
	}
	destination.Status = response.StatusCode
	contentType, _, mediaErr := mime.ParseMediaType(response.Header.Get(headerContentType))
	destination.ContentType = contentType
	body, err := io.ReadAll(io.LimitReader(response.Body, actionResponseLimit+1))
	closeErr := closeResponse(response)
	destination.Raw = string(body)
	if err = errors.Join(err, closeErr); err != nil {
		return fmt.Errorf("action/read: %w", err)
	}
	if len(body) > actionResponseLimit {
		return fmt.Errorf("action/size: response exceeds %d bytes", actionResponseLimit)
	}
	if mediaErr != nil || destination.Status != http.StatusOK || destination.ContentType != media {
		return errors.Join(fmt.Errorf("action/response: requested %s, got HTTP %d %s", media, destination.Status, destination.ContentType), mediaErr)
	}
	return nil
}

// observeRefusals submits each refused operation and records the public refusal.
//
// Takes session (*actionSession) which submits the action.
// Takes attempts ([]casefile.ActionRefusal) which lists the operations expected to fail.
//
// Returns []casefile.ActionRefusal which lists the observed refusals in attempt order.
// Returns error which reports a failed request or a status disagreement, or nil on
// success.
func observeRefusals(ctx context.Context, session *actionSession, attempts []casefile.ActionRefusal) ([]casefile.ActionRefusal, error) {
	var refusals []casefile.ActionRefusal
	for _, attempt := range attempts {
		response, err := session.post(ctx, mediaJSON, attempt.Operation, 0)
		if err != nil {
			return refusals, err
		}
		var result casefile.ActionRefusal
		if err := decodeAction(response, &result); err != nil {
			return refusals, err
		}
		if result.Status != response.StatusCode {
			return refusals, fmt.Errorf("action/refusal-status: HTTP %d payload %d", response.StatusCode, result.Status)
		}
		result.Operation = attempt.Operation
		refusals = append(refusals, result)
	}
	return refusals, nil
}

// observeActionFeatures runs the optional action feature checks the contract selects.
//
// Takes session (*actionSession) which submits the action and carries its tokens.
// Takes want (*casefile.ActionContract) which selects the optional features.
// Takes o (*ActionObservation) which receives the feature evidence.
//
// Returns error which reports the first failed feature observation, or nil on success.
func observeActionFeatures(ctx context.Context, session *actionSession, want *casefile.ActionContract, o *ActionObservation) error {
	var err error
	if want.CancelStream {
		o.Cancel, err = observeDisconnect(ctx, session.post, want.Input)
	}
	if err == nil && want.Cache {
		o.Cache, err = observeActionCache(ctx, session.client, session.base, session.token, session.ephemeral, want.Input.Label)
	}
	if err == nil && want.Upload {
		o.Upload, err = observeUpload(ctx, session.client, session.base, session.token, session.ephemeral, want.Input, want.UploadPublic)
	}
	if err == nil && want.Actors {
		o.Actors, err = observeActors(ctx, session.client, session.base, want.Input)
	}
	if err == nil && want.Notifications {
		o.Notifications, err = observeNotifications(ctx, session.client, session.base, want.Input)
	}
	if err != nil || !want.Batch {
		return err
	}
	for _, parallel := range []bool{false, true} {
		batch, readErr := observeBatch(ctx, session.client, session.base, session.token, session.ephemeral, want.Input, parallel)
		o.Batches = append(o.Batches, batch)
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// setActionHeaders marks a request as a same-origin action call carrying the CSRF token.
//
// Takes request (*http.Request) which receives the host and headers.
// Takes accept (string) which is the requested response media type.
// Takes contentType (string) which is the request body media type.
// Takes token (string) which is the action CSRF token.
func setActionHeaders(request *http.Request, accept, contentType, token string) {
	request.Host = requestHost
	request.Header.Set("Accept", accept)
	request.Header.Set(headerContentType, contentType)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-CSRF-Action-Token", token)
}

// closeResponse discards a bounded remainder of a response body and closes it.
//
// Takes response (*http.Response) which is the response whose body is released.
//
// Returns error which reports a drain or close failure, or nil on success.
func closeResponse(response *http.Response) error {
	_, drainErr := io.CopyN(io.Discard, response.Body, responseDrainLimit)
	if errors.Is(drainErr, io.EOF) {
		drainErr = nil
	}
	return errors.Join(drainErr, response.Body.Close())
}

// decodeAction decodes a bounded action result into the supplied destination.
//
// The response body is drained and closed on every path.
//
// Takes response (*http.Response) which is the action's HTTP response.
// Takes value (any) which receives the decoded public action result.
//
// Returns error which reports an oversized or malformed public action response, or nil on
// success.
func decodeAction(response *http.Response, value any) (err error) {
	defer func() {
		err = errors.Join(err, closeResponse(response))
	}()
	media, _, err := mime.ParseMediaType(response.Header.Get(headerContentType))
	if err != nil || media != mediaJSON {
		return errors.Join(fmt.Errorf("action/json-media: %q", media), err)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, actionResponseLimit+1))
	if err != nil {
		return err
	}
	if len(body) > actionResponseLimit {
		return fmt.Errorf("action/json-size: response exceeds %d bytes", actionResponseLimit)
	}
	return json.Unmarshal(body, value)
}

// observeDisconnect cancels an action stream and reads independent subsequent state.
//
// Takes post (func(context.Context, string, string, int) (*http.Response, error)) which
// submits the selected fixture action with its label and quantity.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns *StreamCancellation which records state around stream cancellation and the
// following request.
// Returns error which reports cancellation-protocol or subsequent-state observation
// failure, or nil on success.
func observeDisconnect(ctx context.Context, post func(context.Context, string, string, int) (*http.Response, error), input casefile.ActionValue) (*StreamCancellation, error) {
	o := &StreamCancellation{}
	query := func(operation string, out *casefile.ActionValue) error {
		response, err := post(ctx, mediaJSON, operation, input.Quantity)
		if err != nil {
			return err
		}
		if response.StatusCode != http.StatusOK {
			return errors.Join(fmt.Errorf("action/poll-status: HTTP %d", response.StatusCode), closeResponse(response))
		}
		return decodeAction(response, out)
	}
	if err := query("status", &o.Before); err != nil {
		return o, err
	}
	first, err := readFirstStreamFrame(ctx, post, input)
	if err != nil {
		return o, err
	}
	events, err := actionEvents(first)
	if err != nil {
		return o, fmt.Errorf("action/watch-first: %w", err)
	}
	if len(events) != 1 {
		return o, fmt.Errorf("action/watch-first: got %v", events)
	}
	o.First = events[0]
	if err := awaitCancelledState(ctx, query, o); err != nil {
		return o, err
	}
	return o, query("", &o.NextRequest)
}

// awaitCancelledState polls the fixture state until the cancelled stream's change lands
// or the settle window closes.
//
// Takes query (func(string, *casefile.ActionValue) error) which reads the fixture state.
// Takes o (*StreamCancellation) which holds the state before cancellation and receives
// the state after it.
//
// Returns error which reports a failed state read or context cancellation, or nil once
// the state changes or the window closes.
func awaitCancelledState(ctx context.Context, query func(string, *casefile.ActionValue) error, o *StreamCancellation) error {
	settle, cancel := context.WithTimeoutCause(ctx, actionCancelSettle, errActionCancelSettle)
	defer cancel()
	ticker := time.NewTicker(actionCancelPoll)
	defer ticker.Stop()
	for {
		if err := query("status", &o.After); err != nil {
			return err
		}
		if o.After.Quantity != o.Before.Quantity {
			return nil
		}
		select {
		case <-settle.Done():
			if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			return nil
		case <-ticker.C:
		}
	}
}

// readFirstStreamFrame opens a watching action stream, reads its first waiting event and
// cancels the stream.
//
// Takes post (func(context.Context, string, string, int) (*http.Response, error)) which
// submits the selected fixture action with its label and quantity.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns string which holds the raw stream text up to the first waiting event.
// Returns error which reports a stream-protocol, read or oversize failure, or nil on
// success.
func readFirstStreamFrame(ctx context.Context, post func(context.Context, string, string, int) (*http.Response, error), input casefile.ActionValue) (string, error) {
	const limit = actionStreamLimit
	watch, cancel := context.WithTimeoutCause(ctx, actionWatchTimeout, errActionWatchTimeout)
	defer cancel()
	response, err := post(watch, mediaEventStream, "watch", input.Quantity)
	if err != nil {
		return "", errors.Join(err, context.Cause(watch))
	}
	media, _, mediaErr := mime.ParseMediaType(response.Header.Get(headerContentType))
	if mediaErr != nil || response.StatusCode != http.StatusOK || media != mediaEventStream {
		return "", errors.Join(fmt.Errorf("action/watch-status: %d %s", response.StatusCode, media), mediaErr, closeResponse(response))
	}
	reader := bufio.NewScanner(io.LimitReader(response.Body, limit+1))
	var first strings.Builder
	waiting := false
	for reader.Scan() {
		line := reader.Text()
		first.WriteString(line + "\n")
		if line == "" && strings.Contains(first.String(), "event: waiting") {
			waiting = true
			break
		}
	}
	scanErr := reader.Err()
	cancel()
	closeErr := response.Body.Close()
	if !waiting {
		if scanErr == nil && first.Len() > limit {
			scanErr = fmt.Errorf("action/watch-size: stream exceeds %d bytes before the first event", limit)
		}
		if scanErr != nil {
			return first.String(), errors.Join(fmt.Errorf("action/watch-read: %w", scanErr), context.Cause(watch))
		}
	}
	if closeErr != nil && !waiting {
		return first.String(), fmt.Errorf("action/watch-close: %w", closeErr)
	}
	return first.String(), nil
}

// actionEvents decodes action events from a bounded server-sent event stream.
//
// Takes raw (string) which holds the server-sent event response body.
//
// Returns []casefile.ActionEvent which lists action event values in their recorded order.
// Returns error which reports a malformed action event stream, or nil on success.
func actionEvents(raw string) ([]casefile.ActionEvent, error) {
	var events []casefile.ActionEvent
	var name string
	var data []string
	for line := range strings.Lines(raw) {
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			if name == "" && len(data) == 0 {
				continue
			}
			event, err := decodeActionEvent(name, data)
			if err != nil {
				return events, err
			}
			events = append(events, event)
			name, data = "", nil
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			name = value
		case "data":
			data = append(data, value)
		}
	}
	if name != "" || len(data) > 0 {
		return events, errors.New("action/sse-framing: truncated event")
	}
	return events, nil
}

// decodeActionEvent decodes one dispatched server-sent action event.
//
// Takes name (string) which is the event type.
// Takes data ([]string) which holds the event's data lines.
//
// Returns casefile.ActionEvent which is the decoded event.
// Returns error which reports an incomplete frame or malformed data, or nil on success.
func decodeActionEvent(name string, data []string) (casefile.ActionEvent, error) {
	var value casefile.ActionValue
	if name == "" || len(data) == 0 {
		return casefile.ActionEvent{Name: name, Value: value}, errors.New("action/sse-framing: incomplete event")
	}
	if err := json.Unmarshal([]byte(strings.Join(data, "\n")), &value); err != nil {
		return casefile.ActionEvent{Name: name, Value: value}, fmt.Errorf("action/sse-data: %w", err)
	}
	return casefile.ActionEvent{Name: name, Value: value}, nil
}

// publicActionResult validates and normalises the public action result.
//
// Takes got (*ActionObservation) which holds the observed result.
// Takes mode (string) which selects the server mode.
//
// Returns *ActionObservation which holds the validated and normalised public action
// fields.
// Returns error which reports a public action response inconsistent with the selected
// mode, or nil on success.
func publicActionResult(got *ActionObservation, mode string) (*ActionObservation, error) {
	if got == nil || mode != modeInterpreted || len(got.Refusals) == 0 {
		return got, nil
	}
	out := *got
	out.Refusals = slices.Clone(got.Refusals)
	for index, refusal := range out.Refusals {
		detail, ok := strings.CutPrefix(refusal.Message, "provider daemon_adapters.actionSingle: ")
		if !ok {
			return nil, fmt.Errorf("action/refusal-detail: %s lacks the interpreted provider frame", refusal.Operation)
		}
		out.Refusals[index].Message = detail
	}
	return &out, nil
}

// checkAction compares action observations with the saved contract.
//
// Takes want (*casefile.ActionContract) which holds the independently expected result.
// Takes got (*ActionObservation) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAction(want *casefile.ActionContract, got *ActionObservation) error {
	if want == nil && got == nil {
		return nil
	}
	if want == nil || got == nil {
		return errors.New("action/missing: action evidence differs")
	}
	if err := checkActionFeatures(want, got); err != nil {
		return err
	}
	if !reflect.DeepEqual(want.Refusals, got.Refusals) {
		return fmt.Errorf("action/refusals: got %+v, expected %+v", got.Refusals, want.Refusals)
	}
	if err := checkCancellation(want, got.Cancel); err != nil {
		return err
	}
	if got.JSON.Status != http.StatusOK || got.JSON.ContentType != mediaJSON || got.Result != want.Result {
		return fmt.Errorf("action/result: got HTTP %d %s %#v, want %#v", got.JSON.Status, got.JSON.ContentType, got.Result, want.Result)
	}
	if got.SSE.Status != http.StatusOK || got.SSE.ContentType != mediaEventStream || !slices.Equal(got.Events, want.Events) {
		return fmt.Errorf("action/events: got HTTP %d %s %#v, want %#v", got.SSE.Status, got.SSE.ContentType, got.Events, want.Events)
	}
	return nil
}

// checkActionFeatures compares the optional action feature evidence with the contract.
//
// Takes want (*casefile.ActionContract) which holds the independently expected result.
// Takes got (*ActionObservation) which holds the observed result.
//
// Returns error which reports the first feature mismatch, or nil when every selected
// feature passes.
func checkActionFeatures(want *casefile.ActionContract, got *ActionObservation) error {
	if want.Notifications {
		if err := checkNotifications(want.Input, got.Notifications); err != nil {
			return err
		}
	}
	if want.Batch {
		if err := checkBatches(want, got.Batches); err != nil {
			return err
		}
	}
	if want.Cache && !slices.Equal(got.Cache, expectedCacheReads(want.Input.Label)) {
		return fmt.Errorf("action/cache: got %+v, want %+v", got.Cache, expectedCacheReads(want.Input.Label))
	}
	if want.Upload {
		if err := checkUploadedObject(want, got.Upload); err != nil {
			return err
		}
	}
	if want.Actors {
		return checkActors(want.Input, got.Actors)
	}
	return nil
}

// checkCancellation compares stream cancellation evidence with the modelled state.
//
// Takes want (*casefile.ActionContract) which holds the independently expected result.
// Takes got (*StreamCancellation) which holds the observed cancellation evidence.
//
// Returns error which reports a cancellation mismatch, or nil when the contract does not
// cancel a stream or the evidence matches.
func checkCancellation(want *casefile.ActionContract, got *StreamCancellation) error {
	if !want.CancelStream {
		return nil
	}
	expected := StreamCancellation{
		First:       casefile.ActionEvent{Name: "waiting", Value: want.Input},
		Before:      casefile.ActionValue{Label: want.Input.Label, Quantity: 0},
		After:       casefile.ActionValue{Label: want.Input.Label, Quantity: 1},
		NextRequest: want.Result,
	}
	if got == nil || *got != expected {
		return fmt.Errorf("action/cancel: got %+v, want %+v", got, expected)
	}
	return nil
}

// compareActions compares action results across server modes.
//
// Takes a (*ActionObservation) which holds the first value to compare.
// Takes b (*ActionObservation) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareActions(a, b *ActionObservation) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil {
		return errors.New("action/parity: missing runtime observation")
	}
	if !reflect.DeepEqual(a.Notifications, b.Notifications) {
		return errors.New("action/parity: notification traces differ")
	}
	if !reflect.DeepEqual(a.Batches, b.Batches) {
		return errors.New("action/parity: ordered batch results differ")
	}
	if !slices.Equal(a.Cache, b.Cache) {
		return errors.New("action/parity: cache execution or response differs")
	}
	if !reflect.DeepEqual(a.Upload, b.Upload) {
		return errors.New("action/parity: upload metadata or disk bytes differ")
	}
	if !slices.Equal(a.Actors, b.Actors) {
		return errors.New("action/parity: overlapping request data differs")
	}
	if !reflect.DeepEqual(a.Refusals, b.Refusals) || !reflect.DeepEqual(a.Cancel, b.Cancel) {
		return errors.New("action/parity: refusal or cancellation differs")
	}
	if a.JSON.Status != b.JSON.Status || a.JSON.ContentType != b.JSON.ContentType ||
		a.Result != b.Result || a.SSE.Status != b.SSE.Status || a.SSE.ContentType != b.SSE.ContentType || !slices.Equal(a.Events, b.Events) {
		return errors.New("action/parity: typed results or stream differ")
	}
	return nil
}
