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

package capabilities_functions

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the capabilities_functions package.
	log = logger_domain.GetLogger("piko/internal/capabilities/capabilities_functions")

	// meter is the OpenTelemetry meter for capabilities_functions metrics.
	meter = otel.Meter("piko/internal/capabilities/capabilities_functions")

	// compilationDuration tracks time spent compiling components.
	compilationDuration metric.Float64Histogram

	// compilationErrorCount tracks the number of compilation failures.
	compilationErrorCount metric.Int64Counter

	// compiledComponentSize tracks the size of compiled component output.
	compiledComponentSize metric.Int64Histogram

	// minificationDuration tracks time spent on minification operations.
	minificationDuration metric.Float64Histogram
)

func init() {
	var err error

	compilationDuration, err = meter.Float64Histogram(
		"capabilities.compilation_duration",
		metric.WithDescription("Duration of component compilations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	compilationErrorCount, err = meter.Int64Counter(
		"capabilities.compilation_error_count",
		metric.WithDescription("Number of compilation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	compiledComponentSize, err = meter.Int64Histogram(
		"capabilities.compiled_component_size",
		metric.WithDescription("Size of compiled components"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	minificationDuration, err = meter.Float64Histogram(
		"capabilities.minification_duration",
		metric.WithDescription("Duration of minification operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
