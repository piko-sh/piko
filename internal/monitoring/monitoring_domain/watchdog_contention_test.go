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
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestWatchdog_RunContentionDiagnosticAlreadyRunning(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.SetProfilingController(&mockProfilingController{})
	watchdog.startedAt = startTime

	watchdog.contentionMu.Lock()
	defer watchdog.contentionMu.Unlock()

	err := watchdog.RunContentionDiagnostic(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentionDiagnosticInProgress)
}

func TestWatchdog_RunContentionDiagnosticErrorsWithoutController(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	err := watchdog.RunContentionDiagnostic(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProfilingControllerNil)
}

func TestWatchdog_RunContentionDiagnosticRespectsCooldown(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContentionDiagnosticCooldown = time.Hour

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.SetProfilingController(&mockProfilingController{})
	watchdog.startedAt = startTime

	watchdog.mu.Lock()
	watchdog.lastContentionDiagnosticAt = mockClock.Now()
	watchdog.mu.Unlock()

	err := watchdog.RunContentionDiagnostic(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentionDiagnosticCooldown)
}

func TestWatchdog_RunContentionDiagnosticErrorsWhenStopped(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.SetProfilingController(&mockProfilingController{})
	watchdog.Stop()

	err := watchdog.RunContentionDiagnostic(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWatchdogStopped)
}

func TestWatchdog_RunContentionDiagnosticHappyPathCapturesProfiles(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 16, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContentionDiagnosticWindowDuration = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	go func() {

		time.Sleep(100 * time.Millisecond)
		mockClock.Advance(2 * time.Second)
	}()

	require.NoError(t, watchdog.RunContentionDiagnostic(context.Background()))

	calls := controller.getCaptureCalls()
	assert.Contains(t, calls, "block", "block profile should have been captured")
	assert.Contains(t, calls, "mutex", "mutex profile should have been captured")
}
