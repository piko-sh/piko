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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type poolExecutor func(ctx context.Context, payload map[string]any) (map[string]any, error)

func (f poolExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	return f(ctx, payload)
}

type poolRig struct {
	core      *d.TaskProcessingCore
	store     d.TaskStore
	pool      *d.ExecutionPool
	finished  chan *d.Task
	abandoned chan *d.Task
}

func startPoolRig(t *testing.T, configure func(*d.DispatcherConfig)) *poolRig {
	t.Helper()
	store, err := otter.NewOtterDAL(otter.Config{Capacity: 1000})
	require.NoError(t, err)
	cfg := d.DefaultDispatcherConfig()
	cfg.HeartbeatInterval = 0
	cfg.DefaultMaxRetries = 1
	if configure != nil {
		configure(&cfg)
	}
	rig := &poolRig{
		core:      d.NewTaskProcessingCore(cfg, nil, store, nil),
		store:     store,
		finished:  make(chan *d.Task, 8),
		abandoned: make(chan *d.Task, 8),
	}
	rig.pool = d.NewExecutionPool(d.ExecutionPoolConfig{
		WatchdogInterval: 5 * time.Millisecond,
		AbandonGrace:     cfg.EffectiveExecutorAbandonGrace(),
	}, nil, func(ctx context.Context, task *d.Task, slot *d.ExecutionSlot) {
		rig.core.RunClaimed(ctx, task, slot)
		rig.finished <- task
	}, func(ctx context.Context, task *d.Task, cause error, start time.Time) {
		rig.core.HandleAbandonedExecution(ctx, task, cause, start)
		rig.abandoned <- task
	})
	rig.pool.Start(context.Background())
	t.Cleanup(func() {
		assert.NoError(t, rig.stop())
		_ = store.Close()
	})
	return rig
}

func (r *poolRig) claim(t *testing.T, name string, executor d.TaskExecutor, configure func(*d.Task)) *d.Task {
	t.Helper()
	r.core.RegisterExecutor(t.Context(), name, executor)
	task := d.NewTask(name, nil)
	if configure != nil {
		configure(task)
	}
	require.NoError(t, r.store.CreateTask(t.Context(), task))
	r.core.TasksDispatched.Add(1)
	require.NoError(t, r.core.ClaimDelivery(t.Context(), task))
	return task
}

func (r *poolRig) run(t *testing.T, task *d.Task) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	slot, err := r.pool.Acquire(ctx, task.Config.Priority)
	require.NoError(t, err, "a worker is free")
	slot.Dispatch(ctx, task)
}

func (r *poolRig) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	grace, cancelGrace := context.WithCancel(context.Background())
	cancelGrace()
	return r.pool.Stop(ctx, grace, errors.New("test: pool stopped"))
}

func await(t *testing.T, tasks <-chan *d.Task, what string) *d.Task {
	t.Helper()
	select {
	case task := <-tasks:
		return task
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

func TestRunClaimed_Outcomes(t *testing.T) {
	testCases := []struct {
		executor    poolExecutor
		wantResult  map[string]any
		name        string
		wantErr     string
		timeout     time.Duration
		defaultTime time.Duration
	}{
		{
			name: "result is stored on the task",
			executor: func(context.Context, map[string]any) (map[string]any, error) {
				return map[string]any{"status": "ok"}, nil
			},
			wantResult: map[string]any{"status": "ok"},
		},
		{
			name: "executor error fails the attempt",
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
			timeout: 20 * time.Millisecond,
			wantErr: "task execution exceeded 20ms timeout",
		},
		{
			name: "a task without a timeout gets the default",
			executor: func(ctx context.Context, _ map[string]any) (map[string]any, error) {
				<-ctx.Done()
				return nil, context.Cause(ctx)
			},
			defaultTime: 30 * time.Millisecond,
			wantErr:     "task execution exceeded 30ms timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rig := startPoolRig(t, func(c *d.DispatcherConfig) {
				if tc.defaultTime > 0 {
					c.DefaultTimeout = tc.defaultTime
				}
			})
			task := rig.claim(t, "test", tc.executor, func(task *d.Task) { task.Config.Timeout = tc.timeout })

			rig.run(t, task)
			await(t, rig.finished, "the execution")

			stored := readTask(t, rig.store, task.ID)
			stats := rig.core.Stats()
			if tc.wantErr != "" {
				assert.Equal(t, d.StatusFailed, stored.Status)
				assert.Contains(t, stored.LastError, tc.wantErr)
				assert.Equal(t, int64(1), stats.Failed)
				assert.Zero(t, stats.Completed)
				return
			}
			assert.Equal(t, d.StatusComplete, stored.Status)
			assert.Equal(t, tc.wantResult, task.Result)
			assert.Equal(t, int64(1), stats.Completed)
			assert.Zero(t, stats.Failed)
		})
	}
}

func TestRunClaimed_AbandonsExecutorIgnoringCancellation(t *testing.T) {
	rig := startPoolRig(t, func(c *d.DispatcherConfig) { c.ExecutorAbandonGrace = 20 * time.Millisecond })

	release := make(chan struct{})
	returned := make(chan struct{})
	stuck := rig.claim(t, "stuck", poolExecutor(func(context.Context, map[string]any) (map[string]any, error) {
		defer close(returned)
		<-release
		return map[string]any{"late": true}, nil
	}), func(task *d.Task) { task.Config.Timeout = 20 * time.Millisecond })
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	rig.run(t, stuck)
	require.Same(t, stuck, await(t, rig.abandoned, "the abandonment"))
	stored := readTask(t, rig.store, stuck.ID)
	assert.Equal(t, d.StatusFailed, stored.Status, "the abandoned attempt is settled while its executor still runs")
	assert.Contains(t, stored.LastError, "abandoned executor that ignored cancellation")
	assert.Contains(t, stored.LastError, "task execution exceeded",
		"the abandonment carries the timeout as its cause")

	quick := rig.claim(t, "quick", poolExecutor(func(context.Context, map[string]any) (map[string]any, error) {
		return nil, nil
	}), nil)
	rig.run(t, quick)
	require.Same(t, quick, await(t, rig.finished, "the task on the replacement worker"),
		"the abandoned worker is replaced, so the priority keeps running tasks")

	close(release)
	<-returned
	require.Same(t, stuck, await(t, rig.finished, "the abandoned executor to return"))
	assert.Empty(t, stuck.Result, "a result arriving after the abandonment is dropped")
	assert.Equal(t, d.StatusFailed, readTask(t, rig.store, stuck.ID).Status)
	stats := rig.core.Stats()
	assert.Equal(t, int64(1), stats.Failed)
	assert.Equal(t, int64(1), stats.Completed)
}

func TestRunClaimed_ExecutorFinishingDuringGraceIsUsed(t *testing.T) {
	rig := startPoolRig(t, func(c *d.DispatcherConfig) { c.ExecutorAbandonGrace = 5 * time.Second })

	task := rig.claim(t, "slow", poolExecutor(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		<-ctx.Done()
		return map[string]any{"partial": true}, errors.New("stopped after cancellation")
	}), func(task *d.Task) { task.Config.Timeout = 10 * time.Millisecond })

	rig.run(t, task)
	await(t, rig.finished, "the execution")

	select {
	case <-rig.abandoned:
		t.Fatal("an executor that returns within the grace is not abandoned")
	default:
	}
	assert.Equal(t, map[string]any{"partial": true}, task.Result)
	stored := readTask(t, rig.store, task.ID)
	assert.Equal(t, d.StatusFailed, stored.Status)
	assert.Contains(t, stored.LastError, "stopped after cancellation")
	assert.Contains(t, stored.LastError, "task execution exceeded", "the error names the timeout that ended it")
}

func TestPoolStop_ReleasesRunningTaskToPending(t *testing.T) {
	rig := startPoolRig(t, nil)

	cancelled := make(chan error, 1)
	proceed := make(chan struct{})
	task := rig.claim(t, "waits", poolExecutor(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		<-ctx.Done()
		cancelled <- context.Cause(ctx)
		<-proceed
		return nil, ctx.Err()
	}), nil)
	claimedAttempt := task.Attempt

	rig.run(t, task)
	require.Eventually(t, func() bool { return rig.pool.Running() == 1 }, 5*time.Second, time.Millisecond)

	stopped := make(chan error, 1)
	go func() { stopped <- rig.stop() }()
	select {
	case cause := <-cancelled:
		assert.EqualError(t, cause, "test: pool stopped")
	case <-time.After(5 * time.Second):
		t.Fatal("stopping the pool did not cancel the running execution")
	}
	assert.Equal(t, d.StatusProcessing, readTask(t, rig.store, task.ID).Status,
		"nothing releases the task behind the back of an executor still running")

	close(proceed)
	await(t, rig.finished, "the execution")
	require.NoError(t, <-stopped)

	stored := readTask(t, rig.store, task.ID)
	assert.Equal(t, d.StatusPending, stored.Status, "an execution cut short by shutdown is released, not failed")
	assert.Equal(t, claimedAttempt, stored.Attempt, "the next start runs the same attempt again")
	assert.Empty(t, stored.OwnerInstance)
	stats := rig.core.Stats()
	assert.Zero(t, stats.Dispatched, "the release gives back the dispatch count")
	assert.Zero(t, stats.Failed)
	assert.Zero(t, stats.Completed)
}
