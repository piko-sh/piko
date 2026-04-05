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
	// mu serialises access to the underlying writer.
	mu *sync.Mutex

	// w is the destination writer.
	w io.Writer
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

// HoldWrites acquires the write lock and returns a release
// function that resumes writes when called.
//
// Returns func() which releases the write lock when called.
//
// Concurrency: acquires lw.mu; the returned function releases
// it.
func (lw *LockedWriter) HoldWrites() func() {
	lw.mu.Lock()
	return lw.mu.Unlock
}

var (
	// terminalMu serialises writes to both os.Stdout and os.Stderr. A single
	// mutex is shared across both streams because they usually point at the
	// same TTY in interactive terminals; independent mutexes would allow log
	// output on one fd to interleave with banner output on the other at the
	// kernel TTY level, corrupting multi-line output.
	terminalMu sync.Mutex

	stderrWriter = &LockedWriter{w: os.Stderr, mu: &terminalMu}

	stdoutWriter = &LockedWriter{w: os.Stdout, mu: &terminalMu}
)

// StderrWriter returns a shared LockedWriter that serialises writes to
// os.Stderr. All callers receive the same instance, so log handlers and
// other stderr writers (such as the startup banner) automatically
// serialise against each other and against stdout writers returned by
// [StdoutWriter].
//
// Returns *LockedWriter which wraps os.Stderr with the shared terminal mutex.
func StderrWriter() *LockedWriter {
	return stderrWriter
}

// StdoutWriter returns a shared LockedWriter that serialises writes to
// os.Stdout using the same mutex as [StderrWriter]. Routing stdout-bound
// handlers through this writer prevents log output from interleaving with
// stderr-bound output (such as the startup banner) at the kernel TTY level
// when both streams share a terminal.
//
// Returns *LockedWriter which wraps os.Stdout with the shared terminal mutex.
func StdoutWriter() *LockedWriter {
	return stdoutWriter
}
