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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_domain")

	// meter is the OpenTelemetry meter for the orchestrator domain.
	meter = otel.Meter("piko/internal/orchestrator/orchestrator_domain")

	// TaskDispatchDuration measures the time taken to dispatch a task.
	TaskDispatchDuration metric.Float64Histogram

	// TaskScheduleDuration measures the time taken to schedule a task.
	TaskScheduleDuration metric.Float64Histogram

	// TaskProcessingDuration records how long each task takes to process.
	TaskProcessingDuration metric.Float64Histogram

	// TaskExecutionDuration measures the time taken to execute a task.
	TaskExecutionDuration metric.Float64Histogram

	// TaskRetryCount counts how many times tasks have been retried.
	TaskRetryCount metric.Int64Counter

	// TaskFailureCount records the number of tasks that have failed.
	TaskFailureCount metric.Int64Counter

	// TaskSuccessCount tracks the number of tasks that complete successfully.
	TaskSuccessCount metric.Int64Counter

	// WorkflowCompletionDuration records how long each workflow takes to complete.
	WorkflowCompletionDuration metric.Float64Histogram

	// WorkflowSuccessCount counts the number of workflows that complete without
	// error.
	WorkflowSuccessCount metric.Int64Counter

	// WorkflowFailureCount counts the number of failed workflows.
	WorkflowFailureCount metric.Int64Counter

	// DispatcherFetchDuration measures the time taken to fetch tasks from the
	// store.
	DispatcherFetchDuration metric.Float64Histogram

	// TaskFetchErrorCount counts the number of task fetch errors.
	TaskFetchErrorCount metric.Int64Counter

	// SchedulerPromotionDuration measures the time taken to promote scheduled
	// tasks.
	SchedulerPromotionDuration metric.Float64Histogram

	// ActiveTasksCount tracks the number of currently active tasks.
	ActiveTasksCount metric.Int64UpDownCounter

	// ActiveWorkflowsCount tracks the number of workflows that are currently
	// running.
	ActiveWorkflowsCount metric.Int64UpDownCounter

	// ExecutorRegistrationCount tracks how many times executors have registered.
	ExecutorRegistrationCount metric.Int64Counter

	// ExecutorRegistrationErrorCount counts executor registration errors.
	ExecutorRegistrationErrorCount metric.Int64Counter

	// WorkflowStatusCheckDuration measures the time taken to check workflow
	// status.
	WorkflowStatusCheckDuration metric.Float64Histogram

	// WorkflowStatusCheckErrorCount counts the number of workflow status check
	// errors.
	WorkflowStatusCheckErrorCount metric.Int64Counter

	// TaskDispatchErrorCount counts the number of task dispatch errors.
	TaskDispatchErrorCount metric.Int64Counter

	// TaskDispatchedCount counts the number of tasks that have been dispatched.
	TaskDispatchedCount metric.Int64Counter

	// TaskPersistenceErrorCount counts how many times task persistence has failed.
	TaskPersistenceErrorCount metric.Int64Counter

	// TaskPersistedCount tracks the number of tasks saved to storage.
	TaskPersistedCount metric.Int64Counter

	// TaskPersistenceDuration measures how long it takes to save a task.
	TaskPersistenceDuration metric.Float64Histogram

	// DelayedTaskPublishErrorCount tracks the number of failures when publishing
	// delayed tasks.
	DelayedTaskPublishErrorCount metric.Int64Counter

	// DelayedTaskPublishedCount records the number of delayed tasks that have been
	// published.
	DelayedTaskPublishedCount metric.Int64Counter

	// DispatcherValidationErrorCount counts the number of task validation errors
	// in the dispatcher.
	DispatcherValidationErrorCount metric.Int64Counter

	// DispatcherTasksQueuedCount counts the number of tasks queued by the
	// dispatcher.
	DispatcherTasksQueuedCount metric.Int64Counter

	// DispatcherBackpressureCount counts the number of times backpressure was
	// applied.
	DispatcherBackpressureCount metric.Int64Counter

	// TaskRecoveryCount tracks the number of stale tasks recovered from PROCESSING
	// state.
	TaskRecoveryCount metric.Int64Counter

	// TaskRecoveryErrorCount tracks the number of errors during task recovery
	// operations.
	TaskRecoveryErrorCount metric.Int64Counter

	// TaskDeduplicationBlockedCount tracks tasks blocked by deduplication.
	TaskDeduplicationBlockedCount metric.Int64Counter

	// TaskGracefulReleaseCount tracks how many tasks were released during a
	// graceful shutdown.
	TaskGracefulReleaseCount metric.Int64Counter
)

func init() {
	var err error

	TaskDispatchDuration, err = meter.Float64Histogram(
		"orchestrator.domain.task_dispatch_duration",
		metric.WithDescription("Duration of task dispatch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskScheduleDuration, err = meter.Float64Histogram(
		"orchestrator.domain.task_schedule_duration",
		metric.WithDescription("Duration of task schedule operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskProcessingDuration, err = meter.Float64Histogram(
		"orchestrator.domain.task_processing_duration",
		metric.WithDescription("Duration of task processing operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskExecutionDuration, err = meter.Float64Histogram(
		"orchestrator.domain.task_execution_duration",
		metric.WithDescription("Duration of task execution operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskRetryCount, err = meter.Int64Counter(
		"orchestrator.domain.task_retry_count",
		metric.WithDescription("Number of task retries"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskFailureCount, err = meter.Int64Counter(
		"orchestrator.domain.task_failure_count",
		metric.WithDescription("Number of task failures"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskSuccessCount, err = meter.Int64Counter(
		"orchestrator.domain.task_success_count",
		metric.WithDescription("Number of task successes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WorkflowCompletionDuration, err = meter.Float64Histogram(
		"orchestrator.domain.workflow_completion_duration",
		metric.WithDescription("Duration of workflow completion operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WorkflowSuccessCount, err = meter.Int64Counter(
		"orchestrator.domain.workflow_success_count",
		metric.WithDescription("Number of successful workflows"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WorkflowFailureCount, err = meter.Int64Counter(
		"orchestrator.domain.workflow_failure_count",
		metric.WithDescription("Number of failed workflows"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DispatcherFetchDuration, err = meter.Float64Histogram(
		"orchestrator.domain.dispatcher_fetch_duration",
		metric.WithDescription("Duration of dispatcher fetch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SchedulerPromotionDuration, err = meter.Float64Histogram(
		"orchestrator.domain.scheduler_promotion_duration",
		metric.WithDescription("Duration of scheduler promotion operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ActiveTasksCount, err = meter.Int64UpDownCounter(
		"orchestrator.domain.active_tasks_count",
		metric.WithDescription("Number of active tasks"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ActiveWorkflowsCount, err = meter.Int64UpDownCounter(
		"orchestrator.domain.active_workflows_count",
		metric.WithDescription("Number of active workflows"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorRegistrationCount, err = meter.Int64Counter(
		"orchestrator.domain.executor_registration_count",
		metric.WithDescription("Number of executor registrations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorRegistrationErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.executor_registration_error_count",
		metric.WithDescription("Number of executor registration errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WorkflowStatusCheckDuration, err = meter.Float64Histogram(
		"orchestrator.domain.workflow_status_check_duration",
		metric.WithDescription("Duration of workflow status check operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WorkflowStatusCheckErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.workflow_status_check_error_count",
		metric.WithDescription("Number of workflow status check errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskFetchErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.task_fetch_error_count",
		metric.WithDescription("Number of task fetch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskDispatchErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.task_dispatch_error_count",
		metric.WithDescription("Number of task dispatch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskDispatchedCount, err = meter.Int64Counter(
		"orchestrator.domain.task_dispatched_count",
		metric.WithDescription("Number of tasks dispatched to priority topics"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskPersistenceErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.task_persistence_error_count",
		metric.WithDescription("Number of task persistence errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskPersistedCount, err = meter.Int64Counter(
		"orchestrator.domain.task_persisted_count",
		metric.WithDescription("Number of tasks persisted to database"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskPersistenceDuration, err = meter.Float64Histogram(
		"orchestrator.domain.task_persistence_duration",
		metric.WithDescription("Duration of task persistence operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DelayedTaskPublishErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.delayed_task_publish_error_count",
		metric.WithDescription("Number of delayed task publish errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DelayedTaskPublishedCount, err = meter.Int64Counter(
		"orchestrator.domain.delayed_task_published_count",
		metric.WithDescription("Number of delayed tasks published"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DispatcherValidationErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.dispatcher_validation_error_count",
		metric.WithDescription("Number of task validation errors in dispatcher"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DispatcherTasksQueuedCount, err = meter.Int64Counter(
		"orchestrator.domain.dispatcher_tasks_queued_count",
		metric.WithDescription("Number of tasks queued by dispatcher"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DispatcherBackpressureCount, err = meter.Int64Counter(
		"orchestrator.domain.dispatcher_backpressure_count",
		metric.WithDescription("Number of times dispatcher applied backpressure due to full queues"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskRecoveryCount, err = meter.Int64Counter(
		"orchestrator.domain.task_recovery_count",
		metric.WithDescription("Number of stale tasks recovered from PROCESSING state"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskRecoveryErrorCount, err = meter.Int64Counter(
		"orchestrator.domain.task_recovery_error_count",
		metric.WithDescription("Number of errors during task recovery operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskDeduplicationBlockedCount, err = meter.Int64Counter(
		"orchestrator.domain.task_deduplication_blocked_count",
		metric.WithDescription("Number of tasks blocked by deduplication"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TaskGracefulReleaseCount, err = meter.Int64Counter(
		"orchestrator.domain.task_graceful_release_count",
		metric.WithDescription("Number of tasks released during graceful shutdown"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
