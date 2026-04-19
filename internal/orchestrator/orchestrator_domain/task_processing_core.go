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
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

var (
	// errTaskIDRequired is returned when a task is submitted without an ID.
	errTaskIDRequired = errors.New("task ID is required")

	// errTaskWorkflowIDRequired is returned when a task is submitted without
	// a workflow ID.
	errTaskWorkflowIDRequired = errors.New("task workflowID is required")

	// errTaskExecutorRequired is returned when a task is submitted without an
	// executor name.
	errTaskExecutorRequired = errors.New("task executor is required")
)

// TaskProcessingCore contains the shared task processing logic used by both
// the local channel dispatcher and the Watermill dispatcher.
//
// Encapsulates:
//   - Executor registry and lookup
//   - Task execution with timeout
//   - Success/failure handling with retry logic
//   - Persistence and event publishing
//   - In-flight task tracking for graceful shutdown
//   - Metric counters
//
// Each dispatcher embeds this core and calls its methods for the actual
// task processing, while handling distribution (channels vs topics) separately.
type TaskProcessingCore struct {
	// Clock provides time functions for task timestamps; defaults to real time.
	Clock clockpkg.Clock

	// OtelPropagator passes trace context between services.
	OtelPropagator propagation.TextMapPropagator

	// DelayedPublisher schedules tasks for delayed execution; nil disables
	// delayed retry scheduling.
	DelayedPublisher DelayedPublisher

	// EventBus publishes task completion events.
	EventBus EventBus

	// TaskStore saves task changes to storage; nil turns off saving.
	TaskStore TaskStore

	// executors maps executor names to their handlers.
	executors map[string]TaskExecutor

	// shutdownCh signals persistence goroutines to abort when closed.
	shutdownCh chan struct{}

	// persistSemaphore bounds the number of concurrent async persistence
	// goroutines. Each Go() call acquires a permit before spawning; on
	// saturation the caller falls back to synchronous persistence so the
	// work still completes but goroutine count stays bounded.
	persistSemaphore chan struct{}

	// heartbeatStopChans maps task ID to a chan struct{} used to stop the
	// heartbeat goroutine for that task.
	heartbeatStopChans sync.Map

	// InFlightTasks tracks tasks that are being processed by this instance.
	// Maps task ID to *Task for graceful shutdown release.
	InFlightTasks sync.Map

	// nodeID uniquely identifies this orchestrator instance for recovery leases.
	nodeID string

	// buildTag is an optional tag that scopes tasks to a particular build run.
	buildTag string

	// Config holds the dispatcher settings for task processing defaults.
	Config DispatcherConfig

	// persistWg tracks goroutines that are saving data in the background.
	persistWg sync.WaitGroup

	// TasksCompleted is the counter of completed tasks (atomic for lock-free reads).
	TasksCompleted int64

	// TasksFailed is the count of tasks that have failed; accessed atomically.
	TasksFailed int64

	// TasksFatalFailed is the subset of TasksFailed that were caused by fatal
	// (non-retryable) errors; accessed atomically.
	TasksFatalFailed int64

	// TasksRetried counts tasks that have been retried; updated atomically.
	TasksRetried int64

	// TasksDispatched counts dispatched tasks. Uses atomic operations for
	// lock-free reads.
	TasksDispatched int64

	// executorsMutex guards access to the executors map.
	executorsMutex sync.RWMutex

	// buildTagMu guards access to the buildTag field.
	buildTagMu sync.RWMutex

	// shutdownOnce guards single closure of the shutdown channel.
	shutdownOnce sync.Once
}

// NewTaskProcessingCore creates a new task processing core with the given
// dependencies.
//
// Takes config (DispatcherConfig) which specifies the processing settings.
// Takes eventBus (EventBus) which handles coordination events.
// Takes taskStore (TaskStore) which provides persistence.
// Takes clock (Clock) which provides time operations.
//
// Returns *TaskProcessingCore ready to have executors registered.
func NewTaskProcessingCore(
	config DispatcherConfig,
	eventBus EventBus,
	taskStore TaskStore,
	clock clockpkg.Clock,
) *TaskProcessingCore {
	if clock == nil {
		clock = clockpkg.RealClock()
	}

	nodeID := config.NodeID
	if nodeID == "" {
		nodeID = uuid.New().String()
	}

	return &TaskProcessingCore{
		EventBus:           eventBus,
		TaskStore:          taskStore,
		Clock:              clock,
		OtelPropagator:     propagation.TraceContext{},
		DelayedPublisher:   nil,
		executors:          make(map[string]TaskExecutor),
		shutdownCh:         make(chan struct{}),
		persistSemaphore:   make(chan struct{}, config.EffectiveMaxConcurrentPersistJobs()),
		Config:             config,
		nodeID:             nodeID,
		InFlightTasks:      sync.Map{},
		heartbeatStopChans: sync.Map{},
		persistWg:          sync.WaitGroup{},
		executorsMutex:     sync.RWMutex{},
		shutdownOnce:       sync.Once{},
		TasksDispatched:    0,
		TasksCompleted:     0,
		TasksFailed:        0,
		TasksRetried:       0,
	}
}

// RegisterExecutor adds a task executor with the given name.
// Must be called before processing starts for all executor types.
//
// Takes ctx (context.Context) which carries logging context.
// Takes name (string) which identifies the executor.
// Takes executor (TaskExecutor) which handles tasks of the named kind.
//
// Safe for concurrent use.
func (c *TaskProcessingCore) RegisterExecutor(ctx context.Context, name string, executor TaskExecutor) {
	c.executorsMutex.Lock()
	defer c.executorsMutex.Unlock()
	c.executors[name] = executor
	_, rl := logger_domain.From(ctx, log)
	rl.Internal("Executor registered",
		logger_domain.String("executor", name))
}

// GetExecutor retrieves a registered executor by name.
//
// Takes name (string) which identifies the executor.
//
// Returns TaskExecutor if found, or error if not registered.
//
// Safe for concurrent use.
func (c *TaskProcessingCore) GetExecutor(name string) (TaskExecutor, error) {
	c.executorsMutex.RLock()
	executor, exists := c.executors[name]
	c.executorsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("executor not found: %s", name)
	}
	return executor, nil
}

// ExecutorCount returns the number of registered executors.
//
// Returns int which is the current count of executors.
//
// Safe for concurrent use.
func (c *TaskProcessingCore) ExecutorCount() int {
	c.executorsMutex.RLock()
	defer c.executorsMutex.RUnlock()
	return len(c.executors)
}

// ValidateTask checks that a task has all required fields set.
//
// Takes task (*Task) which is the task to validate.
//
// Returns error if any required field is missing.
func (*TaskProcessingCore) ValidateTask(task *Task) error {
	if task.ID == "" {
		return errTaskIDRequired
	}
	if task.WorkflowID == "" {
		return errTaskWorkflowIDRequired
	}
	if task.Executor == "" {
		return errTaskExecutorRequired
	}
	return nil
}

// ApplyDefaults sets default values for any unset task settings.
//
// Takes task (*Task) which is updated in place with defaults.
func (c *TaskProcessingCore) ApplyDefaults(task *Task) {
	if task.Config.Timeout <= 0 {
		task.Config.Timeout = c.Config.DefaultTimeout
	}
	if task.Config.MaxRetries <= 0 {
		task.Config.MaxRetries = c.Config.DefaultMaxRetries
	}
	if task.Status == "" {
		task.Status = StatusPending
	}
}

// PrepareTaskExecution sets up a task for execution.
// Increments attempt counter, sets status, and tracks in-flight.
//
// Takes ctx (context.Context) which carries tracing values for persistence.
// Takes task (*Task) which is being prepared for execution.
//
// Returns the timeout duration to use for execution.
func (c *TaskProcessingCore) PrepareTaskExecution(ctx context.Context, task *Task) time.Duration {
	task.Attempt++
	task.UpdatedAt = c.Clock.Now()
	task.Status = StatusProcessing

	c.InFlightTasks.Store(task.ID, task)

	c.PersistTaskUpdate(ctx, task)

	c.StartHeartbeat(ctx, task.ID)

	taskTimeout := task.Config.Timeout
	if taskTimeout <= 0 {
		taskTimeout = c.Config.DefaultTimeout
	}
	return taskTimeout
}

// ExecuteTask runs the executor with timeout and returns any error.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which is being executed.
// Takes executor (TaskExecutor) which handles the task.
// Takes timeout (time.Duration) which limits execution time.
//
// Returns error from the executor, or nil on success.
func (*TaskProcessingCore) ExecuteTask(
	ctx context.Context,
	task *Task,
	executor TaskExecutor,
	timeout time.Duration,
) error {
	ctx, l := logger_domain.From(ctx, log)
	execCtx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("task execution exceeded %s timeout", timeout))
	defer cancel()

	l.Trace("Executing task")
	var execErr error
	var result any

	_ = l.RunInSpan(ctx, "ExecuteTask", func(_ context.Context, _ logger_domain.Logger) error {
		execStartTime := time.Now()
		result, execErr = executor.Execute(execCtx, task.Payload)

		if result != nil {
			if mapResult, ok := result.(map[string]any); ok {
				task.Result = mapResult
			}
		}
		TaskExecutionDuration.Record(ctx, float64(time.Since(execStartTime).Milliseconds()))
		return nil
	})

	return execErr
}

// HandleExecutionResult processes the result of task execution.
// Routes to success or failure handling based on the error.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which was executed.
// Takes execErr (error) which is the execution result, or nil on success.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleExecutionResult(
	ctx context.Context,
	task *Task,
	execErr error,
	startTime time.Time,
) {
	ctx, l := logger_domain.From(ctx, log)
	if execErr != nil {
		l.Warn("Task execution failed",
			logger_domain.Error(execErr),
			logger_domain.Int(attributeKeyAttempt, task.Attempt))
		c.HandleTaskFailure(ctx, task, execErr, startTime)
	} else {
		l.Trace("Task completed successfully")
		c.HandleTaskSuccess(ctx, task, startTime)
	}
}

// HandleTaskSuccess handles successful task completion.
//
// Takes task (*Task) which completed successfully.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleTaskSuccess(ctx context.Context, task *Task, startTime time.Time) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.handleTaskSuccess",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
	)
	defer span.End()

	c.stopHeartbeat(task.ID)

	c.InFlightTasks.Delete(task.ID)

	atomic.AddInt64(&c.TasksCompleted, 1)
	TaskSuccessCount.Add(ctx, 1)

	task.Status = StatusComplete
	task.LastError = ""
	task.UpdatedAt = c.Clock.Now()

	c.PersistTaskUpdate(ctx, task)

	c.PublishCompletionEvent(ctx, task, nil, c.Clock.Now().Sub(startTime))

	l.Trace("Task completed successfully",
		logger_domain.Int("resultSize", len(task.Result)))
	span.SetStatus(codes.Ok, "Task completed")
}

// HandleTaskFailure handles task execution failure with retry logic.
//
// Takes task (*Task) which failed.
// Takes execErr (error) which caused the failure.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleTaskFailure(ctx context.Context, task *Task, execErr error, startTime time.Time) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.handleTaskFailure",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
		logger_domain.Error(execErr),
		logger_domain.Int(attributeKeyAttempt, task.Attempt),
		logger_domain.Int("maxRetries", task.Config.MaxRetries),
	)
	defer span.End()

	task.LastError = execErr.Error()
	task.UpdatedAt = c.Clock.Now()
	task.IsFatal = IsFatalError(execErr)

	if task.IsFatal {
		l.Trace("Task failed with fatal error, skipping retries",
			logger_domain.Error(execErr))
		c.MarkTaskFailed(ctx, task, execErr, startTime, span)
		return
	}

	if !shouldRetryTask(task.Attempt, task.Config.MaxRetries, c.Config.DefaultMaxRetries) {
		c.MarkTaskFailed(ctx, task, execErr, startTime, span)
		return
	}

	c.ScheduleTaskRetry(ctx, task, span)
}

// MarkTaskFailed marks a task as permanently failed after max retries.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which has exhausted retries.
// Takes execErr (error) which caused the final failure.
// Takes startTime (time.Time) which is when execution began.
// Takes span (interface{SetStatus}) which records the error status for tracing.
func (c *TaskProcessingCore) MarkTaskFailed(
	ctx context.Context,
	task *Task,
	execErr error,
	startTime time.Time,
	span interface{ SetStatus(codes.Code, string) },
) {
	ctx, l := logger_domain.From(ctx, log)
	c.stopHeartbeat(task.ID)

	c.InFlightTasks.Delete(task.ID)

	atomic.AddInt64(&c.TasksFailed, 1)
	if task.IsFatal {
		atomic.AddInt64(&c.TasksFatalFailed, 1)
	}
	TaskFailureCount.Add(ctx, 1)

	task.Status = StatusFailed

	if task.IsFatal {
		l.Trace("Task failed with fatal error",
			logger_domain.Int("totalAttempts", task.Attempt))
	} else {
		l.Trace("Task failed after max retries",
			logger_domain.Int("totalAttempts", task.Attempt))
	}

	c.PersistTaskUpdate(ctx, task)
	c.PublishCompletionEvent(ctx, task, execErr, time.Since(startTime))
	span.SetStatus(codes.Error, "Task failed after max retries")
}

// ScheduleTaskRetry schedules a task for retry with exponential backoff.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which needs to be retried.
// Takes span (interface{SetStatus}) which records the operation status.
func (c *TaskProcessingCore) ScheduleTaskRetry(
	ctx context.Context,
	task *Task,
	span interface{ SetStatus(codes.Code, string) },
) {
	ctx, l := logger_domain.From(ctx, log)
	c.stopHeartbeat(task.ID)

	c.InFlightTasks.Delete(task.ID)

	atomic.AddInt64(&c.TasksRetried, 1)
	TaskRetryCount.Add(ctx, 1)

	retryDelay := calculateRetryBackoff(task.Attempt, rand.IntN)
	executeAt := c.Clock.Now().Add(retryDelay)

	task.Status = StatusRetrying
	task.ExecuteAt = executeAt
	task.ScheduledExecuteAt = executeAt

	l.Warn("Task failed, scheduling retry",
		logger_domain.Duration("retryDelay", retryDelay),
		logger_domain.Time("executeAt", executeAt),
		logger_domain.Int("nextAttempt", task.Attempt+1))

	c.PersistTaskUpdate(ctx, task)

	if c.DelayedPublisher != nil {
		if err := c.DelayedPublisher.Schedule(ctx, task); err != nil {
			l.Warn("Failed to schedule retry, task may be lost",
				logger_domain.Error(err))
		}
	}

	span.SetStatus(codes.Ok, "Task scheduled for retry")
}

// PublishCompletionEvent publishes a task.completed event for coordination.
//
// Takes task (*Task) which completed.
// Takes taskErr (error) which is the error, or nil on success.
// Takes duration (time.Duration) which is how long execution took.
func (c *TaskProcessingCore) PublishCompletionEvent(ctx context.Context, task *Task, taskErr error, duration time.Duration) {
	if c.EventBus == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.publishCompletionEvent",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
	)
	defer span.End()

	event := Event{
		Type:    EventType(TopicTaskCompleted),
		Payload: buildCompletionEventPayload(task, taskErr, duration, c.Clock.Now().UTC()),
	}

	if err := c.EventBus.Publish(ctx, TopicTaskCompleted, event); err != nil {
		l.Warn("Failed to publish task completion event",
			logger_domain.Error(err))
		span.RecordError(err)
	} else {
		l.Trace("Task completion event published")
	}
}

// PersistTaskUpdate persists task state to the store.
//
// Uses async persistence by default; set config.SyncPersistence=true for
// synchronous mode. During shutdown, automatically falls back to synchronous
// persistence to avoid data loss. When the persist concurrency cap is
// saturated, the caller also falls back to synchronous persistence so the
// goroutine count stays bounded and the work still completes.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached internally so persistence completes independently.
// Takes task (*Task) which contains the task state to persist.
func (c *TaskProcessingCore) PersistTaskUpdate(ctx context.Context, task *Task) {
	if c.TaskStore == nil {
		return
	}

	detachedCtx := context.WithoutCancel(ctx)

	if c.Config.SyncPersistence {
		c.persistTaskSync(detachedCtx, task)
		return
	}

	select {
	case <-c.shutdownCh:
		c.persistTaskSync(detachedCtx, task)
		return
	default:
	}

	if !c.acquirePersistPermit() {
		c.persistTaskSync(detachedCtx, task)
		return
	}

	taskCopy := *task
	c.persistWg.Go(func() {
		defer c.releasePersistPermit()
		c.persistTaskSync(detachedCtx, &taskCopy)
	})
}

// PersistWithDedup creates a task with deduplication check.
// Uses the store's CreateTaskWithDedup method which handles deduplication
// atomically.
//
// Takes ctx (context.Context) which provides cancellation.
// Takes task (*Task) which has the task to persist.
//
// Returns ErrDuplicateTask if a task with the same deduplication key exists,
// or any other persistence error. During shutdown, automatically falls back
// to synchronous persistence. When the persist concurrency cap is saturated,
// the caller also falls back to synchronous persistence so the goroutine count
// stays bounded.
//
// Safe for concurrent use. The spawned goroutine runs until the
// persistence operation completes or the context is cancelled.
func (c *TaskProcessingCore) PersistWithDedup(ctx context.Context, task *Task) error {
	if c.TaskStore == nil || task.persisted {
		return nil
	}

	if c.Config.SyncPersistence {
		return c.TaskStore.CreateTaskWithDedup(ctx, task)
	}

	select {
	case <-c.shutdownCh:
		return c.TaskStore.CreateTaskWithDedup(ctx, task)
	default:
	}

	if !c.acquirePersistPermit() {
		return c.TaskStore.CreateTaskWithDedup(ctx, task)
	}

	detachedCtx := context.WithoutCancel(ctx)
	_, l := logger_domain.From(ctx, log)
	c.persistWg.Go(func() {
		defer c.releasePersistPermit()
		persistCtx, cancel := context.WithTimeoutCause(detachedCtx, 5*time.Second,
			errors.New("task dedup persist exceeded 5s timeout"))
		defer cancel()

		go func() {
			select {
			case <-c.shutdownCh:
				cancel()
			case <-persistCtx.Done():
			}
		}()

		if err := c.TaskStore.CreateTaskWithDedup(persistCtx, task); err != nil {
			if !errors.Is(err, ErrDuplicateTask) && !errors.Is(err, context.Canceled) {
				l.Warn("Failed to persist task",
					logger_domain.Error(err),
					logger_domain.String(attributeKeyTaskID, task.ID))
			}
		}
	})
	return nil
}

// RecordProcessingMetrics records the final processing duration and span
// attributes.
//
// Takes span (interface{...}) which receives the metric attributes.
// Takes task (*Task) which provides status and attempt information.
// Takes startTime (time.Time) which marks when processing began.
func (*TaskProcessingCore) RecordProcessingMetrics(
	ctx context.Context,
	span interface{ SetAttributes(...attribute.KeyValue) },
	task *Task,
	startTime time.Time,
) {
	duration := time.Since(startTime)
	TaskProcessingDuration.Record(ctx, float64(duration.Milliseconds()))

	span.SetAttributes(
		attribute.Int64("durationMs", duration.Milliseconds()),
		attribute.String("finalStatus", string(task.Status)),
		attribute.Int(attributeKeyAttempt, task.Attempt),
	)
}

// Shutdown signals all persistence tasks to stop and waits for
// completion, safe for use from multiple callers.
//
// Takes ctx (context.Context) which provides a timeout for
// waiting.
//
// Returns error when the wait times out.
func (c *TaskProcessingCore) Shutdown(ctx context.Context) error {
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
	})

	done := make(chan struct{})
	go func() {
		c.persistWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("persistence shutdown timed out: %w", ctx.Err())
	}
}

// ReleaseInFlightTasks releases all in-flight tasks back to pending state,
// allowing other instances to pick up the work during graceful shutdown.
//
// Returns only after all pending persistence operations complete or the context
// times out.
//
// Safe for concurrent use.
func (c *TaskProcessingCore) ReleaseInFlightTasks(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
	})

	c.InFlightTasks.Range(func(key, value any) bool {
		if ctx.Err() != nil {
			l.Warn("Shutdown timeout reached, stopping in-flight task release")
			return false
		}

		task, ok := value.(*Task)
		if !ok {
			return true
		}

		c.stopHeartbeat(task.ID)

		task.Status = StatusPending
		task.UpdatedAt = c.Clock.Now()
		c.PersistTaskUpdate(ctx, task)

		l.Internal("Released in-flight task during shutdown",
			logger_domain.String(attributeKeyTaskID, task.ID))

		c.InFlightTasks.Delete(key)
		return true
	})

	done := make(chan struct{})
	go func() {
		c.persistWg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		l.Warn("Persistence goroutines did not complete in time")
	}
}

// InFlightCount returns the number of tasks currently being processed.
//
// Returns int which is the count of tasks currently in flight.
func (c *TaskProcessingCore) InFlightCount() int {
	count := 0
	c.InFlightTasks.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// RecoverStaleTasks finds and reprocesses tasks that have been stuck in
// PROCESSING state for too long. Uses the store's built-in recovery logic.
//
// Returns int which is the count of recovered tasks.
// Returns error when claiming or recovering stale tasks fails.
func (c *TaskProcessingCore) RecoverStaleTasks(ctx context.Context) (int, error) {
	if c.TaskStore == nil {
		return 0, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.recoverStaleTasks",
		logger_domain.String("nodeID", c.nodeID))
	defer span.End()

	leaseTimeout, batchLimit := c.recoveryParams()
	var (
		claimed []RecoveryClaimedTask
		count   int
	)

	err := c.TaskStore.RunAtomic(ctx, func(ctx context.Context, store TaskStore) error {
		var claimErr error
		claimed, claimErr = store.ClaimStaleTasksForRecovery(
			ctx, c.nodeID, c.Config.StaleTaskThreshold, leaseTimeout, batchLimit,
		)
		if claimErr != nil {
			return fmt.Errorf("claiming stale tasks: %w", claimErr)
		}
		if len(claimed) == 0 {
			return nil
		}

		var recoverErr error
		count, recoverErr = store.RecoverClaimedTasks(
			ctx, c.nodeID, c.Config.DefaultMaxRetries, staleTaskRecoveryError,
		)
		if recoverErr != nil {
			return fmt.Errorf("recovering claimed tasks: %w", recoverErr)
		}
		return nil
	})
	if err != nil {
		l.Warn("Failed to recover stale tasks", logger_domain.Error(err))
		TaskRecoveryErrorCount.Add(ctx, 1)
		return 0, err
	}

	if len(claimed) == 0 {
		span.SetStatus(codes.Ok, "No stale tasks to recover")
		return 0, nil
	}

	l.Internal("Claimed stale tasks for recovery",
		logger_domain.Int("claimed", len(claimed)))

	if count > 0 {
		l.Notice("Recovered stale tasks",
			logger_domain.Int("count", count),
			logger_domain.Duration("staleThreshold", c.Config.StaleTaskThreshold))
		TaskRecoveryCount.Add(ctx, int64(count))
	}

	span.SetStatus(codes.Ok, "Stale tasks recovered")
	return count, nil
}

// ReleaseRecoveryLeases releases all recovery leases held by this node.
// Called during graceful shutdown to allow other nodes to recover the tasks.
//
// Returns int which is the count of leases released.
// Returns error when the release fails.
func (c *TaskProcessingCore) ReleaseRecoveryLeases(ctx context.Context) (int, error) {
	if c.TaskStore == nil {
		return 0, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.releaseRecoveryLeases",
		logger_domain.String("nodeID", c.nodeID))
	defer span.End()

	count, err := c.TaskStore.ReleaseRecoveryLeases(ctx, c.nodeID)
	if err != nil {
		l.Warn("Failed to release recovery leases", logger_domain.Error(err))
		return 0, fmt.Errorf("releasing recovery leases: %w", err)
	}

	if count > 0 {
		l.Internal("Released recovery leases",
			logger_domain.Int("count", count))
	}

	span.SetStatus(codes.Ok, "Recovery leases released")
	return count, nil
}

// NodeID returns the unique identifier for this orchestrator instance.
//
// Returns string which is the unique node identifier.
func (c *TaskProcessingCore) NodeID() string {
	return c.nodeID
}

// ProcessingStats holds the counters returned by TaskProcessingCore.Stats.
type ProcessingStats struct {
	// Dispatched is the total number of tasks sent for processing.
	Dispatched int64

	// Completed is the total number of tasks that finished successfully.
	Completed int64

	// Failed is the total number of tasks that ended in error.
	Failed int64

	// FatalFailed is the subset of failed tasks caused by fatal
	// (non-retryable) errors.
	FatalFailed int64

	// Retried is the total number of tasks that were retried.
	Retried int64
}

// Stats returns current processing statistics.
//
// Returns ProcessingStats which contains the current task counters.
func (c *TaskProcessingCore) Stats() ProcessingStats {
	return ProcessingStats{
		Dispatched:  atomic.LoadInt64(&c.TasksDispatched),
		Completed:   atomic.LoadInt64(&c.TasksCompleted),
		Failed:      atomic.LoadInt64(&c.TasksFailed),
		FatalFailed: atomic.LoadInt64(&c.TasksFatalFailed),
		Retried:     atomic.LoadInt64(&c.TasksRetried),
	}
}

// SetBuildTag sets an optional tag that scopes newly dispatched tasks to a
// particular build run. Pass an empty string to clear the tag.
//
// Takes tag (string) which is the build tag to assign, or empty to clear.
//
// Safe for concurrent use; protected by buildTagMu.
func (c *TaskProcessingCore) SetBuildTag(tag string) {
	c.buildTagMu.Lock()
	c.buildTag = tag
	c.buildTagMu.Unlock()
}

// BuildTag returns the current build tag, or empty if none is set.
//
// Returns string which is the active build tag, or empty when unset.
//
// Safe for concurrent use; protected by buildTagMu.
func (c *TaskProcessingCore) BuildTag() string {
	c.buildTagMu.RLock()
	defer c.buildTagMu.RUnlock()
	return c.buildTag
}

// StartHeartbeat begins a background task that periodically
// updates the task's updated_at timestamp in the database,
// preventing long-running tasks from being recovered by the
// stale task recovery mechanism while still active.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached so heartbeats continue independently of the request.
// Takes taskID (string) which identifies the task to heartbeat.
//
// Concurrent goroutine is spawned that sends periodic heartbeats until
// stopHeartbeat is called for the same task ID.
func (c *TaskProcessingCore) StartHeartbeat(ctx context.Context, taskID string) {
	if c.Config.HeartbeatInterval <= 0 || c.TaskStore == nil {
		return
	}

	detachedCtx := context.WithoutCancel(ctx)
	stopCh := make(chan struct{})
	c.heartbeatStopChans.Store(taskID, stopCh)
	go c.runHeartbeat(detachedCtx, taskID, stopCh)
}

// acquirePersistPermit reserves a non-blocking slot in the persist semaphore.
//
// Returns true when a slot was acquired and the caller may spawn an async
// persistence goroutine. Returns false when the semaphore is saturated or
// unconfigured, signalling the caller to fall back to a synchronous
// persistence path so backpressure flows through to the dispatcher
// rather than spawning unbounded goroutines.
//
// Returns bool which is true when a permit was acquired.
func (c *TaskProcessingCore) acquirePersistPermit() bool {
	if c.persistSemaphore == nil {
		return false
	}
	select {
	case c.persistSemaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// releasePersistPermit returns a slot to the persist semaphore. Called from
// deferred function bodies in goroutines that successfully acquired a permit
// via acquirePersistPermit.
func (c *TaskProcessingCore) releasePersistPermit() {
	if c.persistSemaphore == nil {
		return
	}
	select {
	case <-c.persistSemaphore:
	default:
	}
}

// persistTaskSync synchronously persists the task to the store.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// stripped by the caller so persistence completes independently.
// Takes task (*Task) which is the task to persist.
func (c *TaskProcessingCore) persistTaskSync(ctx context.Context, task *Task) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
		errors.New("task persist exceeded 5s timeout"))
	defer cancel()

	if err := c.TaskStore.UpdateTask(ctx, task); err != nil {
		l.Warn("Failed to persist task update",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, task.ID))
	}
}

// recoveryParams returns the lease timeout and batch limit for task recovery.
//
// Returns time.Duration which is the lease timeout, using the default if not
// configured or non-positive.
// Returns int which is the batch limit, using the default if not configured or
// non-positive.
func (c *TaskProcessingCore) recoveryParams() (time.Duration, int) {
	leaseTimeout := c.Config.RecoveryLeaseTimeout
	if leaseTimeout <= 0 {
		leaseTimeout = defaultRecoveryLeaseTimeout
	}

	batchLimit := c.Config.RecoveryBatchLimit
	if batchLimit <= 0 {
		batchLimit = defaultRecoveryBatchLimit
	}

	return leaseTimeout, batchLimit
}

// runHeartbeat sends periodic heartbeats for a task until signalled to stop.
//
// Takes ctx (context.Context) which carries tracing values with cancellation
// already detached by StartHeartbeat.
// Takes taskID (string) which identifies the task to send heartbeats for.
// Takes stopCh (<-chan struct{}) which signals when to stop sending heartbeats.
func (c *TaskProcessingCore) runHeartbeat(ctx context.Context, taskID string, stopCh <-chan struct{}) {
	ctx, l := logger_domain.From(ctx, log)
	ticker := c.Clock.NewTicker(c.Config.HeartbeatInterval)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(ctx, "orchestrator.runHeartbeat")

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C():
			ctx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
				errors.New("heartbeat update exceeded 5s timeout"))
			err := c.TaskStore.UpdateTaskHeartbeat(ctx, taskID)
			cancel()

			if err != nil {
				l.Trace("Failed to update task heartbeat",
					logger_domain.Error(err),
					logger_domain.String(attributeKeyTaskID, taskID))
			}
		}
	}
}

// stopHeartbeat stops the heartbeat task for the given task,
// safe to call even if no heartbeat is running.
//
// Takes taskID (string) which identifies the task.
func (c *TaskProcessingCore) stopHeartbeat(taskID string) {
	if value, ok := c.heartbeatStopChans.LoadAndDelete(taskID); ok {
		close(value.(chan struct{}))
	}
}
