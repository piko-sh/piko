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

package interp_domain

var (
	// boolStringFalse holds the "false" string header that the inlined strconv.FormatBool
	// tier-1 sub-op stamps into strings[B] when the source bool is false. The underlying
	// bytes live in .rodata, so the data pointer is stable across the program's lifetime; no
	// GC interaction required.
	//
	//nolint:unused // referenced from .s files via Plan-9 symbol resolution.
	boolStringFalse = "false"

	// boolStringTrue holds the "true" string header that the inlined strconv.FormatBool
	// tier-1 sub-op stamps into strings[B] when the source bool is true. Stored as a
	// package-level var so the assembler can LEA the data pointer via *boolStringTrue(SB).
	//
	//nolint:unused // referenced from .s files via Plan-9 symbol resolution.
	boolStringTrue = "true"
)
