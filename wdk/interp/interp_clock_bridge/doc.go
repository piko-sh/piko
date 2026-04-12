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

// Package interp_clock_bridge adapts wdk/clock.Clock instances to the
// interp_provider_piko.Clock interface used by the interpreter.
//
// # Why a bridge?
//
// piko has two clock abstractions because they serve different needs:
//
//   - wdk/clock.Clock is used by piko's *internal services* (caches, handlers). Its
//     NewTimer / NewTicker return wrapper interfaces so MockClock can synchronously
//     advance them in tests.
//   - interp_provider_piko.Clock (re-exported from internal/interp/interp_domain.Clock)
//     is used by *interpreted Go code*. Its NewTimer / NewTicker must return stdlib
//     *time.Timer / *time.Ticker because interpreted code expects those exact types via
//     reflect.
//
// The two interfaces cannot share a single shape without breaking one or the other. The
// package is the recommended adapter for hosts that already use wdk/clock and want to
// plug the same clock into the interpreter.
//
// # Coverage and limitations
//
// The bridge fully virtualises Now, Since, Until, and Sleep - the methods that matter for
// deterministic replay and audit. Sleep is implemented via wdk/clock's NewTimer so it
// advances with mock clocks under test.
//
// NewTimer and NewTicker are PARTIALLY virtualised: they return real stdlib timers, but
// the time those timers observe via Now() goes through the wdk clock. This is appropriate
// for replay and most test scenarios - what changes is the value a script reads from
// time.Now while the timer is pending, not whether the timer fires on virtual time. For
// tests that need fully-synchronous timer firing through interpreted code, implement
// Clock directly.
//
// # Thread safety
//
// The bridge is safe for concurrent use; method calls forward directly to the underlying
// wdk/clock.Clock, which is documented as concurrent safe.
package interp_clock_bridge
