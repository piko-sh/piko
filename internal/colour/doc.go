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

// Package colour handles zero-allocation ANSI colour output for terminal text.
//
// Colours are represented by [Style] values that hold pre-computed ANSI escape
// sequences. Create a Style once with [New] and reuse it for every write.
// The hot-path methods [Style.WriteStart] and [Style.WriteReset] perform zero
// heap allocations -- they write pre-built byte slices directly to an
// [io.Writer].
//
// # Usage
//
//	errorStyle := colour.New(colour.FgRed, colour.Bold)
//
//	// Zero-alloc path (for hot loops like log formatting):
//	errorStyle.WriteStart(writer)
//	writer.WriteString("something went wrong")
//	errorStyle.WriteReset(writer)
//
//	// Convenience path (allocates one string):
//	message := errorStyle.Sprintf("found %d errors", count)
//
// # Environment variables
//
// Colour output is auto-detected from the terminal. The following environment
// variables override detection:
//
//   - NO_COLOUR or NO_COLOR: disables colour output when set (any value).
//   - FORCE_COLOUR or FORCE_COLOR: forces colour output when set (any value).
//   - TERM=dumb: disables colour output.
//
// # Thread safety
//
// [Enabled] and [SetEnabled] are safe for concurrent use. [Style] values are
// immutable after construction and safe to share across goroutines.
package colour
