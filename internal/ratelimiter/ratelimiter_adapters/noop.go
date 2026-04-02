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

package ratelimiter_adapters

import (
	"context"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

var (
	// noopClock is a package-level real clock used by NoopCounterStore to provide
	// a window start time without requiring a constructor.
	noopClock = clock.RealClock()

	_ ratelimiter_domain.TokenBucketStorePort = NoopTokenBucketStore{}

	_ ratelimiter_domain.CounterStorePort = NoopCounterStore{}
)

// NoopTokenBucketStore is a no-op implementation of TokenBucketStorePort that
// always allows requests. Used when rate limiting is disabled or the cache is
// unavailable.
type NoopTokenBucketStore struct{}

// TryTake always returns true (allowed).
//
// Returns bool which is always true, indicating the request is allowed.
// Returns error which is always nil.
func (NoopTokenBucketStore) TryTake(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	return true, nil
}

// WaitDuration always returns zero (no waiting needed).
//
// Returns time.Duration which is always zero for this no-op implementation.
// Returns error which is always nil.
func (NoopTokenBucketStore) WaitDuration(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	return 0, nil
}

// DeleteBucket is a no-op.
//
// Returns error which is always nil.
func (NoopTokenBucketStore) DeleteBucket(_ context.Context, _ string) error {
	return nil
}

// NoopCounterStore is a no-op implementation of CounterStorePort that always
// returns zero. Used when rate limiting is disabled or the cache is unavailable.
type NoopCounterStore struct{}

// IncrementAndGet always returns a zero count with the current time as window
// start (no counting).
//
// Returns ratelimiter_dto.CounterResult which contains zero count and the
// current time as window start.
// Returns error which is always nil for this no-op implementation.
func (NoopCounterStore) IncrementAndGet(_ context.Context, _ string, _ int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
	return ratelimiter_dto.CounterResult{
		Count:       0,
		WindowStart: noopClock.Now(),
	}, nil
}
