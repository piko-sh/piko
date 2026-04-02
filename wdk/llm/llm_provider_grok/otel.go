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

package llm_provider_grok

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/llm/llm_provider_grok")

	// meter is the OpenTelemetry meter for the Grok provider.
	meter = otel.Meter("piko/llm/llm_provider_grok")

	// completeCount tracks the number of Complete calls.
	completeCount metric.Int64Counter

	// completeDuration records the duration of Complete calls.
	completeDuration metric.Float64Histogram

	// completeErrorCount tracks Complete errors.
	completeErrorCount metric.Int64Counter

	// streamCount tracks the number of Stream calls.
	streamCount metric.Int64Counter

	// streamErrorCount tracks Stream errors.
	streamErrorCount metric.Int64Counter
)

func init() {
	var err error

	completeCount, err = meter.Int64Counter(
		"piko.llm.provider.grok.complete.count",
		metric.WithDescription("Number of Grok Complete calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeDuration, err = meter.Float64Histogram(
		"piko.llm.provider.grok.complete.duration",
		metric.WithDescription("Duration of Grok Complete calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completeErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.grok.complete.errors",
		metric.WithDescription("Number of Grok Complete errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamCount, err = meter.Int64Counter(
		"piko.llm.provider.grok.stream.count",
		metric.WithDescription("Number of Grok Stream calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.grok.stream.errors",
		metric.WithDescription("Number of Grok Stream errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
