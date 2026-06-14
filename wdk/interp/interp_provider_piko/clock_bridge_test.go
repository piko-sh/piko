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
	"testing"
	"time"

	"piko.sh/piko/wdk/clock"
)

func TestFromWDKClockNilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := FromWDKClock(nil); got != nil {
		t.Fatalf("FromWDKClock(nil) = %v, want nil", got)
	}
}

func TestFromWDKClockNowDelegates(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	mock := clock.NewMockClock(fixed)
	bridge := FromWDKClock(mock)
	if got := bridge.Now(); !got.Equal(fixed) {
		t.Fatalf("Now() = %v, want %v", got, fixed)
	}
}

func TestFromWDKClockSinceDerivedFromNow(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	mock := clock.NewMockClock(fixed)
	bridge := FromWDKClock(mock)

	earlier := fixed.Add(-5 * time.Minute)
	if got := bridge.Since(earlier); got != 5*time.Minute {
		t.Fatalf("Since(earlier) = %v, want 5m", got)
	}
}

func TestFromWDKClockUntilDerivedFromNow(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	mock := clock.NewMockClock(fixed)
	bridge := FromWDKClock(mock)

	later := fixed.Add(3 * time.Minute)
	if got := bridge.Until(later); got != 3*time.Minute {
		t.Fatalf("Until(later) = %v, want 3m", got)
	}
}

func TestFromWDKClockSleepUsesWDKTimer(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	mock := clock.NewMockClock(fixed)
	bridge := FromWDKClock(mock)

	baseline := mock.TimerCount()
	done := make(chan struct{})
	go func() {
		bridge.Sleep(10 * time.Minute)
		close(done)
	}()

	if !mock.AwaitTimerSetup(baseline, 2*time.Second) {
		t.Fatalf("Sleep did not register a wdk timer")
	}
	mock.Advance(10 * time.Minute)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Sleep did not return after Advance")
	}
}

func TestFromWDKClockSleepZeroIsNoop(t *testing.T) {
	t.Parallel()
	mock := clock.NewMockClock(time.Now())
	bridge := FromWDKClock(mock)
	bridge.Sleep(0)
	bridge.Sleep(-1 * time.Second)
}
