---
title: Clock API
description: Time abstraction for deterministic tests.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 230
---

# Clock API

Piko injects time through a `Clock` interface so tests can drive code that depends on deadlines, timers, or wall-clock time deterministically. Production code receives the real system clock. Tests receive `MockClock` and advance time explicitly. Source file: [`wdk/clock/clock.go`](https://github.com/piko-sh/piko/blob/master/wdk/clock/clock.go).

## Interfaces

```go
type Clock interface {
    Now() time.Time
    AfterFunc(d time.Duration, f func()) Timer
    NewTimer(d time.Duration) ChannelTimer
    NewTicker(d time.Duration) Ticker
}

type Timer interface {
    Stop() bool
}

type ChannelTimer interface {
    Timer
    C() <-chan time.Time
    Reset(d time.Duration) bool
}

type Ticker interface {
    C() <-chan time.Time
    Stop()
}
```

## Constructors

```go
func RealClock() Clock
func NewMockClock(startTime time.Time) *MockClock
```

## `MockClock`

```go
func (m *MockClock) Now() time.Time
func (m *MockClock) AfterFunc(d time.Duration, f func()) Timer
func (m *MockClock) NewTimer(d time.Duration) ChannelTimer
func (m *MockClock) NewTicker(d time.Duration) Ticker
func (m *MockClock) Set(t time.Time)
func (m *MockClock) Advance(d time.Duration)
func (m *MockClock) Rewind(d time.Duration)
func (m *MockClock) Freeze() time.Time
func (m *MockClock) TimerCount() int64
func (m *MockClock) AwaitTimerSetup(baseline int64, timeout time.Duration) bool
```

`Advance` moves the clock forward and fires all scheduled timers whose deadlines fall inside the interval. Fires execute inline, not on new goroutines. `AwaitTimerSetup` is the synchronisation primitive for tests where the code under test schedules a timer on a goroutine.

## See also

- [How to testing](../how-to/testing.md) for using `MockClock` in action tests.
