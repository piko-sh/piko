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

package monitoring_domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/clock"
)

func TestHeapTrendBuffer_Slope_LinearGrowth(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)

	for step := range 10 {
		buffer.add(uint64(100 + step*100))
	}

	slope := buffer.slope()

	assert.InDelta(t, 100.0, slope, 0.001, "slope should be 100 bytes per sample for linear growth")
}

func TestHeapTrendBuffer_Slope_StableHeap(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)

	for range 10 {
		buffer.add(500)
	}

	slope := buffer.slope()

	assert.InDelta(t, 0.0, slope, 0.001, "slope should be zero for a stable heap")
}

func TestHeapTrendBuffer_Slope_InsufficientSamples(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)
	buffer.add(100)

	slope := buffer.slope()

	assert.Equal(t, 0.0, slope, "slope should be zero with fewer than 2 samples")
}

func TestHeapTrendBuffer_RingBufferWraps(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(5)

	for range 5 {
		buffer.add(0)
	}

	for step := range 5 {
		buffer.add(uint64(100 + step*100))
	}

	assert.True(t, buffer.isFull())

	slope := buffer.slope()

	assert.InDelta(t, 100.0, slope, 0.001, "slope should be 100 after ring buffer wraps")
}

func TestWatchdog_HeapTrendWarning(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.TrendWindowSize = 20
	config.TrendEvaluationInterval = 0
	config.TrendWarningHorizon = 5 * time.Minute
	config.CheckInterval = 500 * time.Millisecond
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	watchdog.heapTrendBuffer = newHeapTrendBuffer(config.TrendWindowSize)

	watchdog.gomemlimit = 1024 * 1024 * 1024

	baseMiB := uint64(800 * 1024 * 1024)
	stepMiB := uint64(10 * 1024 * 1024)

	for step := range 20 {
		heapAlloc := baseMiB + uint64(step)*stepMiB

		stats := SystemStats{
			Memory: MemoryInfo{HeapAlloc: heapAlloc},
		}

		mockClock.Advance(500 * time.Millisecond)
		watchdog.evaluate(context.Background(), &stats)
	}

	watchdog.backgroundWG.Wait()

	assert.NotEmpty(t, notifier.getEventsByType(WatchdogEventHeapTrendWarning),
		"heap trend warning notification should have been sent")
}

func TestWatchdog_HeapTrendNoWarningWhenStable(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.TrendWindowSize = 20
	config.TrendEvaluationInterval = 0
	config.TrendWarningHorizon = 5 * time.Minute
	config.CheckInterval = 500 * time.Millisecond

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime
	watchdog.heapTrendBuffer = newHeapTrendBuffer(config.TrendWindowSize)
	watchdog.gomemlimit = 1024 * 1024 * 1024

	for range 20 {
		stats := SystemStats{
			Memory: MemoryInfo{HeapAlloc: 200 * 1024 * 1024},
		}

		mockClock.Advance(500 * time.Millisecond)
		watchdog.evaluate(context.Background(), &stats)
	}

	trendEvents := notifier.getEventsByType(WatchdogEventHeapTrendWarning)
	assert.Empty(t, trendEvents, "no trend warning should be emitted for a stable heap")
}
