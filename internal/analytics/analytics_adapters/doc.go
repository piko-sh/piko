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

// Package analytics_adapters implements driven adapters for the backend
// analytics subsystem.
//
// It provides an HTTP middleware that automatically fires page view
// events for every request, and a built-in webhook collector that
// batches events and POSTs them as JSON to a configurable endpoint.
//
// # Middleware
//
// [AnalyticsMiddleware] is installed after auth and before rate
// limiting in the HTTP middleware chain. It wraps the response writer
// to capture the status code and duration, enriches events from
// [daemon_dto.PikoRequestCtx] (client IP, locale, matched pattern,
// user ID), and sends them to the analytics service. When no
// collectors are registered the middleware is not installed, ensuring
// zero overhead.
//
// # Webhook collector
//
// [WebhookCollector] is a built-in adapter that demonstrates the
// collector pattern. It batches event snapshots in an internal buffer
// and flushes them as JSON to a configurable URL on a timer or when
// the batch reaches a configurable size. Custom headers (e.g.
// Authorization) can be set via [WithWebhookHeaders].
//
// # Thread safety
//
// All adapters are safe for concurrent use.
package analytics_adapters
