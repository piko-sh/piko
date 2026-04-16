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

package provider_disk

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the provider_disk package.
	log = logger_domain.GetLogger("piko/internal/email/email_adapters/provider_disk")

	// meter is the OpenTelemetry meter for the provider_disk package.
	meter = otel.Meter("piko/internal/email/email_adapters/provider_disk")

	// sendTotal tracks the total number of email send attempts.
	// Labels: status (success|error), send_type (single|bulk).
	sendTotal metric.Int64Counter

	// sendDuration tracks the duration of email send operations in milliseconds.
	// Labels: status (success|error), send_type (single|bulk).
	sendDuration metric.Float64Histogram
)

func init() {
	var err error

	sendTotal, err = meter.Int64Counter(
		"email.provider.disk.send.total",
		metric.WithDescription("Total number of email send attempts"),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	sendDuration, err = meter.Float64Histogram(
		"email.provider.disk.send.duration",
		metric.WithDescription("Duration of email send operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
