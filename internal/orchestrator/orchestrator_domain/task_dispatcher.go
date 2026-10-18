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
	"time"

	clockpkg "piko.sh/piko/wdk/clock"
)

const (
	// backoffBase is the base value used in exponential backoff calculations.
	backoffBase = 10.0

	// dispatchFailureRetryDelay is the delay before retrying a failed dispatch. Used when
	// the delayed task publisher fails to dispatch a due task.
	dispatchFailureRetryDelay = 5 * time.Second

	// defaultTaskTimeout is the default time limit for running a task.
	defaultTaskTimeout = 5 * time.Minute

	// defaultMaxRetries is the maximum number of retry attempts for failed tasks.
	defaultMaxRetries = 3

	// defaultRecoveryInterval is how often to check for stale PROCESSING tasks.
	defaultRecoveryInterval = 30 * time.Second

	// defaultStaleTaskThreshold is how long before a PROCESSING task is considered stuck.
	defaultStaleTaskThreshold = 10 * time.Minute

	// defaultHeartbeatInterval sets how often the node lease is renewed and running tasks
	// refreshed. Must be significantly less than StaleTaskThreshold to prevent premature
	// task recovery.
	defaultHeartbeatInterval = 30 * time.Second

	// defaultRecoveryBatchLimit is the maximum number of tasks to claim per recovery cycle.
	defaultRecoveryBatchLimit = 100

	// staleTaskRecoveryError is the error message saved on tasks that are recovered after a
	// worker has crashed or timed out.
	staleTaskRecoveryError = "task recovered: worker crashed or timed out"

	// defaultWatermillHighHandlers is the default handler count for high-priority tasks. The
	// ratio 10:5:2 mirrors the default polling weights for priority fairness.
	defaultWatermillHighHandlers = 10

	// defaultWatermillNormalHandlers is the default handler count for normal-priority tasks.
	defaultWatermillNormalHandlers = 5

	// defaultWatermillLowHandlers is the default handler count for low-priority tasks.
	defaultWatermillLowHandlers = 2

	// attributeKeyTaskID is the logging attribute key for task identifiers.
	attributeKeyTaskID = "taskID"

	// attributeKeyWorkflowID is the logging attribute key for workflow identifiers.
	attributeKeyWorkflowID = "workflowID"

	// attributeKeyAttempt is the attribute key for the task attempt number.
	attributeKeyAttempt = "attempt"

	// defaultOverflowLimit bounds task references held outside the transport.
	defaultOverflowLimit = 1 << 16

	// defaultMaxRetryBackoff caps the exponential retry delay so large retry budgets cannot
	// produce unbounded or overflowing waits.
	defaultMaxRetryBackoff = 30 * time.Minute

	// defaultExecutorAbandonGrace is how long an executor may keep running after its timeout
	// before the dispatcher stops waiting for it.
	defaultExecutorAbandonGrace = 30 * time.Second

	// defaultMaxPendingReruns is the default number of deduplication keys that may each hold
	// one pending rerun.
	defaultMaxPendingReruns = 10_000

	// defaultSweepInterval is how often a dispatcher re-dispatches due tasks whose messages
	// may have been lost.
	defaultSweepInterval = 30 * time.Second
)

// DispatchRequirement reports whether the work a task would do is required.
//
// DispatchIfRequired evaluates it after the task's deduplication key has been claimed and
// before the task is published. Because no other task with the same key can start or
// finish in between, a decision that was based on state read before the claim can be
// confirmed against current state without a window in which the work could complete
// unnoticed.
//
// Returns bool which is true when the task must still run.
// Returns error when the requirement cannot be determined.
type DispatchRequirement func(ctx context.Context) (bool, error)

// TaskDispatcher distributes tasks to workers using competing-consumer semantics. Each
// task is processed by exactly one worker, regardless of how many are running.
//
// Architecture:
//   - Tasks are published to Watermill topics per priority level
//   - Handler pools subscribe to topics (competing consumers)
//   - Event bus used for coordination signals (task.completed, workflow.registered)
//   - TaskStore used for persistence and crash recovery
//
// Create a TaskDispatcher using orchestrator_adapters.CreateTaskDispatcher().
type TaskDispatcher interface {
	// Dispatch queues a task for immediate processing by the worker pool. Non-blocking
	// unless the queue is full, which applies backpressure.
	//
	// Takes task (*Task) which specifies the work to be processed.
	//
	// Returns error when the context is cancelled or task validation fails.
	Dispatch(ctx context.Context, task *Task) error

	// DispatchIfRequired dispatches a task once required confirms the work is needed.
	//
	// The check runs after the task's deduplication key has been claimed and before the task
	// is published. When required reports false the claim is released, nothing is published
	// and ErrTaskNotRequired is returned. When required fails the task is published anyway,
	// because running work twice is recoverable but silently skipping it is not.
	//
	// When an active task holds the key, the request is coalesced into one pending rerun
	// (later requests replace it rather than queue behind it), and once the active task
	// settles the rerun is dispatched through the same requirement check, so it is skipped
	// when the finished task already covered it.
	//
	// Takes task (*Task) which specifies the work to be processed.
	// Takes required (DispatchRequirement) which confirms the work is still needed.
	//
	// Returns error when validation fails, the key is already claimed by an active task
	// (ErrDuplicateTask, with the request kept as the key's pending rerun), the work is no
	// longer needed (ErrTaskNotRequired) or publishing fails.
	DispatchIfRequired(ctx context.Context, task *Task, required DispatchRequirement) error

	// DispatchDelayed schedules a task to run at a later time. Uses a min-heap with a
	// sleep-until-due pattern (no polling).
	//
	// Takes task (*Task) which is the task to schedule.
	// Takes executeAt (time.Time) which is when the task should run.
	//
	// Returns error when the task cannot be scheduled.
	DispatchDelayed(ctx context.Context, task *Task, executeAt time.Time) error

	// RegisterExecutor adds a task executor with the given name. Must be called before
	// Start() for all executor types.
	//
	// Takes name (string) which identifies the executor.
	// Takes executor (TaskExecutor) which handles tasks of the named kind.
	RegisterExecutor(ctx context.Context, name string, executor TaskExecutor)

	// Start begins the worker pool and delayed task processor. Blocks until the context is
	// cancelled.
	//
	// Returns error when the worker pool fails to start or encounters a fatal error.
	Start(ctx context.Context) error

	// Stats returns the current dispatcher statistics for monitoring.
	//
	// Returns DispatcherStats which is a snapshot of queue and worker counters.
	Stats() DispatcherStats

	// IsIdle reports whether all dispatched tasks have settled and no queued, delayed, or
	// executing work remains.
	//
	// Returns bool which reports whether the dispatcher has no work left.
	//
	// Deprecated: use CheckBuildComplete, which also asks the task store, so work this
	// process never dispatched, or whose message was lost, still counts.
	IsIdle() bool

	// FailedTasks returns a summary of all tasks currently in the FAILED state. When a build
	// tag is set, only tasks matching that tag are returned.
	//
	// Returns []FailedTaskSummary which contains the failed task details.
	// Returns error when the underlying store query fails.
	FailedTasks(ctx context.Context) ([]FailedTaskSummary, error)

	// SetBuildTag sets an optional tag that scopes newly dispatched tasks to a particular
	// build run.
	//
	// Pass an empty string to clear the tag.
	//
	// Takes tag (string) which scopes newly dispatched tasks to one build run.
	SetBuildTag(tag string)

	// BuildTag returns the current build tag, or empty if none is set.
	//
	// Returns string which is the current build tag, or empty when none is set.
	BuildTag() string
}

// DispatcherConfig configures the task dispatcher behaviour.
type DispatcherConfig struct {
	// Clock provides time operations, defaulting to the real system clock when nil and
	// accepting a mock clock for deterministic testing of retry backoff and delayed task
	// scheduling.
	Clock clockpkg.Clock

	// NodeID identifies this node; each process derives its instance ID, which owns the
	// tasks it claims and the node lease, from it. If empty, a UUID is generated at startup.
	NodeID string

	// RecoveryInterval is the interval for checking stale PROCESSING tasks. Set to 0 to
	// disable recovery; default is 30 seconds.
	RecoveryInterval time.Duration

	// StaleTaskThreshold is how long a task can be in PROCESSING status before being
	// considered stuck and eligible for recovery. Default is 10 minutes.
	StaleTaskThreshold time.Duration

	// HeartbeatInterval controls node lease renewal and running-task refresh.
	//
	// It must be well below StaleTaskThreshold. Zero disables heartbeats; the default is 30
	// seconds.
	HeartbeatInterval time.Duration

	// DefaultMaxRetries is the retry count for tasks that do not specify one.
	DefaultMaxRetries int

	// RecoveryBatchLimit is the maximum number of tasks to claim per recovery cycle. Default
	// is 100.
	RecoveryBatchLimit int

	// WatermillHighHandlers is the number of concurrent handlers for high-priority tasks,
	// where more handlers increase throughput for that priority level, defaulting to 10.
	WatermillHighHandlers int

	// WatermillNormalHandlers is the number of concurrent handlers for normal-priority
	// tasks. Default is 5.
	WatermillNormalHandlers int

	// WatermillLowHandlers is the number of concurrent handlers for low-priority tasks.
	// Default is 2.
	WatermillLowHandlers int

	// DefaultTimeout is the timeout used for tasks that do not set their own.
	DefaultTimeout time.Duration

	// MaxRetryBackoff caps the exponential delay between retries of a failed task. Zero or
	// negative uses the default of 30 minutes.
	MaxRetryBackoff time.Duration

	// ExecutorAbandonGrace is how long an executor may run on after its timeout.
	//
	// Once it passes, the dispatcher stops waiting and fails the attempt with the timeout as
	// its cause, which bounds the damage of an executor that ignores cancellation. Zero or
	// negative uses the default of 30 seconds.
	ExecutorAbandonGrace time.Duration

	// MaxPendingReruns bounds how many deduplication keys may each hold a pending rerun.
	//
	// A key gets one pending rerun when a dispatch is requested while it is in use. Beyond
	// the limit new rerun requests are dropped with a warning. Zero or negative uses the
	// default of 10,000.
	MaxPendingReruns int

	// OverflowLimit bounds task references held outside the transport.
	//
	// At capacity, dispatch before startup and deferred dispatch return ErrOverflowFull;
	// other tasks remain pending for a sweep. Nonpositive values use 65,536.
	OverflowLimit int

	// SweepInterval controls periodic republication of due tasks.
	//
	// Zero uses 30 seconds. A negative value disables periodic sweeps but retains the
	// startup sweep.
	SweepInterval time.Duration

	// Transport selects how tasks reach workers. TransportAuto, the default, delivers them
	// in-process when the event bus is node-local, and over the bus otherwise.
	Transport TransportMode

	// SkipRecovery disables orphan recovery when running from another process's task-store
	// snapshot.
	SkipRecovery bool
}

// EffectiveDefaultTimeout returns the configured default task timeout, falling back to
// the built-in default when unset or non-positive.
//
// Returns time.Duration which is the timeout for tasks that do not set their own.
func (c DispatcherConfig) EffectiveDefaultTimeout() time.Duration {
	if c.DefaultTimeout > 0 {
		return c.DefaultTimeout
	}
	return defaultTaskTimeout
}

// EffectiveOverflowLimit returns the configured limit on task references held outside the
// transport, falling back to the default when unset or non-positive.
//
// Returns int which is the effective overflow limit.
func (c DispatcherConfig) EffectiveOverflowLimit() int {
	if c.OverflowLimit > 0 {
		return c.OverflowLimit
	}
	return defaultOverflowLimit
}

// EffectiveMaxPendingReruns returns the configured limit on pending reruns, falling back
// to the default when unset or non-positive.
//
// Returns int which is the effective pending rerun limit.
func (c DispatcherConfig) EffectiveMaxPendingReruns() int {
	if c.MaxPendingReruns > 0 {
		return c.MaxPendingReruns
	}
	return defaultMaxPendingReruns
}

// EffectiveSweepInterval returns how often to sweep for lost dispatches, falling back to
// the default when unset.
//
// Returns time.Duration which is the interval, or a negative value when periodic sweeps
// are disabled.
func (c DispatcherConfig) EffectiveSweepInterval() time.Duration {
	if c.SweepInterval == 0 {
		return defaultSweepInterval
	}
	return c.SweepInterval
}

// EffectiveMaxRetryBackoff returns the configured cap on retry delays, falling back to
// the default when unset or non-positive.
//
// Returns time.Duration which is the effective retry delay cap.
func (c DispatcherConfig) EffectiveMaxRetryBackoff() time.Duration {
	if c.MaxRetryBackoff > 0 {
		return c.MaxRetryBackoff
	}
	return defaultMaxRetryBackoff
}

// EffectiveExecutorAbandonGrace returns how long to keep waiting for an executor after
// its timeout, falling back to the default when unset or non-positive.
//
// Returns time.Duration which is the effective abandon grace.
func (c DispatcherConfig) EffectiveExecutorAbandonGrace() time.Duration {
	if c.ExecutorAbandonGrace > 0 {
		return c.ExecutorAbandonGrace
	}
	return defaultExecutorAbandonGrace
}

// DispatcherStats holds runtime figures about the dispatcher for monitoring.
type DispatcherStats struct {
	// HighQueueLen is the number of tasks in the high priority queue.
	HighQueueLen int

	// NormalQueueLen is the number of tasks waiting in the normal priority queue.
	NormalQueueLen int

	// LowQueueLen is the number of tasks in the low priority queue.
	LowQueueLen int

	// ActiveWorkers is the number of workers that are running tasks right now.
	ActiveWorkers int32

	// TotalWorkers is the number of workers in the dispatcher pool.
	TotalWorkers int

	// TasksDispatched is the total number of tasks sent to workers.
	TasksDispatched int64

	// TasksCompleted is the number of tasks that have finished successfully.
	TasksCompleted int64

	// TasksFailed is the number of tasks that have failed.
	TasksFailed int64

	// TasksRetried is the number of tasks that have been retried after failure.
	TasksRetried int64

	// TasksFatalFailed is the subset of TasksFailed caused by fatal (non-retryable) errors.
	TasksFatalFailed int64

	// HeldTasks counts dispatched references retained until publication capacity becomes
	// available.
	HeldTasks int

	// InFlightTasks is the number of tasks this dispatcher is executing now, plus the
	// finished attempts whose settled state is still waiting to be written.
	InFlightTasks int

	// DelayedTasks is the number of foreground tasks waiting in the delayed publisher,
	// including one being handed to the dispatcher. Background tasks are not counted.
	DelayedTasks int

	// PendingReruns is the number of dispatches waiting for an active task with the same
	// deduplication key to settle.
	PendingReruns int
}

// FailedTaskSummary holds the key details of a task that ended in the FAILED state. Used
// for user-facing error reporting.
type FailedTaskSummary struct {
	// TaskID is the unique identifier for the failed task.
	TaskID string

	// WorkflowID is the workflow the task belongs to.
	WorkflowID string

	// Executor is the name of the executor that ran the task.
	Executor string

	// LastError is the error message from the final attempt.
	LastError string

	// Attempt is the number of attempts made before the task was marked failed.
	Attempt int

	// IsFatal is true when the task failed due to a non-retryable error.
	IsFatal bool
}

// DefaultDispatcherConfig returns sensible defaults for production use.
//
// Returns DispatcherConfig which contains default values for handler counts, timeout, and
// retry settings.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		DefaultTimeout:          defaultTaskTimeout,
		DefaultMaxRetries:       defaultMaxRetries,
		RecoveryInterval:        defaultRecoveryInterval,
		StaleTaskThreshold:      defaultStaleTaskThreshold,
		HeartbeatInterval:       defaultHeartbeatInterval,
		NodeID:                  "",
		RecoveryBatchLimit:      defaultRecoveryBatchLimit,
		WatermillHighHandlers:   defaultWatermillHighHandlers,
		WatermillNormalHandlers: defaultWatermillNormalHandlers,
		WatermillLowHandlers:    defaultWatermillLowHandlers,
		Clock:                   nil,
		MaxRetryBackoff:         defaultMaxRetryBackoff,
		ExecutorAbandonGrace:    defaultExecutorAbandonGrace,
		MaxPendingReruns:        defaultMaxPendingReruns,
		OverflowLimit:           defaultOverflowLimit,
		SweepInterval:           defaultSweepInterval,
		Transport:               TransportAuto,
		SkipRecovery:            false,
	}
}
