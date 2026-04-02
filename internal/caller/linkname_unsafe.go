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

import _ "unsafe" // Required for go:linkname

// callers captures multiple stack frames into the provided slice.
// This is a direct link to runtime.callers, avoiding the overhead of
// runtime.Callers which creates a slice header on each call.
//
// Takes skip (int) which specifies the number of stack frames to skip.
// Takes pc ([]PC) which is the slice to fill with program counter values.
//
// Returns int which is the number of frames actually captured.
//
//go:noescape
//go:linkname callers runtime.callers
func callers(skip int, pc []PC) int

// caller1 captures a single stack frame at the given skip depth.
//
// By passing a pointer with explicit length and capacity, this avoids slice
// header allocation entirely. This is the same runtime.callers function but
// called with pointer semantics for maximum efficiency.
//
// Takes skip (int) which specifies the number of stack frames to skip.
// Takes pc (*PC) which receives the captured program counter.
// Takes length (int) which specifies the current length of the buffer.
// Takes capacity (int) which specifies the maximum capacity of the buffer.
//
// Returns int which is 1 if a frame was captured, 0 otherwise.
//
//go:noescape
//go:linkname caller1 runtime.callers
func caller1(skip int, pc *PC, length, capacity int) int //nolint:revive // go:linkname signature
