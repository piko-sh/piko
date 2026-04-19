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
	"time"
)

// sseDeadlineWriter wraps an http.ResponseWriter and applies a per-write
// deadline before each Write call so a slow consumer cannot hold the
// connection open indefinitely. The wrapper preserves http.Flusher semantics
// for SSE streaming.
type sseDeadlineWriter struct {
	// ResponseWriter is the underlying writer the deadline is applied to.
	http.ResponseWriter

	// controller drives SetWriteDeadline on the underlying connection.
	controller *http.ResponseController

	// timeout bounds how long any single Write may block.
	timeout time.Duration
}

// newSSEDeadlineWriter wraps w so each subsequent Write call is preceded by
// SetWriteDeadline(now + timeout). When timeout is non-positive the wrapper
// is returned unmodified for callers that prefer to disable the deadline.
//
// Takes w (http.ResponseWriter) which is the underlying writer for the SSE
// response.
// Takes timeout (time.Duration) which bounds individual write durations.
//
// Returns http.ResponseWriter which applies the deadline before each write.
func newSSEDeadlineWriter(w http.ResponseWriter, timeout time.Duration) http.ResponseWriter {
	if timeout <= 0 {
		return w
	}
	return &sseDeadlineWriter{
		ResponseWriter: w,
		controller:     http.NewResponseController(w),
		timeout:        timeout,
	}
}

// Write applies the configured write deadline to the underlying connection
// and forwards the data. SetWriteDeadline failures are ignored so backends
// without deadline support degrade to the original behaviour.
//
// Takes p ([]byte) which holds the bytes to write.
//
// Returns int which is the number of bytes written.
// Returns error which is the underlying writer's error, if any.
func (w *sseDeadlineWriter) Write(p []byte) (int, error) {
	_ = w.controller.SetWriteDeadline(time.Now().Add(w.timeout))
	return w.ResponseWriter.Write(p)
}

// Flush forwards Flush to the underlying writer's controller so SSE clients
// receive events immediately.
//
// Returns nothing; flush errors are ignored, mirroring the standard
// http.Flusher contract.
func (w *sseDeadlineWriter) Flush() {
	_ = w.controller.Flush()
}

// Unwrap exposes the wrapped writer for response controllers that walk
// wrappers via the Unwrap convention.
//
// Returns http.ResponseWriter which is the wrapped writer.
func (w *sseDeadlineWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
