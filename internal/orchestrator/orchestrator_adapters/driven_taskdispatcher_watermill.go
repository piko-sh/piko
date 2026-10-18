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
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// attributeKeyTaskID is the attribute key for the task ID in log entries.
	attributeKeyTaskID = "taskID"

	// attributeKeyWorkflowID is the logging attribute key for workflow identifiers.
	attributeKeyWorkflowID = "workflowID"

	// attributeKeyAttempt is the attribute key for the task attempt number.
	attributeKeyAttempt = "attempt"

	// attributeKeyCount is the attribute key for the number of tasks an operation handled.
	attributeKeyCount = "count"

	// payloadKeyPriority is the task message field holding the task's priority.
	payloadKeyPriority = "priority"

	// payloadKeyExecutor is the payload map key for the task executor name.
	payloadKeyExecutor = "executor"

	// payloadKeyDeduplicationKey is the payload key for deduplication.
	payloadKeyDeduplicationKey = "deduplicationKey"

	// shutdownReleaseTimeout bounds each release step performed while the dispatcher shuts
	// down.
	shutdownReleaseTimeout = 5 * time.Second

	// heldRetryInterval is how long the held-task publisher waits before retrying held
	// messages when the bus has not signalled freed capacity.
	heldRetryInterval = 100 * time.Millisecond

	// sweepPassTimeout bounds one lost-message sweep pass.
	sweepPassTimeout = 10 * time.Second

	// sweepBudget caps how many tasks one sweep pass re-dispatches.
	sweepBudget = 1000

	// sweepRoomPollInterval is how often a start sweep that filled its budget checks for
	// room to take the next page.
	sweepRoomPollInterval = 50 * time.Millisecond
)

// dispatcherPhase records whether dispatched tasks are held, published or refused.
type dispatcherPhase int

const (
	// dispatcherPhaseNotStarted holds task references until topic subscriptions exist.
	dispatcherPhaseNotStarted dispatcherPhase = iota

	// dispatcherPhaseRunning publishes dispatched tasks straight to their topics.
	dispatcherPhaseRunning

	// dispatcherPhaseStopped refuses dispatches because no handler would receive them.
	dispatcherPhaseStopped
)

// publishMode selects when a dispatched task's message is published.
type publishMode int

const (
	// publishImmediately publishes the message from the dispatching goroutine.
	publishImmediately publishMode = iota

	// publishDeferred counts the task as dispatched straight away but hands its message to
	// the held-task publisher goroutine. It is used when dispatching from inside a task
	// handler, where publishing to a task topic could wait on the handler itself.
	publishDeferred
)

// publishFailure classifies why a task message could not be published.
type publishFailure int

const (
	// publishTransient means publication should be retried when the bus can accept the
	// message.
	publishTransient publishFailure = iota

	// publishPermanent means the message can never be published, for example because it is
	// larger than the bus accepts; the task is failed.
	publishPermanent

	// publishUnavailable means the bus is gone; the task stays pending in the store for a
	// later sweep to publish.
	publishUnavailable
)

var (
	_ orchestrator_domain.TaskDispatcher = (*watermillTaskDispatcher)(nil)

	_ orchestrator_domain.TaskStoreProvider = (*watermillTaskDispatcher)(nil)

	// errDispatcherAlreadyStarted is returned when Start is called on a dispatcher that has
	// already been started.
	errDispatcherAlreadyStarted = errors.New("dispatcher already started")

	// errNilTask is returned when a nil task is dispatched.
	errNilTask = errors.New("task is nil")

	// errNilDispatchRequirement is returned when DispatchIfRequired is given no requirement.
	errNilDispatchRequirement = errors.New("dispatch requirement is nil")

	// errDispatcherShutdown is the cancellation cause recorded when the dispatcher's run
	// context ends because its parent context was cancelled.
	errDispatcherShutdown = errors.New("task dispatcher shutting down")

	// errSweepTimeout is the cancellation cause recorded when a sweep pass runs too long.
	errSweepTimeout = errors.New("lost-message sweep pass exceeded its timeout")

	// errTaskAlreadyClaimed reports work claimed before its deduplication key could be
	// released.
	errTaskAlreadyClaimed = errors.New("task was claimed before it could be released")
)

// watermillTaskDispatcher implements TaskDispatcher using Watermill for distributed task
// processing. Tasks are published to priority-specific topics and processed by handlers
// with competing-consumer semantics.
//
// Architecture:
//   - Watermill topics per priority level (task.dispatch.high/normal/low)
//   - Handler count per topic controls priority weighting (10:5:2 by default)
//   - EventBus provides Ack/Nack semantics via SubscribeWithHandler
//   - TaskStore used for persistence, deduplication, and crash recovery
//
// This enables distributed task processing across multiple instances whilst maintaining
// the same code path for single-node deployments (GoChannel backend).
type watermillTaskDispatcher struct {
	// eventBus publishes and subscribes to task topics.
	eventBus orchestrator_domain.EventBus

	// rawBus carries raw task payloads when eventBus supports them; otherwise it is nil.
	rawBus orchestrator_domain.RawEventBus

	// runCtx is the context for the dispatcher's lifecycle; cancelled on shutdown.
	runCtx context.Context

	// capacity reports when the event bus frees room for held messages; nil when the bus
	// does not bound the messages it holds.
	capacity orchestrator_domain.PublishCapacity

	// core provides shared task processing logic.
	*orchestrator_domain.TaskProcessingCore

	// cancel stops the dispatcher's background processing when called.
	cancel context.CancelCauseFunc

	// reruns holds the one pending rerun of each deduplication key requested while the key
	// was in use.
	reruns *rerunCoalescer

	// heldWake signals the held-task publisher that deferred tasks are waiting.
	heldWake chan struct{}

	// pool runs tasks on persistent workers per priority, whichever transport delivers them.
	pool *orchestrator_domain.ExecutionPool

	// requirements maps executors to the requirement checks of their tasks dispatched
	// without one; guarded by requirementsMu.
	requirements map[string]orchestrator_domain.RequirementFactory

	// recurring holds the registered recurring task chains by key; guarded by recurringMu.
	recurring map[string]orchestrator_domain.RecurringTask

	// held contains deferred task references and is guarded by phaseMutex.
	held []orchestrator_domain.TaskRef

	// localOverflow holds references the local rings refused.
	localOverflow localOverflow

	// requirementsMu guards requirements.
	requirementsMu sync.RWMutex

	// wg tracks background goroutines for graceful shutdown.
	wg sync.WaitGroup

	// pendingTasks counts tasks that have been sent but not yet handled.
	pendingTasks atomic.Int64

	// phase records whether dispatched tasks are held, published or refused.
	phase dispatcherPhase

	// phaseMutex guards phase and held.
	phaseMutex sync.Mutex

	// recurringMu guards recurring.
	recurringMu sync.Mutex

	// lastMaintenance is when the sweep last purged old tasks and leases, in Unix
	// nanoseconds.
	lastMaintenance atomic.Int64

	// activeHandlers counts handlers that are processing tasks at this moment.
	activeHandlers atomic.Int32

	// started indicates whether the dispatcher is running.
	started atomic.Bool

	// localTransport is true when tasks are delivered through in-process rings rather than
	// as messages on the bus.
	localTransport bool
}

// watermillDispatcherOption configures a watermillTaskDispatcher during creation.
type watermillDispatcherOption func(*watermillTaskDispatcher)

// RegisterExecutor adds a task executor with the given name.
//
// Takes name (string) which identifies the executor for task routing.
// Takes executor (TaskExecutor) which handles tasks of the given type.
func (d *watermillTaskDispatcher) RegisterExecutor(ctx context.Context, name string, executor orchestrator_domain.TaskExecutor) {
	d.TaskProcessingCore.RegisterExecutor(ctx, name, executor)
	orchestrator_domain.ExecutorRegistrationCount.Add(ctx, 1)
}

// Dispatch queues a task for processing by publishing it to the topic that matches its
// priority.
//
// If the task has a DeduplicationKey set, it is saved with a check to prevent duplicates
// before publishing. This keeps only one active task per key across all instances. Tasks
// dispatched before Start has subscribed to the topics are held, up to the overflow
// limit, and published once the subscriptions exist.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to dispatch.
//
// Returns error when the task is nil, validation fails, a duplicate exists, the
// dispatcher has stopped, its overflow is full, or publishing fails.
func (d *watermillTaskDispatcher) Dispatch(ctx context.Context, task *orchestrator_domain.Task) error {
	return d.dispatch(ctx, task, nil, publishImmediately)
}

// DispatchIfRequired dispatches a task like Dispatch, but confirms with required that the
// work is still needed once the task's deduplication key has been claimed and before the
// task is published.
//
// A decision to dispatch is usually made from state read earlier. If another task with
// the same key finished in between, its key is free again and the stale decision would
// run the work a second time. Checking after the claim closes that window, because after
// the claim no other task with the key can start or finish.
//
// When the key is already held by an active task, the request becomes the key's single
// pending rerun, dispatched through the same check once the active task settles.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to dispatch.
// Takes required (orchestrator_domain.DispatchRequirement) which confirms the work is
// still needed.
//
// Returns error when required is nil, the task is a duplicate (the request is then kept
// as the key's pending rerun), the work is no longer required
// (orchestrator_domain.ErrTaskNotRequired), or dispatching fails.
func (d *watermillTaskDispatcher) DispatchIfRequired(
	ctx context.Context,
	task *orchestrator_domain.Task,
	required orchestrator_domain.DispatchRequirement,
) error {
	if required == nil {
		return fmt.Errorf("dispatching task: %w", errNilDispatchRequirement)
	}
	err := d.dispatch(ctx, task, required, publishImmediately)
	if task.DeduplicationKey == "" || !errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
		return err
	}
	return d.coalesceRerun(ctx, task, required, publishImmediately, err)
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
	executeAt = executeAt.UTC()
	task.ScheduledExecuteAt = executeAt
	task.ExecuteAt = executeAt
	task.Status = orchestrator_domain.StatusScheduled

	if err := d.persistDelayed(ctx, task); err != nil {
		return err
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

// Start begins processing tasks from the Watermill topics.
//
// It spawns background goroutines for the recovery loop and delayed publisher, and
// publishes tasks dispatched before the topic subscriptions existed once they do. Blocks
// until the context is cancelled, then refuses further dispatches, stops the execution
// workers, which release the tasks they were running back to pending, and waits for all
// goroutines to complete.
//
// Returns error when the dispatcher was already started or fails to subscribe.
func (d *watermillTaskDispatcher) Start(ctx context.Context) error {
	if d.started.Swap(true) {
		return errDispatcherAlreadyStarted
	}

	d.runCtx, d.cancel = context.WithCancelCause(ctx)
	defer d.cancel(errDispatcherShutdown)

	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting task dispatcher",
		logger_domain.String("transport", d.transportName()),
		logger_domain.Int("highWorkers", d.Config.WatermillHighHandlers),
		logger_domain.Int("normalWorkers", d.Config.WatermillNormalHandlers),
		logger_domain.Int("lowWorkers", d.Config.WatermillLowHandlers))

	heartbeat := d.Config.HeartbeatInterval > 0 && d.TaskStore != nil
	leased := heartbeat && d.registerNode(ctx, d.Config.HeartbeatInterval)

	d.pool.Start(context.WithoutCancel(d.runCtx))
	if !d.localTransport {
		if err := d.subscribeHandlers(); err != nil {
			d.dropHeld(ctx)
			d.stopLocal(ctx)
			return fmt.Errorf("subscribing handlers: %w", err)
		}
	}

	d.DelayedPublisher.Start(d.runCtx)
	if !d.Config.SkipRecovery {
		d.recoverOrphansAtStart(ctx)
	}
	d.startBackgroundLoops(ctx, heartbeat, leased)

	l.Internal("Task dispatcher started")

	<-d.runCtx.Done()

	l.Internal("Task dispatcher shutting down")
	d.dropHeld(ctx)
	d.stopLocal(ctx)

	d.wg.Wait()
	d.DelayedPublisher.Stop()

	l.Internal("Task dispatcher stopped")
	return nil
}

// Stats returns dispatcher statistics that may overcount work moving between stages but
// do not miss it.
//
// Returns orchestrator_domain.DispatcherStats which contains queue lengths, worker
// counts, and task processing metrics.
func (d *watermillTaskDispatcher) Stats() orchestrator_domain.DispatcherStats {
	pendingReruns := d.reruns.size()
	delayed := 0
	if d.DelayedPublisher != nil {
		delayed = d.DelayedPublisher.ForegroundPendingCount()
	}
	ps := d.TaskProcessingCore.Stats()

	stats := orchestrator_domain.DispatcherStats{
		HighQueueLen:     0,
		NormalQueueLen:   0,
		LowQueueLen:      0,
		ActiveWorkers:    d.activeHandlers.Load(),
		TotalWorkers:     d.Config.WatermillHighHandlers + d.Config.WatermillNormalHandlers + d.Config.WatermillLowHandlers,
		TasksDispatched:  ps.Dispatched,
		TasksCompleted:   ps.Completed,
		TasksFailed:      ps.Failed,
		TasksFatalFailed: ps.FatalFailed,
		TasksRetried:     ps.Retried,
		HeldTasks:        d.heldCount(),
		InFlightTasks:    d.PendingSettleCount(),
		DelayedTasks:     delayed,
		PendingReruns:    pendingReruns,
	}
	if d.localTransport {
		stats.HighQueueLen = d.pool.Queued(orchestrator_domain.PriorityHigh)
		stats.NormalQueueLen = d.pool.Queued(orchestrator_domain.PriorityNormal)
		stats.LowQueueLen = d.pool.Queued(orchestrator_domain.PriorityLow)
		stats.HeldTasks += d.localOverflow.size()
	}
	stats.TotalWorkers = d.pool.Workers()
	stats.InFlightTasks += d.pool.Running()
	return stats
}

// Store returns the task store the dispatcher persists tasks to.
//
// Returns orchestrator_domain.TaskStore which is the store, or nil when the dispatcher
// persists nothing.
func (d *watermillTaskDispatcher) Store() orchestrator_domain.TaskStore {
	return d.TaskStore
}

// IsIdle reports whether this dispatcher has no work in hand, by the same counters a
// build's completion check reads.
//
// Returns bool which is true when every dispatched task has settled and nothing is held,
// running, delayed, or queued for a rerun.
//
// Deprecated: use orchestrator_domain.CheckBuildComplete, which also asks the store, so a
// task this process never dispatched, or one whose message was lost, still counts.
func (d *watermillTaskDispatcher) IsIdle() bool {
	if d.pendingTasks.Load() > 0 {
		return false
	}
	_, complete, err := orchestrator_domain.CheckBuildComplete(context.Background(),
		orchestrator_domain.BuildCompletionCheck{
			Dispatcher:    d,
			Store:         nil,
			Outstanding:   nil,
			BuildTag:      "",
			AllForeground: false,
		})
	return err == nil && complete
}

// FailedTasks queries the task store for all tasks in the FAILED state and returns a
// summary of each for user-facing error reporting.
//
// Returns []orchestrator_domain.FailedTaskSummary which contains details of every failed
// task.
// Returns error when the store query fails.
func (d *watermillTaskDispatcher) FailedTasks(ctx context.Context) ([]orchestrator_domain.FailedTaskSummary, error) {
	if d.TaskStore == nil {
		return nil, nil
	}

	buildTag := d.BuildTag()
	var (
		tasks []*orchestrator_domain.Task
		err   error
	)
	if store, ok := d.TaskStore.(orchestrator_domain.BuildCompletionStore); ok && buildTag != "" {
		tasks, err = store.ListFailedForBuild(ctx, buildTag)
	} else {
		tasks, err = d.TaskStore.ListFailedTasks(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("querying failed tasks: %w", err)
	}

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

// persistDelayed creates a deduplicated delayed task or reschedules an existing task
// using its stored attempt.
//
// Takes task (*orchestrator_domain.Task) which is scheduled, with its due time set.
//
// Returns error which wraps ErrDuplicateTask when an active task holds the key, or a
// storage error.
func (d *watermillTaskDispatcher) persistDelayed(ctx context.Context, task *orchestrator_domain.Task) error {
	if task.Persisted() && d.TaskStore != nil {
		_, err := d.TaskStore.TransitionTask(ctx, orchestrator_domain.Transition{
			Task: task,
			From: []orchestrator_domain.TaskStatus{
				orchestrator_domain.StatusPending, orchestrator_domain.StatusScheduled, orchestrator_domain.StatusRetrying,
			},
		})
		if errors.Is(err, orchestrator_domain.ErrStoreUnsupported) {
			err = d.TaskStore.UpdateTask(ctx, task)
		}
		if err != nil {
			return fmt.Errorf("rescheduling delayed task %q: %w", task.ID, err)
		}
		return nil
	}

	if err := d.PersistWithDedup(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
			_, l := logger_domain.From(ctx, log)
			l.Trace("Duplicate delayed task blocked by deduplication",
				logger_domain.String(payloadKeyDeduplicationKey, task.DeduplicationKey))
			orchestrator_domain.TaskDeduplicationBlockedCount.Add(ctx, 1)
		}
		return fmt.Errorf("persisting delayed task with deduplication for %q: %w", task.ID, err)
	}
	return nil
}

// publishRerun dispatches a recorded rerun after checking whether its work is still
// required.
//
// A rerun that cannot be published remains pending for the next sweep.
//
// Takes task (*orchestrator_domain.Task) which is the rerun.
func (d *watermillTaskDispatcher) publishRerun(ctx context.Context, task *orchestrator_domain.Task) {
	err := d.dispatch(context.WithoutCancel(ctx), task, d.requirementFor(task), publishImmediately)
	if errors.Is(err, orchestrator_domain.ErrTaskNotRequired) {
		return
	}
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Rerun left pending for the next sweep",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(err))
	}
}

// startBackgroundLoops publishes held references and starts publication, sweep, recovery,
// and heartbeat goroutines as configured.
//
// Takes heartbeat (bool) which is true when the node heartbeat should run.
// Takes leased (bool) which reports whether Start registered this node's lease.
func (d *watermillTaskDispatcher) startBackgroundLoops(ctx context.Context, heartbeat, leased bool) {
	heldAtStart := d.heldIDs()
	d.startPublishing(ctx)
	if d.localTransport {
		d.wg.Go(d.runOverflowPublisher)
	} else {
		d.wg.Go(d.runHeldPublisher)
	}
	d.wg.Go(func() { d.runSweepLoop(heldAtStart) })

	if d.Config.RecoveryInterval > 0 && !d.Config.SkipRecovery {
		d.wg.Go(d.runRecoveryLoop)
	}
	if heartbeat {
		d.wg.Go(func() { d.runNodeHeartbeat(leased) })
	}
}

// transportName names how the dispatcher delivers tasks, for logs.
//
// Returns string which is "local" or "bus".
func (d *watermillTaskDispatcher) transportName() string {
	if d.localTransport {
		return orchestrator_domain.TransportLocal.String()
	}
	return orchestrator_domain.TransportBus.String()
}

// dispatch validates, persists and publishes a task. When required is set it is checked
// between the persist, which claims the task's deduplication key, and the publish.
//
// A fresh claim also clears the key's pending rerun because the claimed task reads its
// inputs when it runs, after this point, so it covers every request made before it.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to dispatch.
// Takes required (orchestrator_domain.DispatchRequirement) which confirms the work is
// still needed, or nil to publish unconditionally.
// Takes mode (publishMode) which selects whether the message is published now or by the
// held-task publisher.
//
// Returns error when the task is nil or invalid, the dispatcher refuses work, persisting
// fails, the work is no longer required, or publishing fails.
func (d *watermillTaskDispatcher) dispatch(
	ctx context.Context,
	task *orchestrator_domain.Task,
	required orchestrator_domain.DispatchRequirement,
	mode publishMode,
) error {
	if task == nil {
		return rejectNilTask(ctx)
	}

	ctx, span, l := startDispatchSpan(ctx, task)
	defer span.End()

	if err := d.ValidateTask(task); err != nil {
		l.ReportError(span, err, "Task validation failed")
		orchestrator_domain.DispatcherValidationErrorCount.Add(ctx, 1)
		return fmt.Errorf("validating task %q: %w", task.ID, err)
	}

	if err := d.acceptingDispatches(mode); err != nil {
		return fmt.Errorf("dispatching task %q: %w", task.ID, err)
	}

	d.ApplyDefaults(task)

	created, err := d.persistOrUpdateTask(ctx, task)
	if err != nil {
		return fmt.Errorf("persisting task %q: %w", task.ID, err)
	}

	if required != nil {
		err := d.confirmRequired(ctx, task, required)
		if errors.Is(err, errTaskAlreadyClaimed) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	if created && task.DeduplicationKey != "" {
		d.reruns.forget(task.DeduplicationKey)
	}

	if err := d.publishTask(ctx, task, mode); err != nil {
		if created && classifyPublishError(err) == publishPermanent {
			d.abandonTask(ctx, task, err)
		}
		l.ReportError(span, err, "Failed to publish task to topic")
		orchestrator_domain.TaskDispatchErrorCount.Add(ctx, 1)
		return fmt.Errorf("publishing task %q: %w", task.ID, err)
	}

	span.SetStatus(codes.Ok, "Task dispatched")
	return nil
}

// rejectNilTask records a dispatch of a nil task as a validation failure.
//
// Returns error which wraps errNilTask.
func rejectNilTask(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "watermillTaskDispatcher.Dispatch")
	defer span.End()
	span.RecordError(errNilTask)
	orchestrator_domain.DispatcherValidationErrorCount.Add(ctx, 1)
	return fmt.Errorf("dispatching task: %w", errNilTask)
}

// startDispatchSpan opens the span of one dispatch when spans are being recorded, so a
// dispatch without tracing creates no span.
//
// Takes task (*orchestrator_domain.Task) which is the task being dispatched.
//
// Returns context.Context which carries the span when one was opened.
// Returns trace.Span which is the dispatch's span, or a no-op span.
// Returns logger_domain.Logger which logs within the span.
func startDispatchSpan(ctx context.Context, task *orchestrator_domain.Task) (context.Context, trace.Span, logger_domain.Logger) {
	l := log
	var span trace.Span = noop.Span{}
	if logger_domain.SpansActive(ctx) {
		ctx, l = logger_domain.From(ctx, log)
		ctx, span, l = l.Span(ctx, "watermillTaskDispatcher.Dispatch",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.String(attributeKeyWorkflowID, task.WorkflowID),
			logger_domain.String(payloadKeyExecutor, task.Executor),
			logger_domain.Int("priority", int(task.Config.Priority)),
		)
	}
	return ctx, span, l
}

// persistOrUpdateTask handles persisting new tasks or updating existing retries.
//
// Takes task (*orchestrator_domain.Task) which is the task to persist or update.
//
// Returns bool which is true when this call created the task's record.
// Returns error when persistence fails or deduplication blocks the task.
func (d *watermillTaskDispatcher) persistOrUpdateTask(ctx context.Context, task *orchestrator_domain.Task) (bool, error) {
	l := log
	if tag := d.BuildTag(); tag != "" && task.BuildTag == "" && !task.Background {
		task.BuildTag = tag
	}

	isRetry := task.Attempt > 0 || task.Status == orchestrator_domain.StatusRetrying

	if isRetry {
		l.Trace("Retry task updated for re-dispatch",
			logger_domain.Int(attributeKeyAttempt, task.Attempt))
		return false, nil
	}

	alreadyPersisted := task.Persisted()
	if err := d.PersistWithDedup(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
			l.Trace("Duplicate task blocked by deduplication",
				logger_domain.String(payloadKeyDeduplicationKey, task.DeduplicationKey))
			orchestrator_domain.TaskDeduplicationBlockedCount.Add(ctx, 1)
		}
		return false, fmt.Errorf("persisting task with deduplication for %q: %w", task.ID, err)
	}
	return !alreadyPersisted, nil
}

// confirmRequired evaluates required for a task whose deduplication key is claimed.
//
// When the work is no longer needed the claim is released and ErrTaskNotRequired is
// returned. When required fails, or the release cannot be written, the task proceeds to
// publish. The redundant run completes normally and frees the key, whereas a skipped run
// could leave the work undone.
//
// Takes task (*orchestrator_domain.Task) which is the claimed task.
// Takes required (orchestrator_domain.DispatchRequirement) which confirms the work is
// still needed.
//
// Returns error wrapping ErrTaskNotRequired when the claim was released, or
// errTaskAlreadyClaimed when the task moved on before it could be released, so it must
// not be published again.
func (d *watermillTaskDispatcher) confirmRequired(
	ctx context.Context,
	task *orchestrator_domain.Task,
	required orchestrator_domain.DispatchRequirement,
) error {
	ctx, l := logger_domain.From(ctx, log)
	stillRequired, err := required(ctx)
	if err != nil {
		l.Warn("Dispatch requirement check failed, publishing task anyway",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(err))
		return nil
	}
	if stillRequired {
		return nil
	}

	if err := d.ReleaseUnrequiredTask(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrObsoleteDelivery) {
			l.Trace("Unrequired task was already claimed; leaving it to its owner",
				logger_domain.String(attributeKeyTaskID, task.ID))
			return errTaskAlreadyClaimed
		}
		l.Warn("Releasing unrequired task failed, publishing task so its completion frees the key",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(err))
		return nil
	}

	l.Trace("Task no longer required, released without publishing",
		logger_domain.String(payloadKeyDeduplicationKey, task.DeduplicationKey))
	return fmt.Errorf("dispatching task %q: %w", task.ID, orchestrator_domain.ErrTaskNotRequired)
}

// publishTask publishes a reference to the persisted task for a worker to claim.
//
// Takes task (*orchestrator_domain.Task) which is the task to publish.
// Takes mode (publishMode) which selects whether the reference is published now or by the
// held-task publisher.
//
// Returns error when the dispatcher has stopped, its overflow is full, or the bus refuses
// the message for good; a task whose reference is held returns nil.
func (d *watermillTaskDispatcher) publishTask(ctx context.Context, task *orchestrator_domain.Task, mode publishMode) error {
	return d.publishRef(ctx, orchestrator_domain.RefOf(task), mode)
}

// nudge publishes a stored task's reference without validating or changing its record.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
//
// Returns error when the dispatcher has stopped, its overflow is full, or the bus refuses
// the message for good.
func (d *watermillTaskDispatcher) nudge(ctx context.Context, ref orchestrator_domain.TaskRef) error {
	return d.publishRef(ctx, ref, publishImmediately)
}

// publishRef publishes a task reference or holds it until the bus can accept it.
//
// Dispatch counters are updated before publication so idle detection includes the task.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
// Takes mode (publishMode) which selects whether the reference is published now or by the
// held-task publisher.
//
// Returns error when the dispatcher has stopped, its overflow is full, or the bus refuses
// the message for good; a held reference returns nil.
func (d *watermillTaskDispatcher) publishRef(ctx context.Context, ref orchestrator_domain.TaskRef, mode publishMode) error {
	if d.localTransport {
		return d.publishLocal(ctx, ref, mode)
	}

	held, err := d.holdForPublish(ref, mode)
	if err != nil {
		return err
	}
	if held {
		orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)
		return nil
	}

	d.TasksDispatched.Add(1)
	d.pendingTasks.Add(1)

	if err := d.sendRef(ctx, ref); err != nil {
		if classifyPublishError(err) == publishTransient && d.holdForCapacity(ref) {
			orchestrator_domain.TaskHeldForCapacityCount.Add(ctx, 1)
			orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)
			return nil
		}
		d.TasksDispatched.Add(-1)
		d.pendingTasks.Add(-1)
		return fmt.Errorf("publishing to topic %s: %w", d.topicForPriority(ref.Priority), err)
	}

	orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)
	return nil
}

// sendRef publishes a task reference as raw binary when supported, otherwise as a JSON
// event.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
//
// Returns error when the bus refuses the message.
func (d *watermillTaskDispatcher) sendRef(ctx context.Context, ref orchestrator_domain.TaskRef) error {
	topic := d.topicForPriority(ref.Priority)
	if d.rawBus != nil {
		payload := orchestrator_domain.AppendTaskRef(make([]byte, 0, orchestrator_domain.MaxTaskRefSize), ref)
		return d.rawBus.PublishRaw(ctx, topic, payload)
	}
	return d.eventBus.Publish(ctx, topic, orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType(topic),
		Payload: refPayload(ref),
	})
}

// acceptingDispatches reports whether a new dispatch can be taken at all, so a task that
// could never be published is rejected before its record is written.
//
// Takes mode (publishMode) which selects whether the dispatch would be held.
//
// Returns error when the dispatcher has stopped or its overflow is full.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) acceptingDispatches(mode publishMode) error {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()

	if d.phase == dispatcherPhaseStopped {
		return orchestrator_domain.ErrDispatcherStopped
	}
	holds := d.phase == dispatcherPhaseNotStarted || mode == publishDeferred
	if holds && len(d.held) >= d.Config.EffectiveOverflowLimit() {
		return orchestrator_domain.ErrOverflowFull
	}
	return nil
}

// holdForPublish holds the reference, counting its task as dispatched, when the
// dispatcher has not yet subscribed to its topics or the publish is deferred. A deferred
// reference wakes the held-task publisher.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task being dispatched.
// Takes mode (publishMode) which selects whether a running dispatcher holds the task.
//
// Returns bool which is true when the reference was held rather than left to the caller
// to publish.
// Returns error when the dispatcher has stopped or its overflow is full.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) holdForPublish(ref orchestrator_domain.TaskRef, mode publishMode) (bool, error) {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()

	switch d.phase {
	case dispatcherPhaseRunning:
		if mode == publishImmediately {
			return false, nil
		}
	case dispatcherPhaseStopped:
		return false, orchestrator_domain.ErrDispatcherStopped
	default:
	}

	if len(d.held) >= d.Config.EffectiveOverflowLimit() {
		return false, orchestrator_domain.ErrOverflowFull
	}

	d.held = append(d.held, ref)
	d.TasksDispatched.Add(1)
	d.pendingTasks.Add(1)

	if d.phase == dispatcherPhaseRunning {
		d.wakeHeldPublisher()
	}
	return true, nil
}

// holdForCapacity retains a counted task reference until publication capacity becomes
// available and wakes the retry publisher.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task being dispatched.
//
// Returns bool which is true when the reference was held; false when the dispatcher is
// not running or its overflow is full, in which case the task stays pending in the store.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) holdForCapacity(ref orchestrator_domain.TaskRef) bool {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()

	if d.phase != dispatcherPhaseRunning || len(d.held) >= d.Config.EffectiveOverflowLimit() {
		return false
	}
	d.held = append(d.held, ref)
	d.wakeHeldPublisher()
	return true
}

// heldCount returns the number of held references.
//
// Returns int which is the count.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) heldCount() int {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()
	return len(d.held)
}

// wakeHeldPublisher signals the held-task publisher without blocking.
func (d *watermillTaskDispatcher) wakeHeldPublisher() {
	select {
	case d.heldWake <- struct{}{}:
	default:
	}
}

// startPublishing switches the dispatcher to publishing directly and publishes every
// reference held while the topic subscriptions did not exist.
//
// Safe for concurrent use; the phase is switched under phaseMutex.
func (d *watermillTaskDispatcher) startPublishing(ctx context.Context) {
	d.phaseMutex.Lock()
	d.phase = dispatcherPhaseRunning
	d.phaseMutex.Unlock()

	d.publishHeld(ctx)
}

// publishHeld publishes as many held references as the bus has room for, in the order
// they were held, while the dispatcher is running.
//
// The phaseMutex-protected helper takes held references before publication.
//
// A message the bus cannot take now is held again, ahead of anything held since, and the
// rest wait for the next attempt. A message that can never be published fails its task.
func (d *watermillTaskDispatcher) publishHeld(ctx context.Context) {
	held := d.takeHeld()
	if len(held) == 0 {
		return
	}

	for i, ref := range held {
		err := d.sendRef(ctx, ref)
		if err == nil {
			continue
		}
		switch classifyPublishError(err) {
		case publishPermanent:
			d.failHeldRef(ctx, ref, fmt.Errorf("publishing held task to topic %s: %w", d.topicForPriority(ref.Priority), err))
		default:
			d.requeueHeld(held[i:])
			_, l := logger_domain.From(ctx, log)
			l.Trace("Bus cannot take held task messages yet",
				logger_domain.Int("waiting", len(held)-i),
				logger_domain.Error(err))
			return
		}
	}
}

// takeHeld removes and returns the held references the bus can currently accept.
//
// Returns []orchestrator_domain.TaskRef which holds the references to publish, oldest
// first; nil when the dispatcher is not running, nothing is held, or the bus is full.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) takeHeld() []orchestrator_domain.TaskRef {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()

	if d.phase != dispatcherPhaseRunning || len(d.held) == 0 {
		return nil
	}
	count := len(d.held)
	if d.capacity != nil {
		count = min(count, d.capacity.FreeCapacity())
	}
	if count <= 0 {
		return nil
	}

	taken := make([]orchestrator_domain.TaskRef, count)
	copy(taken, d.held)
	remaining := copy(d.held, d.held[count:])
	clear(d.held[remaining:])
	d.held = d.held[:remaining]
	return taken
}

// requeueHeld returns references that could not be published to the front of the held
// list.
//
// Takes refs ([]orchestrator_domain.TaskRef) which are the references to hold again,
// oldest first.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) requeueHeld(refs []orchestrator_domain.TaskRef) {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()
	d.held = append(slices.Clip(refs), d.held...)
}

// runHeldPublisher publishes held references when they are held, when the bus frees
// capacity, and periodically while any are waiting, until the dispatcher stops.
func (d *watermillTaskDispatcher) runHeldPublisher() {
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.watermillTaskDispatcher.runHeldPublisher")

	var capacityFreed <-chan struct{}
	if d.capacity != nil {
		capacityFreed = d.capacity.CapacitySignal()
	}
	retry := time.NewTimer(heldRetryInterval)
	if d.heldCount() == 0 {
		retry.Stop()
	}
	defer retry.Stop()

	for {
		select {
		case <-d.heldWake:
		case <-capacityFreed:
		case <-retry.C:
		case <-d.runCtx.Done():
			return
		}
		d.publishHeld(d.runCtx)
		if d.heldCount() > 0 {
			retry.Reset(heldRetryInterval)
		} else {
			retry.Stop()
		}
	}
}

// dropHeld stops accepting work and clears held references, leaving their tasks pending
// for a later sweep.
//
// Safe for concurrent use; the held references are taken under phaseMutex.
func (d *watermillTaskDispatcher) dropHeld(ctx context.Context) {
	d.phaseMutex.Lock()
	held := d.held
	d.held = nil
	d.phase = dispatcherPhaseStopped
	d.phaseMutex.Unlock()

	if len(held) == 0 {
		return
	}
	d.pendingTasks.Add(-int64(len(held)))
	d.TasksDispatched.Add(-int64(len(held)))
	_, l := logger_domain.From(ctx, log)
	l.Internal("Left held tasks pending for the next sweep", logger_domain.Int(attributeKeyCount, len(held)))
}

// failHeldRef records a permanently unpublishable held task as failed.
//
// The dispatch count is released after storing the failure. An obsolete attempt is only
// uncounted.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the held task.
// Takes cause (error) which explains why it cannot be published.
func (d *watermillTaskDispatcher) failHeldRef(ctx context.Context, ref orchestrator_domain.TaskRef, cause error) {
	ctx, l := logger_domain.From(ctx, log)
	d.pendingTasks.Add(-1)
	task, current := d.storedAttempt(ctx, ref)
	if !current {
		d.TasksDispatched.Add(-1)
		l.Warn("Held task message can never be published; its attempt has already moved on",
			logger_domain.String(attributeKeyTaskID, ref.ID),
			logger_domain.Error(cause))
		return
	}
	d.AbandonUnpublishedTask(ctx, task, cause)
	d.TasksFailed.Add(1)
	orchestrator_domain.TaskFailureCount.Add(ctx, 1)
	d.SignalSettled()
	l.Warn("Held task can never be published",
		logger_domain.String(attributeKeyTaskID, ref.ID),
		logger_domain.Error(cause))
}

// storedAttempt reads the task a reference names from the store.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
//
// Returns *orchestrator_domain.Task which is the stored task, or one holding only the
// reference's fields when there is no store, the read fails or the store has no record of
// it; the fenced write that fails it then decides.
// Returns bool which is false when the store holds a different attempt.
func (d *watermillTaskDispatcher) storedAttempt(ctx context.Context, ref orchestrator_domain.TaskRef) (*orchestrator_domain.Task, bool) {
	fallback := refTask(ref)
	fallback.Config.Priority = ref.Priority
	if d.TaskStore == nil {
		return fallback, true
	}
	rows, err := d.TaskStore.GetTasksByID(ctx, []string{ref.ID})
	if err != nil || len(rows) != 1 {
		return fallback, true
	}
	if rows[0].Attempt != ref.Attempt {
		return nil, false
	}
	return rows[0], true
}

// abandonTask fails an uncounted task whose message cannot be published.
//
// Dispatch and failure counters advance together after the failure is stored.
//
// Takes task (*orchestrator_domain.Task) which is the task to fail.
// Takes cause (error) which explains why it cannot be published.
func (d *watermillTaskDispatcher) abandonTask(ctx context.Context, task *orchestrator_domain.Task, cause error) {
	d.AbandonUnpublishedTask(ctx, task, cause)
	d.TasksDispatched.Add(1)
	d.TasksFailed.Add(1)
	orchestrator_domain.TaskFailureCount.Add(ctx, 1)
	d.SignalSettled()
}

// classifyPublishError decides whether a failed publish can be retried.
//
// Takes err (error) which is the publish failure.
//
// Returns publishFailure which is permanent for messages the bus can never accept,
// unavailable for a closed bus, and transient otherwise.
func classifyPublishError(err error) publishFailure {
	var permanent interface{ Permanent() bool }
	switch {
	case errors.Is(err, ErrEventPayloadTooLarge), errors.As(err, &permanent) && permanent.Permanent():
		return publishPermanent
	case errors.Is(err, orchestrator_domain.ErrServiceClosed):
		return publishUnavailable
	default:
		return publishTransient
	}
}

// subscribeHandlers sets up a single subscription per priority topic.
//
// Each topic receives one Watermill subscription, processed sequentially by the message
// processor goroutine. The Watermill GoChannel pub/sub broadcasts every message to every
// subscriber on a topic, so creating N subscriptions per topic causes the same message to
// be delivered N times and processed N times concurrently. The compiler's Put-then-Rename
// pattern is not safe under that fanout: after the first Rename, subsequent Renames fail
// with ENOENT because the temporary blob has already been moved.
//
// The configured WatermillHighHandlers/Normal/Low values are recorded for observability
// but only one handler runs per topic. True competing-consumer concurrency requires
// either a pub/sub backend that supports it natively or an in-process worker pool that
// drains a single subscription channel.
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
		var err error
		if d.rawBus != nil {
			err = d.rawBus.SubscribeRaw(d.runCtx, p.topic, d.handleTaskMessage)
		} else {
			err = d.eventBus.SubscribeWithHandler(d.runCtx, p.topic, func(ctx context.Context, event orchestrator_domain.Event) error {
				return d.handleTaskEvent(ctx, event, 0)
			})
		}
		if err != nil {
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
//
// Returns error when processing fails, though malformed tasks are acknowledged to prevent
// infinite redelivery. A malformed task is counted as failed, because it was counted as
// dispatched and can never run.
//
// The handler is wrapped in a recover so a panic does not crash the process; recovered
// panics are logged and the message is dropped.
func (d *watermillTaskDispatcher) handleTaskEvent(ctx context.Context, event orchestrator_domain.Event, _ int) error {
	task, err := d.taskFromPayload(event.Payload)
	return d.handleDelivery(ctx, task, err)
}

// handleTaskMessage processes binary task references and legacy JSON task events.
//
// Takes payload ([]byte) which is the message body, valid until the call returns.
//
// Returns error when the task should be delivered again; malformed messages are
// acknowledged and counted as failed.
func (d *watermillTaskDispatcher) handleTaskMessage(ctx context.Context, payload []byte) error {
	if len(payload) > 0 && payload[0] == '{' {
		var event orchestrator_domain.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return d.handleDelivery(ctx, nil, fmt.Errorf("decoding task event: %w", err))
		}
		return d.handleTaskEvent(ctx, event, 0)
	}
	ref, err := orchestrator_domain.DecodeTaskRef(payload)
	if err != nil {
		return d.handleDelivery(ctx, nil, err)
	}
	task := refTask(ref)
	task.Config.Priority = ref.Priority
	return d.handleDelivery(ctx, task, nil)
}

// handleDelivery claims a delivered task's attempt and hands it to a worker of its
// priority.
//
// Takes task (*orchestrator_domain.Task) which identifies the delivered task and attempt;
// the claim fills in the rest from the store.
// Takes decodeErr (error) which is why the message could not be decoded, or nil.
//
// Returns error when delivery should be retried because no worker could be had, or the
// claim failed for a reason that may pass. Obsolete, premature and malformed deliveries
// are acknowledged.
//
// The handler is wrapped in a recover so a panic does not crash the process; recovered
// panics are logged and the message is dropped.
func (d *watermillTaskDispatcher) handleDelivery(ctx context.Context, task *orchestrator_domain.Task, decodeErr error) error {
	acknowledged := true
	defer func() {
		if acknowledged {
			d.pendingTasks.Add(-1)
		}
	}()
	defer goroutine.RecoverPanic(ctx, "orchestrator.watermillTaskDispatcher.handleDelivery")

	if decodeErr != nil {
		d.RecordUndeliverableTask(ctx, decodeErr)
		return nil
	}

	slot, err := d.pool.Acquire(ctx, task.Config.Priority)
	if err != nil {
		acknowledged = false
		return fmt.Errorf("waiting for a worker for task %s: %w", task.ID, err)
	}
	if err := d.ClaimDelivery(ctx, task); err != nil {
		slot.Release()
		if errors.Is(err, orchestrator_domain.ErrPrematureDelivery) {
			d.deferPrematureDelivery(ctx, task)
			return nil
		}
		if errors.Is(err, orchestrator_domain.ErrObsoleteDelivery) {
			d.TasksDispatched.Add(-1)
			orchestrator_domain.ObsoleteDeliveryCount.Add(ctx, 1)
			_, l := logger_domain.From(ctx, log)
			l.Trace("Acknowledged obsolete task delivery", logger_domain.Error(err))
			return nil
		}
		acknowledged = false
		return err
	}
	slot.Dispatch(ctx, task)
	return nil
}

// deferPrematureDelivery reschedules an attempt delivered before its due time.
//
// Its dispatch count is released until republication, while the delayed publisher counts
// the pending work.
//
// Takes task (*orchestrator_domain.Task) which holds the stored state and due time.
func (d *watermillTaskDispatcher) deferPrematureDelivery(ctx context.Context, task *orchestrator_domain.Task) {
	_, l := logger_domain.From(ctx, log)
	taskID, attempt, dueAt := task.ID, task.Attempt, task.ScheduledExecuteAt

	orchestrator_domain.PrematureDeliveryCount.Add(ctx, 1)
	if err := d.DelayedPublisher.Schedule(context.WithoutCancel(ctx), task); err != nil {
		l.Warn("Premature task delivery left for recovery",
			logger_domain.String(attributeKeyTaskID, taskID),
			logger_domain.Int(attributeKeyAttempt, attempt),
			logger_domain.Error(err))
	}
	d.TasksDispatched.Add(-1)

	l.Debug("Re-armed premature task delivery",
		logger_domain.String(attributeKeyTaskID, taskID),
		logger_domain.Int(attributeKeyAttempt, attempt),
		logger_domain.Time("dueAt", dueAt))
}

// processTask runs a delivered task on a worker, waiting for one to be free; before the
// dispatcher starts, it runs the task on the caller.
//
// Takes task (*orchestrator_domain.Task) which is the task to run.
func (d *watermillTaskDispatcher) processTask(ctx context.Context, task *orchestrator_domain.Task, _ int) {
	slot, err := d.pool.Acquire(ctx, task.Config.Priority)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("No worker to run task; it stays pending for the sweep",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(err))
		return
	}
	slot.Dispatch(ctx, task)
}

// heldIDs returns the IDs of the held task references.
//
// Returns map[string]struct{} which holds the IDs, or nil when nothing is held.
//
// Safe for concurrent use; guarded by phaseMutex.
func (d *watermillTaskDispatcher) heldIDs() map[string]struct{} {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()
	if len(d.held) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(d.held))
	for _, ref := range d.held {
		ids[ref.ID] = struct{}{}
	}
	return ids
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

// refPayload encodes a task reference for event buses without raw payload support.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
//
// Returns map[string]any which holds the task's ID, attempt and priority.
func refPayload(ref orchestrator_domain.TaskRef) map[string]any {
	return map[string]any{
		"id":                ref.ID,
		attributeKeyAttempt: ref.Attempt,
		payloadKeyPriority:  int(ref.Priority),
	}
}

// taskFromPayload decodes a task reference or a legacy full-task payload.
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
		Status:             "",
		Config:             orchestrator_domain.TaskConfig{},
		Attempt:            0,
		IsFatal:            false,
		Background:         false,
		OwnerInstance:      "",
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
	}

	d.parseTaskConfigField(payload, task)
	parseTaskStatusField(payload, task)
	parseTaskAttemptField(payload, task)
	parseTaskPriorityField(payload, task)

	return task, nil
}

// parseTaskPriorityField reads the priority a task reference carries. A message from an
// older release carries it inside its config instead, which parseTaskConfigField reads.
//
// Takes payload (map[string]any) which contains the message fields.
// Takes task (*orchestrator_domain.Task) which receives the priority.
func parseTaskPriorityField(payload map[string]any, task *orchestrator_domain.Task) {
	switch priority := payload[payloadKeyPriority].(type) {
	case int:
		task.Config.Priority = orchestrator_domain.TaskPriority(priority)
	case float64:
		task.Config.Priority = orchestrator_domain.TaskPriority(int(priority))
	}
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

// parseTaskConfig decodes a JSON map using TaskConfig's JSON field tags.
//
// Takes configMap (map[string]any) which contains the raw task configuration.
//
// Returns orchestrator_domain.TaskConfig which holds the parsed settings, or the zero
// config when the map cannot be decoded.
func (*watermillTaskDispatcher) parseTaskConfig(configMap map[string]any) orchestrator_domain.TaskConfig {
	var config orchestrator_domain.TaskConfig
	raw, err := json.Marshal(configMap)
	if err != nil {
		return orchestrator_domain.TaskConfig{}
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return orchestrator_domain.TaskConfig{}
	}
	return config
}

// newWatermillTaskDispatcher creates a new Watermill-based task dispatcher.
//
// The delayed publisher is created here rather than in Start, so tasks scheduled before
// the dispatcher starts are kept rather than dropped, and so the field is never written
// while other goroutines read it.
//
// Takes config (DispatcherConfig) which specifies the dispatcher settings.
// Takes eventBus (EventBus) which handles pub/sub for task distribution.
// Takes taskStore (TaskStore) which provides persistence and crash recovery.
// Takes opts (...watermillDispatcherOption) which provides optional configuration.
//
// Returns *watermillTaskDispatcher which is ready to have executors registered.
func newWatermillTaskDispatcher(
	config orchestrator_domain.DispatcherConfig,
	eventBus orchestrator_domain.EventBus,
	taskStore orchestrator_domain.TaskStore,
	opts ...watermillDispatcherOption,
) *watermillTaskDispatcher {
	core := orchestrator_domain.NewTaskProcessingCore(config, eventBus, taskStore, config.Clock)

	d := &watermillTaskDispatcher{}
	d.TaskProcessingCore = core
	d.eventBus = eventBus
	if rawBus, ok := eventBus.(orchestrator_domain.RawEventBus); ok {
		d.rawBus = rawBus
	}
	d.phase = dispatcherPhaseNotStarted
	d.heldWake = make(chan struct{}, 1)
	d.reruns = newRerunCoalescer(config.EffectiveMaxPendingReruns())
	d.OnTaskSettled = d.onTaskSettled
	d.OnRerunCreated = d.publishRerun
	if capacity, ok := eventBus.(orchestrator_domain.PublishCapacity); ok {
		d.capacity = capacity
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.DelayedPublisher == nil {
		d.DelayedPublisher = orchestrator_domain.NewDelayedTaskPublisher(d.nudge, d.Clock)
	}
	d.localTransport = usesLocalTransport(config, eventBus, taskStore)
	d.localOverflow.limit = config.EffectiveOverflowLimit()
	d.newExecutionPool()

	return d
}

// withWatermillDelayedPublisher sets a custom DelayedPublisher for the dispatcher.
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
// Returns string which is the value if found, or empty string if the key is missing or
// the value is not a string.
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
// Returns map[string]any which is the extracted map, or nil if the key does not exist or
// the value is not a map.
func payloadMap(payload map[string]any, key string) map[string]any {
	if v, ok := payload[key].(map[string]any); ok {
		return v
	}
	return nil
}

// payloadTime extracts a time.Time value from a payload map.
//
// JSON-bus round-tripping converts time.Time to an RFC3339 string, so this helper accepts
// both. RFC3339Nano is tried first to preserve sub-second precision, then RFC3339 as a
// fallback. Other types and unparseable strings yield zero time and a logged warning.
//
// Takes payload (map[string]any) which contains the data to search.
// Takes key (string) which identifies the value to extract.
//
// Returns time.Time which is the extracted value, or zero time if the key is missing, an
// unsupported type, or an unparseable string.
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
// Takes task (*orchestrator_domain.Task) which receives the parsed attempt value.
func parseTaskAttemptField(payload map[string]any, task *orchestrator_domain.Task) {
	if attempt, ok := payload[attributeKeyAttempt].(int); ok {
		task.Attempt = attempt
		return
	}
	if attempt, ok := payload[attributeKeyAttempt].(float64); ok {
		task.Attempt = int(attempt)
	}
}
