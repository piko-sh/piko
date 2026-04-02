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

package events_provider_gochannel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("piko/wdk/events/events_provider_gochannel")

	// providerStartDuration records the time taken for provider lifecycle startup.
	providerStartDuration metric.Float64Histogram

	// providerCloseDuration records the time taken to close providers.
	providerCloseDuration metric.Float64Histogram

	// providerStartCount counts the number of times providers have been started.
	providerStartCount metric.Int64Counter

	// providerCloseCount is a counter metric that tracks the number of provider
	// close operations.
	providerCloseCount metric.Int64Counter

	// providerStartErrorCount is a counter metric that tracks the number of
	// provider start failures.
	providerStartErrorCount metric.Int64Counter

	// providerCloseErrorCount tracks the number of errors when closing providers.
	providerCloseErrorCount metric.Int64Counter
)

func init() {
	var err error

	providerStartDuration, err = meter.Float64Histogram(
		"events.adapters.gochannel.provider_start_duration",
		metric.WithDescription("Duration of GoChannel provider start operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	providerCloseDuration, err = meter.Float64Histogram(
		"events.adapters.gochannel.provider_close_duration",
		metric.WithDescription("Duration of GoChannel provider close operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	providerStartCount, err = meter.Int64Counter(
		"events.adapters.gochannel.provider_start_count",
		metric.WithDescription("Number of GoChannel provider start operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	providerCloseCount, err = meter.Int64Counter(
		"events.adapters.gochannel.provider_close_count",
		metric.WithDescription("Number of GoChannel provider close operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	providerStartErrorCount, err = meter.Int64Counter(
		"events.adapters.gochannel.provider_start_error_count",
		metric.WithDescription("Number of GoChannel provider start errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	providerCloseErrorCount, err = meter.Int64Counter(
		"events.adapters.gochannel.provider_close_error_count",
		metric.WithDescription("Number of GoChannel provider close errors"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
