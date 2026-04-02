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
	"sync"
	"sync/atomic"
	"time"
)

// Clock provides time operations that can be replaced for testing.
// It implements clock.Clock, cache.Clock, and cache_dto.Clock interfaces.
type Clock interface {
	// Now returns the current time.
	//
	// Returns time.Time which is the current time. In production this uses
	// time.Now(), whilst in tests it can return a fixed value.
	Now() time.Time

	// AfterFunc waits for the duration to elapse and then calls f in its own
	// goroutine.
	//
	// Takes d (time.Duration) which specifies how long to wait before calling f.
	// Takes f (func()) which is the function to call after the duration elapses.
	//
	// Returns Timer which can be used to cancel the call using its Stop method.
	//
	// In production, this delegates to time.AfterFunc. In tests, the timer fires
	// when the mock clock is advanced past the deadline.
	AfterFunc(d time.Duration, f func()) Timer

	// NewTimer creates a new Timer that will send the current time on its
	// channel after at least the specified duration.
	//
	// Takes d (time.Duration) which specifies the delay before the timer fires.
	//
	// Returns ChannelTimer which provides access to the timer's channel.
	//
	// In production, this delegates to time.NewTimer. In tests, the timer fires
	// when the mock clock is advanced past the deadline.
	NewTimer(d time.Duration) ChannelTimer

	// NewTicker returns a new Ticker containing a channel that sends the current
	// time after each tick.
	//
	// Takes d (time.Duration) which specifies the period between ticks.
	//
	// Returns Ticker which provides a channel for receiving tick events.
	//
	// In production, this delegates to time.NewTicker. In tests, the ticker fires
	// when the mock clock is advanced past each tick period.
	NewTicker(d time.Duration) Ticker
}

// Timer provides a wrapper around time.Timer for testing purposes. It
// implements clock.Timer with the methods needed to control the timer
// lifecycle.
type Timer interface {
	// Stop prevents the timer from firing.
	//
	// Returns bool which is true if this call stops the timer, or false if the
	// timer has already fired or been stopped.
	Stop() bool
}

// ChannelTimer provides a timer that sends its expiry time on a channel.
// It extends Timer with Reset for reuse and C for receiving the fire event.
type ChannelTimer interface {
	Timer

	// C returns the channel that delivers the time.
	C() <-chan time.Time

	// Reset changes the timer to expire after duration d.
	//
	// Takes d (time.Duration) which specifies when the timer should expire.
	//
	// Returns bool which is true if the timer was active, or false if it had
	// already expired or been stopped.
	Reset(d time.Duration) bool
}

// Ticker provides a channel that sends time values at regular intervals.
// It implements the clock.Ticker interface.
type Ticker interface {
	// C returns the channel that receives tick events.
	C() <-chan time.Time

	// Stop disables the ticker. After Stop, no more ticks will be sent.
	Stop()
}

// realClock provides the production implementation of the Clock interface
// using the system clock.
type realClock struct{}

// Now returns the current system time.
//
// Returns time.Time which is the current local time.
func (realClock) Now() time.Time {
	return time.Now()
}

// AfterFunc delegates to time.AfterFunc and returns a Timer wrapper.
//
// Takes d (time.Duration) which specifies how long to wait before calling f.
// Takes f (func()) which is the function to call after the duration elapses.
//
// Returns Timer which wraps the underlying time.Timer for cancellation.
func (realClock) AfterFunc(d time.Duration, f func()) Timer {
	return &realTimer{timer: time.AfterFunc(d, f)}
}

// NewTimer creates a new Timer that delegates to time.NewTimer.
//
// Takes d (time.Duration) which specifies how long until the timer fires.
//
// Returns ChannelTimer which wraps the standard library timer.
func (realClock) NewTimer(d time.Duration) ChannelTimer {
	return &realChannelTimer{timer: time.NewTimer(d)}
}

// NewTicker creates a new Ticker that delegates to time.NewTicker.
//
// Takes d (time.Duration) which specifies the tick interval.
//
// Returns Ticker which wraps the standard library ticker.
func (realClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(d)}
}

// realTimer wraps time.Timer to implement the Timer interface.
type realTimer struct {
	// timer is the underlying standard library timer.
	timer *time.Timer
}

// Stop prevents the timer from firing.
//
// Returns bool which is true if the timer was active and is now stopped.
func (t *realTimer) Stop() bool {
	return t.timer.Stop()
}

// realChannelTimer wraps time.Timer to implement the ChannelTimer interface.
type realChannelTimer struct {
	// timer is the underlying standard library timer.
	timer *time.Timer
}

// Stop prevents the timer from firing.
//
// Returns bool which is true if the timer was stopped before it fired.
func (t *realChannelTimer) Stop() bool {
	return t.timer.Stop()
}

// C returns the channel on which the time is delivered.
//
// Returns <-chan time.Time which receives the current time when the timer
// fires.
func (t *realChannelTimer) C() <-chan time.Time {
	return t.timer.C
}

// Reset changes the timer to expire after duration d.
//
// Takes d (time.Duration) which specifies when the timer should expire.
//
// Returns bool which is true if the timer was active before the reset.
func (t *realChannelTimer) Reset(d time.Duration) bool {
	return t.timer.Reset(d)
}

// realTicker wraps time.Ticker to implement the Ticker interface.
type realTicker struct {
	// ticker is the wrapped standard library ticker.
	ticker *time.Ticker
}

// C returns the channel on which the ticks are delivered.
//
// Returns <-chan time.Time which receives the tick events.
func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

// Stop turns off the ticker.
func (t *realTicker) Stop() {
	t.ticker.Stop()
}

// MockClock implements Clock for testing, allowing controlled time advancement.
// It is thread-safe; advancing time causes scheduled timers and tickers to fire.
type MockClock struct {
	// currentTime holds the simulated time returned by Now.
	currentTime time.Time

	// timerSetupSignal is closed and replaced on each timer setup event,
	// acting as a broadcast wakeup for AwaitTimerSetup callers.
	timerSetupSignal chan struct{}

	// timers holds callback timers set up via AfterFunc.
	timers []*mockTimer

	// channelTimers holds timers created by NewTimer that fire when Advance is called.
	channelTimers []*mockChannelTimer

	// tickers holds all tickers created by this mock clock.
	tickers []*mockTicker

	// timerSetupCount is incremented each time a timer, channel timer, or
	// ticker is created or reset. Tests use this to synchronise with
	// goroutines that set up timers after an Advance fires their previous one.
	timerSetupCount atomic.Int64

	// mu protects the mock clock's mutable state from concurrent access.
	mu sync.RWMutex

	// signalMu protects replacement of timerSetupSignal. Separate from mu
	// to avoid deadlock (notifyTimerSetup is called after mu is released).
	signalMu sync.Mutex
}

// NewMockClock creates a new mock clock starting at the specified time.
// If zero time is provided, defaults to Unix epoch.
//
// Takes startTime (time.Time) which specifies the initial time for the clock.
//
// Returns *MockClock which is ready for use in tests.
func NewMockClock(startTime time.Time) *MockClock {
	if startTime.IsZero() {
		startTime = time.Unix(0, 0).UTC()
	}
	return &MockClock{
		currentTime:      startTime,
		timers:           []*mockTimer{},
		channelTimers:    []*mockChannelTimer{},
		tickers:          []*mockTicker{},
		timerSetupSignal: make(chan struct{}),
	}
}

// Now returns the current mocked time.
//
// Returns time.Time which is the current time set on this mock
// clock.
//
// Safe for concurrent use; protected by a read lock.
func (m *MockClock) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTime
}

// AfterFunc schedules a function to be called after the
// specified duration. The function will be called when Advance
// moves the clock past the scheduled time.
//
// Takes d (time.Duration) which specifies how long to wait
// before calling f.
// Takes f (func()) which is the function to call when the timer
// fires.
//
// Returns Timer which can be used to cancel the scheduled call.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockClock) AfterFunc(d time.Duration, f func()) Timer {
	m.mu.Lock()
	timer := &mockTimer{
		fireAt:   m.currentTime.Add(d),
		callback: f,
		clock:    m,
		stopped:  false,
	}
	m.timers = append(m.timers, timer)
	m.mu.Unlock()

	m.notifyTimerSetup()
	return timer
}

// NewTimer creates a new channel-based Timer that fires when
// the clock advances.
//
// Takes d (time.Duration) which specifies when the timer should
// fire.
//
// Returns ChannelTimer which can be used to receive the fire
// event or stop the timer.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockClock) NewTimer(d time.Duration) ChannelTimer {
	m.mu.Lock()
	timer := &mockChannelTimer{
		fireAt:  m.currentTime.Add(d),
		c:       make(chan time.Time, 1),
		clock:   m,
		stopped: false,
	}
	m.channelTimers = append(m.channelTimers, timer)
	m.mu.Unlock()

	m.notifyTimerSetup()
	return timer
}

// NewTicker creates a new Ticker that fires periodically when
// the clock advances.
//
// Takes d (time.Duration) which specifies the interval between
// tick events.
//
// Returns Ticker which can be used to receive tick events on
// its channel.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockClock) NewTicker(d time.Duration) Ticker {
	m.mu.Lock()
	ticker := &mockTicker{
		period:   d,
		nextTick: m.currentTime.Add(d),
		c:        make(chan time.Time, 1),
		clock:    m,
		stopped:  false,
	}
	m.tickers = append(m.tickers, ticker)
	m.mu.Unlock()

	m.notifyTimerSetup()
	return ticker
}

// Set changes the current time to the specified value.
// Use it to test specific time points.
//
// Takes t (time.Time) which specifies the new current time.
//
// Safe for concurrent use.
func (m *MockClock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = t
}

// tickerFireEvent holds a pending ticker fire event and its scheduled time.
type tickerFireEvent struct {
	// ticker is the mock ticker whose channel receives the fire time.
	ticker *mockTicker

	// fireTime is the time value sent to the ticker channel when fired.
	fireTime time.Time
}

// Advance moves the current time forward by the specified duration.
//
// Any timers scheduled to fire during this window will be executed. This is
// useful for testing time progression and expiration.
//
// Takes d (time.Duration) which specifies how far to advance the clock.
func (m *MockClock) Advance(d time.Duration) {
	currentTime, toFire, channelTimersToFire, tickersToFire := m.advanceAndCollect(d)

	m.fireTimerCallbacks(toFire)
	m.fireChannelTimers(channelTimersToFire, currentTime)
	m.fireTickerEvents(tickersToFire)
}

// Rewind moves the current time backward by the specified duration.
// Use it to test time-sensitive edge cases.
//
// Takes d (time.Duration) which specifies how far back to move the clock.
//
// Safe for concurrent use.
func (m *MockClock) Rewind(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = m.currentTime.Add(-d)
}

// Freeze captures the current time and returns it. Use it in tests where you
// need to ensure time does not progress during a test operation.
//
// Returns time.Time which is the current frozen time.
//
// Safe for concurrent use.
func (m *MockClock) Freeze() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTime
}

// TimerCount returns the number of timer setup events (NewTimer, AfterFunc,
// NewTicker, and Reset calls) that have occurred on this clock. Use the
// returned value as the baseline argument to AwaitTimerSetup.
//
// Returns int64 which is the current monotonic counter value.
func (m *MockClock) TimerCount() int64 {
	return m.timerSetupCount.Load()
}

// AwaitTimerSetup blocks until at least one timer setup event has occurred
// after the given baseline count, or the timeout elapses.
//
// Takes baseline (int64) which is the TimerCount snapshot taken before the
// operation that should trigger a new timer setup.
// Takes timeout (time.Duration) which is the maximum real-world time to wait.
//
// Returns bool which is true if a new timer setup was observed, or false if
// the timeout was reached.
//
// Safe for concurrent use.
func (m *MockClock) AwaitTimerSetup(baseline int64, timeout time.Duration) bool {
	if m.timerSetupCount.Load() > baseline {
		return true
	}

	m.signalMu.Lock()
	signalChannel := m.timerSetupSignal
	m.signalMu.Unlock()

	select {
	case <-signalChannel:
		return m.timerSetupCount.Load() > baseline
	case <-time.After(timeout):
		return m.timerSetupCount.Load() > baseline
	}
}

// notifyTimerSetup increments the timer setup counter and broadcasts to any
// goroutine waiting in AwaitTimerSetup. Must be called after mu is released.
func (m *MockClock) notifyTimerSetup() {
	m.timerSetupCount.Add(1)

	m.signalMu.Lock()
	old := m.timerSetupSignal
	m.timerSetupSignal = make(chan struct{})
	m.signalMu.Unlock()

	close(old)
}

// advanceAndCollect advances the clock and collects all ready timers, channel
// timers, and ticker events under the mutex. Caller fires callbacks outside
// the lock to prevent deadlocks.
//
// Takes d (time.Duration) which specifies how far to advance the clock.
//
// Returns time.Time which is the new current time after advancing.
// Returns []*mockTimer which contains timers that are ready to fire.
// Returns []*mockChannelTimer which contains channel timers that are ready.
// Returns []tickerFireEvent which contains ticker events that are ready.
//
// Safe for concurrent use. Holds the mutex during collection.
func (m *MockClock) advanceAndCollect(d time.Duration) (time.Time, []*mockTimer, []*mockChannelTimer, []tickerFireEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentTime = m.currentTime.Add(d)
	currentTime := m.currentTime

	return currentTime,
		m.collectReadyTimers(currentTime),
		m.collectReadyChannelTimers(currentTime),
		m.collectReadyTickerEvents(currentTime)
}

// collectReadyTimers returns all callback timers ready to fire and marks them
// stopped. Caller must hold mu.
//
// Takes currentTime (time.Time) which specifies the time to check against.
//
// Returns []*mockTimer which contains timers ready to fire.
func (m *MockClock) collectReadyTimers(currentTime time.Time) []*mockTimer {
	var toFire []*mockTimer
	for _, t := range m.timers {
		if !t.stopped && !t.fireAt.After(currentTime) {
			toFire = append(toFire, t)
			t.stopped = true
		}
	}
	return toFire
}

// collectReadyChannelTimers returns all channel timers ready to fire and marks
// them stopped.
//
// Takes currentTime (time.Time) which specifies the time to compare against.
//
// Returns []*mockChannelTimer which contains timers ready to fire.
//
// Caller must hold mu.
func (m *MockClock) collectReadyChannelTimers(currentTime time.Time) []*mockChannelTimer {
	var toFire []*mockChannelTimer
	for _, t := range m.channelTimers {
		if !t.stopped && !t.fireAt.After(currentTime) {
			toFire = append(toFire, t)
			t.stopped = true
		}
	}
	return toFire
}

// collectReadyTickerEvents returns all ticker events ready to fire and
// advances their nextTick.
//
// Caller must hold mu.
//
// Takes currentTime (time.Time) which specifies the time to check against.
//
// Returns []tickerFireEvent which contains the events ready to fire.
func (m *MockClock) collectReadyTickerEvents(currentTime time.Time) []tickerFireEvent {
	var events []tickerFireEvent
	for _, t := range m.tickers {
		if t.stopped {
			continue
		}
		for !t.nextTick.After(currentTime) {
			events = append(events, tickerFireEvent{ticker: t, fireTime: t.nextTick})
			t.nextTick = t.nextTick.Add(t.period)
		}
	}
	return events
}

// fireTimerCallbacks invokes all timer callback functions.
//
// Takes timers ([]*mockTimer) which contains the timers whose callbacks to invoke.
func (*MockClock) fireTimerCallbacks(timers []*mockTimer) {
	for _, t := range timers {
		t.callback()
	}
}

// fireChannelTimers sends the current time to all ready channel timers.
//
// Takes timers ([]*mockChannelTimer) which contains the timers to fire.
// Takes currentTime (time.Time) which is the time to send to each timer.
func (*MockClock) fireChannelTimers(timers []*mockChannelTimer, currentTime time.Time) {
	for _, t := range timers {
		select {
		case t.c <- currentTime:
		default:
		}
	}
}

// fireTickerEvents sends fire times to all ready ticker channels.
//
// Takes events ([]tickerFireEvent) which specifies the tickers and times to
// fire.
func (*MockClock) fireTickerEvents(events []tickerFireEvent) {
	for _, tf := range events {
		select {
		case tf.ticker.c <- tf.fireTime:
		default:
		}
	}
}

// mockTimer implements clock.Timer for use with MockClock in tests.
type mockTimer struct {
	// fireAt is the time when the timer should fire.
	fireAt time.Time

	// callback is the function to run when the timer fires.
	callback func()

	// clock is the parent MockClock that manages this timer.
	clock *MockClock

	// stopped indicates whether this timer has been stopped or has already fired.
	stopped bool
}

// Stop prevents the timer from firing.
//
// Returns bool which is true if the timer was stopped before it
// fired, or false if it had already stopped or fired.
//
// Safe for concurrent use; protected by the parent clock's mutex.
func (t *mockTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()

	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}

// mockChannelTimer implements ChannelTimer for use in tests with MockClock.
type mockChannelTimer struct {
	// fireAt is when this timer should fire.
	fireAt time.Time

	// c is the channel that receives the current time when the timer fires.
	c chan time.Time

	// clock is the parent mock clock that controls this timer.
	clock *MockClock

	// stopped indicates whether the timer has been stopped or has fired.
	stopped bool
}

// Stop prevents the timer from firing.
//
// Returns bool which is true if the timer was stopped before
// firing, false otherwise.
//
// Safe for concurrent use; protected by the parent clock's mutex.
func (t *mockChannelTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()

	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}

// C returns the channel on which the time is delivered.
//
// Returns <-chan time.Time which delivers the timer's tick.
func (t *mockChannelTimer) C() <-chan time.Time {
	return t.c
}

// Reset changes the timer to expire after duration d.
//
// Takes d (time.Duration) which specifies the new expiry
// duration.
//
// Returns bool which is true if the timer was active, false if
// it had expired or been stopped.
//
// Safe for concurrent use; protected by the parent clock's mutex.
func (t *mockChannelTimer) Reset(d time.Duration) bool {
	t.clock.mu.Lock()
	wasActive := !t.stopped
	t.stopped = false
	t.fireAt = t.clock.currentTime.Add(d)
	t.clock.mu.Unlock()

	t.clock.notifyTimerSetup()
	return wasActive
}

// mockTicker implements the Ticker interface for use in tests with MockClock.
type mockTicker struct {
	// nextTick is when the ticker should next fire.
	nextTick time.Time

	// c is the channel that sends tick events to the ticker's consumer.
	c chan time.Time

	// clock is the parent MockClock that owns this ticker.
	clock *MockClock

	// period is the duration between each tick.
	period time.Duration

	// stopped indicates whether the ticker has been stopped.
	stopped bool
}

// C returns the channel on which the ticks are delivered.
//
// Returns <-chan time.Time which receives tick events.
func (t *mockTicker) C() <-chan time.Time {
	return t.c
}

// Stop halts the ticker and prevents further ticks.
//
// Safe for concurrent use; protected by the parent clock's mutex.
func (t *mockTicker) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.stopped = true
}

// RealClock returns a Clock that uses the real system time.
//
// Returns Clock which provides access to the current time for use in
// production code.
func RealClock() Clock {
	return realClock{}
}
