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

package provider_mock

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// meter is the OpenTelemetry meter for the mock provider.
	meter = otel.Meter("piko/internal/llm/llm_adapters/provider_mock")

	// completeCount tracks the number of Complete calls.
	completeCount metric.Int64Counter

	// streamCount tracks the number of Stream calls.
	streamCount metric.Int64Counter
)

func init() {
	var err error

	completeCount, err = meter.Int64Counter(
		"piko.llm.provider.mock.complete.count",
		metric.WithDescription("Number of mock Complete calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamCount, err = meter.Int64Counter(
		"piko.llm.provider.mock.stream.count",
		metric.WithDescription("Number of mock Stream calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
