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

package analytics_collector_webhook

import (
	"piko.sh/piko/internal/analytics/analytics_adapters"
	"piko.sh/piko/wdk/analytics"
)

// Option configures a webhook collector.
type Option = analytics_adapters.WebhookOption

// NewCollector creates an analytics collector that batches events
// and POSTs them as JSON to the given URL.
//
// Takes url (string) which is the webhook endpoint.
// Takes opts (...Option) which configure the collector.
//
// Returns analytics.Collector which posts JSON batches to the URL.
func NewCollector(url string, opts ...Option) analytics.Collector {
	return analytics_adapters.NewWebhookCollector(url, opts...)
}

// WithHeaders sets custom HTTP headers sent with each batch POST
// (e.g. Authorization).
var WithHeaders = analytics_adapters.WithWebhookHeaders

// WithBatchSize sets the maximum number of events per batch.
// Defaults to 10.
var WithBatchSize = analytics_adapters.WithWebhookBatchSize

// WithFlushInterval sets the time between automatic batch flushes.
// Defaults to 5 seconds.
var WithFlushInterval = analytics_adapters.WithWebhookFlushInterval

// WithTimeout sets the HTTP client timeout for batch POSTs.
// Defaults to 10 seconds.
var WithTimeout = analytics_adapters.WithWebhookTimeout

// WithRetry enables retry with exponential backoff for failed batch
// sends. Only retryable errors (network failures, 5xx) are retried;
// permanent errors fail immediately.
var WithRetry = analytics_adapters.WithWebhookRetry

// WithCircuitBreaker enables a circuit breaker that stops sending
// batches after consecutive failures. The circuit reopens after the
// timeout expires and a probe request succeeds.
var WithCircuitBreaker = analytics_adapters.WithWebhookCircuitBreaker
