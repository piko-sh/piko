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

package tui_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	// log is the package-level logger for the tui_domain package.
	log = logger.GetLogger("piko/internal/tui/tui_domain")

	// Meter is the OpenTelemetry meter for TUI domain metrics.
	Meter = otel.Meter("piko/internal/tui/tui_domain")

	// ProviderRefreshDuration measures the latency of provider refresh operations.
	ProviderRefreshDuration metric.Float64Histogram

	// ProviderRefreshErrorCount counts how many times a provider refresh has failed.
	ProviderRefreshErrorCount metric.Int64Counter

	// ProviderHealthCheckDuration records the time taken for provider health checks.
	ProviderHealthCheckDuration metric.Float64Histogram
)

func init() {
	var err error

	ProviderRefreshDuration, err = Meter.Float64Histogram(
		"tui.domain.provider_refresh.duration",
		metric.WithDescription("Duration of provider refresh operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ProviderRefreshErrorCount, err = Meter.Int64Counter(
		"tui.domain.provider_refresh.error_count",
		metric.WithDescription("Total number of provider refresh errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ProviderHealthCheckDuration, err = Meter.Float64Histogram(
		"tui.domain.provider_health_check.duration",
		metric.WithDescription("Duration of provider health check operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
