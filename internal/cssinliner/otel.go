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

package cssinliner

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/cssinliner/cssinliner_domain")

	// Meter is the OpenTelemetry meter for the CSS inliner package.
	Meter = otel.Meter("piko/internal/cssinliner/cssinliner_domain")

	// ProcessCount tracks the number of CSS processing tasks that have run.
	ProcessCount metric.Int64Counter

	// ProcessDuration tracks how long CSS processing tasks take.
	ProcessDuration metric.Float64Histogram

	// ProcessErrorCount tracks the number of errors during CSS processing.
	ProcessErrorCount metric.Int64Counter
)

func init() {
	var err error

	ProcessCount, err = Meter.Int64Counter(
		"cssinliner.process.count",
		metric.WithDescription("Number of CSS processing tasks"),
	)
	if err != nil {
		log.Error("Failed to create ProcessCount metric", logger_domain.Error(err))
	}

	ProcessDuration, err = Meter.Float64Histogram(
		"cssinliner.process.duration",
		metric.WithDescription("Duration of CSS processing tasks in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		log.Error("Failed to create ProcessDuration metric", logger_domain.Error(err))
	}

	ProcessErrorCount, err = Meter.Int64Counter(
		"cssinliner.process.error.count",
		metric.WithDescription("Number of CSS processing errors"),
	)
	if err != nil {
		log.Error("Failed to create ProcessErrorCount metric", logger_domain.Error(err))
	}
}
