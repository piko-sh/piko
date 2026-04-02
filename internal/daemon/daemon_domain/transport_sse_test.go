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
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type mockFlushWriter struct {
	writer     bytes.Buffer
	flushCount int
}

func (m *mockFlushWriter) Header() http.Header         { return http.Header{} }
func (m *mockFlushWriter) Write(b []byte) (int, error) { return m.writer.Write(b) }
func (m *mockFlushWriter) WriteHeader(_ int)           {}
func (m *mockFlushWriter) Flush()                      { m.flushCount++ }

type mockNonFlushWriter struct{}

func (*mockNonFlushWriter) Header() http.Header         { return http.Header{} }
func (*mockNonFlushWriter) Write(b []byte) (int, error) { return len(b), nil }
func (*mockNonFlushWriter) WriteHeader(_ int)           {}

type errorWriter struct{}

func (*errorWriter) Header() http.Header { return http.Header{} }
func (*errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
func (*errorWriter) WriteHeader(_ int) {}
func (*errorWriter) Flush()            {}

func TestNewSSEStream(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		if stream == nil {
			t.Fatal("expected non-nil SSEStream")
		}
	})

	t.Run("NotFlusher", func(t *testing.T) {
		t.Parallel()

		w := &mockNonFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		if stream != nil {
			t.Error("expected nil SSEStream for non-flusher writer")
		}
	})
}

func TestSSEStreamSend(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "event: update\n") {
			t.Errorf("expected 'event: update' in output, got %q", output)
		}
		if !strings.Contains(output, `data: {"key":"value"}`) {
			t.Errorf("expected JSON data in output, got %q", output)
		}
		if w.flushCount != 1 {
			t.Errorf("expected 1 flush, got %d", w.flushCount)
		}
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", "data")
		if err == nil {
			t.Error("expected error for disconnected client")
		}
		if !errors.Is(err, errClientDisconnected) {
			t.Errorf("expected errClientDisconnected, got %v", err)
		}
	})

	t.Run("MarshalError", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", func() {})
		if err == nil {
			t.Error("expected error for non-marshallable data")
		}
		if !strings.Contains(err.Error(), "encoding SSE data") {
			t.Errorf("expected 'encoding SSE data' error, got %v", err)
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()

		w := &errorWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", "data")
		if err == nil {
			t.Error("expected error for write failure")
		}
		if !strings.Contains(err.Error(), "writing SSE event") {
			t.Errorf("expected 'writing SSE event' error, got %v", err)
		}
	})
}

func TestSSEStreamSendData(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData(map[string]int{"count": 42})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if strings.Contains(output, "event:") {
			t.Errorf("expected no event type in output, got %q", output)
		}
		if !strings.Contains(output, `data: {"count":42}`) {
			t.Errorf("expected JSON data in output, got %q", output)
		}
		if w.flushCount != 1 {
			t.Errorf("expected 1 flush, got %d", w.flushCount)
		}
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendData("data")
		if err == nil {
			t.Error("expected error for disconnected client")
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()

		w := &errorWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData("data")
		if err == nil {
			t.Error("expected error for write failure")
		}
		if !strings.Contains(err.Error(), "writing SSE data") {
			t.Errorf("expected 'writing SSE data' error, got %v", err)
		}
	})
}

func TestSSEStreamSendComplete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendComplete(map[string]string{"status": "done"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "event: complete\n") {
			t.Errorf("expected 'event: complete' in output, got %q", output)
		}
	})
}

func TestSSEStreamSendError(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendError(errors.New("something went wrong"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "event: error\n") {
			t.Errorf("expected 'event: error' in output, got %q", output)
		}
		if !strings.Contains(output, "something went wrong") {
			t.Errorf("expected error message in output, got %q", output)
		}
	})
}

func TestSSEStreamSendHeartbeat(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendHeartbeat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, ": heartbeat\n\n") {
			t.Errorf("expected heartbeat comment in output, got %q", output)
		}
		if w.flushCount != 1 {
			t.Errorf("expected 1 flush, got %d", w.flushCount)
		}
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendHeartbeat()
		if err == nil {
			t.Error("expected error for disconnected client")
		}
	})
}

func TestSSEStreamDone(t *testing.T) {
	t.Run("ReturnsChannel", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		doneChannel := stream.Done()
		if doneChannel == nil {
			t.Fatal("expected non-nil done channel")
		}

		select {
		case <-doneChannel:
			t.Error("expected channel to be open")
		default:

		}
	})

	t.Run("ClosedChannel", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		doneChannel := stream.Done()
		select {
		case <-doneChannel:

		default:
			t.Error("expected channel to be closed")
		}
	})
}

func TestSSEStreamEnableEventIDs(t *testing.T) {
	t.Run("SendIncludesIDWhenEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.Send("update", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, "event: update\n") {
			t.Errorf("expected 'event: update' in output, got %q", output)
		}
	})

	t.Run("SendOmitsIDWhenNotEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if strings.Contains(output, "id:") {
			t.Errorf("expected no 'id:' in output when IDs not enabled, got %q", output)
		}
	})

	t.Run("IDsAutoIncrement", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		for i := 1; i <= 3; i++ {
			err := stream.Send("update", map[string]int{"n": i})
			if err != nil {
				t.Fatalf("unexpected error on send %d: %v", i, err)
			}
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 2\n") {
			t.Errorf("expected 'id: 2' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 3\n") {
			t.Errorf("expected 'id: 3' in output, got %q", output)
		}
	})
}

func TestSSEStreamEventIDsWithSendData(t *testing.T) {
	t.Run("SendDataIncludesIDWhenEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.SendData(map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, `data: {"key":"value"}`) {
			t.Errorf("expected JSON data in output, got %q", output)
		}
	})

	t.Run("SendDataOmitsIDWhenNotEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData(map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if strings.Contains(output, "id:") {
			t.Errorf("expected no 'id:' in output when IDs not enabled, got %q", output)
		}
	})
}

func TestSSEStreamEventIDsWithSendComplete(t *testing.T) {
	t.Run("SendCompleteIncludesIDWhenEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.SendComplete(map[string]string{"status": "done"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, "event: complete\n") {
			t.Errorf("expected 'event: complete' in output, got %q", output)
		}
	})
}

func TestSSEStreamEventIDsAutoIncrement(t *testing.T) {
	t.Run("MixedSendMethodsIncrementSequentially", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.Send("update", map[string]string{"step": "first"})
		if err != nil {
			t.Fatalf("unexpected error on Send: %v", err)
		}

		err = stream.SendData(map[string]string{"step": "second"})
		if err != nil {
			t.Fatalf("unexpected error on SendData: %v", err)
		}

		err = stream.Send("update", map[string]string{"step": "third"})
		if err != nil {
			t.Fatalf("unexpected error on second Send: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 2\n") {
			t.Errorf("expected 'id: 2' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 3\n") {
			t.Errorf("expected 'id: 3' in output, got %q", output)
		}
	})
}

func TestSSEStreamLastEventID(t *testing.T) {
	t.Run("EmptyWhenNotProvided", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		if got := stream.LastEventID(); got != "" {
			t.Errorf("expected empty LastEventID, got %q", got)
		}
	})

	t.Run("ReturnsInjectedValue", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "42")

		if got := stream.LastEventID(); got != "42" {
			t.Errorf("expected LastEventID %q, got %q", "42", got)
		}
	})
}

func TestSSEStreamHeartbeatNoID(t *testing.T) {
	t.Run("HeartbeatOmitsIDEvenWhenEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.SendHeartbeat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if strings.Contains(output, "id:") {
			t.Errorf("expected no 'id:' in heartbeat output, got %q", output)
		}
		if !strings.Contains(output, ": heartbeat\n\n") {
			t.Errorf("expected heartbeat comment in output, got %q", output)
		}
	})

	t.Run("NextSendAfterHeartbeatGetsCorrectID", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.Send("update", "first")
		if err != nil {
			t.Fatalf("unexpected error on first Send: %v", err)
		}

		err = stream.SendHeartbeat()
		if err != nil {
			t.Fatalf("unexpected error on SendHeartbeat: %v", err)
		}

		err = stream.Send("update", "second")
		if err != nil {
			t.Fatalf("unexpected error on second Send: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 2\n") {
			t.Errorf("expected 'id: 2' in output, got %q", output)
		}
	})
}

func TestSSEStreamSendWithID(t *testing.T) {
	t.Run("SendsCustomEventID", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendWithID("42", "chat", map[string]string{"text": "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := w.writer.String()
		if !strings.Contains(output, "id: 42\n") {
			t.Errorf("expected 'id: 42' in output, got %q", output)
		}
		if !strings.Contains(output, "event: chat\n") {
			t.Errorf("expected 'event: chat' in output, got %q", output)
		}
		if !strings.Contains(output, `"text":"hello"`) {
			t.Errorf("expected JSON data in output, got %q", output)
		}
		if w.flushCount == 0 {
			t.Error("expected Flush to be called")
		}
	})

	t.Run("DoesNotAffectAutoIncrementingIDs", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		_ = stream.SendWithID("100", "chat", "msg1")

		_ = stream.Send("chat", "msg2")

		output := w.writer.String()
		if !strings.Contains(output, "id: 100\n") {
			t.Errorf("expected 'id: 100' in output, got %q", output)
		}
		if !strings.Contains(output, "id: 1\n") {
			t.Errorf("expected 'id: 1' in output, got %q", output)
		}
	})

	t.Run("ReturnsErrorOnClientDisconnect", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendWithID("1", "chat", "message")
		if err == nil {
			t.Fatal("expected error for disconnected client")
		}
	})
}
