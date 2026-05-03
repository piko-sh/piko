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

// Package mem wraps unsafe operations to provide zero-allocation memory
// utilities.
//
// These functions avoid allocation by creating values that share underlying
// memory, or provide direct access to memory layout information. Callers must
// adhere to the safety contracts documented on each function.
//
// # When to use
//
// Use these helpers when profiling shows memory operations are a bottleneck.
// Prefer safe alternatives by default:
//
//   - string(b) instead of mem.String(b)
//   - []byte(s) instead of mem.Bytes(s)
//
// # Safety
//
// Functions here use [unsafe] operations. Misuse can cause:
//
//   - Data corruption (modifying immutable strings)
//   - Use-after-free (if backing memory is freed/reused)
//   - Undefined behaviour
//
// Each function documents its specific safety requirements.
//
// # Safe mode
//
// Build with the "safe" tag to use allocating implementations that
// avoid unsafe operations. This is useful for debugging memory issues
// or running in environments that prohibit unsafe code.
package mem
