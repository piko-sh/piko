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

package formatter_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the formatter_domain package.
	log = logger_domain.GetLogger("piko/internal/formatter/formatter_domain")

	// meter provides OpenTelemetry metrics for the formatter domain package.
	meter = otel.Meter("piko/internal/formatter/formatter_domain")

	// formatDuration records the time taken to format documentation output.
	formatDuration metric.Float64Histogram

	// formatCount tracks how many format operations have run.
	formatCount metric.Int64Counter

	// formatErrorCount is the metric counter for format error occurrences.
	formatErrorCount metric.Int64Counter

	// formatBytesIn counts the bytes read during formatting operations.
	formatBytesIn metric.Int64Counter

	// formatBytesOut counts the total bytes written during format operations.
	formatBytesOut metric.Int64Counter
)

func init() {
	var err error

	formatDuration, err = meter.Float64Histogram(
		"formatter.format.duration",
		metric.WithDescription("Duration of formatting operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	formatCount, err = meter.Int64Counter(
		"formatter.format.count",
		metric.WithDescription("Total number of format operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	formatErrorCount, err = meter.Int64Counter(
		"formatter.format.error_count",
		metric.WithDescription("Total number of format errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	formatBytesIn, err = meter.Int64Counter(
		"formatter.format.bytes_in",
		metric.WithDescription("Total bytes processed (input)"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	formatBytesOut, err = meter.Int64Counter(
		"formatter.format.bytes_out",
		metric.WithDescription("Total bytes produced (output)"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
