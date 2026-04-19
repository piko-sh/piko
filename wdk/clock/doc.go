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

// Package clock provides a time abstraction for testability.
//
// Enables dependency injection of time operations, allowing production code to
// use the real system clock whilst tests can control time deterministically.
// Essential for testing time-sensitive logic such as caching, expiration, and
// scheduling.
//
// # Production usage
//
// Use [RealClock] to obtain a Clock that delegates to the standard library:
//
//	clock := clock.RealClock()
//	expiresAt := clock.Now().Add(24 * time.Hour)
//
// # Test usage
//
// Use [NewMockClock] to create a clock with controllable time:
//
//	mock := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
//	service := NewService(WithClock(mock))
//
//	mock.Advance(25 * time.Hour)
//	// Now service.CheckExpiration() sees an expired token
//
// The mock clock fires timers and tickers when [MockClock.Advance] is called,
// so tests can exercise time-dependent behaviour deterministically.
//
// # Thread safety
//
// All Clock implementations are safe for concurrent use. The [MockClock] type
// uses internal synchronisation to protect its state during Advance and timer
// operations.
package clock
