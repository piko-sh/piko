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

package orchestrator_test_bench

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	soakEnv             = "PIKO_ORCHESTRATOR_SOAK"
	soakDurationEnv     = "PIKO_ORCHESTRATOR_SOAK_DURATION"
	defaultSoakDuration = 5 * time.Minute
	soakBatch           = 500
	soakGoroutineSlack  = 20
	soakHeapSlack       = 32 << 20
)

func TestSoak(t *testing.T) {
	if os.Getenv(soakEnv) == "" {
		t.Skip("set " + soakEnv + "=1 to run the soak test")
	}
	duration := defaultSoakDuration
	if value := os.Getenv(soakDurationEnv); value != "" {
		parsed, err := time.ParseDuration(value)
		require.NoError(t, err, "%s must be a Go duration", soakDurationEnv)
		duration = parsed
	}

	for _, store := range storeKinds {
		t.Run(string(store), func(t *testing.T) {
			soak(t, newRig(t, rigOptions{store: store, transport: transportGoChannel}), duration)
		})
	}
}

func soak(t *testing.T, r *rig, duration time.Duration) {
	purger, ok := r.store.(orchestrator_domain.TaskStoreV2)
	require.True(t, ok, "the store must purge settled tasks")
	purge := func() {
		for {
			purged, err := purger.PurgeTerminal(r.ctx, time.Now().Add(time.Minute), 10_000)
			require.NoError(t, err)
			if purged == 0 {
				return
			}
		}
	}

	purge()
	goroutinesBefore, heapBefore := settledFootprint()
	startCalls := r.executor.calls.Load()

	dispatched := 0
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		base := r.settledCount()
		for range soakBatch {
			require.NoError(t, r.dispatcher.Dispatch(r.ctx, newBenchTask()))
		}
		_, err := r.completions.await(soakBatch, base)
		require.NoError(t, err, "every dispatched task settles")
		dispatched += soakBatch
		purge()
	}

	stats := r.dispatcher.Stats()
	assert.Zero(t, stats.TasksFailed, "no task fails")
	assert.Equal(t, int64(dispatched), r.executor.calls.Load()-startCalls, "every task runs exactly once")

	goroutinesAfter, heapAfter := settledFootprint()
	t.Logf("soaked %d tasks over %s: goroutines %d -> %d, heap %d MiB -> %d MiB",
		dispatched, duration, goroutinesBefore, goroutinesAfter, heapBefore>>20, heapAfter>>20)
	assert.LessOrEqual(t, goroutinesAfter, goroutinesBefore+soakGoroutineSlack, "goroutines do not accumulate")
	assert.LessOrEqual(t, heapAfter, heapBefore+soakHeapSlack, "the heap after GC returns to where it started")
}

func settledFootprint() (int, uint64) {
	runtime.GC()
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return runtime.NumGoroutine(), stats.HeapInuse
}
