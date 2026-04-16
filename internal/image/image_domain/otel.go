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

package image_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the image_domain package.
	log = logger_domain.GetLogger("piko/internal/image/image_domain")

	// meter is the OpenTelemetry meter for the image domain package.
	meter = otel.Meter("piko/internal/image/image_domain")

	// transformDuration tracks the time taken for image transformations.
	transformDuration metric.Float64Histogram

	// transformCount tracks the total number of transformation requests.
	transformCount metric.Int64Counter

	// transformErrorCount tracks the number of transformations that have failed.
	transformErrorCount metric.Int64Counter

	// securityViolationCount tracks the number of security violations found.
	securityViolationCount metric.Int64Counter
)

func init() {
	var err error

	transformDuration, err = meter.Float64Histogram(
		"image.transform.duration",
		metric.WithDescription("Duration of image transformation operations in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transformCount, err = meter.Int64Counter(
		"image.transform.count",
		metric.WithDescription("Total number of image transformation requests"),
		metric.WithUnit("{transformation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transformErrorCount, err = meter.Int64Counter(
		"image.transform.error.count",
		metric.WithDescription("Total number of failed image transformations"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	securityViolationCount, err = meter.Int64Counter(
		"image.security.violation.count",
		metric.WithDescription("Number of security violations detected (dimension limits, SSRF attempts, etc.)"),
		metric.WithUnit("{violation}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
