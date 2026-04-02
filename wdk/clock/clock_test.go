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

package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealClock(t *testing.T) {
	clock := RealClock()

	before := time.Now()
	clockTime := clock.Now()
	after := time.Now()

	assert.True(t, clockTime.After(before) || clockTime.Equal(before))
	assert.True(t, clockTime.Before(after) || clockTime.Equal(after))
}

func TestMockClock_Now(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewMockClock(startTime)

	assert.Equal(t, startTime, clock.Now())
	assert.Equal(t, startTime, clock.Now(), "should return same time on multiple calls")
}

func TestMockClock_Set(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	newTime := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	clock.Set(newTime)

	assert.Equal(t, newTime, clock.Now())
}

func TestMockClock_Advance(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(startTime)

	clock.Advance(24 * time.Hour)
	expected := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())

	clock.Advance(7 * 24 * time.Hour)
	expected = time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())
}

func TestMockClock_Rewind(t *testing.T) {
	startTime := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(startTime)

	clock.Rewind(24 * time.Hour)
	expected := time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())
}

func TestMockClock_Freeze(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(startTime)

	frozen := clock.Freeze()
	assert.Equal(t, startTime, frozen)

	clock.Advance(1 * time.Hour)
	assert.NotEqual(t, frozen, clock.Now(), "frozen time should not change")
	assert.Equal(t, startTime, frozen, "frozen time should remain original")
}

func TestMockClock_ZeroTime(t *testing.T) {
	clock := NewMockClock(time.Time{})

	assert.Equal(t, time.Unix(0, 0).UTC(), clock.Now())
}

func TestMockClock_ThreadSafety(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	done := make(chan bool)

	for range 10 {
		go func() {
			for range 100 {
				clock.Advance(1 * time.Second)
				_ = clock.Now()
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	expected := time.Date(2025, 1, 1, 0, 16, 40, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())
}

func TestRealClock_AfterFunc(t *testing.T) {
	clock := RealClock()

	fired := make(chan bool, 1)
	timer := clock.AfterFunc(10*time.Millisecond, func() {
		fired <- true
	})

	select {
	case <-fired:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timer did not fire within expected time")
	}

	stopped := timer.Stop()
	assert.False(t, stopped)
}

func TestRealClock_AfterFunc_Stop(t *testing.T) {
	clock := RealClock()

	fired := make(chan bool, 1)
	timer := clock.AfterFunc(100*time.Millisecond, func() {
		fired <- true
	})

	stopped := timer.Stop()
	assert.True(t, stopped, "should successfully stop unfired timer")

	select {
	case <-fired:
		t.Fatal("timer fired after being stopped")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestMockClock_AfterFunc_Fires(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	fired := false
	clock.AfterFunc(10*time.Second, func() {
		fired = true
	})

	assert.False(t, fired)

	clock.Advance(5 * time.Second)
	assert.False(t, fired, "timer should not fire after 5 seconds")

	clock.Advance(5 * time.Second)
	assert.True(t, fired, "timer should fire after 10 seconds")
}

func TestMockClock_AfterFunc_Stop(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	fired := false
	timer := clock.AfterFunc(10*time.Second, func() {
		fired = true
	})

	stopped := timer.Stop()
	assert.True(t, stopped, "should successfully stop timer")

	clock.Advance(15 * time.Second)
	assert.False(t, fired, "timer should not fire after being stopped")

	stoppedAgain := timer.Stop()
	assert.False(t, stoppedAgain, "stopping already-stopped timer should return false")
}

func TestMockClock_AfterFunc_MultipleTimers(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	var fired []int
	clock.AfterFunc(5*time.Second, func() {
		fired = append(fired, 1)
	})
	clock.AfterFunc(10*time.Second, func() {
		fired = append(fired, 2)
	})
	clock.AfterFunc(15*time.Second, func() {
		fired = append(fired, 3)
	})

	clock.Advance(7 * time.Second)
	assert.Equal(t, []int{1}, fired)

	clock.Advance(5 * time.Second)
	assert.Equal(t, []int{1, 2}, fired)

	clock.Advance(5 * time.Second)
	assert.Equal(t, []int{1, 2, 3}, fired)
}

func TestMockClock_AfterFunc_StopBeforeFire(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	var fired []int
	timer1 := clock.AfterFunc(5*time.Second, func() {
		fired = append(fired, 1)
	})
	timer2 := clock.AfterFunc(10*time.Second, func() {
		fired = append(fired, 2)
	})
	clock.AfterFunc(15*time.Second, func() {
		fired = append(fired, 3)
	})

	timer2.Stop()

	clock.Advance(20 * time.Second)
	assert.Equal(t, []int{1, 3}, fired, "stopped timer should not fire")

	stopped := timer1.Stop()
	assert.False(t, stopped, "cannot stop already-fired timer")
}

func TestMockClock_AfterFunc_ZeroDuration(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	fired := false
	clock.AfterFunc(0*time.Second, func() {
		fired = true
	})

	clock.Advance(1 * time.Nanosecond)
	assert.True(t, fired, "zero-duration timer should fire immediately on any advance")
}

func TestMockClock_AfterFunc_OrderOfExecution(t *testing.T) {
	clock := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	var order []string

	clock.AfterFunc(10*time.Second, func() {
		order = append(order, "a")
	})
	clock.AfterFunc(10*time.Second, func() {
		order = append(order, "b")
	})
	clock.AfterFunc(10*time.Second, func() {
		order = append(order, "c")
	})

	clock.Advance(10 * time.Second)

	assert.Len(t, order, 3, "all three timers should fire")
	assert.Contains(t, order, "a")
	assert.Contains(t, order, "b")
	assert.Contains(t, order, "c")
}

func TestRealClock_NewTimer(t *testing.T) {
	clk := RealClock()

	timer := clk.NewTimer(10 * time.Millisecond)
	defer timer.Stop()

	select {
	case received := <-timer.C():
		assert.False(t, received.IsZero(), "received time should not be zero")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timer did not fire within expected time")
	}
}

func TestRealClock_NewTimer_Stop(t *testing.T) {
	clk := RealClock()

	timer := clk.NewTimer(100 * time.Millisecond)
	stopped := timer.Stop()
	assert.True(t, stopped, "should successfully stop unfired timer")

	select {
	case <-timer.C():
		t.Fatal("timer fired after being stopped")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRealClock_NewTimer_Reset(t *testing.T) {
	clk := RealClock()

	timer := clk.NewTimer(500 * time.Millisecond)
	timer.Reset(10 * time.Millisecond)

	select {
	case received := <-timer.C():
		assert.False(t, received.IsZero())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("reset timer did not fire within expected time")
	}
}

func TestRealClock_NewTicker(t *testing.T) {
	clk := RealClock()

	ticker := clk.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for range 2 {
		select {
		case tick := <-ticker.C():
			assert.False(t, tick.IsZero())
		case <-time.After(200 * time.Millisecond):
			t.Fatal("ticker did not deliver tick within expected time")
		}
	}
}

func TestRealClock_NewTicker_Stop(t *testing.T) {
	clk := RealClock()

	ticker := clk.NewTicker(10 * time.Millisecond)

	select {
	case <-ticker.C():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("ticker did not deliver first tick")
	}

	ticker.Stop()

	select {
	case <-ticker.C():

	case <-time.After(100 * time.Millisecond):
	}
}

func TestMockClock_NewTimer_Fires(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)

	clk.Advance(5 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("timer should not fire after only 5 seconds")
	default:
	}

	clk.Advance(5 * time.Second)
	select {
	case received := <-timer.C():
		expected := time.Date(2025, 1, 1, 12, 0, 10, 0, time.UTC)
		assert.Equal(t, expected, received)
	default:
		t.Fatal("timer should have fired after 10 seconds")
	}
}

func TestMockClock_NewTimer_Stop(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)

	stopped := timer.Stop()
	assert.True(t, stopped, "should successfully stop unfired timer")

	clk.Advance(15 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("stopped timer should not fire")
	default:
	}

	stoppedAgain := timer.Stop()
	assert.False(t, stoppedAgain, "stopping already-stopped timer should return false")
}

func TestMockClock_NewTimer_Reset(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)

	wasActive := timer.Reset(5 * time.Second)
	assert.True(t, wasActive, "timer was active before reset")

	clk.Advance(5 * time.Second)
	select {
	case received := <-timer.C():
		expected := time.Date(2025, 1, 1, 12, 0, 5, 0, time.UTC)
		assert.Equal(t, expected, received)
	default:
		t.Fatal("reset timer should have fired after new duration")
	}
}

func TestMockClock_NewTimer_ResetAfterStop(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)
	timer.Stop()

	wasActive := timer.Reset(5 * time.Second)
	assert.False(t, wasActive, "timer was stopped so Reset should return false")

	clk.Advance(5 * time.Second)
	select {
	case received := <-timer.C():
		expected := time.Date(2025, 1, 1, 12, 0, 5, 0, time.UTC)
		assert.Equal(t, expected, received)
	default:
		t.Fatal("timer should fire after reset following stop")
	}
}

func TestMockClock_NewTimer_MultipleTimers(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer1 := clk.NewTimer(5 * time.Second)
	timer2 := clk.NewTimer(10 * time.Second)
	timer3 := clk.NewTimer(15 * time.Second)

	clk.Advance(7 * time.Second)

	select {
	case <-timer1.C():
	default:
		t.Fatal("timer1 should have fired at 5s")
	}
	select {
	case <-timer2.C():
		t.Fatal("timer2 should not fire at 7s")
	default:
	}

	clk.Advance(5 * time.Second)

	select {
	case <-timer2.C():
	default:
		t.Fatal("timer2 should have fired at 12s")
	}
	select {
	case <-timer3.C():
		t.Fatal("timer3 should not fire at 12s")
	default:
	}

	clk.Advance(5 * time.Second)

	select {
	case <-timer3.C():
	default:
		t.Fatal("timer3 should have fired at 17s")
	}
}

func TestMockClock_NewTimer_StopAfterFire(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(5 * time.Second)
	clk.Advance(5 * time.Second)

	select {
	case <-timer.C():
	default:
		t.Fatal("timer should have fired")
	}

	stopped := timer.Stop()
	assert.False(t, stopped, "stopping already-fired timer should return false")
}

func TestMockClock_NewTicker_SingleTick(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)
	defer ticker.Stop()

	clk.Advance(10 * time.Second)

	select {
	case tick := <-ticker.C():
		expected := time.Date(2025, 1, 1, 12, 0, 10, 0, time.UTC)
		assert.Equal(t, expected, tick)
	default:
		t.Fatal("ticker should have fired after one period")
	}
}

func TestMockClock_NewTicker_MultipleTicks(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for i := range 3 {
		clk.Advance(10 * time.Second)

		select {
		case tick := <-ticker.C():
			expectedSeconds := (i + 1) * 10
			expected := time.Date(2025, 1, 1, 12, 0, expectedSeconds, 0, time.UTC)
			assert.Equal(t, expected, tick)
		default:
			t.Fatalf("ticker should have fired on advance %d", i+1)
		}
	}
}

func TestMockClock_NewTicker_Stop(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)

	clk.Advance(10 * time.Second)

	select {
	case <-ticker.C():
	default:
		t.Fatal("ticker should have fired first tick")
	}

	ticker.Stop()

	clk.Advance(10 * time.Second)
	select {
	case <-ticker.C():
		t.Fatal("stopped ticker should not fire")
	default:
	}
}

func TestMockClock_NewTicker_SkippedTicks(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)
	defer ticker.Stop()

	clk.Advance(25 * time.Second)

	select {
	case tick := <-ticker.C():
		assert.False(t, tick.IsZero(), "should receive at least one tick")
	default:
		t.Fatal("should have received at least one tick")
	}
}

func TestMockClock_NewTicker_StoppedTickerSkipped(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)
	ticker.Stop()

	clk.Advance(20 * time.Second)

	select {
	case <-ticker.C():
		t.Fatal("stopped ticker should not fire")
	default:
	}
}

func TestMockClock_NewTicker_MultipleTickers(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	tickerFast := clk.NewTicker(5 * time.Second)
	tickerSlow := clk.NewTicker(10 * time.Second)
	defer tickerFast.Stop()
	defer tickerSlow.Stop()

	clk.Advance(10 * time.Second)

	select {
	case <-tickerFast.C():
	default:
		t.Fatal("fast ticker should have fired")
	}

	select {
	case <-tickerSlow.C():
	default:
		t.Fatal("slow ticker should have fired")
	}
}

func TestMockClock_NewTicker_C(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	ticker := clk.NewTicker(10 * time.Second)
	defer ticker.Stop()

	tickerChannel := ticker.C()
	require.NotNil(t, tickerChannel, "ticker channel should not be nil")
}

func TestMockClock_NewTimer_C(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)
	defer timer.Stop()

	timerChannel := timer.C()
	require.NotNil(t, timerChannel, "timer channel should not be nil")
}

func TestMockClock_TimerCount_IncrementsOnNewTimer(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	assert.Equal(t, int64(0), clk.TimerCount())

	clk.NewTimer(time.Second)
	assert.Equal(t, int64(1), clk.TimerCount())

	clk.AfterFunc(time.Second, func() {})
	assert.Equal(t, int64(2), clk.TimerCount())

	clk.NewTicker(time.Second)
	assert.Equal(t, int64(3), clk.TimerCount())
}

func TestMockClock_TimerCount_IncrementsOnReset(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	timer := clk.NewTimer(10 * time.Second)
	assert.Equal(t, int64(1), clk.TimerCount())

	timer.Reset(5 * time.Second)
	assert.Equal(t, int64(2), clk.TimerCount())
}

func TestMockClock_AwaitTimerSetup_ReturnsImmediatelyWhenSatisfied(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	clk.NewTimer(time.Second)

	ok := clk.AwaitTimerSetup(0, time.Second)
	assert.True(t, ok, "should return immediately when counter already exceeds baseline")
}

func TestMockClock_AwaitTimerSetup_BlocksUntilEvent(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	baseline := clk.TimerCount()

	done := make(chan bool, 1)
	go func() {
		done <- clk.AwaitTimerSetup(baseline, time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	clk.NewTimer(time.Second)

	select {
	case ok := <-done:
		assert.True(t, ok, "should have observed the timer setup")
	case <-time.After(time.Second):
		t.Fatal("AwaitTimerSetup did not return in time")
	}
}

func TestMockClock_AwaitTimerSetup_TimesOut(t *testing.T) {
	clk := NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	baseline := clk.TimerCount()

	ok := clk.AwaitTimerSetup(baseline, 20*time.Millisecond)
	assert.False(t, ok, "should return false on timeout when no timer setup occurs")
}
