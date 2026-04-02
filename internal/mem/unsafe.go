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

package mem

import "unsafe"

// String converts a byte slice to a string without memory allocation.
//
// The returned string shares memory with the input slice. The caller must
// ensure the input slice outlives the string and is not changed while the
// string is in use.
//
// Takes b ([]byte) which is the byte slice to convert.
//
// Returns string which shares memory with the input slice.
func String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

// Bytes converts a string to a byte slice without memory allocation.
//
// The returned slice shares memory with the input string. The caller must not
// change the returned slice, as strings cannot be changed in Go. Changing the
// slice will corrupt the string and cause undefined behaviour.
//
// Takes s (string) which is the string to convert.
//
// Returns []byte which shares memory with the input string, or nil if empty.
func Bytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
