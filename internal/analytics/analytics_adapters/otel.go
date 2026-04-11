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

package analytics_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/analytics/analytics_adapters")

	meter = otel.Meter("piko/internal/analytics/analytics_adapters")

	// webhookSendCount is the total number of webhook batch POSTs.
	webhookSendCount metric.Int64Counter

	// webhookSendDuration is the time taken for each webhook POST.
	webhookSendDuration metric.Float64Histogram

	// webhookErrorCount is the number of failed webhook POSTs.
	webhookErrorCount metric.Int64Counter

	// webhookBatchSize tracks the number of events per batch.
	webhookBatchSize metric.Int64Histogram

	// ga4SendCount is the total number of GA4 Measurement Protocol batch POSTs.
	ga4SendCount metric.Int64Counter

	// ga4SendDuration is the time taken for each GA4 POST.
	ga4SendDuration metric.Float64Histogram

	// ga4ErrorCount is the number of failed GA4 POSTs.
	ga4ErrorCount metric.Int64Counter

	// ga4BatchSize tracks the number of events per GA4 batch.
	ga4BatchSize metric.Int64Histogram
)

func init() {
	var err error

	webhookSendCount, err = meter.Int64Counter(
		"analytics.webhook.send_count",
		metric.WithDescription("Total webhook batch POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	webhookSendDuration, err = meter.Float64Histogram(
		"analytics.webhook.send_duration",
		metric.WithDescription("Duration of webhook POST requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	webhookErrorCount, err = meter.Int64Counter(
		"analytics.webhook.error_count",
		metric.WithDescription("Failed webhook POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	webhookBatchSize, err = meter.Int64Histogram(
		"analytics.webhook.batch_size",
		metric.WithDescription("Number of events per webhook batch"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ga4SendCount, err = meter.Int64Counter(
		"analytics.ga4.send_count",
		metric.WithDescription("Total GA4 Measurement Protocol batch POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ga4SendDuration, err = meter.Float64Histogram(
		"analytics.ga4.send_duration",
		metric.WithDescription("Duration of GA4 POST requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ga4ErrorCount, err = meter.Int64Counter(
		"analytics.ga4.error_count",
		metric.WithDescription("Failed GA4 POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ga4BatchSize, err = meter.Int64Histogram(
		"analytics.ga4.batch_size",
		metric.WithDescription("Number of events per GA4 batch"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
