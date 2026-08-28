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
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// defaultThrottleBacklogTimeout bounds how long a request waits for a slot before it is
	// shed. It matches the value the previous middleware used, so behaviour under load is
	// unchanged.
	defaultThrottleBacklogTimeout = 60 * time.Second
)

// streamAwareThrottle bounds how many requests run at once, releasing a stream's slot as
// soon as it starts streaming.
//
// Takes limit (int) which is the number of concurrent requests allowed, or zero to
// disable. Takes backlogTimeout (time.Duration) which bounds the wait for a slot.
//
// Returns func(http.Handler) http.Handler which is the middleware.
func streamAwareThrottle(limit int, backlogTimeout time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	if backlogTimeout <= 0 {
		backlogTimeout = defaultThrottleBacklogTimeout
	}

	slots := make(chan struct{}, limit)
	queue := make(chan struct{}, limit*2)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			select {
			case queue <- struct{}{}:
				defer func() { <-queue }()
			default:
				writeThrottled(writer, backlogTimeout)

				return
			}

			if !awaitThrottleSlot(writer, request, slots, backlogTimeout) {
				return
			}

			throttled := &throttleReleasingWriter{ResponseWriter: writer, release: func() { <-slots }}
			defer throttled.releaseOnce()

			next.ServeHTTP(throttled, request)
		})
	}
}

// awaitThrottleSlot waits for a concurrency slot, shedding the request if it waits too
// long or the client leaves.
//
// Takes writer (http.ResponseWriter) which receives the rejection.
// Takes request (*http.Request) which supplies the cancellation signal.
// Takes slots (chan struct{}) which is the pool of running slots.
// Takes backlogTimeout (time.Duration) which bounds the wait.
//
// Returns bool which is true when a slot was taken and the caller must release it.
func awaitThrottleSlot(
	writer http.ResponseWriter,
	request *http.Request,
	slots chan struct{},
	backlogTimeout time.Duration,
) bool {
	select {
	case slots <- struct{}{}:
		return true
	default:
	}

	timer := time.NewTimer(backlogTimeout)
	defer timer.Stop()

	select {
	case slots <- struct{}{}:
		return true
	case <-timer.C:
		writeThrottled(writer, backlogTimeout)

		return false
	case <-request.Context().Done():
		writeThrottled(writer, backlogTimeout)

		return false
	}
}

// writeThrottled reports that the server is already running as many requests as it
// allows.
//
// Takes writer (http.ResponseWriter) which receives the response.
// Takes retryAfter (time.Duration) which tells the caller when to try again.
func writeThrottled(writer http.ResponseWriter, retryAfter time.Duration) {
	writer.Header().Set("Retry-After", strconv.Itoa(max(int(retryAfter/time.Second), 1)))
	http.Error(writer, "Too Many Requests", http.StatusTooManyRequests)
}

// throttleReleasingWriter releases a concurrency slot as soon as the response it wraps
// declares itself a stream.
type throttleReleasingWriter struct {
	// ResponseWriter is the underlying writer the response is written to.
	http.ResponseWriter

	// release returns the slot to the pool. It runs at most once.
	release func()

	// once guards release so the deferred call after a stream ends is a no-op.
	once sync.Once
}

// WriteHeader releases the slot when the response declares itself a stream, then writes
// the status.
//
// Takes status (int) which is the response status code.
func (w *throttleReleasingWriter) WriteHeader(status int) {
	w.releaseIfStream()

	w.ResponseWriter.WriteHeader(status)
}

// Write releases the slot when the response declares itself a stream, then writes the
// body.
//
// Takes data ([]byte) which is the body to write.
//
// Returns int which is the number of bytes written.
// Returns error which is any write failure.
func (w *throttleReleasingWriter) Write(data []byte) (int, error) {
	w.releaseIfStream()

	return w.ResponseWriter.Write(data)
}

// Flush forwards a flush to the underlying writer when it supports one, which a stream
// needs after every event.
func (w *throttleReleasingWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap returns the underlying writer so http.ResponseController can reach the
// connection, which the streaming write deadline depends on.
//
// Returns http.ResponseWriter which is the writer this one wraps.
func (w *throttleReleasingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// releaseIfStream returns the slot when the response has declared itself a stream.
func (w *throttleReleasingWriter) releaseIfStream() {
	if strings.HasPrefix(w.Header().Get("Content-Type"), mediaTypeEventStream) {
		w.releaseOnce()
	}
}

// releaseOnce returns the slot to the pool, and does nothing on any later call.
func (w *throttleReleasingWriter) releaseOnce() {
	w.once.Do(w.release)
}
