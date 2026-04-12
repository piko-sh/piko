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

// Package analytics provides a provider-agnostic framework for
// backend analytics event collection, with support for batching,
// retry, and circuit breaking.
//
// Create collectors from the analytics_collector_* sub-packages
// and register them with the Piko server via
// [piko.WithBackendAnalytics].
//
// # Usage
//
// Registering collectors at startup:
//
//	import (
//	    ga4 "piko.sh/piko/wdk/analytics/analytics_collector_ga4"
//	    webhook "piko.sh/piko/wdk/analytics/analytics_collector_webhook"
//	)
//
//	server := piko.New(
//	    piko.WithBackendAnalytics(
//	        webhook.NewCollector("https://ingest.example.com/events"),
//	        ga4.NewCollector("G-XXXXXXXXXX", "api-secret"),
//	    ),
//	)
//
// # Collectors
//
// Collector adapters for various analytics backends are available
// in the analytics_collector_* sub-packages.
//
// # Thread safety
//
// [Collector] implementations are safe for concurrent use once
// started by the analytics service.
package analytics
