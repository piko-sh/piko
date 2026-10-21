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

package dalconformance

import (
	"slices"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

const (
	// nodeA is the node instance the cases claim tasks, register leases and queue receipts
	// for.
	nodeA = "node-a"
)

var (
	// Epoch is the time every conformance store's clock starts at. It is a whole second, so
	// stores that keep second precision round-trip it exactly.
	Epoch = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

// Backend describes one task store implementation under test.
type Backend struct {
	// Open creates an empty store whose own timestamps and time comparisons use clock. The
	// store must be closed by a cleanup registered on t.
	Open func(t *testing.T, clock clockpkg.Clock) Instance

	// Gaps maps unsupported conformance cases to their skip reasons.
	Gaps map[string]string
}

// Instance is one open store.
type Instance struct {
	// Store is the store under test.
	Store orchestrator_domain.TaskStore

	// Reopen closes Store and opens the same data again, as a restart would. It is nil when
	// the store keeps nothing across a restart.
	Reopen func(t *testing.T) orchestrator_domain.TaskStore
}

// Harness is what each conformance case runs against.
type Harness struct {
	// Instance is the open store.
	Instance

	// Clock is the store's clock, starting at Epoch.
	Clock *clockpkg.MockClock
}

// testCase is one named conformance check.
type testCase struct {
	// run performs the check.
	run func(t *testing.T, h *Harness)

	// name identifies the case in test output and in Backend.Gaps.
	name string
}

// Run runs every conformance case against backend, each as a subtest on a fresh store.
//
// Takes t (*testing.T) which reports test failures.
// Takes backend (Backend) which describes the store under test.
func Run(t *testing.T, backend Backend) {
	t.Helper()
	cases := allCases()

	names := make([]string, len(cases))
	for i, c := range cases {
		names[i] = c.name
	}
	for name := range backend.Gaps {
		if !slices.Contains(names, name) {
			t.Errorf("gap names an unknown conformance case %q", name)
		}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if reason, known := backend.Gaps[c.name]; known {
				t.Skip(reason)
			}
			clock := clockpkg.NewMockClock(Epoch)
			instance := backend.Open(t, clock)
			c.run(t, &Harness{Instance: instance, Clock: clock})
		})
	}
}

// allCases returns every conformance case in a stable order.
//
// Returns []testCase which lists the cases.
func allCases() []testCase {
	return slices.Concat(
		createCases(),
		dedupCases(),
		readCases(),
		atomicCases(),
		receiptCases(),
		semanticsCases(),
		durabilityCases(),
		casCases(),
		completionCases(),
		maintenanceCases(),
	)
}
