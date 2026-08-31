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
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
)

func TestWatchdog_ContinuousProfilingDisabledByDefault(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContinuousProfilingInterval = time.Minute

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	watchdog.captureRoutineProfiles(context.Background())
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()

	assert.Contains(t, calls, "heap", "captureRoutineProfiles dispatches based on Types regardless of Enabled flag")
}

func TestWatchdog_ContinuousProfilingValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mutate func(*WatchdogConfig)
		name   string
	}{
		{name: "interval too short", mutate: func(c *WatchdogConfig) {
			c.ContinuousProfilingEnabled = true
			c.ContinuousProfilingInterval = time.Second
			c.ContinuousProfilingRetention = 6
			c.ContinuousProfilingTypes = []string{"heap"}
		}},
		{name: "retention zero", mutate: func(c *WatchdogConfig) {
			c.ContinuousProfilingEnabled = true
			c.ContinuousProfilingInterval = 5 * time.Minute
			c.ContinuousProfilingRetention = 0
			c.ContinuousProfilingTypes = []string{"heap"}
		}},
		{name: "type cpu rejected", mutate: func(c *WatchdogConfig) {
			c.ContinuousProfilingEnabled = true
			c.ContinuousProfilingInterval = 5 * time.Minute
			c.ContinuousProfilingRetention = 6
			c.ContinuousProfilingTypes = []string{"cpu"}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultWatchdogConfig()
			tt.mutate(&config)
			err := validateWatchdogConfig(&config)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
		})
	}
}

func TestWatchdog_RoutineProfileIsUploaded(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContinuousProfilingTypes = []string{"heap"}
	config.ContinuousProfilingUpload = true

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.SetProfilingController(&mockProfilingController{})
	uploader := &mockWatchdogProfileUploader{}
	watchdog.profileUploader = uploader
	watchdog.startedAt = startTime

	watchdog.captureAndStoreRoutineProfile(context.Background(), "heap")

	watchdog.backgroundWG.Wait()

	uploads := uploader.getUploads()
	require.Len(t, uploads, 1, "a routine capture must reach the uploader")

	up := uploads[0]

	assert.Equal(t, "heap", up.profileType,
		"the wire must carry the bare profile type, not the routine- storage prefix")

	assert.Equal(t, "routine", up.metadata["reason"],
		"reason identifies what triggered the capture; without it a sink renders a blank trigger")
	assert.NotEmpty(t, up.metadata["host"],
		"host is the key a sink reads for the node column")
	assert.Equal(t, up.metadata["host"], up.metadata["hostname"],
		"both spellings are emitted so a sink reading either one resolves the node")

	assert.Equal(t, strconv.Itoa(os.Getpid()), up.metadata["pid"],
		"pid distinguishes a restarted process from a still-running one on the same host")
	assert.NotEmpty(t, up.metadata["version"],
		"version is how a receiver attributes a capture to a build")

	assert.Positive(t, up.dataLength, "the captured bytes must be forwarded")
}

func TestWatchdog_ContinuousProfilingProducesRoutinePrefixedFiles(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContinuousProfilingRetention = 3
	config.ContinuousProfilingTypes = []string{"heap"}

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	for range 5 {
		watchdog.captureAndStoreRoutineProfile(context.Background(), "heap")
		mockClock.Advance(time.Minute)
	}

	entries, err := watchdog.profileStore.sandbox.ReadDir(".")
	require.NoError(t, err)

	routinePbgz := 0
	routineJSON := 0
	threshold := 0
	for _, e := range entries {
		switch {
		case strings.HasPrefix(e.Name(), "routine-heap-") && strings.HasSuffix(e.Name(), profileFileExtension):
			routinePbgz++
		case strings.HasPrefix(e.Name(), "routine-heap-") && strings.HasSuffix(e.Name(), profileSidecarExtension):
			routineJSON++
		case strings.HasPrefix(e.Name(), "heap-"):
			threshold++
		}
	}
	assert.Equal(t, 3, routinePbgz, "routine retention bound applies independently from threshold rotation")
	assert.Equal(t, 3, routineJSON, "rotated routine .pb.gz files should also rotate their sidecars")
	assert.Equal(t, 0, threshold, "routine captures must not collide with threshold-triggered prefix")
}

func TestWatchdog_ContinuousProfilingLoopFiresOnTickerAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 17, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContinuousProfilingEnabled = true
	config.ContinuousProfilingInterval = 5 * time.Minute
	config.ContinuousProfilingTypes = []string{"heap"}
	config.ContinuousProfilingRetention = 3

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan struct{})
	go func() {
		watchdog.continuousProfilingLoop(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		mockClock.Advance(6 * time.Minute)
		return len(controller.getCaptureCalls()) > 0
	}, 5*time.Second, 50*time.Millisecond, "expected at least one routine capture")

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("continuousProfilingLoop did not exit after ctx cancel")
	}
}

func TestBuildVersionString_MatchesTheServiceVersionEnvironment(t *testing.T) {
	t.Setenv("PIKO_SERVICE_VERSION", "v9.9.9")
	assert.Equal(t, "v9.9.9", buildVersionString())
}

func TestBuildVersionString_TheEnvironmentOutranksTheLdflagVersion(t *testing.T) {
	t.Setenv("PIKO_SERVICE_VERSION", "v9.9.9")

	previous := Version
	t.Cleanup(func() { Version = previous })
	Version = "v1.2.3"

	assert.Equal(t, "v9.9.9", buildVersionString(),
		"a redeployed binary is retagged by its environment, not by what was linked into it")
}

func TestBuildVersionString_FallsBackToTheLdflagVersion(t *testing.T) {
	t.Setenv("PIKO_SERVICE_VERSION", "")

	previous := Version
	t.Cleanup(func() { Version = previous })
	Version = "v1.2.3"

	assert.Equal(t, "v1.2.3", buildVersionString())
}

func TestWatchdog_RoutineProfileIsNotUploadedByDefault(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.ContinuousProfilingTypes = []string{"heap"}

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.SetProfilingController(&mockProfilingController{})
	uploader := &mockWatchdogProfileUploader{}
	watchdog.profileUploader = uploader
	watchdog.startedAt = startTime

	watchdog.captureAndStoreRoutineProfile(context.Background(), "heap")
	watchdog.backgroundWG.Wait()

	assert.Empty(t, uploader.getUploads(),
		"a routine capture stays local unless uploading was asked for")
}
