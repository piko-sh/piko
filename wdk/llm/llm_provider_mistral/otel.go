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

package llm_provider_mistral

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/llm/llm_provider_mistral")

	// meter is the OpenTelemetry meter for the Mistral provider.
	meter = otel.Meter("piko/llm/llm_provider_mistral")

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

	// streamErrorCount tracks the number of errors from streams.
	streamErrorCount metric.Int64Counter

	// embedCount tracks the number of Embed calls.
	embedCount metric.Int64Counter

	// embedDuration records the duration of Embed calls.
	embedDuration metric.Float64Histogram

	// embedErrorCount tracks Embed errors.
	embedErrorCount metric.Int64Counter
)

func init() {
	var err error

	completeCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.complete.count",
		metric.WithDescription("Number of Mistral Complete calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeDuration, err = meter.Float64Histogram(
		"piko.llm.provider.mistral.complete.duration",
		metric.WithDescription("Duration of Mistral Complete calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.complete.errors",
		metric.WithDescription("Number of Mistral Complete errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.stream.count",
		metric.WithDescription("Number of Mistral Stream calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamDuration, err = meter.Float64Histogram(
		"piko.llm.provider.mistral.stream.duration",
		metric.WithDescription("Duration of Mistral Stream calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.stream.errors",
		metric.WithDescription("Number of Mistral Stream errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embedCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.embed.count",
		metric.WithDescription("Number of Mistral Embed calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embedDuration, err = meter.Float64Histogram(
		"piko.llm.provider.mistral.embed.duration",
		metric.WithDescription("Duration of Mistral Embed calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embedErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.mistral.embed.errors",
		metric.WithDescription("Number of Mistral Embed errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
