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

// Caller returns the program counter of the caller's stack frame.
//
// The argument skip is the number of frames to ascend, with 0 identifying the
// caller of Caller.
//
// Achieves zero allocations by using go:linkname to access runtime.callers
// directly with pointer semantics.
//
// Takes skip (int) specifying how many stack frames to skip.
//
// Returns PC which is the program counter of the target frame, or 0 if the
// frame could not be captured.
func Caller(skip int) PC {
	var pc PC
	caller1(1+skip, &pc, 1, 1)
	return pc
}

// Callers returns a stack trace of up to n frames, starting from skip frames
// above the caller.
//
// The only allocation is the result slice itself.
//
// Takes skip (int) specifying how many stack frames to skip.
// Takes n (int) specifying the maximum number of frames to capture.
//
// Returns PCs which is a slice of program counters. The slice length is the
// actual number of frames captured, which may be less than n.
func Callers(skip, n int) PCs {
	pcs := make([]PC, n)
	captured := callers(1+skip, pcs)
	return pcs[:captured]
}

// CallersFill captures stack frames into the provided slice, avoiding the
// allocation of a new slice.
//
// Takes skip (int) specifying how many stack frames to skip.
// Takes pcs (PCs) which is the slice to fill with program counters.
//
// Returns PCs which is a slice of the captured frames. The returned slice
// shares backing memory with the input slice.
func CallersFill(skip int, pcs PCs) PCs {
	captured := callers(1+skip, pcs)
	return pcs[:captured]
}
