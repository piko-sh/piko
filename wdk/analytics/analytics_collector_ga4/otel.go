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

package analytics_collector_ga4

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/wdk/analytics/analytics_collector_ga4")

	meter = otel.Meter("piko/wdk/analytics/analytics_collector_ga4")

	sendCount metric.Int64Counter

	sendDuration metric.Float64Histogram

	errorCount metric.Int64Counter

	batchSize metric.Int64Histogram
)

func init() {
	var err error

	sendCount, err = meter.Int64Counter(
		"analytics.ga4.send_count",
		metric.WithDescription("Total GA4 Measurement Protocol batch POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	sendDuration, err = meter.Float64Histogram(
		"analytics.ga4.send_duration",
		metric.WithDescription("Duration of GA4 POST requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	errorCount, err = meter.Int64Counter(
		"analytics.ga4.error_count",
		metric.WithDescription("Failed GA4 POST requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchSize, err = meter.Int64Histogram(
		"analytics.ga4.batch_size",
		metric.WithDescription("Number of events per GA4 batch"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
