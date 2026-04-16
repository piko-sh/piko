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

package compiler_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the compiler_adapters package.
	log = logger_domain.GetLogger("piko/internal/compiler/compiler_adapters")

	// meter is the OpenTelemetry meter for compiler adapter metrics.
	meter = otel.Meter("piko/internal/compiler/compiler_adapters")

	// fileReadCount tracks the number of files read from disk.
	fileReadCount metric.Int64Counter

	// fileReadErrorCount counts how many times a file read has failed.
	fileReadErrorCount metric.Int64Counter

	// fileReadDuration records the duration of file read operations.
	fileReadDuration metric.Float64Histogram

	// fileReadSize records how large each file is when read from disk.
	fileReadSize metric.Int64Histogram

	// memoryReadCount tracks the number of in-memory reads.
	memoryReadCount metric.Int64Counter

	// memoryReadErrorCount tracks the number of read errors from memory.
	memoryReadErrorCount metric.Int64Counter

	// memoryReadSize records the size of in-memory reads.
	memoryReadSize metric.Int64Histogram
)

func init() {
	var err error

	fileReadCount, err = meter.Int64Counter(
		"compiler.file_read_count",
		metric.WithDescription("Number of files read"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fileReadErrorCount, err = meter.Int64Counter(
		"compiler.file_read_error_count",
		metric.WithDescription("Number of file read errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fileReadDuration, err = meter.Float64Histogram(
		"compiler.file_read_duration",
		metric.WithDescription("Duration of file read operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fileReadSize, err = meter.Int64Histogram(
		"compiler.file_read_size",
		metric.WithDescription("Size of files read"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	memoryReadCount, err = meter.Int64Counter(
		"compiler.memory_read_count",
		metric.WithDescription("Number of memory reads"),
	)
	if err != nil {
		otel.Handle(err)
	}

	memoryReadErrorCount, err = meter.Int64Counter(
		"compiler.memory_read_error_count",
		metric.WithDescription("Number of memory read errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	memoryReadSize, err = meter.Int64Histogram(
		"compiler.memory_read_size",
		metric.WithDescription("Size of memory reads"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

}
