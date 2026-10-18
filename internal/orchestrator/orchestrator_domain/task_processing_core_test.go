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

	store := NewFakeTaskStore()
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	first := newDedupTask("first")
	require.NoError(t, core.PersistWithDedup(t.Context(), first))
	assert.True(t, first.Persisted(), "a created task is marked as persisted")

	second := newDedupTask("second")
	err := core.PersistWithDedup(t.Context(), second)
	require.ErrorIs(t, err, ErrDuplicateTask,
		"the duplicate must be reported to the caller before it can publish")
	assert.False(t, second.Persisted())
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
	assert.LessOrEqual(t, delay, time.Minute, "jitter never takes the delay past the cap")
	assert.GreaterOrEqual(t, delay, 30*time.Second, "jitter shortens the capped delay by at most half")
	assert.Equal(t, StatusRetrying, task.Status)
}

func TestTaskProcessingCore_TaskClock(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, (&TaskProcessingCore{}).taskClock(), "a core without a clock falls back to real time")

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Same(t, mockClock, (&TaskProcessingCore{Clock: mockClock}).taskClock())
}

func newRecoveryStore(orphans []*Task, listErr error) (*MockTaskStore, *[]TaskStatus) {
	store := &MockTaskStore{}
	var transitions []TaskStatus
	store.ListOrphanedTasksFunc = func(context.Context, OrphanQuery) ([]*Task, error) {
		return orphans, listErr
	}
	store.TransitionTaskFunc = func(_ context.Context, transition Transition) (TransitionOutcome, error) {
		transitions = append(transitions, transition.Task.Status)
		return TransitionOutcome{}, nil
	}
	return store, &transitions
}

func TestTaskProcessingCore_RecoverStaleTasks_ReturnsTasksToRetry(t *testing.T) {
	t.Parallel()

	retrying := &Task{ID: "stale-retry", Status: StatusProcessing, Attempt: 1}
	failed := &Task{ID: "stale-failed", Status: StatusProcessing, Attempt: 3}
	store, transitions := newRecoveryStore([]*Task{retrying, failed}, nil)
	config := DefaultDispatcherConfig()
	config.DefaultMaxRetries = 3
	core := NewTaskProcessingCore(config, nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	require.Len(t, retry, 1, "only tasks moved back to RETRYING need dispatching again")
	assert.Same(t, retrying, retry[0])
	assert.True(t, retry[0].Persisted(), "dispatching a recovered task must update its record, not insert it")
	assert.Equal(t, []TaskStatus{StatusRetrying, StatusFailed}, *transitions,
		"an orphan with retries left is retried; one whose retries are spent fails")
}

func TestTaskProcessingCore_RecoverStaleTasks_ListFailure(t *testing.T) {
	t.Parallel()

	store, _ := newRecoveryStore(nil, errors.New("read failed"))
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing orphaned tasks")
	assert.Empty(t, retry)
}

func TestTaskProcessingCore_RecoverStaleTasks_StoreWithoutClaims(t *testing.T) {
	t.Parallel()

	store := &MockTaskStore{}
	core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, store, nil)

	retry, err := core.RecoverStaleTasks(t.Context())
	require.NoError(t, err, "a store without the claim API has nothing to recover")
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

			if tc.wantNoCalls {
				assert.Empty(t, recorder.settledStatuses())
				return
			}
			assert.Equal(t, tc.want, recorder.settledStatuses(),
				"the hook runs once, after the terminal state is in the store")
		})
	}
}

func TestTaskProcessingCore_ApplyDefaults_StampsAdmission(t *testing.T) {
	t.Parallel()

	for _, now := range []time.Time{
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run("stamps unset times from the core clock at "+now.Format("2006"), func(t *testing.T) {
			t.Parallel()

			config := DefaultDispatcherConfig()
			config.DefaultMaxRetries = 2
			config.DefaultTimeout = 7 * time.Second
			core := NewTaskProcessingCore(config, nil, nil, clockpkg.NewMockClock(now))

			task := NewTask("test-executor", nil)
			core.ApplyDefaults(task)

			assert.Equal(t, now, task.ExecuteAt)
			assert.Equal(t, now, task.CreatedAt)
			assert.Equal(t, now, task.UpdatedAt)
			assert.Equal(t, 2, task.Config.MaxRetries)
			assert.Equal(t, 7*time.Second, task.Config.Timeout)
			assert.Equal(t, PriorityNormal, task.Config.Priority)
		})
	}

	t.Run("keeps times and settings the caller set", func(t *testing.T) {
		t.Parallel()

		core := NewTaskProcessingCore(DefaultDispatcherConfig(), nil, nil,
			clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		created := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

		task := NewTask("test-executor", nil)
		task.Status = StatusScheduled
		task.ExecuteAt = due
		task.CreatedAt = created
		task.Config.MaxRetries = 9
		task.Config.Timeout = time.Hour
		core.ApplyDefaults(task)

		assert.Equal(t, due, task.ExecuteAt)
		assert.Equal(t, created, task.CreatedAt)
		assert.Equal(t, StatusScheduled, task.Status)
		assert.Equal(t, 9, task.Config.MaxRetries)
		assert.Equal(t, time.Hour, task.Config.Timeout)
	})

	t.Run("falls back to the built-in timeout when the default is unset", func(t *testing.T) {
		t.Parallel()

		config := DefaultDispatcherConfig()
		config.DefaultTimeout = 0
		core := NewTaskProcessingCore(config, nil, nil, nil)

		task := NewTask("test-executor", nil)
		core.ApplyDefaults(task)

		assert.Equal(t, defaultTaskTimeout, task.Config.Timeout)
	})
}
