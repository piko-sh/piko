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

package provider_stdout

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/email/email_adapters/provider_stdout")

	// meter is the package-level OpenTelemetry meter for stdout provider metrics.
	meter = otel.Meter("piko/internal/email/email_adapters/provider_stdout")

	// sendTotal is a counter that tracks the total number of messages sent.
	sendTotal metric.Int64Counter

	// sendDuration records the time taken to send messages.
	sendDuration metric.Float64Histogram
)

func init() {
	var err error
	sendTotal, err = meter.Int64Counter(
		"email.provider.stdout.send.total",
		metric.WithDescription("Total number of email send attempts"),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	sendDuration, err = meter.Float64Histogram(
		"email.provider.stdout.send.duration",
		metric.WithDescription("Duration of email send operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
