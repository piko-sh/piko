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

package daemon_adapters

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
)

func TestCachedMetricOption_ReusesTheOptionForAPathAndMethod(t *testing.T) {
	t.Parallel()

	first := cachedMetricOption("/otel-test/reuse", "GET")
	second := cachedMetricOption("/otel-test/reuse", "GET")
	other := cachedMetricOption("/otel-test/reuse", "POST")

	require.NotNil(t, first)
	assert.Same(t, first, second, "the same path and method must share one option")
	assert.NotSame(t, first, other, "a different method must get its own option")
}

func TestCachedMetricOption_IsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	const workers = 16

	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Go(func() {
			assert.NotNil(t, cachedMetricOption("/otel-test/concurrent", "GET"))
		})
	}
	waitGroup.Wait()

	cache := metricAttrCache.Load()
	require.NotNil(t, cache)
	assert.Contains(t, *cache, metricAttrKey{path: "/otel-test/concurrent", method: "GET"})
}

func TestCachedMetricOption_ConcurrentWritersNeverLoseEntries(t *testing.T) {
	t.Parallel()

	const workers = 32

	options := make([]metric.MeasurementOption, workers)
	var waitGroup sync.WaitGroup
	for index := range workers {
		waitGroup.Go(func() {
			options[index] = cachedMetricOption(fmt.Sprintf("/otel-test/writer-%d", index), "POST")
		})
	}
	waitGroup.Wait()

	cache := metricAttrCache.Load()
	require.NotNil(t, cache)
	for index := range workers {
		key := metricAttrKey{path: fmt.Sprintf("/otel-test/writer-%d", index), method: "POST"}
		require.Contains(t, *cache, key, "a concurrent copy-on-write update must not drop another writer's entry")
		assert.Same(t, options[index], (*cache)[key], "the published option must be the one returned to the writer")
	}
}

func TestMetricInstrumentsAreRegistered(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, httpResponseSize)
	assert.NotNil(t, pageRequestCount)
	assert.NotNil(t, actionRequestCount)
	assert.NotNil(t, serverStartupDuration)
	assert.NotNil(t, serverShutdownDuration)
	assert.NotNil(t, tlsCertificateReloadErrorCount)
}

func BenchmarkCachedMetricOption_Parallel(b *testing.B) {
	paths := []string{"/bench/a", "/bench/b", "/bench/c", "/bench/d"}
	for _, path := range paths {
		cachedMetricOption(path, "GET")
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			_ = cachedMetricOption(paths[index&3], "GET")
			index++
		}
	})
}
