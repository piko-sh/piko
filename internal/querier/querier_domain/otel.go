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

package querier_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/querier/querier_domain")

	meter = otel.Meter("piko/internal/querier/querier_domain")

	catalogueBuildCount metric.Int64Counter

	catalogueBuildDuration metric.Float64Histogram

	queryAnalysisCount metric.Int64Counter

	queryAnalysisDuration metric.Float64Histogram

	generationCount metric.Int64Counter

	generationDuration metric.Float64Histogram

	generationErrorCount metric.Int64Counter
)

func init() {
	var err error

	catalogueBuildCount, err = meter.Int64Counter(
		"querier.service.catalogue_build.count",
		metric.WithDescription("Number of catalogue build operations."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	catalogueBuildDuration, err = meter.Float64Histogram(
		"querier.service.catalogue_build.duration",
		metric.WithDescription("Duration of catalogue build operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	queryAnalysisCount, err = meter.Int64Counter(
		"querier.service.query_analysis.count",
		metric.WithDescription("Number of query analysis operations."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	queryAnalysisDuration, err = meter.Float64Histogram(
		"querier.service.query_analysis.duration",
		metric.WithDescription("Duration of query analysis operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generationCount, err = meter.Int64Counter(
		"querier.service.generation.count",
		metric.WithDescription("Number of full generation pipeline operations."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generationDuration, err = meter.Float64Histogram(
		"querier.service.generation.duration",
		metric.WithDescription("Duration of full generation pipeline operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generationErrorCount, err = meter.Int64Counter(
		"querier.service.generation.errors",
		metric.WithDescription("Number of errors during generation operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
