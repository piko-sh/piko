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

package ast_domain

// Provides the zero-allocation version of string interning using unsafe memory operations for production builds.
// Creates temporary string views for map lookups without allocating memory, optimising performance for high-frequency string operations.

import "piko.sh/piko/internal/mem"

// internBytes returns a stored string if the bytes match a known value, or
// creates a new string from the bytes.
//
// Avoids memory allocation by using mem.String to create a temporary string
// view for the map lookup. If found, returns the stored string.
//
// Takes b ([]byte) which contains the bytes to look up or convert.
//
// Returns string which is the stored string if found, or a new string from the
// bytes.
func internBytes(b []byte) string {
	key := mem.String(b)
	if interned, ok := internedStrings[key]; ok {
		return interned
	}
	return string(b)
}
