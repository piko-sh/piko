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

// FailPolicy determines behaviour when the backing store is unavailable.
type FailPolicy uint8

const (
	// FailOpen allows the request when the store is unreachable.
	// This is the recommended default for most use cases, ensuring
	// availability is not degraded by rate limiter infrastructure failures.
	FailOpen FailPolicy = iota

	// FailClosed denies the request when the store is unreachable.
	// Use this when security is more important than availability.
	FailClosed
)

// TokenBucketConfig configures a token bucket rate limiter.
//
// The token bucket allows a sustained rate of operations per second with
// short bursts up to the Burst capacity. Tokens are continuously refilled
// at the Rate per second.
type TokenBucketConfig struct {
	// Rate is the number of operations allowed per second.
	// Must be greater than zero.
	Rate float64

	// Burst is the maximum number of tokens the bucket can hold, allowing
	// short bursts above the steady-state rate. If zero, defaults to Rate.
	Burst int
}

// FixedWindowConfig configures a fixed window rate limiter.
//
// The fixed window divides time into discrete windows of the specified
// duration and allows up to Limit operations per window.
type FixedWindowConfig struct {
	// Limit is the maximum number of operations per window.
	// Must be greater than zero.
	Limit int

	// Window is the duration of each rate limit window.
	// Must be greater than zero.
	Window time.Duration
}

// MaxTokens returns the effective maximum token count for the bucket.
// If Burst is zero, it defaults to Rate.
//
// Returns float64 which is the maximum number of tokens the bucket can hold.
func (c TokenBucketConfig) MaxTokens() float64 {
	if c.Burst > 0 {
		return float64(c.Burst)
	}
	return c.Rate
}
