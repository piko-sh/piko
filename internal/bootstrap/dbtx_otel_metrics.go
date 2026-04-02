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

package bootstrap

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	dbMeter = otel.Meter("piko/bootstrap/db")

	dbOperationDuration metric.Float64Histogram

	dbOperationCount metric.Int64Counter

	dbOperationErrorCount metric.Int64Counter
)

func init() {
	var err error

	dbOperationDuration, err = dbMeter.Float64Histogram(
		"db.client.operation.duration",
		metric.WithDescription("Duration of database client operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dbOperationCount, err = dbMeter.Int64Counter(
		"db.client.operation.count",
		metric.WithDescription("Total number of database client operations."),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dbOperationErrorCount, err = dbMeter.Int64Counter(
		"db.client.operation.error.count",
		metric.WithDescription("Total number of failed database client operations."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
