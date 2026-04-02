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

package llm_provider_voyage

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/llm/llm_provider_voyage")

	// meter is the OpenTelemetry meter for the Voyage provider.
	meter = otel.Meter("piko/llm/llm_provider_voyage")

	// embedCount tracks the number of Embed calls.
	embedCount metric.Int64Counter

	// embedDuration records the duration of Embed calls.
	embedDuration metric.Float64Histogram

	// embedErrorCount tracks Embed errors.
	embedErrorCount metric.Int64Counter
)

func init() {
	var err error

	embedCount, err = meter.Int64Counter(
		"piko.llm.provider.voyage.embed.count",
		metric.WithDescription("Number of Voyage Embed calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embedDuration, err = meter.Float64Histogram(
		"piko.llm.provider.voyage.embed.duration",
		metric.WithDescription("Duration of Voyage Embed calls."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embedErrorCount, err = meter.Int64Counter(
		"piko.llm.provider.voyage.embed.errors",
		metric.WithDescription("Number of Voyage Embed errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
