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
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

const (
	// attributeKeyTaskID is the attribute key for the task ID in log entries.
	attributeKeyTaskID = "taskID"

	// attributeKeyWorkflowID is the logging attribute key for workflow identifiers.
	attributeKeyWorkflowID = "workflowID"

	// attributeKeyAttempt is the attribute key for the task attempt number.
	attributeKeyAttempt = "attempt"

	// payloadKeyExecutor is the payload map key for the task executor name.
	payloadKeyExecutor = "executor"

	// payloadKeyDeduplicationKey is the payload key for deduplication.
	payloadKeyDeduplicationKey = "deduplicationKey"
)

var _ orchestrator_domain.TaskDispatcher = (*watermillTaskDispatcher)(nil)

// errDispatcherAlreadyStarted is returned when Start is called on a
// dispatcher that has already been started.
var errDispatcherAlreadyStarted = errors.New("dispatcher already started")

// watermillTaskDispatcher implements TaskDispatcher using Watermill for
// distributed task processing. Tasks are published to priority-specific topics
// and processed by handlers with competing-consumer semantics.
//
// Architecture:
//   - Watermill topics per priority level (task.dispatch.high/normal/low)
//   - Handler count per topic controls priority weighting (10:5:2 by default)
//   - EventBus provides Ack/Nack semantics via SubscribeWithHandler
//   - TaskStore used for persistence, deduplication, and crash recovery
//
// This enables distributed task processing across multiple instances whilst
// maintaining the same code path for single-node deployments (GoChannel
// backend).
type watermillTaskDispatcher struct {
	// core provides shared task processing logic.
	*orchestrator_domain.TaskProcessingCore

	// eventBus publishes and subscribes to task topics.
	eventBus orchestrator_domain.EventBus

	// runCtx is the context for the dispatcher's lifecycle; cancelled on shutdown.
	runCtx context.Context

	// cancel stops the dispatcher's background processing when called.
	cancel context.CancelCauseFunc

	// wg tracks background goroutines for graceful shutdown.
	wg sync.WaitGroup

	// pendingTasks counts tasks that have been sent but not yet handled.
	pendingTasks int64

	// activeHandlers counts handlers that are processing tasks at this moment.
	activeHandlers int32

	// started indicates whether the dispatcher is running.
	started atomic.Bool
}

// watermillDispatcherOption configures a watermillTaskDispatcher during
// creation.
type watermillDispatcherOption func(*watermillTaskDispatcher)

// RegisterExecutor adds a task executor with the given name.
//
// Takes ctx (context.Context) which carries logging context.
// Takes name (string) which identifies the executor for task routing.
// Takes executor (TaskExecutor) which handles tasks of the given type.
func (d *watermillTaskDispatcher) RegisterExecutor(ctx context.Context, name string, executor orchestrator_domain.TaskExecutor) {
	d.TaskProcessingCore.RegisterExecutor(ctx, name, executor)
	orchestrator_domain.ExecutorRegistrationCount.Add(ctx, 1)
}

// Dispatch queues a task for processing by publishing it to the topic that
// matches its priority.
//
// If the task has a DeduplicationKey set, it is saved with a check to prevent
// duplicates before publishing. This keeps only one active task per key
// exists across all instances.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to dispatch.
//
// Returns error when the task is nil, validation fails, a duplicate exists, or
// publishing fails.
func (d *watermillTaskDispatcher) Dispatch(ctx context.Context, task *orchestrator_domain.Task) error {
	if task == nil {
		ctx, l := logger_domain.From(ctx, log)
		ctx, span, _ := l.Span(ctx, "watermillTaskDispatcher.Dispatch")
		defer span.End()
		err := errors.New("task is nil")
		span.RecordError(err)
		orchestrator_domain.DispatcherValidationErrorCount.Add(ctx, 1)
		return fmt.Errorf("dispatching task: %w", err)
	}

	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "watermillTaskDispatcher.Dispatch",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
		logger_domain.String(payloadKeyExecutor, task.Executor),
		logger_domain.Int("priority", int(task.Config.Priority)),
	)
	defer span.End()

	if err := d.ValidateTask(task); err != nil {
		l.ReportError(span, err, "Task validation failed")
		orchestrator_domain.DispatcherValidationErrorCount.Add(ctx, 1)
		return fmt.Errorf("validating task %q: %w", task.ID, err)
	}

	d.ApplyDefaults(task)

	if err := d.persistOrUpdateTask(ctx, task); err != nil {
		return fmt.Errorf("persisting task %q: %w", task.ID, err)
	}

	topic := d.topicForPriority(task.Config.Priority)

	l.Trace("Publishing task to Watermill topic",
		logger_domain.String("topic", topic))

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType(topic),
		Payload: d.taskToPayload(task),
	}

	if err := d.eventBus.Publish(ctx, topic, event); err != nil {
		l.ReportError(span, err, "Failed to publish task to topic")
		orchestrator_domain.TaskDispatchErrorCount.Add(ctx, 1)
		return fmt.Errorf("publishing task to topic %s: %w", topic, err)
	}

	atomic.AddInt64(&d.TasksDispatched, 1)
	atomic.AddInt64(&d.pendingTasks, 1)
	orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)

	l.Trace("Task published to topic",
		logger_domain.String("topic", topic))
	span.SetStatus(codes.Ok, "Task dispatched")
	return nil
}

// DispatchDelayed schedules a task to run at a given time.
//
// Takes task (*orchestrator_domain.Task) which is the task to schedule.
// Takes executeAt (time.Time) which is when the task should run.
//
// Returns error when the task is not valid, saving fails, or scheduling fails.
func (d *watermillTaskDispatcher) DispatchDelayed(ctx context.Context, task *orchestrator_domain.Task, executeAt time.Time) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "watermillTaskDispatcher.DispatchDelayed",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
		logger_domain.Time("executeAt", executeAt),
	)
	defer span.End()

	if err := d.ValidateTask(task); err != nil {
		l.ReportError(span, err, "Task validation failed")
		orchestrator_domain.DispatcherValidationErrorCount.Add(ctx, 1)
		return fmt.Errorf("validating delayed task %q: %w", task.ID, err)
	}

	d.ApplyDefaults(task)
	task.ScheduledExecuteAt = executeAt
	task.ExecuteAt = executeAt
	task.Status = orchestrator_domain.StatusScheduled

	if err := d.PersistWithDedup(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
			l.Trace("Duplicate delayed task blocked by deduplication",
				logger_domain.String(payloadKeyDeduplicationKey, task.DeduplicationKey))
			orchestrator_domain.TaskDeduplicationBlockedCount.Add(ctx, 1)
		}
		return fmt.Errorf("persisting delayed task with deduplication for %q: %w", task.ID, err)
	}

	if d.DelayedPublisher == nil {
		l.Warn("No delayed publisher configured, task will not be executed",
			logger_domain.String(attributeKeyTaskID, task.ID))
		return nil
	}

	if err := d.DelayedPublisher.Schedule(ctx, task); err != nil {
		l.ReportError(span, err, "Failed to schedule delayed task")
		orchestrator_domain.DelayedTaskPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("scheduling delayed task: %w", err)
	}

	l.Trace("Task scheduled for delayed execution",
		logger_domain.Duration("delay", executeAt.Sub(d.Clock.Now())))
	span.SetStatus(codes.Ok, "Task scheduled")
	return nil
}

// Start begins processing tasks from Watermill topics.
//
// Returns error when the dispatcher fails to start.
//
// Concurrent goroutines are spawned for the recovery loop and delayed
// publisher. Blocks until the context is cancelled, then waits for all
// goroutines to complete before releasing in-flight tasks and recovery
// leases.
func (d *watermillTaskDispatcher) Start(ctx context.Context) error {
	if d.started.Swap(true) {
		return errDispatcherAlreadyStarted
	}

	d.runCtx, d.cancel = context.WithCancelCause(ctx)

	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting Watermill task dispatcher",
		logger_domain.Int("highHandlers", d.Config.WatermillHighHandlers),
		logger_domain.Int("normalHandlers", d.Config.WatermillNormalHandlers),
		logger_domain.Int("lowHandlers", d.Config.WatermillLowHandlers))

	if err := d.subscribeHandlers(); err != nil {
		return fmt.Errorf("subscribing handlers: %w", err)
	}

	if d.DelayedPublisher == nil {
		d.DelayedPublisher = orchestrator_domain.NewDelayedTaskPublisher(d.Dispatch, d.Clock)
	}
	d.DelayedPublisher.Start(d.runCtx)

	if d.Config.RecoveryInterval > 0 {
		d.wg.Add(1)
		go d.runRecoveryLoop()
	}

	l.Internal("Watermill task dispatcher started")

	<-d.runCtx.Done()

	l.Internal("Watermill task dispatcher shutting down")

	d.wg.Wait()
	d.DelayedPublisher.Stop()

	releaseCtx, releaseCancel := context.WithTimeoutCause(context.WithoutCancel(d.runCtx), 5*time.Second,
		errors.New("task message release exceeded 5s timeout"))
	d.ReleaseInFlightTasks(releaseCtx)
	releaseCancel()

	ctx, cancel := context.WithTimeoutCause(context.WithoutCancel(d.runCtx), 5*time.Second,
		errors.New("task message processing exceeded 5s timeout"))
	if count, err := d.ReleaseRecoveryLeases(ctx); err != nil {
		l.Warn("Failed to release recovery leases during shutdown",
			logger_domain.Error(err))
	} else if count > 0 {
		l.Internal("Released recovery leases during shutdown",
			logger_domain.Int("count", count))
	}
	cancel()

	l.Internal("Watermill task dispatcher stopped")
	return nil
}

// Stats returns current dispatcher statistics.
//
// Returns orchestrator_domain.DispatcherStats which contains queue lengths,
// worker counts, and task processing metrics.
func (d *watermillTaskDispatcher) Stats() orchestrator_domain.DispatcherStats {
	ps := d.TaskProcessingCore.Stats()

	return orchestrator_domain.DispatcherStats{
		HighQueueLen:     0,
		NormalQueueLen:   0,
		LowQueueLen:      0,
		ActiveWorkers:    atomic.LoadInt32(&d.activeHandlers),
		TotalWorkers:     d.Config.WatermillHighHandlers + d.Config.WatermillNormalHandlers + d.Config.WatermillLowHandlers,
		TasksDispatched:  ps.Dispatched,
		TasksCompleted:   ps.Completed,
		TasksFailed:      ps.Failed,
		TasksFatalFailed: ps.FatalFailed,
		TasksRetried:     ps.Retried,
	}
}

// IsIdle returns true when the dispatcher has no work remaining.
//
// Returns bool which is true when there are no pending tasks, no tasks in
// flight, no delayed tasks, and all dispatched tasks have completed or failed.
func (d *watermillTaskDispatcher) IsIdle() bool {
	pending := atomic.LoadInt64(&d.pendingTasks)
	if pending > 0 {
		return false
	}

	ps := d.TaskProcessingCore.Stats()
	if ps.Dispatched > ps.Completed+ps.Failed+ps.Retried {
		return false
	}

	if d.DelayedPublisher != nil && d.DelayedPublisher.PendingCount() > 0 {
		return false
	}

	return d.InFlightCount() == 0
}

// FailedTasks queries the task store for all tasks in the FAILED state and
// returns a summary of each for user-facing error reporting.
//
// Returns []orchestrator_domain.FailedTaskSummary which contains details of
// every failed task.
// Returns error when the store query fails.
func (d *watermillTaskDispatcher) FailedTasks(ctx context.Context) ([]orchestrator_domain.FailedTaskSummary, error) {
	if d.TaskStore == nil {
		return nil, nil
	}

	tasks, err := d.TaskStore.ListFailedTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying failed tasks: %w", err)
	}

	buildTag := d.BuildTag()
	summaries := make([]orchestrator_domain.FailedTaskSummary, 0, len(tasks))
	for _, t := range tasks {
		if buildTag != "" && t.BuildTag != buildTag {
			continue
		}
		summaries = append(summaries, orchestrator_domain.FailedTaskSummary{
			TaskID:     t.ID,
			WorkflowID: t.WorkflowID,
			Executor:   t.Executor,
			LastError:  t.LastError,
			Attempt:    t.Attempt,
			IsFatal:    t.IsFatal,
		})
	}

	return summaries, nil
}

// persistOrUpdateTask handles persisting new tasks or updating existing
// retries.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*orchestrator_domain.Task) which is the task to persist or
// update.
//
// Returns error when persistence fails or deduplication blocks the task.
func (d *watermillTaskDispatcher) persistOrUpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	ctx, l := logger_domain.From(ctx, log)
	if tag := d.BuildTag(); tag != "" && task.BuildTag == "" {
		task.BuildTag = tag
	}

	isRetry := task.Attempt > 0 || task.Status == orchestrator_domain.StatusRetrying

	if isRetry {
		task.Status = orchestrator_domain.StatusPending
		d.PersistTaskUpdate(ctx, task)
		l.Trace("Retry task updated for re-dispatch",
			logger_domain.Int(attributeKeyAttempt, task.Attempt))
		return nil
	}

	if err := d.PersistWithDedup(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
			l.Trace("Duplicate task blocked by deduplication",
				logger_domain.String(payloadKeyDeduplicationKey, task.DeduplicationKey))
			orchestrator_domain.TaskDeduplicationBlockedCount.Add(ctx, 1)
		}
		return fmt.Errorf("persisting task with deduplication for %q: %w", task.ID, err)
	}
	return nil
}

// subscribeHandlers sets up a single subscription per priority topic.
//
// Each topic receives one Watermill subscription, processed sequentially by
// the message processor goroutine. The Watermill GoChannel pub/sub broadcasts
// every message to every subscriber on a topic, so creating N subscriptions
// per topic causes the same message to be delivered N times and processed
// N times concurrently. The compiler's Put-then-Rename pattern is not safe
// under that fanout: after the first Rename, subsequent Renames fail with
// ENOENT because the temporary blob has already been moved.
//
// The configured WatermillHighHandlers/Normal/Low values are recorded for
// observability but only one handler runs per topic. True competing-consumer
// concurrency requires either a pub/sub backend that supports it natively or
// an in-process worker pool that drains a single subscription channel.
//
// Returns error when subscribing to any priority topic fails.
func (d *watermillTaskDispatcher) subscribeHandlers() error {
	priorities := []struct {
		topic             string
		configuredWorkers int
	}{
		{orchestrator_domain.TopicTaskDispatchHigh, d.Config.WatermillHighHandlers},
		{orchestrator_domain.TopicTaskDispatchNormal, d.Config.WatermillNormalHandlers},
		{orchestrator_domain.TopicTaskDispatchLow, d.Config.WatermillLowHandlers},
	}

	_, sl := logger_domain.From(d.runCtx, log)

	for _, p := range priorities {
		handler := func(ctx context.Context, event orchestrator_domain.Event) error {
			return d.handleTaskEvent(ctx, event, 0)
		}

		if err := d.eventBus.SubscribeWithHandler(d.runCtx, p.topic, handler); err != nil {
			return fmt.Errorf("subscribing to %s: %w", p.topic, err)
		}

		sl.Trace("Subscribed to topic with single handler",
			logger_domain.String("topic", p.topic),
			logger_domain.Int("configuredWorkers", p.configuredWorkers))
	}

	return nil
}

// handleTaskEvent processes a task received from a topic.
//
// Takes event (orchestrator_domain.Event) which contains the task payload.
// Takes handlerID (int) which identifies the handler processing this task.
//
// Returns error when processing fails, though malformed tasks are
// acknowledged to prevent infinite redelivery.
//
// The deserialisation step is wrapped in a recover so a malformed payload
// that triggers a panic during reflective parsing does not crash the
// process; recovered panics are logged and the message is dropped.
func (d *watermillTaskDispatcher) handleTaskEvent(ctx context.Context, event orchestrator_domain.Event, handlerID int) error {
	ctx, l := logger_domain.From(ctx, log)
	atomic.AddInt32(&d.activeHandlers, 1)
	defer atomic.AddInt32(&d.activeHandlers, -1)
	defer atomic.AddInt64(&d.pendingTasks, -1)
	defer goroutine.RecoverPanic(ctx, "orchestrator.watermillTaskDispatcher.handleTaskEvent")

	task, err := d.taskFromPayload(event.Payload)
	if err != nil {
		l.Warn("Failed to deserialise task from event",
			logger_domain.Error(err))
		return nil
	}

	d.processTask(ctx, task, handlerID)
	return nil
}

// processTask executes a task using the registered executor.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to execute.
// Takes handlerID (int) which identifies the handler processing this task.
func (d *watermillTaskDispatcher) processTask(ctx context.Context, task *orchestrator_domain.Task, handlerID int) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "watermillTaskDispatcher.processTask",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
		logger_domain.String("executor", task.Executor),
		logger_domain.Int("priority", int(task.Config.Priority)),
		logger_domain.Int("handlerID", handlerID),
	)
	defer span.End()

	startTime := d.Clock.Now()

	defer func() {
		if recovered := recover(); recovered != nil {
			panicError := fmt.Errorf("panic in task executor: %v", recovered)
			l.Error("Task executor panicked",
				logger_domain.String(attributeKeyTaskID, task.ID),
				logger_domain.String("panic_info", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack_trace", string(debug.Stack())),
			)
			span.SetStatus(codes.Error, panicError.Error())
			d.HandleTaskFailure(ctx, task, panicError, startTime)
			goroutine.PanicRecoveryCount.Add(ctx, 1)
		}
	}()

	taskTimeout := d.PrepareTaskExecution(ctx, task)
	l.Trace("Processing task",
		logger_domain.Int(attributeKeyAttempt, task.Attempt),
		logger_domain.Duration("timeout", taskTimeout))

	executor, err := d.GetExecutor(task.Executor)
	if err != nil {
		l.ReportError(span, err, "Executor not found")
		d.HandleTaskFailure(ctx, task, err, startTime)
		return
	}

	execErr := d.ExecuteTask(ctx, task, executor, taskTimeout)
	d.HandleExecutionResult(ctx, task, execErr, startTime)

	d.RecordProcessingMetrics(ctx, span, task, startTime)
}

// runRecoveryLoop checks for stale tasks at regular intervals.
func (d *watermillTaskDispatcher) runRecoveryLoop() {
	defer d.wg.Done()
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.runRecoveryLoop")

	ticker := d.Clock.NewTicker(d.Config.RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			d.runRecoverySweep()
		case <-d.runCtx.Done():
			return
		}
	}
}

// runRecoverySweep performs one stale-task recovery pass and logs the
// outcome. Errors are logged but not returned because the recovery loop
// retries on the next tick.
func (d *watermillTaskDispatcher) runRecoverySweep() {
	recovered, err := d.RecoverStaleTasks(d.runCtx)
	_, l := logger_domain.From(d.runCtx, log)
	if err != nil {
		l.Warn("stale task recovery failed", logger_domain.Error(err))
		return
	}
	if recovered > 0 {
		l.Info("recovered stale tasks", logger_domain.Int("count", recovered))
	}
}

// topicForPriority returns the topic name for a given priority level.
//
// Takes priority (TaskPriority) which specifies the task priority level.
//
// Returns string which is the topic name for task dispatch.
func (*watermillTaskDispatcher) topicForPriority(priority orchestrator_domain.TaskPriority) string {
	switch priority {
	case orchestrator_domain.PriorityHigh:
		return orchestrator_domain.TopicTaskDispatchHigh
	case orchestrator_domain.PriorityLow:
		return orchestrator_domain.TopicTaskDispatchLow
	default:
		return orchestrator_domain.TopicTaskDispatchNormal
	}
}

// taskToPayload serialises a task into an event payload.
//
// Takes task (*orchestrator_domain.Task) which is the task to serialise.
//
// Returns map[string]any which contains the task fields as key-value pairs.
func (*watermillTaskDispatcher) taskToPayload(task *orchestrator_domain.Task) map[string]any {
	return map[string]any{
		"id":                       task.ID,
		"workflowID":               task.WorkflowID,
		payloadKeyExecutor:         task.Executor,
		"payload":                  task.Payload,
		"config":                   task.Config,
		"status":                   task.Status,
		attributeKeyAttempt:        task.Attempt,
		"result":                   task.Result,
		"lastError":                task.LastError,
		payloadKeyDeduplicationKey: task.DeduplicationKey,
		"executeAt":                task.ExecuteAt,
		"scheduledExecuteAt":       task.ScheduledExecuteAt,
		"createdAt":                task.CreatedAt,
		"updatedAt":                task.UpdatedAt,
		"buildTag":                 task.BuildTag,
	}
}

// taskFromPayload deserialises a task from an event payload.
//
// Takes payload (map[string]any) which contains the serialised task data.
//
// Returns *orchestrator_domain.Task which is the reconstructed task.
// Returns error when the task ID is missing or invalid.
func (d *watermillTaskDispatcher) taskFromPayload(payload map[string]any) (*orchestrator_domain.Task, error) {
	id, ok := payload["id"].(string)
	if !ok {
		return nil, errors.New("missing or invalid task ID")
	}

	task := &orchestrator_domain.Task{
		ID:                 id,
		WorkflowID:         payloadString(payload, "workflowID"),
		Executor:           payloadString(payload, payloadKeyExecutor),
		Payload:            payloadMap(payload, "payload"),
		Result:             payloadMap(payload, "result"),
		LastError:          payloadString(payload, "lastError"),
		DeduplicationKey:   payloadString(payload, payloadKeyDeduplicationKey),
		BuildTag:           payloadString(payload, "buildTag"),
		ExecuteAt:          payloadTime(payload, "executeAt"),
		ScheduledExecuteAt: payloadTime(payload, "scheduledExecuteAt"),
		CreatedAt:          payloadTime(payload, "createdAt"),
		UpdatedAt:          payloadTime(payload, "updatedAt"),
	}

	d.parseTaskConfigField(payload, task)
	parseTaskStatusField(payload, task)
	parseTaskAttemptField(payload, task)

	return task, nil
}

// parseTaskConfigField parses the config field from payload into task.
//
// Takes payload (map[string]any) which contains the raw task data.
// Takes task (*orchestrator_domain.Task) which receives the parsed config.
func (d *watermillTaskDispatcher) parseTaskConfigField(payload map[string]any, task *orchestrator_domain.Task) {
	if config, ok := payload["config"].(orchestrator_domain.TaskConfig); ok {
		task.Config = config
		return
	}
	if configMap, ok := payload["config"].(map[string]any); ok {
		task.Config = d.parseTaskConfig(configMap)
	}
}

// parseTaskConfig extracts TaskConfig from a map.
//
// Takes configMap (map[string]any) which contains the raw task configuration.
//
// Returns orchestrator_domain.TaskConfig which holds the parsed settings.
func (*watermillTaskDispatcher) parseTaskConfig(configMap map[string]any) orchestrator_domain.TaskConfig {
	config := orchestrator_domain.TaskConfig{}

	if priority, ok := configMap["Priority"].(float64); ok {
		config.Priority = orchestrator_domain.TaskPriority(int(priority))
	}

	if timeout, ok := configMap["Timeout"].(float64); ok {
		config.Timeout = time.Duration(int64(timeout))
	}

	if maxRetries, ok := configMap["MaxRetries"].(float64); ok {
		config.MaxRetries = int(maxRetries)
	}

	return config
}

// newWatermillTaskDispatcher creates a new Watermill-based task dispatcher.
//
// Takes config (DispatcherConfig) which specifies the dispatcher settings.
// Takes eventBus (EventBus) which handles pub/sub for task
// distribution.
// Takes taskStore (TaskStore) which provides persistence and crash recovery.
// Takes opts (...watermillDispatcherOption) which provides optional
// configuration.
//
// Returns *watermillTaskDispatcher which is ready to have executors registered.
func newWatermillTaskDispatcher(
	config orchestrator_domain.DispatcherConfig,
	eventBus orchestrator_domain.EventBus,
	taskStore orchestrator_domain.TaskStore,
	opts ...watermillDispatcherOption,
) *watermillTaskDispatcher {
	core := orchestrator_domain.NewTaskProcessingCore(config, eventBus, taskStore, config.Clock)

	d := &watermillTaskDispatcher{
		TaskProcessingCore: core,
		eventBus:           eventBus,
		runCtx:             nil,
		cancel:             nil,
		wg:                 sync.WaitGroup{},
		pendingTasks:       0,
		activeHandlers:     0,
		started:            atomic.Bool{},
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// withWatermillDelayedPublisher sets a custom DelayedPublisher for the
// dispatcher.
//
// Takes dp (DelayedPublisher) which provides delayed message publishing.
//
// Returns watermillDispatcherOption which configures the dispatcher.
func withWatermillDelayedPublisher(dp orchestrator_domain.DelayedPublisher) watermillDispatcherOption {
	return func(d *watermillTaskDispatcher) {
		d.DelayedPublisher = dp
	}
}

// withWatermillClock sets a custom Clock for testing.
//
// Takes c (clockpkg.Clock) which provides the time source.
//
// Returns watermillDispatcherOption which sets up the dispatcher.
func withWatermillClock(c clockpkg.Clock) watermillDispatcherOption {
	return func(d *watermillTaskDispatcher) {
		d.Clock = c
	}
}

// payloadString extracts a string value from a payload map.
//
// Takes payload (map[string]any) which contains the data to search.
// Takes key (string) which identifies the value to extract.
//
// Returns string which is the value if found, or empty string if the key is
// missing or the value is not a string.
func payloadString(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}

// payloadMap extracts a map value from a payload map.
//
// Takes payload (map[string]any) which is the source map to extract from.
// Takes key (string) which is the key to look up in the payload.
//
// Returns map[string]any which is the extracted map, or nil if the key does
// not exist or the value is not a map.
func payloadMap(payload map[string]any, key string) map[string]any {
	if v, ok := payload[key].(map[string]any); ok {
		return v
	}
	return nil
}

// payloadTime extracts a time.Time value from a payload map.
//
// JSON-bus round-tripping converts time.Time to an RFC3339 string, so this
// helper accepts both. RFC3339Nano is tried first to preserve sub-second
// precision, then RFC3339 as a fallback. Other types and unparseable
// strings yield zero time and a logged warning.
//
// Takes payload (map[string]any) which contains the data to search.
// Takes key (string) which identifies the value to extract.
//
// Returns time.Time which is the extracted value, or zero time if the key
// is missing, an unsupported type, or an unparseable string.
func payloadTime(payload map[string]any, key string) time.Time {
	value, exists := payload[key]
	if !exists || value == nil {
		return time.Time{}
	}

	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		if typed == "" {
			return time.Time{}
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed
		}
		_, l := logger_domain.From(context.Background(), log)
		l.Warn("payloadTime received unparseable RFC3339 string",
			logger_domain.String("key", key),
			logger_domain.String("value", typed))
		return time.Time{}
	default:
		_, l := logger_domain.From(context.Background(), log)
		l.Warn("payloadTime received unsupported type",
			logger_domain.String("key", key),
			logger_domain.String("type", fmt.Sprintf("%T", value)))
		return time.Time{}
	}
}

// parseTaskStatusField parses the status field from payload into task.
//
// Takes payload (map[string]any) which contains the raw task data.
// Takes task (*orchestrator_domain.Task) which receives the parsed status.
func parseTaskStatusField(payload map[string]any, task *orchestrator_domain.Task) {
	if status, ok := payload["status"].(orchestrator_domain.TaskStatus); ok {
		task.Status = status
		return
	}
	if statusString, ok := payload["status"].(string); ok {
		task.Status = orchestrator_domain.TaskStatus(statusString)
	}
}

// parseTaskAttemptField parses the attempt field from payload into task.
//
// Takes payload (map[string]any) which contains the raw task data.
// Takes task (*orchestrator_domain.Task) which receives the parsed attempt
// value.
func parseTaskAttemptField(payload map[string]any, task *orchestrator_domain.Task) {
	if attempt, ok := payload[attributeKeyAttempt].(int); ok {
		task.Attempt = attempt
		return
	}
	if attempt, ok := payload[attributeKeyAttempt].(float64); ok {
		task.Attempt = int(attempt)
	}
}
