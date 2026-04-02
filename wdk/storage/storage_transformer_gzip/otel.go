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

package storage_transformer_gzip

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	// log is the package-level logger for storage transformer gzip operations.
	log = logger.GetLogger("piko/storage/storage_transformer_gzip")

	// Meter is the OpenTelemetry meter for Gzip transformer metrics.
	Meter = otel.Meter("piko/storage/storage_transformer_gzip")

	// OperationDuration tracks the duration of transform operations.
	OperationDuration metric.Float64Histogram

	// TransformOperationsTotal tracks the total number of transform operations
	// (compress/decompress).
	TransformOperationsTotal metric.Int64Counter

	// TransformErrorsTotal counts the total number of failed transform operations.
	TransformErrorsTotal metric.Int64Counter

	// BytesProcessed tracks the total number of bytes compressed or decompressed.
	BytesProcessed metric.Int64Counter

	// CompressionRatio tracks the compression ratio that the system achieves.
	CompressionRatio metric.Float64Histogram
)

func init() {
	var err error

	OperationDuration, err = Meter.Float64Histogram(
		"storage.transformer.gzip.operation.duration",
		metric.WithDescription("Duration of Gzip transform operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TransformOperationsTotal, err = Meter.Int64Counter(
		"storage.transformer.gzip.operations.total",
		metric.WithDescription("Total number of Gzip transform operations (compress/decompress)"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TransformErrorsTotal, err = Meter.Int64Counter(
		"storage.transformer.gzip.errors.total",
		metric.WithDescription("Total number of failed Gzip transform operations"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BytesProcessed, err = Meter.Int64Counter(
		"storage.transformer.gzip.bytes.processed",
		metric.WithDescription("Total bytes compressed or decompressed by Gzip transformer"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompressionRatio, err = Meter.Float64Histogram(
		"storage.transformer.gzip.compression.ratio",
		metric.WithDescription("Compression ratio achieved by Gzip (compressed_size / original_size)"),
		metric.WithUnit("{ratio}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
