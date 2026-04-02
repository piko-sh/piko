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

package storage_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for storage domain operations.
	log = logger_domain.GetLogger("piko/internal/storage/storage_domain")

	// meter is the OpenTelemetry meter for storage domain metrics.
	meter = otel.Meter("piko/internal/storage/storage_domain")

	// operationDuration tracks the duration of domain-level storage operations.
	operationDuration metric.Float64Histogram

	// operationsTotal tracks the total number of domain operations by type.
	operationsTotal metric.Int64Counter

	// operationErrorsTotal tracks the total number of failed domain operations.
	operationErrorsTotal metric.Int64Counter

	// batchOperationsTotal tracks the count of batch operations such as PutObjects
	// and RemoveObjects.
	batchOperationsTotal metric.Int64Counter

	// batchItemsTotal tracks the total number of items processed in batch
	// operations.
	batchItemsTotal metric.Int64Counter
)

func init() {
	var err error

	operationDuration, err = meter.Float64Histogram(
		"storage.domain.operation.duration",
		metric.WithDescription("Duration of domain-level storage operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	operationsTotal, err = meter.Int64Counter(
		"storage.domain.operations.total",
		metric.WithDescription("Total number of domain-level storage operations by type"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	operationErrorsTotal, err = meter.Int64Counter(
		"storage.domain.operation.errors.total",
		metric.WithDescription("Total number of failed domain-level storage operations"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchOperationsTotal, err = meter.Int64Counter(
		"storage.domain.batch.operations.total",
		metric.WithDescription("Total number of batch operations (PutObjects, RemoveObjects)"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	batchItemsTotal, err = meter.Int64Counter(
		"storage.domain.batch.items.total",
		metric.WithDescription("Total number of items processed in batch operations"),
		metric.WithUnit("{item}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
