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

//go:build integration

package orchestrator_pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/events/events_provider_nats"
)

func startNATSContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	if url := os.Getenv("NATS_URL"); url != "" {
		return nil, url
	}

	request := testcontainers.ContainerRequest{
		Image:        "nats:2-alpine",
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor: wait.ForLog("Server is ready").
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "4222/tcp")
	require.NoError(t, err)

	return container, fmt.Sprintf("nats://%s:%s", host, port.Port())
}

type dispatcherNode struct {
	provider   *events_provider_nats.NATSProvider
	eventBus   orchestrator_domain.EventBus
	dispatcher orchestrator_domain.TaskDispatcher
	ctx        context.Context
	cancel     context.CancelCauseFunc
	wg         sync.WaitGroup
}

type distributedHarness struct {
	natsContainer testcontainers.Container
	natsURL       string
	taskStore     orchestrator_domain.TaskStore
	nodes         []*dispatcherNode
	executors     map[string]*controllableExecutor
	ctx           context.Context
	cancel        context.CancelFunc
	t             *testing.T
}

type distributedOption func(*distributedConfig)

type distributedConfig struct {
	dispatcherConfig orchestrator_domain.DispatcherConfig
	executors        map[string]*controllableExecutor
}

func withDistributedMaxRetries(n int) distributedOption {
	return func(config *distributedConfig) {
		config.dispatcherConfig.DefaultMaxRetries = n
	}
}

func withDistributedExecutor(name string, exec *controllableExecutor) distributedOption {
	return func(config *distributedConfig) {
		config.executors[name] = exec
	}
}

func withRecoveryInterval(d time.Duration) distributedOption {
	return func(config *distributedConfig) {
		config.dispatcherConfig.RecoveryInterval = d
	}
}

func withStaleTaskThreshold(d time.Duration) distributedOption {
	return func(config *distributedConfig) {
		config.dispatcherConfig.StaleTaskThreshold = d
	}
}

func newDistributedHarness(t *testing.T, nodeCount int, opts ...distributedOption) *distributedHarness {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping distributed orchestrator pipeline integration tests in short mode")
	}

	config := &distributedConfig{
		dispatcherConfig: orchestrator_domain.DispatcherConfig{
			DefaultTimeout:          30 * time.Second,
			DefaultMaxRetries:       1,
			RecoveryInterval:        0,
			StaleTaskThreshold:      2 * time.Second,
			HeartbeatInterval:       0,
			SyncPersistence:         true,
			WatermillHighHandlers:   2,
			WatermillNormalHandlers: 2,
			WatermillLowHandlers:    1,
			Clock:                   clockpkg.RealClock(),
		},
		executors: make(map[string]*controllableExecutor),
	}

	for _, opt := range opts {
		opt(config)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 120*time.Second, fmt.Errorf("test: integration test exceeded 120s timeout"))

	h := &distributedHarness{
		ctx:       ctx,
		cancel:    cancel,
		t:         t,
		executors: config.executors,
	}

	container, natsURL := startNATSContainer(ctx, t)
	h.natsContainer = container
	h.natsURL = natsURL

	dal, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{Capacity: 10000})
	require.NoError(t, err)
	h.taskStore = dal

	h.nodes = make([]*dispatcherNode, nodeCount)
	for i := range nodeCount {
		node := &dispatcherNode{}
		node.ctx, node.cancel = context.WithCancelCause(ctx)

		natsConfig := events_provider_nats.DefaultConfig()
		natsConfig.URL = natsURL
		natsConfig.ClusterID = fmt.Sprintf("test-node-%d", i)
		natsConfig.QueueGroupPrefix = "test-pipeline"
		natsConfig.JetStream.Disabled = true
		natsConfig.SubscribersCount = 1
		natsConfig.AckWaitTimeout = 10 * time.Second
		natsConfig.CloseTimeout = 5 * time.Second

		provider, err := events_provider_nats.NewNATSProvider(natsConfig)
		require.NoError(t, err, "creating NATS provider for node %d", i)
		node.provider = provider

		err = provider.Start(node.ctx)
		require.NoError(t, err, "starting NATS provider for node %d", i)

		eventBus := orchestrator_adapters.NewWatermillEventBus(
			provider.Publisher(),
			provider.Subscriber(),
			provider.Router(),
		)
		node.eventBus = eventBus

		dispatcher := orchestrator_adapters.CreateTaskDispatcher(
			context.Background(),
			config.dispatcherConfig,
			node.eventBus,
			h.taskStore,
		)
		require.NotNil(t, dispatcher, "CreateTaskDispatcher returned nil for node %d", i)
		node.dispatcher = dispatcher

		for name, exec := range config.executors {
			dispatcher.RegisterExecutor(context.Background(), name, exec)
		}

		node.wg.Go(func() {
			_ = dispatcher.Start(node.ctx)
		})

		h.nodes[i] = node
	}

	time.Sleep(500 * time.Millisecond)

	t.Cleanup(func() {

		for i := len(h.nodes) - 1; i >= 0; i-- {
			h.nodes[i].cancel(fmt.Errorf("test: cleanup"))
			h.nodes[i].wg.Wait()
			_ = h.nodes[i].provider.Close()
		}
		cancel()
		_ = dal.Close()
		if h.natsContainer != nil {
			_ = h.natsContainer.Terminate(context.Background())
		}
	})

	return h
}

func TestDistributed_CompetingConsumersProcessAllTasks(t *testing.T) {
	exec := newControllableExecutor()

	h := newDistributedHarness(t, 2,
		withDistributedMaxRetries(1),
		withDistributedExecutor("test-exec", exec),
	)

	const taskCount = 20

	for i := range taskCount {
		task := &orchestrator_domain.Task{
			ID:         fmt.Sprintf("competing-%d", i),
			WorkflowID: fmt.Sprintf("workflow-competing-%d", i),
			Executor:   "test-exec",
			Status:     orchestrator_domain.StatusPending,
			Config: orchestrator_domain.TaskConfig{
				Priority: orchestrator_domain.PriorityNormal,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := h.nodes[0].dispatcher.Dispatch(h.ctx, task)
		require.NoError(t, err, "dispatching task %d", i)
	}

	allProcessed := waitForCondition(30*time.Second, func() bool {
		return exec.getCallCount() >= taskCount
	})
	require.True(t, allProcessed,
		"expected %d executor calls, got %d", taskCount, exec.getCallCount())

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, taskCount, exec.getCallCount(),
		"each task should be processed exactly once")

	failures, err := h.nodes[0].dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	assert.Empty(t, failures, "no tasks should have failed")
}

func TestDistributed_StaleTaskRecovery(t *testing.T) {
	exec := newControllableExecutor()

	h := newDistributedHarness(t, 1,
		withDistributedMaxRetries(2),
		withDistributedExecutor("test-exec", exec),
		withRecoveryInterval(500*time.Millisecond),
		withStaleTaskThreshold(2*time.Second),
	)

	staleTask := &orchestrator_domain.Task{
		ID:         "stale-task-1",
		WorkflowID: "stale-workflow-1",
		Executor:   "test-exec",
		Status:     orchestrator_domain.StatusProcessing,
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
		Attempt:   0,
		CreatedAt: time.Now().Add(-10 * time.Minute),
		UpdatedAt: time.Now().Add(-10 * time.Minute),
	}
	err := h.taskStore.CreateTask(h.ctx, staleTask)
	require.NoError(t, err)

	failTask := &orchestrator_domain.Task{
		ID:         "stale-task-fail",
		WorkflowID: "stale-workflow-fail",
		Executor:   "test-exec",
		Status:     orchestrator_domain.StatusProcessing,
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
		Attempt:   1,
		CreatedAt: time.Now().Add(-10 * time.Minute),
		UpdatedAt: time.Now().Add(-10 * time.Minute),
	}
	err = h.taskStore.CreateTask(h.ctx, failTask)
	require.NoError(t, err)

	failDetected := waitForCondition(10*time.Second, func() bool {
		failures, _ := h.taskStore.ListFailedTasks(h.ctx)
		for _, f := range failures {
			if f.ID == "stale-task-fail" {
				return true
			}
		}
		return false
	})
	require.True(t, failDetected,
		"stale task with exhausted retries should be marked FAILED by recovery")

	failures, err := h.taskStore.ListFailedTasks(h.ctx)
	require.NoError(t, err)

	var foundFail bool
	for _, f := range failures {
		if f.ID == "stale-task-fail" {
			foundFail = true
			assert.Equal(t, orchestrator_domain.StatusFailed, f.Status)
			assert.Equal(t, 2, f.Attempt, "attempt should be incremented to maxRetries")
			assert.Contains(t, f.LastError, "task recovered")
		}
	}
	assert.True(t, foundFail, "stale-task-fail should appear in failed tasks")
}

func TestDistributed_DeduplicationAcrossNodes(t *testing.T) {
	exec := newControllableExecutor()

	h := newDistributedHarness(t, 2,
		withDistributedMaxRetries(1),
		withDistributedExecutor("test-exec", exec),
	)

	taskA := &orchestrator_domain.Task{
		ID:               "dedup-a",
		WorkflowID:       "dedup-workflow",
		Executor:         "test-exec",
		Status:           orchestrator_domain.StatusPending,
		DeduplicationKey: "shared-dedup-key",
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	taskB := &orchestrator_domain.Task{
		ID:               "dedup-b",
		WorkflowID:       "dedup-workflow-b",
		Executor:         "test-exec",
		Status:           orchestrator_domain.StatusPending,
		DeduplicationKey: "shared-dedup-key",
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	errA := h.nodes[0].dispatcher.Dispatch(h.ctx, taskA)
	errB := h.nodes[1].dispatcher.Dispatch(h.ctx, taskB)

	oneSucceeded := (errA == nil) != (errB == nil)
	if !oneSucceeded {

		if errA == nil && errB == nil {

			processed := waitForCondition(10*time.Second, func() bool {
				return exec.getCallCount() >= 2
			})
			if processed {
				assert.Equal(t, 2, exec.getCallCount(),
					"both tasks processed when first completed before second dispatch")
				return
			}
		}
	}

	var succeeded, failed error
	if errA == nil {
		succeeded, failed = errA, errB
	} else {
		succeeded, failed = errB, errA
	}
	assert.NoError(t, succeeded, "one dispatch should succeed")
	assert.True(t, errors.Is(failed, orchestrator_domain.ErrDuplicateTask),
		"other dispatch should fail with ErrDuplicateTask, got: %v", failed)

	processed := waitForCondition(10*time.Second, func() bool {
		return exec.getCallCount() >= 1
	})
	require.True(t, processed, "the non-duplicate task should be processed")

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.getCallCount(),
		"executor should be called exactly once due to deduplication")
}
