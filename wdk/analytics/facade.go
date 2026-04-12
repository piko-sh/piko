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

package analytics

import (
	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/retry"
)

// Collector is the interface that backend analytics adapters
// implement. Collector implementations must be safe for concurrent
// use from multiple goroutines.
type Collector = analytics_domain.Collector

// Event carries the data for a single backend analytics event.
type Event = analytics_dto.Event

// EventType classifies a backend analytics event.
type EventType = analytics_dto.EventType

// RetryConfig configures retry with exponential backoff for
// analytics collectors.
type RetryConfig = retry.Config

// CircuitBreakerConfig configures the circuit breaker that protects
// the send path of analytics collectors.
type CircuitBreakerConfig = analytics_domain.CircuitBreakerConfig

const (
	// EventPageView is fired automatically for each page request.
	EventPageView = analytics_dto.EventPageView

	// EventAction is fired when a server action executes.
	EventAction = analytics_dto.EventAction

	// EventCustom is a user-defined event fired manually from action
	// handlers.
	EventCustom = analytics_dto.EventCustom
)
