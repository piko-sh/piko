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

import (
	"reflect"
	"time"
)

// Clock is the time source the interpreter consults for time.Now, time.Since, time.Sleep,
// and timer/ticker construction.
//
// Hosts install a Clock when they need deterministic replay (recording real timestamps
// during one run and replaying them during forensic analysis), test-controllable time
// (advancing a virtual clock from test code without sleeping), or audited time access
// (logging every wall-clock read).
//
// The default Clock is WallClock; interpreted code observes the host process's real time
// when no override is installed. Hosts that load untrusted code should install a Clock
// when virtualised time is part of their replay or audit story.
//
// Implementations MUST be safe for concurrent use; the interpreter invokes Clock methods
// from any goroutine that interpreted code has spawned.
type Clock interface {
	// Now returns the current time.
	//
	// Returns time.Time which is the active clock's current instant.
	Now() time.Time

	// Since returns the time elapsed since t.
	//
	// Equivalent to Now().Sub(t) but allowed to be implemented directly for efficiency or
	// virtualisation.
	//
	// Takes t (time.Time) which is the reference instant.
	//
	// Returns time.Duration which is the elapsed time since t.
	Since(t time.Time) time.Duration

	// Until returns t.Sub(Now()).
	//
	// Takes t (time.Time) which is the target instant.
	//
	// Returns time.Duration which is the remaining time until t.
	Until(t time.Time) time.Duration

	// Sleep blocks for at least duration d.
	//
	// Implementations may return early when the host context is cancelled; interpreted code
	// observes this through normal cancellation paths.
	//
	// Takes d (time.Duration) which is the requested sleep duration.
	Sleep(d time.Duration)

	// NewTimer creates a time.Timer that fires after d.
	//
	// Virtual clocks may return a timer backed by host-controlled signalling rather than
	// real wall-clock time.
	//
	// Takes d (time.Duration) which is the fire delay.
	//
	// Returns *time.Timer which is the configured timer.
	NewTimer(d time.Duration) *time.Timer

	// NewTicker creates a time.Ticker that fires every d.
	//
	// d must be greater than zero or the implementation panics, matching time.NewTicker.
	//
	// Takes d (time.Duration) which is the tick interval.
	//
	// Returns *time.Ticker which is the configured ticker.
	NewTicker(d time.Duration) *time.Ticker
}

var (
	// WallClock is the default Clock implementation.
	//
	// Every method delegates directly to the stdlib [time] package. Allocated once per
	// process; safe for concurrent use.
	WallClock Clock = wallClock{}
)

// wallClock implements Clock via stdlib time package.
type wallClock struct{}

// Now delegates to time.Now.
//
// Returns time.Time which is the host wall-clock instant.
func (wallClock) Now() time.Time { return time.Now() }

// Since delegates to time.Since.
//
// Takes t (time.Time) which is the reference instant.
//
// Returns time.Duration which is the elapsed wall-clock time since t.
func (wallClock) Since(t time.Time) time.Duration { return time.Since(t) }

// Until delegates to time.Until.
//
// Takes t (time.Time) which is the target instant.
//
// Returns time.Duration which is the remaining wall-clock time until t.
func (wallClock) Until(t time.Time) time.Duration { return time.Until(t) }

// Sleep delegates to time.Sleep.
//
// Takes d (time.Duration) which is the requested sleep duration.
func (wallClock) Sleep(d time.Duration) { time.Sleep(d) }

// NewTimer delegates to time.NewTimer.
//
// Takes d (time.Duration) which is the fire delay.
//
// Returns *time.Timer which is the configured wall-clock timer.
func (wallClock) NewTimer(d time.Duration) *time.Timer { return time.NewTimer(d) }

// NewTicker delegates to time.NewTicker.
//
// Takes d (time.Duration) which is the tick interval.
//
// Returns *time.Ticker which is the configured wall-clock ticker.
func (wallClock) NewTicker(d time.Duration) *time.Ticker { return time.NewTicker(d) }

// effectiveClock returns the supplied clock when non-nil, otherwise WallClock. Used by
// Service initialisation so the rest of the codebase can assume a non-nil clock.
//
// Takes clock (Clock) which may be nil.
//
// Returns a non-nil Clock.
func effectiveClock(clock Clock) Clock {
	if clock == nil {
		return WallClock
	}
	return clock
}

// clockOverrideSymbols returns the (Now, Since, Until, Sleep, NewTimer, NewTicker)
// entries that should replace the stdlib bindings in the "time" package symbol map for a
// Service running with the supplied clock.
//
// Hosts that install a non-default clock get these wrapped entries merged into the
// service's symbol registry so interpreted code observes the virtualised time. The
// default WallClock has identical behaviour to direct stdlib calls; the override is a
// no-op for it.
//
// Takes clock (Clock) which is the active clock.
//
// Returns a map[string]reflect.Value containing the six entries.
func clockOverrideSymbols(clock Clock) map[string]reflect.Value {
	active := effectiveClock(clock)
	return map[string]reflect.Value{
		"Now":       reflect.ValueOf(active.Now),
		"Since":     reflect.ValueOf(active.Since),
		"Until":     reflect.ValueOf(active.Until),
		"Sleep":     reflect.ValueOf(active.Sleep),
		"NewTimer":  reflect.ValueOf(active.NewTimer),
		"NewTicker": reflect.ValueOf(active.NewTicker),
	}
}
