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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

const (
	// taskPersistTimeout bounds every individual task store write made by the core.
	taskPersistTimeout = 5 * time.Second

	// instanceSuffixStart locates the final twelve hexadecimal digits of a canonical UUID.
	instanceSuffixStart = 24
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

// TaskProcessingCore provides shared execution, settlement, retry, and completion
// tracking for task dispatchers.
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

	// settleWake coalesces task settlements into a notification for completion checks.
	settleWake chan struct{}

	// completionSink receives task completions in place of task.completed events when the
	// dispatcher runs every task in this process; nil publishes events.
	completionSink atomic.Pointer[CompletionSink]

	// completionObserver also sees every task completion, however it is delivered; nil when
	// nothing observes them.
	completionObserver atomic.Pointer[CompletionSink]

	// OnRerunCreated publishes newly created reruns; nil leaves them pending for the next
	// sweep.
	OnRerunCreated func(ctx context.Context, task *Task)

	// pendingSettles holds unwritten settlement results that still count as unsettled,
	// protected by pendingSettlesMu.
	pendingSettles []*pendingSettle

	// EventBus publishes task completion events.
	EventBus EventBus

	// TaskStore saves task changes to storage; nil turns off saving.
	TaskStore TaskStore

	// executors maps executor names to their handlers.
	executors map[string]TaskExecutor

	// nodeID identifies this node; instanceID is derived from it.
	nodeID string

	// instanceID combines the node ID with a random per-process suffix.
	instanceID string

	// buildTag is an optional tag that scopes tasks to a particular build run.
	buildTag string

	// Config holds the dispatcher settings for task processing defaults.
	Config DispatcherConfig

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

	// pendingSettlesMu guards pendingSettles.
	pendingSettlesMu sync.Mutex

	// buildTagMu guards access to the buildTag field.
	buildTagMu sync.RWMutex
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
		completionSink:     atomic.Pointer[CompletionSink]{},
		completionObserver: atomic.Pointer[CompletionSink]{},
		OnRerunCreated:     nil,
		pendingSettles:     nil,
		executors:          make(map[string]TaskExecutor),
		settleWake:         make(chan struct{}, 1),
		Config:             config,
		nodeID:             nodeID,
		instanceID:         nodeID + "-" + NewTaskID()[instanceSuffixStart:],
		executorsMutex:     sync.RWMutex{},
		pendingSettlesMu:   sync.Mutex{},
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

// ApplyDefaults sets default values for any unset task settings and stamps unset
// admission times from the core's clock.
//
// Takes task (*Task) which is updated in place with defaults.
func (c *TaskProcessingCore) ApplyDefaults(task *Task) {
	if task.Config.Timeout <= 0 {
		task.Config.Timeout = c.Config.EffectiveDefaultTimeout()
	}
	if task.Config.MaxRetries <= 0 {
		task.Config.MaxRetries = c.Config.DefaultMaxRetries
	}
	if task.Status == "" {
		task.Status = StatusPending
	}
	task.stampAdmission(c.taskClock().Now())
}

// TimeSource returns the clock the core uses for task times, so components that admit
// tasks alongside it can share one clock.
//
// Returns clockpkg.Clock which is the configured clock, or the real clock when unset.
func (c *TaskProcessingCore) TimeSource() clockpkg.Clock {
	return c.taskClock()
}

// HandleExecutionResult processes the result of task execution. Routes to success or
// failure handling based on the error.
//
// Takes task (*Task) which was executed.
// Takes execErr (error) which is the execution result, or nil on success.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleExecutionResult(
	ctx context.Context,
	task *Task,
	execErr error,
	startTime time.Time,
) {
	if execErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Task execution failed",
			logger_domain.Error(execErr),
			logger_domain.Int(attributeKeyAttempt, task.Attempt))
		c.HandleTaskFailure(ctx, task, execErr, startTime)
		return
	}
	c.HandleTaskSuccess(ctx, task, startTime)
}

// HandleTaskSuccess handles successful task completion.
//
// The completed state, any recurring successor and any requested rerun are written
// together before the task is counted, so a completion is never reported for work the
// store does not record. Each attempt settles once; a second report is ignored.
//
// Takes task (*Task) which completed successfully.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleTaskSuccess(ctx context.Context, task *Task, startTime time.Time) {
	if !beginSettle(task) {
		return
	}
	l := log
	var span trace.Span = noop.Span{}
	if logger_domain.SpansActive(ctx) {
		ctx, l = logger_domain.From(ctx, log)
		ctx, span, l = l.Span(ctx, "TaskProcessingCore.handleTaskSuccess",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
		)
	}
	defer span.End()

	task.Status = StatusComplete
	task.LastError = ""
	task.UpdatedAt = c.Clock.Now()

	next := task.continuation.take()
	task.continuation = nil
	if next != nil {
		c.ApplyDefaults(next)
	}
	settled := c.finishAttempt(ctx, pendingSettle{startTime: startTime, task: task, next: next, cause: nil, kind: settleComplete})
	if !settled {
		return
	}

	if l.Enabled(logger_domain.LevelTrace) {
		l.Trace("Task completed successfully",
			logger_domain.Int("resultSize", len(task.Result)))
	}
	span.SetStatus(codes.Ok, "Task completed")
}

// HandleTaskFailure handles task execution failure with retry logic.
//
// Takes task (*Task) which failed.
// Takes execErr (error) which caused the failure.
// Takes startTime (time.Time) which is when execution began.
func (c *TaskProcessingCore) HandleTaskFailure(ctx context.Context, task *Task, execErr error, startTime time.Time) {
	if task.settled {
		return
	}
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
// The failed state is written before the task is counted and reported. Each attempt
// settles once; a second report is ignored.
//
// Takes task (*Task) which has exhausted retries.
// Takes execErr (error) which caused the final failure.
// Takes startTime (time.Time) which is when execution began.
// Takes span (spanStatus) which records the error status for tracing.
func (c *TaskProcessingCore) MarkTaskFailed(
	ctx context.Context,
	task *Task,
	execErr error,
	startTime time.Time,
	span spanStatus,
) {
	if !beginSettle(task) {
		return
	}
	ctx, l := logger_domain.From(ctx, log)

	task.Status = StatusFailed

	if task.IsFatal {
		l.Trace("Task failed with fatal error",
			logger_domain.Int("totalAttempts", task.Attempt))
	} else {
		l.Trace("Task failed after max retries",
			logger_domain.Int("totalAttempts", task.Attempt))
	}

	task.continuation.take()
	task.continuation = nil
	settled := c.finishAttempt(ctx, pendingSettle{startTime: startTime, task: task, next: nil, cause: execErr, kind: settleFailed})
	if settled {
		span.SetStatus(codes.Error, "Task failed after max retries")
	}
}

// ScheduleTaskRetry schedules a task for retry with exponential backoff.
//
// The retrying state is written before the task is counted and handed to the delayed
// publisher for its due time. Each attempt settles once; a second report is ignored.
//
// Takes task (*Task) which needs to be retried.
// Takes span (spanStatus) which records the operation status.
func (c *TaskProcessingCore) ScheduleTaskRetry(
	ctx context.Context,
	task *Task,
	span spanStatus,
) {
	if !beginSettle(task) {
		return
	}
	ctx, l := logger_domain.From(ctx, log)

	retryDelay := calculateRetryBackoff(task.Attempt, c.Config.EffectiveMaxRetryBackoff(), rand.IntN)
	executeAt := c.Clock.Now().Add(retryDelay)

	task.Status = StatusRetrying
	task.ExecuteAt = executeAt
	task.ScheduledExecuteAt = executeAt

	l.Warn("Task failed, scheduling retry",
		logger_domain.Duration("retryDelay", retryDelay),
		logger_domain.Time("executeAt", executeAt),
		logger_domain.Int("nextAttempt", task.Attempt+1))

	task.continuation.take()
	task.continuation = nil

	settled := c.finishAttempt(ctx, pendingSettle{startTime: time.Time{}, task: task, next: nil, cause: nil, kind: settleRetry})
	if settled {
		span.SetStatus(codes.Ok, "Task scheduled for retry")
	}
}

// SetCompletionSink delivers task completions to sink instead of publishing
// task.completed events.
//
// The sink is replaced atomically.
//
// Takes sink (CompletionSink) which receives the completions; nil restores events.
func (c *TaskProcessingCore) SetCompletionSink(sink CompletionSink) {
	if sink == nil {
		c.completionSink.Store(nil)
		return
	}
	c.completionSink.Store(&sink)
}

// ObserveCompletions calls observer for every task completion as well as delivering it,
// for measurement and tests.
//
// The observer is replaced atomically.
//
// Takes observer (CompletionSink) which sees the completions; nil stops observing.
func (c *TaskProcessingCore) ObserveCompletions(observer CompletionSink) {
	if observer == nil {
		c.completionObserver.Store(nil)
		return
	}
	c.completionObserver.Store(&observer)
}

// PublishCompletionEvent sends completion to the configured sink or publishes a
// task.completed event.
//
// Takes task (*Task) which completed.
// Takes taskErr (error) which is the error, or nil on success.
// Takes duration (time.Duration) which is how long execution took.
func (c *TaskProcessingCore) PublishCompletionEvent(ctx context.Context, task *Task, taskErr error, duration time.Duration) {
	if observer := c.completionObserver.Load(); observer != nil {
		(*observer)(ctx, task, taskErr)
	}
	if sink := c.completionSink.Load(); sink != nil {
		(*sink)(ctx, task, taskErr)
		return
	}
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

// PersistWithDedup synchronously creates a task with an atomic deduplication check
// bounded by taskPersistTimeout.
//
// Takes task (*Task) which has the task to persist.
//
// Returns ErrDuplicateTask if an active task with the same deduplication key exists, or
// any other persistence error.
func (c *TaskProcessingCore) PersistWithDedup(ctx context.Context, task *Task) error {
	if c.TaskStore == nil || task.persisted {
		return nil
	}

	persistCtx, cancel := c.storeContext(ctx, false, errTaskDedupPersistTimeout)
	defer cancel()

	if err := c.TaskStore.CreateTaskWithDedup(persistCtx, task); err != nil {
		return err
	}
	task.persisted = true
	return nil
}

// ReleaseUnrequiredTask completes unnecessary work and frees its deduplication key
// synchronously.
//
// The write is fenced against newer attempts. Cancellation of ctx is detached until the
// write finishes.
//
// Takes task (*Task) which is the task to release.
//
// Returns error when the store write fails, in which case the key may still be held.
func (c *TaskProcessingCore) ReleaseUnrequiredTask(ctx context.Context, task *Task) error {
	task.Status = StatusComplete
	task.LastError = ""
	task.UpdatedAt = c.Clock.Now()

	if c.TaskStore == nil {
		return nil
	}

	if task.claimed {
		outcome, err := c.writeSettlement(ctx, task, nil)
		if err != nil {
			return fmt.Errorf("releasing unrequired task %q: %w", task.ID, err)
		}
		c.publishRerun(ctx, outcome.Rerun)
	} else if err := c.transitionTask(ctx, task, StatusPending, StatusScheduled, StatusRetrying); err != nil {
		return fmt.Errorf("releasing unrequired task %q: %w", task.ID, err)
	}

	TaskNotRequiredCount.Add(ctx, 1)
	c.notifySettled(ctx, task)
	return nil
}

// AbandonUnpublishedTask records a permanent publication failure and releases the task's
// deduplication key.
//
// The write uses a timeout independent of ctx cancellation and emits a completion event.
// The caller updates dispatch counters.
//
// Takes task (*Task) which is the task whose message cannot be published.
// Takes cause (error) which explains why publishing failed.
func (c *TaskProcessingCore) AbandonUnpublishedTask(ctx context.Context, task *Task, cause error) {
	ctx, l := logger_domain.From(ctx, log)
	task.Status = StatusFailed
	task.LastError = cause.Error()
	task.UpdatedAt = c.Clock.Now()
	TaskAbandonedCount.Add(ctx, 1)

	if c.TaskStore != nil {
		if err := c.transitionTask(ctx, task, StatusPending, StatusScheduled, StatusRetrying); err != nil {
			l.Warn("Failed to record unpublished task as failed",
				logger_domain.Error(err),
				logger_domain.String(attributeKeyTaskID, task.ID))
			return
		}
	}
	c.PublishCompletionEvent(ctx, task, cause, 0)
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

// RecoverStaleTasks recovers processing tasks whose owner lease and write activity have
// expired.
//
// Attempts move to RETRYING or, after exhausting retries, FAILED. Callers must dispatch
// returned retries. Stores without compare-and-set support recover nothing.
//
// Returns []*Task which holds the recovered tasks that must be dispatched again, already
// marked as persisted.
// Returns error when the orphans cannot be read.
func (c *TaskProcessingCore) RecoverStaleTasks(ctx context.Context) ([]*Task, error) {
	if c.TaskStore == nil {
		return nil, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "TaskProcessingCore.recoverStaleTasks",
		logger_domain.String("nodeID", c.nodeID))
	defer span.End()

	now := c.taskClock().Now()
	return c.recoverOrphans(ctx, span, now, now.Add(-c.Config.StaleTaskThreshold))
}

// RecoverOrphansAtStart recovers processing tasks whose owners have no live lease,
// regardless of their last update time.
//
// Returns []*Task which holds the recovered tasks that must be dispatched again.
// Returns error when the orphans cannot be read.
func (c *TaskProcessingCore) RecoverOrphansAtStart(ctx context.Context) ([]*Task, error) {
	if c.TaskStore == nil {
		return nil, nil
	}
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "TaskProcessingCore.recoverOrphansAtStart")
	defer span.End()
	now := c.taskClock().Now()
	return c.recoverOrphans(ctx, span, now, now.Add(time.Second))
}

// InstanceID returns the identifier of this process, which owns the tasks it claims and
// holds its node lease.
//
// Returns string which is the node ID with a per-process suffix.
func (c *TaskProcessingCore) InstanceID() string {
	return c.instanceID
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

// Stats returns processing counters read in an order that can overcount outstanding work
// but cannot miss it.
//
// Returns ProcessingStats which contains the current task counters.
func (c *TaskProcessingCore) Stats() ProcessingStats {
	completed := c.TasksCompleted.Load()
	failed := c.TasksFailed.Load()
	fatalFailed := c.TasksFatalFailed.Load()
	retried := c.TasksRetried.Load()
	return ProcessingStats{
		Dispatched:  c.TasksDispatched.Load(),
		Completed:   completed,
		Failed:      failed,
		FatalFailed: fatalFailed,
		Retried:     retried,
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

// SettleSignal returns a channel that coalesces task settlement notifications.
//
// Other state changes do not signal this channel, so completion checks must also poll.
//
// Returns <-chan struct{} which receives after one or more settles.
func (c *TaskProcessingCore) SettleSignal() <-chan struct{} {
	return c.settleWake
}

// SignalSettled wakes a waiter on SettleSignal without blocking. The core calls it for
// the settles it accounts; an adapter that settles a task itself calls it after counting.
func (c *TaskProcessingCore) SignalSettled() {
	select {
	case c.settleWake <- struct{}{}:
	default:
	}
}

// recoverOrphans recovers processing tasks last written before staleBefore whose owner
// holds no live lease. A store without the compare-and-set API recovers nothing.
//
// Takes span (spanStatus) which records the outcome.
// Takes now (time.Time) which is the recovery time.
// Takes staleBefore (time.Time) which is the write time orphans predate.
//
// Returns []*Task which holds the recovered tasks to dispatch again.
// Returns error when the orphans cannot be read.
func (c *TaskProcessingCore) recoverOrphans(ctx context.Context, span spanStatus, now, staleBefore time.Time) ([]*Task, error) {
	_, l := logger_domain.From(ctx, log)
	batchLimit := c.recoveryBatchLimit()
	orphans, err := c.TaskStore.ListOrphanedTasks(ctx, OrphanQuery{
		Now:         now,
		StaleBefore: staleBefore,
		Limit:       batchLimit,
	})
	if errors.Is(err, ErrStoreUnsupported) {
		span.SetStatus(codes.Ok, "Store keeps no claims to recover")
		return nil, nil
	}
	if err != nil {
		l.Warn("Failed to recover stale tasks", logger_domain.Error(err))
		TaskRecoveryErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("listing orphaned tasks: %w", err)
	}

	retry := make([]*Task, 0, len(orphans))
	recovered := 0
	for _, task := range orphans {
		if ctx.Err() != nil {
			break
		}
		if c.instanceID != "" && task.OwnerInstance == c.instanceID {

			continue
		}
		if !c.recoverOrphan(ctx, task, now) {
			continue
		}
		recovered++
		if task.Status == StatusRetrying {
			retry = append(retry, task)
		}
	}

	if recovered > 0 {
		l.Notice("Recovered stale tasks",
			logger_domain.Int("count", recovered),
			logger_domain.Int("toRetry", len(retry)),
			logger_domain.Duration("staleThreshold", c.Config.StaleTaskThreshold))
		TaskRecoveryCount.Add(ctx, int64(recovered))
	}
	span.SetStatus(codes.Ok, "Stale tasks recovered")
	return retry, nil
}

// recoverOrphan moves an orphan to RETRYING or FAILED without advancing its attempt.
//
// Takes task (*Task) which is the orphan, updated in place.
// Takes now (time.Time) which is the recovery time.
//
// Returns bool which is true when this node recovered the task; false when it settled or
// was recovered elsewhere first, or the write failed.
func (c *TaskProcessingCore) recoverOrphan(ctx context.Context, task *Task, now time.Time) bool {
	task.LastError = staleTaskRecoveryError
	task.UpdatedAt = now
	task.OwnerInstance = ""
	if shouldRetryTask(task.Attempt, task.Config.MaxRetries, c.Config.DefaultMaxRetries) {
		task.Status = StatusRetrying
		task.ExecuteAt = now
		task.ScheduledExecuteAt = now
	} else {
		task.Status = StatusFailed
	}

	if err := c.transitionTask(ctx, task, StatusProcessing); err != nil {
		if !errors.Is(err, ErrObsoleteDelivery) && !errors.Is(err, ErrTaskNotFound) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Failed to recover stale task",
				logger_domain.String(attributeKeyTaskID, task.ID),
				logger_domain.Error(err))
			TaskRecoveryErrorCount.Add(ctx, 1)
		}
		return false
	}
	task.persisted = true

	if task.Status == StatusFailed {
		TaskFailureCount.Add(ctx, 1)
		c.PublishCompletionEvent(ctx, task, errors.New(staleTaskRecoveryError), 0)
		c.notifySettled(ctx, task)
		c.SignalSettled()
	}
	return true
}

// transitionTask updates a task using its stored attempt and status and publishes any
// resulting rerun.
//
// Stores without compare-and-set support write the task directly.
//
// Takes task (*Task) which holds the new state at its stored attempt.
// Takes from (...TaskStatus) which lists the statuses the stored task may be in.
//
// Returns error which is ErrObsoleteDelivery when the stored task has moved on, or a
// storage error.
func (c *TaskProcessingCore) transitionTask(ctx context.Context, task *Task, from ...TaskStatus) error {
	outcome, err := c.writeTransition(ctx, task, from)
	if err != nil {
		return err
	}
	c.publishRerun(ctx, outcome.Rerun)
	return nil
}

// writeTransition writes a transition, bounded by taskPersistTimeout. Cancellation of ctx
// is detached so the write is not abandoned part way.
//
// Takes task (*Task) which holds the new state at its stored attempt.
// Takes from ([]TaskStatus) which lists the statuses the stored task may be in.
//
// Returns TransitionOutcome which reports the rerun the move created, if any.
// Returns error which is ErrObsoleteDelivery when the stored task has moved on, or a
// storage error.
func (c *TaskProcessingCore) writeTransition(ctx context.Context, task *Task, from []TaskStatus) (TransitionOutcome, error) {
	ctx, cancel := c.storeContext(ctx, true, errTaskPersistTimeout)
	defer cancel()
	outcome, err := c.TaskStore.TransitionTask(ctx, Transition{Task: task, From: from})
	if errors.Is(err, ErrStoreUnsupported) {
		return TransitionOutcome{}, c.TaskStore.UpdateTask(ctx, task)
	}
	return outcome, err
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

// taskClock returns the configured clock, or the real clock when none is set.
//
// Returns clockpkg.Clock which provides time for the core.
func (c *TaskProcessingCore) taskClock() clockpkg.Clock {
	if c.Clock == nil {
		return clockpkg.RealClock()
	}
	return c.Clock
}

// recoveryBatchLimit returns how many orphans one recovery pass reads.
//
// Returns int which is the configured limit, or the default when it is not positive.
func (c *TaskProcessingCore) recoveryBatchLimit() int {
	if c.Config.RecoveryBatchLimit <= 0 {
		return defaultRecoveryBatchLimit
	}
	return c.Config.RecoveryBatchLimit
}
