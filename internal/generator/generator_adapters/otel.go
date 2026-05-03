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

package generator_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is a package-level logger instance specific to the generator adapters.
	log = logger_domain.GetLogger("piko/internal/generator/generator_adapters")

	// meter is the OpenTelemetry Meter for the generator adapters, used to create metrics for
	// all adapters.
	meter = otel.Meter("piko/internal/generator/generator_adapters")

	// fileReadCount tracks the total number of file read operations.
	fileReadCount metric.Int64Counter

	// fileReadDuration measures the time taken for each file read operation, in
	// milliseconds.
	fileReadDuration metric.Float64Histogram

	// fileReadErrorCount counts errors during file read operations.
	fileReadErrorCount metric.Int64Counter

	// fileWriteCount tracks the total number of file write operations.
	fileWriteCount metric.Int64Counter

	// fileWriteDuration measures the time taken for each file write operation, in
	// milliseconds.
	fileWriteDuration metric.Float64Histogram

	// fileWriteErrorCount counts errors during file write operations.
	fileWriteErrorCount metric.Int64Counter
)

func init() {
	var err error

	fileReadCount, err = meter.Int64Counter(
		"piko.generator.adapter.fs.read.count",
		metric.WithDescription("Total number of file read operations initiated by the generator."),
		metric.WithUnit("{file}"),
	)
	if err != nil {
		otel.Handle(err)
	}
	fileReadDuration, err = meter.Float64Histogram(
		"piko.generator.adapter.fs.read.duration",
		metric.WithDescription("The duration of a single file read operation."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
	fileReadErrorCount, err = meter.Int64Counter(
		"piko.generator.adapter.fs.read.errors",
		metric.WithDescription("Total number of errors encountered during file read operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fileWriteCount, err = meter.Int64Counter(
		"piko.generator.adapter.fs.write.count",
		metric.WithDescription("Total number of file write operations initiated by the generator."),
		metric.WithUnit("{file}"),
	)
	if err != nil {
		otel.Handle(err)
	}
	fileWriteDuration, err = meter.Float64Histogram(
		"piko.generator.adapter.fs.write.duration",
		metric.WithDescription("The duration of a single file write operation."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
	fileWriteErrorCount, err = meter.Int64Counter(
		"piko.generator.adapter.fs.write.errors",
		metric.WithDescription("Total number of errors encountered during file write operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

}
