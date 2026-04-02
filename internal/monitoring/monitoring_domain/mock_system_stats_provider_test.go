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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockSystemStatsProvider_GetStats(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStatsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockSystemStatsProvider{}

		result := mock.GetStats()

		assert.Equal(t, SystemStats{}, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStatsCallCount))
	})

	t.Run("delegates to GetStatsFunc", func(t *testing.T) {
		t.Parallel()

		expected := SystemStats{
			MonitoringListenAddr: "127.0.0.1:9090",
			NumCPU:               8,
			GOMAXPROCS:           8,
			NumGoroutines:        42,
			TimestampMs:          1700000000000,
			UptimeMs:             360000,
			CPUMillicores:        1250.5,
			Memory: MemoryInfo{
				Alloc:     1024 * 1024 * 64,
				HeapAlloc: 1024 * 1024 * 48,
				Sys:       1024 * 1024 * 128,
			},
			Build: BuildInfo{
				GoVersion: "go1.23.0",
				Version:   "1.0.0",
				Commit:    "abc123def",
			},
			Process: ProcessInfo{
				PID:         12345,
				ThreadCount: 16,
				Hostname:    "test-host",
			},
		}

		mock := &MockSystemStatsProvider{
			GetStatsFunc: func() SystemStats {
				return expected
			},
		}

		result := mock.GetStats()

		assert.Equal(t, expected, result)
		assert.Equal(t, "127.0.0.1:9090", result.MonitoringListenAddr)
		assert.Equal(t, int32(8), result.NumCPU)
		assert.Equal(t, int32(42), result.NumGoroutines)
		assert.Equal(t, "go1.23.0", result.Build.GoVersion)
		assert.Equal(t, int32(12345), result.Process.PID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStatsCallCount))
	})
}

func TestMockSystemStatsProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockSystemStatsProvider

	result := mock.GetStats()

	assert.Equal(t, SystemStats{}, result)
	assert.Equal(t, int64(0), result.TimestampMs)
	assert.Equal(t, int32(0), result.NumCPU)
	assert.Equal(t, int32(0), result.NumGoroutines)
	assert.Equal(t, "", result.MonitoringListenAddr)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStatsCallCount))
}

func TestMockSystemStatsProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockSystemStatsProvider{
		GetStatsFunc: func() SystemStats {
			return SystemStats{
				NumGoroutines: 100,
				TimestampMs:   1700000000000,
			}
		},
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			result := mock.GetStats()
			assert.Equal(t, int32(100), result.NumGoroutines)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetStatsCallCount))
}

func TestMockSystemStatsProvider_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ SystemStatsProvider = (*MockSystemStatsProvider)(nil)
}
