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

// Package safeconv provides safe integer type conversions with bounds checking.
//
// These functions convert between integer types by clamping values to the
// target type's valid range rather than allowing overflow. This prevents gosec
// G115 (integer overflow) issues and provides predictable behaviour when
// narrowing integer types.
//
// # Design rationale
//
// Clamping (saturating at the target type's minimum or maximum) was chosen
// over truncation or panicking. Truncation silently discards high bits,
// which can turn a large positive value into a small or even negative one
// with no indication that anything went wrong. Panicking would crash at
// runtime, which is disproportionate for application code where overflow
// typically indicates unexpected input rather than a logic error. Clamping
// preserves the closest representable value and keeps the program running.
//
// # Conversion behaviour
//
// All functions follow a consistent clamping pattern:
//
//   - Negative values are clamped to 0 for unsigned targets
//   - Values exceeding the target's maximum are clamped to that maximum
//   - Values below the target's minimum (for signed types) are clamped to
//     that minimum
//
// # Usage
//
//	// Convert potentially large int to uint32 safely
//	size := safeconv.IntToUint32(someInt)
//
//	// Generic conversion for cross-platform code
//	bytes := safeconv.ToUint64(statfs.Bsize)
//
// # Thread safety
//
// All functions are pure and stateless; they are safe for concurrent use.
package safeconv
