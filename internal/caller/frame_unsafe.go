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

//go:build !safe && !(js && wasm)

package caller

import (
	"runtime"
	"unsafe"
)

// runtimeFrame is an alias for runtime.Frame, used for type clarity.
type runtimeFrame = runtime.Frame

// runtimeFrames mirrors the internal layout of runtime.Frames as of Go 1.23+.
//
// This struct must be kept in sync with Go's internal
// implementation. The layout was verified against Go 1.23 and later
// versions. The nextPC field was added in Go 1.23 to support
// improved inlining information.
//
// IMPORTANT: Field order is load-bearing - it must match
// runtime.Frames exactly. Do NOT reorder these fields. The struct
// is cast via unsafe.Pointer.
//
//nolint:govet // must match runtime.Frames layout
type runtimeFrames struct {
	// ptr points to the current program counter in the slice.
	ptr *PC //nolint:unused // accessed via unsafe.Pointer

	// length holds the number of program counters remaining.
	length int //nolint:unused // accessed via unsafe.Pointer

	// buffer stores a single program counter for stack-only resolution.
	buffer PC //nolint:unused // accessed via unsafe.Pointer

	// nextPC holds the return address used for inlining information since Go 1.23.
	nextPC PC //nolint:unused // accessed via unsafe.Pointer

	// frames holds the resolved runtime frames for iteration.
	frames []runtimeFrame //nolint:unused // accessed via unsafe.Pointer

	// frameStore provides inline backing storage to avoid heap allocation.
	frameStore [2]runtimeFrame //nolint:unused // accessed via unsafe.Pointer
}

// resolveFrame returns the function name, file path, and line number for this
// program counter.
//
// This method builds a runtime.Frames struct on the stack to avoid heap
// allocation. It then casts the struct to *runtime.Frames and calls Next()
// to get the frame details.
//
// Returns name (string) which is the fully qualified function name.
// Returns file (string) which is the full file path.
// Returns line (int) which is the line number.
//
//go:nocheckptr
func (pc PC) resolveFrame() (name, file string, line int) {
	fs := &runtimeFrames{}
	fs.buffer = pc
	fs.ptr = &fs.buffer
	fs.length = 1
	fs.frames = fs.frameStore[:0]

	r := (*runtime.Frames)(unsafe.Pointer(fs))
	f, _ := r.Next()

	return f.Function, f.File, f.Line
}
