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

// Package caller captures stack frames with zero allocations on the
// hot path.
//
// It is an optimised alternative to Go's [runtime.Caller] and
// [runtime.Callers] functions, using go:linkname to access
// runtime.callers directly, constructing [runtime.Frames] on the
// stack to avoid heap allocation, and caching resolved frame
// information for repeated lookups.
//
// # Usage
//
//	pc := caller.Caller(0) // Get current location
//	name, file, line := pc.NameFileLine()
//
//	// Capture a stack trace
//	pcs := caller.Callers(0, 10)
//	for _, pc := range pcs {
//	    fmt.Println(pc.FormattedFrame())
//	}
//
// # Performance
//
// The default implementation uses unsafe operations to avoid
// allocations. [Caller] and cached [PC.NameFileLine] calls are
// allocation-free. Resolved frame information is cached in a
// [sync.Map] for efficient concurrent access.
//
// # Safe mode
//
// Build with the "safe" tag to use allocating implementations
// that avoid unsafe operations. This is useful for debugging
// memory issues.
//
// # Thread safety
//
// All exported functions and methods are safe for concurrent use.
// The frame cache uses [sync.Map] internally.
package caller
