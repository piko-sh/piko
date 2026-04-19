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

	// jitterMilliseconds is the maximum random jitter added to backoff delays.
	jitterMilliseconds = 1000

	// dispatchFailureRetryDelay is the delay before retrying a failed dispatch.
	// Used when the delayed task publisher fails to dispatch a due task.
	dispatchFailureRetryDelay = 5 * time.Second

	// defaultTaskTimeout is the default time limit for running a task.
	defaultTaskTimeout = 5 * time.Minute

	// defaultMaxRetries is the maximum number of retry attempts for failed tasks.
	defaultMaxRetries = 3

	// defaultRecoveryInterval is how often to check for stale PROCESSING tasks.
	defaultRecoveryInterval = 30 * time.Second

	// defaultStaleTaskThreshold is how long before a PROCESSING task is considered
	// stuck.
	defaultStaleTaskThreshold = 10 * time.Minute

	// defaultHeartbeatInterval sets how often task timestamps are updated.
	// Must be significantly less than StaleTaskThreshold to prevent premature
	// task recovery.
	defaultHeartbeatInterval = 30 * time.Second

	// defaultRecoveryLeaseTimeout is how long a recovery claim is valid. Should be
	// long enough to complete recovery but short enough to not block others.
	defaultRecoveryLeaseTimeout = 5 * time.Minute

	// defaultRecoveryBatchLimit is the maximum number of tasks to claim per
	// recovery cycle.
	defaultRecoveryBatchLimit = 100

	// staleTaskRecoveryError is the error message saved on tasks that are
	// recovered after a worker has crashed or timed out.
	staleTaskRecoveryError = "task recovered: worker crashed or timed out"

	// defaultWatermillHighHandlers is the default handler count for high-priority
	// tasks. The ratio 10:5:2 mirrors the default polling weights for priority
	// fairness.
	defaultWatermillHighHandlers = 10

	// defaultWatermillNormalHandlers is the default handler count for
	// normal-priority tasks.
	defaultWatermillNormalHandlers = 5

	// defaultWatermillLowHandlers is the default handler count for low-priority
	// tasks.
	defaultWatermillLowHandlers = 2

	// attributeKeyTaskID is the logging attribute key for task identifiers.
	attributeKeyTaskID = "taskID"

	// attributeKeyWorkflowID is the logging attribute key for workflow identifiers.
	attributeKeyWorkflowID = "workflowID"

	// attributeKeyAttempt is the attribute key for the task attempt number.
	attributeKeyAttempt = "attempt"
)

// TaskDispatcher distributes tasks to workers using competing-consumer
// semantics. Each task is processed by exactly one worker, regardless of how
// many are running.
//
// Architecture:
//   - Tasks are published to Watermill topics per priority level
//   - Handler pools subscribe to topics (competing consumers)
//   - Event bus used for coordination signals (task.completed,
//     workflow.registered)
//   - TaskStore used for persistence and crash recovery
//
// Create a TaskDispatcher using orchestrator_adapters.CreateTaskDispatcher().
type TaskDispatcher interface {
	// Dispatch queues a task for immediate processing by the worker pool.
	// Non-blocking unless the queue is full, which applies backpressure.
	//
	// Takes task (*Task) which specifies the work to be processed.
	//
	// Returns error when the context is cancelled or task validation fails.
	Dispatch(ctx context.Context, task *Task) error

	// DispatchDelayed schedules a task to run at a later time.
	// Uses a min-heap with a sleep-until-due pattern (no polling).
	//
	// Takes task (*Task) which is the task to schedule.
	// Takes executeAt (time.Time) which is when the task should run.
	//
	// Returns error when the task cannot be scheduled.
	DispatchDelayed(ctx context.Context, task *Task, executeAt time.Time) error

	// RegisterExecutor adds a task executor with the given name.
	// Must be called before Start() for all executor types.
	//
	// Takes ctx (context.Context) which carries logging context.
	// Takes name (string) which identifies the executor.
	// Takes executor (TaskExecutor) which handles tasks of the named kind.
	RegisterExecutor(ctx context.Context, name string, executor TaskExecutor)

	// Start begins the worker pool and delayed task processor.
	// Blocks until the context is cancelled.
	//
	// Returns error when the worker pool fails to start or encounters a fatal
	// error.
	Start(ctx context.Context) error

	// Stats returns the current dispatcher statistics for monitoring.
	Stats() DispatcherStats

	// IsIdle returns true when the dispatcher has no work remaining:
	// - All dispatched tasks have completed or failed
	// - All queues are empty
	// - No delayed tasks are pending
	// - No workers are actively processing
	// Used for build completion detection.
	IsIdle() bool

	// FailedTasks returns a summary of all tasks currently in the FAILED state.
	// When a build tag is set, only tasks matching that tag are returned.
	//
	// Returns []FailedTaskSummary which contains the failed task details.
	// Returns error when the underlying store query fails.
	FailedTasks(ctx context.Context) ([]FailedTaskSummary, error)

	// SetBuildTag sets an optional tag that scopes newly dispatched tasks to a
	// particular build run. Pass an empty string to clear the tag.
	SetBuildTag(tag string)

	// BuildTag returns the current build tag, or empty if none is set.
	BuildTag() string
}

// DispatcherConfig configures the task dispatcher behaviour.
type DispatcherConfig struct {
	// Clock provides time operations, defaulting to the real system clock
	// when nil and accepting a mock clock for deterministic testing of
	// retry backoff and delayed task scheduling.
	Clock clockpkg.Clock

	// NodeID uniquely identifies this orchestrator instance for recovery lease
	// ownership, preventing multiple nodes from recovering the same task. If
	// empty, a UUID is generated at startup.
	NodeID string

	// RecoveryLeaseTimeout is how long a recovery claim is valid, needing
	// to be long enough to complete recovery but short enough to not
	// block other nodes, defaulting to 5 minutes.
	RecoveryLeaseTimeout time.Duration

	// RecoveryInterval is the interval for checking stale PROCESSING tasks.
	// Set to 0 to disable recovery; default is 30 seconds.
	RecoveryInterval time.Duration

	// StaleTaskThreshold is how long a task can be in PROCESSING status
	// before being considered stuck and eligible for recovery.
	// Default is 10 minutes.
	StaleTaskThreshold time.Duration

	// HeartbeatInterval is how often to update task timestamps during
	// execution, preventing long-running tasks from being recovered while
	// still active and needing to be significantly less than
	// StaleTaskThreshold, where 0 disables heartbeats and the default
	// is 30 seconds.
	HeartbeatInterval time.Duration

	// DefaultMaxRetries is the retry count for tasks that do not specify one.
	DefaultMaxRetries int

	// RecoveryBatchLimit is the maximum number of tasks to claim per recovery
	// cycle. Default is 100.
	RecoveryBatchLimit int

	// WatermillHighHandlers is the number of concurrent handlers for
	// high-priority tasks, where more handlers increase throughput for that
	// priority level, defaulting to 10.
	WatermillHighHandlers int

	// WatermillNormalHandlers is the number of concurrent handlers for
	// normal-priority tasks. Default is 5.
	WatermillNormalHandlers int

	// WatermillLowHandlers is the number of concurrent handlers for low-priority
	// tasks. Default is 2.
	WatermillLowHandlers int

	// DefaultTimeout is the timeout used for tasks that do not set their own.
	DefaultTimeout time.Duration

	// SyncPersistence controls whether task updates persist synchronously.
	// Default is false (async persistence); set to true for testing.
	SyncPersistence bool
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

	// TasksFatalFailed is the subset of TasksFailed caused by fatal
	// (non-retryable) errors.
	TasksFatalFailed int64
}

// FailedTaskSummary holds the key details of a task that ended in the FAILED
// state. Used for user-facing error reporting.
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
// Returns DispatcherConfig which contains default values for handler counts,
// timeout, and retry settings.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		DefaultTimeout:          defaultTaskTimeout,
		DefaultMaxRetries:       defaultMaxRetries,
		RecoveryInterval:        defaultRecoveryInterval,
		StaleTaskThreshold:      defaultStaleTaskThreshold,
		HeartbeatInterval:       defaultHeartbeatInterval,
		NodeID:                  "",
		RecoveryLeaseTimeout:    defaultRecoveryLeaseTimeout,
		RecoveryBatchLimit:      defaultRecoveryBatchLimit,
		SyncPersistence:         false,
		WatermillHighHandlers:   defaultWatermillHighHandlers,
		WatermillNormalHandlers: defaultWatermillNormalHandlers,
		WatermillLowHandlers:    defaultWatermillLowHandlers,
	}
}
