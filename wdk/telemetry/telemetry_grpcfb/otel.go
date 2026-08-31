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

package telemetry_grpcfb

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the telemetry_grpcfb package.
	log = logger_domain.GetLogger("piko/wdk/telemetry/telemetry_grpcfb")

	// meter provides OpenTelemetry metrics for the telemetry_grpcfb package.
	meter = otel.Meter("piko/wdk/telemetry/telemetry_grpcfb")

	// stringsTruncatedCount tracks values shortened to fit their cap.
	stringsTruncatedCount metric.Int64Counter
)

func init() {
	var err error

	stringsTruncatedCount, err = meter.Int64Counter(
		"telemetry.strings_truncated_count",
		metric.WithDescription("Number of telemetry values shortened to fit their cap"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
