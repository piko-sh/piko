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

package healthprobe_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// meter is the OpenTelemetry meter for health probe instrumentation.
	meter = otel.Meter("piko/internal/healthprobe/healthprobe_domain")

	// HealthCheckErrorCount is a metric counter that tracks health check failures.
	HealthCheckErrorCount metric.Int64Counter

	// DrainSignalledCount tracks the number of times the drain signal was sent
	// during shutdown.
	DrainSignalledCount metric.Int64Counter
)

func init() {
	var err error

	HealthCheckErrorCount, err = meter.Int64Counter(
		"healthprobe.check.errors",
		metric.WithDescription("Total number of failed health checks"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DrainSignalledCount, err = meter.Int64Counter(
		"healthprobe.drain.signalled",
		metric.WithDescription("Number of times the drain signal was sent during shutdown"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
