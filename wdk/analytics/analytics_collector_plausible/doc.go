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

// Package analytics_collector_plausible provides an analytics collector
// that sends events to the Plausible Analytics Events API.
//
// Plausible is a privacy-friendly, cookie-free analytics service.
// This collector sends events server-side, so no client-side
// JavaScript is needed and events cannot be blocked by ad blockers.
//
// # Usage
//
//	collector, err := analytics_collector_plausible.NewCollector("example.com")
//
//	server := piko.New(
//	    piko.WithBackendAnalytics(collector),
//	)
//
// # Self-hosted
//
//	collector, err := analytics_collector_plausible.NewCollector("example.com",
//	    analytics_collector_plausible.WithEndpoint("https://analytics.example.com"),
//	)
//
// # Authentication
//
// Plausible uses domain-based identification for the Events API.
// No API key is required, the domain must match a site configured
// in your Plausible account.
package analytics_collector_plausible
