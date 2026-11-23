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
	"net/http/cookiejar"
	"slices"
	"strconv"
	"strings"
	"time"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// noticeFrameMax bounds the bytes of one server-sent notification frame, including
	// ignored lines read before it is dispatched.
	noticeFrameMax = 32 << 10

	// noticeTimeout bounds one whole notification observation.
	noticeTimeout = 9 * time.Second

	// noticePollInterval separates subscriber-count polls during teardown.
	noticePollInterval = 15 * time.Millisecond

	// noticeFinalSequence is the last notification sequence the observation publishes.
	noticeFinalSequence = 3

	// noticeScannerInitialBuffer is the initial line-buffer capacity of a notice stream.
	noticeScannerInitialBuffer = 4096
)

var (
	// errNoticeFrameOversize reports a server-sent frame above noticeFrameMax bytes.
	errNoticeFrameOversize = errors.New("action/notice-frame: frame exceeds byte limit")

	// errNoticeStreamEnded reports a notification stream that closed before a frame was
	// complete.
	errNoticeStreamEnded = errors.New("action/notice-stream: stream ended before frame completed")

	// errNoticeTimeout is the cause recorded when a notification observation overruns.
	errNoticeTimeout = errors.New("action/notifications: observation timed out")
)

// NoticeFrame records one server-sent notification.
type NoticeFrame struct {
	// Event is the server-sent event type.
	Event string

	// ID is the server-sent event identifier.
	ID string

	// Text is the notification text.
	Text string

	// Sequence is the notification sequence number.
	Sequence int
}

// NoticeStatus records subscriber and notification sequence counts.
type NoticeStatus struct {
	// Active counts connected subscribers.
	Active int

	// Latest is the latest notification sequence number.
	Latest int
}

// NotificationTrace records notification delivery and reconnection.
type NotificationTrace struct {
	// First holds the frames delivered to the first subscriber.
	First []NoticeFrame

	// Second holds the frames delivered to the second subscriber before it disconnects.
	Second []NoticeFrame

	// Reconnect holds the frames delivered to the second subscriber after reconnecting.
	Reconnect []NoticeFrame

	// Statuses holds subscriber and sequence-count snapshots.
	Statuses []NoticeStatus
}

// noticeComparison pairs one observed frame sequence with its expected sequence.
type noticeComparison struct {
	// name identifies the compared subscriber phase.
	name string

	// got holds the observed frames.
	got []NoticeFrame

	// want holds the expected frames.
	want []NoticeFrame
}

// noticePayload is the JSON data of one notice event.
type noticePayload struct {
	// Text is the notification text.
	Text string

	// ID is the notification sequence number.
	ID int
}

// noticeClient holds a notification client's transport and CSRF state.
type noticeClient struct {
	// client is the HTTP transport used for requests.
	client *http.Client

	// base is the application base URL.
	base string

	// token is the issued CSRF action token.
	token string

	// ephemeral is the issued ephemeral CSRF token.
	ephemeral string
}

// noticeStream reads bounded server-sent notification frames.
type noticeStream struct {
	// body is the open response body.
	body io.Closer

	// scanner reads bounded event-stream lines.
	scanner *bufio.Scanner
}

// noticeObservation drives the subscribers of one notification observation.
type noticeObservation struct {
	// left is the publishing and first subscribing client.
	left *noticeClient

	// trace receives the observed evidence.
	trace *NotificationTrace

	// label prefixes published notification text.
	label string
}

// post submits a notification action with the client's CSRF tokens.
//
// Takes name (string) which names the public action to invoke.
// Takes media (string) which is the requested response media type.
// Takes last (string) which is the last received server-sent event ID.
// Takes fields (map[string]any) which supplies action fields in a fresh map. The function
// adds the CSRF ephemeral token to the map.
//
// Returns *http.Response which is the open action response whose body belongs to the
// caller.
// Returns error which reports request construction, HTTP transport or response-decoding
// failure, or nil on success.
func (c *noticeClient) post(ctx context.Context, name, media, last string, fields map[string]any) (*http.Response, error) {
	fields["_csrf_ephemeral_token"] = c.ephemeral
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/_piko/actions/fuzz."+name, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	setActionHeaders(request, media, mediaJSON, c.token)
	if last != "" {
		request.Header.Set("Last-Event-ID", last)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	content, _, err := mime.ParseMediaType(response.Header.Get(headerContentType))
	if response.StatusCode != http.StatusOK || err != nil || content != media {
		mismatch := fmt.Errorf("action/notice-response: %s HTTP %d %s", name, response.StatusCode, content)
		return nil, errors.Join(mismatch, err, closeResponse(response))
	}
	return response, nil
}

// subscribe opens a notification stream from the selected event ID.
//
// Takes last (string) which is the last received server-sent event ID.
//
// Returns *noticeStream which owns the open notification response body and frame reader.
// Returns error which reports request construction, HTTP transport or response-decoding
// failure, or nil on success.
func (c *noticeClient) subscribe(ctx context.Context, last string) (*noticeStream, error) {
	response, err := c.post(ctx, "Subscribe", mediaEventStream, last, map[string]any{})
	if err != nil {
		return nil, err
	}
	return newNoticeStream(response.Body), nil
}

// close closes the stream's response body without draining the unbounded event stream.
//
// Closing an already closed body is harmless, so deferred and explicit closes may
// overlap.
//
// Returns error which reports a close failure, or nil on success.
func (s *noticeStream) close() error {
	return s.body.Close()
}

// read reads one bounded notification frame into the receipt.
//
// Several data lines of one event are joined with a line feed, as the server-sent events
// specification requires. Each frame is bounded separately, so a long-lived stream is not
// cut short by the bytes of earlier frames.
//
// Takes destination (*[]NoticeFrame) which receives the decoded result.
//
// Returns error which reports an unreadable or malformed frame, wraps
// errNoticeFrameOversize for an oversized frame, or wraps errNoticeStreamEnded when the
// stream closes before the frame completes.
func (s *noticeStream) read(destination *[]NoticeFrame) error {
	frame := NoticeFrame{}
	var data []string
	consumed := 0
	for s.scanner.Scan() {
		line := s.scanner.Text()
		consumed += len(line) + 1
		if consumed > noticeFrameMax {
			return fmt.Errorf("%w: more than %d bytes", errNoticeFrameOversize, noticeFrameMax)
		}
		if line != "" {
			applyNoticeField(&frame, &data, line)
			continue
		}
		if frame.Event == "" {
			continue
		}
		if err := decodeNoticeData(&frame, strings.Join(data, "\n")); err != nil {
			return err
		}
		*destination = append(*destination, frame)
		return nil
	}
	if err := s.scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("%w: line exceeds %d bytes", errNoticeFrameOversize, noticeFrameMax)
		}
		return err
	}
	return errNoticeStreamEnded
}

// status publishes or inspects notifications and returns the subscriber snapshot.
//
// Takes sequence (int) which is the notification to publish, or 0 to only inspect.
//
// Returns NoticeStatus which is the server's subscriber and sequence snapshot.
// Returns error which reports action or decoding failure, or nil on success.
func (n *noticeObservation) status(ctx context.Context, sequence int) (NoticeStatus, error) {
	var result NoticeStatus
	fields := map[string]any{"Inspect": sequence == 0, "Text": fmt.Sprintf("%s-%d", n.label, sequence)}
	response, err := n.left.post(ctx, "SendNotice", mediaJSON, "", fields)
	if err != nil {
		return result, err
	}
	err = decodeAction(response, &result)
	return result, err
}

// publish publishes one notification and records the resulting snapshot.
//
// Takes sequence (int) which is the notification sequence to publish.
//
// Returns error which reports action or decoding failure, or nil on success.
func (n *noticeObservation) publish(ctx context.Context, sequence int) error {
	result, err := n.status(ctx, sequence)
	if err != nil {
		return err
	}
	n.trace.Statuses = append(n.trace.Statuses, result)
	return nil
}

// waitActive polls until the server reports the wanted subscriber count.
//
// Takes count (int) which is the wanted active subscriber count.
//
// Returns error which reports action failure, or the cancellation cause when the count is
// not reached in time.
func (n *noticeObservation) waitActive(ctx context.Context, count int) error {
	ticker := time.NewTicker(noticePollInterval)
	defer ticker.Stop()
	for {
		result, err := n.status(ctx, 0)
		if err != nil {
			return err
		}
		if result.Active == count {
			n.trace.Statuses = append(n.trace.Statuses, result)
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("action/notice-teardown: active %d, wanted %d: %w", result.Active, count, context.Cause(ctx))
		case <-ticker.C:
		}
	}
}

// expectedNotices models notification delivery and reconnection.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns *NotificationTrace which holds the independently modelled notification delivery
// sequence.
func expectedNotices(input casefile.ActionValue) *NotificationTrace {
	ready := NoticeFrame{Event: "ready", ID: "", Text: "", Sequence: 0}
	notice := func(sequence int) NoticeFrame {
		return NoticeFrame{Event: "notice", ID: strconv.Itoa(sequence), Sequence: sequence, Text: fmt.Sprintf("%s-%d", input.Label, sequence)}
	}
	return &NotificationTrace{
		First:     []NoticeFrame{ready, notice(1), notice(2), notice(noticeFinalSequence)},
		Second:    []NoticeFrame{ready, notice(1)},
		Reconnect: []NoticeFrame{notice(2), ready, notice(noticeFinalSequence)},
		Statuses: []NoticeStatus{
			{Active: 2, Latest: 1},
			{Active: 1, Latest: 1},
			{Active: 1, Latest: 2},
			{Active: 2, Latest: noticeFinalSequence},
			{Active: 0, Latest: noticeFinalSequence},
		},
	}
}

// checkNotifications compares notification receipts with the independent model.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
// Takes got (*NotificationTrace) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkNotifications(input casefile.ActionValue, got *NotificationTrace) error {
	want := expectedNotices(input)
	if got == nil {
		return errors.New("action/notifications: missing trace")
	}
	comparisons := []noticeComparison{
		{name: "first", got: got.First, want: want.First},
		{name: "second", got: got.Second, want: want.Second},
		{name: "reconnect", got: got.Reconnect, want: want.Reconnect},
	}
	for _, comparison := range comparisons {
		if !slices.Equal(comparison.got, comparison.want) {
			return fmt.Errorf("action/notice-%s: got %+v want %+v", comparison.name, comparison.got, comparison.want)
		}
	}
	if !slices.Equal(got.Statuses, want.Statuses) {
		return fmt.Errorf("action/notice-teardown: got %+v want %+v", got.Statuses, want.Statuses)
	}
	return nil
}

// newNoticeClient creates an independent subscriber with issued CSRF tokens.
//
// Takes original (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
//
// Returns *noticeClient which holds an independent subscriber transport and issued CSRF
// tokens.
// Returns error which reports session setup or missing action tokens, or nil on success.
func newNoticeClient(ctx context.Context, original *http.Client, base string) (*noticeClient, error) {
	client := *original
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.Jar = jar
	status, _, page, err := get(ctx, &client, base, "/main")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("action/notice-session: HTTP %d", status)
	}
	token, ephemeral, err := pageActionTokens(string(page))
	return &noticeClient{client: &client, base: base, token: token, ephemeral: ephemeral}, err
}

// newNoticeStream wraps an event-stream body in a line reader bounded per frame.
//
// Takes body (io.ReadCloser) which is the open event-stream response body.
//
// Returns *noticeStream which owns body.
func newNoticeStream(body io.ReadCloser) *noticeStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, noticeScannerInitialBuffer), noticeFrameMax)
	return &noticeStream{body: body, scanner: scanner}
}

// applyNoticeField records one server-sent event field line in the pending frame.
//
// Takes frame (*NoticeFrame) which receives event and id fields.
// Takes data (*[]string) which accumulates data lines.
// Takes line (string) which is the non-empty field line.
func applyNoticeField(frame *NoticeFrame, data *[]string, line string) {
	field, value, ok := strings.Cut(line, ":")
	if !ok || field == "" {
		return
	}
	value = strings.TrimPrefix(value, " ")
	switch field {
	case "event":
		frame.Event = value
	case "id":
		frame.ID = value
	case "data":
		*data = append(*data, value)
	}
}

// decodeNoticeData validates and decodes the joined data of one dispatched frame.
//
// Takes frame (*NoticeFrame) which receives the decoded sequence and text.
// Takes data (string) which is the frame's joined data.
//
// Returns error which reports a malformed ready or notice payload, or nil on success.
func decodeNoticeData(frame *NoticeFrame, data string) error {
	switch frame.Event {
	case "ready":
		if data != "true" {
			return fmt.Errorf("action/notice-ready: %q", data)
		}
	case "notice":
		var event noticePayload
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("action/notice-payload: %w", err)
		}
		frame.Sequence, frame.Text = event.ID, event.Text
	}
	return nil
}

// observeNotifications records subscriber delivery and reconnection.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns *NotificationTrace which records delivery and reconnect evidence collected
// before any failure.
// Returns error which reports subscription, notification-read or reconnection failure, or
// nil on success.
func observeNotifications(ctx context.Context, client *http.Client, base string, input casefile.ActionValue) (trace *NotificationTrace, err error) {
	ctx, cancel := context.WithTimeoutCause(ctx, noticeTimeout, errNoticeTimeout)
	defer cancel()
	trace = &NotificationTrace{}
	left, err := newNoticeClient(ctx, client, base)
	if err != nil {
		return trace, err
	}
	right, err := newNoticeClient(ctx, client, base)
	if err != nil {
		return trace, err
	}
	observation := &noticeObservation{left: left, trace: trace, label: input.Label}
	first, err := left.subscribe(ctx, "")
	if err != nil {
		return trace, err
	}
	defer func() { err = errors.Join(err, first.close()) }()
	if err := first.read(&trace.First); err != nil {
		return trace, err
	}
	if err := observeSecondSubscriber(ctx, observation, right, first); err != nil {
		return trace, err
	}
	return trace, observeReconnect(ctx, observation, right, first)
}

// observeSecondSubscriber delivers the first notice to two subscribers and tears the
// second down.
//
// Takes observation (*noticeObservation) which publishes notices and records the trace.
// Takes right (*noticeClient) which is the second subscriber's client.
// Takes first (*noticeStream) which is the first subscriber's open stream.
//
// Returns error which reports subscription, notification-read or teardown failure, or nil
// on success.
func observeSecondSubscriber(ctx context.Context, observation *noticeObservation, right *noticeClient, first *noticeStream) error {
	trace := observation.trace
	second, err := right.subscribe(ctx, "")
	if err != nil {
		return err
	}
	if err := runSteps(
		func() error { return second.read(&trace.Second) },
		func() error { return observation.publish(ctx, 1) },
		func() error { return first.read(&trace.First) },
		func() error { return second.read(&trace.Second) },
	); err != nil {
		return errors.Join(err, second.close())
	}
	return runSteps(
		second.close,
		func() error { return observation.waitActive(ctx, 1) },
		func() error { return observation.publish(ctx, 2) },
		func() error { return first.read(&trace.First) },
	)
}

// observeReconnect resumes the second subscriber from its last event and drains both
// subscribers.
//
// Takes observation (*noticeObservation) which publishes notices and records the trace.
// Takes right (*noticeClient) which is the second subscriber's client.
// Takes first (*noticeStream) which is the first subscriber's open stream.
//
// Returns error which reports reconnection, notification-read or teardown failure, or nil
// on success.
func observeReconnect(ctx context.Context, observation *noticeObservation, right *noticeClient, first *noticeStream) (err error) {
	trace := observation.trace
	if len(trace.Second) == 0 {
		return errors.New("action/notice-reconnect: second subscriber recorded no event")
	}
	reconnected, err := right.subscribe(ctx, trace.Second[len(trace.Second)-1].ID)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reconnected.close()) }()
	if err := runSteps(
		func() error { return reconnected.read(&trace.Reconnect) },
		func() error { return reconnected.read(&trace.Reconnect) },
		func() error { return observation.publish(ctx, noticeFinalSequence) },
		func() error { return first.read(&trace.First) },
		func() error { return reconnected.read(&trace.Reconnect) },
		first.close,
		reconnected.close,
	); err != nil {
		return err
	}
	return observation.waitActive(ctx, 0)
}
