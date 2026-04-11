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

package analytics_collector_ga4

import (
	"piko.sh/piko/internal/analytics/analytics_adapters"
	"piko.sh/piko/wdk/analytics"
)

// Option configures a GA4 collector.
type Option = analytics_adapters.GA4Option

// NewCollector creates an analytics collector that sends events to
// the GA4 Measurement Protocol endpoint.
//
// Takes measurementID (string) which is the GA4 measurement ID
// (e.g. "G-XXXXXXXXXX").
// Takes opts (...Option) which configure the collector.
//
// Returns analytics.Collector which posts events to the GA4
// Measurement Protocol endpoint.
func NewCollector(measurementID, apiSecret string, opts ...Option) analytics.Collector {
	return analytics_adapters.NewGA4Collector(measurementID, apiSecret, opts...)
}

// WithBatchSize sets the maximum number of events buffered before
// triggering a flush. The value is clamped to the GA4 maximum of 25
// events per request.
var WithBatchSize = analytics_adapters.WithGA4BatchSize

// WithFlushInterval sets the time between automatic batch flushes.
// Defaults to 5 seconds.
var WithFlushInterval = analytics_adapters.WithGA4FlushInterval

// WithTimeout sets the HTTP client timeout for GA4 POSTs.
// Defaults to 10 seconds.
var WithTimeout = analytics_adapters.WithGA4Timeout

// WithClientIDFunc sets a custom function to derive the GA4
// client_id from request data. The default hashes client IP and
// user agent with SHA-256.
var WithClientIDFunc = analytics_adapters.WithGA4ClientIDFunc

// WithDebug enables the GA4 debug endpoint which validates events
// but does not record them in Google Analytics. Useful for testing.
var WithDebug = analytics_adapters.WithGA4Debug

// WithRetry enables retry with exponential backoff for failed batch
// sends. Only retryable errors (network failures, 5xx) are retried;
// permanent errors fail immediately.
var WithRetry = analytics_adapters.WithGA4Retry

// WithCircuitBreaker enables a circuit breaker that stops sending
// batches after consecutive failures. The circuit reopens after the
// timeout expires and a probe request succeeds.
var WithCircuitBreaker = analytics_adapters.WithGA4CircuitBreaker
