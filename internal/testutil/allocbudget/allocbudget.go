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

package allocbudget

import (
	"runtime"
	"testing"
	"unsafe"
)

const (
	// defaultRuns is the number of measured calls when a budget does not set Runs.
	defaultRuns = 200

	// pointerSize64 is the size in bytes of a pointer on a 64-bit platform.
	pointerSize64 = 8
)

// Budget is the most a single call of a hot path may allocate.
type Budget struct {
	// Allocs is the maximum average number of heap allocations per call.
	Allocs float64

	// Bytes is the maximum average number of heap bytes allocated per call; zero or less
	// leaves bytes unchecked.
	Bytes int64

	// Runs is the number of measured calls; zero or less uses defaultRuns.
	Runs int
}

// Result is the measured average cost of one call.
type Result struct {
	// Allocs is the average number of heap allocations per call.
	Allocs float64

	// Bytes is the average number of heap bytes allocated per call.
	Bytes float64
}

// Measure reports the average heap allocations and bytes of one call of fn.
//
// fn is called once to warm caches and pools, then runs times while the process runs on a
// single P, so allocations made by other goroutines during the measurement are rare. The
// counts are process-wide, so goroutines started by fn are included.
//
// Takes runs (int) which is the number of measured calls; zero or less uses defaultRuns.
// Takes fn (func()) which is the call to measure.
//
// Returns Result which holds the average allocations and bytes per call.
func Measure(runs int, fn func()) Result {
	if runs <= 0 {
		runs = defaultRuns
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	fn()

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs {
		fn()
	}
	runtime.ReadMemStats(&after)

	return Result{
		Allocs: float64(after.Mallocs-before.Mallocs) / float64(runs),
		Bytes:  float64(after.TotalAlloc-before.TotalAlloc) / float64(runs),
	}
}

// Assert fails the test when one call of fn allocates more than the budget allows.
//
// The measured cost is always logged so budgets can be tightened as hot paths improve.
// The check is skipped under the race detector and on 32-bit platforms.
//
// Takes t (testing.TB) which reports test failures.
// Takes budget (Budget) which is the most one call may allocate.
// Takes fn (func()) which is the call to measure.
func Assert(t testing.TB, budget Budget, fn func()) {
	t.Helper()
	if reason := skipReason(); reason != "" {
		t.Skip(reason)
	}

	result := Measure(budget.Runs, fn)
	t.Logf("allocs/op=%.2f (budget %.2f) bytes/op=%.0f (budget %d)",
		result.Allocs, budget.Allocs, result.Bytes, budget.Bytes)

	if result.Allocs > budget.Allocs {
		t.Errorf("allocation budget exceeded: %.2f allocs/op, budget %.2f", result.Allocs, budget.Allocs)
	}
	if budget.Bytes > 0 && result.Bytes > float64(budget.Bytes) {
		t.Errorf("byte budget exceeded: %.0f B/op, budget %d", result.Bytes, budget.Bytes)
	}
}

// skipReason explains why allocation budgets cannot be checked in this build, if they
// cannot.
//
// Returns string which is the reason, or empty when budgets can be checked.
func skipReason() string {
	switch {
	case raceEnabled:
		return "allocation budgets are not checked under the race detector"
	case unsafe.Sizeof(uintptr(0)) < pointerSize64:
		return "allocation budgets are only checked on 64-bit platforms"
	default:
		return ""
	}
}
