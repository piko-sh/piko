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

package ratelimiter_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the ratelimiter_domain package.
	log = logger_domain.GetLogger("piko/internal/ratelimiter/ratelimiter_domain")

	// meter is the OpenTelemetry meter for the rate limiter domain package.
	meter = otel.Meter("piko/internal/ratelimiter/ratelimiter_domain")

	// checksTotal tracks the total number of rate limit checks.
	checksTotal metric.Int64Counter

	// allowedTotal tracks the number of allowed requests.
	allowedTotal metric.Int64Counter

	// deniedTotal tracks the number of denied requests.
	deniedTotal metric.Int64Counter

	// errorsTotal tracks the number of store errors encountered.
	errorsTotal metric.Int64Counter

	// checkDuration records the latency of rate limit checks.
	checkDuration metric.Float64Histogram
)

func init() {
	var err error

	checksTotal, err = meter.Int64Counter(
		"piko.ratelimiter.checks.total",
		metric.WithDescription("Total number of rate limit checks."),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	allowedTotal, err = meter.Int64Counter(
		"piko.ratelimiter.allowed.total",
		metric.WithDescription("Number of requests allowed by the rate limiter."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	deniedTotal, err = meter.Int64Counter(
		"piko.ratelimiter.denied.total",
		metric.WithDescription("Number of requests denied by the rate limiter."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	errorsTotal, err = meter.Int64Counter(
		"piko.ratelimiter.errors.total",
		metric.WithDescription("Number of rate limiter store errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	checkDuration, err = meter.Float64Histogram(
		"piko.ratelimiter.check.duration",
		metric.WithDescription("Duration of rate limit checks."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
