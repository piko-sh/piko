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
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// taskPersistTimeout bounds every individual task store write made by the core.
	taskPersistTimeout = 5 * time.Second
)

var (
	// errTaskIDRequired is returned when a task is submitted without an ID.
	errTaskIDRequired = errors.New("task ID is required")

	// errTaskWorkflowIDRequired is returned when a task is submitted without a workflow ID.
	errTaskWorkflowIDRequired = errors.New("task workflowID is required")

	// errTaskExecutorRequired is returned when a task is submitted without an executor name.
	errTaskExecutorRequired = errors.New("task executor is required")

	// errTaskPersistTimeout is the cancellation cause when a task store write exceeds
	// taskPersistTimeout.
	errTaskPersistTimeout = errors.New("task persist exceeded 5s timeout")

	// errTaskDedupPersistTimeout is the cancellation cause when creating a task with
	// deduplication exceeds taskPersistTimeout.
	errTaskDedupPersistTimeout = errors.New("task dedup persist exceeded 5s timeout")
)

// staleRecoverySweep describes one stale task recovery pass.
type staleRecoverySweep struct {
	// claimed lists the stale tasks this node claimed.
	claimed []RecoveryClaimedTask

	// retry holds the recovered tasks moved back to RETRYING.
	retry []*Task

	// count is the number of tasks the store recovered.
	count int
}

// executionOutcome carries an executor's result back to the goroutine waiting for it.
type executionOutcome struct {
	// result is the map the executor returned.
	result map[string]any

	// err is the executor's error, or the recovered panic.
	err error
}

// TaskProcessingCore contains the shared task processing logic used by both the local
// channel dispatcher and the Watermill dispatcher.
//
// Encapsulates:
//   - Executor registry and lookup
//   - Task execution with timeout
//   - Success/failure handling with retry logic
//   - Persistence and event publishing
//   - In-flight task tracking for graceful shutdown
//   - Metric counters
//
// Each dispatcher embeds this core and calls its methods for the actual task processing,
// while handling distribution (channels vs topics) separately.
type TaskProcessingCore struct {
	// Clock provides time functions for task timestamps; defaults to real time.
	Clock clockpkg.Clock

	// OtelPropagator passes trace context between services.
	OtelPropagator propagation.TextMapPropagator

	// DelayedPublisher schedules tasks for delayed execution; nil disables delayed retry
	// scheduling.
	DelayedPublisher DelayedPublisher

	// OnTaskSettled is called once a task carrying a deduplication key has settled.
	//
	// A task settles when it reaches a terminal state (complete, failed, or released without
	// running) and its record has been written, so its key is free. Nil disables the
	// notification. Set it before the core is used concurrently.
	OnTaskSettled func(ctx context.Context, task *Task)

	// EventBus publishes task completion events.
	EventBus EventBus

	// TaskStore saves task changes to storage; nil turns off saving.
	TaskStore TaskStore

	// executors maps executor names to their handlers.
	executors map[string]TaskExecutor

	// shutdownCh signals persistence goroutines to abort when closed.
	shutdownCh chan struct{}

	// persistSemaphore bounds the number of concurrent async persistence goroutines. Each
	// Go() call acquires a permit before spawning; on saturation the caller falls back to
	// synchronous persistence so the work still completes but goroutine count stays bounded.
	persistSemaphore chan struct{}

	// heartbeatStopChans maps task ID to a chan struct{} used to stop the heartbeat
	// goroutine for that task.
	heartbeatStopChans sync.Map

	// InFlightTasks tracks tasks that are being processed by this instance for graceful
	// shutdown release. It maps a task ID to a *Task snapshot taken when execution started,
	// so releasing it never touches the task its handler is still running.
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
	TasksCompleted atomic.Int64

	// TasksFailed is the count of tasks that have failed; accessed atomically.
	TasksFailed atomic.Int64

	// TasksFatalFailed is the subset of TasksFailed that were caused by fatal
	// (non-retryable) errors; accessed atomically.
	TasksFatalFailed atomic.Int64

	// TasksRetried counts tasks that have been retried; updated atomically.
	TasksRetried atomic.Int64

	// TasksDispatched counts dispatched tasks. Uses atomic operations for lock-free reads.
	TasksDispatched atomic.Int64

	// executorsMutex guards access to the executors map.
	executorsMutex sync.RWMutex

	// buildTagMu guards access to the buildTag field.
	buildTagMu sync.RWMutex

	// shutdownOnce guards single closure of the shutdown channel.
	shutdownOnce sync.Once
}

// NewTaskProcessingCore creates a new task processing core with the given dependencies.
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
		OnTaskSettled:      nil,
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
		buildTag:           "",
		TasksCompleted:     atomic.Int64{},
		TasksFailed:        atomic.Int64{},
		TasksFatalFailed:   atomic.Int64{},
		TasksRetried:       atomic.Int64{},
		TasksDispatched:    atomic.Int64{},
		buildTagMu:         sync.RWMutex{},
	}
}

// RegisterExecutor adds a task executor with the given name. Must be called before
// processing starts for all executor types.
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
		return nil, fmt.Errorf("%w: %s", ErrExecutorNotFound, name)
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

// PrepareTaskExecution sets up a task for execution. Increments attempt counter, sets
// status, and tracks in-flight.
//
// Takes ctx (context.Context) which carries tracing values for persistence.
// Takes task (*Task) which is being prepared for execution.
//
// Returns the timeout duration to use for execution.
func (c *TaskProcessingCore) PrepareTaskExecution(ctx context.Context, task *Task) time.Duration {
	task.Attempt++
	task.UpdatedAt = c.Clock.Now()
	task.Status = StatusProcessing

	c.InFlightTasks.Store(task.ID, new(*task))

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
// The executor runs on its own goroutine so the wait for it is bounded even when it
// ignores cancellation. Once the timeout has passed and the executor has still not
// returned after the abandon grace, the wait ends with the timeout as the cause. A stuck
// executor therefore fails its task instead of holding the task's topic, and idle
// detection, forever.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which is being executed.
// Takes executor (TaskExecutor) which handles the task.
// Takes timeout (time.Duration) which limits execution time.
//
// Returns error from the executor, the timeout cause when the executor was abandoned, or
// nil on success.
func (c *TaskProcessingCore) ExecuteTask(
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
	var result map[string]any

	_ = l.RunInSpan(ctx, "ExecuteTask", func(_ context.Context, _ logger_domain.Logger) error {
		execStartTime := time.Now()
		result, execErr = c.runExecutor(execCtx, task, executor)

		if result != nil {
			task.Result = result
		}
		TaskExecutionDuration.Record(ctx, float64(time.Since(execStartTime).Milliseconds()))
		return nil
	})

	return execErr
}

// HandleExecutionResult processes the result of task execution. Routes to success or
// failure handling based on the error.
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

	c.TasksCompleted.Add(1)
	TaskSuccessCount.Add(ctx, 1)

	task.Status = StatusComplete
	task.LastError = ""
	task.UpdatedAt = c.Clock.Now()

	c.persistSettledTask(ctx, task)

	c.PublishCompletionEvent(ctx, task, nil, c.Clock.Now().Sub(startTime))
	c.notifySettled(ctx, task)

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

	c.TasksFailed.Add(1)
	if task.IsFatal {
		c.TasksFatalFailed.Add(1)
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

	c.persistSettledTask(ctx, task)
	c.PublishCompletionEvent(ctx, task, execErr, time.Since(startTime))
	c.notifySettled(ctx, task)
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

	c.TasksRetried.Add(1)
	TaskRetryCount.Add(ctx, 1)

	retryDelay := calculateRetryBackoff(task.Attempt, c.Config.EffectiveMaxRetryBackoff(), rand.IntN)
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
// Uses async persistence by default; set config.SyncPersistence=true for synchronous
// mode. During shutdown, automatically falls back to synchronous persistence to avoid
// data loss. When the persist concurrency cap is saturated, the caller also falls back to
// synchronous persistence so the goroutine count stays bounded and the work still
// completes.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is detached
// internally so persistence completes independently.
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

// PersistWithDedup creates a task with deduplication check. Uses the store's
// CreateTaskWithDedup method which handles deduplication atomically.
//
// The write is always synchronous, whatever SyncPersistence says. Its outcome decides
// whether the task may be published at all, so it must be known before the caller
// publishes. Writing in the background would let a duplicate through, and would let a
// handler update the task before its record exists. The write is bounded by
// taskPersistTimeout with a cause.
//
// Takes ctx (context.Context) which provides cancellation.
// Takes task (*Task) which has the task to persist.
//
// Returns ErrDuplicateTask if an active task with the same deduplication key exists, or
// any other persistence error.
func (c *TaskProcessingCore) PersistWithDedup(ctx context.Context, task *Task) error {
	if c.TaskStore == nil || task.persisted {
		return nil
	}

	persistCtx, cancel := context.WithTimeoutCause(ctx, taskPersistTimeout, errTaskDedupPersistTimeout)
	defer cancel()

	if err := c.TaskStore.CreateTaskWithDedup(persistCtx, task); err != nil {
		return err
	}
	task.persisted = true
	return nil
}

// ReleaseUnrequiredTask completes a claimed task without running it.
//
// A requirement check found the task's work already satisfied. Completing the record
// frees the deduplication key for later dispatches, and the write is synchronous so the
// key is free as soon as this returns.
//
// Cancellation of ctx is detached so the release is not abandoned part way.
//
// Takes task (*Task) which is the claimed task to release.
//
// Returns error when the store write fails, in which case the key may still be held.
func (c *TaskProcessingCore) ReleaseUnrequiredTask(ctx context.Context, task *Task) error {
	task.Status = StatusComplete
	task.LastError = ""
	task.UpdatedAt = c.Clock.Now()

	if c.TaskStore == nil {
		return nil
	}

	if err := c.updateTaskWithTimeout(context.WithoutCancel(ctx), task); err != nil {
		return fmt.Errorf("releasing unrequired task %q: %w", task.ID, err)
	}

	TaskNotRequiredCount.Add(ctx, 1)
	c.notifySettled(ctx, task)
	return nil
}

// AbandonUnpublishedTask marks a task record as failed because its message could not be
// published, so its deduplication key no longer blocks a later dispatch of the same work
// by a task that will never run. The write is synchronous and bounded by a timeout.
//
// Cancellation of ctx is detached so the write is not abandoned part way.
//
// Takes task (*Task) which is the task whose message was not published.
// Takes cause (error) which explains why publishing failed.
func (c *TaskProcessingCore) AbandonUnpublishedTask(ctx context.Context, task *Task, cause error) {
	ctx, l := logger_domain.From(ctx, log)
	task.Status = StatusFailed
	task.LastError = cause.Error()
	task.UpdatedAt = c.Clock.Now()

	if c.TaskStore == nil {
		return
	}

	if err := c.updateTaskWithTimeout(context.WithoutCancel(ctx), task); err != nil {
		l.Warn("Failed to record unpublished task as failed",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, task.ID))
		return
	}
	c.notifySettled(ctx, task)
}

// RecordUndeliverableTask accounts for a task message that reached a handler but could
// not be decoded back into a task. The task was counted as dispatched and can never run,
// so it is counted as failed; otherwise idle detection would wait for it forever.
//
// Takes cause (error) which explains why the message could not be decoded.
func (c *TaskProcessingCore) RecordUndeliverableTask(ctx context.Context, cause error) {
	ctx, l := logger_domain.From(ctx, log)
	c.TasksFailed.Add(1)
	TaskFailureCount.Add(ctx, 1)
	l.Warn("Dropping task message that cannot be decoded",
		logger_domain.Error(errors.Join(ErrUndeliverableTask, cause)))
}

// RecordProcessingMetrics records the final processing duration and span attributes.
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

// Shutdown signals all persistence tasks to stop and waits for completion, safe for use
// from multiple callers.
//
// Takes ctx (context.Context) which provides a timeout for waiting.
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

// ReleaseInFlightTasks releases all in-flight tasks back to pending state, allowing other
// instances to pick up the work during graceful shutdown.
//
// Returns only after all pending persistence operations complete or the context times
// out.
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
// Returns int which is the count of active tasks.
func (c *TaskProcessingCore) InFlightCount() int {
	count := 0
	c.InFlightTasks.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// RecoverStaleTasks recovers tasks that have been stuck in PROCESSING state for too long.
//
// The store claims the stale tasks for this node and moves each back to RETRYING, or to
// FAILED once its retries are spent. A RETRYING task has no message on the bus any more,
// so tasks moved back to RETRYING are read back in the same transaction and returned. The
// caller must dispatch them again or the work never happens and their deduplication keys
// stay held.
//
// Returns []*Task which holds the recovered tasks that must be dispatched again, already
// marked as persisted.
// Returns error when claiming, recovering or reading the stale tasks fails.
func (c *TaskProcessingCore) RecoverStaleTasks(ctx context.Context) ([]*Task, error) {
	if c.TaskStore == nil {
		return nil, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.recoverStaleTasks",
		logger_domain.String("nodeID", c.nodeID))
	defer span.End()

	var sweep staleRecoverySweep
	err := c.TaskStore.RunAtomic(ctx, func(ctx context.Context, store TaskStore) error {
		var sweepErr error
		sweep, sweepErr = c.recoverStaleInTransaction(ctx, store)
		return sweepErr
	})
	claimed, retry, count := sweep.claimed, sweep.retry, sweep.count
	if err != nil {
		l.Warn("Failed to recover stale tasks", logger_domain.Error(err))
		TaskRecoveryErrorCount.Add(ctx, 1)
		return nil, err
	}

	if len(claimed) == 0 {
		span.SetStatus(codes.Ok, "No stale tasks to recover")
		return nil, nil
	}

	l.Internal("Claimed stale tasks for recovery",
		logger_domain.Int("claimed", len(claimed)))

	if count > 0 {
		l.Notice("Recovered stale tasks",
			logger_domain.Int("count", count),
			logger_domain.Int("toRetry", len(retry)),
			logger_domain.Duration("staleThreshold", c.Config.StaleTaskThreshold))
		TaskRecoveryCount.Add(ctx, int64(count))
	}

	span.SetStatus(codes.Ok, "Stale tasks recovered")
	return retry, nil
}

// ReleaseRecoveryLeases releases all recovery leases held by this node. Called during
// graceful shutdown to allow other nodes to recover the tasks.
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

	// FatalFailed is the subset of failed tasks caused by fatal (non-retryable) errors.
	FatalFailed int64

	// Retried is the total number of tasks that were retried.
	Retried int64
}

// Stats returns current processing statistics.
//
// Returns ProcessingStats which contains the current task counters.
func (c *TaskProcessingCore) Stats() ProcessingStats {
	return ProcessingStats{
		Dispatched:  c.TasksDispatched.Load(),
		Completed:   c.TasksCompleted.Load(),
		Failed:      c.TasksFailed.Load(),
		FatalFailed: c.TasksFatalFailed.Load(),
		Retried:     c.TasksRetried.Load(),
	}
}

// SetBuildTag sets an optional tag that scopes newly dispatched tasks to a particular
// build run. Pass an empty string to clear the tag.
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

// StartHeartbeat begins a background task that periodically updates the task's updated_at
// timestamp in the database, preventing long-running tasks from being recovered by the
// stale task recovery mechanism while still active.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is detached so
// heartbeats continue independently of the request.
// Takes taskID (string) which identifies the task to heartbeat.
//
// Concurrent goroutine is spawned that sends periodic heartbeats until stopHeartbeat is
// called for the same task ID.
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
// Returns true when a slot was acquired and the caller may spawn an async persistence
// goroutine.
// Returns false when the semaphore is saturated or unconfigured, signalling the caller to
// fall back to a synchronous persistence path so backpressure flows through to the
// dispatcher rather than spawning unbounded goroutines.
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

// releasePersistPermit returns a slot to the persist semaphore. Called from deferred
// function bodies in goroutines that successfully acquired a permit via
// acquirePersistPermit.
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
// Takes ctx (context.Context) which carries tracing values; cancellation is stripped by
// the caller so persistence completes independently.
// Takes task (*Task) which is the task to persist.
func (c *TaskProcessingCore) persistTaskSync(ctx context.Context, task *Task) {
	ctx, l := logger_domain.From(ctx, log)
	if err := c.updateTaskWithTimeout(ctx, task); err != nil {
		l.Warn("Failed to persist task update",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, task.ID))
	}
}

// persistSettledTask writes a task's terminal state. A task carrying a deduplication key
// is written synchronously, so the key is free in the store before anything is told the
// task has settled; other tasks use the normal persistence path.
//
// Cancellation of ctx is detached so the write is not abandoned part way.
//
// Takes task (*Task) which is the task in its terminal state.
func (c *TaskProcessingCore) persistSettledTask(ctx context.Context, task *Task) {
	if task.DeduplicationKey == "" || c.TaskStore == nil {
		c.PersistTaskUpdate(ctx, task)
		return
	}
	c.persistTaskSync(context.WithoutCancel(ctx), task)
}

// notifySettled calls OnTaskSettled for a task carrying a deduplication key.
//
// Takes task (*Task) which is the task that has settled.
func (c *TaskProcessingCore) notifySettled(ctx context.Context, task *Task) {
	if task.DeduplicationKey == "" || c.OnTaskSettled == nil {
		return
	}
	c.OnTaskSettled(ctx, task)
}

// updateTaskWithTimeout writes the task to the store, bounded by taskPersistTimeout with
// a cause.
//
// Takes task (*Task) which is the task to write.
//
// Returns error when the store write fails or times out.
func (c *TaskProcessingCore) updateTaskWithTimeout(ctx context.Context, task *Task) error {
	ctx, cancel := context.WithTimeoutCause(ctx, taskPersistTimeout, errTaskPersistTimeout)
	defer cancel()

	return c.TaskStore.UpdateTask(ctx, task)
}

// runExecutor calls the executor on its own goroutine and waits for it, bounded by the
// context plus the abandon grace. A panic inside the executor is recovered and returned
// as an error without a stack; the stack is logged once.
//
// Takes task (*Task) which supplies the payload and identifies the task in logs.
// Takes executor (TaskExecutor) which runs the task.
//
// Returns map[string]any which is the executor's result.
// Returns error when the executor fails or panics, or when it was abandoned, in which
// case the error wraps the context's cause.
func (c *TaskProcessingCore) runExecutor(
	ctx context.Context,
	task *Task,
	executor TaskExecutor,
) (map[string]any, error) {
	outcome := make(chan executionOutcome, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				_, l := logger_domain.From(ctx, log)
				l.Warn("Task executor panicked",
					logger_domain.String(attributeKeyTaskID, task.ID),
					logger_domain.String("stack_trace", string(debug.Stack())))
				goroutine.PanicRecoveryCount.Add(ctx, 1)
				outcome <- executionOutcome{result: nil, err: fmt.Errorf("panic in task executor: %v", recovered)}
			}
		}()
		result, err := executor.Execute(ctx, task.Payload)
		outcome <- executionOutcome{result: result, err: err}
	}()

	select {
	case finished := <-outcome:
		return finished.result, finished.err
	case <-ctx.Done():
	}

	grace := c.taskClock().NewTimer(c.Config.EffectiveExecutorAbandonGrace())
	defer grace.Stop()

	select {
	case finished := <-outcome:
		return finished.result, finished.err
	case <-grace.C():
		_, l := logger_domain.From(ctx, log)
		TaskExecutorAbandonedCount.Add(ctx, 1)
		cause := context.Cause(ctx)
		l.Error("Abandoned task executor that ignored cancellation",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(cause))
		return nil, fmt.Errorf("abandoned executor that ignored cancellation: %w", cause)
	}
}

// taskClock returns the configured clock, or the real clock when none is set.
//
// Returns clockpkg.Clock which provides time for the core.
func (c *TaskProcessingCore) taskClock() clockpkg.Clock {
	if c.Clock == nil {
		return clockpkg.RealClock()
	}
	return c.Clock
}

// recoveryParams returns the lease timeout and batch limit for task recovery.
//
// Returns time.Duration which is the lease timeout, using the default if not configured
// or non-positive.
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
// Takes ctx (context.Context) which carries tracing values with cancellation already
// detached by StartHeartbeat.
// Takes taskID (string) which identifies the task to send heartbeats for.
// Takes stopCh (<-chan struct{}) which signals when to stop sending heartbeats.
func (c *TaskProcessingCore) runHeartbeat(ctx context.Context, taskID string, stopCh <-chan struct{}) {
	ctx = goroutine.Label(ctx, "orchestrator.runHeartbeat", "task_id", taskID)
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

// stopHeartbeat stops the heartbeat task for the given task, safe to call even if no
// heartbeat is running.
//
// Takes taskID (string) which identifies the task.
func (c *TaskProcessingCore) stopHeartbeat(taskID string) {
	if value, ok := c.heartbeatStopChans.LoadAndDelete(taskID); ok {
		close(value.(chan struct{}))
	}
}

// recoverStaleInTransaction claims this node's stale tasks, recovers them, and reads back
// those moved to RETRYING, all through the store bound to one transaction.
//
// Takes store (TaskStore) which is the store bound to the recovery transaction.
//
// Returns staleRecoverySweep which describes what was claimed and recovered.
// Returns error when claiming, recovering or reading back fails.
func (c *TaskProcessingCore) recoverStaleInTransaction(ctx context.Context, store TaskStore) (staleRecoverySweep, error) {
	leaseTimeout, batchLimit := c.recoveryParams()
	sweep := staleRecoverySweep{claimed: nil, retry: nil, count: 0}

	claimed, err := store.ClaimStaleTasksForRecovery(
		ctx, c.nodeID, c.Config.StaleTaskThreshold, leaseTimeout, batchLimit,
	)
	if err != nil {
		return sweep, fmt.Errorf("claiming stale tasks: %w", err)
	}
	sweep.claimed = claimed
	if len(claimed) == 0 {
		return sweep, nil
	}

	sweep.count, err = store.RecoverClaimedTasks(
		ctx, c.nodeID, c.Config.DefaultMaxRetries, staleTaskRecoveryError,
	)
	if err != nil {
		return sweep, fmt.Errorf("recovering claimed tasks: %w", err)
	}

	sweep.retry, err = readTasksToRetry(ctx, store, claimed)
	return sweep, err
}

// readTasksToRetry reads back the claimed tasks after recovery and keeps those moved to
// RETRYING, marking them as persisted so dispatching them updates their record rather
// than inserting it again.
//
// Takes store (TaskStore) which is the store bound to the recovery transaction.
// Takes claimed ([]RecoveryClaimedTask) which lists the tasks this node recovered.
//
// Returns []*Task which holds the recovered tasks to dispatch again.
// Returns error when the read fails.
func readTasksToRetry(ctx context.Context, store TaskStore, claimed []RecoveryClaimedTask) ([]*Task, error) {
	ids := make([]string, len(claimed))
	for i := range claimed {
		ids[i] = claimed[i].ID
	}

	tasks, err := store.GetTasksByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading recovered tasks: %w", err)
	}

	retry := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Status != StatusRetrying {
			continue
		}
		task.persisted = true
		retry = append(retry, task)
	}
	return retry, nil
}
