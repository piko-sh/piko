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
)

type recordingResponseWriter struct {
	http.ResponseWriter
	deadlineCalls int64
	flushCalls    int64
	lastDeadline  time.Time
	deadlineErr   error
}

func (r *recordingResponseWriter) SetWriteDeadline(deadline time.Time) error {
	atomic.AddInt64(&r.deadlineCalls, 1)
	r.lastDeadline = deadline
	return r.deadlineErr
}

func (r *recordingResponseWriter) Flush() {
	atomic.AddInt64(&r.flushCalls, 1)
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestSSEDeadlineWriter_AppliesDeadlineBeforeEachWrite(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 5*time.Second)

	if _, err := writer.Write([]byte("event: foo\n\n")); err != nil {
		t.Fatalf("first write returned error: %v", err)
	}
	if _, err := writer.Write([]byte("event: bar\n\n")); err != nil {
		t.Fatalf("second write returned error: %v", err)
	}

	if calls := atomic.LoadInt64(&recorder.deadlineCalls); calls != 2 {
		t.Fatalf("expected SetWriteDeadline to be called twice, got %d", calls)
	}

	now := time.Now()
	if recorder.lastDeadline.Before(now.Add(4 * time.Second)) {
		t.Fatalf("expected deadline at least 4s in the future, got %v", recorder.lastDeadline.Sub(now))
	}
	if recorder.lastDeadline.After(now.Add(6 * time.Second)) {
		t.Fatalf("expected deadline within 6s of now, got %v", recorder.lastDeadline.Sub(now))
	}
}

func TestSSEDeadlineWriter_FlushesViaController(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 100*time.Millisecond)

	flusher, ok := writer.(http.Flusher)
	if !ok {
		t.Fatal("expected wrapped writer to implement http.Flusher")
	}
	flusher.Flush()

	if atomic.LoadInt64(&recorder.flushCalls) == 0 {
		t.Fatal("expected Flush to reach the underlying writer")
	}
}

func TestSSEDeadlineWriter_NonPositiveTimeoutIsPassthrough(t *testing.T) {
	t.Parallel()

	recorder := &recordingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	writer := newSSEDeadlineWriter(recorder, 0)

	if writer != http.ResponseWriter(recorder) {
		t.Fatal("expected zero timeout to return the writer unchanged")
	}
}
