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

// Package analytics_domain coordinates the distribution of backend
// analytics events to pluggable collector backends.
//
// It defines the [Collector] port interface that third-party analytics
// adapters can implement, and a fanout [Service] that distributes events
// to all registered collectors via buffered channels and background workers.
//
// # Collector interface
//
// Adapters implement the [Collector] interface with four methods:
// Collect receives events, Flush sends any buffered batch, Close
// releases resources, and Name returns a human-readable identifier
// for logging and metrics.
//
// # Fanout service
//
// The [Service] owns one buffered channel and goroutine per collector.
// Track sends events to all collectors via non-blocking channel
// sends. When a channel is full the event is dropped and an OTEL
// counter is incremented, ensuring analytics never affects request
// latency. Start launches the worker goroutines and Close drains the
// channels, flushes, and closes each collector.
//
// # Thread safety
//
// The [Service] is safe for concurrent use. Each collector receives
// events on its own buffered channel, drained by a dedicated
// goroutine. Events are dropped (not blocked) when a channel is full,
// preventing analytics from affecting request latency.
package analytics_domain
