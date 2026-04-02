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

//go:build safe || (js && wasm)

package caller

import "runtime"

// callers captures multiple stack frames into the provided slice.
//
// This is the safe version that uses runtime.Callers from the standard library.
// It allocates a temporary []uintptr buffer per call.
//
// Takes skip (int) which specifies the number of stack frames to skip.
// Takes pcs ([]PC) which is the slice to store the captured program counters.
//
// Returns int which is the number of frames actually captured.
func callers(skip int, pcs []PC) int {
	buffer := make([]uintptr, len(pcs))
	n := runtime.Callers(skip+2, buffer)
	for i := range n {
		pcs[i] = PC(buffer[i])
	}
	return n
}

// caller1 captures a single stack frame at the given skip depth.
//
// This is the safe version that uses runtime.Callers from the standard library.
// It uses a stack-allocated buffer to minimise overhead.
//
// The +2 skip accounts for both this function and [runtime.Callers] itself.
// The public dispatch functions (Caller, Callers, CallersFill) are marked
// //go:noinline in the safe build to guarantee a consistent frame layout,
// so the skip arithmetic is always correct.
//
// Takes skip (int) which specifies the number of stack frames to skip
// before capturing.
// Takes pc (*PC) which receives the captured program counter value.
//
// Returns int which is 1 if a frame was captured, 0 otherwise.
func caller1(skip int, pc *PC, _, _ int) int { //nolint:revive // go:linkname signature
	var buffer [1]uintptr
	n := runtime.Callers(skip+2, buffer[:])
	if n > 0 {
		*pc = PC(buffer[0])
	}
	return n
}
