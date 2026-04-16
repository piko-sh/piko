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

package email_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the email_domain package.
	log = logger_domain.GetLogger("piko/internal/email/email_domain")

	// meter is the OpenTelemetry meter for the email domain package.
	meter = otel.Meter("piko/internal/email/email_domain")

	// dispatcherStartCount tracks the number of times a dispatcher has been
	// started during its lifecycle.
	dispatcherStartCount metric.Int64Counter

	// dispatcherStopCount is a metric that tracks how many times the dispatcher
	// has been stopped.
	dispatcherStopCount metric.Int64Counter

	// emailQueuedCount tracks the number of emails added to the queue.
	emailQueuedCount metric.Int64Counter

	// flushCount is a counter metric that tracks the number of flush operations.
	flushCount metric.Int64Counter

	// batchSentCount counts the number of batches sent.
	batchSentCount metric.Int64Counter

	// batchSizeMetric records the spread of batch sizes handled.
	batchSizeMetric metric.Int64Histogram

	// emailSentCount tracks the number of emails sent.
	emailSentCount metric.Int64Counter

	// emailSendDuration records the time taken to send emails.
	emailSendDuration metric.Float64Histogram

	// emailSendErrorCount is a counter metric that tracks email sending failures.
	emailSendErrorCount metric.Int64Counter

	// retryScheduledCount is a metric that tracks the number of retries scheduled
	// for dead-letter handling.
	retryScheduledCount metric.Int64Counter

	// retryAttemptCount is a counter metric that tracks the number of retry
	// attempts made during operations.
	retryAttemptCount metric.Int64Counter

	// deadLetterCount is the metric counter for messages sent to the dead letter
	// queue.
	deadLetterCount metric.Int64Counter

	// builderSendCount tracks the number of send operations at the builder level.
	builderSendCount metric.Int64Counter

	// builderSendDuration records the time taken to send data to a builder.
	builderSendDuration metric.Float64Histogram

	// builderSendErrorCount tracks the number of errors when sending diagnostics.
	builderSendErrorCount metric.Int64Counter
)

func init() {
	var err error

	dispatcherStartCount, err = meter.Int64Counter(
		"piko.email.dispatcher.start",
		metric.WithDescription("Number of times the email dispatcher has been started."),
		metric.WithUnit("{start}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dispatcherStopCount, err = meter.Int64Counter(
		"piko.email.dispatcher.stop",
		metric.WithDescription("Number of times the email dispatcher has been stopped."),
		metric.WithUnit("{stop}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	emailQueuedCount, err = meter.Int64Counter(
		"piko.email.dispatcher.queued",
		metric.WithDescription("Total number of emails queued for dispatch."),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	flushCount, err = meter.Int64Counter(
		"piko.email.dispatcher.flush",
		metric.WithDescription("Number of flush operations triggered."),
		metric.WithUnit("{flush}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchSentCount, err = meter.Int64Counter(
		"piko.email.dispatcher.batch.sent",
		metric.WithDescription("Total number of email batches sent."),
		metric.WithUnit("{batch}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchSizeMetric, err = meter.Int64Histogram(
		"piko.email.dispatcher.batch.size",
		metric.WithDescription("Size of email batches sent."),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	emailSentCount, err = meter.Int64Counter(
		"piko.email.dispatcher.sent",
		metric.WithDescription("Total number of emails sent successfully."),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	emailSendDuration, err = meter.Float64Histogram(
		"piko.email.dispatcher.send.duration",
		metric.WithDescription("Duration of email send operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	emailSendErrorCount, err = meter.Int64Counter(
		"piko.email.dispatcher.send.errors",
		metric.WithDescription("Total number of email send errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryScheduledCount, err = meter.Int64Counter(
		"piko.email.dispatcher.retry.scheduled",
		metric.WithDescription("Number of emails scheduled for retry."),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryAttemptCount, err = meter.Int64Counter(
		"piko.email.dispatcher.retry.attempts",
		metric.WithDescription("Number of retry attempts performed."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	deadLetterCount, err = meter.Int64Counter(
		"piko.email.dispatcher.deadletter.sent",
		metric.WithDescription("Number of emails sent to the dead letter queue."),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendCount, err = meter.Int64Counter(
		"piko.email.builder.send.count",
		metric.WithDescription("Number of EmailBuilder.Send invocations."),
		metric.WithUnit("{send}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendDuration, err = meter.Float64Histogram(
		"piko.email.builder.send.duration",
		metric.WithDescription("Duration of EmailBuilder.Send operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendErrorCount, err = meter.Int64Counter(
		"piko.email.builder.send.errors",
		metric.WithDescription("Number of EmailBuilder.Send errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
