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
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestParseThreadCountLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "valid thread count",
			line:     "Threads:\t8",
			expected: 8,
		},
		{
			name:     "thread count with spaces",
			line:     "Threads:    12",
			expected: 12,
		},
		{
			name:     "single thread",
			line:     "Threads:\t1",
			expected: 1,
		},
		{
			name:     "large thread count",
			line:     "Threads:\t1000",
			expected: 1000,
		},
		{
			name:     "empty line",
			line:     "",
			expected: 0,
		},
		{
			name:     "only prefix",
			line:     "Threads:",
			expected: 0,
		},
		{
			name:     "invalid number",
			line:     "Threads:\tabc",
			expected: 0,
		},
		{
			name:     "negative number",
			line:     "Threads:\t-5",
			expected: -5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseThreadCountLine(tc.line)
			if result != tc.expected {
				t.Errorf("parseThreadCountLine(%q) = %d, want %d", tc.line, result, tc.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		bytes    int64
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0B",
		},
		{
			name:     "small bytes",
			bytes:    512,
			expected: "512B",
		},
		{
			name:     "exactly 1 KiB",
			bytes:    1024,
			expected: "1.0KiB",
		},
		{
			name:     "1.5 KiB",
			bytes:    1536,
			expected: "1.5KiB",
		},
		{
			name:     "exactly 1 MiB",
			bytes:    1024 * 1024,
			expected: "1.0MiB",
		},
		{
			name:     "exactly 1 GiB",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0GiB",
		},
		{
			name:     "2.5 GiB",
			bytes:    int64(2.5 * 1024 * 1024 * 1024),
			expected: "2.5GiB",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := formatBytes(tc.bytes)
			if result != tc.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tc.bytes, result, tc.expected)
			}
		})
	}
}

func TestSystemCollector_UptimeCalculation(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	stats := collector.GetStats()
	if stats.UptimeMs != 0 {
		t.Errorf("expected initial uptime 0, got %d", stats.UptimeMs)
	}

	mockClock.Advance(time.Hour)
	stats = collector.GetStats()

	expectedUptimeMs := time.Hour.Milliseconds()
	if stats.UptimeMs != expectedUptimeMs {
		t.Errorf("expected uptime %d ms, got %d ms", expectedUptimeMs, stats.UptimeMs)
	}

	mockClock.Advance(30 * time.Minute)
	stats = collector.GetStats()

	expectedUptimeMs = (time.Hour + 30*time.Minute).Milliseconds()
	if stats.UptimeMs != expectedUptimeMs {
		t.Errorf("expected uptime %d ms, got %d ms", expectedUptimeMs, stats.UptimeMs)
	}
}

func TestSystemCollector_GetStats(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	collector.sample()

	stats := collector.GetStats()

	if stats.NumCPU <= 0 {
		t.Error("expected NumCPU > 0")
	}
	if stats.GOMAXPROCS <= 0 {
		t.Error("expected GOMAXPROCS > 0")
	}
	if stats.NumGoroutines <= 0 {
		t.Error("expected NumGoroutines > 0")
	}
	if stats.TimestampMs != startTime.UnixMilli() {
		t.Errorf("expected TimestampMs %d, got %d", startTime.UnixMilli(), stats.TimestampMs)
	}

	if stats.Memory.Sys == 0 {
		t.Error("expected Memory.Sys > 0")
	}

	if stats.Build.GoVersion == "" {
		t.Error("expected Build.GoVersion to be set")
	}

	if stats.Runtime.GOGC == "" {
		t.Error("expected Runtime.GOGC to be set")
	}
}

func TestSystemCollector_StopIdempotent(t *testing.T) {
	t.Parallel()

	collector := NewSystemCollector()

	collector.Stop()
	collector.Stop()
	collector.Stop()
}

func TestBuildBuildInfo(t *testing.T) {
	t.Parallel()

	info := buildBuildInfo()

	if info.GoVersion == "" {
		t.Error("expected GoVersion to be set")
	}
	if info.OS == "" {
		t.Error("expected OS to be set")
	}
	if info.Arch == "" {
		t.Error("expected Arch to be set")
	}

	if info.Version == "" {
		t.Error("expected Version to be set")
	}
}

func TestBuildRuntimeConfig(t *testing.T) {
	t.Parallel()

	config := buildRuntimeConfig()

	if config.GOGC == "" {
		t.Error("expected GOGC to be set")
	}

	if config.GOMEMLIMIT == "" {
		t.Error("expected GOMEMLIMIT to be set")
	}
}

func TestWithListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "sets localhost address",
			address:  "127.0.0.1:9091",
			expected: "127.0.0.1:9091",
		},
		{
			name:     "sets empty address",
			address:  "",
			expected: "",
		},
		{
			name:     "sets ipv4 address",
			address:  "10.0.0.1:8080",
			expected: "10.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			mockClock := clock.NewMockClock(startTime)

			collector := NewSystemCollector(
				WithSystemCollectorClock(mockClock),
				WithListenAddress(tt.address),
			)

			stats := collector.GetStats()
			assert.Equal(t, tt.expected, stats.MonitoringListenAddr)
		})
	}
}

func TestSystemCollector_StartAndStopViaContext(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	ctx, cancel := context.WithCancelCause(context.Background())

	collector.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))

	time.Sleep(50 * time.Millisecond)

	collector.Stop()
}

func TestSystemCollector_LoopTicksOnClock(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	collector.Start(ctx)

	mockClock.Advance(2 * time.Second)

	time.Sleep(100 * time.Millisecond)

	stats := collector.GetStats()
	assert.Greater(t, stats.Memory.Sys, uint64(0), "sample should have been called")

	cancel(fmt.Errorf("test: cleanup"))
	time.Sleep(50 * time.Millisecond)
	collector.Stop()
}

func TestSystemCollector_SampleUpdatesMemStats(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	collector.mu.RLock()
	preSampleSys := collector.memStats.Sys
	collector.mu.RUnlock()

	assert.Equal(t, uint64(0), preSampleSys)

	collector.sample()

	collector.mu.RLock()
	postSampleSys := collector.memStats.Sys
	collector.mu.RUnlock()

	assert.Greater(t, postSampleSys, uint64(0))
}

func TestSystemCollector_SampleCPUCalculation(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	collector.sample()

	collector.mu.RLock()
	firstCPUTime := collector.lastCPUTime
	collector.mu.RUnlock()

	mockClock.Advance(time.Second)
	collector.sample()

	collector.mu.RLock()
	secondCPUTime := collector.lastCPUTime
	millicores := collector.cpuMillicores
	collector.mu.RUnlock()

	assert.GreaterOrEqual(t, secondCPUTime, firstCPUTime)

	assert.GreaterOrEqual(t, millicores, float64(0))
}

func TestSystemCollector_GetStats_ProcessInfo(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	collector.sample()

	stats := collector.GetStats()

	assert.Greater(t, stats.Process.PID, int32(0))
	assert.Greater(t, stats.Process.ThreadCount, int32(0))
	assert.Greater(t, stats.Process.FDCount, int32(0))
	assert.NotEmpty(t, stats.Process.Executable)
	assert.NotEmpty(t, stats.Process.CWD)
	assert.NotEmpty(t, stats.Process.Hostname)
}

func TestSystemCollector_GetStats_BuildInfo(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	stats := collector.GetStats()

	assert.NotEmpty(t, stats.Build.GoVersion)
	assert.NotEmpty(t, stats.Build.OS)
	assert.NotEmpty(t, stats.Build.Arch)
	assert.NotEmpty(t, stats.Build.Version)
}

func TestSystemCollector_GetStats_RuntimeInfo(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	stats := collector.GetStats()

	assert.NotEmpty(t, stats.Runtime.GOGC)
	assert.NotEmpty(t, stats.Runtime.GOMEMLIMIT)
	assert.NotEmpty(t, stats.Runtime.Compiler)
}

func TestSystemCollector_GetStats_SystemUptime(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	stats := collector.GetStats()

	assert.Greater(t, stats.SystemUptimeMs, int64(0))
}

func TestSystemCollector_GetStats_MemoryInfo(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	collector.sample()

	stats := collector.GetStats()

	assert.Greater(t, stats.Memory.Alloc, uint64(0))
	assert.Greater(t, stats.Memory.Sys, uint64(0))
	assert.Greater(t, stats.Memory.HeapAlloc, uint64(0))
	assert.Greater(t, stats.Memory.HeapSys, uint64(0))
}

func TestSystemCollector_GetStats_GCInfo(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	collector.sample()

	stats := collector.GetStats()

	require.NotNil(t, stats.GC.RecentPauses)
}

func TestReadProcessCPUTime(t *testing.T) {
	t.Parallel()

	burnCPU()

	cpuTime := readProcessCPUTime()

	assert.Greater(t, cpuTime, uint64(0))
}

func TestReadThreadCount(t *testing.T) {
	t.Parallel()

	count := readThreadCount()

	assert.Greater(t, count, 0)
}

func TestReadFDCount(t *testing.T) {
	t.Parallel()

	count := readFDCount()

	assert.Greater(t, count, 0)
}

func TestReadRSS(t *testing.T) {
	t.Parallel()

	rss := readRSS()

	assert.Greater(t, rss, uint64(0))
}

func TestReadMaxOpenFiles(t *testing.T) {
	t.Parallel()

	soft, hard := readMaxOpenFiles()

	assert.Greater(t, soft, int64(0))
	assert.Greater(t, hard, int64(0))
	assert.GreaterOrEqual(t, hard, soft)
}

func TestReadIOStats(t *testing.T) {
	t.Parallel()

	stats := readIOStats()

	assert.Greater(t, stats.Rchar, uint64(0))
}

func TestReadSystemUptime(t *testing.T) {
	t.Parallel()

	uptime := readSystemUptime()

	assert.Greater(t, uptime, int64(0))
}

func TestReadCgroupPath(t *testing.T) {
	t.Parallel()

	path := readCgroupPath()
	_ = path
}

func TestBuildProcessInfo(t *testing.T) {
	t.Parallel()

	info := buildProcessInfo()

	assert.Greater(t, info.PID, 0)
	assert.Greater(t, info.ThreadCount, 0)
	assert.Greater(t, info.FDCount, 0)
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.Executable)
	assert.NotEmpty(t, info.CWD)
	assert.Greater(t, info.RSS, uint64(0))
	assert.Greater(t, info.MaxOpenFilesSoft, int64(0))
	assert.Greater(t, info.MaxOpenFilesHard, int64(0))
}

func burnCPU() {
	deadline := time.Now().Add(20 * time.Millisecond)
	sink := 0
	for time.Now().Before(deadline) {
		for range 1000 {
			sink++
		}
	}
	runtime.KeepAlive(sink)
}
