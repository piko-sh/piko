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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	rerunTestKey = "artefact:profile"
)

type gatedRerunExecutor struct {
	needed  *atomic.Bool
	gate    chan struct{}
	entered chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func newGatedRerunExecutor(needed *atomic.Bool) *gatedRerunExecutor {
	return &gatedRerunExecutor{
		needed:  needed,
		gate:    make(chan struct{}),
		entered: make(chan struct{}),
	}
}

func (e *gatedRerunExecutor) Execute(ctx context.Context, _ map[string]any) (map[string]any, error) {
	e.needed.Store(false)
	call := e.calls.Add(1)
	if call == 1 {
		e.once.Do(func() { close(e.entered) })
		select {
		case <-e.gate:
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
	return map[string]any{"call": call}, nil
}

func newKeyedTask(id string) *orchestrator_domain.Task {
	task := newHighPriorityTask(id)
	task.DeduplicationKey = rerunTestKey
	task.Payload = map[string]any{payloadKeyTaskID: id}
	return task
}

func newRerunTestDispatcher(t *testing.T) (*watermillTaskDispatcher, *mockEventBus) {
	t.Helper()

	store, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{Capacity: 1000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	config := newLifecycleTestConfig()
	config.SyncPersistence = false
	config.DefaultMaxRetries = 1
	eventBus := newMockEventBus()
	return newWatermillTaskDispatcher(config, eventBus, store), eventBus
}

func requiredWhile(needed *atomic.Bool) orchestrator_domain.DispatchRequirement {
	return func(context.Context) (bool, error) {
		return needed.Load(), nil
	}
}

func TestWatermillTaskDispatcher_CoalescesReruns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		changesDuringRun  int
		coveredBeforeDone bool
		wantExecutions    int32
	}{
		{name: "no change during the run needs no rerun", changesDuringRun: 0, wantExecutions: 1},
		{name: "a change during the run reruns exactly once", changesDuringRun: 1, wantExecutions: 2},
		{name: "several changes during the run still rerun exactly once", changesDuringRun: 4, wantExecutions: 2},
		{name: "a rerun no longer required is skipped", changesDuringRun: 1, coveredBeforeDone: true, wantExecutions: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d, _ := newRerunTestDispatcher(t)
			var needed atomic.Bool
			executor := newGatedRerunExecutor(&needed)
			d.RegisterExecutor(t.Context(), "test-executor", executor)
			runDispatcher(t, d)
			require.Eventually(t, func() bool {
				return dispatcherPhaseOf(d) == dispatcherPhaseRunning
			}, 5*time.Second, 5*time.Millisecond)

			needed.Store(true)
			require.NoError(t, d.DispatchIfRequired(t.Context(), newKeyedTask("first"), requiredWhile(&needed)))
			waitForTestSignal(t, executor.entered)

			for change := range tc.changesDuringRun {
				needed.Store(true)
				err := d.DispatchIfRequired(t.Context(), newKeyedTask(fmt.Sprintf("change-%d", change)), requiredWhile(&needed))
				require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask,
					"a dispatch while the key is in use is coalesced, not run alongside")
			}
			assert.Equal(t, min(tc.changesDuringRun, 1), d.reruns.size(), "one pending rerun at most")

			if tc.coveredBeforeDone {
				needed.Store(false)
			}
			close(executor.gate)

			require.Eventually(t, func() bool {
				return d.IsIdle() && d.Stats().TasksCompleted == int64(tc.wantExecutions)
			}, 5*time.Second, 5*time.Millisecond)

			stats := d.Stats()
			assert.Equal(t, tc.wantExecutions, executor.calls.Load())
			assert.Equal(t, int64(tc.wantExecutions), stats.TasksDispatched)
			assert.Equal(t, int64(0), stats.TasksFailed)
			assert.Equal(t, 0, d.reruns.size(), "no rerun is left pending once the key is idle")
		})
	}
}

func TestWatermillTaskDispatcher_coalesceRerun(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		secondClaimErr error
		wantErr        error
		required       orchestrator_domain.DispatchRequirement
		name           string
		wantPublished  int
		wantClaims     int64
		wantPending    int
		limitReached   bool
	}{
		{
			name:          "key released before the request was recorded runs at once",
			required:      func(context.Context) (bool, error) { return true, nil },
			wantPublished: 1,
			wantClaims:    2,
		},
		{
			name:           "key still in use keeps the request pending",
			required:       func(context.Context) (bool, error) { return true, nil },
			secondClaimErr: orchestrator_domain.ErrDuplicateTask,
			wantErr:        orchestrator_domain.ErrDuplicateTask,
			wantClaims:     2,
			wantPending:    1,
		},
		{
			name:       "work covered by the released key leaves nothing pending",
			required:   func(context.Context) (bool, error) { return false, nil },
			wantErr:    orchestrator_domain.ErrTaskNotRequired,
			wantClaims: 3,
		},
		{
			name:           "store failure on the second claim keeps the request pending",
			required:       func(context.Context) (bool, error) { return true, nil },
			secondClaimErr: errors.New("store unavailable"),
			wantClaims:     2,
			wantPending:    1,
		},
		{
			name:         "request dropped once the pending rerun limit is reached",
			required:     func(context.Context) (bool, error) { return true, nil },
			limitReached: true,
			wantErr:      orchestrator_domain.ErrDuplicateTask,
			wantClaims:   1,
			wantPending:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var claims atomic.Int32
			store := &orchestrator_domain.MockTaskStore{
				CreateTaskWithDedupFunc: func(context.Context, *orchestrator_domain.Task) error {
					if claims.Add(1) == 1 {
						return orchestrator_domain.ErrDuplicateTask
					}
					return tc.secondClaimErr
				},
			}
			config := newLifecycleTestConfig()
			config.MaxPendingReruns = 1
			eventBus := newMockEventBus()
			d := newWatermillTaskDispatcher(config, eventBus, store)
			startPublishing(t, d)
			if tc.limitReached {
				require.True(t, d.reruns.request("another-key", &rerunRequest{task: newKeyedTask("other"), required: tc.required}))
			}

			err := d.DispatchIfRequired(t.Context(), newKeyedTask("task"), tc.required)
			switch {
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			case tc.secondClaimErr != nil:
				require.ErrorIs(t, err, tc.secondClaimErr)
			default:
				require.NoError(t, err)
			}

			assert.Len(t, eventBus.getPublishedEvents(), tc.wantPublished)
			assert.Equal(t, tc.wantClaims, store.CreateTaskWithDedupCallCount.Load())
			assert.Equal(t, tc.wantPending, d.reruns.size())
		})
	}
}

func TestWatermillTaskDispatcher_onTaskSettled(t *testing.T) {
	t.Parallel()

	t.Run("nothing pending does nothing", func(t *testing.T) {
		t.Parallel()

		store := &orchestrator_domain.MockTaskStore{}
		d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store)

		d.onTaskSettled(t.Context(), newKeyedTask("settled"))
		assert.Equal(t, int64(0), store.CreateTaskWithDedupCallCount.Load())
	})

	t.Run("pending rerun is held for the publisher", func(t *testing.T) {
		t.Parallel()

		store := &orchestrator_domain.MockTaskStore{}
		eventBus := newMockEventBus()
		d := newWatermillTaskDispatcher(newLifecycleTestConfig(), eventBus, store)
		startPublishing(t, d)
		require.True(t, d.reruns.request(rerunTestKey, &rerunRequest{
			task:     newKeyedTask("rerun"),
			required: func(context.Context) (bool, error) { return true, nil },
		}))

		d.onTaskSettled(t.Context(), newKeyedTask("settled"))

		assert.Empty(t, eventBus.getPublishedEvents(), "a rerun dispatched from a handler is not published inline")
		assert.Equal(t, int64(1), d.Stats().TasksDispatched, "the held rerun already counts as dispatched")
		assert.False(t, d.IsIdle())
		select {
		case <-d.heldWake:
		default:
			require.FailNow(t, "the held-task publisher was not woken")
		}
		d.publishHeld(t.Context())
		assert.Len(t, eventBus.getPublishedEvents(), 1)
		assert.Equal(t, 0, d.reruns.size())
	})

	t.Run("key taken again keeps the rerun pending", func(t *testing.T) {
		t.Parallel()

		store := &orchestrator_domain.MockTaskStore{
			CreateTaskWithDedupFunc: func(context.Context, *orchestrator_domain.Task) error {
				return orchestrator_domain.ErrDuplicateTask
			},
		}
		d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store)
		startPublishing(t, d)
		require.True(t, d.reruns.request(rerunTestKey, &rerunRequest{
			task:     newKeyedTask("rerun"),
			required: func(context.Context) (bool, error) { return true, nil },
		}))

		d.onTaskSettled(t.Context(), newKeyedTask("settled"))
		assert.Equal(t, 1, d.reruns.size())
	})

	t.Run("rerun on a stopped dispatcher is dropped", func(t *testing.T) {
		t.Parallel()

		d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), &orchestrator_domain.MockTaskStore{})
		d.failBacklog(t.Context(), orchestrator_domain.ErrDispatcherStopped)
		require.True(t, d.reruns.request(rerunTestKey, &rerunRequest{
			task:     newKeyedTask("rerun"),
			required: func(context.Context) (bool, error) { return true, nil },
		}))

		d.onTaskSettled(t.Context(), newKeyedTask("settled"))
		assert.Equal(t, 0, d.reruns.size())
		assert.Equal(t, int64(0), d.Stats().TasksDispatched)
	})
}

func TestRerunCoalescer(t *testing.T) {
	t.Parallel()

	first := &rerunRequest{task: newKeyedTask("first")}
	second := &rerunRequest{task: newKeyedTask("second")}
	other := &rerunRequest{task: newKeyedTask("other")}

	coalescer := newRerunCoalescer(2)
	require.True(t, coalescer.request("a", first))
	require.True(t, coalescer.request("a", second), "a newer request replaces the pending one")
	require.True(t, coalescer.request("b", other))
	assert.False(t, coalescer.request("c", first), "a new key is refused at the limit")
	assert.Equal(t, 2, coalescer.size())

	coalescer.withdraw("a", first)
	assert.Equal(t, 2, coalescer.size(), "withdrawing a replaced request leaves the newer one")

	taken, ok := coalescer.take("a")
	require.True(t, ok)
	assert.Same(t, second, taken)
	_, ok = coalescer.take("a")
	assert.False(t, ok)

	coalescer.withdraw("b", other)
	assert.Equal(t, 0, coalescer.size())

	require.True(t, coalescer.request("b", other))
	coalescer.forget("b")
	assert.Equal(t, 0, coalescer.size())
}

func TestNewRerunTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		payloadTaskID string
		wantSameIDKey bool
	}{
		{name: "payload carrying the task ID follows the new ID", payloadTaskID: "original", wantSameIDKey: true},
		{name: "payload carrying another ID is left alone", payloadTaskID: "something-else", wantSameIDKey: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := newKeyedTask("original")
			original.Payload = map[string]any{payloadKeyTaskID: tc.payloadTaskID, "artefactID": "a"}
			original.BuildTag = "build-1"
			original.Config.MaxRetries = 7

			rerun := newRerunTask(original)

			assert.NotEqual(t, original.ID, rerun.ID)
			assert.Equal(t, original.WorkflowID, rerun.WorkflowID)
			assert.Equal(t, original.Executor, rerun.Executor)
			assert.Equal(t, original.Config, rerun.Config)
			assert.Equal(t, original.DeduplicationKey, rerun.DeduplicationKey)
			assert.Equal(t, original.BuildTag, rerun.BuildTag)
			assert.Equal(t, "a", rerun.Payload["artefactID"])
			assert.False(t, rerun.Persisted())
			if tc.wantSameIDKey {
				assert.Equal(t, rerun.ID, rerun.Payload[payloadKeyTaskID])
			} else {
				assert.Equal(t, tc.payloadTaskID, rerun.Payload[payloadKeyTaskID])
			}
			assert.Equal(t, tc.payloadTaskID, original.Payload[payloadKeyTaskID], "the original payload is untouched")
		})
	}
}

func waitForTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for signal")
	}
}
