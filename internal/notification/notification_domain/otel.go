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

package notification_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the notification_domain package.
	log = logger_domain.GetLogger("piko/internal/notification/notification_domain")

	// meter is the OpenTelemetry meter for the notification domain package.
	meter = otel.Meter("piko/internal/notification/notification_domain")

	// dispatcherStartCount tracks the number of times a dispatcher has been started.
	dispatcherStartCount metric.Int64Counter

	// dispatcherStopCount tracks how many times the dispatcher has been stopped.
	dispatcherStopCount metric.Int64Counter

	// notificationQueuedCount tracks the number of notifications added to the queue.
	notificationQueuedCount metric.Int64Counter

	// flushCount tracks the number of flush operations triggered.
	flushCount metric.Int64Counter

	// batchSentCount counts the number of batches sent.
	batchSentCount metric.Int64Counter

	// batchSizeMetric records the distribution of batch sizes.
	batchSizeMetric metric.Int64Histogram

	// notificationSentCount tracks the number of notifications sent successfully.
	notificationSentCount metric.Int64Counter

	// notificationSendDuration records the time taken to send notifications.
	notificationSendDuration metric.Float64Histogram

	// notificationSendErrorCount tracks notification sending failures.
	notificationSendErrorCount metric.Int64Counter

	// retryScheduledCount tracks the number of retries scheduled.
	retryScheduledCount metric.Int64Counter

	// retryAttemptCount tracks the number of retry attempts made.
	retryAttemptCount metric.Int64Counter

	// deadLetterCount counts messages sent to the dead letter queue.
	deadLetterCount metric.Int64Counter

	// circuitBreakerStateChangeCount tracks circuit breaker state changes.
	circuitBreakerStateChangeCount metric.Int64Counter

	// builderSendCount tracks the number of send operations at the builder level.
	builderSendCount metric.Int64Counter

	// builderSendDuration records the time taken for builder send operations.
	builderSendDuration metric.Float64Histogram

	// builderSendErrorCount tracks builder send errors.
	builderSendErrorCount metric.Int64Counter

	// multiCastCount tracks the number of multi-cast notifications sent.
	multiCastCount metric.Int64Counter

	// partialFailureCount tracks notifications with partial failures where some
	// providers succeeded and some failed.
	partialFailureCount metric.Int64Counter
)

func init() {
	var err error

	dispatcherStartCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.start",
		metric.WithDescription("Number of times the notification dispatcher has been started."),
		metric.WithUnit("{start}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dispatcherStopCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.stop",
		metric.WithDescription("Number of times the notification dispatcher has been stopped."),
		metric.WithUnit("{stop}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	notificationQueuedCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.queued",
		metric.WithDescription("Total number of notifications queued for dispatch."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	flushCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.flush",
		metric.WithDescription("Number of flush operations triggered."),
		metric.WithUnit("{flush}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchSentCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.batch.sent",
		metric.WithDescription("Total number of notification batches sent."),
		metric.WithUnit("{batch}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchSizeMetric, err = meter.Int64Histogram(
		"piko.notification.dispatcher.batch.size",
		metric.WithDescription("Size of notification batches sent."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	notificationSentCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.sent",
		metric.WithDescription("Total number of notifications sent successfully."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	notificationSendDuration, err = meter.Float64Histogram(
		"piko.notification.dispatcher.send.duration",
		metric.WithDescription("Duration of notification send operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	notificationSendErrorCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.send.errors",
		metric.WithDescription("Total number of notification send errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryScheduledCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.retry.scheduled",
		metric.WithDescription("Number of notifications scheduled for retry."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryAttemptCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.retry.attempts",
		metric.WithDescription("Number of retry attempts performed."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	deadLetterCount, err = meter.Int64Counter(
		"piko.notification.dispatcher.deadletter.sent",
		metric.WithDescription("Number of notifications sent to the dead letter queue."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	circuitBreakerStateChangeCount, err = meter.Int64Counter(
		"piko.notification.circuitbreaker.state.change",
		metric.WithDescription("Number of circuit breaker state changes."),
		metric.WithUnit("{change}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendCount, err = meter.Int64Counter(
		"piko.notification.builder.send.count",
		metric.WithDescription("Number of NotificationBuilder.Send invocations."),
		metric.WithUnit("{send}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendDuration, err = meter.Float64Histogram(
		"piko.notification.builder.send.duration",
		metric.WithDescription("Duration of NotificationBuilder.Send operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderSendErrorCount, err = meter.Int64Counter(
		"piko.notification.builder.send.errors",
		metric.WithDescription("Number of NotificationBuilder.Send errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	multiCastCount, err = meter.Int64Counter(
		"piko.notification.multicast.count",
		metric.WithDescription("Number of multi-cast notifications sent (sent to multiple providers)."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	partialFailureCount, err = meter.Int64Counter(
		"piko.notification.partial.failure.count",
		metric.WithDescription("Number of notifications with partial failures (some providers succeeded, some failed)."),
		metric.WithUnit("{notification}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
