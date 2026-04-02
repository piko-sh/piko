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

package logger_domain

import (
	"io"
	"os"
	"sync"
)

// LockedWriter serialises writes to an underlying io.Writer using a mutex.
// All writes through LockedWriter instances sharing the same mutex are
// guaranteed not to interleave, preventing corrupted output when multiple
// goroutines write concurrently to the same destination.
//
// Use HoldWrites to temporarily prevent all writes (for example, while
// printing a startup banner that must not be interrupted by log output).
type LockedWriter struct {
	mu *sync.Mutex

	w  io.Writer
}

// NewLockedWriter creates a LockedWriter that serialises writes to w
// using mu.
//
// Takes w (io.Writer) which is the destination writer.
// Takes mu (*sync.Mutex) which serialises access.
//
// Returns *LockedWriter which serialises all writes through mu.
func NewLockedWriter(w io.Writer, mu *sync.Mutex) *LockedWriter {
	return &LockedWriter{w: w, mu: mu}
}

// Write acquires the mutex, writes p to the underlying writer, and releases
// the mutex. This ensures that concurrent Write calls do not interleave.
//
// Takes p ([]byte) which is the data to write.
//
// Returns int which is the number of bytes written.
// Returns error when the underlying writer fails.
func (lw *LockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}

// HoldWrites acquires the write lock and returns a release function. While
// the lock is held, no writes through this LockedWriter (or any writer
// sharing the same mutex) can proceed. Call the returned function to resume
// writes.
//
// This is intended for exclusive access to the underlying file descriptor,
// for example when printing a multi-line startup banner that must appear
// intact.
//
// Returns func() which releases the write lock when called.
func (lw *LockedWriter) HoldWrites() func() {
	lw.mu.Lock()
	return lw.mu.Unlock
}

var (
	stderrMu     sync.Mutex

	stderrWriter = &LockedWriter{w: os.Stderr, mu: &stderrMu}
)

// StderrWriter returns a shared LockedWriter that serialises writes to
// os.Stderr. All callers receive the same instance, so log handlers and
// other stderr writers (such as the startup banner) automatically
// serialise against each other.
//
// Returns *LockedWriter which wraps os.Stderr with a shared mutex.
func StderrWriter() *LockedWriter {
	return stderrWriter
}
