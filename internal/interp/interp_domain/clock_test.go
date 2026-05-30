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
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

type fixedClock struct {
	now           time.Time
	nowCalls      atomic.Int64
	sinceCalls    atomic.Int64
	untilCalls    atomic.Int64
	sleepCalls    atomic.Int64
	timerCalls    atomic.Int64
	tickerCalls   atomic.Int64
	lastSleepFor  atomic.Int64
	lastTimerFor  atomic.Int64
	lastTickerFor atomic.Int64
}

func (c *fixedClock) Now() time.Time {
	c.nowCalls.Add(1)
	return c.now
}

func (c *fixedClock) Since(t time.Time) time.Duration {
	c.sinceCalls.Add(1)
	return c.now.Sub(t)
}

func (c *fixedClock) Until(t time.Time) time.Duration {
	c.untilCalls.Add(1)
	return t.Sub(c.now)
}

func (c *fixedClock) Sleep(d time.Duration) {
	c.sleepCalls.Add(1)
	c.lastSleepFor.Store(int64(d))
}

func (c *fixedClock) NewTimer(d time.Duration) *time.Timer {
	c.timerCalls.Add(1)
	c.lastTimerFor.Store(int64(d))
	return time.NewTimer(d)
}

func (c *fixedClock) NewTicker(d time.Duration) *time.Ticker {
	c.tickerCalls.Add(1)
	c.lastTickerFor.Store(int64(d))
	return time.NewTicker(d)
}

func TestWallClockDefaults(t *testing.T) {
	t.Parallel()
	before := time.Now()
	got := WallClock.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("WallClock.Now() outside expected range: got %v, before=%v after=%v", got, before, after)
	}

	WallClock.Sleep(1 * time.Millisecond)
	if d := WallClock.Since(before); d <= 0 {
		t.Fatalf("WallClock.Since returned non-positive duration: %v", d)
	}
}

func TestEffectiveClockNilSubstitutesWallClock(t *testing.T) {
	t.Parallel()
	if got := effectiveClock(nil); got != WallClock {
		t.Fatalf("effectiveClock(nil) should return WallClock")
	}
	custom := &fixedClock{}
	if got := effectiveClock(custom); got != custom {
		t.Fatalf("effectiveClock(custom) should return the custom clock")
	}
}

func TestWithClockOption(t *testing.T) {
	t.Parallel()
	clock := &fixedClock{now: time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)}
	service := NewService(WithClock(clock))
	if service.config.clock != clock {
		t.Fatalf("service.config.clock did not receive WithClock value")
	}
}

func TestClockOverridesTimeNowInInterpretedCode(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{now: fixed}
	service := NewService(WithClock(clock), WithForceGoDispatch())
	service.UseSymbols(NewSymbolRegistry(SymbolExports{
		"time": {
			"Now":   reflect.ValueOf(time.Now),
			"Since": reflect.ValueOf(time.Since),
		},
	}))

	result, err := service.Eval(context.Background(), `
		import "time"
		time.Now()
	`)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	got, ok := result.(time.Time)
	if !ok {
		t.Fatalf("Eval result is not time.Time: %T %v", result, result)
	}
	if !got.Equal(fixed) {
		t.Fatalf("interpreted time.Now() = %v, want fixed %v", got, fixed)
	}
	if clock.nowCalls.Load() == 0 {
		t.Fatalf("fixed clock Now() was never called")
	}
}

func TestClockOverlaysPreserveOtherTimeSymbols(t *testing.T) {
	t.Parallel()
	clock := &fixedClock{now: time.Now()}
	service := NewService(WithClock(clock))
	service.UseSymbols(NewSymbolRegistry(SymbolExports{
		"time": {
			"Now":         reflect.ValueOf(time.Now),
			"Hour":        reflect.ValueOf(time.Hour),
			"Millisecond": reflect.ValueOf(time.Millisecond),
		},
	}))

	pkg, ok := service.symbols.PackageSymbols("time")
	if !ok {
		t.Fatalf("time package missing from registry after clock overlay")
	}
	if _, ok := pkg["Hour"]; !ok {
		t.Fatalf("time.Hour symbol lost after clock overlay")
	}
	if _, ok := pkg["Millisecond"]; !ok {
		t.Fatalf("time.Millisecond symbol lost after clock overlay")
	}
	if _, ok := pkg["Now"]; !ok {
		t.Fatalf("time.Now symbol missing after clock overlay")
	}
}

func TestOverlayPackageOnAbsentPackageCreatesIt(t *testing.T) {
	t.Parallel()
	registry := NewSymbolRegistry(nil)
	registry.OverlayPackage("custom/pkg", map[string]reflect.Value{
		"X": reflect.ValueOf(42),
	})
	got, ok := registry.PackageSymbols("custom/pkg")
	if !ok {
		t.Fatalf("OverlayPackage did not create the package")
	}
	if value, ok := got["X"]; !ok || value.Interface() != 42 {
		t.Fatalf("OverlayPackage did not store the entry")
	}
}
