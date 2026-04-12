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

package analytics_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/analytics/analytics_domain")

	meter = otel.Meter("piko/internal/analytics/analytics_domain")

	// eventsTrackedCount is the total number of events sent to Track().
	eventsTrackedCount metric.Int64Counter

	// eventsDroppedCount is the number of events dropped because a
	// collector's channel was full.
	eventsDroppedCount metric.Int64Counter

	// eventsCollectedCount is the number of events successfully delivered
	// to collectors.
	eventsCollectedCount metric.Int64Counter

	// eventsFailedCount is the number of events that a collector's Collect
	// method rejected with an error.
	eventsFailedCount metric.Int64Counter

	// batcherRetriesCount is the number of batch send retries across
	// all batchers.
	batcherRetriesCount metric.Int64Counter

	// batcherCircuitOpenCount is the number of times a batcher's
	// circuit breaker transitioned to the open state.
	batcherCircuitOpenCount metric.Int64Counter
)

func init() {
	var err error

	eventsTrackedCount, err = meter.Int64Counter(
		"analytics.events_tracked",
		metric.WithDescription("Total analytics events sent to Track()"),
	)
	if err != nil {
		otel.Handle(err)
	}

	eventsDroppedCount, err = meter.Int64Counter(
		"analytics.events_dropped",
		metric.WithDescription("Analytics events dropped due to full channel"),
	)
	if err != nil {
		otel.Handle(err)
	}

	eventsCollectedCount, err = meter.Int64Counter(
		"analytics.events_collected",
		metric.WithDescription("Analytics events successfully delivered to collectors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	eventsFailedCount, err = meter.Int64Counter(
		"analytics.events_failed",
		metric.WithDescription("Analytics events rejected by collectors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batcherRetriesCount, err = meter.Int64Counter(
		"analytics.batcher.retries",
		metric.WithDescription("Total batch send retry attempts"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batcherCircuitOpenCount, err = meter.Int64Counter(
		"analytics.batcher.circuit_open",
		metric.WithDescription("Circuit breaker transitions to open state"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
