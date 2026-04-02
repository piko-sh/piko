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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testBatchSize       = 150
	testBatchTimeout    = 10 * time.Millisecond
	testInsertQueueSize = 4096
)

type MockTaskDispatcher struct {
	DispatchFunc         func(ctx context.Context, task *Task) error
	DispatchDelayedFunc  func(ctx context.Context, task *Task, executeAt time.Time) error
	StartFunc            func(ctx context.Context) error
	StatsFunc            func() DispatcherStats
	RegisteredExecutors  map[string]TaskExecutor
	DispatchCalls        []*Task
	DispatchDelayedCalls []mockDelayedCall
	mu                   sync.Mutex
	StartCalled          bool
}

type mockDelayedCall struct {
	Task      *Task
	ExecuteAt time.Time
}

func NewMockTaskDispatcher() *MockTaskDispatcher {
	return &MockTaskDispatcher{
		DispatchCalls:        make([]*Task, 0),
		DispatchDelayedCalls: make([]mockDelayedCall, 0),
		RegisteredExecutors:  make(map[string]TaskExecutor),
	}
}

func (m *MockTaskDispatcher) Dispatch(ctx context.Context, task *Task) error {
	m.mu.Lock()
	m.DispatchCalls = append(m.DispatchCalls, task)
	executor := m.RegisteredExecutors[task.Executor]
	m.mu.Unlock()

	if m.DispatchFunc != nil {
		return m.DispatchFunc(ctx, task)
	}

	if executor != nil {
		go func() {
			_, _ = executor.Execute(ctx, task.Payload)
		}()
	}

	return nil
}

func (m *MockTaskDispatcher) DispatchDelayed(ctx context.Context, task *Task, executeAt time.Time) error {
	m.mu.Lock()
	m.DispatchDelayedCalls = append(m.DispatchDelayedCalls, mockDelayedCall{
		Task:      task,
		ExecuteAt: executeAt,
	})
	m.mu.Unlock()

	if m.DispatchDelayedFunc != nil {
		return m.DispatchDelayedFunc(ctx, task, executeAt)
	}
	return nil
}

func (m *MockTaskDispatcher) RegisterExecutor(_ context.Context, name string, executor TaskExecutor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RegisteredExecutors[name] = executor
}

func (m *MockTaskDispatcher) Start(ctx context.Context) error {
	m.mu.Lock()
	m.StartCalled = true
	m.mu.Unlock()

	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	<-ctx.Done()
	return nil
}

func (m *MockTaskDispatcher) Stats() DispatcherStats {
	if m.StatsFunc != nil {
		return m.StatsFunc()
	}
	return DispatcherStats{}
}

func (m *MockTaskDispatcher) IsIdle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.DispatchCalls) == 0 && len(m.DispatchDelayedCalls) == 0
}

func (m *MockTaskDispatcher) FailedTasks(_ context.Context) ([]FailedTaskSummary, error) {
	return nil, nil
}
func (m *MockTaskDispatcher) SetBuildTag(_ string) {}
func (m *MockTaskDispatcher) BuildTag() string     { return "" }

func (m *MockTaskDispatcher) GetDispatchCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.DispatchCalls)
}

func (m *MockTaskDispatcher) GetDispatchDelayedCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.DispatchDelayedCalls)
}

type TestOrchestratorService = orchestratorService

type ServiceTestBuilder struct {
	t              *testing.T
	store          TaskStore
	eventBus       EventBus
	taskDispatcher TaskDispatcher
	executors      map[string]TaskExecutor
	config         ServiceConfig
}

func NewServiceTestBuilder(t *testing.T) *ServiceTestBuilder {
	return &ServiceTestBuilder{
		t:         t,
		store:     NewFakeTaskStore(),
		executors: make(map[string]TaskExecutor),
		config: ServiceConfig{
			SchedulerInterval: 10 * time.Second,
			BatchSize:         testBatchSize,
			BatchTimeout:      testBatchTimeout,
			InsertQueueSize:   testInsertQueueSize,
		},
	}
}

func (b *ServiceTestBuilder) WithStore(s TaskStore) *ServiceTestBuilder {
	b.store = s
	return b
}

func (b *ServiceTestBuilder) WithEventBus(eb EventBus) *ServiceTestBuilder {
	b.eventBus = eb
	return b
}

func (b *ServiceTestBuilder) WithMockDispatcher(d TaskDispatcher) *ServiceTestBuilder {
	b.taskDispatcher = d
	return b
}

func (b *ServiceTestBuilder) WithExecutor(name string, e TaskExecutor) *ServiceTestBuilder {
	b.executors[name] = e
	return b
}

func (b *ServiceTestBuilder) WithSchedulerInterval(interval time.Duration) *ServiceTestBuilder {
	b.config.SchedulerInterval = interval
	return b
}

func (b *ServiceTestBuilder) WithBatchConfig(size int, timeout time.Duration) *ServiceTestBuilder {
	b.config.BatchSize = size
	b.config.BatchTimeout = timeout
	return b
}

func (b *ServiceTestBuilder) Build() *TestOrchestratorService {
	var opts []ServiceOption

	opts = append(opts,
		WithSchedulerInterval(b.config.SchedulerInterval),
		WithBatchConfig(b.config.BatchSize, b.config.BatchTimeout),
		WithInsertQueueSize(b.config.InsertQueueSize),
	)

	if b.taskDispatcher != nil {
		opts = append(opts, WithTaskDispatcher(b.taskDispatcher))
	}

	service, ok := NewService(context.Background(), b.store, b.eventBus, opts...).(*orchestratorService)
	require.True(b.t, ok, "expected NewService to return *orchestratorService")

	for name, executor := range b.executors {
		_ = service.RegisterExecutor(context.Background(), name, executor)
	}

	return service
}

func (b *ServiceTestBuilder) BuildAndRun(ctx context.Context) (*TestOrchestratorService, context.CancelFunc) {
	runCtx, cancel := context.WithCancelCause(ctx)
	service := b.Build()

	go service.Run(runCtx)

	return service, func() {
		cancel(fmt.Errorf("test: cleanup"))
		service.Stop()
	}
}
