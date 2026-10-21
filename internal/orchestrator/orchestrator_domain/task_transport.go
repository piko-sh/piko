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
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
)

// TransportMode selects how a dispatcher delivers tasks to workers.
type TransportMode uint8

const (
	// TransportAuto delivers tasks in-process when the event bus is node-local, and over the
	// bus otherwise.
	TransportAuto TransportMode = iota

	// TransportLocal delivers references through in-memory worker rings without blocking
	// publication.
	TransportLocal

	// TransportBus delivers tasks as messages on the event bus, so every node subscribed to
	// the task topics competes for them.
	TransportBus
)

// String returns the mode's name.
//
// Returns string which is "auto", "local" or "bus".
func (m TransportMode) String() string {
	switch m {
	case TransportLocal:
		return "local"
	case TransportBus:
		return "bus"
	default:
		return "auto"
	}
}

// RecurringTask describes a task whose successor is scheduled after each run and restored
// if lost.
type RecurringTask struct {
	// Payload is each seeded run's payload.
	Payload map[string]any

	// Key is the deduplication key every run of the chain holds, which identifies it.
	Key string

	// Executor runs the task.
	Executor string

	// Delay is how long after seeding the seeded run is due.
	Delay time.Duration

	// Priority is the runs' priority.
	Priority TaskPriority

	// Background marks runs no build waits for.
	Background bool
}

// RecurringRegistry is implemented by a dispatcher that keeps recurring task chains
// alive.
type RecurringRegistry interface {
	// RegisterRecurring seeds a recurring task's chain unless a run is already scheduled,
	// and seeds it again whenever the dispatcher finds it with none.
	//
	// Takes task (RecurringTask) which describes the chain.
	//
	// Returns error when the first seeding fails for a reason other than a run already being
	// scheduled.
	RegisterRecurring(ctx context.Context, task RecurringTask) error
}

// RequirementFactory rebuilds the requirement check of a task from the task itself, for a
// task dispatched without its original check, such as a rerun the store created when the
// task holding its key settled.
type RequirementFactory func(task *Task) DispatchRequirement

// RequirementRegistry is implemented by a dispatcher that runs requirement checks on
// tasks it did not receive one for.
type RequirementRegistry interface {
	// RegisterRequirementFactory makes the dispatcher check, before running a task for
	// executor that came without a check, the requirement factory builds for it.
	//
	// Takes executor (string) which names the tasks' executor.
	// Takes factory (RequirementFactory) which builds their checks.
	RegisterRequirementFactory(executor string, factory RequirementFactory)
}

// LocalCompletions is implemented by a dispatcher that can deliver task completions
// straight to a function in this process instead of publishing task.completed events.
type LocalCompletions interface {
	// DeliverCompletionsTo routes every task completion to sink.
	//
	// Takes sink (CompletionSink) which receives the completions.
	//
	// Returns bool which is false when the dispatcher's tasks may run on other nodes, whose
	// completions only arrive as events; sink is then not used.
	DeliverCompletionsTo(sink CompletionSink) bool
}

// CompletionSink receives a task's outcome, with nil taskErr indicating success.
type CompletionSink func(ctx context.Context, task *Task, taskErr error)

// RunClaimed executes a claimed task on a pool slot and settles it.
//
// A recurring executor may schedule a successor, which the settle creates. When the
// dispatcher is shutting down and the execution ends in an error, the task is released
// back to pending at its attempt instead of being failed, so the next start runs it.
//
// Takes task (*Task) which is the claimed task.
// Takes slot (*ExecutionSlot) which runs it.
func (c *TaskProcessingCore) RunClaimed(ctx context.Context, task *Task, slot *ExecutionSlot) {
	start := c.Clock.Now()
	executor, err := c.GetExecutor(task.Executor)
	if err != nil {
		c.HandleTaskFailure(ctx, task, NewFatalError(err), start)
		return
	}
	timeout := task.Config.Timeout
	if timeout <= 0 {
		timeout = c.Config.EffectiveDefaultTimeout()
	}
	execCtx := ctx
	if recurring, ok := executor.(interface{ SchedulesSuccessor() bool }); ok && recurring.SchedulesSuccessor() {
		task.continuation = &settledContinuation{}
		execCtx = context.WithValue(ctx, continuationKey{}, task.continuation)
	}

	result, abandoned, execErr := slot.Run(execCtx, task, executor, timeout)
	if abandoned {
		return
	}
	if result != nil {
		task.Result = result
	}
	TaskExecutionDuration.Record(ctx, float64(c.Clock.Now().Sub(start).Milliseconds()))

	if execErr != nil && ctx.Err() != nil {
		c.ReleaseClaimed(ctx, task)
		return
	}
	c.HandleExecutionResult(ctx, task, execErr, start)
}

// ReleaseClaimed returns a claimed task to pending at the same attempt and releases its
// dispatch count without settling it.
//
// Takes task (*Task) which is the claimed task.
func (c *TaskProcessingCore) ReleaseClaimed(ctx context.Context, task *Task) {
	if !beginSettle(task) {
		return
	}
	task.Status = StatusPending
	task.UpdatedAt = c.Clock.Now()
	task.OwnerInstance = ""
	c.TasksDispatched.Add(-1)
	if c.TaskStore == nil {
		return
	}
	if err := c.transitionTask(ctx, task, StatusProcessing); err != nil && !errors.Is(err, ErrObsoleteDelivery) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to release interrupted task",
			logger_domain.String(attributeKeyTaskID, task.ID),
			logger_domain.Error(err))
	}
}

// HandleAbandonedExecution settles the task of an execution the pool's watchdog
// abandoned, as a failure that may be retried.
//
// Takes task (*Task) which was running.
// Takes cause (error) which explains the abandonment.
// Takes startTime (time.Time) which is when the execution began.
func (c *TaskProcessingCore) HandleAbandonedExecution(ctx context.Context, task *Task, cause error, startTime time.Time) {
	c.HandleTaskFailure(context.WithoutCancel(ctx), task, cause, startTime)
}
