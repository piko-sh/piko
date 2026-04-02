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

package pdfwriter_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// logFieldTransformer is the log field key for the transformer name.
	logFieldTransformer = "transformer"

	// logFieldPriority is the log field key for transformer priority.
	logFieldPriority = "priority"
)

var (
	log = logger_domain.GetLogger("piko/internal/pdfwriter/pdfwriter_domain")

	// meter is the OpenTelemetry meter for PDF writer domain metrics.
	meter = otel.Meter("piko/internal/pdfwriter/pdfwriter_domain")

	// transformDuration tracks the duration of individual PDF
	// transformations run by the transformer chain.
	transformDuration metric.Float64Histogram

	// transformsTotal tracks the total number of PDF transformations by
	// name.
	transformsTotal metric.Int64Counter

	// transformErrorsTotal tracks the total number of failed PDF
	// transformations.
	transformErrorsTotal metric.Int64Counter
)

func init() {
	var err error

	transformDuration, err = meter.Float64Histogram(
		"pdfwriter.domain.transform.duration",
		metric.WithDescription("Duration of individual PDF transformation steps"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transformsTotal, err = meter.Int64Counter(
		"pdfwriter.domain.transforms.total",
		metric.WithDescription("Total number of PDF transformations by name"),
		metric.WithUnit("{transform}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transformErrorsTotal, err = meter.Int64Counter(
		"pdfwriter.domain.transform.errors.total",
		metric.WithDescription("Total number of failed PDF transformations"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
