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

import "time"

// CounterResult holds the outcome of a counter store increment operation.
// It provides both the updated count and the window start time so that
// callers can compute accurate ResetAt and RetryAfter values.
type CounterResult struct {
	// WindowStart is when the current fixed window began.
	WindowStart time.Time

	// Count is the counter value after incrementing.
	Count int64
}

// Result holds the outcome of a rate limit check.
// It provides all the data needed for setting rate limit headers in HTTP
// responses and for callers to make informed retry decisions.
type Result struct {
	// ResetAt is when the current rate limit window resets.
	ResetAt time.Time

	// Limit is the maximum number of operations allowed in the window.
	Limit int

	// Remaining is the number of operations remaining in the current window.
	Remaining int

	// RetryAfter is how long to wait before the next request is allowed.
	// Only meaningful when Allowed is false.
	RetryAfter time.Duration

	// Allowed indicates whether the request is permitted under the rate limit.
	Allowed bool
}
