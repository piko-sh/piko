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

package daemon_domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// errClientDisconnected is returned when the SSE client has disconnected.
var errClientDisconnected = errors.New("client disconnected")

// SSECapable is an interface that actions can implement to support
// Server-Sent Events (SSE) streaming. When an action implements this
// interface, clients can request SSE transport for progressive updates.
type SSECapable interface {
	// StreamProgress handles SSE streaming for the action.
	// The stream is automatically closed when the call returns.
	StreamProgress(stream *SSEStream) error
}

// SSEStream provides an interface for sending Server-Sent Events to the client.
// It wraps the underlying HTTP response writer with SSE-specific methods.
type SSEStream struct {
	// writer is the output destination for SSE events and heartbeats.
	writer io.Writer

	// flusher sends buffered data to the client immediately.
	flusher http.Flusher

	// done signals when the client has disconnected.
	done <-chan struct{}

	// lastEventID holds the Last-Event-ID header from the client's
	// reconnection request. Empty on first connection.
	lastEventID string

	// nextID is the auto-incrementing event ID counter. Only used
	// when idsEnabled is true.
	nextID uint64

	// idsEnabled controls whether Send/SendData/SendComplete include
	// an id: field in the SSE output. Activated by EnableEventIDs().
	idsEnabled bool
}

// NewSSEStream creates a new SSE stream from an HTTP response writer.
//
// Takes w (http.ResponseWriter) which is the response writer to wrap.
// Takes done (<-chan struct{}) which signals when the stream should close.
// Takes lastEventID (string) which is the Last-Event-ID header from the
// client's reconnection request (empty on first connection).
//
// Returns *SSEStream which is the configured stream, or nil if the writer
// does not support flushing.
func NewSSEStream(w http.ResponseWriter, done <-chan struct{}, lastEventID string) *SSEStream {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return &SSEStream{
		writer:      w,
		flusher:     flusher,
		done:        done,
		lastEventID: lastEventID,
	}
}

// EnableEventIDs activates automatic event ID generation for the SSE stream.
// When enabled, Send, SendData, and SendComplete calls include an
// auto-incrementing id: field in the SSE output.
//
// Clients can send the Last-Event-ID header on reconnection, allowing the
// action to skip already-sent events via LastEventID.
func (s *SSEStream) EnableEventIDs() {
	s.idsEnabled = true
	s.nextID = 1
}

// LastEventID returns the Last-Event-ID header value from the client's
// reconnection request. Returns an empty string on first connection.
//
// This value is client-provided input and must be validated before use.
// Actions can use this to skip already-sent events when a client reconnects:
// if lastID := stream.LastEventID(); lastID != "" {
//
//	if parsed, err := strconv.Atoi(lastID); err == nil && parsed > 0 && parsed <= total {
//	    startIndex = parsed
//	}
//
// }
//
// Returns string which is the Last-Event-ID header value, or empty
// on first connection.
func (s *SSEStream) LastEventID() string {
	return s.lastEventID
}

// Send transmits a JSON-encoded SSE event to the client stream. When event
// IDs are enabled via EnableEventIDs(), an auto-incrementing id field is
// included.
//
// Takes event (string) which specifies the SSE event type.
// Takes data (any) which is the payload to JSON-encode and send.
//
// Returns error when the client has disconnected or encoding fails.
func (s *SSEStream) Send(event string, data any) error {
	select {
	case <-s.done:
		return errClientDisconnected
	default:
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding SSE data: %w", err)
	}

	if s.idsEnabled {
		_, err = fmt.Fprintf(s.writer, "id: %d\nevent: %s\ndata: %s\n\n", s.nextID, event, jsonData)
		s.nextID++
	} else {
		_, err = fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", event, jsonData)
	}
	if err != nil {
		return fmt.Errorf("writing SSE event: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// SendWithID sends an SSE event with a caller-specified event ID.
// Use this instead of EnableEventIDs when the event ID must match an
// application-specific value such as a database record ID rather than an
// auto-incrementing counter.
//
// Takes id (string) which is the SSE event ID to include.
// Takes event (string) which specifies the SSE event type.
// Takes data (any) which is the payload to JSON-encode and send.
//
// Returns error when the client has disconnected or encoding fails.
func (s *SSEStream) SendWithID(id string, event string, data any) error {
	select {
	case <-s.done:
		return errClientDisconnected
	default:
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding SSE data: %w", err)
	}

	_, err = fmt.Fprintf(s.writer, "id: %s\nevent: %s\ndata: %s\n\n", id, event, jsonData)
	if err != nil {
		return fmt.Errorf("writing SSE event: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// SendData sends an SSE event with only the data field, using the default
// 'message' event type on the client. When event IDs are enabled, an id: field
// is included.
//
// Takes data (any) which is the payload to send, marshalled as JSON.
//
// Returns error when the client has disconnected, encoding fails, or writing
// fails.
func (s *SSEStream) SendData(data any) error {
	select {
	case <-s.done:
		return errClientDisconnected
	default:
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding SSE data: %w", err)
	}

	if s.idsEnabled {
		_, err = fmt.Fprintf(s.writer, "id: %d\ndata: %s\n\n", s.nextID, jsonData)
		s.nextID++
	} else {
		_, err = fmt.Fprintf(s.writer, "data: %s\n\n", jsonData)
	}
	if err != nil {
		return fmt.Errorf("writing SSE data: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// SendComplete sends a "complete" event signalling the stream is done.
// This should be called at the end of successful streaming.
//
// Takes data (any) which is the final payload to send with the complete event.
//
// Returns error when the event cannot be sent.
func (s *SSEStream) SendComplete(data any) error {
	return s.Send("complete", data)
}

// SendError sends an "error" event with error details.
//
// Takes err (error) which provides the error to send to the client.
//
// Returns error when the event cannot be sent.
func (s *SSEStream) SendError(err error) error {
	return s.Send("error", map[string]string{
		"message": err.Error(),
	})
}

// SendHeartbeat sends a comment (ping) to keep the connection alive.
// Heartbeats do not include event IDs per the SSE specification
// (comments are not events).
//
// Returns error when the client has disconnected or the write fails.
func (s *SSEStream) SendHeartbeat() error {
	select {
	case <-s.done:
		return errClientDisconnected
	default:
	}

	_, err := fmt.Fprint(s.writer, ": heartbeat\n\n")
	if err != nil {
		return fmt.Errorf("writing heartbeat: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// Done returns a channel that is closed when the client disconnects.
// Use this to detect early termination and clean up resources.
//
// Returns <-chan struct{} which yields a signal when the stream ends.
func (s *SSEStream) Done() <-chan struct{} {
	return s.done
}
