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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

type pipelineHarness struct {
	eventBusProvider *events_provider_gochannel.GoChannelProvider
	eventBus         orchestrator_domain.EventBus
	taskStore        orchestrator_domain.TaskStore
	dispatcher       orchestrator_domain.TaskDispatcher
	bridge           *orchestrator_adapters.ArtefactWorkflowBridge
	registryService  registry_domain.RegistryService
	executors        map[string]*controllableExecutor
	clock            *clockpkg.MockClock
	ctx              context.Context
	cancel           context.CancelFunc
	t                *testing.T
	wg               sync.WaitGroup
}

type controllableExecutor struct {
	mu            sync.Mutex
	callCount     int
	failUntil     int
	alwaysFail    bool
	failError     string
	executionTime time.Duration
	calls         []executorCall
}

type executorCall struct {
	Payload   map[string]any
	Timestamp time.Time
}

func newControllableExecutor() *controllableExecutor {
	return &controllableExecutor{
		failError: "executor failed",
	}
}

func (e *controllableExecutor) Execute(_ context.Context, payload map[string]any) (map[string]any, error) {
	e.mu.Lock()
	e.callCount++
	attempt := e.callCount
	e.calls = append(e.calls, executorCall{
		Payload:   payload,
		Timestamp: time.Now(),
	})
	shouldFail := e.alwaysFail || attempt <= e.failUntil
	execTime := e.executionTime
	failErr := e.failError
	e.mu.Unlock()

	if execTime > 0 {
		time.Sleep(execTime)
	}

	if shouldFail {
		return nil, fmt.Errorf("%s (attempt %d)", failErr, attempt)
	}

	return map[string]any{"status": "success"}, nil
}

func (e *controllableExecutor) getCallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.callCount
}

type harnessOption func(*harnessConfig)

type harnessConfig struct {
	dispatcherConfig orchestrator_domain.DispatcherConfig
	executors        map[string]*controllableExecutor
	clock            clockpkg.Clock
}

func withMaxRetries(n int) harnessOption {
	return func(config *harnessConfig) {
		config.dispatcherConfig.DefaultMaxRetries = n
	}
}

func withExecutor(name string, exec *controllableExecutor) harnessOption {
	return func(config *harnessConfig) {
		config.executors[name] = exec
	}
}

func withClock(c clockpkg.Clock) harnessOption {
	return func(config *harnessConfig) {
		config.clock = c
	}
}

func newPipelineHarness(t *testing.T, opts ...harnessOption) *pipelineHarness {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping orchestrator pipeline integration tests in short mode")
	}

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	config := &harnessConfig{
		dispatcherConfig: orchestrator_domain.DispatcherConfig{
			DefaultTimeout:          30 * time.Second,
			DefaultMaxRetries:       3,
			RecoveryInterval:        0,
			StaleTaskThreshold:      10 * time.Minute,
			HeartbeatInterval:       0,
			SyncPersistence:         true,
			WatermillHighHandlers:   2,
			WatermillNormalHandlers: 2,
			WatermillLowHandlers:    1,
			Clock:                   mockClock,
		},
		executors: make(map[string]*controllableExecutor),
	}

	for _, opt := range opts {
		opt(config)
	}

	var harnessClock *clockpkg.MockClock
	if config.clock != nil {
		config.dispatcherConfig.Clock = config.clock
	} else {
		harnessClock = mockClock
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 60*time.Second, fmt.Errorf("test: integration test exceeded 60s timeout"))

	h := &pipelineHarness{
		ctx:       ctx,
		cancel:    cancel,
		t:         t,
		executors: config.executors,
		clock:     harnessClock,
	}

	provider, err := events_provider_gochannel.NewGoChannelProvider(
		events_provider_gochannel.Config{
			OutputChannelBuffer:            256,
			Persistent:                     false,
			BlockPublishUntilSubscriberAck: false,
		},
	)
	require.NoError(t, err)
	h.eventBusProvider = provider

	err = provider.Start(ctx)
	require.NoError(t, err)

	eventBus := orchestrator_adapters.NewWatermillEventBus(
		provider.Publisher(),
		provider.Subscriber(),
		provider.Router(),
	)
	h.eventBus = eventBus

	dal, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{Capacity: 10000})
	require.NoError(t, err)
	h.taskStore = dal

	h.dispatcher = orchestrator_adapters.CreateTaskDispatcher(
		context.Background(),
		config.dispatcherConfig,
		h.eventBus,
		h.taskStore,
	)
	require.NotNil(t, h.dispatcher, "CreateTaskDispatcher returned nil")

	for name, exec := range config.executors {
		h.dispatcher.RegisterExecutor(context.Background(), name, exec)
	}

	metaStore, err := registry_otter.NewOtterDAL(registry_otter.Config{Capacity: 10000})
	require.NoError(t, err)

	tempDir := t.TempDir()
	blobDir := filepath.Join(tempDir, "blobs")
	require.NoError(t, os.MkdirAll(blobDir, 0o755))
	diskProvider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: blobDir,
	})
	require.NoError(t, err)
	blobStore, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   diskProvider,
		Repository: "",
	})
	require.NoError(t, err)

	h.registryService = registry_domain.NewRegistryService(
		metaStore,
		map[string]registry_domain.BlobStore{
			"test_store": blobStore,
		},
		h.eventBus,
		nil,
	)

	h.bridge = orchestrator_adapters.NewArtefactWorkflowBridge(
		h.registryService,
		nil,
		h.dispatcher,
		h.eventBus,
	)

	wait, err := h.bridge.StartListening(ctx, h.eventBus)
	require.NoError(t, err)
	h.wg.Go(func() {
		wait()
	})

	h.wg.Go(func() {
		_ = h.dispatcher.Start(ctx)
	})

	time.Sleep(100 * time.Millisecond)

	t.Cleanup(func() {
		cancel()
		h.wg.Wait()
		_ = provider.Close()
		_ = dal.Close()
		_ = metaStore.Close()
	})

	return h
}

func (h *pipelineHarness) waitForIdle(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.dispatcher.IsIdle() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func (h *pipelineHarness) advancePastRetry(attempt int) bool {
	h.t.Helper()
	baseline := h.clock.TimerCount()
	if !h.clock.AwaitTimerSetup(baseline, 5*time.Second) {
		return false
	}

	backoff := time.Duration(1) * time.Second
	for range attempt {
		backoff *= 10
	}
	h.clock.Advance(backoff + 2*time.Second)
	return true
}

func (h *pipelineHarness) advanceUntilIdle(timeout time.Duration) bool {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.dispatcher.IsIdle() {
			return true
		}

		h.clock.Advance(12 * time.Second)

		time.Sleep(100 * time.Millisecond)
	}
	return h.dispatcher.IsIdle()
}

func (h *pipelineHarness) waitForFlush(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		published := h.registryService.ArtefactEventsPublished()
		if published > 0 && h.bridge.ArtefactEventsProcessed() >= published {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func (h *pipelineHarness) dispatchTask(executor string, priority orchestrator_domain.TaskPriority) error {
	task := &orchestrator_domain.Task{
		ID:         fmt.Sprintf("task-%d", time.Now().UnixNano()),
		WorkflowID: fmt.Sprintf("workflow-%d", time.Now().UnixNano()),
		Executor:   executor,
		Config: orchestrator_domain.TaskConfig{
			Priority: priority,
		},
	}
	return h.dispatcher.Dispatch(h.ctx, task)
}

func (h *pipelineHarness) seedArtefact(artefactID string, profiles []registry_dto.NamedProfile) {
	h.t.Helper()
	h.seedArtefactWithContent(artefactID, []byte("test content"), profiles)
}

func (h *pipelineHarness) seedArtefactWithContent(artefactID string, content []byte, profiles []registry_dto.NamedProfile) {
	h.t.Helper()
	_, err := h.registryService.UpsertArtefact(
		h.ctx,
		artefactID,
		artefactID,
		bytes.NewReader(content),
		"test_store",
		profiles,
	)
	require.NoError(h.t, err)
}

func (h *pipelineHarness) waitUntilIdle(flushTimeout, idleTimeout time.Duration) (flushed bool, idle bool) {
	h.t.Helper()

	flushed = h.waitForFlush(flushTimeout)
	if !flushed {
		return false, false
	}

	idle = h.waitForIdle(idleTimeout)
	return flushed, idle
}

func withProductionConfig() harnessOption {
	return func(config *harnessConfig) {
		prod := orchestrator_domain.DefaultDispatcherConfig()
		config.dispatcherConfig.DefaultTimeout = prod.DefaultTimeout
		config.dispatcherConfig.DefaultMaxRetries = prod.DefaultMaxRetries
		config.dispatcherConfig.RecoveryInterval = prod.RecoveryInterval
		config.dispatcherConfig.StaleTaskThreshold = prod.StaleTaskThreshold
		config.dispatcherConfig.HeartbeatInterval = prod.HeartbeatInterval
		config.dispatcherConfig.WatermillHighHandlers = prod.WatermillHighHandlers
		config.dispatcherConfig.WatermillNormalHandlers = prod.WatermillNormalHandlers
		config.dispatcherConfig.WatermillLowHandlers = prod.WatermillLowHandlers
	}
}

func makeProfile(name string, capabilityName string) registry_dto.NamedProfile {
	return registry_dto.NamedProfile{
		Name: name,
		Profile: registry_dto.DesiredProfile{
			CapabilityName: capabilityName,
		},
	}
}
