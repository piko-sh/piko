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
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

var (
	throughputBatchSizes = []int{1, 100, 10_000}
)

func BenchmarkThroughput(b *testing.B) {
	for _, size := range throughputBatchSizes {
		for _, store := range storeKinds {
			for _, transport := range transportKinds {
				opts := rigOptions{store: store, transport: transport}
				for _, mode := range logModes {
					b.Run(fmt.Sprintf("N=%d/%s/log=%s", size, opts, mode), func(b *testing.B) {
						applyLogMode(b, mode)
						runThroughput(b, newRig(b, opts), size)
					})
				}
			}
		}
	}
}

func runThroughput(b *testing.B, r *rig, size int) {
	var (
		elapsed   time.Duration
		rejected  int
		lost      int
		completed int
		latencies = make([]time.Duration, 0, size)
		before    memSample
		after     memSample
		first     = true
	)

	for b.Loop() {
		b.StopTimer()
		tasks := make([]*orchestrator_domain.Task, size)
		tracker := &latencyTracker{index: make(map[string]int, size), ends: make([]atomic.Int64, size)}
		for i := range tasks {
			tasks[i] = newBenchTask()
			tracker.index[tasks[i].ID] = i
		}
		starts := make([]int64, size)
		accepted := make([]bool, size)
		r.completions.tracker.Store(tracker)
		if first {
			before = sampleMem()
			first = false
		}
		b.StartTimer()

		base := r.settledCount()
		begin := time.Now()
		batchAccepted := 0
		for i, task := range tasks {
			starts[i] = time.Now().UnixNano()
			if err := r.dispatcher.Dispatch(r.ctx, task); err != nil {
				rejected++
				continue
			}
			accepted[i] = true
			batchAccepted++
		}
		received, err := r.completions.await(batchAccepted, base)
		if err != nil {
			b.Fatal(err)
		}
		elapsed += time.Since(begin)
		completed += batchAccepted
		lost += batchAccepted - received

		b.StopTimer()
		r.completions.tracker.Store(nil)
		for i := range tasks {
			if accepted[i] && tracker.ends[i].Load() != 0 {
				latencies = append(latencies, time.Duration(tracker.ends[i].Load()-starts[i]))
			}
		}
		b.StartTimer()
	}
	after = sampleMem()

	operations := b.N
	reportPerTask(b, before, after, completed+rejected)
	if completed > 0 {
		b.ReportMetric(float64(elapsed.Nanoseconds())/float64(completed), "ns/task")
	}
	b.ReportMetric(float64(rejected)/float64(operations), "rejected/op")
	b.ReportMetric(float64(lost)/float64(operations), "lost-events/op")
	if len(latencies) > 0 {
		slices.Sort(latencies)
		b.ReportMetric(float64(percentile(latencies, 50).Nanoseconds()), "p50-ns")
		b.ReportMetric(float64(percentile(latencies, 99).Nanoseconds()), "p99-ns")
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	rank := max((p*len(sorted)+99)/100, 1)
	return sorted[rank-1]
}
