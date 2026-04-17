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

package analytics_domain

import (
	"context"

	"piko.sh/piko/internal/analytics/analytics_dto"
)

// Collector is the driven port that analytics backends implement.
//
// Implementations must be safe for concurrent use from multiple
// goroutines. The Event passed to Collect must not be retained after
// the method returns; copy any needed data.
type Collector interface {
	// Start launches any background goroutines (e.g. flush loops)
	// needed by the collector.
	Start(ctx context.Context)

	// Collect receives a single analytics event. The implementation
	// may buffer events internally for batching.
	//
	// Takes event (*analytics_dto.Event) which carries the event data.
	//
	// Returns error when the event cannot be accepted.
	Collect(ctx context.Context, event *analytics_dto.Event) error

	// Flush sends any buffered events to the backend. Called during
	// graceful shutdown and optionally on a timer.
	//
	// Returns error when flushing fails.
	Flush(ctx context.Context) error

	// Close releases resources held by the collector. Called once
	// during shutdown after Flush.
	//
	// Returns error when cleanup fails.
	Close(ctx context.Context) error

	// HealthCheck verifies that the collector can reach its backend.
	// Returns nil when the collector is healthy.
	//
	// Returns error when the backend is unreachable or degraded.
	HealthCheck(ctx context.Context) error

	// Name returns a human-readable identifier for logging and metrics.
	//
	// Returns string which identifies this collector.
	Name() string
}
