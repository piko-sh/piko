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

package provider_grpc

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("tui.provider_grpc")

	// Meter is the package-level OpenTelemetry meter for gRPC provider metrics.
	Meter = otel.Meter("piko/internal/tui/tui_adapters/provider_grpc")

	// GRPCCallDuration records how long gRPC calls take, measured in milliseconds.
	GRPCCallDuration metric.Float64Histogram

	// GRPCCallErrorCount tracks the number of failed gRPC calls.
	GRPCCallErrorCount metric.Int64Counter

	// GRPCCallCount tracks the total number of gRPC calls made.
	GRPCCallCount metric.Int64Counter
)

// instrumentedCall wraps a gRPC call with timing and error metrics.
// It records the call duration and increments appropriate counters.
//
// Takes operation (func() error) which is the gRPC call to execute.
//
// Returns error from the wrapped function.
func instrumentedCall(ctx context.Context, operation func() error) error {
	startTime := time.Now()
	defer func() {
		duration := float64(time.Since(startTime).Milliseconds())
		GRPCCallDuration.Record(ctx, duration)
	}()

	GRPCCallCount.Add(ctx, 1)

	err := operation()
	if err != nil {
		GRPCCallErrorCount.Add(ctx, 1)
	}

	return err
}

// refreshProvider is a generic helper that fetches data via gRPC, converts it,
// and stores the result under a mutex. It eliminates structural duplication
// across provider Refresh methods.
//
// Takes ctx (context.Context) which carries tracing and cancellation values.
// Takes fetch (func(context.Context) (T, error)) which performs the gRPC call
// and converts the response.
// Takes store (func(T)) which persists the converted data under a lock.
// Takes resourceName (string) which identifies the resource for logging and
// error wrapping.
//
// Returns error when the gRPC call fails.
func refreshProvider[T any](ctx context.Context, fetch func(context.Context) (T, error), store func(T), resourceName string) error {
	ctx, l := logger_domain.From(ctx, log)

	return instrumentedCall(ctx, func() error {
		data, err := fetch(ctx)
		if err != nil {
			l.Debug("Failed to fetch "+resourceName, logger.Error(err))
			return fmt.Errorf("fetching %s: %w", resourceName, err)
		}

		store(data)
		return nil
	})
}

func init() {
	var err error

	GRPCCallDuration, err = Meter.Float64Histogram(
		"tui.provider_grpc.call.duration",
		metric.WithDescription("Duration of gRPC calls in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	GRPCCallErrorCount, err = Meter.Int64Counter(
		"tui.provider_grpc.call.errors",
		metric.WithDescription("Number of failed gRPC calls"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	GRPCCallCount, err = Meter.Int64Counter(
		"tui.provider_grpc.call.count",
		metric.WithDescription("Total number of gRPC calls"),
		metric.WithUnit("{calls}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
