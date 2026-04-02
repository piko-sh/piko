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

package pikotest_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// Meter is the OpenTelemetry meter for the pikotest domain package.
	Meter = otel.Meter("piko/internal/pikotest/pikotest_domain")

	// TestRenderDuration records how long it takes to render test output.
	TestRenderDuration metric.Float64Histogram

	// TestRenderCount is a counter metric for tracking test render operations.
	TestRenderCount metric.Int64Counter

	// TestActionDuration records the duration of test actions in seconds.
	TestActionDuration metric.Float64Histogram

	// TestActionCount is a counter metric for tracking test actions.
	TestActionCount metric.Int64Counter
)

func init() {
	var err error

	TestRenderDuration, err = Meter.Float64Histogram(
		"pikotest.render.duration",
		metric.WithDescription("Duration of component render operations in tests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TestRenderCount, err = Meter.Int64Counter(
		"pikotest.render.count",
		metric.WithDescription("Number of component renders executed in tests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TestActionDuration, err = Meter.Float64Histogram(
		"pikotest.action.duration",
		metric.WithDescription("Duration of server action invocations in tests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TestActionCount, err = Meter.Int64Counter(
		"pikotest.action.count",
		metric.WithDescription("Number of server action invocations in tests"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
