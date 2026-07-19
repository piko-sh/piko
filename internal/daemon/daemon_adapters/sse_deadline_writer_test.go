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

package daemon_adapters

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingResponseWriter struct {
	http.ResponseWriter
	deadlineCalls atomic.Int64
	flushCalls    atomic.Int64
	lastDeadline  time.Time
	deadlineErr   error
}

func (r *recordingResponseWriter) SetWriteDeadline(deadline time.Time) error {
	r.deadlineCalls.Add(1)
	r.lastDeadline = deadline
	return r.deadlineErr
}

func (r *recordingResponseWriter) Flush() {
	r.flushCalls.Add(1)
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestSSEDeadlineWriter_AppliesDeadlineBeforeEachWrite(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 5*time.Second)

	_, err := writer.Write([]byte("event: foo\n\n"))
	require.NoError(t, err, "first write returned error")
	_, err = writer.Write([]byte("event: bar\n\n"))
	require.NoError(t, err, "second write returned error")

	require.Equal(t, int64(2), recorder.deadlineCalls.Load(), "expected SetWriteDeadline to be called twice")

	now := time.Now()
	require.Falsef(t, recorder.lastDeadline.Before(now.Add(4*time.Second)),
		"expected deadline at least 4s in the future, got %v", recorder.lastDeadline.Sub(now))
	require.Falsef(t, recorder.lastDeadline.After(now.Add(6*time.Second)),
		"expected deadline within 6s of now, got %v", recorder.lastDeadline.Sub(now))
}

func TestSSEDeadlineWriter_FlushesViaController(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 100*time.Millisecond)

	flusher, ok := writer.(http.Flusher)
	require.True(t, ok, "expected wrapped writer to implement http.Flusher")
	flusher.Flush()

	require.NotZero(t, recorder.flushCalls.Load(), "expected Flush to reach the underlying writer")
}

func TestSSEDeadlineWriter_NonPositiveTimeoutIsPassthrough(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 0)

	require.Same(t, recorder, writer, "expected zero timeout to return the writer unchanged")
}
