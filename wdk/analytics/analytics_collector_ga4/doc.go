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

// Package analytics_collector_ga4 provides an analytics collector
// that sends events to the Google Analytics 4 Measurement Protocol.
//
// This collector is ideal for server-side conversion tracking,
// purchase events, and enriching GA4 with backend-only data.
// Events are batched (up to 25 per request per the GA4 protocol
// limit) and POSTed as JSON.
//
// # Usage
//
//	collector := analytics_collector_ga4.NewCollector(
//	    "G-XXXXXXXXXX",
//	    "api-secret",
//	    analytics_collector_ga4.WithDebug(true),
//	)
//
//	server := piko.New(
//	    piko.WithBackendAnalytics(collector),
//	)
package analytics_collector_ga4
