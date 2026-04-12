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

// Package analytics_collector_webhook provides an analytics collector
// that batches events and POSTs them as JSON to a configurable
// webhook endpoint.
//
// This collector is suitable for forwarding analytics events to
// custom ingest endpoints, data warehouses, or third-party services
// that accept JSON payloads.
//
// # Usage
//
//	collector := analytics_collector_webhook.NewCollector(
//	    "https://ingest.example.com/events",
//	    analytics_collector_webhook.WithBatchSize(25),
//	    analytics_collector_webhook.WithRetry(analytics.RetryConfig{
//	        MaxRetries:    3,
//	        InitialDelay:  time.Second,
//	        MaxDelay:      30 * time.Second,
//	        BackoffFactor: 2.0,
//	    }),
//	)
//
//	server := piko.New(
//	    piko.WithBackendAnalytics(collector),
//	)
package analytics_collector_webhook
