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

// resolveFrame returns the function name, file path, and line number for this
// program counter.
//
// This is the safe (allocating) version used when building with -tags safe.
// It uses runtime.CallersFrames which allocates internally.
//
// Returns name (string) which is the fully qualified function name.
// Returns file (string) which is the full file path.
// Returns line (int) which is the line number.
func (pc PC) resolveFrame() (name, file string, line int) {
	fs := runtime.CallersFrames([]uintptr{uintptr(pc)})
	f, _ := fs.Next()
	return f.Function, f.File, f.Line
}
