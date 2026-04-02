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

package shutdown

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/shutdown")

	// Meter is the OpenTelemetry meter for shutdown tracking.
	Meter = otel.Meter("piko/internal/shutdown")

	// ShutdownSignalCount is a counter metric that tracks how many shutdown
	// signals the application has received.
	ShutdownSignalCount metric.Int64Counter

	// CleanupFunctionCount is the counter metric that tracks cleanup function calls.
	CleanupFunctionCount metric.Int64Counter

	// CleanupFunctionExecutedCount tracks the number of cleanup functions that have
	// been executed.
	CleanupFunctionExecutedCount metric.Int64Counter

	// CleanupFunctionErrorCount is a counter that tracks the number of errors
	// that happen when cleanup functions run.
	CleanupFunctionErrorCount metric.Int64Counter

	// CleanupFunctionTimeoutCount is a counter that tracks how many times a
	// cleanup function has timed out.
	CleanupFunctionTimeoutCount metric.Int64Counter

	// CleanupFunctionPanicCount counts cleanup functions that have panicked.
	CleanupFunctionPanicCount metric.Int64Counter

	// CleanupDuration records how long it takes to clean up resources.
	CleanupDuration metric.Float64Histogram
)

func init() {
	var err error

	ShutdownSignalCount, err = Meter.Int64Counter(
		"shutdown.signal_count",
		metric.WithDescription("Number of shutdown signals received"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupFunctionCount, err = Meter.Int64Counter(
		"shutdown.cleanup_function_count",
		metric.WithDescription("Number of cleanup functions registered"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupFunctionExecutedCount, err = Meter.Int64Counter(
		"shutdown.cleanup_function_executed_count",
		metric.WithDescription("Number of cleanup functions executed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupFunctionErrorCount, err = Meter.Int64Counter(
		"shutdown.cleanup_function_error_count",
		metric.WithDescription("Number of cleanup function errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupFunctionTimeoutCount, err = Meter.Int64Counter(
		"shutdown.cleanup_function_timeout_count",
		metric.WithDescription("Number of cleanup functions that timed out"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupFunctionPanicCount, err = Meter.Int64Counter(
		"shutdown.cleanup_function_panic_count",
		metric.WithDescription("Number of cleanup functions that panicked"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CleanupDuration, err = Meter.Float64Histogram(
		"shutdown.cleanup_duration",
		metric.WithDescription("Duration of cleanup operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
