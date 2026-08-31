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

package security_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// meter provides OpenTelemetry metrics for the security adapters package.
	meter = otel.Meter("piko/internal/security/security_adapters")

	// userAgentClampedCount tracks User-Agent headers that exceeded the capture limit and
	// were shortened.
	userAgentClampedCount metric.Int64Counter

	// forwardedRequestIDRejectedCount tracks forwarded request IDs rejected for containing
	// characters outside the accepted set.
	forwardedRequestIDRejectedCount metric.Int64Counter
)

func init() {
	var err error

	userAgentClampedCount, err = meter.Int64Counter(
		"security.user_agent_clamped_count",
		metric.WithDescription("Number of User-Agent headers shortened at capture"),
	)
	if err != nil {
		otel.Handle(err)
	}

	forwardedRequestIDRejectedCount, err = meter.Int64Counter(
		"security.forwarded_request_id_rejected_count",
		metric.WithDescription("Number of forwarded request IDs rejected as malformed"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
