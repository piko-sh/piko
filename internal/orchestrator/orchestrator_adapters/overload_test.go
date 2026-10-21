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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

type countingExecutor struct {
	calls atomic.Int64
}

func (e *countingExecutor) Execute(context.Context, map[string]any) (map[string]any, error) {
	e.calls.Add(1)
	return nil, nil
}

func startOverloadDispatcher(t *testing.T, seed func(store orchestrator_domain.TaskStore)) (*watermillTaskDispatcher, orchestrator_domain.TaskStore, *countingExecutor) {
	t.Helper()
	dal, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{Capacity: 100_000})
	require.NoError(t, err)
	if seed != nil {
		seed(dal)
	}
	d, executor := startDispatcherOver(t, dal)
	t.Cleanup(func() { _ = dal.Close() })
	return d, dal, executor
}

func startDispatcherOver(t *testing.T, dal orchestrator_domain.TaskStore) (*watermillTaskDispatcher, *countingExecutor) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := events_provider_gochannel.NewGoChannelProvider(events_provider_gochannel.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, provider.Start(ctx))
	bus := NewWatermillEventBus(provider.Publisher(), provider.Subscriber(), provider.Router())

	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, bus, dal)
	executor := &countingExecutor{}
	d.RegisterExecutor(ctx, "count", executor)

	var running sync.WaitGroup
	running.Go(func() { _ = d.Start(ctx) })
	t.Cleanup(func() {
		cancel()
		running.Wait()
		_ = bus.Close(context.Background())
		_ = provider.Close()
	})
	return d, executor
}

func TestOverload_NoDispatchIsLost(t *testing.T) {
	const tasks = 3000
	d, store, executor := startOverloadDispatcher(t, nil)
	ctx := t.Context()

	for range tasks {
		task := orchestrator_domain.NewTask("count", nil)
		require.NoError(t, d.Dispatch(ctx, task), "a full bus must not refuse a dispatch")
	}

	require.Eventually(t, func() bool { return executor.calls.Load() == tasks }, time.Minute, 10*time.Millisecond,
		"every dispatched task runs")
	require.Eventually(t, func() bool { return d.Stats().TasksCompleted == tasks }, 10*time.Second, 10*time.Millisecond)

	stats := d.Stats()
	assert.Zero(t, stats.TasksFailed)
	assert.Zero(t, stats.HeldTasks)
	failed, err := store.ListFailedTasks(ctx)
	require.NoError(t, err)
	assert.Empty(t, failed, "no task was failed because the bus was full")
}

func TestStart_RunsTasksLeftPendingByEarlierRun(t *testing.T) {
	leftBehind := orchestrator_domain.NewTask("count", nil)
	now := time.Now()
	leftBehind.CreatedAt, leftBehind.UpdatedAt, leftBehind.ExecuteAt = now, now, now
	_, _, executor := startOverloadDispatcher(t, func(store orchestrator_domain.TaskStore) {
		require.NoError(t, store.CreateTask(context.Background(), leftBehind))
	})

	require.Eventually(t, func() bool { return executor.calls.Load() == 1 }, 10*time.Second, 10*time.Millisecond,
		"the task left pending by an earlier run is published when the dispatcher starts")
}

func TestStart_RerunsTaskClaimedByCrashedOwner(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	crashed, closeCrashed := openPersistedStore(t, dir)
	task := orchestrator_domain.NewTask("count", nil)
	task.DeduplicationKey = "artefact:crashed"
	now := time.Now()
	task.CreatedAt, task.UpdatedAt, task.ExecuteAt = now, now, now
	require.NoError(t, crashed.CreateTaskWithDedup(ctx, task))
	claimer, ok := crashed.(orchestrator_domain.TaskStoreV2)
	require.True(t, ok, "the persisted store must support claims")
	_, err := claimer.ClaimTask(ctx, orchestrator_domain.ClaimRequest{
		Now: now, ID: task.ID, Owner: "crashed-instance", Attempt: task.Attempt,
	})
	require.NoError(t, err, "the earlier owner claims the task")
	closeCrashed()

	store, closeStore := openPersistedStore(t, dir)
	t.Cleanup(closeStore)
	_, executor := startDispatcherOver(t, store)

	require.Eventually(t, func() bool { return executor.calls.Load() == 1 }, 10*time.Second, 10*time.Millisecond,
		"the next owner reruns the orphaned task at start")
	require.Eventually(t, func() bool {
		stored, err := store.GetTasksByID(ctx, []string{task.ID})
		return err == nil && len(stored) == 1 && stored[0].Status == orchestrator_domain.StatusComplete
	}, 10*time.Second, 10*time.Millisecond, "the rerun settles the task")

	stored, err := store.GetTasksByID(ctx, []string{task.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, stored[0].Attempt,
		"the crashed claim and the rerun each charge one attempt; recovery itself charges none")

	again := orchestrator_domain.NewTask("count", nil)
	again.DeduplicationKey = task.DeduplicationKey
	again.CreatedAt, again.UpdatedAt, again.ExecuteAt = now, now, now
	assert.NoError(t, store.CreateTaskWithDedup(ctx, again), "the settled task frees its deduplication key")
}

func openPersistedStore(t *testing.T, dir string) (orchestrator_domain.TaskStore, func()) {
	t.Helper()
	provider := persistence.NewProvider(persistence.Config{
		Persistence: &persistence.PersistenceProviderConfig{WALDir: dir, Enabled: true},
	})
	require.NoError(t, provider.Connect(context.Background()))
	factory, err := provider.OrchestratorDALFactory()
	require.NoError(t, err)
	dal, err := factory.NewOrchestratorDAL()
	require.NoError(t, err)
	store, ok := dal.(orchestrator_domain.TaskStore)
	require.True(t, ok)

	var once sync.Once
	return store, func() { once.Do(func() { _ = provider.Close(context.Background()) }) }
}

func TestSettleSignal_WakesAsTasksSettle(t *testing.T) {
	d, _, executor := startOverloadDispatcher(t, nil)
	signal := d.SettleSignal()
	select {
	case <-signal:
	default:
	}

	require.NoError(t, d.Dispatch(t.Context(), orchestrator_domain.NewTask("count", nil)))
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("no settle signal after the task ran")
	}
	assert.Equal(t, int64(1), executor.calls.Load())
}

func TestStart_AdoptsMoreLeftoversThanOneSweepPage(t *testing.T) {
	const leftovers = 2500
	now := time.Now().Add(-time.Minute)
	_, _, executor := startOverloadDispatcher(t, func(store orchestrator_domain.TaskStore) {
		for range leftovers {
			task := orchestrator_domain.NewTask("count", nil)
			task.CreatedAt, task.UpdatedAt, task.ExecuteAt = now, now, now
			require.NoError(t, store.CreateTask(context.Background(), task))
		}
	})

	require.Eventually(t, func() bool { return executor.calls.Load() == leftovers }, 10*time.Second, 10*time.Millisecond,
		"every task an earlier run left pending runs at start, well within one sweep interval")
}
