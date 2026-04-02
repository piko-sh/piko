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

// Package events_domain defines the [Provider] port interface and
// configuration types for the Watermill-based message bus
// infrastructure. Concrete providers (GoChannel, NATS) implement the
// port; OpenTelemetry metrics are collected for provider operations.
//
// # Provider lifecycle
//
// Providers follow a standard lifecycle:
//
//  1. Create provider with NewXxxProvider(config)
//  2. Call Start(ctx) to initialise the Watermill router
//  3. Use Router(), Publisher(), Subscriber() to access components
//  4. Call Close() for graceful shutdown
package events_domain
