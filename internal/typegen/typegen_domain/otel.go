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

package typegen_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the typegen_domain package.
	log = logger_domain.GetLogger("piko/internal/typegen/typegen_domain")

	// meter is the OpenTelemetry meter for the typegen_domain package.
	meter = otel.Meter("piko/internal/typegen/typegen_domain")

	// typeDefsWritten counts the number of type definition files written.
	typeDefsWritten metric.Int64Counter

	// typeDefsWriteErrors counts errors when writing type definitions.
	typeDefsWriteErrors metric.Int64Counter
)

func init() {
	var err error

	typeDefsWritten, err = meter.Int64Counter(
		"typegen.type_defs_written",
		metric.WithDescription("Number of type definition files written"),
	)
	if err != nil {
		otel.Handle(err)
	}

	typeDefsWriteErrors, err = meter.Int64Counter(
		"typegen.type_defs_write_errors",
		metric.WithDescription("Number of errors writing type definition files"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
