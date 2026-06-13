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

package worker_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// attrKind is the metric and log attribute key for a job's kind.
	attrKind = "kind"

	// attrQueue is the metric and log attribute key for a job's queue.
	attrQueue = "queue"

	// attrOutcome is the metric attribute key describing why a job reached a terminal state.
	attrOutcome = "outcome"

	// logJobID is the log field key for a job's row id.
	logJobID = "job_id"
)

const (
	// outcomeFatal is a Fatal-wrapped error: terminal after one attempt, never retried.
	outcomeFatal = "fatal"

	// outcomeTimeout is an attempt budget exceeded on the final attempt (deadline-exceeded).
	outcomeTimeout = "timeout"

	// outcomeExhausted is a plain (non-fatal, non-timeout) error that used up every attempt.
	outcomeExhausted = "exhausted"

	// outcomePanic is a recovered panic in the handler: terminal after one attempt, exactly
	// like a Fatal error, never retried.
	outcomePanic = "panic"
)

var (
	// log is the package-level logger for the worker_domain package.
	log = logger_domain.GetLogger("piko/internal/worker/worker_domain")

	// meter is the package-level meter for OpenTelemetry metrics.
	meter = otel.Meter("piko/internal/worker/worker_domain")

	// jobsEnqueued counts the total number of worker jobs enqueued.
	jobsEnqueued metric.Int64Counter

	// jobsClaimed counts the total number of worker jobs claimed for execution.
	jobsClaimed metric.Int64Counter

	// jobsCompleted counts the total number of worker jobs that completed successfully.
	jobsCompleted metric.Int64Counter

	// jobsFailed counts the total number of worker jobs that failed.
	jobsFailed metric.Int64Counter

	// jobsRetried counts the total number of worker job retries scheduled.
	jobsRetried metric.Int64Counter

	// jobsRecovered counts the total number of worker jobs reclaimed after being orphaned.
	jobsRecovered metric.Int64Counter

	// jobRecoveryErrors counts the total number of recovery-sweep failures.
	jobRecoveryErrors metric.Int64Counter

	// jobWorkDuration records the wall-clock duration of worker job executions.
	jobWorkDuration metric.Float64Histogram

	// jobsQueueDepth observes the claimable job depth per queue.
	jobsQueueDepth metric.Int64ObservableGauge

	// jobsPending observes the total non-terminal jobs across all queues.
	jobsPending metric.Int64ObservableGauge
)

func init() {
	var err error

	jobsEnqueued, err = meter.Int64Counter(
		"worker.job.enqueued",
		metric.WithDescription("Total number of worker jobs enqueued"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsClaimed, err = meter.Int64Counter(
		"worker.job.claimed",
		metric.WithDescription("Total number of worker jobs claimed"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsCompleted, err = meter.Int64Counter(
		"worker.job.completed",
		metric.WithDescription("Total number of worker jobs completed successfully"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsFailed, err = meter.Int64Counter(
		"worker.job.failed",
		metric.WithDescription("Total number of worker jobs that reached terminal failed state"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsRetried, err = meter.Int64Counter(
		"worker.job.retried",
		metric.WithDescription("Total number of worker jobs re-scheduled for retry"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsRecovered, err = meter.Int64Counter(
		"worker.job.recovered",
		metric.WithDescription("Total number of worker jobs recovered"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobRecoveryErrors, err = meter.Int64Counter(
		"worker.job.recovery.errors",
		metric.WithDescription("Total number of worker recovery-sweep failures"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobWorkDuration, err = meter.Float64Histogram(
		"worker.work.duration",
		metric.WithDescription("Duration of job work per attempt"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsQueueDepth, err = meter.Int64ObservableGauge(
		"worker.queue.depth",
		metric.WithDescription("Claimable jobs per queue"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	jobsPending, err = meter.Int64ObservableGauge(
		"worker.pending",
		metric.WithDescription("Total non-terminal jobs across all queues"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
