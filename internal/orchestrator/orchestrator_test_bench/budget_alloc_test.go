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
	"context"
	"testing"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/testutil/allocbudget"
)

func TestAllocBudget_Lifecycle(t *testing.T) {
	applyLogMode(t, "off")
	r := newRig(t, rigOptions{store: storeOtterMem, transport: transportGoChannel})

	allocbudget.Assert(t, allocbudget.Budget{Allocs: 19, Bytes: 4096, Runs: 200}, func() {
		base := r.settledCount()
		if err := r.dispatcher.Dispatch(r.ctx, newBenchTask()); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if _, err := r.completions.await(1, base); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAllocBudget_StoreCycle(t *testing.T) {
	applyLogMode(t, "off")
	ctx := context.Background()
	store, closeStore := newStore(t, storeOtterMem)
	t.Cleanup(closeStore)

	config := orchestrator_domain.DefaultDispatcherConfig()
	config.NodeID = "budget"
	core := orchestrator_domain.NewTaskProcessingCore(config, nil, store, nil)

	allocbudget.Assert(t, allocbudget.Budget{Allocs: 18, Bytes: 4 << 10, Runs: 500}, func() {
		task := newBenchTask()
		task.DeduplicationKey = task.ID
		core.ApplyDefaults(task)
		if err := core.PersistWithDedup(ctx, task); err != nil {
			t.Fatalf("admit: %v", err)
		}
		if err := core.ClaimDelivery(ctx, task); err != nil {
			t.Fatalf("claim: %v", err)
		}
		core.HandleTaskSuccess(ctx, task, time.Time{})
		if task.Status != orchestrator_domain.StatusComplete {
			t.Fatalf("settled as %s", task.Status)
		}
	})
}

func TestAllocBudget_HotPathLog(t *testing.T) {
	applyLogMode(t, "off")
	logger := logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_test_bench")
	ctx := context.Background()

	t.Run("op=trace", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 1}, func() {
			logger.Trace("Task completed successfully",
				logger_domain.String("taskID", "task"),
				logger_domain.Int("attempt", 1))
		})
	})
	t.Run("op=span", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 15}, func() {
			_, span, _ := logger.Span(ctx, "TaskProcessingCore.handleTaskSuccess",
				logger_domain.String("taskID", "task"),
				logger_domain.String("workflowID", "task"))
			span.End()
		})
	})
}
