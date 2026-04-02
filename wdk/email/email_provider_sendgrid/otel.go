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

package email_provider_sendgrid

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/email/email_provider_sendgrid")

	// Meter is the package-level meter for SendGrid provider metrics.
	Meter = otel.Meter("piko/email/email_provider_sendgrid")

	// SendTotal tracks the total number of email send attempts.
	// Labels: status (success|error), send_type (single|bulk).
	SendTotal metric.Int64Counter

	// SendDuration tracks the duration of email send operations in milliseconds.
	// Labels: status (success|error), send_type (single|bulk).
	SendDuration metric.Float64Histogram
)

func init() {
	var err error

	SendTotal, err = Meter.Int64Counter(
		"email.provider.sendgrid.send.total",
		metric.WithDescription("Total number of email send attempts"),
		metric.WithUnit("{email}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SendDuration, err = Meter.Float64Histogram(
		"email.provider.sendgrid.send.duration",
		metric.WithDescription("Duration of email send operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
