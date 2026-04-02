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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_adapters")

	// meter is the OpenTelemetry meter for the orchestrator adapters package.
	meter = otel.Meter("piko/internal/orchestrator/orchestrator_adapters")

	// WatermillEventBusPublishDuration records the time taken to publish events
	// through the Watermill EventBus.
	WatermillEventBusPublishDuration metric.Float64Histogram

	// WatermillEventBusSubscribeDuration records the time taken to subscribe to
	// a Watermill event bus topic.
	WatermillEventBusSubscribeDuration metric.Float64Histogram

	// WatermillEventBusCloseDuration records how long it takes to close the event
	// bus.
	WatermillEventBusCloseDuration metric.Float64Histogram

	// WatermillEventBusPublishErrorCount tracks the number of failed event
	// publish attempts through the Watermill event bus.
	WatermillEventBusPublishErrorCount metric.Int64Counter

	// WatermillEventBusSubscribeErrorCount tracks the number of subscription errors
	// from the Watermill event bus.
	WatermillEventBusSubscribeErrorCount metric.Int64Counter

	// WatermillEventBusCloseErrorCount tracks the number of errors that occur
	// when closing a Watermill event bus.
	WatermillEventBusCloseErrorCount metric.Int64Counter

	// WatermillEventBusMessageUnmarshalErrorCount tracks the number of message
	// unmarshalling failures in the Watermill event bus.
	WatermillEventBusMessageUnmarshalErrorCount metric.Int64Counter

	// WatermillEventBusPublishedEvents counts the total number of events
	// published through the Watermill event bus.
	WatermillEventBusPublishedEvents metric.Int64Counter

	// WatermillEventBusReceivedEvents counts the total events received by the
	// Watermill event bus.
	WatermillEventBusReceivedEvents metric.Int64Counter

	// WatermillEventBusDroppedEvents counts events dropped by the Watermill bus.
	WatermillEventBusDroppedEvents metric.Int64Counter

	// WatermillEventBusSubscriberCount tracks the number of active subscribers.
	WatermillEventBusSubscriberCount metric.Int64UpDownCounter

	// SQLiteTaskStoreCreateTaskDuration records the time taken to create a task in
	// the SQLite task store.
	SQLiteTaskStoreCreateTaskDuration metric.Float64Histogram

	// SQLiteTaskStoreUpdateTaskDuration records the duration of task update
	// operations in the SQLite task store.
	SQLiteTaskStoreUpdateTaskDuration metric.Float64Histogram

	// SQLiteTaskStoreFetchDueTasksDuration records the time taken to fetch due
	// tasks from the SQLite task store.
	SQLiteTaskStoreFetchDueTasksDuration metric.Float64Histogram

	// SQLiteTaskStoreGetWorkflowStatusDuration records the duration in seconds
	// of GetWorkflowStatus operations on the SQLite task store.
	SQLiteTaskStoreGetWorkflowStatusDuration metric.Float64Histogram

	// SQLiteTaskStorePromoteTasksDuration records the duration of task promotion
	// operations in the SQLite task store.
	SQLiteTaskStorePromoteTasksDuration metric.Float64Histogram

	// SQLiteTaskStoreCreateTaskErrorCount tracks the number of errors when
	// creating tasks in the SQLite task store.
	SQLiteTaskStoreCreateTaskErrorCount metric.Int64Counter

	// SQLiteTaskStoreUpdateTaskErrorCount counts errors during task updates in the
	// SQLite task store.
	SQLiteTaskStoreUpdateTaskErrorCount metric.Int64Counter

	// SQLiteTaskStoreFetchDueTasksErrorCount tracks the number of errors that
	// occur when fetching due tasks from the SQLite task store.
	SQLiteTaskStoreFetchDueTasksErrorCount metric.Int64Counter

	// SQLiteTaskStoreGetWorkflowErrorCount tracks the number of errors when
	// retrieving workflow data from the SQLite task store.
	SQLiteTaskStoreGetWorkflowErrorCount metric.Int64Counter

	// SQLiteTaskStorePromoteTasksErrorCount counts errors when promoting tasks.
	SQLiteTaskStorePromoteTasksErrorCount metric.Int64Counter

	// SQLiteTaskStoreTasksCreatedCount tracks the number of tasks created in the
	// SQLite task store.
	SQLiteTaskStoreTasksCreatedCount metric.Int64Counter

	// SQLiteTaskStoreTasksUpdatedCount counts the number of tasks updated in the
	// SQLite task store.
	SQLiteTaskStoreTasksUpdatedCount metric.Int64Counter

	// SQLiteTaskStoreTasksFetchedCount records the number of tasks fetched from
	// the SQLite task store.
	SQLiteTaskStoreTasksFetchedCount metric.Int64Counter

	// SQLiteTaskStoreTasksPromotedCount tracks the number of tasks promoted in the
	// SQLite task store.
	SQLiteTaskStoreTasksPromotedCount metric.Int64Counter

	// BridgeEventHandlingDuration records the time taken to handle events in the
	// ArtefactWorkflowBridge.
	BridgeEventHandlingDuration metric.Float64Histogram

	// BridgeEventHandlingErrorCount tracks the number of errors that occur when
	// handling bridge events.
	BridgeEventHandlingErrorCount metric.Int64Counter

	// BridgeArtefactFetchDuration records the time taken to fetch artefacts from
	// the bridge service.
	BridgeArtefactFetchDuration metric.Float64Histogram

	// BridgeArtefactFetchErrorCount tracks the number of failed artefact fetch
	// operations from the bridge service.
	BridgeArtefactFetchErrorCount metric.Int64Counter

	// BridgeTaskDispatchDuration records the time taken to dispatch bridge tasks.
	BridgeTaskDispatchDuration metric.Float64Histogram

	// BridgeTaskDispatchErrorCount tracks the number of errors that occur when
	// sending bridge tasks.
	BridgeTaskDispatchErrorCount metric.Int64Counter

	// BridgeEventsProcessedCount tracks the total number of bridge events that
	// have been processed.
	BridgeEventsProcessedCount metric.Int64Counter

	// BridgeTasksDispatchedCount tracks the number of tasks dispatched by the bridge.
	BridgeTasksDispatchedCount metric.Int64Counter

	// ExecutorCompilationDuration records the time spent compiling expressions.
	ExecutorCompilationDuration metric.Float64Histogram

	// ExecutorCompilationErrorCount is a metric that counts compilation errors
	// in the executor.
	ExecutorCompilationErrorCount metric.Int64Counter

	// ExecutorArtefactFetchDuration records the time taken to fetch artefacts.
	ExecutorArtefactFetchDuration metric.Float64Histogram

	// ExecutorArtefactFetchErrorCount counts errors when fetching executor artefacts.
	ExecutorArtefactFetchErrorCount metric.Int64Counter

	// ExecutorCapabilityExecutionDuration records the time taken to execute
	// capabilities in seconds.
	ExecutorCapabilityExecutionDuration metric.Float64Histogram

	// ExecutorCapabilityExecutionErrorCount counts errors during capability execution.
	ExecutorCapabilityExecutionErrorCount metric.Int64Counter

	// ExecutorVariantCreationDuration records the time taken to create executor
	// variants.
	ExecutorVariantCreationDuration metric.Float64Histogram

	// ExecutorVariantCreationErrorCount tracks the number of errors that occur
	// when creating executor variants.
	ExecutorVariantCreationErrorCount metric.Int64Counter

	// ExecutorPayloadParsingDuration records the time spent parsing payloads.
	ExecutorPayloadParsingDuration metric.Float64Histogram

	// ExecutorPayloadParsingErrorCount tracks how many times the executor fails to
	// parse a payload.
	ExecutorPayloadParsingErrorCount metric.Int64Counter
)

func init() {
	var err error

	WatermillEventBusPublishDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.watermill_event_bus_publish_duration",
		metric.WithDescription("Duration of Watermill publish operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusSubscribeDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.watermill_event_bus_subscribe_duration",
		metric.WithDescription("Duration of Watermill subscribe operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusCloseDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.watermill_event_bus_close_duration",
		metric.WithDescription("Duration of Watermill event bus close operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusPublishErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_publish_error_count",
		metric.WithDescription("Number of Watermill publish errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusSubscribeErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_subscribe_error_count",
		metric.WithDescription("Number of Watermill subscribe errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusCloseErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_close_error_count",
		metric.WithDescription("Number of Watermill event bus close errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusMessageUnmarshalErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_message_unmarshal_error_count",
		metric.WithDescription("Number of message unmarshal errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusPublishedEvents, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_published_events",
		metric.WithDescription("Number of events published via Watermill"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusReceivedEvents, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_received_events",
		metric.WithDescription("Number of events received via Watermill"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusDroppedEvents, err = meter.Int64Counter(
		"orchestrator.adapters.watermill_event_bus_dropped_events",
		metric.WithDescription("Number of events dropped via Watermill"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WatermillEventBusSubscriberCount, err = meter.Int64UpDownCounter(
		"orchestrator.adapters.watermill_event_bus_subscriber_count",
		metric.WithDescription("Number of active Watermill subscribers"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreCreateTaskDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.sqlite_task_store_create_task_duration",
		metric.WithDescription("Duration of task creation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreUpdateTaskDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.sqlite_task_store_update_task_duration",
		metric.WithDescription("Duration of task update operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreFetchDueTasksDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.sqlite_task_store_fetch_due_tasks_duration",
		metric.WithDescription("Duration of fetching due tasks operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreGetWorkflowStatusDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.sqlite_task_store_get_workflow_status_duration",
		metric.WithDescription("Duration of getting workflow status operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStorePromoteTasksDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.sqlite_task_store_promote_tasks_duration",
		metric.WithDescription("Duration of promoting scheduled tasks operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreCreateTaskErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_create_task_error_count",
		metric.WithDescription("Number of task creation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreUpdateTaskErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_update_task_error_count",
		metric.WithDescription("Number of task update errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreFetchDueTasksErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_fetch_due_tasks_error_count",
		metric.WithDescription("Number of fetching due tasks errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreGetWorkflowErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_get_workflow_error_count",
		metric.WithDescription("Number of getting workflow status errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStorePromoteTasksErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_promote_tasks_error_count",
		metric.WithDescription("Number of promoting scheduled tasks errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreTasksCreatedCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_tasks_created_count",
		metric.WithDescription("Number of tasks created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreTasksUpdatedCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_tasks_updated_count",
		metric.WithDescription("Number of tasks updated"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreTasksFetchedCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_tasks_fetched_count",
		metric.WithDescription("Number of tasks fetched"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SQLiteTaskStoreTasksPromotedCount, err = meter.Int64Counter(
		"orchestrator.adapters.sqlite_task_store_tasks_promoted_count",
		metric.WithDescription("Number of scheduled tasks promoted"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeEventHandlingDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.bridge_event_handling_duration",
		metric.WithDescription("Duration of event handling operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeEventHandlingErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.bridge_event_handling_error_count",
		metric.WithDescription("Number of event handling errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeArtefactFetchDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.bridge_artefact_fetch_duration",
		metric.WithDescription("Duration of artefact fetch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeArtefactFetchErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.bridge_artefact_fetch_error_count",
		metric.WithDescription("Number of artefact fetch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeTaskDispatchDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.bridge_task_dispatch_duration",
		metric.WithDescription("Duration of task dispatch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeTaskDispatchErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.bridge_task_dispatch_error_count",
		metric.WithDescription("Number of task dispatch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeEventsProcessedCount, err = meter.Int64Counter(
		"orchestrator.adapters.bridge_events_processed_count",
		metric.WithDescription("Number of events processed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BridgeTasksDispatchedCount, err = meter.Int64Counter(
		"orchestrator.adapters.bridge_tasks_dispatched_count",
		metric.WithDescription("Number of tasks dispatched"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorCompilationDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.executor_compilation_duration",
		metric.WithDescription("Duration of artefact compilation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorCompilationErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.executor_compilation_error_count",
		metric.WithDescription("Number of compilation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorArtefactFetchDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.executor_artefact_fetch_duration",
		metric.WithDescription("Duration of artefact fetch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorArtefactFetchErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.executor_artefact_fetch_error_count",
		metric.WithDescription("Number of artefact fetch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorCapabilityExecutionDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.executor_capability_execution_duration",
		metric.WithDescription("Duration of capability execution operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorCapabilityExecutionErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.executor_capability_execution_error_count",
		metric.WithDescription("Number of capability execution errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorVariantCreationDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.executor_variant_creation_duration",
		metric.WithDescription("Duration of variant creation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorVariantCreationErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.executor_variant_creation_error_count",
		metric.WithDescription("Number of variant creation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorPayloadParsingDuration, err = meter.Float64Histogram(
		"orchestrator.adapters.executor_payload_parsing_duration",
		metric.WithDescription("Duration of payload parsing operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ExecutorPayloadParsingErrorCount, err = meter.Int64Counter(
		"orchestrator.adapters.executor_payload_parsing_error_count",
		metric.WithDescription("Number of payload parsing errors"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
