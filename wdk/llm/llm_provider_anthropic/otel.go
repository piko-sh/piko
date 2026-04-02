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

package llm_provider_anthropic

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/llm/llm_provider_anthropic")

	// meter is the OpenTelemetry meter for the Anthropic provider.
	meter = otel.Meter("piko/llm/llm_provider_anthropic")

	// completeCount tracks the number of Complete calls.
	completeCount metric.Int64Counter

	// completeDuration records the duration of Complete calls.
	completeDuration metric.Float64Histogram

	// completeErrorCount tracks Complete errors.
	completeErrorCount metric.Int64Counter

	// streamCount tracks the number of Stream calls.
	streamCount metric.Int64Counter

	// streamDuration records the duration of Stream calls.
	streamDuration metric.Float64Histogram

	// streamErrorCount tracks Stream errors.
	streamErrorCount metric.Int64Counter
)

func init() {
	var err error

	completeCount, err = meter.Int64Counter(
		"piko.llm.provider.anthropic.complete.count",
		metric.WithDescription("Number of Anthropic Complete calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeDuration, err = meter.Float64Histogram(
		"piko.llm.provider.anthropic.complete.duration",
		metric.WithDescription("Duration of Anthropic Complete calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.anthropic.complete.errors",
		metric.WithDescription("Number of Anthropic Complete errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamCount, err = meter.Int64Counter(
		"piko.llm.provider.anthropic.stream.count",
		metric.WithDescription("Number of Anthropic Stream calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamDuration, err = meter.Float64Histogram(
		"piko.llm.provider.anthropic.stream.duration",
		metric.WithDescription("Duration of Anthropic Stream calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.anthropic.stream.errors",
		metric.WithDescription("Number of Anthropic Stream errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
