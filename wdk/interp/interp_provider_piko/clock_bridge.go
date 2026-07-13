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

package interp_provider_piko

import (
	"time"

	"piko.sh/piko/wdk/clock"
)

// FromWDKClock adapts a wdk/clock.Clock so it implements the interpreter-side Clock
// interface used by interp_provider_piko.WithClock.
//
// A nil source returns nil; callers should treat that as "use the interpreter default
// wall clock."
//
// Takes source (clock.Clock) which is the wdk clock to adapt.
//
// Returns Clock (the interpreter's Clock interface) backed by source.
func FromWDKClock(source clock.Clock) Clock {
	if source == nil {
		return nil
	}
	return &clockBridge{source: source}
}

// clockBridge wraps a wdk/clock.Clock to expose the interpreter Clock surface.
//
// It is safe for concurrent use; method calls forward directly to the underlying
// wdk/clock.Clock, which is documented as concurrent safe.
type clockBridge struct {
	// source is the underlying wdk clock supplying time values.
	source clock.Clock
}

// Now returns the wdk clock's current time.
//
// Returns time.Time which is the current time from the wdk source.
func (b *clockBridge) Now() time.Time {
	return b.source.Now()
}

// Since returns time elapsed since t, derived from Now().
//
// Takes t (time.Time) which is the reference point.
//
// Returns time.Duration which is the elapsed time from t to now.
func (b *clockBridge) Since(t time.Time) time.Duration {
	return b.source.Now().Sub(t)
}

// Until returns time until t, derived from Now().
//
// Takes t (time.Time) which is the target time.
//
// Returns time.Duration which is the duration from now until t.
func (b *clockBridge) Until(t time.Time) time.Duration {
	return t.Sub(b.source.Now())
}

// Sleep blocks for at least d, using the wdk clock's timer machinery.
//
// MockClock-based tests can advance through the sleep synchronously.
//
// Takes d (time.Duration) which is the minimum duration to block.
func (b *clockBridge) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	timer := b.source.NewTimer(d)
	defer timer.Stop()
	<-timer.C()
}

// NewTimer returns a real stdlib *time.Timer.
//
// The timer fires on wall-clock time, not on the wdk clock's mock schedule.
//
// Takes d (time.Duration) which is the timer duration.
//
// Returns *time.Timer which is the stdlib timer scheduled for d.
func (*clockBridge) NewTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}

// NewTicker returns a real stdlib *time.Ticker.
//
// Takes d (time.Duration) which is the ticker period.
//
// Returns *time.Ticker which is the stdlib ticker firing every d.
func (*clockBridge) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
