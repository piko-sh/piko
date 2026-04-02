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
	"sync"

	"piko.sh/piko/internal/caller"
)

var (
	pcPool = sync.Pool{
		New: func() any {
			return new(make(caller.PCs, 32))
		},
	}

	frameSlicePool = sync.Pool{
		New: func() any {
			return new(make([]string, 0, 32))
		},
	}
)

// stackTraceProvider abstracts the mechanism for capturing stack traces,
// enabling mock implementations to be injected for deterministic testing.
type stackTraceProvider interface {
	// CaptureStackTrace captures a stack trace starting from the specified number
	// of frames to skip.
	//
	// Takes skip (int) which is the number of stack frames to skip.
	// Takes maxFrames (int) which is the maximum number of frames to capture.
	//
	// Returns pc (uintptr) which is the program counter for the caller.
	// Returns trace (StackTrace) which is the full stack trace.
	CaptureStackTrace(skip int, maxFrames int) (pc uintptr, trace StackTrace)
}

// callerStackTraceProvider captures stack traces using the internal caller
// package. It implements stackTraceProvider for production use.
type callerStackTraceProvider struct{}

// CaptureStackTrace implements stackTraceProvider using caller.CallersFill
// with pooled buffers to avoid allocation after warmup.
//
// Takes skip (int) which is the number of stack frames to skip before
// capturing.
// Takes maxFrames (int) which limits the number of frames to capture.
//
// Returns uintptr which is the program counter of the first frame.
// Returns StackTrace which contains the formatted file:line entries.
//
// The returned StackTrace should have Release() called after use to return
// the backing slice to the pool.
func (*callerStackTraceProvider) CaptureStackTrace(skip int, maxFrames int) (uintptr, StackTrace) {
	bufferPointer, ok := pcPool.Get().(*caller.PCs)
	if !ok {
		bufferPointer = new(make(caller.PCs, maxFrames))
	}
	buffer := *bufferPointer

	if maxFrames > len(buffer) {
		buffer = make(caller.PCs, maxFrames)
	}

	pcs := caller.CallersFill(skip-1, buffer[:maxFrames])
	if len(pcs) == 0 {
		pcPool.Put(bufferPointer)
		return 0, StackTrace{}
	}

	firstPC := uintptr(pcs[0])

	framesPtr, ok := frameSlicePool.Get().(*[]string)
	if !ok {
		framesPtr = new(make([]string, 0, maxFrames))
	}
	frames := (*framesPtr)[:0]

	for _, pc := range pcs {
		if frame := pc.FormattedFrame(); frame != "" {
			frames = append(frames, frame)
		}
	}

	pcPool.Put(bufferPointer)

	return firstPC, StackTrace{
		frames:  frames,
		poolPtr: framesPtr,
	}
}

// mockStackTraceProvider is a test double that returns set stack traces.
type mockStackTraceProvider struct { //nolint:unused // exported via export_test.go
	// trace is the stack trace returned by CaptureStackTrace.
	trace StackTrace //nolint:unused // exported via export_test.go

	// pc is the program counter value returned by CaptureStackTrace.
	pc uintptr //nolint:unused // exported via export_test.go
}

// CaptureStackTrace returns the predefined program counter and stack trace.
//
// Returns uintptr which is the mock program counter value.
// Returns StackTrace which is the mock stack trace data.
func (m *mockStackTraceProvider) CaptureStackTrace(_ int, _ int) (uintptr, StackTrace) { //nolint:unused // exported via export_test.go
	return m.pc, m.trace
}

// newRuntimeStackTraceProvider creates a stack trace provider for production use.
//
// Returns stackTraceProvider which captures stack traces using caller.StackTrace.
func newRuntimeStackTraceProvider() stackTraceProvider {
	return &callerStackTraceProvider{}
}

// newMockStackTraceProvider creates a new mock stack trace provider with
// predefined values.
//
// Takes pc (uintptr) which specifies the program counter value.
// Takes trace (StackTrace) which provides the predefined stack trace.
//
// Returns stackTraceProvider which is the configured mock provider.
func newMockStackTraceProvider(pc uintptr, trace StackTrace) stackTraceProvider { //nolint:unused // exported via export_test.go
	return &mockStackTraceProvider{
		pc:    pc,
		trace: trace,
	}
}

// newStackTraceFromFrames creates a StackTrace from a slice of frame strings.
// This is intended for testing; the returned StackTrace does not use pooling.
//
// Takes frames ([]string) which contains the stack frame strings.
//
// Returns StackTrace which is a non-pooled trace for testing purposes.
func newStackTraceFromFrames(frames []string) StackTrace { //nolint:unused // exported via export_test.go
	return StackTrace{frames: frames, poolPtr: nil}
}
