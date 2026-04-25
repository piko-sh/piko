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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func TestWatchdog_StartupHistoryDetectsUncleanPreviousExit(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	history := startupHistoryFile{
		Entries: []startupHistoryEntry{{
			StartedAt: startTime.Add(-30 * time.Second),
			PID:       1234,
			Hostname:  "previous-host",
			Version:   "v0.0.1",
		}},
	}
	encoded, err := json.MarshalIndent(history, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.CrashLoopWindow = 60 * time.Second
	config.CrashLoopThreshold = 5

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock

	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	t.Cleanup(watchdog.Stop)

	watchdog.backgroundWG.Wait()

	previousEvents := notifier.getEventsByType(WatchdogEventPreviousCrashClassified)
	require.Len(t, previousEvents, 1, "previous crash should have been classified")
	assert.Equal(t, "1234", previousEvents[0].Fields["prev_pid"])
	assert.Equal(t, "previous-host", previousEvents[0].Fields["prev_hostname"])
}

func TestWatchdog_StartupHistoryDetectsCrashLoop(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	history := startupHistoryFile{
		Entries: []startupHistoryEntry{
			{StartedAt: startTime.Add(-50 * time.Second), PID: 100},
			{StartedAt: startTime.Add(-30 * time.Second), PID: 200},
			{StartedAt: startTime.Add(-10 * time.Second), PID: 300},
		},
	}
	encoded, err := json.MarshalIndent(history, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.CrashLoopWindow = 60 * time.Second
	config.CrashLoopThreshold = 3

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock

	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	t.Cleanup(watchdog.Stop)

	watchdog.backgroundWG.Wait()

	loopEvents := notifier.getEventsByType(WatchdogEventCrashLoopDetected)
	require.Len(t, loopEvents, 1, "three unclean entries within window should trigger crash loop alert")
	assert.Equal(t, WatchdogPriorityCritical, loopEvents[0].Priority)
	assert.Equal(t, "3", loopEvents[0].Fields["unclean_in_window"])
}

func TestWatchdog_StartupHistoryFirstRunNoEvents(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEventsByType(WatchdogEventPreviousCrashClassified))
	assert.Empty(t, notifier.getEventsByType(WatchdogEventCrashLoopDetected))

	data, err := watchdog.profileStore.sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.Len(t, file.Entries, 1)
	assert.Nil(t, file.Entries[0].StoppedAt)
}

func TestWatchdog_StartupHistoryStopMarksClean(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())
	watchdog.markStartupHistoryStopped(context.Background())

	data, err := watchdog.profileStore.sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.Len(t, file.Entries, 1)
	require.NotNil(t, file.Entries[0].StoppedAt)
	assert.Equal(t, "clean", file.Entries[0].Reason)
}

func TestWatchdog_StartupHistoryRingTrimsToTen(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	stopped := startTime.Add(-time.Hour)
	entries := make([]startupHistoryEntry, 10)
	for i := range entries {
		entries[i] = startupHistoryEntry{
			StartedAt: startTime.Add(-time.Duration(20-i) * time.Minute),
			StoppedAt: &stopped,
			PID:       1000 + i,
			Reason:    "clean",
		}
	}
	encoded, err := json.MarshalIndent(startupHistoryFile{Entries: entries}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())

	data, err := sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	assert.LessOrEqual(t, len(file.Entries), maxStartupHistoryEntries, "ring must not exceed maxStartupHistoryEntries")

	assert.Nil(t, file.Entries[len(file.Entries)-1].StoppedAt)
}

func TestProfileStore_WriteAndRotate(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	maxProfiles := 3
	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: maxProfiles,
	}

	for i := range 5 {
		mockClock.Advance(time.Minute)

		data := []byte("profile-data-" + string(rune('A'+i)))
		_, err := store.write("heap", data)
		require.NoError(t, err, "write %d should succeed", i)
	}

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)

	heapFileCount := 0
	for _, entry := range entries {
		if entry.Name() != "" && len(entry.Name()) > 5 && entry.Name()[:5] == "heap-" {
			heapFileCount++
		}
	}

	assert.Equal(t, maxProfiles, heapFileCount, "rotation should keep only %d profiles", maxProfiles)
}

func TestProfileStore_WriteReturnsTimestampForSidecarPairing(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 5,
	}

	timestamp, err := store.write("heap", []byte("test"))
	require.NoError(t, err)
	assert.Equal(t, "20260101T123045", timestamp, "timestamp matches profileTimestampFormat")
}

func TestProfileStore_WriteMetadataPairsWithProfile(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 5,
	}

	timestamp, err := store.write("heap", []byte("dummy-pprof"))
	require.NoError(t, err)

	meta := captureMetadata{
		CapturedAt:     startTime,
		RuleFired:      "heap_high_water",
		ProfileType:    "heap",
		ObservedValue:  1024,
		Threshold:      512,
		Hostname:       "test-host",
		Version:        "v0.0.0-test",
		PID:            42,
		HeapAllocBytes: 1024,
	}
	require.NoError(t, store.writeMetadata("heap", timestamp, meta))

	sidecarName := "heap-" + timestamp + profileSidecarExtension
	data, err := sandbox.ReadFile(sidecarName)
	require.NoError(t, err)

	var decoded captureMetadata
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "heap_high_water", decoded.RuleFired)
	assert.Equal(t, uint64(1024), decoded.ObservedValue)
	assert.Equal(t, uint64(512), decoded.Threshold)
	assert.Equal(t, "test-host", decoded.Hostname)
}

func TestProfileStore_RotateRemovesPairedSidecar(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 2,
	}

	for i := range 4 {
		mockClock.Advance(time.Minute)

		timestamp, err := store.write("heap", []byte("p"+string(rune('A'+i))))
		require.NoError(t, err)
		require.NoError(t, store.writeMetadata("heap", timestamp, captureMetadata{
			RuleFired:   "heap_high_water",
			ProfileType: "heap",
		}))
	}

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)

	pbgz := 0
	jsonCount := 0
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Name(), profileFileExtension):
			pbgz++
		case strings.HasSuffix(e.Name(), profileSidecarExtension):
			jsonCount++
		}
	}

	assert.Equal(t, 2, pbgz, "rotation keeps the most recent N profiles")
	assert.Equal(t, 2, jsonCount, "rotation deletes paired sidecars alongside profiles")
}

func TestProfileStore_GzipCompression(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 5,
	}

	originalData := []byte("this is test profile data for gzip verification")

	_, err = store.write("goroutine", originalData)
	require.NoError(t, err)

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	compressedData, err := sandbox.ReadFile(entries[0].Name())
	require.NoError(t, err)

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err, "stored file must be valid gzip")

	decompressedData, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())

	assert.Equal(t, originalData, decompressedData, "decompressed data should match the original input")
}

func TestProfileStore_NewProfileStoreCreatesSandbox(t *testing.T) {
	t.Parallel()

	tempDirectory := t.TempDir()
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	store, err := newProfileStore(tempDirectory, 5, mockClock)
	require.NoError(t, err)
	require.NotNil(t, store)
	t.Cleanup(func() { _ = store.close() })

	assert.Equal(t, 5, store.maxProfilesPerType)
	assert.NotNil(t, store.sandbox)
}

func TestProfileStore_ListSortsNewestFirstAndPairsSidecars(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 10,
	}

	mockClock.Advance(time.Second)
	tsOld, err := store.write("heap", []byte("old"))
	require.NoError(t, err)
	require.NoError(t, store.writeMetadata("heap", tsOld, captureMetadata{
		RuleFired: "heap_high_water", ProfileType: "heap",
	}))

	mockClock.Advance(time.Minute)
	tsNew, err := store.write("heap", []byte("new"))
	require.NoError(t, err)

	entries, err := store.list()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.True(t, entries[0].Timestamp.After(entries[1].Timestamp), "newest entry first")
	assert.Equal(t, "heap-"+tsNew+profileFileExtension, entries[0].Filename)
	assert.False(t, entries[0].HasSidecar, "newest profile has no paired sidecar")
	assert.True(t, entries[1].HasSidecar, "older profile has paired sidecar")
}

func TestProfileStore_ListIgnoresMalformedAndDirectoryEntries(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 10,
	}

	mockClock.Advance(time.Second)
	_, err = store.write("heap", []byte("valid"))
	require.NoError(t, err)

	require.NoError(t, sandbox.WriteFile("malformed-without-timestamp.pb.gz", []byte("x"), 0o640))
	require.NoError(t, sandbox.WriteFile("heap-INVALID.pb.gz", []byte("x"), 0o640))
	require.NoError(t, sandbox.WriteFile("notaprofile.txt", []byte("x"), 0o640))
	require.NoError(t, sandbox.MkdirAll("subdirectory", 0o750))

	entries, err := store.list()
	require.NoError(t, err)
	require.Len(t, entries, 1, "only the well-formed profile should be returned")
	assert.Equal(t, "heap", entries[0].Type)
}

func TestProfileStore_ReadReturnsBytesAndErrorsOnEmpty(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 10,
	}

	timestamp, err := store.write("heap", []byte("compressed-payload"))
	require.NoError(t, err)

	data, err := store.read("heap-" + timestamp + profileFileExtension)
	require.NoError(t, err)
	assert.NotEmpty(t, data, "stored bytes should be readable back")

	_, err = store.read("")
	require.ErrorIs(t, err, errEmptyFilename)

	_, err = store.read("nonexistent.pb.gz")
	require.Error(t, err)
}

func TestProfileStore_DeleteByTypeRemovesPairsAndCounts(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 10,
	}

	for index := range 3 {
		mockClock.Advance(time.Second)
		ts, err := store.write("heap", []byte("h"+string(rune('A'+index))))
		require.NoError(t, err)
		require.NoError(t, store.writeMetadata("heap", ts, captureMetadata{
			RuleFired: "heap_high_water", ProfileType: "heap",
		}))
	}

	mockClock.Advance(time.Second)
	_, err = store.write("goroutine", []byte("g"))
	require.NoError(t, err)

	deleted, err := store.deleteByType("heap")
	require.NoError(t, err)
	assert.Equal(t, 3, deleted, "deleteByType returns the number of profiles removed")

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasPrefix(entry.Name(), "heap-"), "no heap files should remain")
	}
}

func TestProfileStore_DeleteAllRemovesEveryProfile(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 10,
	}

	mockClock.Advance(time.Second)
	_, err = store.write("heap", []byte("h"))
	require.NoError(t, err)
	mockClock.Advance(time.Second)
	_, err = store.write("goroutine", []byte("g"))
	require.NoError(t, err)

	deleted, err := store.deleteAll()
	require.NoError(t, err)
	assert.Equal(t, 2, deleted, "deleteAll returns the number of profiles removed")

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), profileFileExtension),
			"no .pb.gz files should remain after deleteAll")
	}
}
