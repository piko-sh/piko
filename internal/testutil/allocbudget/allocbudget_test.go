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
	"testing"
)

var (
	sink []byte
)

func TestMeasure_CountsAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}

	free := Measure(100, func() {})
	if free.Allocs != 0 {
		t.Errorf("empty call: got %.2f allocs/op, want 0", free.Allocs)
	}

	one := Measure(100, func() { sink = make([]byte, 64) })
	if one.Allocs != 1 {
		t.Errorf("single allocation: got %.2f allocs/op, want 1", one.Allocs)
	}
	if one.Bytes < 64 {
		t.Errorf("single allocation: got %.0f B/op, want at least 64", one.Bytes)
	}
}

func TestAssert_WithinBudget(t *testing.T) {
	Assert(t, Budget{Allocs: 1, Bytes: 128}, func() { sink = make([]byte, 64) })
}

func TestAssert_ReportsExceededBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}

	recorder := &failureRecorder{TB: t}
	Assert(recorder, Budget{Allocs: 0}, func() { sink = make([]byte, 64) })
	if !recorder.failed {
		t.Error("expected an allocation over budget to fail the test")
	}
}

type failureRecorder struct {
	testing.TB
	failed bool
}

func (r *failureRecorder) Errorf(string, ...any) { r.failed = true }

func (*failureRecorder) Helper() {}
