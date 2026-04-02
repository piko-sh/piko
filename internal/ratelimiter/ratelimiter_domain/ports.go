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

package ratelimiter_domain

import (
	"context"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

// TokenBucketState holds the persistent state of a token bucket rate limiter.
// This state can be stored externally (such as in Redis) and shared across
// instances for distributed rate limiting.
type TokenBucketState struct {
	// Tokens is the current number of tokens in the bucket.
	Tokens float64

	// MaxTokens is the maximum number of tokens the bucket can hold.
	MaxTokens float64

	// RefillRate is the rate at which tokens are added, in tokens per nanosecond.
	RefillRate float64

	// LastRefillNano is the Unix timestamp in nanoseconds of the last refill.
	LastRefillNano int64
}

// TokenBucketStorePort is the driven port for token bucket state storage.
// Implementations must be safe for concurrent access and provide atomic
// read-modify-write operations for token bucket state.
type TokenBucketStorePort interface {
	// TryTake atomically attempts to take n tokens from the bucket. It first
	// refills based on elapsed time, then attempts to deduct tokens.
	//
	// Takes key (string) which identifies the rate limit bucket.
	// Takes n (float64) which is the number of tokens to take.
	// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket
	// parameters.
	//
	// Returns bool which is true if tokens were successfully taken.
	// Returns error when the operation fails.
	TryTake(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error)

	// WaitDuration returns the estimated time until n tokens become available.
	// This can be used for backoff calculations.
	//
	// Takes key (string) which identifies the rate limit bucket.
	// Takes n (float64) which is the number of tokens needed.
	// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket
	// parameters.
	//
	// Returns time.Duration which is how long to wait for tokens.
	// Returns error when the operation fails.
	WaitDuration(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error)

	// DeleteBucket removes a bucket's state from storage.
	//
	// Takes key (string) which identifies the rate limit bucket.
	//
	// Returns error when the deletion fails.
	DeleteBucket(ctx context.Context, key string) error
}

// CounterStorePort is the driven port for fixed-window counter storage.
// Implementations must provide atomic increment operations with TTL support.
type CounterStorePort interface {
	// IncrementAndGet atomically increments the counter for key by delta.
	// The TTL is set only when the counter is first created, implementing
	// a fixed-window strategy where the window starts on first request.
	//
	// Takes key (string) which identifies the rate limit counter.
	// Takes delta (int64) which is the amount to increment by.
	// Takes window (time.Duration) which is the TTL for new counters.
	//
	// Returns ratelimiter_dto.CounterResult which contains the counter value
	// after incrementing and the window start time.
	// Returns error when the operation fails.
	IncrementAndGet(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error)
}

// RateLimiterInspector provides read-only access to rate limiter state for
// monitoring. It is implemented by the Limiter and consumed by the monitoring
// gRPC service.
type RateLimiterInspector interface {
	// GetStatus returns the current status of the rate limiter, including
	// store types, fail policy, and aggregate counters.
	//
	// Returns ratelimiter_dto.Status which contains the inspectable state.
	// Returns error when the status cannot be retrieved.
	GetStatus(ctx context.Context) (ratelimiter_dto.Status, error)
}
