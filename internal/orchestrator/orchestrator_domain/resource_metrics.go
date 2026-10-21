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
)

var (
	// ObsoleteDeliveryCount counts acknowledged duplicate or stale messages.
	ObsoleteDeliveryCount = resourceCounter("orchestrator.domain.obsolete_deliveries", "Duplicate or obsolete task messages acknowledged without execution")

	// PrematureDeliveryCount counts deferred attempts delivered early and re-armed.
	PrematureDeliveryCount = resourceCounter("orchestrator.domain.premature_deliveries", "Retrying or scheduled task messages delivered before their due time and re-armed")

	// StaleWriteCount counts executor updates rejected by attempt fencing.
	StaleWriteCount = resourceCounter("orchestrator.domain.stale_writes", "Executor writes rejected after the stored attempt changed")

	// ReceiptCleanupCount counts expired receipt records removed by maintenance.
	ReceiptCleanupCount = resourceCounter("orchestrator.domain.receipts_cleaned", "Resolved or timed out receipts reclaimed by bounded maintenance")

	// SweepRedispatchCount counts due tasks re-dispatched by the lost-message sweep.
	SweepRedispatchCount = resourceCounter("orchestrator.domain.sweep_redispatches", "Due pending or retrying tasks re-dispatched because their messages may have been lost")

	// SweepRedispatchErrorCount counts sweep re-dispatches that failed.
	SweepRedispatchErrorCount = resourceCounter("orchestrator.domain.sweep_redispatch_errors", "Lost-message sweep re-dispatches that failed and are left for the next pass")

	// TaskHeldForCapacityCount counts task messages held because the bus was full.
	TaskHeldForCapacityCount = resourceCounter("orchestrator.domain.tasks_held_for_capacity", "Task messages held by the dispatcher until the event bus frees capacity")

	// RecurringRestartCount counts recurring task chains found dead and seeded again.
	RecurringRestartCount = resourceCounter("orchestrator.domain.recurring_restarts", "Recurring task chains found with no scheduled run and seeded again")

	// TaskAbandonedCount counts dispatched tasks failed because their message can never be
	// published.
	TaskAbandonedCount = resourceCounter("orchestrator.domain.tasks_abandoned", "Dispatched tasks failed because their message can never be published")
)

// resourceCounter creates an orchestration counter and reports setup errors to
// OpenTelemetry.
//
// Takes name (string) which identifies the metric.
// Takes description (string) which describes the counted events.
//
// Returns metric.Int64Counter which is the instrument returned by the meter.
func resourceCounter(name, description string) metric.Int64Counter {
	instrument, err := meter.Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		otel.Handle(err)
	}
	return instrument
}
