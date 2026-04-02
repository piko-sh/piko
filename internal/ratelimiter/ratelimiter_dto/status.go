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

package ratelimiter_dto

// Status holds the inspectable state of the centralised rate limiter.
// It is returned by the RateLimiterInspector interface for monitoring
// and CLI display.
type Status struct {
	// TokenBucketStore is the name of the active token bucket store
	// (e.g. "cache", "inmemory", "noop").
	TokenBucketStore string

	// CounterStore is the name of the active fixed window counter store
	// (e.g. "cache", "noop").
	CounterStore string

	// FailPolicy is the current failure behaviour ("open" or "closed").
	FailPolicy string

	// KeyPrefix is the prefix prepended to all rate limit keys.
	KeyPrefix string

	// TotalChecks is the total number of rate limit checks performed.
	TotalChecks int64

	// TotalAllowed is the number of requests that were allowed.
	TotalAllowed int64

	// TotalDenied is the number of requests that were denied.
	TotalDenied int64

	// TotalErrors is the number of store errors encountered.
	TotalErrors int64
}
