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

package orchestrator_domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

func TestSweepPendingPagesAcrossPassesWithoutRepeating(t *testing.T) {
	clock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	core, store := claimCoreWithClock(t, clock)
	const total = 250
	for range total {
		storeClaimTask(t, store, d.StatusPending, 0, clock.Now().Add(-time.Minute))
	}

	seen := map[string]int{}
	redispatch := func(_ context.Context, task *d.Task) error {
		seen[task.ID]++
		return nil
	}
	request := d.SweepRequest{Through: clock.Now(), Redispatch: redispatch, Budget: 100}
	var passes []int
	for {
		result, err := core.SweepPending(t.Context(), request)
		require.NoError(t, err)
		passes = append(passes, result.Swept)
		if result.Swept < request.Budget {
			break
		}
		request.After = result.Next
	}

	assert.Equal(t, []int{100, 100, 50}, passes, "each pass continues where the last stopped")
	assert.Len(t, seen, total, "every due task is swept")
	for id, times := range seen {
		assert.Equal(t, 1, times, "task %s swept once", id)
	}
}

func TestSweepPendingDeadBeforeExemptsAnEarlierProcessesTasks(t *testing.T) {
	clock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	core, store := claimCoreWithClock(t, clock)
	left := storeClaimTask(t, store, d.StatusPending, 0, clock.Now().Add(-time.Second))

	sweep := func(deadBefore time.Time) int {
		result, err := core.SweepPending(t.Context(), d.SweepRequest{
			LastActiveBefore: clock.Now().Add(-time.Minute),
			DeadBefore:       deadBefore,
			Redispatch:       func(context.Context, *d.Task) error { return nil },
			Budget:           10,
		})
		require.NoError(t, err)
		return result.Swept
	}

	assert.Zero(t, sweep(time.Time{}), "a task written within the age gate is left alone")
	assert.Equal(t, 1, sweep(left.UpdatedAt.Add(time.Second)),
		"a task written before this process started is swept at once: its message died with its publisher")
}
