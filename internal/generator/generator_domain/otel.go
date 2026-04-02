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

package generator_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/generator/generator_domain")

	// meter is the OpenTelemetry meter for the generator domain.
	meter = otel.Meter("piko/internal/generator/generator_domain")

	// generateCount tracks the number of single-file generate operations.
	generateCount metric.Int64Counter

	// generateDuration measures the duration of single-file generate operations.
	generateDuration metric.Float64Histogram

	// generateErrorCount tracks the number of errors during single-file generate
	// operations.
	generateErrorCount metric.Int64Counter

	// generateProjectCount tracks the number of full project generation
	// operations.
	generateProjectCount metric.Int64Counter

	// generateProjectDuration measures the duration of full project generation
	// operations.
	generateProjectDuration metric.Float64Histogram

	// generateProjectErrorCount tracks the number of errors during full project
	// generation operations.
	generateProjectErrorCount metric.Int64Counter
)

func init() {
	var err error

	generateCount, err = meter.Int64Counter(
		"generator.service.generate.count",
		metric.WithDescription("Number of single-file generate operations."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateDuration, err = meter.Float64Histogram(
		"generator.service.generate.duration",
		metric.WithDescription("Duration of single-file generate operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateErrorCount, err = meter.Int64Counter(
		"generator.service.generate.errors",
		metric.WithDescription("Number of errors during single-file generate operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateProjectCount, err = meter.Int64Counter(
		"generator.service.generate_project.count",
		metric.WithDescription("Number of full project generation operations."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateProjectDuration, err = meter.Float64Histogram(
		"generator.service.generate_project.duration",
		metric.WithDescription("Duration of full project generation operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateProjectErrorCount, err = meter.Int64Counter(
		"generator.service.generate_project.errors",
		metric.WithDescription("Number of errors during full project generation operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
