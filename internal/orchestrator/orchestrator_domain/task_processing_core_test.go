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

package orchestrator_domain

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	clockpkg "piko.sh/piko/wdk/clock"
)

func newDedupTask(id string) *Task {
	return &Task{
		ID:               id,
		WorkflowID:       "wf-" + id,
		Executor:         "test-executor",
		Status:           StatusPending,
		DeduplicationKey: "artefact:profile",
		Payload:          map[string]any{},
	}
}

func TestTaskProcessingCore_PersistWithDedup_DecidesBeforeReturning(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		syncPersistence bool
	}{
		{name: "synchronous persistence", syncPersistence: true},
		{name: "asynchronous persistence", syncPersistence: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewFakeTaskStore()
			config := DefaultDispatcherConfig()
			config.SyncPersistence = tc.syncPersistence
			core := NewTaskProcessingCore(config, nil, store, nil)

			first := newDedupTask("first")
			require.NoError(t, core.PersistWithDedup(t.Context(), first))
			assert.True(t, first.Persisted(), "a created task is marked as persisted")

			second := newDedupTask("second")
			err := core.PersistWithDedup(t.Context(), second)
			require.ErrorIs(t, err, ErrDuplicateTask,
				"the duplicate must be reported to the caller before it can publish")
			assert.False(t, second.Persisted())
		})
	}
}

func TestTaskProcessingCore_PersistWithDedup_SkipsTaskAlreadyPersisted(t *testing.T) {
	t.Parallel()

	store := &MockTaskStore{}
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	task := newDedupTask("task")
	require.NoError(t, core.PersistWithDedup(t.Context(), task))
	require.NoError(t, core.PersistWithDedup(t.Context(), task))

	assert.Equal(t, int64(1), store.CreateTaskWithDedupCallCount.Load(),
		"a task whose record exists is not inserted a second time")
}

func TestTaskProcessingCore_PersistWithDedup_BoundsStoreWrite(t *testing.T) {
	t.Parallel()

	store := &MockTaskStore{
		CreateTaskWithDedupFunc: func(ctx context.Context, _ *Task) error {
			deadline, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				return errors.New("store write has no deadline")
			}
			if time.Until(deadline) > taskPersistTimeout {
				return errors.New("store write deadline exceeds the persist timeout")
			}
			return nil
		},
	}
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	require.NoError(t, core.PersistWithDedup(t.Context(), newDedupTask("task")))
}

func TestTaskProcessingCore_ReleaseUnrequiredTask(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))

	t.Run("frees the deduplication key synchronously", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()
		config := DefaultDispatcherConfig()
		config.SyncPersistence = false
		core := NewTaskProcessingCore(config, nil, store, mockClock)

		claimed := newDedupTask("claimed")
		claimed.LastError = "previous error"
		require.NoError(t, core.PersistWithDedup(t.Context(), claimed))

		require.NoError(t, core.ReleaseUnrequiredTask(t.Context(), claimed))
		assert.Equal(t, StatusComplete, claimed.Status)
		assert.Empty(t, claimed.LastError)
		assert.Equal(t, mockClock.Now(), claimed.UpdatedAt)

		require.NoError(t, core.PersistWithDedup(t.Context(), newDedupTask("later")),
			"a later dispatch with the same key is no longer blocked")
	})

	t.Run("reports a failed store write", func(t *testing.T) {
		t.Parallel()

		store := &MockTaskStore{
			UpdateTaskFunc: func(context.Context, *Task) error {
				return errors.New("store unavailable")
			},
		}
		core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, mockClock)

		err := core.ReleaseUnrequiredTask(t.Context(), newDedupTask("claimed"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "store unavailable")
	})

	t.Run("succeeds without a store", func(t *testing.T) {
		t.Parallel()

		core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil, mockClock)
		task := newDedupTask("claimed")

		require.NoError(t, core.ReleaseUnrequiredTask(t.Context(), task))
		assert.Equal(t, StatusComplete, task.Status)
	})
}

func TestTaskProcessingCore_AbandonUnpublishedTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		store          func() *MockTaskStore
		name           string
		wantStoreWrite bool
	}{
		{
			name:           "records the task as failed with the cause",
			store:          func() *MockTaskStore { return &MockTaskStore{} },
			wantStoreWrite: true,
		},
		{
			name: "tolerates a failed store write",
			store: func() *MockTaskStore {
				return &MockTaskStore{
					UpdateTaskFunc: func(context.Context, *Task) error {
						return errors.New("store unavailable")
					},
				}
			},
			wantStoreWrite: true,
		},
		{
			name:           "updates the task without a store",
			store:          nil,
			wantStoreWrite: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var store *MockTaskStore
			core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil, nil)
			if tc.store != nil {
				store = tc.store()
				core = NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)
			}

			task := newDedupTask("unpublished")
			core.AbandonUnpublishedTask(t.Context(), task, errors.New("bus closed"))

			assert.Equal(t, StatusFailed, task.Status)
			assert.Equal(t, "bus closed", task.LastError)
			if tc.wantStoreWrite {
				assert.Equal(t, int64(1), store.UpdateTaskCallCount.Load())
			}
		})
	}
}

func TestTaskProcessingCore_RecordUndeliverableTask(t *testing.T) {
	t.Parallel()

	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil, nil)
	core.TasksDispatched.Add(1)

	core.RecordUndeliverableTask(t.Context(), errors.New("missing task ID"))

	stats := core.Stats()
	assert.Equal(t, int64(1), stats.Failed)
	assert.Equal(t, stats.Dispatched, stats.Completed+stats.Failed+stats.Retried,
		"an undecodable task no longer counts as outstanding work")
}

func TestTaskProcessingCore_GetExecutor_UnknownExecutor(t *testing.T) {
	t.Parallel()

	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil, nil)

	_, err := core.GetExecutor("missing")
	require.ErrorIs(t, err, ErrExecutorNotFound)
	assert.Contains(t, err.Error(), "missing")
}

func TestTaskProcessingCore_ScheduleTaskRetry_CapsBackoff(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	config := DefaultDispatcherConfig()
	config.MaxRetryBackoff = time.Minute
	core := NewTaskProcessingCore(config, nil, nil, mockClock)

	task := newDedupTask("retry")
	task.Attempt = 12

	core.ScheduleTaskRetry(t.Context(), task, trace.SpanFromContext(t.Context()))

	delay := task.ScheduledExecuteAt.Sub(mockClock.Now())
	assert.GreaterOrEqual(t, delay, time.Minute)
	assert.Less(t, delay, time.Minute+time.Second, "only jitter may be added to the capped delay")
	assert.Equal(t, StatusRetrying, task.Status)
}

type executorFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)

func (f executorFunc) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	return f(ctx, payload)
}

func TestTaskProcessingCore_ExecuteTask_Outcomes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		executor    executorFunc
		wantResult  map[string]any
		name        string
		wantErr     string
		wantTimeout bool
	}{
		{
			name: "result is stored on the task",
			executor: func(context.Context, map[string]any) (map[string]any, error) {
				return map[string]any{"status": "ok"}, nil
			},
			wantResult: map[string]any{"status": "ok"},
		},
		{
			name: "executor error is returned",
			executor: func(context.Context, map[string]any) (map[string]any, error) {
				return nil, errors.New("compile failed")
			},
			wantErr: "compile failed",
		},
		{
			name: "executor panic becomes an error",
			executor: func(context.Context, map[string]any) (map[string]any, error) {
				panic("executor exploded")
			},
			wantErr: "panic in task executor: executor exploded",
		},
		{
			name: "executor that honours cancellation reports the timeout",
			executor: func(ctx context.Context, _ map[string]any) (map[string]any, error) {
				<-ctx.Done()
				return nil, context.Cause(ctx)
			},
			wantErr:     "task execution exceeded",
			wantTimeout: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil, nil)
			task := &Task{ID: "task", Payload: map[string]any{}}

			timeout := 5 * time.Second
			if tc.wantTimeout {
				timeout = 20 * time.Millisecond
			}

			err := core.ExecuteTask(t.Context(), task, tc.executor, timeout)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantResult, task.Result)
		})
	}
}

func TestTaskProcessingCore_ExecuteTask_AbandonsExecutorIgnoringCancellation(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	finished := make(chan struct{})
	t.Cleanup(func() {
		close(release)
		<-finished
	})

	config := DefaultDispatcherConfig()
	config.ExecutorAbandonGrace = 20 * time.Millisecond
	core := NewTaskProcessingCore(config, nil, nil, nil)

	stuck := executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
		defer close(finished)
		<-release
		return map[string]any{"late": true}, nil
	})
	task := &Task{ID: "stuck", Payload: map[string]any{}}

	err := core.ExecuteTask(t.Context(), task, stuck, 20*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "abandoned executor that ignored cancellation")
	assert.Contains(t, err.Error(), "task execution exceeded",
		"the abandonment carries the timeout as its cause")
	assert.Empty(t, task.Result)
}

func TestTaskProcessingCore_ExecuteTask_ExecutorFinishingDuringGraceIsUsed(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.ExecutorAbandonGrace = 5 * time.Second
	core := NewTaskProcessingCore(config, nil, nil, nil)

	slowToStop := executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		<-ctx.Done()
		return map[string]any{"partial": true}, errors.New("stopped after cancellation")
	})
	task := &Task{ID: "slow", Payload: map[string]any{}}

	err := core.ExecuteTask(t.Context(), task, slowToStop, 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after cancellation")
	assert.Equal(t, map[string]any{"partial": true}, task.Result)
}

func TestTaskProcessingCore_TaskClock(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, (&TaskProcessingCore{}).taskClock(), "a core without a clock falls back to real time")

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Same(t, mockClock, (&TaskProcessingCore{Clock: mockClock}).taskClock())
}

func newRecoveryStore(readBack []*Task, readErr error) *MockTaskStore {
	store := &MockTaskStore{}
	store.ClaimStaleTasksForRecoveryFunc = func(context.Context, string, time.Duration, time.Duration, int) ([]RecoveryClaimedTask, error) {
		return []RecoveryClaimedTask{{ID: "stale-retry"}, {ID: "stale-failed"}}, nil
	}
	store.RecoverClaimedTasksFunc = func(context.Context, string, int, string) (int, error) {
		return 2, nil
	}
	store.GetTasksByIDFunc = func(_ context.Context, ids []string) ([]*Task, error) {
		if readErr != nil {
			return nil, readErr
		}
		if len(ids) != 2 {
			return nil, errors.New("recovery must read back exactly the claimed tasks")
		}
		return readBack, nil
	}
	return store
}

func TestTaskProcessingCore_RecoverStaleTasks_ReturnsTasksToRetry(t *testing.T) {
	t.Parallel()

	retrying := &Task{ID: "stale-retry", Status: StatusRetrying, Attempt: 1}
	failed := &Task{ID: "stale-failed", Status: StatusFailed, Attempt: 3}
	store := newRecoveryStore([]*Task{retrying, failed}, nil)
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	require.Len(t, retry, 1, "only tasks moved back to RETRYING need dispatching again")
	assert.Same(t, retrying, retry[0])
	assert.True(t, retry[0].Persisted(), "dispatching a recovered task must update its record, not insert it")
}

func TestTaskProcessingCore_RecoverStaleTasks_ReadFailure(t *testing.T) {
	t.Parallel()

	store := newRecoveryStore(nil, errors.New("read failed"))
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading recovered tasks")
	assert.Empty(t, retry)
}

func TestTaskProcessingCore_RecoverStaleTasks_NothingStale(t *testing.T) {
	t.Parallel()

	store := &MockTaskStore{}
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	assert.Empty(t, retry)
	assert.Equal(t, int64(0), store.GetTasksByIDCallCount.Load())
}

type settledRecorder struct {
	store    *MockTaskStore
	statuses []TaskStatus
	settled  []TaskStatus
	mu       sync.Mutex
}

func newSettledRecorder(updateErr error) *settledRecorder {
	recorder := &settledRecorder{store: &MockTaskStore{}}
	recorder.store.UpdateTaskFunc = func(_ context.Context, task *Task) error {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		recorder.statuses = append(recorder.statuses, task.Status)
		return updateErr
	}
	return recorder
}

func (r *settledRecorder) hook(_ context.Context, task *Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.statuses) == 0 || r.statuses[len(r.statuses)-1] != task.Status {
		r.settled = append(r.settled, "NOT-WRITTEN-FIRST")
		return
	}
	r.settled = append(r.settled, task.Status)
}

func (r *settledRecorder) settledStatuses() []TaskStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.settled)
}

func TestTaskProcessingCore_OnTaskSettled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		updateErr   error
		settle      func(core *TaskProcessingCore, task *Task)
		name        string
		dedupKey    string
		want        []TaskStatus
		maxRetries  int
		wantNoCalls bool
	}{
		{
			name:     "success settles after the record is written",
			dedupKey: "artefact:profile",
			settle: func(core *TaskProcessingCore, task *Task) {
				core.HandleTaskSuccess(context.Background(), task, time.Now())
			},
			want: []TaskStatus{StatusComplete},
		},
		{
			name:       "terminal failure settles after the record is written",
			dedupKey:   "artefact:profile",
			maxRetries: 1,
			settle: func(core *TaskProcessingCore, task *Task) {
				core.HandleTaskFailure(context.Background(), task, errors.New("boom"), time.Now())
			},
			want: []TaskStatus{StatusFailed},
		},
		{
			name:       "retryable failure keeps the key and does not settle",
			dedupKey:   "artefact:profile",
			maxRetries: 3,
			settle: func(core *TaskProcessingCore, task *Task) {
				core.HandleTaskFailure(context.Background(), task, errors.New("boom"), time.Now())
			},
			wantNoCalls: true,
		},
		{
			name:     "release without running settles",
			dedupKey: "artefact:profile",
			settle: func(core *TaskProcessingCore, task *Task) {
				_ = core.ReleaseUnrequiredTask(context.Background(), task)
			},
			want: []TaskStatus{StatusComplete},
		},
		{
			name:     "abandoned publish settles",
			dedupKey: "artefact:profile",
			settle: func(core *TaskProcessingCore, task *Task) {
				core.AbandonUnpublishedTask(context.Background(), task, errors.New("bus closed"))
			},
			want: []TaskStatus{StatusFailed},
		},
		{
			name:      "abandoned publish whose record was not written does not settle",
			dedupKey:  "artefact:profile",
			updateErr: errors.New("store unavailable"),
			settle: func(core *TaskProcessingCore, task *Task) {
				core.AbandonUnpublishedTask(context.Background(), task, errors.New("bus closed"))
			},
			wantNoCalls: true,
		},
		{
			name:     "task without a deduplication key does not settle",
			dedupKey: "",
			settle: func(core *TaskProcessingCore, task *Task) {
				core.HandleTaskSuccess(context.Background(), task, time.Now())
			},
			wantNoCalls: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := newSettledRecorder(tc.updateErr)
			config := DefaultDispatcherConfig()
			config.SyncPersistence = false
			config.HeartbeatInterval = 0
			config.DefaultMaxRetries = max(tc.maxRetries, 1)
			core := NewTaskProcessingCore(config, nil, recorder.store, nil)
			core.DelayedPublisher = &MockDelayedPublisher{}
			core.OnTaskSettled = recorder.hook

			task := newDedupTask("task")
			task.DeduplicationKey = tc.dedupKey
			task.Attempt = 1
			task.Config.MaxRetries = config.DefaultMaxRetries

			tc.settle(core, task)
			require.NoError(t, core.Shutdown(t.Context()))

			if tc.wantNoCalls {
				assert.Empty(t, recorder.settledStatuses())
				return
			}
			assert.Equal(t, tc.want, recorder.settledStatuses(),
				"the hook runs once, after the terminal state is in the store")
		})
	}
}

func TestTaskProcessingCore_ReleaseInFlightTasks_LeavesRunningTaskAlone(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, store, nil)

	running := newDedupTask("running")
	core.PrepareTaskExecution(t.Context(), running)

	stored, ok := core.InFlightTasks.Load(running.ID)
	require.True(t, ok)
	snapshot, ok := stored.(*Task)
	require.True(t, ok)
	assert.NotSame(t, running, snapshot, "the in-flight entry is a snapshot, not the task its handler owns")

	core.ReleaseInFlightTasks(t.Context())

	assert.Equal(t, StatusProcessing, running.Status, "shutdown release never writes to the running task")
	assert.Equal(t, StatusPending, snapshot.Status)
	released, err := store.GetTasksByID(t.Context(), []string{running.ID})
	require.NoError(t, err)
	require.Len(t, released, 1)
	assert.Equal(t, StatusPending, released[0].Status)
}
