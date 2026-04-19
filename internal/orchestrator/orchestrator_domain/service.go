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
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

var _ OrchestratorService = (*orchestratorService)(nil)

const (
	// defaultSchedulerInterval is the default time between scheduler runs.
	defaultSchedulerInterval = 10 * time.Second

	// defaultBatchSize is the number of items to process in each batch.
	defaultBatchSize = 250

	// defaultBatchTimeout is the default time to wait for a batch to fill.
	defaultBatchTimeout = 10 * time.Millisecond

	// defaultInsertQueueSize is the default size of the insert queue.
	defaultInsertQueueSize = 8192

	// backgroundLoopCount is the number of background goroutines started
	// during orchestrator startup.
	backgroundLoopCount = 3
)

// ServiceConfig holds settings for the orchestrator service.
// Use DefaultServiceConfig to get ready-to-use defaults.
type ServiceConfig struct {
	// TaskDispatcher allows injecting a custom task dispatcher, primarily for
	// testing. If nil, a new dispatcher is created using DispatcherConfig.
	TaskDispatcher TaskDispatcher

	// DispatcherConfig holds settings for the internal task dispatcher.
	// If nil, the service uses DefaultDispatcherConfig().
	DispatcherConfig *DispatcherConfig

	// Clock provides time operations; if nil, defaults to RealClock().
	// Useful for testing to make timing deterministic.
	Clock clock.Clock

	// SchedulerInterval is how often the scheduler checks for tasks to move from
	// scheduled to pending status.
	SchedulerInterval time.Duration

	// BatchTimeout is the maximum time to wait before sending a partial batch.
	BatchTimeout time.Duration

	// BatchSize is the maximum number of tasks to group before saving to the store.
	BatchSize int

	// InsertQueueSize is the buffer size for the task insertion channel;
	// 0 means unbuffered.
	InsertQueueSize int
}

// ServiceOption configures a ServiceConfig during service construction.
// Use these options to change service behaviour or inject test dependencies.
type ServiceOption func(*ServiceConfig)

// orchestratorService manages the lifecycle of tasks, including dispatching,
// scheduling, and coordinating workers. It implements OrchestratorService and
// HealthProbe interfaces, and is the central component of the orchestration
// domain.
type orchestratorService struct {
	// clock provides time operations for scheduling and timing.
	clock clock.Clock

	// taskStore stores and retrieves tasks.
	taskStore TaskStore

	// eventBus handles event subscriptions for workflow completion tracking.
	eventBus EventBus

	// taskDispatcher sends tasks to executors for processing.
	taskDispatcher TaskDispatcher

	// workflowCheckGroup deduplicates concurrent workflow status checks.
	workflowCheckGroup singleflight.Group

	// runCtx is the context for running background tasks.
	runCtx context.Context

	// cancel stops the run context to signal shutdown.
	cancel context.CancelCauseFunc

	// executors maps executor names to their TaskExecutor implementations.
	executors map[string]TaskExecutor

	// taskInsertChan receives tasks for batch insertion.
	taskInsertChan chan *Task

	// receipts maps workflow IDs to their pending receipts waiting for completion.
	receipts map[string][]*WorkflowReceipt

	// nodeID uniquely identifies this orchestrator instance for receipt tracking.
	nodeID string

	// wg tracks active goroutines to allow graceful shutdown.
	wg sync.WaitGroup

	// schedulerInterval is the time between scheduler loop ticks.
	schedulerInterval time.Duration

	// batchSize is the maximum number of tasks to collect before inserting.
	batchSize int

	// batchTimeout is the maximum time to wait before flushing a partial batch.
	batchTimeout time.Duration

	// executorsMutex guards access to the executors map.
	executorsMutex sync.RWMutex

	// stopMutex guards the isStopped flag during shutdown.
	stopMutex sync.Mutex

	// receiptsMutex protects access to the receipts map.
	receiptsMutex sync.Mutex

	// isStopped indicates whether the service has been stopped.
	isStopped bool
}

// ActiveTasks returns the number of tasks currently being processed by workers.
//
// Returns int64 which is the count of active workers, or zero if no dispatcher
// is configured.
func (s *orchestratorService) ActiveTasks(_ context.Context) int64 {
	if s.taskDispatcher != nil {
		stats := s.taskDispatcher.Stats()
		return int64(stats.ActiveWorkers)
	}
	return 0
}

// PendingTasks queries the task store for the number of tasks that are
// scheduled, pending, or retrying.
//
// Returns int64 which is the count of pending tasks, or zero if an error
// occurs.
func (s *orchestratorService) PendingTasks(ctx context.Context) int64 {
	count, err := s.taskStore.PendingTaskCount(ctx)
	if err != nil {
		return 0
	}
	return count
}

// GetTaskDispatcher returns the internal TaskDispatcher for direct task dispatch.
// Used by the bridge for competing-consumer task distribution to the worker pool.
//
// Returns TaskDispatcher which handles task distribution, or nil if none was
// configured.
func (s *orchestratorService) GetTaskDispatcher() TaskDispatcher {
	return s.taskDispatcher
}

// RegisterExecutor adds a new TaskExecutor to the service under a given name.
//
// Takes ctx (context.Context) which carries logging context.
// Takes name (string) which identifies the executor for later retrieval.
// Takes executor (TaskExecutor) which handles task execution.
//
// Returns error when an executor with the same name is already registered.
//
// Safe for concurrent use; protected by a mutex.
func (s *orchestratorService) RegisterExecutor(ctx context.Context, name string, executor TaskExecutor) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.RegisterExecutor",
		logger_domain.String("executorName", name),
	)
	defer span.End()

	s.executorsMutex.Lock()
	defer s.executorsMutex.Unlock()

	if _, exists := s.executors[name]; exists {
		err := fmt.Errorf("executor with name '%s' already registered", name)
		l.ReportError(span, err, "Executor already registered")
		ExecutorRegistrationErrorCount.Add(ctx, 1)
		return err
	}

	s.executors[name] = executor
	l.Internal("Executor registered successfully", logger_domain.Int("totalExecutors", len(s.executors)))
	ExecutorRegistrationCount.Add(ctx, 1)
	return nil
}

// Dispatch queues a task for batch insertion and returns a receipt immediately.
// This is an asynchronous, non-blocking operation that avoids waiting for
// database I/O, making the call very fast.
//
// Takes task (*Task) which specifies the task to be queued for processing.
//
// Returns *WorkflowReceipt which tracks the dispatched workflow's progress.
// Returns error when the task insertion queue is full due to backpressure.
func (s *orchestratorService) Dispatch(ctx context.Context, task *Task) (*WorkflowReceipt, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.Dispatch",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String("executor", task.Executor),
	)
	defer span.End()

	startTime := s.clock.Now()

	task.Status = StatusPending
	task.ExecuteAt = s.clock.Now()

	receiptID := uuid.New().String()
	receipt := newWorkflowReceipt(task.WorkflowID)
	s.registerReceipt(ctx, receiptID, receipt)

	select {
	case s.taskInsertChan <- task:
		l.Trace("Task queued for batch insertion", logger_domain.String(attributeKeyTaskID, task.ID))
	default:
		s.removeReceipt(receipt)
		err := errors.New("orchestrator overloaded: task insertion queue is full")
		l.ReportError(span, err, "Failed to dispatch task due to backpressure")
		TaskFailureCount.Add(ctx, 1)
		return nil, err
	}

	TaskDispatchDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))
	return receipt, nil
}

// DispatchDirect synchronously persists and dispatches a task, bypassing
// the async batch insertion queue. This is primarily for testing where
// deterministic, synchronous task dispatch is required.
//
// Unlike Dispatch, blocks until the task is persisted and dispatched, making tests
// predictable without timing dependencies.
//
// Takes task (*Task) which specifies the task to persist and dispatch.
//
// Returns *WorkflowReceipt which tracks the workflow execution status.
// Returns error when the task cannot be persisted.
func (s *orchestratorService) DispatchDirect(ctx context.Context, task *Task) (*WorkflowReceipt, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.DispatchDirect",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String("executor", task.Executor),
	)
	defer span.End()

	startTime := s.clock.Now()

	task.Status = StatusPending
	task.ExecuteAt = s.clock.Now()

	receiptID := uuid.New().String()
	receipt := newWorkflowReceipt(task.WorkflowID)

	err := s.taskStore.RunAtomic(ctx, func(ctx context.Context, store TaskStore) error {
		if err := store.CreateTask(ctx, task); err != nil {
			return fmt.Errorf("persisting task: %w", err)
		}
		if err := store.CreateWorkflowReceipt(ctx, receiptID, receipt.WorkflowID, s.nodeID); err != nil {
			return fmt.Errorf("persisting workflow receipt: %w", err)
		}
		return nil
	})
	if err != nil {
		l.ReportError(span, err, "Failed to persist task and receipt")
		TaskFailureCount.Add(ctx, 1)
		return nil, fmt.Errorf("dispatching task directly: %w", err)
	}

	s.registerReceiptInMemory(ctx, receipt)

	if s.taskDispatcher != nil {
		if err := s.taskDispatcher.Dispatch(ctx, task); err != nil {
			l.Warn("Failed to dispatch task to dispatcher",
				logger_domain.Error(err),
				logger_domain.String(attributeKeyTaskID, task.ID))
		}
	}

	l.Trace("Task dispatched directly", logger_domain.String(attributeKeyTaskID, task.ID))
	TaskDispatchDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))
	return receipt, nil
}

// Schedule queues a task to run at a given time in the future. This is
// non-blocking and returns a receipt straight away.
//
// Takes task (*Task) which specifies the task to be scheduled.
// Takes executeAt (time.Time) which specifies when the task should run.
//
// Returns *WorkflowReceipt which tracks the scheduled workflow's progress.
// Returns error when the task queue is full.
func (s *orchestratorService) Schedule(ctx context.Context, task *Task, executeAt time.Time) (*WorkflowReceipt, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.Schedule",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String("executor", task.Executor),
		logger_domain.Time("executeAt", executeAt),
	)
	defer span.End()

	startTime := s.clock.Now()
	delay := executeAt.Sub(startTime)

	task.Status = StatusScheduled
	task.ExecuteAt = executeAt

	receiptID := uuid.New().String()
	receipt := newWorkflowReceipt(task.WorkflowID)
	s.registerReceipt(ctx, receiptID, receipt)

	select {
	case s.taskInsertChan <- task:
		l.Trace("Scheduled task queued for batch insertion",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Float64("delaySeconds", delay.Seconds()))
	default:
		s.removeReceipt(receipt)
		err := errors.New("orchestrator overloaded: task insertion queue is full")
		l.ReportError(span, err, "Failed to schedule task due to backpressure")
		TaskFailureCount.Add(ctx, 1)
		return nil, err
	}

	TaskScheduleDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))
	return receipt, nil
}

// Run starts the orchestrator's background processes.
//
// Spawns goroutines for task dispatching, batch insertion, scheduled task
// promotion, and completion event handling. Blocks until the context is
// cancelled.
//
// Safe for concurrent use.
func (s *orchestratorService) Run(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.Run")
	defer span.End()

	s.runCtx, s.cancel = context.WithCancelCause(ctx)

	l.Internal("Starting Orchestrator Service with TaskDispatcher")

	if s.taskDispatcher != nil {
		s.executorsMutex.RLock()
		for name, executor := range s.executors {
			s.taskDispatcher.RegisterExecutor(ctx, name, executor)
		}
		s.executorsMutex.RUnlock()

		s.wg.Go(func() {
			if err := s.taskDispatcher.Start(s.runCtx); err != nil {
				l.Error("TaskDispatcher exited with error", logger_domain.Error(err))
			}
		})
	}

	s.wg.Add(backgroundLoopCount)
	go s.batchInsertLoop()
	go s.schedulerLoop()
	go s.subscribeToCompletionEvents()

	l.Internal("Orchestrator Service started successfully")

	<-s.runCtx.Done()
	l.Internal("Orchestrator Service shutting down",
		logger_domain.String("reason", s.runCtx.Err().Error()))
}

// Stop gracefully shuts down the orchestrator service.
//
// Safe for concurrent use. Blocks until all background goroutines have
// finished.
func (s *orchestratorService) Stop() {
	s.stopMutex.Lock()
	if s.isStopped {
		s.stopMutex.Unlock()
		return
	}
	s.isStopped = true
	s.stopMutex.Unlock()

	_, stl := logger_domain.From(context.WithoutCancel(s.runCtx), log)
	stl.Internal("Orchestrator Service stopping")

	if s.cancel != nil {
		s.cancel(errors.New("orchestrator service stopped"))
	}

	s.wg.Wait()
	stl.Internal("Orchestrator Service stopped gracefully")
}

// schedulerLoop runs at regular intervals to move tasks from SCHEDULED to
// PENDING status.
func (s *orchestratorService) schedulerLoop() {
	defer s.wg.Done()
	defer goroutine.RecoverPanic(s.runCtx, "orchestrator.schedulerLoop")
	ticker := s.clock.NewTicker(s.schedulerInterval)
	defer ticker.Stop()

	ctx, scl := logger_domain.From(s.runCtx, log)
	scl.Internal("Scheduler loop started", logger_domain.Duration("interval", s.schedulerInterval))

	for {
		select {
		case <-ctx.Done():
			scl.Internal("Scheduler loop shutting down.")
			return
		case <-ticker.C():
			s.promoteScheduledTasks(ctx)
		}
	}
}

// promoteScheduledTasks moves scheduled tasks to pending status when their
// scheduled time has passed.
func (s *orchestratorService) promoteScheduledTasks(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.promoteScheduledTasks")
	defer span.End()

	startTime := s.clock.Now()
	l.Trace("Checking for scheduled tasks to promote")

	promotedCount, err := s.taskStore.PromoteScheduledTasks(ctx)
	if err != nil {
		l.ReportError(span, err, "Failed to promote scheduled tasks")
	} else if promotedCount > 0 {
		l.Trace("Promoted scheduled tasks to pending", logger_domain.Int("count", promotedCount))
	}

	SchedulerPromotionDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))
}

// DefaultServiceConfig returns service settings with sensible defaults.
//
// Returns ServiceConfig which contains default values for scheduler interval,
// batch size, batch timeout, and insert queue size.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		SchedulerInterval: defaultSchedulerInterval,
		BatchSize:         defaultBatchSize,
		BatchTimeout:      defaultBatchTimeout,
		InsertQueueSize:   defaultInsertQueueSize,
		DispatcherConfig:  nil,
		TaskDispatcher:    nil,
	}
}

// WithSchedulerInterval sets the time between scheduler loop runs.
//
// Takes interval (time.Duration) which specifies how often the scheduler runs.
//
// Returns ServiceOption which sets the scheduler interval when applied.
func WithSchedulerInterval(interval time.Duration) ServiceOption {
	return func(c *ServiceConfig) {
		c.SchedulerInterval = interval
	}
}

// WithBatchConfig sets the batch insertion settings.
//
// Takes batchSize (int) which sets how many items to include in each batch.
// Takes batchTimeout (time.Duration) which sets how long to wait before
// sending a batch that is not yet full.
//
// Returns ServiceOption which applies the batch settings to a service.
func WithBatchConfig(batchSize int, batchTimeout time.Duration) ServiceOption {
	return func(c *ServiceConfig) {
		c.BatchSize = batchSize
		c.BatchTimeout = batchTimeout
	}
}

// WithInsertQueueSize sets the buffer size for the task insertion channel.
//
// Takes size (int) which sets how many tasks the channel can hold.
//
// Returns ServiceOption which sets the insertion queue size.
func WithInsertQueueSize(size int) ServiceOption {
	return func(c *ServiceConfig) {
		c.InsertQueueSize = size
	}
}

// WithDispatcherConfig sets a custom dispatcher configuration.
//
// Takes config (DispatcherConfig) which specifies the dispatcher settings.
//
// Returns ServiceOption which applies the dispatcher configuration.
func WithDispatcherConfig(config DispatcherConfig) ServiceOption {
	return func(c *ServiceConfig) {
		c.DispatcherConfig = &config
	}
}

// WithTaskDispatcher sets a custom TaskDispatcher for the service.
// Useful for testing without running real worker goroutines.
//
// Takes dispatcher (TaskDispatcher) which provides the task dispatch behaviour.
//
// Returns ServiceOption which configures the service to use the given
// dispatcher.
func WithTaskDispatcher(dispatcher TaskDispatcher) ServiceOption {
	return func(c *ServiceConfig) {
		c.TaskDispatcher = dispatcher
	}
}

// WithServiceClock sets a custom clock for time operations on the service.
// This is mainly used for testing to make timing predictable.
//
// Takes c (clock.Clock) which provides the time source for the service.
//
// Returns ServiceOption which sets the service to use the given clock.
//
// Use WithClock for the dispatcher's clock settings.
func WithServiceClock(c clock.Clock) ServiceOption {
	return func(config *ServiceConfig) {
		config.Clock = c
	}
}

// NewService creates a new orchestrator service with configurable settings.
//
// It requires a TaskStore for persistence and an EventBus for event-driven
// coordination. Use ServiceOption functions to customise behaviour or inject
// test dependencies.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation through background goroutines.
// Takes store (TaskStore) which provides persistence for tasks.
// Takes eventBus (EventBus) which handles event-driven coordination.
// Takes opts (...ServiceOption) which customises behaviour or injects
// test dependencies.
//
// Returns OrchestratorService which is the configured service ready for use.
func NewService(ctx context.Context, store TaskStore, eventBus EventBus, opts ...ServiceOption) OrchestratorService {
	config := DefaultServiceConfig()

	for _, opt := range opts {
		opt(&config)
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	ctx, nl := logger_domain.From(ctx, log)
	taskDispatcher := config.TaskDispatcher
	if taskDispatcher == nil {
		nl.Warn("No TaskDispatcher injected. " +
			"Use orchestrator_adapters.CreateTaskDispatcher() with WithTaskDispatcher() " +
			"for task processing. Task dispatch will not function without a dispatcher.")
	}

	nodeID := ""
	if config.DispatcherConfig != nil && config.DispatcherConfig.NodeID != "" {
		nodeID = config.DispatcherConfig.NodeID
	}
	if nodeID == "" {
		nodeID = uuid.New().String()
	}

	return &orchestratorService{
		clock:              clk,
		taskStore:          store,
		eventBus:           eventBus,
		taskDispatcher:     taskDispatcher,
		workflowCheckGroup: singleflight.Group{},
		runCtx:             nil,
		taskInsertChan:     make(chan *Task, config.InsertQueueSize),
		executors:          make(map[string]TaskExecutor),
		cancel:             nil,
		receipts:           make(map[string][]*WorkflowReceipt),
		wg:                 sync.WaitGroup{},
		schedulerInterval:  config.SchedulerInterval,
		batchSize:          config.BatchSize,
		batchTimeout:       config.BatchTimeout,
		executorsMutex:     sync.RWMutex{},
		stopMutex:          sync.Mutex{},
		receiptsMutex:      sync.Mutex{},
		isStopped:          false,
		nodeID:             nodeID,
	}
}

// getPayloadString extracts a string value from a payload map.
// Returns an empty string if the key does not exist or if the value is not a
// string.
//
// Takes payload (map[string]any) which contains the key-value pairs to search.
// Takes key (string) which specifies the key to look up.
//
// Returns string which is the value if found, or an empty string otherwise.
func getPayloadString(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
