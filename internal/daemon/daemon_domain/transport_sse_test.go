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
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/safeerror"
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

		require.NotNil(t, stream, "expected non-nil SSEStream")
	})

	t.Run("NotFlusher", func(t *testing.T) {
		t.Parallel()

		w := &mockNonFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		assert.Nil(t, stream, "expected nil SSEStream for non-flusher writer")
	})
}

func TestSSEStreamSend(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", map[string]string{"key": "value"})
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "event: update\n", "expected 'event: update' in output")
		assert.Contains(t, output, `data: {"key":"value"}`, "expected JSON data in output")
		assert.Equal(t, 1, w.flushCount, "expected 1 flush")
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", "data")
		assert.Error(t, err, "expected error for disconnected client")
		assert.ErrorIs(t, err, errClientDisconnected, "expected errClientDisconnected")
	})

	t.Run("MarshalError", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", func() {})
		require.Error(t, err, "expected error for non-marshallable data")
		assert.ErrorContains(t, err, "encoding SSE data", "expected 'encoding SSE data' error")
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()

		w := &errorWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", "data")
		require.Error(t, err, "expected error for write failure")
		assert.ErrorContains(t, err, "writing SSE event", "expected 'writing SSE event' error")
	})
}

func TestSSEStreamSendData(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData(map[string]int{"count": 42})
		require.NoError(t, err)

		output := w.writer.String()
		assert.NotContains(t, output, "event:", "expected no event type in output")
		assert.Contains(t, output, `data: {"count":42}`, "expected JSON data in output")
		assert.Equal(t, 1, w.flushCount, "expected 1 flush")
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendData("data")
		assert.Error(t, err, "expected error for disconnected client")
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()

		w := &errorWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData("data")
		require.Error(t, err, "expected error for write failure")
		assert.ErrorContains(t, err, "writing SSE data", "expected 'writing SSE data' error")
	})
}

func TestSSEStreamSendComplete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendComplete(map[string]string{"status": "done"})
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "event: complete\n", "expected 'event: complete' in output")
	})
}

func TestSSEStreamSendError(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.SetDevelopmentMode(true)

		err := stream.SendError(errors.New("something went wrong"))
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "event: error\n", "expected 'event: error' in output")
		assert.Contains(t, output, "something went wrong", "expected error message in output")
	})
}

func TestSendError_ProductionRedactsErrorText(t *testing.T) {
	t.Parallel()

	w := &mockFlushWriter{}
	done := make(chan struct{})
	stream := NewSSEStream(w, done, "")

	rawMessage := "internal database connection refused at host db-prod-7"
	err := stream.SendError(errors.New(rawMessage))
	require.NoError(t, err)

	output := w.writer.String()
	assert.NotContains(t, output, rawMessage, "expected raw error to be redacted in production")
	assert.Contains(t, output, "An internal error occurred", "expected production placeholder message")
	assert.Contains(t, output, "event: error\n", "expected 'event: error' in output")
}

func TestSendError_DevelopmentExposesErrorText(t *testing.T) {
	t.Parallel()

	w := &mockFlushWriter{}
	done := make(chan struct{})
	stream := NewSSEStream(w, done, "")
	stream.SetDevelopmentMode(true)

	rawMessage := "internal database connection refused at host db-prod-7"
	err := stream.SendError(errors.New(rawMessage))
	require.NoError(t, err)

	output := w.writer.String()
	assert.Contains(t, output, rawMessage, "expected raw error visible in development")
}

func TestSendError_ProductionPropagatesSafeMessage(t *testing.T) {
	t.Parallel()

	w := &mockFlushWriter{}
	done := make(chan struct{})
	stream := NewSSEStream(w, done, "")

	safeErr := safeerror.NewError("user-facing summary", errors.New("internal: connection pool exhausted"))
	require.NoError(t, stream.SendError(safeErr))

	output := w.writer.String()
	assert.Contains(t, output, "user-facing summary", "expected safe message in output")
	assert.NotContains(t, output, "connection pool exhausted", "expected internal cause to be redacted")
}

func TestSetDevelopmentModeFromContext_HonoursPikoRequestCtx(t *testing.T) {
	t.Parallel()

	t.Run("DevelopmentEnabledInCtx", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		pctx := &daemon_dto.PikoRequestCtx{DevelopmentMode: true}
		ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)
		stream.SetDevelopmentModeFromContext(ctx)

		require.NoError(t, stream.SendError(errors.New("verbose internal detail")))
		assert.Contains(t, w.writer.String(), "verbose internal detail", "expected dev message visible")
	})

	t.Run("NoCarrierFallsBackToProduction", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.SetDevelopmentModeFromContext(context.Background())

		require.NoError(t, stream.SendError(errors.New("verbose internal detail")))
		assert.NotContains(t, w.writer.String(), "verbose internal detail",
			"expected raw detail redacted with no carrier")
	})
}

func TestSSEStreamSendHeartbeat(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendHeartbeat()
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, ": heartbeat\n\n", "expected heartbeat comment in output")
		assert.Equal(t, 1, w.flushCount, "expected 1 flush")
	})

	t.Run("ClientDisconnected", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendHeartbeat()
		assert.Error(t, err, "expected error for disconnected client")
	})
}

func TestSSEStreamDone(t *testing.T) {
	t.Run("ReturnsChannel", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		doneChannel := stream.Done()
		require.NotNil(t, doneChannel, "expected non-nil done channel")

		select {
		case <-doneChannel:
			assert.Fail(t, "expected channel to be open")
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
			assert.Fail(t, "expected channel to be closed")
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
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, "event: update\n", "expected 'event: update' in output")
	})

	t.Run("SendOmitsIDWhenNotEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.Send("update", map[string]string{"key": "value"})
		require.NoError(t, err)

		output := w.writer.String()
		assert.NotContains(t, output, "id:", "expected no 'id:' in output when IDs not enabled")
	})

	t.Run("IDsAutoIncrement", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		for i := 1; i <= 3; i++ {
			err := stream.Send("update", map[string]int{"n": i})
			require.NoErrorf(t, err, "unexpected error on send %d", i)
		}

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, "id: 2\n", "expected 'id: 2' in output")
		assert.Contains(t, output, "id: 3\n", "expected 'id: 3' in output")
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
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, `data: {"key":"value"}`, "expected JSON data in output")
	})

	t.Run("SendDataOmitsIDWhenNotEnabled", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendData(map[string]string{"key": "value"})
		require.NoError(t, err)

		output := w.writer.String()
		assert.NotContains(t, output, "id:", "expected no 'id:' in output when IDs not enabled")
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
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, "event: complete\n", "expected 'event: complete' in output")
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
		require.NoError(t, err, "unexpected error on Send")

		err = stream.SendData(map[string]string{"step": "second"})
		require.NoError(t, err, "unexpected error on SendData")

		err = stream.Send("update", map[string]string{"step": "third"})
		require.NoError(t, err, "unexpected error on second Send")

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, "id: 2\n", "expected 'id: 2' in output")
		assert.Contains(t, output, "id: 3\n", "expected 'id: 3' in output")
	})
}

func TestSSEStreamLastEventID(t *testing.T) {
	t.Run("EmptyWhenNotProvided", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		assert.Empty(t, stream.LastEventID(), "expected empty LastEventID")
	})

	t.Run("ReturnsInjectedValue", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "42")

		assert.Equal(t, "42", stream.LastEventID(), "expected LastEventID")
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
		require.NoError(t, err)

		output := w.writer.String()
		assert.NotContains(t, output, "id:", "expected no 'id:' in heartbeat output")
		assert.Contains(t, output, ": heartbeat\n\n", "expected heartbeat comment in output")
	})

	t.Run("NextSendAfterHeartbeatGetsCorrectID", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")
		stream.EnableEventIDs()

		err := stream.Send("update", "first")
		require.NoError(t, err, "unexpected error on first Send")

		err = stream.SendHeartbeat()
		require.NoError(t, err, "unexpected error on SendHeartbeat")

		err = stream.Send("update", "second")
		require.NoError(t, err, "unexpected error on second Send")

		output := w.writer.String()
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
		assert.Contains(t, output, "id: 2\n", "expected 'id: 2' in output")
	})
}

func TestSSEStreamSendWithID(t *testing.T) {
	t.Run("SendsCustomEventID", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		stream := NewSSEStream(w, done, "")

		err := stream.SendWithID("42", "chat", map[string]string{"text": "hello"})
		require.NoError(t, err)

		output := w.writer.String()
		assert.Contains(t, output, "id: 42\n", "expected 'id: 42' in output")
		assert.Contains(t, output, "event: chat\n", "expected 'event: chat' in output")
		assert.Contains(t, output, `"text":"hello"`, "expected JSON data in output")
		assert.Positive(t, w.flushCount, "expected Flush to be called")
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
		assert.Contains(t, output, "id: 100\n", "expected 'id: 100' in output")
		assert.Contains(t, output, "id: 1\n", "expected 'id: 1' in output")
	})

	t.Run("ReturnsErrorOnClientDisconnect", func(t *testing.T) {
		t.Parallel()

		w := &mockFlushWriter{}
		done := make(chan struct{})
		close(done)
		stream := NewSSEStream(w, done, "")

		err := stream.SendWithID("1", "chat", "message")
		require.Error(t, err, "expected error for disconnected client")
	})
}
