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

package goroutine

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/goroutine")

	// meter is the OpenTelemetry meter for goroutine panic tracking.
	meter = otel.Meter("piko/internal/goroutine")

	// PanicRecoveryCount tracks the total number of panics recovered across
	// all goroutines. Each increment represents a goroutine that would have
	// crashed the process without recovery.
	PanicRecoveryCount metric.Int64Counter

	// ProviderTimeoutCount tracks context deadline or cancellation errors
	// that originated inside a provider rather than from the caller's
	// context. Each increment represents a provider that timed out or
	// cancelled its own internal context.
	ProviderTimeoutCount metric.Int64Counter
)

func init() {
	var err error

	PanicRecoveryCount, err = meter.Int64Counter(
		"goroutine.panic_recovery_count",
		metric.WithDescription("Number of panics recovered in goroutines"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ProviderTimeoutCount, err = meter.Int64Counter(
		"goroutine.provider_timeout_count",
		metric.WithDescription("Number of provider-internal context timeouts detected"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
