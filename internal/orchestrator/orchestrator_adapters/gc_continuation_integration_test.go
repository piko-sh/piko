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

package orchestrator_adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestGCHintsKeepsOneSuccessorForFullAndEmptyBatches(t *testing.T) {
	for _, full := range []bool{false, true} {
		name := "empty"
		if full {
			name = "full"
		}
		t.Run(name, func(t *testing.T) {
			store := storedDispatcherTasks(t)
			registry := &registry_domain.MockRegistryService{
				PopGCHintsFunc: func(context.Context, int) ([]registry_dto.GCHint, error) {
					if full {
						return []registry_dto.GCHint{{BackendID: "store", StorageKey: "blob"}}, nil
					}
					return nil, nil
				},
				GetBlobStoreFunc: func(string) (registry_domain.BlobStore, error) { return &registry_domain.MockBlobStore{}, nil },
			}
			service := &d.MockOrchestratorService{}
			core := d.NewTaskProcessingCore(d.DefaultDispatcherConfig(), nil, store, nil)
			core.RegisterExecutor(t.Context(), ExecutorNameBlobGC, NewGCExecutor(registry, service))
			task := d.NewTask(ExecutorNameBlobGC, map[string]any{"mode": "hints", "batch_size": 1, "reschedule_seconds": 60})
			var successor *d.Task
			var predecessorAtSchedule d.TaskStatus
			core.DelayedPublisher = &d.MockDelayedPublisher{ScheduleFunc: func(ctx context.Context, scheduled *d.Task) error {
				successor = scheduled
				rows, err := store.GetTasksByID(ctx, []string{task.ID})
				require.NoError(t, err)
				predecessorAtSchedule = rows[0].Status
				return nil
			}}
			task.Status = d.StatusPending
			task.DeduplicationKey = "blob.gc.hints"
			require.NoError(t, store.CreateTaskWithDedup(t.Context(), task))
			require.NoError(t, core.ClaimDelivery(t.Context(), task))
			before := time.Now()
			pool := d.NewExecutionPool(d.ExecutionPoolConfig{}, nil, core.RunClaimed, core.HandleAbandonedExecution)
			slot, err := pool.Acquire(t.Context(), task.Config.Priority)
			require.NoError(t, err)
			slot.Dispatch(t.Context(), task)
			require.Equal(t, d.StatusComplete, task.Status)
			require.Zero(t, service.ScheduleCallCount.Load(), "the successor is created by the settle, not scheduled by the executor")
			require.NotNil(t, successor)
			require.Equal(t, d.StatusComplete, predecessorAtSchedule, "the successor waits for its predecessor's settlement")
			require.Equal(t, task.DeduplicationKey, successor.DeduplicationKey)
			require.False(t, successor.ScheduledExecuteAt.IsZero())
			if full {
				require.WithinDuration(t, before, successor.ScheduledExecuteAt, time.Second)
			} else {
				require.True(t, successor.ScheduledExecuteAt.After(before.Add(59*time.Second)))
			}
			require.True(t, successor.ExecuteAt.Equal(successor.ScheduledExecuteAt))
			core.HandleTaskSuccess(t.Context(), task, before)
			require.EqualValues(t, 1, core.DelayedPublisher.(*d.MockDelayedPublisher).ScheduleCallCount.Load())
			inspector, ok := store.(d.OrchestratorInspector)
			require.True(t, ok)
			rows, err := inspector.ListRecentTasks(t.Context(), 10)
			require.NoError(t, err)
			require.Len(t, rows, 2)
		})
	}
}
