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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

type executorFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)

func (f executorFunc) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	return f(ctx, payload)
}

type transportRig struct {
	dispatcher *watermillTaskDispatcher
	store      orchestrator_domain.TaskStore
	bus        *watermillEventBus
}

func startTransportRig(
	t *testing.T,
	mode orchestrator_domain.TransportMode,
	configure func(*orchestrator_domain.DispatcherConfig),
	executors map[string]orchestrator_domain.TaskExecutor,
) *transportRig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	provider, err := events_provider_gochannel.NewGoChannelProvider(events_provider_gochannel.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, provider.Start(ctx))
	bus := NewWatermillEventBus(provider.Publisher(), provider.Subscriber(), provider.Router())
	dal, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{Capacity: 100_000})
	require.NoError(t, err)

	config := orchestrator_domain.DefaultDispatcherConfig()
	config.Transport = mode
	config.SweepInterval = -1
	if configure != nil {
		configure(&config)
	}
	d := newWatermillTaskDispatcher(config, bus, dal)
	require.Equal(t, mode == orchestrator_domain.TransportLocal, d.localTransport)
	for name, executor := range executors {
		d.RegisterExecutor(ctx, name, executor)
	}

	var running sync.WaitGroup
	running.Go(func() { _ = d.Start(ctx) })
	t.Cleanup(func() {
		cancel()
		running.Wait()
		_ = bus.Close(context.Background())
		_ = provider.Close()
		_ = dal.Close()
	})
	watermillBus, ok := bus.(*watermillEventBus)
	require.True(t, ok)
	return &transportRig{dispatcher: d, store: dal, bus: watermillBus}
}

func (r *transportRig) dispatchTask(t *testing.T, executor string, priority orchestrator_domain.TaskPriority) *orchestrator_domain.Task {
	t.Helper()
	return r.dispatchPayload(t, executor, priority, nil)
}

func (r *transportRig) dispatchPayload(t *testing.T, executor string, priority orchestrator_domain.TaskPriority, payload map[string]any) *orchestrator_domain.Task {
	t.Helper()
	task := orchestrator_domain.NewTask(executor, payload)
	task.Config.Priority = priority
	require.NoError(t, r.dispatcher.Dispatch(t.Context(), task))
	return task
}

func (r *transportRig) status(t *testing.T, id string) orchestrator_domain.TaskStatus {
	t.Helper()
	rows, err := r.store.GetTasksByID(t.Context(), []string{id})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0].Status
}

func forEachTransport(t *testing.T, run func(t *testing.T, mode orchestrator_domain.TransportMode)) {
	for _, mode := range []orchestrator_domain.TransportMode{orchestrator_domain.TransportLocal, orchestrator_domain.TransportBus} {
		t.Run("transport="+mode.String(), func(t *testing.T) { run(t, mode) })
	}
}

func TestTransport_RunsEveryTaskOnce(t *testing.T) {
	forEachTransport(t, func(t *testing.T, mode orchestrator_domain.TransportMode) {
		const tasks = 500
		var runs [tasks]atomic.Int32
		count := executorFunc(func(_ context.Context, payload map[string]any) (map[string]any, error) {
			n, ok := payload["n"].(int)
			if !ok {
				return nil, errors.New("payload has no task number")
			}
			runs[n].Add(1)
			return nil, nil
		})
		r := startTransportRig(t, mode, nil, map[string]orchestrator_domain.TaskExecutor{"count": count})
		ids := make([]string, tasks)
		for i := range ids {
			ids[i] = r.dispatchPayload(t, "count", orchestrator_domain.TaskPriority(i%3), map[string]any{"n": i}).ID
		}
		require.Eventually(t, func() bool { return r.dispatcher.Stats().TasksCompleted == tasks }, 30*time.Second, 5*time.Millisecond)
		for i, id := range ids {
			assert.Equal(t, int32(1), runs[i].Load(), "task %d ran %d times", i, runs[i].Load())
			assert.Equal(t, orchestrator_domain.StatusComplete, r.status(t, id))
		}
	})
}

func TestTransport_BusyLowPriorityNeverStarvesHigh(t *testing.T) {
	forEachTransport(t, func(t *testing.T, mode orchestrator_domain.TransportMode) {
		gate := make(chan struct{})
		block := executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
			select {
			case <-gate:
			case <-ctx.Done():
			}
			return nil, nil
		})
		quick := executorFunc(func(context.Context, map[string]any) (map[string]any, error) { return nil, nil })
		r := startTransportRig(t, mode, func(c *orchestrator_domain.DispatcherConfig) {
			c.WatermillLowHandlers, c.WatermillNormalHandlers, c.WatermillHighHandlers = 1, 1, 1
		}, map[string]orchestrator_domain.TaskExecutor{"block": block, "quick": quick})
		t.Cleanup(func() { close(gate) })

		for range 3 {
			r.dispatchTask(t, "block", orchestrator_domain.PriorityLow)
		}
		high := r.dispatchTask(t, "quick", orchestrator_domain.PriorityHigh)
		require.Eventually(t, func() bool { return r.status(t, high.ID) == orchestrator_domain.StatusComplete },
			5*time.Second, 5*time.Millisecond, "a high-priority task runs while every low-priority worker is busy")
	})
}

func TestTransport_WorkersRunTasksConcurrently(t *testing.T) {
	forEachTransport(t, func(t *testing.T, mode orchestrator_domain.TransportMode) {
		const parallel = 4
		var running atomic.Int32
		allRunning := make(chan struct{})
		var once sync.Once
		together := executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
			if running.Add(1) == parallel {
				once.Do(func() { close(allRunning) })
			}
			select {
			case <-allRunning:
				return nil, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})
		r := startTransportRig(t, mode, func(c *orchestrator_domain.DispatcherConfig) {
			c.WatermillNormalHandlers = parallel
		}, map[string]orchestrator_domain.TaskExecutor{"together": together})

		for range parallel {
			r.dispatchTask(t, "together", orchestrator_domain.PriorityNormal)
		}
		select {
		case <-allRunning:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d tasks ran at once", running.Load(), parallel)
		}
		require.Eventually(t, func() bool { return r.dispatcher.Stats().TasksCompleted == parallel }, 5*time.Second, 5*time.Millisecond)
	})
}

func TestTransport_HungExecutorIsAbandonedAndItsWorkerReplaced(t *testing.T) {
	forEachTransport(t, func(t *testing.T, mode orchestrator_domain.TransportMode) {
		release := make(chan struct{})
		hung := executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
			<-release
			return nil, nil
		})
		quick := executorFunc(func(context.Context, map[string]any) (map[string]any, error) { return nil, nil })
		r := startTransportRig(t, mode, func(c *orchestrator_domain.DispatcherConfig) {
			c.WatermillNormalHandlers = 1
			c.DefaultMaxRetries = 1
			c.ExecutorAbandonGrace = 100 * time.Millisecond
		}, map[string]orchestrator_domain.TaskExecutor{"hung": hung, "quick": quick})
		t.Cleanup(func() { close(release) })

		stuck := orchestrator_domain.NewTask("hung", nil)
		stuck.Config.Timeout = 100 * time.Millisecond
		require.NoError(t, r.dispatcher.Dispatch(t.Context(), stuck))
		require.Eventually(t, func() bool { return r.status(t, stuck.ID) == orchestrator_domain.StatusFailed },
			5*time.Second, 10*time.Millisecond, "a task whose executor ignores its timeout is failed")

		next := r.dispatchTask(t, "quick", orchestrator_domain.PriorityNormal)
		require.Eventually(t, func() bool { return r.status(t, next.ID) == orchestrator_domain.StatusComplete },
			5*time.Second, 10*time.Millisecond, "the abandoned worker is replaced, so the priority keeps running tasks")
	})
}

func TestTransport_PoisonMessageDoesNotBlockItsTopic(t *testing.T) {
	quick := executorFunc(func(context.Context, map[string]any) (map[string]any, error) { return nil, nil })
	r := startTransportRig(t, orchestrator_domain.TransportBus, func(c *orchestrator_domain.DispatcherConfig) {
		c.WatermillNormalHandlers = 1
	}, map[string]orchestrator_domain.TaskExecutor{"quick": quick})

	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)
	poison := orchestrator_domain.Event{Type: orchestrator_domain.EventType(orchestrator_domain.TopicTaskDispatchNormal),
		Payload: map[string]any{"id": 42}}
	require.NoError(t, r.bus.Publish(t.Context(), orchestrator_domain.TopicTaskDispatchNormal, poison))

	task := r.dispatchTask(t, "quick", orchestrator_domain.PriorityNormal)
	require.Eventually(t, func() bool { return r.status(t, task.ID) == orchestrator_domain.StatusComplete },
		5*time.Second, 10*time.Millisecond, "a message that cannot be decoded is acknowledged, not retried forever")
}

func TestRecurring_DeadChainIsSeededAgain(t *testing.T) {
	var runs atomic.Int32
	once := executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
		runs.Add(1)
		return nil, nil
	})
	r := startTransportRig(t, orchestrator_domain.TransportLocal, func(c *orchestrator_domain.DispatcherConfig) {
		c.SweepInterval = 50 * time.Millisecond
	}, map[string]orchestrator_domain.TaskExecutor{"once": once})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)

	chain := orchestrator_domain.RecurringTask{Key: "chain", Executor: "once", Priority: orchestrator_domain.PriorityLow, Background: true}
	require.NoError(t, r.dispatcher.RegisterRecurring(t.Context(), chain))
	require.NoError(t, r.dispatcher.RegisterRecurring(t.Context(), chain), "registering a live chain again is not an error")

	require.Eventually(t, func() bool { return runs.Load() >= 2 }, 10*time.Second, 10*time.Millisecond,
		"a run that schedules no successor leaves the chain dead, so the sweep seeds it again")
}

func TestLease_LapseCancelsRunningTasksAndRegistersAgain(t *testing.T) {
	cancelled := make(chan error, 1)
	waits := executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		<-ctx.Done()
		cancelled <- context.Cause(ctx)
		return nil, ctx.Err()
	})
	r := startTransportRig(t, orchestrator_domain.TransportLocal, func(c *orchestrator_domain.DispatcherConfig) {
		c.HeartbeatInterval = 50 * time.Millisecond
	}, map[string]orchestrator_domain.TaskExecutor{"waits": waits})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)

	r.dispatchTask(t, "waits", orchestrator_domain.PriorityNormal)
	require.Eventually(t, func() bool { return r.dispatcher.pool.Running() == 1 }, 5*time.Second, time.Millisecond)
	require.NoError(t, r.store.DeregisterNode(t.Context(), r.dispatcher.InstanceID()), "the lease lapses")

	select {
	case cause := <-cancelled:
		assert.ErrorIs(t, cause, errNodeLeaseLost)
	case <-time.After(5 * time.Second):
		t.Fatal("a task kept running after its node's lease lapsed")
	}
	require.Eventually(t, func() bool {
		return r.store.RenewNode(t.Context(), orchestrator_domain.NodeLease{
			Instance: r.dispatcher.InstanceID(), ExpiresAt: time.Now().Add(time.Minute),
		}) == nil
	}, 5*time.Second, 10*time.Millisecond, "the node registers its lease again")
}

func TestLease_PeerRecoversTaskOfExpiredInstance(t *testing.T) {
	var runs atomic.Int32
	count := executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
		runs.Add(1)
		return nil, nil
	})
	r := startTransportRig(t, orchestrator_domain.TransportLocal, func(c *orchestrator_domain.DispatcherConfig) {
		c.RecoveryInterval = 50 * time.Millisecond
		c.StaleTaskThreshold = 100 * time.Millisecond
	}, map[string]orchestrator_domain.TaskExecutor{"count": count})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)

	past := time.Now().Add(-time.Minute)
	require.NoError(t, r.store.RegisterNode(t.Context(), orchestrator_domain.NodeLease{
		Instance: "dead-node", NodeID: "dead-node", StartedAt: past, RenewedAt: past, ExpiresAt: past.Add(time.Second),
	}))
	orphan := orchestrator_domain.NewTask("count", nil)
	orphan.Status = orchestrator_domain.StatusProcessing
	orphan.Attempt = 1
	orphan.OwnerInstance = "dead-node"
	orphan.CreatedAt, orphan.UpdatedAt, orphan.ExecuteAt = past, past, past
	require.NoError(t, r.store.CreateTask(t.Context(), orphan))

	require.Eventually(t, func() bool { return r.status(t, orphan.ID) == orchestrator_domain.StatusComplete },
		5*time.Second, 10*time.Millisecond, "a peer recovers and runs a task whose owner's lease lapsed")
	assert.Equal(t, int32(1), runs.Load())
}

func TestSweep_LeavesRecentlyQueuedTasksAlone(t *testing.T) {
	const queued = 200
	gate := make(chan struct{})
	held := executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		select {
		case <-gate:
		case <-ctx.Done():
		}
		return nil, nil
	})
	const interval = 200 * time.Millisecond
	r := startTransportRig(t, orchestrator_domain.TransportLocal, func(c *orchestrator_domain.DispatcherConfig) {
		c.SweepInterval = interval
		c.WatermillNormalHandlers = 1
	}, map[string]orchestrator_domain.TaskExecutor{"held": held})
	t.Cleanup(func() { close(gate) })
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	for range queued {
		r.dispatchTask(t, "held", orchestrator_domain.PriorityNormal)
	}
	dispatched := r.dispatcher.Stats().TasksDispatched
	time.Sleep(2*interval - 50*time.Millisecond)
	assert.Equal(t, dispatched, r.dispatcher.Stats().TasksDispatched,
		"no queued task is republished within two sweep intervals of its dispatch")
}

func TestTransport_BusCarriesBinaryReferences(t *testing.T) {
	var runs atomic.Int32
	r := startTransportRig(t, orchestrator_domain.TransportBus, nil, map[string]orchestrator_domain.TaskExecutor{
		"counts": executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
			runs.Add(1)
			return nil, nil
		}),
	})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)
	require.NotNil(t, r.dispatcher.rawBus, "a watermill bus carries raw payloads")

	sent := make(chan []byte, 1)
	require.NoError(t, r.bus.SubscribeRaw(t.Context(), orchestrator_domain.TopicTaskDispatchLow,
		func(_ context.Context, payload []byte) error {
			sent <- append([]byte(nil), payload...)
			return nil
		}))
	task := r.dispatchTask(t, "counts", orchestrator_domain.PriorityLow)

	select {
	case payload := <-sent:
		ref, err := orchestrator_domain.DecodeTaskRef(payload)
		require.NoError(t, err, "the bus carries the binary reference, not a JSON event")
		assert.Equal(t, task.ID, ref.ID)
	case <-time.After(5 * time.Second):
		t.Fatal("no task message was published")
	}
}

func TestTransport_BusRunsTaskFromAnOlderReleaseMessage(t *testing.T) {
	var runs atomic.Int32
	r := startTransportRig(t, orchestrator_domain.TransportBus, nil, map[string]orchestrator_domain.TaskExecutor{
		"counts": executorFunc(func(context.Context, map[string]any) (map[string]any, error) {
			runs.Add(1)
			return nil, nil
		}),
	})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)

	task := orchestrator_domain.NewTask("counts", nil)
	task.Config.Priority = orchestrator_domain.PriorityNormal
	require.NoError(t, r.store.CreateTask(t.Context(), task))
	legacy := map[string]any{
		"id": task.ID, "workflowID": task.WorkflowID, "executor": task.Executor, "attempt": task.Attempt,
		"status": string(task.Status), "config": map[string]any{"priority": int(task.Config.Priority)},
	}
	require.NoError(t, r.bus.Publish(t.Context(), orchestrator_domain.TopicTaskDispatchNormal, orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType(orchestrator_domain.TopicTaskDispatchNormal),
		Payload: legacy,
	}), "an older replica publishes the whole task as a JSON event")

	require.Eventually(t, func() bool {
		return runs.Load() == 1 && r.status(t, task.ID) == orchestrator_domain.StatusComplete
	}, 5*time.Second, 5*time.Millisecond, "a replica on this release still runs it")
}

func TestTransport_RerunOfAReleasedTaskRunsWithoutTheSweep(t *testing.T) {
	ran := make(chan map[string]any, 2)
	r := startTransportRig(t, orchestrator_domain.TransportLocal, nil, map[string]orchestrator_domain.TaskExecutor{
		"work": executorFunc(func(_ context.Context, payload map[string]any) (map[string]any, error) {
			ran <- payload
			return nil, nil
		}),
	})
	require.Eventually(t, func() bool { return r.dispatcher.started.Load() }, time.Second, time.Millisecond)

	checking, release := make(chan struct{}), make(chan struct{})
	first := orchestrator_domain.NewTask("work", map[string]any{"version": "first"})
	first.DeduplicationKey = "dedup:work"
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- r.dispatcher.DispatchIfRequired(t.Context(), first, func(context.Context) (bool, error) {
			close(checking)
			<-release
			return false, nil
		})
	}()
	<-checking

	second := orchestrator_domain.NewTask("work", map[string]any{"version": "second"})
	second.DeduplicationKey = "dedup:work"
	err := r.dispatcher.DispatchIfRequired(t.Context(), second, func(context.Context) (bool, error) { return true, nil })
	require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask, "the key is held, so a rerun is recorded on its task")

	close(release)
	require.ErrorIs(t, <-firstDone, orchestrator_domain.ErrTaskNotRequired)

	select {
	case payload := <-ran:
		assert.Equal(t, "second", payload["version"], "the recorded rerun runs")
	case <-time.After(5 * time.Second):
		t.Fatal("the rerun created when the unrequired task was released was left for the sweep")
	}
}
