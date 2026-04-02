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

package storage_provider_s3

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	// log is the package-level logger for S3 provider operations.
	log = logger.GetLogger("piko/storage/storage_provider_s3")

	// Meter is the OpenTelemetry meter for S3 provider metrics.
	Meter = otel.Meter("piko/storage/storage_provider_s3")

	// OperationDuration tracks the duration of storage operations.
	OperationDuration metric.Float64Histogram

	// OperationsTotal tracks the total number of operations by type.
	OperationsTotal metric.Int64Counter

	// OperationErrorsTotal tracks the total number of failed operations.
	OperationErrorsTotal metric.Int64Counter

	// BytesTransferred tracks the total bytes read and written.
	BytesTransferred metric.Int64Counter

	// BatchOperationsTotal counts batch operations like PutMany and RemoveMany.
	BatchOperationsTotal metric.Int64Counter

	// MultipartUploadsTotal tracks the total count of multipart upload operations.
	MultipartUploadsTotal metric.Int64Counter
)

func init() {
	var err error

	OperationDuration, err = Meter.Float64Histogram(
		"storage.provider.s3.operation.duration",
		metric.WithDescription("Duration of S3 storage operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OperationsTotal, err = Meter.Int64Counter(
		"storage.provider.s3.operations.total",
		metric.WithDescription("Total number of S3 storage operations by type"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OperationErrorsTotal, err = Meter.Int64Counter(
		"storage.provider.s3.operation.errors.total",
		metric.WithDescription("Total number of failed S3 storage operations"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BytesTransferred, err = Meter.Int64Counter(
		"storage.provider.s3.bytes.transferred",
		metric.WithDescription("Total bytes read from or written to S3"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BatchOperationsTotal, err = Meter.Int64Counter(
		"storage.provider.s3.batch.operations.total",
		metric.WithDescription("Total number of batch operations (PutMany, RemoveMany)"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	MultipartUploadsTotal, err = Meter.Int64Counter(
		"storage.provider.s3.multipart.uploads.total",
		metric.WithDescription("Total number of multipart upload operations"),
		metric.WithUnit("{upload}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
