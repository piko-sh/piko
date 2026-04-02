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

// Package provider_multilevel implements a two-tier cache provider that
// orchestrates an L1 (fast, local) and L2 (slower, distributed) cache.
//
// The adapter presents a single cache provider interface whilst transparently
// managing cache lookups across both levels. On a read, L1 is checked first;
// on an L1 miss the adapter falls through to L2 and automatically
// back-populates L1 with the result. Writes use a write-through policy,
// storing values in both levels.
//
// # Resilience
//
// All L2 operations are guarded by a circuit breaker. When consecutive L2
// failures exceed a configurable threshold, the circuit opens and L2 calls
// are short-circuited so that the system degrades gracefully to L1-only
// operation. The circuit breaker automatically re-tests L2 after a
// configurable timeout.
//
// # Thread safety
//
// All exported methods are safe for concurrent use. L2 write-back and bulk
// store operations are performed asynchronously via goroutines, so they may
// complete after the calling method returns.
package provider_multilevel
